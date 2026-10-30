package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

type FilesInfoCmd struct {
	models.CoreFlags        `embed:""`
	models.FilterFlags      `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.MediaFilterFlags `embed:""`

	JSON bool `help:"Output results as JSON" short:"j" group:"Display"`

	Args []string `help:"Database file(s) or files/directories to scan" required:"true" arg:""`

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
		if strings.HasSuffix(arg, ".db") && fileutil.IsSQLite(arg) {
			c.Databases = append(c.Databases, arg)
		} else {
			c.ScanPaths = append(c.ScanPaths, arg)
		}
	}
	return nil
}

func (c *FilesInfoCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		FilterFlags:      c.FilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
	}

	var allMedia []models.MediaWithDB

	// Handle databases
	if len(c.Databases) > 0 {
		media, err := query.MediaQuery(ctx, c.Databases, flags)
		if err != nil {
			return err
		}
		allMedia = append(allMedia, media...)
	}

	// Handle paths
	scanned, err := ScanMediaPaths(ctx, c.ScanPaths, scanOptionsForFlags(flags))
	if err != nil {
		return err
	}
	allMedia = append(allMedia, filterScannedMedia(scanned, flags)...)

	if c.JSON {
		return fileutil.PrintJSON(allMedia)
	}

	// Basic print if not JSON
	for _, m := range allMedia {
		fmt.Printf("Path: %s\n", m.Path)
		if m.Title != nil {
			fmt.Printf("  Title: %s\n", *m.Title)
		}
		if m.MediaType != nil {
			fmt.Printf("  MediaType: %s\n", *m.MediaType)
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
