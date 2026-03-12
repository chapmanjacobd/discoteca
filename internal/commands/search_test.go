package commands

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestSearchCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	// We need a title to search for
	addCmd.AfterApply()
	addCmd.Run(nil)

	// Manually set title so we can search it
	dbConn := fixture.GetDB()
	dbConn.Exec("UPDATE media SET title = 'Super Secret Movie' WHERE path = ?", f1)
	dbConn.Close()

	cmd := &SearchCmd{
		FilterFlags: models.FilterFlags{Search: []string{"Secret"}},
		Databases:   []string{fixture.DBPath},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("SearchCmd failed: %v", err)
	}
}
