package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/alecthomas/kong"

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

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *HistoryCmd) Run(ctx *kong.Context) error {
	flags := models.BuildQueryGlobalFlags(
		c.CoreFlags,
		models.QueryFlags{},
		c.PathFilterFlags,
		c.FilterFlags,
		c.MediaFilterFlags,
		c.TimeFilterFlags,
		c.DeletedFlags,
		c.SortFlags,
		c.DisplayFlags,
		models.FTSFlags{},
	)
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

	return RunQuery(context.Background(), c.Databases, flags, func(media []models.MediaWithDB) error {
		HideRedundantFirstPlayed(media)

		if flags.JSON {
			return utils.PrintJSON(media)
		}

		if flags.Completed {
			fmt.Println("Completed:")
		} else if flags.InProgress {
			fmt.Println("In progress:")
		} else {
			fmt.Println("History:")
		}

		if flags.DeleteRows {
			for _, dbPath := range c.Databases {
				var paths []string
				for _, m := range media {
					if m.DB == dbPath {
						paths = append(paths, m.Path)
					}
				}
				if len(paths) > 0 {
					if err := history.DeleteHistoryByPaths(dbPath, paths); err != nil {
						return err
					}
				}
			}
			fmt.Printf("Deleted history for %d items\n", len(media))
			return nil
		}

		if flags.Partial != "" {
			query.SortHistory(media, flags.Partial, flags.Reverse)
		} else {
			query.SortMedia(media, flags)
		}
		return PrintMedia(flags.DisplayFlags, flags.Columns, media)
	})
}

type HistoryAddCmd struct {
	models.CoreFlags `embed:""`

	Done bool     `help:"Mark as done"`
	Args []string `help:"Database file followed by paths to mark as played" required:"" name:"args" arg:""`

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

func (c *HistoryAddCmd) Run(ctx *kong.Context) error {
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

	err := history.UpdateHistorySimple(c.Database, absPaths, 0, c.Done)
	if err == nil {
		slog.Info("History added", "count", len(absPaths), "database", c.Database)
	}
	return err
}
