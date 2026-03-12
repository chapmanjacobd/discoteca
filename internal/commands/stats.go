package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type StatsCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.DisplayFlags     `embed:""`

	Facet     string   `arg:"" required:"" help:"One of: watched, deleted, created, modified"`
	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

func (c *StatsCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		DisplayFlags:     c.DisplayFlags,
	}

	timeCol := "time_last_played"
	switch c.Facet {
	case "deleted":
		timeCol = "time_deleted"
		flags.MarkDeleted = true // Ensure we don't hide deleted in query
	case "created":
		timeCol = "time_created"
	case "modified":
		timeCol = "time_modified"
	}

	for _, dbPath := range c.Databases {
		sqlDB, err := db.Connect(dbPath)
		if err != nil {
			return err
		}

		if err := db.InitDB(sqlDB); err != nil {
			sqlDB.Close()
			return fmt.Errorf("failed to initialize database %s: %w", dbPath, err)
		}

		if c.Frequency != "" {
			stats, err := query.HistoricalUsage(context.Background(), dbPath, c.Frequency, timeCol)
			if err != nil {
				sqlDB.Close()
				return err
			}

			if c.JSON {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(stats); err != nil {
					sqlDB.Close()
					return err
				}
				sqlDB.Close()
				continue
			}

			fmt.Printf("%s media (%s) for %s:\n", utils.Title(c.Facet), c.Frequency, dbPath)
			if err := PrintFrequencyStats(stats); err != nil {
				sqlDB.Close()
				return err
			}
			sqlDB.Close()
			continue
		}

		queries := db.New(sqlDB)
		stats, err := queries.GetStats(context.Background())
		if err != nil {
			sqlDB.Close()
			return err
		}

		typeStats, err := queries.GetStatsByType(context.Background())
		if err != nil {
			sqlDB.Close()
			return err
		}

		if c.JSON {
			result := map[string]any{
				"database":  dbPath,
				"summary":   stats,
				"breakdown": typeStats,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				sqlDB.Close()
				return err
			}
			sqlDB.Close()
			continue
		}

		fmt.Printf("Statistics for %s:\n", dbPath)
		fmt.Printf("  Total Files:      %d\n", stats.TotalCount)
		fmt.Printf("  Total Size:       %s\n", utils.FormatSize(utils.GetInt64(stats.TotalSize)))
		fmt.Printf("  Total Duration:   %s\n", utils.FormatDuration(int(utils.GetInt64(stats.TotalDuration))))
		fmt.Printf("  Watched Files:    %d\n", stats.WatchedCount)
		fmt.Printf("  Unwatched Files:  %d\n", stats.UnwatchedCount)

		if len(typeStats) > 0 {
			fmt.Println("\n  Breakdown by Type:")
			for _, ts := range typeStats {
				t := "unknown"
				if ts.Type.Valid {
					t = ts.Type.String
				}
				fmt.Printf("    %-10s: %d files, %s, %s\n",
					t, ts.Count,
					utils.FormatSize(utils.GetInt64(ts.TotalSize)),
					utils.FormatDuration(int(utils.GetInt64(ts.TotalDuration))))
			}
		}
		fmt.Println()
		sqlDB.Close()
	}
	return nil
}

func PrintFrequencyStats(stats []query.FrequencyStats) error {
	fmt.Printf("%-15s\t%-10s\t%-10s\t%-15s\n", "Period", "Count", "Size", "Duration")
	for _, s := range stats {
		fmt.Printf("%-15s\t%-10d\t%-10s\t%-15s\n",
			s.Label, s.Count, utils.FormatSize(s.TotalSize), utils.FormatDuration(int(s.TotalDuration)))
	}
	return nil
}
