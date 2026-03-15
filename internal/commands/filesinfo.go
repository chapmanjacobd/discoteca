package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/metadata"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type FilesInfoCmd struct {
	models.CoreFlags        `embed:""`
	models.FilterFlags      `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.MediaFilterFlags `embed:""`
	models.DisplayFlags     `embed:""`

	Args []string `arg:"" required:"" help:"Database file(s) or files/directories to scan"`

	Databases []string `kong:"-"`
	ScanPaths []string `kong:"-"`
}

func (c *FilesInfoCmd) AfterApply() error {
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

func (c *FilesInfoCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		FilterFlags:      c.FilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		DisplayFlags:     c.DisplayFlags,
	}

	var allMedia []models.MediaWithDB

	// Handle databases
	if len(c.Databases) > 0 {
		media, err := query.MediaQuery(context.Background(), c.Databases, flags)
		if err != nil {
			return err
		}
		allMedia = append(allMedia, media...)
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
			meta, err := metadata.Extract(context.Background(), path, c.ScanSubtitles, false, false, "", false, "")
			if err != nil {
				return nil
			}
			allMedia = append(allMedia, models.MediaWithDB{
				Media: models.Media{
					Path:     meta.Media.Path,
					Title:    models.NullStringPtr(meta.Media.Title),
					Type:     models.NullStringPtr(meta.Media.Type),
					Size:     models.NullInt64Ptr(meta.Media.Size),
					Duration: models.NullInt64Ptr(meta.Media.Duration),
				},
			})
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking %s: %v\n", root, err)
		}
	}

	if c.JSON {
		return utils.PrintJSON(allMedia)
	}

	// Basic print if not JSON
	for _, m := range allMedia {
		fmt.Printf("Path: %s\n", m.Path)
		if m.Title != nil {
			fmt.Printf("  Title: %s\n", *m.Title)
		}
		if m.Type != nil {
			fmt.Printf("  Type: %s\n", *m.Type)
		}
		if m.Size != nil {
			fmt.Printf("  Size: %d\n", *m.Size)
		}
		if m.Duration != nil {
			fmt.Printf("  Duration: %d\n", *m.Duration)
		}
		fmt.Println()
	}

	return nil
}
