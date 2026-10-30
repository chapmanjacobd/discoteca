package commands

import (
	"context"
	"fmt"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/formatutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

type PrintCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.AggregateFlags   `embed:""`
	models.TextFlags        `embed:""`
	models.FTSFlags         `embed:""`

	Args []string `help:"Database file(s) or files/directories to scan" required:"true" arg:""`

	Databases []string `kong:"-"`
	ScanPaths []string `kong:"-"`
}

func (c *PrintCmd) AfterApply() error {
	var err error
	c.Databases, c.ScanPaths, err = ParseDatabaseAndScanPaths(c.Args, &c.CoreFlags, &c.MediaFilterFlags)
	return err
}

func (c *PrintCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	flags := models.BuildQueryGlobalFlags(models.BuildQueryOptions{
		Core:        c.CoreFlags,
		Query:       c.QueryFlags,
		PathFilter:  c.PathFilterFlags,
		Filter:      c.FilterFlags,
		MediaFilter: c.MediaFilterFlags,
		TimeFilter:  c.TimeFilterFlags,
		Deleted:     c.DeletedFlags,
		Sort:        c.SortFlags,
		Display:     c.DisplayFlags,
		FTS:         c.FTSFlags,
	})
	flags.AggregateFlags = c.AggregateFlags
	flags.TextFlags = c.TextFlags

	media, err := loadMedia(ctx, c.Databases, c.ScanPaths, flags)
	if err != nil {
		return err
	}

	return renderPrint(media, flags)
}

func renderPrint(media []models.MediaWithDB, flags models.GlobalFlags) error {
	HideRedundantFirstPlayed(media)

	isAggregated := flags.BigDirs || flags.GroupByExtensions || flags.GroupBySize || flags.Depth > 0 ||
		flags.Parents ||
		flags.FoldersOnly ||
		len(flags.FolderSizes) > 0 ||
		flags.FolderCounts != ""

	if flags.JSON {
		if isAggregated {
			folders := query.AggregateMedia(media, flags)
			query.SortFolders(folders, flags.SortBy, flags.Reverse)
			return PrintFolders(flags.DisplayFlags, flags.Columns, folders)
		}
		if flags.Summarize {
			summary := query.SummarizeMedia(media)
			return fileutil.PrintJSON(summary)
		}
		return PrintMedia(flags.DisplayFlags, flags.Columns, media)
	}

	if flags.Summarize {
		summary := query.SummarizeMedia(media)
		for _, s := range summary {
			fmt.Printf("%s: %d files, %s, %s\n",
				s.Label, s.Count, formatutil.FormatSize(s.TotalSize), formatutil.FormatDuration(int(s.TotalDuration)))
		}
		if !isAggregated {
			fmt.Println()
		}
	}

	if isAggregated {
		folders := query.AggregateMedia(media, flags)
		query.SortFolders(folders, flags.SortBy, flags.Reverse)
		return PrintFolders(flags.DisplayFlags, flags.Columns, folders)
	}

	if flags.RegexSort {
		media = query.RegexSortMedia(media, flags)
		query.ApplyReRank(media, flags)
	} else {
		query.SortMedia(media, flags)
	}

	return PrintMedia(flags.DisplayFlags, flags.Columns, media)
}
