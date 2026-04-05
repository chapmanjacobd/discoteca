package commands_test

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestStatsCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("media1.mp4")
	addCmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	t.Run("DefaultStats", func(t *testing.T) {
		cmd := &commands.StatsCmd{
			Facet:     "watched",
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.StatsCmd failed: %v", err)
		}
	})

	t.Run("JSONStats", func(t *testing.T) {
		cmd := &commands.StatsCmd{
			DisplayFlags: models.DisplayFlags{JSON: true},
			Facet:        "watched",
			Databases:    []string{fixture.DBPath},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.StatsCmd failed: %v", err)
		}
	})
}
