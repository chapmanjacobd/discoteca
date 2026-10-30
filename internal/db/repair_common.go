package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var repairLocks sync.Map

// backupDirName returns a collision-resistant directory name for a corrupted
// database backup. Two repairs starting in the same second must not share a
// directory.
func backupDirName(dbPath string) string {
	return fmt.Sprintf("%s.corrupt.%d.%d", dbPath, time.Now().UnixNano(), os.Getpid())
}

func getLock(path string) *sync.Mutex {
	v, _ := repairLocks.LoadOrStore(path, &sync.Mutex{})
	if mu, ok := v.(*sync.Mutex); ok {
		return mu
	}
	return nil
}

// IsCorruptionError checks if the error is a database corruption error
func IsCorruptionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "database disk image is malformed")
}

func IsHealthy(ctx context.Context, dbPath string) bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		Log.Debug("Health check: failed to open connection", "path", dbPath, "error", err)
		return false
	}
	defer db.Close()

	_, _ = db.ExecContext(ctx, "PRAGMA journal_mode=WAL")

	if !checkIntegrity(ctx, db) {
		return false
	}

	if !checkSchema(ctx, db) {
		return false
	}

	if !checkWriteConsistency(ctx, db) {
		return false
	}

	if !checkFTSConsistency(ctx, db) {
		return false
	}

	return true
}

func checkIntegrity(ctx context.Context, db *sql.DB) bool {
	rows, err := db.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		Log.Debug("Health check: PRAGMA integrity_check query failed", "error", err)
		return false
	}
	defer rows.Close()

	foundOk := false
	for rows.Next() {
		var res string
		if scanErr := rows.Scan(&res); scanErr != nil {
			Log.Debug("Health check: failed to scan integrity row", "error", scanErr)
			return false
		}
		if res != "ok" {
			Log.Warn("Health check: integrity error found", "msg", res)
			return false
		}
		foundOk = true
	}
	if err2 := rows.Err(); err2 != nil {
		Log.Debug("Health check: rows iteration error", "error", err2)
		return false
	}
	if !foundOk {
		Log.Debug("Health check: integrity_check returned no rows")
		return false
	}
	return true
}

func checkSchema(ctx context.Context, db *sql.DB) bool {
	row := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master LIMIT 1")
	var name string
	if scanErr := row.Scan(&name); scanErr != nil && scanErr != sql.ErrNoRows {
		Log.Debug("Health check: schema check failed", "error", scanErr)
		return false
	}
	return true
}

func checkWriteConsistency(ctx context.Context, db *sql.DB) bool {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		Log.Debug("Health check: failed to begin transaction", "error", err)
		return false
	}
	defer func() { _ = tx.Rollback() }()

	var hasMedia bool
	_ = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media')").
		Scan(&hasMedia)
	if hasMedia {
		return checkMediaWrite(ctx, tx, db)
	}

	// Generic write check for non-media DBs
	if _, err = tx.ExecContext(
		ctx,
		"CREATE TEMP TABLE _health_check(id INT); DROP TABLE _health_check;",
	); err != nil {
		Log.Debug("Health check: generic write check failed", "error", err)
		return false
	}
	return true
}

func checkMediaWrite(ctx context.Context, tx *sql.Tx, db *sql.DB) bool {
	var somePath string
	_ = db.QueryRowContext(ctx, "SELECT path FROM media LIMIT 1").Scan(&somePath)

	var err error
	if somePath != "" {
		_, err = tx.ExecContext(
			ctx,
			"UPDATE media SET time_deleted = time_deleted WHERE path = ?",
			somePath,
		)
		if err != nil {
			Log.Warn("Health check: write consistency check (media triggers) failed", "path", somePath, "error", err)
			return false
		}
	} else {
		_, err = tx.ExecContext(ctx, "UPDATE media SET time_deleted = time_deleted WHERE rowid = -1")
		if err != nil {
			Log.Warn("Health check: write consistency check (media) failed", "error", err)
			return false
		}
	}
	return true
}

func checkFTSConsistency(ctx context.Context, db *sql.DB) bool {
	var hasFTS bool
	_ = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')").
		Scan(&hasFTS)
	if hasFTS {
		if _, err := db.ExecContext(ctx, "SELECT rowid FROM media_fts LIMIT 1"); err != nil &&
			!errors.Is(err, sql.ErrNoRows) {

			Log.Warn("Health check: FTS check (media_fts) failed", "error", err)
			return false
		}
	}
	return true
}

func rebuildFTSTableIfExists(ctx context.Context, db *sql.DB, tableName string) {
	var exists bool
	_ = db.QueryRowContext(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)",
		tableName,
	).Scan(&exists)
	if !exists {
		return
	}
	query := fmt.Sprintf("INSERT INTO %s(%s) VALUES('rebuild');", tableName, tableName)
	if _, err := db.ExecContext(ctx, query); err != nil {
		Log.Warn("FTS rebuild failed", "table", tableName, "error", err)
	}
}
