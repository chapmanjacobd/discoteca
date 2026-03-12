package db

import (
	"database/sql"
	"log/slog"
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
		slog.Debug("Health check: failed to open connection", "path", dbPath, "error", err)
		return false
	}
	defer db.Close()

	// Enable WAL mode to match application behavior and detect WAL corruption
	_, _ = db.Exec("PRAGMA journal_mode=WAL")

	// 1. Thorough integrity check
	rows, err := db.Query("PRAGMA integrity_check")
	if err != nil {
		slog.Debug("Health check: PRAGMA integrity_check query failed", "error", err)
		return false
	}
	defer rows.Close()

	foundOk := false
	for rows.Next() {
		var res string
		if err := rows.Scan(&res); err != nil {
			slog.Debug("Health check: failed to scan integrity row", "error", err)
			return false
		}
		if res == "ok" {
			foundOk = true
		} else {
			slog.Warn("Health check: integrity error found", "msg", res)
			return false
		}
	}
	if !foundOk {
		slog.Debug("Health check: integrity_check returned no rows")
		return false
	}

	// 2. Schema check
	row := db.QueryRow("SELECT name FROM sqlite_master LIMIT 1")
	var name string
	if err := row.Scan(&name); err != nil && err != sql.ErrNoRows {
		slog.Debug("Health check: schema check failed", "error", err)
		return false
	}

	// 3. Write check
	// We attempt to perform a real write inside a transaction and roll it back.
	// This ensures that indices and FTS triggers are actually working.
	tx, err := db.Begin()
	if err != nil {
		slog.Debug("Health check: failed to begin transaction", "error", err)
		return false
	}
	defer tx.Rollback()

	// Check for media table and perform a REAL write if possible
	var hasMedia bool
	_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media')").Scan(&hasMedia)
	if hasMedia {
		var somePath string
		_ = db.QueryRow("SELECT path FROM media LIMIT 1").Scan(&somePath)

		// If the table is not empty, we MUST update a real row to trigger the FTS and index logic.
		// If we only update a non-existent row, the triggers will not fire and we won't detect FTS corruption.
		if somePath != "" {
			if _, err = tx.Exec("UPDATE media SET time_deleted = time_deleted WHERE path = ?", somePath); err != nil {
				slog.Warn("Health check: write consistency check (media triggers) failed", "path", somePath, "error", err)
				return false
			}
		} else {
			if _, err = tx.Exec("UPDATE media SET time_deleted = time_deleted WHERE rowid = -1"); err != nil {
				slog.Warn("Health check: write consistency check (media) failed", "error", err)
				return false
			}
		}
	} else {
		// Generic write check for non-media DBs (e.g. in tests)
		if _, err = tx.Exec("CREATE TEMP TABLE _health_check(id INT); DROP TABLE _health_check;"); err != nil {
			slog.Debug("Health check: generic write check failed", "error", err)
			return false
		}
	}

	// Specifically check FTS virtual table consistency
	var hasFTS bool
	_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')").Scan(&hasFTS)
	if hasFTS {
		if _, err = tx.Exec("SELECT rowid FROM media_fts LIMIT 1"); err != nil && err != sql.ErrNoRows {
			slog.Warn("Health check: FTS check (media_fts) failed", "error", err)
			return false
		}
	}

	return true
}
