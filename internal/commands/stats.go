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

	Facet     string   `help:"Time facet for --frequency (watched, deleted, created, modified); requires --frequency" group:"Display"`
	Frequency string   `help:"Group statistics by time frequency (daily, weekly, monthly, quarterly, yearly, decadally, hourly, minutely)" short:"f" group:"Display"`
	JSON      bool     `help:"Output results as JSON" short:"j" group:"Display"`
	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *StatsCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)

	if c.Facet != "" && c.Frequency == "" {
		return fmt.Errorf("--facet %q requires --frequency", c.Facet)
	}
	if c.Frequency != "" && c.Facet == "" {
		c.Facet = "watched"
	}

	timeCol, err := c.getTimeColumn()
	if err != nil {
		return err
	}

	if !c.OnlyDeleted && c.Facet != "deleted" && !c.HideDeleted {
		c.HideDeleted = true
	}
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
	}

	for _, dbPath := range c.Databases {
		if err := c.processDatabase(ctx, dbPath, flags, timeCol); err != nil {
			return err
		}
	}
	return nil
}

func (c *StatsCmd) getTimeColumn() (string, error) {
	switch c.Facet {
	case "watched", "":
		return "time_last_played", nil
	case "deleted":
		return "time_deleted", nil
	case "created":
		return "time_created", nil
	case "modified":
		return "time_modified", nil
	default:
		return "", fmt.Errorf("invalid facet %q; expected watched, deleted, created, or modified", c.Facet)
	}
}

func (c *StatsCmd) processDatabase(
	ctx context.Context,
	dbPath string,
	flags models.GlobalFlags,
	timeCol string,
) error {
	// Ensure schema drift is handled before the read-only stats queries run.
	initDB, _, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return err
	}
	initDB.Close()

	if c.Frequency != "" {
		return c.printFrequencyStats(ctx, dbPath, flags, timeCol)
	}

	return c.printGeneralStats(ctx, dbPath, flags)
}

func (c *StatsCmd) printFrequencyStats(
	ctx context.Context,
	dbPath string,
	flags models.GlobalFlags,
	timeCol string,
) error {
	stats, err := query.HistoricalUsageFiltered(ctx, dbPath, c.Frequency, timeCol, flags)
	if err != nil {
		return err
	}

	if c.JSON {
		return utils.PrintJSON(stats)
	}

	fmt.Printf("%s media (%s) for %s:\n", utils.Title(c.Frequency), c.Frequency, dbPath)
	return PrintFrequencyStats(stats)
}

func (c *StatsCmd) printGeneralStats(ctx context.Context, dbPath string, flags models.GlobalFlags) error {
	stats, err := query.GetStatsFiltered(ctx, dbPath, flags)
	if err != nil {
		return err
	}

	typeStats, err := query.GetStatsByTypeFiltered(ctx, dbPath, flags)
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
