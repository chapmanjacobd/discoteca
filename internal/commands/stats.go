package commands

import (
	"context"
	"fmt"

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

	Facet     string   `help:"One of: watched, deleted, created, modified" required:"" arg:""`
	Databases []string `help:"SQLite database files"                       required:"" arg:"" type:"existingfile"`
}

func (c *StatsCmd) Run(ctx context.Context) error {
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
		sqlDB, queries, err := db.ConnectWithInit(dbPath)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		if c.Frequency != "" {
			stats, err := query.HistoricalUsage(ctx, dbPath, c.Frequency, timeCol)
			if err != nil {
				return err
			}

			if c.JSON {
				if err := utils.PrintJSON(stats); err != nil {
					return err
				}
				continue
			}

			fmt.Printf("%s media (%s) for %s:\n", utils.Title(c.Facet), c.Frequency, dbPath)
			if err := PrintFrequencyStats(stats); err != nil {
				return err
			}
			continue
		}

		stats, err := queries.GetStats(ctx)
		if err != nil {
			return err
		}

		typeStats, err := queries.GetStatsByType(ctx)
		if err != nil {
			return err
		}

		if c.JSON {
			result := map[string]any{
				"database":  dbPath,
				"summary":   stats,
				"breakdown": typeStats,
			}
			if err := utils.PrintJSON(result); err != nil {
				return err
			}
			continue
		}

		fmt.Printf("Statistics for %s:\n", dbPath)
		fmt.Printf("  Total Files:      %d\n", stats.TotalCount)
		fmt.Printf("  Total Size:       %s\n", utils.FormatSize(utils.GetInt64(stats.TotalSize)))
		fmt.Printf("  Total Duration:   %s\n", utils.FormatDuration(int(utils.GetInt64(stats.TotalDuration))))
		fmt.Printf("  Watched Files:    %d\n", stats.WatchedCount)
		fmt.Printf("  Unwatched Files:  %d\n", stats.UnwatchedCount)

		if len(typeStats) > 0 {
			fmt.Println("\n  Breakdown by MediaType:")
			for _, ts := range typeStats {
				t := "unknown"
				if ts.MediaType.Valid {
					t = ts.MediaType.String
				}
				fmt.Printf("    %-10s: %d files, %s, %s\n",
					t, ts.Count,
					utils.FormatSize(utils.GetInt64(ts.TotalSize)),
					utils.FormatDuration(int(utils.GetInt64(ts.TotalDuration))))
			}
		}
		fmt.Println()
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
