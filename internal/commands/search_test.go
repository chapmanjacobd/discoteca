package commands_test

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestSearchCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	// We need a title to search for
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	// Manually set title so we can search it
	dbConn := fixture.GetDB()
	dbConn.Exec("UPDATE media SET title = 'Super Secret Movie' WHERE path = ?", f1)
	dbConn.Close()

	cmd := &commands.SearchCmd{
		FilterFlags: models.FilterFlags{Search: []string{"Secret"}},
		Databases:   []string{fixture.DBPath},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.SearchCmd failed: %v", err)
	}
}
