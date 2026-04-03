package commands

import (
	"context"
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type PlaylistsCmd struct {
	models.CoreFlags    `embed:""`
	models.DisplayFlags `embed:""`

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *PlaylistsCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	for _, dbPath := range c.Databases {
		sqlDB, queries, err := db.ConnectWithInit(dbPath)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		playlists, err := queries.GetPlaylists(context.Background())
		if err != nil {
			return err
		}

		if c.JSON {
			return utils.PrintJSON(playlists)
		}

		fmt.Printf("Playlists in %s:\n", dbPath)
		for _, pl := range playlists {
			fmt.Printf(
				"  %s (%s)\n",
				utils.StringValue(models.NullStringPtr(pl.Path)),
				utils.StringValue(models.NullStringPtr(pl.ExtractorKey)),
			)
		}
		fmt.Println()
	}
	return nil
}
