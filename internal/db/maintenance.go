package db

import (
	"context"
	"database/sql"
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

// NeedsRefresh checks if maintenance tasks need to be run based on the last refresh time
func NeedsRefresh(ctx context.Context, db *sql.DB, interval time.Duration) (bool, error) {
	status, err := GetMaintenanceStatus(ctx, db)
	if err != nil {
		return true, err // If we can't check, assume we need to refresh
	}

	needsRefresh := time.Since(status.FolderStatsLastRefresh) > interval ||
		time.Since(status.FTSLastRebuild) > interval

	return needsRefresh, nil
}

// RefreshFolderStats rebuilds the folder_stats materialized view
func RefreshFolderStats(ctx context.Context, db *sql.DB) error {
	Log.Info("Refreshing folder_stats materialized view...")
	start := time.Now()

	// Clear existing data
	if _, err := db.ExecContext(ctx, "DELETE FROM folder_stats"); err != nil {
		return fmt.Errorf("failed to clear folder_stats: %w", err)
	}

	// Use Go approach (SQLite doesn't have reverse() function)
	if err := PopulateFolderStatsInGo(ctx, db); err != nil {
		return fmt.Errorf("failed to populate folder_stats: %w", err)
	}

	// Update metadata
	now := time.Now().Unix()
	_, err := db.ExecContext(ctx, `
		INSERT OR REPLACE INTO _maintenance_meta (key, value, last_updated)
		VALUES ('folder_stats_last_refresh', ?, ?)
	`, "success", now)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	Log.Info("folder_stats refresh completed", "duration", time.Since(start))
	return nil
}

// RebuildFTS rebuilds the FTS index
func RebuildFTS(ctx context.Context, db *sql.DB, dbPath string) error {
	Log.Info("Rebuilding FTS index...", "db", dbPath)
	start := time.Now()

	// Check if FTS table exists
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='virtual' AND name='media_fts'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		Log.Debug("FTS table does not exist, skipping rebuild", "db", dbPath)
		return nil
	}

	// Rebuild FTS using the special rebuild command
	_, err = db.ExecContext(ctx, "INSERT INTO media_fts(media_fts) VALUES('rebuild')")
	if err != nil {
		return fmt.Errorf("failed to rebuild FTS: %w", err)
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
	if err := RefreshFolderStats(db); err != nil {
		Log.Error("Failed to refresh folder_stats", "db", dbPath, "error", err)
		// Continue with FTS rebuild even if folder_stats fails
	}

	if err := RebuildFTS(db, dbPath); err != nil {
		Log.Error("Failed to rebuild FTS", "db", dbPath, "error", err)
		return err
	}

	Log.Info("Scheduled maintenance completed", "db", dbPath)
	return nil
}

// PopulateFolderStatsInGo populates folder_stats using Go path manipulation
// This is used by both maintenance and migration code
func PopulateFolderStatsInGo(ctx context.Context, db *sql.DB) error {
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

	// Insert into folder_stats
	for parent, data := range folderMap {
		depth := strings.Count(strings.ReplaceAll(parent, "\\", "/"), "/")
		if parent != "" && parent != "." {
			depth = strings.Count(strings.ReplaceAll(parent, "\\", "/"), "/")
		}
		_, err := db.ExecContext(ctx, `
			INSERT OR REPLACE INTO folder_stats (parent, depth, file_count, total_size, total_duration)
			VALUES (?, ?, ?, ?, ?)
		`, parent, depth, data.count, data.size, data.duration)
		if err != nil {
			return err
		}
	}

	return nil
}
