package commands

import (
	"context"

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

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *SearchCmd) Run(ctx context.Context) error {
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
	// We prefer FTS if not specified
	if !flags.FTS && !flags.NoFTS {
		// Check if FTS table exists in first database
		if len(c.Databases) > 0 {
			if sqlDB, err := db.Connect(ctx, c.Databases[0]); err == nil {
				var name string
				err := sqlDB.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='media_fts'").
					Scan(&name)
				if err == nil {
					// Verify FTS5 actually works by running a simple query
					rows, testErr := sqlDB.QueryContext(ctx, "SELECT 1 FROM media_fts LIMIT 1")
					if testErr == nil {
						defer rows.Close()
						_ = rows.Err() // Check for row errors
						flags.FTS = true
					}
				}
				sqlDB.Close()
			}
		}
	}

	return RunQuery(ctx, c.Databases, flags, func(media []models.MediaWithDB) error {
		query.SortMedia(media, flags)

		if flags.JSON {
			return utils.PrintJSON(media)
		}

		return PrintMedia(flags.DisplayFlags, flags.Columns, media)
	})
}
