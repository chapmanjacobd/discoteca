package commands

import (
	"context"

	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
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

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *SearchCmd) Run(ctx *kong.Context) error {
	flags := models.BuildQueryGlobalFlags(
		c.CoreFlags,
		c.QueryFlags,
		c.PathFilterFlags,
		c.FilterFlags,
		c.MediaFilterFlags,
		c.TimeFilterFlags,
		c.DeletedFlags,
		c.SortFlags,
		c.DisplayFlags,
		c.FTSFlags,
	)
	// We prefer FTS if not specified
	if !flags.FTS && !flags.NoFTS {
		// Check if FTS table exists in first database
		if len(c.Databases) > 0 {
			if sqlDB, err := db.Connect(c.Databases[0]); err == nil {
				var name string
				err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='media_fts'").
					Scan(&name)
				if err == nil {
					// Verify FTS5 actually works by running a simple query
					_, testErr := sqlDB.Query("SELECT 1 FROM media_fts LIMIT 1")
					if testErr == nil {
						flags.FTS = true
					}
				}
				sqlDB.Close()
			}
		}
	}

	return RunQuery(context.Background(), c.Databases, flags, func(media []models.MediaWithDB) error {
		query.SortMedia(media, flags)

		if flags.JSON {
			return utils.PrintJSON(media)
		}

		return PrintMedia(flags.DisplayFlags, flags.Columns, media)
	})
}
