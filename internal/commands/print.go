package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/query"
	"github.com/chapmanjacobd/discotheque/internal/utils"
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

	Args []string `arg:"" required:"" help:"Database file(s) or files/directories to scan"`

	Databases []string `kong:"-"`
	ScanPaths []string `kong:"-"`
}

func (c *PrintCmd) AfterApply() error {
	if err := c.CoreFlags.AfterApply(); err != nil {
		return err
	}
	if err := c.MediaFilterFlags.AfterApply(); err != nil {
		return err
	}
	for _, arg := range c.Args {
		if strings.HasSuffix(arg, ".db") && utils.IsSQLite(arg) {
			c.Databases = append(c.Databases, arg)
		} else {
			c.ScanPaths = append(c.ScanPaths, arg)
		}
	}
	return nil
}

func (c *PrintCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		QueryFlags:       c.QueryFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		SortFlags:        c.SortFlags,
		DisplayFlags:     c.DisplayFlags,
		AggregateFlags:   c.AggregateFlags,
		TextFlags:        c.TextFlags,
		FTSFlags:         c.FTSFlags,
	}

	var allMedia []models.MediaWithDB

	// Handle databases
	if len(c.Databases) > 0 {
		dbMedia, err := query.MediaQuery(context.Background(), c.Databases, flags)
		if err != nil {
			return err
		}
		allMedia = append(allMedia, dbMedia...)
	}

	// Handle scan paths
	if len(c.ScanPaths) > 0 {
		// (Scanning logic for print if needed, usually it just queries DBs)
	}

	media := query.FilterMedia(allMedia, flags)
	HideRedundantFirstPlayed(media)

	isAggregated := c.BigDirs || c.GroupByExtensions || c.GroupByMimeTypes || c.GroupBySize || c.Depth > 0 || c.Parents || c.FoldersOnly || len(c.FolderSizes) > 0 || c.FolderCounts != ""

	if c.JSON {
		if isAggregated {
			folders := query.AggregateMedia(media, flags)
			query.SortFolders(folders, c.SortBy, c.Reverse)
			return PrintFolders(c.DisplayFlags, c.Columns, folders)
		}
		if c.Summarize {
			summary := query.SummarizeMedia(media)
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(summary)
		}
		return PrintMedia(c.DisplayFlags, c.Columns, media)
	}

	if c.Summarize {
		summary := query.SummarizeMedia(media)
		for _, s := range summary {
			fmt.Printf("%s: %d files, %s, %s\n",
				s.Label, s.Count, utils.FormatSize(s.TotalSize), utils.FormatDuration(int(s.TotalDuration)))
		}
		if !isAggregated {
			fmt.Println()
		}
	}

	if isAggregated {
		folders := query.AggregateMedia(media, flags)
		query.SortFolders(folders, c.SortBy, c.Reverse)
		return PrintFolders(c.DisplayFlags, c.Columns, folders)
	}

	if c.RegexSort {
		media = query.RegexSortMedia(media, flags)
	} else {
		query.SortMedia(media, flags)
	}

	return PrintMedia(c.DisplayFlags, c.Columns, media)
}
