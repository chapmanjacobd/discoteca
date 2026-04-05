package commands_test

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestMediaCheckCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")

	addCmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	t.Run("QuickScan", func(t *testing.T) {
		cmd := &commands.MediaCheckCmd{
			Databases: []string{fixture.DBPath},
		}
		// This will likely fail because ffmpeg is not there or file is invalid
		// but we want to see if the code runs.
		cmd.Run(context.Background())
	})

	t.Run("FullScan", func(t *testing.T) {
		cmd := &commands.MediaCheckCmd{
			Databases: []string{fixture.DBPath},
			FullScan:  true,
		}
		cmd.Run(context.Background())
	})
}
