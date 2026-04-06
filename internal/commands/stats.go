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

	Facet     string   `help:"One of: watched, deleted, created, modified" required:"true" arg:""`
	Databases []string `help:"SQLite database files"                       required:"true" arg:"" type:"existingfile"`
}

func (c *StatsCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)

	timeCol := c.getTimeColumn()

	for _, dbPath := range c.Databases {
		if err := c.processDatabase(ctx, dbPath, timeCol); err != nil {
			return err
		}
	}
	return nil
}

func (c *StatsCmd) getTimeColumn() string {
	switch c.Facet {
	case "deleted":
		return "time_deleted"
	case "created":
		return "time_created"
	case "modified":
		return "time_modified"
	default:
		return "time_last_played"
	}
}

func (c *StatsCmd) processDatabase(ctx context.Context, dbPath, timeCol string) error {
	sqlDB, queries, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if c.Frequency != "" {
		return c.printFrequencyStats(ctx, dbPath, timeCol)
	}

	return c.printGeneralStats(ctx, dbPath, queries)
}

func (c *StatsCmd) printFrequencyStats(ctx context.Context, dbPath, timeCol string) error {
	stats, err := query.HistoricalUsage(ctx, dbPath, c.Frequency, timeCol)
	if err != nil {
		return err
	}

	if c.JSON {
		return utils.PrintJSON(stats)
	}

	fmt.Printf("%s media (%s) for %s:\n", utils.Title(c.Frequency), c.Frequency, dbPath)
	return PrintFrequencyStats(stats)
}

func (c *StatsCmd) printGeneralStats(ctx context.Context, dbPath string, queries *db.Queries) error {
	stats, err := queries.GetStats(ctx)
	if err != nil {
		return err
	}

	typeStats, err := queries.GetStatsByType(ctx)
	if err != nil {
		return err
	}

	if c.JSON {
		return utils.PrintJSON(map[string]any{
			"database":  dbPath,
			"summary":   stats,
			"breakdown": typeStats,
		})
	}

	printGeneralStatsOutput(dbPath, stats, typeStats)
	return nil
}

func printGeneralStatsOutput(dbPath string, stats db.GetStatsRow, typeStats []db.GetStatsByTypeRow) {
	fmt.Printf("Statistics for %s:\n", dbPath)
	fmt.Printf("  Total Files:      %d\n", stats.TotalCount)
	fmt.Printf("  Total Size:       %s\n", utils.FormatSize(utils.GetInt64(stats.TotalSize)))
	fmt.Printf("  Total Duration:   %s\n", utils.FormatDuration(int(utils.GetInt64(stats.TotalDuration))))
	fmt.Printf("  Watched Files:    %d\n", stats.WatchedCount)
	fmt.Printf("  Unwatched Files:  %d\n", stats.UnwatchedCount)

	if len(typeStats) > 0 {
		printTypeBreakdown(typeStats)
	}
	fmt.Println()
}

func printTypeBreakdown(typeStats []db.GetStatsByTypeRow) {
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

func PrintFrequencyStats(stats []query.FrequencyStats) error {
	fmt.Printf("%-15s\t%-10s\t%-10s\t%-15s\n", "Period", "Count", "Size", "Duration")
	for _, s := range stats {
		fmt.Printf("%-15s\t%-10d\t%-10s\t%-15s\n",
			s.Label, s.Count, utils.FormatSize(s.TotalSize), utils.FormatDuration(int(s.TotalDuration)))
	}
	return nil
}
