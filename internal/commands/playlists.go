package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/utils"
)

type PlaylistsCmd struct {
	models.CoreFlags    `embed:""`
	models.DisplayFlags `embed:""`
	Databases           []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

func (c *PlaylistsCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	for _, dbPath := range c.Databases {
		sqlDB, err := db.Connect(dbPath)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		queries := db.New(sqlDB)
		playlists, err := queries.GetPlaylists(context.Background())
		if err != nil {
			return err
		}

		if c.JSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(playlists); err != nil {
				return err
			}
			continue
		}

		fmt.Printf("Playlists in %s:\n", dbPath)
		for _, pl := range playlists {
			fmt.Printf("  %s (%s)\n", utils.StringValue(models.NullStringPtr(pl.Path)), utils.StringValue(models.NullStringPtr(pl.ExtractorKey)))
		}
		fmt.Println()
	}
	return nil
}
