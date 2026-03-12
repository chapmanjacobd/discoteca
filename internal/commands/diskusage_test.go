package commands

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestDiskUsageCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("dir1/media1.mp4")

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	t.Run("DefaultDU", func(t *testing.T) {
		cmd := &DiskUsageCmd{
			Args: []string{fixture.DBPath},
		}
		cmd.AfterApply()
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("DiskUsageCmd failed: %v", err)
		}
	})
}
