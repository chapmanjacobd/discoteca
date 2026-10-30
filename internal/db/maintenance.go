package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// MaintenanceConfig holds configuration for maintenance tasks
type MaintenanceConfig struct {
	// RefreshInterval is the minimum time between automatic refreshes
	// Default: 72 hours
	RefreshInterval time.Duration
}

// DefaultMaintenanceConfig returns the default maintenance configuration
func DefaultMaintenanceConfig() MaintenanceConfig {
	return MaintenanceConfig{
		RefreshInterval: 72 * time.Hour,
	}
}

// MaintenanceStatus holds the status of maintenance tasks
type MaintenanceStatus struct {
	FolderStatsLastRefresh time.Time
	FTSLastRebuild         time.Time
}

// GetMaintenanceStatus returns the current status of maintenance tasks
func GetMaintenanceStatus(ctx context.Context, db *sql.DB) (MaintenanceStatus, error) {
	var status MaintenanceStatus

	// Get folder_stats last refresh
	var folderStatsStr string
	var folderStatsTime int64
	err := db.QueryRowContext(ctx, "SELECT value, last_updated FROM _maintenance_meta WHERE key = 'folder_stats_last_refresh'").
		Scan(&folderStatsStr, &folderStatsTime)
	if err == nil && folderStatsTime > 0 {
		status.FolderStatsLastRefresh = time.Unix(folderStatsTime, 0)
	}

	// Get FTS last rebuild
	var ftsStr string
	var ftsTime int64
	err = db.QueryRowContext(ctx, "SELECT value, last_updated FROM _maintenance_meta WHERE key = 'fts_last_rebuild'").
		Scan(&ftsStr, &ftsTime)
	if err == nil && ftsTime > 0 {
		status.FTSLastRebuild = time.Unix(ftsTime, 0)
	}

	return status, nil
}

// NeedsRefresh checks if maintenance tasks need to be run.
//
// folder_stats uses a dirty bit written by media triggers and cleared after a
// successful refresh. FTS still uses the timestamp interval because rewriting
// the full-text index is expensive and not tied to a single write.
func NeedsRefresh(ctx context.Context, db *sql.DB, interval time.Duration) (bool, error) {
	status, err := GetMaintenanceStatus(ctx, db)
	if err != nil {
		return true, err // If we can't check, assume we need to refresh
	}

	var dirty int
	err = db.QueryRowContext(ctx, "SELECT dirty FROM folder_stats_meta WHERE id = 1").Scan(&dirty)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return true, err
	}

	folderStatsDirty := dirty != 0 || status.FolderStatsLastRefresh.IsZero()
	ftsStale := status.FTSLastRebuild.IsZero() || time.Since(status.FTSLastRebuild) > interval

	return folderStatsDirty || ftsStale, nil
}

// RefreshFolderStats rebuilds the folder_stats materialized view and clears
// the dirty bit.
func RefreshFolderStats(ctx context.Context, db *sql.DB) error {
	Log.Info("Refreshing folder_stats materialized view...")
	start := time.Now()

	if err := PopulateFolderStatsInGo(ctx, db); err != nil {
		return fmt.Errorf("failed to populate folder_stats: %w", err)
	}

	// Update metadata and clear the dirty bit only after a successful rebuild.
	if err := markFolderStatsRefreshed(ctx, db); err != nil {
		return err
	}

	Log.Info("folder_stats refresh completed", "duration", time.Since(start))
	return nil
}

func markFolderStatsRefreshed(ctx context.Context, db dbExecer) error {
	now := time.Now().Unix()
	if _, err := db.ExecContext(ctx, `
		INSERT OR REPLACE INTO _maintenance_meta (key, value, last_updated)
		VALUES ('folder_stats_last_refresh', ?, ?)
	`, "success", now); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE folder_stats_meta SET dirty = 0 WHERE id = 1"); err != nil {
		return fmt.Errorf("failed to clear folder_stats dirty bit: %w", err)
	}
	return nil
}

