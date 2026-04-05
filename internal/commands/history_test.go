package commands_test

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestHistoryCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	if err := addCmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.AddCmd failed: %v", err)
	}

	// Add history
	addHist := &commands.HistoryAddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addHist.AfterApply()
	addHist.Run(context.Background())

	t.Run("DefaultHistory", func(t *testing.T) {
		cmd := &commands.HistoryCmd{
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.HistoryCmd failed: %v", err)
		}
	})

	t.Run("DeleteHistory", func(t *testing.T) {
		cmd := &commands.HistoryCmd{
			PostActionFlags: models.PostActionFlags{
				DeleteRows: true,
			},
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.HistoryCmd failed: %v", err)
		}
	})
}

func TestHistoryAddCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")

	cmd := &commands.HistoryAddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}

	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.HistoryAddCmd failed: %v", err)
	}
}
