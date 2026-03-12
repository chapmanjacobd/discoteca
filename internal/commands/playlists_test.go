package commands

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestPlaylistsCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	cmd := &PlaylistsCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("PlaylistsCmd failed: %v", err)
	}
}
