package commands

import (
	"context"

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
	// We prefer FTS only when every database supports it. A mixed list must not
	// run an FTS join against a database that has no FTS table.
	if !flags.FTS && !flags.NoFTS {
		if query.DatabasesHaveFTS(ctx, c.Databases) {
			flags.FTS = true
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
