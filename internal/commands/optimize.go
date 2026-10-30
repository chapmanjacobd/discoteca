package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type OptimizeCmd struct {
	models.CoreFlags `embed:""`

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *OptimizeCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	for _, dbPath := range c.Databases {
		if c.Simulate {
			fmt.Printf(
				"[Simulate] would VACUUM, ANALYZE, optimize media_fts, and backfill NULL media types in %s\n",
				dbPath,
			)
			continue
		}

		logging.Log.Info("Optimizing database", "path", dbPath)
		sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		logging.Log.Info("Running VACUUM...")
		if _, err := sqlDB.ExecContext(ctx, "VACUUM"); err != nil {
			return fmt.Errorf("VACUUM failed on %s: %w", dbPath, err)
		}

		logging.Log.Info("Running ANALYZE...")
		if _, err := sqlDB.ExecContext(ctx, "ANALYZE"); err != nil {
			return fmt.Errorf("ANALYZE failed on %s: %w", dbPath, err)
		}

		logging.Log.Info("Optimizing FTS indexes...")
		// FTS5 optimize commands
		if _, err := sqlDB.ExecContext(ctx, "INSERT INTO media_fts(media_fts) VALUES('optimize')"); err != nil {
			logging.Log.Warn("media_fts optimize failed (maybe table doesn't exist?)", "path", dbPath, "error", err)
		}
		if _, err := sqlDB.ExecContext(ctx, "INSERT INTO captions_fts(captions_fts) VALUES('optimize')"); err != nil {
			logging.Log.Warn("captions_fts optimize failed (maybe table doesn't exist?)", "path", dbPath, "error", err)
		}

		if err := c.BackfillMediaTypes(ctx, sqlDB); err != nil {
			logging.Log.Warn("BackfillMediaTypes failed", "path", dbPath, "error", err)
		}

		logging.Log.Info("Optimization complete", "path", dbPath)
	}
	return nil
}

// BackfillMediaTypes fills media_type for rows where it is NULL, based on the
// file extension. It never overwrites an existing media_type.
func (c *OptimizeCmd) BackfillMediaTypes(ctx context.Context, sqlDB *sql.DB) error {
	logging.Log.Info("Backfilling NULL media types...")
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, "SELECT path FROM media WHERE media_type IS NULL")
	if err != nil {
		return err
	}
	defer rows.Close()

	var updates []struct {
		path  string
		mtype string
	}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		mtype := ""
		if media.VideoExtensionMap[ext] {
			mtype = "video"
		} else if media.AudioExtensionMap[ext] {
			mtype = "audio"
		} else if media.ImageExtensionMap[ext] {
			mtype = "image"
		} else if media.TextExtensionMap[ext] || media.ComicExtensionMap[ext] {
			mtype = "text"
		}

		if mtype != "" {
			updates = append(updates, struct {
				path  string
				mtype string
			}{path, mtype})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(updates) > 0 {
		stmt, err := tx.PrepareContext(ctx, "UPDATE media SET media_type = ? WHERE path = ?")
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, u := range updates {
			if _, err := stmt.ExecContext(ctx, u.mtype, u.path); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

type SampleHashCmd struct {
	models.CoreFlags    `embed:""`
	models.HashingFlags `embed:""`

	JSON bool `help:"Output results as JSON" short:"j" group:"Display"`

	Paths []string `help:"Files to hash" required:"true" arg:"" type:"existingfile"`
}

func (c *SampleHashCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		HashingFlags: c.HashingFlags,
	}

	type result struct {
		Path string `json:"path"`
		Hash string `json:"hash"`
	}
	var results []result

	for _, path := range c.Paths {
		h, err := fileutil.SampleHashFile(path, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error hashing %s: %v\n", path, err)
			continue
		}
		if c.JSON {
			results = append(results, result{Path: path, Hash: h})
		} else {
			fmt.Printf("%s\t%s\n", h, path)
		}
	}

	if c.JSON {
		return fileutil.PrintJSON(results)
	}
	return nil
}