// RebuildFTS rebuilds the FTS indexes.
func RebuildFTS(ctx context.Context, db *sql.DB, dbPath string) error {
	Log.Info("Rebuilding FTS index...", "db", dbPath)
	start := time.Now()

	// Check if FTS table exists
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='media_fts'
		)
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check FTS table: %w", err)
	}
	if !exists {
		Log.Debug("FTS table does not exist, skipping rebuild", "db", dbPath)
		return nil
	}

	if _, err = db.ExecContext(ctx, "INSERT INTO media_fts(media_fts) VALUES('rebuild')"); err != nil {
		return fmt.Errorf("failed to rebuild media_fts: %w", err)
	}

	var hasCaptionsFTS bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='captions_fts'
		)
	`).Scan(&hasCaptionsFTS)
	if err != nil {
		return fmt.Errorf("failed to check captions FTS table: %w", err)
	}
	if hasCaptionsFTS {
		if _, err = db.ExecContext(ctx, "INSERT INTO captions_fts(captions_fts) VALUES('rebuild')"); err != nil {
			return fmt.Errorf("failed to rebuild captions_fts: %w", err)
		}
	}

	// Update metadata
	now := time.Now().Unix()
	_, err = db.ExecContext(ctx, `
		INSERT OR REPLACE INTO _maintenance_meta (key, value, last_updated)
		VALUES ('fts_last_rebuild', ?, ?)
	`, "success", now)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	Log.Info("FTS rebuild completed", "duration", time.Since(start))
	return nil
}

// RunMaintenance runs all maintenance tasks if needed
func RunMaintenance(ctx context.Context, db *sql.DB, config MaintenanceConfig, dbPath string) error {
	needsRefresh, err := NeedsRefresh(ctx, db, config.RefreshInterval)
	if err != nil {
		return err
	}

	if !needsRefresh {
		// Don't log anything when maintenance is not needed
		return nil
	}

	Log.Info("Running scheduled maintenance...", "db", dbPath, "interval", config.RefreshInterval)

	// Run maintenance tasks
	if err := RefreshFolderStats(ctx, db); err != nil {
		Log.Error("Failed to refresh folder_stats", "db", dbPath, "error", err)
		// Continue with FTS rebuild even if folder_stats fails
	}

	if err := RebuildFTS(ctx, db, dbPath); err != nil {
		Log.Error("Failed to rebuild FTS", "db", dbPath, "error", err)
		return err
	}

	Log.Info("Scheduled maintenance completed", "db", dbPath)
	return nil
}

// PopulateFolderStatsInGo populates folder_stats using Go path manipulation.
// It replaces the whole materialized view so folders that no longer contain
// media do not leave stale rows behind.
func PopulateFolderStatsInGo(ctx context.Context, db *sql.DB) error {
	return populateFolderStats(ctx, db)
}

func populateFolderStats(ctx context.Context, db dbExecer) error {
	rows, err := db.QueryContext(ctx, `
		SELECT path, COALESCE(size, 0), COALESCE(duration, 0)
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type folderData struct {
		count    int64
		size     int64
		duration int64
	}
	folderMap := make(map[string]*folderData)

	for rows.Next() {
		var path string
		var size, duration int64
		if err := rows.Scan(&path, &size, &duration); err != nil {
			continue
		}
		parent := filepath.Dir(path)
		if _, ok := folderMap[parent]; !ok {
			folderMap[parent] = &folderData{}
		}
		folderMap[parent].count++
		folderMap[parent].size += size
		folderMap[parent].duration += duration
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM folder_stats"); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO folder_stats (parent, file_count, total_size, total_duration)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for parent, data := range folderMap {
		if _, err := stmt.ExecContext(ctx, parent, data.count, data.size, data.duration); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func folderStatsEmpty(ctx context.Context, db dbExecer) (bool, error) {
	var empty bool
	err := db.QueryRowContext(ctx, "SELECT NOT EXISTS(SELECT 1 FROM folder_stats)").Scan(&empty)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return true, nil
		}
		return false, err
	}
	return empty, nil
}
