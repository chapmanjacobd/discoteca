package query

import (
	"context"
	"database/sql"

	database "github.com/chapmanjacobd/discoteca/internal/db"
)

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
