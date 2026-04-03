package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/tui"
)

type TuiCmd struct {
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

func (c *TuiCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	// Override DeletedFlags to load all media (including deleted) for the TUI
	// The TUI's internal filters will handle showing/hiding deleted items
	deletedFlags := c.DeletedFlags
	deletedFlags.HideDeleted = false
	deletedFlags.OnlyDeleted = false

	flags := models.BuildQueryGlobalFlags(
		c.CoreFlags,
		c.QueryFlags,
		c.PathFilterFlags,
		c.FilterFlags,
		c.MediaFilterFlags,
		c.TimeFilterFlags,
		deletedFlags,
		c.SortFlags,
		c.DisplayFlags,
		c.FTSFlags,
	)

	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		return err
	}

	if len(media) == 0 {
		return errors.New("no media found")
	}

	query.SortMedia(media, flags)

	var customCats []string
	for _, dbPath := range c.Databases {
		sqlDB, queries, err := db.ConnectWithInit(dbPath)
		if err == nil {
			cats, err := queries.GetCustomCategories(context.Background())
			if err == nil {
				customCats = append(customCats, cats...)
			}
			sqlDB.Close()
		}
	}

	m := tui.NewModel(media, c.Databases, flags, customCats)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	choice := finalModel.(*tui.Model).GetChoice()
	if choice != nil {
		// Play the chosen media
		fmt.Printf("Playing: %s\n", choice.Path)

		args := []string{"mpv", choice.Path}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		return cmd.Run()
	}

	return nil
}
