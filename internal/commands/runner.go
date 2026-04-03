package commands

import (
	"context"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

// RunQuery handles the common boilerplate for executing a media query
func RunQuery(
	ctx context.Context,
	dbs []string,
	flags models.GlobalFlags,
	process func([]models.MediaWithDB) error,
) error {
	models.SetupLogging(flags.Verbose)

	media, err := query.MediaQuery(ctx, dbs, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)

	return process(media)
}
