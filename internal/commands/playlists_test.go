package commands

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestPlaylistsCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	cmd := &PlaylistsCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("PlaylistsCmd failed: %v", err)
	}
}
