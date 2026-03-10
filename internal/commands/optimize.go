package commands

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/utils"
)

type OptimizeCmd struct {
	models.CoreFlags `embed:""`
	Databases        []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

func (c *OptimizeCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	for _, dbPath := range c.Databases {
		slog.Info("Optimizing database", "path", dbPath)
		sqlDB, err := db.Connect(dbPath)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		slog.Info("Running VACUUM...")
		if _, err := sqlDB.Exec("VACUUM"); err != nil {
			return fmt.Errorf("VACUUM failed on %s: %w", dbPath, err)
		}

		slog.Info("Running ANALYZE...")
		if _, err := sqlDB.Exec("ANALYZE"); err != nil {
			return fmt.Errorf("ANALYZE failed on %s: %w", dbPath, err)
		}

		slog.Info("Optimizing FTS index...")
		// FTS5 optimize command
		if _, err := sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('optimize')"); err != nil {
			slog.Warn("FTS optimize failed (maybe table doesn't exist?)", "path", dbPath, "error", err)
		}

		slog.Info("Optimization complete", "path", dbPath)
	}
	return nil
}

type SampleHashCmd struct {
	models.CoreFlags    `embed:""`
	models.HashingFlags `embed:""`
	models.DisplayFlags `embed:""`
	Paths               []string `arg:"" required:"" help:"Files to hash" type:"existingfile"`
}

func (c *SampleHashCmd) Run(ctx *kong.Context) error {
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
		return json.NewEncoder(os.Stdout).Encode(results)
	}
	return nil
}
