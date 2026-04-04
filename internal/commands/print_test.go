package commands

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestPrintCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	t.Run("PrintFromDB", func(t *testing.T) {
		cmd := &PrintCmd{
			Args: []string{fixture.DBPath},
		}
		cmd.AfterApply()
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}
	})

	t.Run("PrintFromFS", func(t *testing.T) {
		cmd := &PrintCmd{
			Args: []string{fixture.TempDir},
		}
		cmd.AfterApply()
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}
	})

	t.Run("PrintJSONAggregated", func(t *testing.T) {
		cmd := &PrintCmd{
			DisplayFlags:   models.DisplayFlags{JSON: true},
			AggregateFlags: models.AggregateFlags{BigDirs: true},
			Args:           []string{fixture.DBPath},
		}
		cmd.AfterApply()
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}
	})
}
