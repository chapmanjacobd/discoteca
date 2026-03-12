package commands

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestStatsCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	t.Run("DefaultStats", func(t *testing.T) {
		cmd := &StatsCmd{
			Facet:     "watched",
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("StatsCmd failed: %v", err)
		}
	})

	t.Run("JSONStats", func(t *testing.T) {
		cmd := &StatsCmd{
			DisplayFlags: models.DisplayFlags{JSON: true},
			Facet:        "watched",
			Databases:    []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("StatsCmd failed: %v", err)
		}
	})
}
