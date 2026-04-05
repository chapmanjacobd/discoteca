package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chapmanjacobd/discoteca/internal/metadata"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/tui"
)

type DiskUsageCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.AggregateFlags   `embed:""`
	models.FTSFlags         `embed:""`

	Args []string `help:"Database file(s) or files/directories to scan" required:"true" arg:""`

	Databases []string `kong:"-"`
	ScanPaths []string `kong:"-"`
}

func (c *DiskUsageCmd) AfterApply() error {
	var err error
	c.Databases, c.ScanPaths, err = ParseDatabaseAndScanPaths(c.Args, &c.CoreFlags, &c.MediaFilterFlags)
	return err
}

func (c *DiskUsageCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
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
		FTS:         c.FTSFlags,
	})
	flags.AggregateFlags = c.AggregateFlags

	var allMedia []models.MediaWithDB

	// Handle databases
	if len(c.Databases) > 0 {
		dbMedia, err := query.MediaQuery(ctx, c.Databases, flags)
		if err != nil {
			return err
		}
		allMedia = append(allMedia, dbMedia...)
	}

	// Handle paths
	for _, root := range c.ScanPaths {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			// Use path as-is
			meta, err := metadata.Extract(ctx, path, metadata.ExtractOptions{
				ScanSubtitles: flags.ScanSubtitles,
				ProbeImages:   c.ProbeImages,
			})
			if err != nil {
				return err
			}
			allMedia = append(allMedia, models.MediaWithDB{
				Media: models.Media{
					Path:         meta.Media.Path,
					Title:        models.NullStringPtr(meta.Media.Title),
					MediaType:    models.NullStringPtr(meta.Media.MediaType),
					Size:         models.NullInt64Ptr(meta.Media.Size),
					Duration:     models.NullInt64Ptr(meta.Media.Duration),
					TimeCreated:  models.NullInt64Ptr(meta.Media.TimeCreated),
					TimeModified: models.NullInt64Ptr(meta.Media.TimeModified),
				},
			})
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking %s: %v\n", root, err)
		}
	}

	if c.TUI {
		if len(allMedia) == 0 {
			return errors.New("no media found")
		}

		m := tui.NewDUModel(allMedia, flags)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}

	// Disk usage is essentially Print with aggregation by default if no depth specified
	if !c.BigDirs && !c.GroupByExtensions && !c.GroupBySize && c.Depth == 0 && !c.Parents {
		c.BigDirs = true
	}
	printCmd := PrintCmd{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		SortFlags:        c.SortFlags,
		DisplayFlags:     c.DisplayFlags,
		AggregateFlags:   c.AggregateFlags,
		Databases:        c.Databases,
		ScanPaths:        c.ScanPaths,
	}
	return printCmd.Run(ctx)
}
