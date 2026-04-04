package commands

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestBigDirsCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("dir1/media1.mp4")
	fixture.CreateDummyFile("dir2/media2.mp4")

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	if err := addCmd.Run(context.Background()); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	cmd := &BigDirsCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("BigDirsCmd failed: %v", err)
	}
}
