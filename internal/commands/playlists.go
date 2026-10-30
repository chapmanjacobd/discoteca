package commands

import (
	"context"
	"fmt"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type PlaylistsCmd struct {
	models.CoreFlags `embed:""`

	JSON      bool     `help:"Output results as JSON" short:"j"`
	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

type PlaylistEntry struct {
	Database string       `json:"database"`
	Playlist db.Playlists `json:"playlist"`
}

func (c *PlaylistsCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)

	var all []PlaylistEntry
	for _, dbPath := range c.Databases {
		sqlDB, queries, err := db.ConnectWithInit(ctx, dbPath)
		if err != nil {
			return err
		}

		playlists, err := queries.GetPlaylists(ctx)
		sqlDB.Close()
		if err != nil {
			return err
		}

		if c.JSON {
			for _, playlist := range playlists {
				all = append(all, PlaylistEntry{Database: dbPath, Playlist: playlist})
			}
			continue
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

	if c.JSON {
		return utils.PrintJSON(all)
	}
	return nil
}
