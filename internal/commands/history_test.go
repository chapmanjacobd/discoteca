package commands

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestHistoryCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	// Add history
	addHist := &HistoryAddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addHist.AfterApply()
	addHist.Run(nil)

	t.Run("DefaultHistory", func(t *testing.T) {
		cmd := &HistoryCmd{
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("HistoryCmd failed: %v", err)
		}
	})

	t.Run("DeleteHistory", func(t *testing.T) {
		cmd := &HistoryCmd{
			PostActionFlags: models.PostActionFlags{
				DeleteRows: true,
			},
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("HistoryCmd failed: %v", err)
		}
	})
}

func TestHistoryAddCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")

	cmd := &HistoryAddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}

	if err := cmd.Run(nil); err != nil {
		t.Fatalf("HistoryAddCmd failed: %v", err)
	}
}
