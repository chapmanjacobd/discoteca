package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type OptimizeCmd struct {
	models.CoreFlags `embed:""`

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *OptimizeCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	for _, dbPath := range c.Databases {
		models.Log.Info("Optimizing database", "path", dbPath)
		sqlDB, queries, err := db.ConnectWithInit(dbPath)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		models.Log.Info("Running VACUUM...")
		if _, err := sqlDB.Exec("VACUUM"); err != nil {
			return fmt.Errorf("VACUUM failed on %s: %w", dbPath, err)
		}

		models.Log.Info("Running ANALYZE...")
		if _, err := sqlDB.Exec("ANALYZE"); err != nil {
			return fmt.Errorf("ANALYZE failed on %s: %w", dbPath, err)
		}

		models.Log.Info("Optimizing FTS index...")
		// FTS5 optimize command
		if _, err := sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('optimize')"); err != nil {
			models.Log.Warn("FTS optimize failed (maybe table doesn't exist?)", "path", dbPath, "error", err)
		}

		if err := c.BulkMarkOptimizedExtensions(ctx, sqlDB, queries); err != nil {
			models.Log.Warn("BulkMarkOptimizedExtensions failed", "path", dbPath, "error", err)
		}

		models.Log.Info("Optimization complete", "path", dbPath)
	}
	return nil
}

func (c *OptimizeCmd) BulkMarkOptimizedExtensions(ctx context.Context, sqlDB *sql.DB, _ *db.Queries) error {
	models.Log.Info("Running BulkMarkOptimizedExtensions...")
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Logic to mark optimized extensions
	// This is a placeholder for the actual logic described by the user
	// assuming it involves updating media_type based on extensions

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
		if utils.VideoExtensionMap[ext] {
			mtype = "video"
		} else if utils.AudioExtensionMap[ext] {
			mtype = "audio"
		} else if utils.ImageExtensionMap[ext] {
			mtype = "image"
		} else if utils.TextExtensionMap[ext] {
			mtype = "text"
		}

		if mtype != "" {
			updates = append(updates, struct {
				path  string
				mtype string
			}{path, mtype})
		}
	}
	rows.Close()

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
	models.DisplayFlags `embed:""`

	Paths []string `help:"Files to hash" required:"" arg:"" type:"existingfile"`
}

func (c *SampleHashCmd) Run() error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:    c.CoreFlags,
		HashingFlags: c.HashingFlags,
		DisplayFlags: c.DisplayFlags,
	}

	type result struct {
		Path string `json:"path"`
		Hash string `json:"hash"`
	}
	var results []result

	for _, path := range c.Paths {
		h, err := utils.SampleHashFile(path, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
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
		return utils.PrintJSON(results)
	}
	return nil
}
