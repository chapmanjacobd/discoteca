package query

import (
	"context"
	"database/sql"

	database "github.com/chapmanjacobd/discoteca/internal/db"
)

// SearchMode represents the search backend being used
type SearchMode int

const (
	SearchModeUnknown   SearchMode = iota
	SearchModeSubstring            // LIKE-based search
	SearchModeFTS5                 // SQLite FTS5
)

func (s SearchMode) String() string {
	switch s {
	case SearchModeUnknown:
		return "Unknown"
	case SearchModeSubstring:
		return "Substring"
	case SearchModeFTS5:
		return "FTS5"
	}
	return "Unknown"
}

// DetectSearchMode detects the best available search backend for one database.
// Detection is intentionally per call: a process-global cache would leak the
// first database's capabilities into every later query.
func DetectSearchMode(ctx context.Context, db *sql.DB) SearchMode {
	if database.FtsEnabled && db != nil && hasFTS5Table(ctx, db) {
		return SearchModeFTS5
	}
	return SearchModeSubstring
}

// hasFTS5Table checks if the media_fts table exists in the database
func hasFTS5Table(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		return false
	}

	var exists bool
	err := db.QueryRowContext(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')",
	).Scan(&exists)
	return err == nil && exists
}

// IsSearchAvailable checks if a specific search mode is available
func IsSearchAvailable(ctx context.Context, mode SearchMode, db *sql.DB) bool {
	switch mode {
	case SearchModeUnknown:
		return false
	case SearchModeFTS5:
		return db != nil && hasFTS5Table(ctx, db)
	case SearchModeSubstring:
		return true // Always available
	}
	return false
}

// DatabasesHaveFTS reports whether every database in the list has a usable
// media_fts table. Mixed lists fall back to substring search for all databases
// so one database's capabilities never change another's query.
func DatabasesHaveFTS(ctx context.Context, dbPaths []string) bool {
	if len(dbPaths) == 0 {
		return false
	}
	for _, dbPath := range dbPaths {
		found := false
		err := func() error {
			sqlDB, err := database.Connect(ctx, dbPath)
			if err != nil {
				return err
			}
			defer sqlDB.Close()
			found = hasFTS5Table(ctx, sqlDB)
			return nil
		}()
		if err != nil || !found {
			return false
		}
	}
	return true
}
