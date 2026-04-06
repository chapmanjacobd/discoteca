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
