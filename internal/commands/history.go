package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/chapmanjacobd/discoteca/internal/history"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type HistoryCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.PostActionFlags  `embed:""`

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *HistoryCmd) Run(ctx context.Context) error {
	flags := c.buildFlags()

	return RunQuery(ctx, c.Databases, flags, func(media []models.MediaWithDB) error {
		HideRedundantFirstPlayed(media)

		if flags.JSON {
			return utils.PrintJSON(media)
		}

		c.printHeader(flags)

		if flags.DeleteRows {
			return c.handleDeleteRows(ctx, media)
		}

		if flags.Partial != "" {
			query.SortHistory(media, flags.Partial, flags.Reverse)
		} else {
			query.SortMedia(media, flags)
		}
		return PrintMedia(flags.DisplayFlags, flags.Columns, media)
	})
}

func (c *HistoryCmd) buildFlags() models.GlobalFlags {
	flags := models.BuildQueryGlobalFlags(models.BuildQueryOptions{
		Core:        c.CoreFlags,
		Query:       models.QueryFlags{},
		PathFilter:  c.PathFilterFlags,
		Filter:      c.FilterFlags,
		MediaFilter: c.MediaFilterFlags,
		TimeFilter:  c.TimeFilterFlags,
		Deleted:     c.DeletedFlags,
		Sort:        c.SortFlags,
		Display:     c.DisplayFlags,
		FTS:         models.FTSFlags{},
	})
	flags.PostActionFlags = c.PostActionFlags
	// Set default sort for history
	if flags.SortBy == "path" || flags.SortBy == "" {
		flags.SortBy = "time_last_played"
		flags.Reverse = true
	}

	// Filter for only watched items if not otherwise specified
	if flags.Watched == nil && !flags.InProgress && !flags.Completed {
		watched := true
		flags.Watched = &watched
	}
	return flags
}

func (c *HistoryCmd) printHeader(flags models.GlobalFlags) {
	if flags.Completed {
		fmt.Println("Completed:")
	} else if flags.InProgress {
		fmt.Println("In progress:")
	} else {
		fmt.Println("History:")
	}
}

func (c *HistoryCmd) handleDeleteRows(ctx context.Context, media []models.MediaWithDB) error {
	for _, dbPath := range c.Databases {
		var paths []string
		for _, m := range media {
			if m.DB == dbPath {
				paths = append(paths, m.Path)
			}
		}
		if len(paths) > 0 {
			if err := history.DeleteHistoryByPaths(ctx, dbPath, paths); err != nil {
				return err
			}
		}
	}
	fmt.Printf("Deleted history for %d items\n", len(media))
	return nil
}

type HistoryAddCmd struct {
	models.CoreFlags `embed:""`

	Done bool     `help:"Mark as done"`
	Args []string `help:"Database file followed by paths to mark as played" required:"true" name:"args" arg:""`

	Paths    []string `kong:"-"`
	Database string   `kong:"-"`
}

func (c *HistoryAddCmd) AfterApply() error {
	if err := c.CoreFlags.AfterApply(); err != nil {
		return err
	}
	if len(c.Args) < 2 {
		return errors.New("at least one database file and one path are required")
	}
	c.Database = c.Args[0]
	c.Paths = c.Args[1:]
	return nil
}

func (c *HistoryAddCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)

	var absPaths []string
	for _, p := range c.Paths {
		abs, err := filepath.Abs(p)
		if err == nil {
			absPaths = append(absPaths, abs)
		} else {
			absPaths = append(absPaths, p)
		}
	}

	err := history.UpdateHistorySimple(ctx, c.Database, absPaths, 0, c.Done)
	if err == nil {
		models.Log.Info("History added", "count", len(absPaths), "database", c.Database)
	}
	return err
}
