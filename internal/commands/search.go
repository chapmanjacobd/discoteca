package commands

import (
	"context"
	"encoding/json"
	"os"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

type SearchCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.FTSFlags         `embed:""`

	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

func (c *SearchCmd) Run(ctx *kong.Context) error {
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
		FTSFlags:         c.FTSFlags,
	}
	// We prefer FTS if not specified
	if !c.FTS {
		// Check if FTS table exists in first database
		if len(c.Databases) > 0 {
			if sqlDB, err := db.Connect(c.Databases[0]); err == nil {
				var name string
				err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='media_fts'").Scan(&name)
				if err == nil {
					c.FTS = true
				}
				sqlDB.Close()
			}
		}
	}

	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)
	query.SortMedia(media, flags)

	if c.JSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(media)
	}

	return PrintMedia(c.DisplayFlags, c.Columns, media)
}
