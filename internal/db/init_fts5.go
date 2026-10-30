package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/chapmanjacobd/discoteca/internal/db/schema"
)

func InitDB(ctx context.Context, sqlDB *sql.DB) error {
	if err := requireStrictSupport(ctx, sqlDB); err != nil {
		return err
	}

	// 1. Create Core Tables (media, playlists, history, meta)
	// This does NOT include captions or FTS tables
	if _, err := sqlDB.ExecContext(ctx, schema.GetCoreTables()); err != nil {
		return fmt.Errorf("failed to create core tables: %w", err)
	}

	// 2. Migrate (Ensure columns exist, strict mode, etc)
	if err := Migrate(ctx, sqlDB); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// 3. Create Core Indexes
	if _, err := sqlDB.ExecContext(ctx, schema.GetCoreIndexes()); err != nil {
		return fmt.Errorf("failed to create core indexes: %w", err)
	}

	// 4. Create Captions table (ONLY if enabled)
	if IsFtsEnabled() {
		// Create captions table if not exists
		if _, err := sqlDB.ExecContext(ctx, schema.GetCaptionsTable()); err != nil {
			return fmt.Errorf("failed to create captions table: %w", err)
		}
	}

	// 5. Create FTS tables (ONLY if enabled)
	if IsFtsEnabled() {
		// Remember whether the indexes already existed. A newly created FTS
		// index starts empty and must be rebuilt from the content tables when
		// this is an existing database.
		mediaFTSExisted, err := tableExists(ctx, sqlDB, "media_fts")
		if err != nil {
			return err
		}
		captionsFTSExisted, err := tableExists(ctx, sqlDB, "captions_fts")
		if err != nil {
			return err
		}

		// Create FTS tables (media_fts, captions_fts)
		// We do this LAST because triggers might depend on columns added/renamed during Migrate
		if _, err := sqlDB.ExecContext(ctx, schema.GetFTSTables()); err != nil {
			return fmt.Errorf("failed to create fts tables: %w", err)
		}

		if err := ensureMediaFTSTriggers(ctx, sqlDB); err != nil {
			return err
		}
		if err := ensureCaptionsFTSTriggers(ctx, sqlDB); err != nil {
			return err
		}

		if !mediaFTSExisted {
			if err := rebuildFTSData(ctx, sqlDB, "media_fts"); err != nil {
				return err
			}
		}
		if !captionsFTSExisted {
			if err := rebuildFTSData(ctx, sqlDB, "captions_fts"); err != nil {
				return err
			}
		}
	}

	return nil
}
