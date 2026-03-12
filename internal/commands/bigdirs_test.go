package commands

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestBigDirsCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("dir1/media1.mp4")
	fixture.CreateDummyFile("dir2/media2.mp4")

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	cmd := &BigDirsCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("BigDirsCmd failed: %v", err)
	}
}
