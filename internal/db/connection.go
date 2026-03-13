package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a SQLite database and applies performance tuning PRAGMAs
func Connect(dbPath string) (*sql.DB, error) {
	// Add busy timeout and immediate locking to handle concurrent writes better
	dsn := fmt.Sprintf("%s?_busy_timeout=30000&_txlock=immediate", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Performance Tuning
	tuning := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA foreign_keys=ON",
		"PRAGMA mmap_size=268435456",
	}

	for _, pragma := range tuning {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to apply pragma %q: %w", pragma, err)
		}
	}

	return db, nil
}
