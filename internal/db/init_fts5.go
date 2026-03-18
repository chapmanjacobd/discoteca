package db

import (
	"database/sql"
	"fmt"

	"github.com/chapmanjacobd/discoteca/internal/db/schema"
)

func InitDB(sqlDB *sql.DB) error {
	// 1. Create Core Tables (media, playlists, history, meta)
	// This does NOT include captions or FTS tables
	if _, err := sqlDB.Exec(schema.GetCoreTables()); err != nil {
		return fmt.Errorf("failed to create core tables: %w", err)
	}

	// 2. Migrate (Ensure columns exist, strict mode, etc)
	if err := Migrate(sqlDB); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// 3. Create Core Indexes
	if _, err := sqlDB.Exec(schema.GetCoreIndexes()); err != nil {
		return fmt.Errorf("failed to create core indexes: %w", err)
	}

	// 4. Create Captions table (ONLY if enabled)
	if IsFtsEnabled() {
		// Create captions table if not exists
		if _, err := sqlDB.Exec(schema.GetCaptionsTable()); err != nil {
			return fmt.Errorf("failed to create captions table: %w", err)
		}
	}

	// 5. Create FTS tables (ONLY if enabled)
	if IsFtsEnabled() {
		// Create FTS tables (media_fts, captions_fts)
		// We do this LAST because triggers might depend on columns added/renamed during Migrate
		if _, err := sqlDB.Exec(schema.GetFTSTables()); err != nil {
			return fmt.Errorf("failed to create fts tables: %w", err)
		}
	}

	return nil
}
