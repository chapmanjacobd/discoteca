package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
)

var repairLocks sync.Map

func getLock(path string) *sync.Mutex {
	v, _ := repairLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// IsCorruptionError checks if the error is a database corruption error
func IsCorruptionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "database disk image is malformed")
}

func isHealthy(dbPath string) bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}

	// Use sql.Open directly instead of Connect to avoid connection pool deadlocks.
	// isHealthy needs to be able to open its own connection regardless of global pool limits.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		Log.Debug("Health check: failed to open connection", "path", dbPath, "error", err)
		return false
	}
	defer db.Close()

	// Enable WAL mode to match application behavior and detect WAL corruption
	_, _ = db.ExecContext(context.Background(), "PRAGMA journal_mode=WAL")

	// 1. Thorough integrity check
	rows, err := db.QueryContext(context.Background(), "PRAGMA integrity_check")
	if err != nil {
		Log.Debug("Health check: PRAGMA integrity_check query failed", "error", err)
		return false
	}
	defer rows.Close()

	foundOk := false
	for rows.Next() {
		var res string
		if err := rows.Scan(&res); err != nil {
			Log.Debug("Health check: failed to scan integrity row", "error", err)
			return false
		}
		if res == "ok" {
			foundOk = true
		} else {
			Log.Warn("Health check: integrity error found", "msg", res)
			return false
		}
	}
	if err := rows.Err(); err != nil {
		Log.Debug("Health check: rows iteration error", "error", err)
		return false
	}
	if !foundOk {
		Log.Debug("Health check: integrity_check returned no rows")
		return false
	}

	// 2. Schema check
	row := db.QueryRowContext(context.Background(), "SELECT name FROM sqlite_master LIMIT 1")
	var name string
	if err := row.Scan(&name); err != nil && err != sql.ErrNoRows {
		Log.Debug("Health check: schema check failed", "error", err)
		return false
	}

	// 3. Write check
	// We attempt to perform a real write inside a transaction and roll it back.
	// This ensures that indices and FTS triggers are actually working.
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		Log.Debug("Health check: failed to begin transaction", "error", err)
		return false
	}
	defer tx.Rollback()

	// Check for media table and perform a REAL write if possible
	var hasMedia bool
	_ = db.QueryRowContext(context.Background(), "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media')").
		Scan(&hasMedia)
	if hasMedia {
		var somePath string
		_ = db.QueryRowContext(context.Background(), "SELECT path FROM media LIMIT 1").Scan(&somePath)

		// If the table is not empty, we MUST update a real row to trigger the FTS and index logic.
		// If we only update a non-existent row, the triggers will not fire and we won't detect FTS corruption.
		if somePath != "" {
			if _, err = tx.ExecContext(
				context.Background(),
				"UPDATE media SET time_deleted = time_deleted WHERE path = ?",
				somePath,
			); err != nil {
				Log.Warn(
					"Health check: write consistency check (media triggers) failed",
					"path",
					somePath,
					"error",
					err,
				)
				return false
			}
		} else {
			if _, err = tx.ExecContext(
				context.Background(),
				"UPDATE media SET time_deleted = time_deleted WHERE rowid = -1",
			); err != nil {
				Log.Warn("Health check: write consistency check (media) failed", "error", err)
				return false
			}
		}
	} else {
		// Generic write check for non-media DBs (e.g. in tests)
		if _, err = tx.ExecContext(
			context.Background(),
			"CREATE TEMP TABLE _health_check(id INT); DROP TABLE _health_check;",
		); err != nil {
			Log.Debug("Health check: generic write check failed", "error", err)
			return false
		}
	}

	// Specifically check FTS virtual table consistency
	var hasFTS bool
	_ = db.QueryRowContext(context.Background(), "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')").
		Scan(&hasFTS)
	if hasFTS {
		if _, err = tx.ExecContext(
			context.Background(),
			"SELECT rowid FROM media_fts LIMIT 1",
		); err != nil &&
			!errors.Is(err, sql.ErrNoRows) {

			Log.Warn("Health check: FTS check (media_fts) failed", "error", err)
			return false
		}
	}

	return true
}
