package commands_test

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestDiskUsageCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("dir1/media1.mp4")

	addCmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	t.Run("DefaultDU", func(t *testing.T) {
		cmd := &commands.DiskUsageCmd{
			Args: []string{fixture.DBPath},
		}
		cmd.AfterApply()
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.DiskUsageCmd failed: %v", err)
		}
	})
}
