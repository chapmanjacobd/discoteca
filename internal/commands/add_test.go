package commands

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestAddCmd_Run(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")
	f2 := fixture.CreateDummyFile("audio1.mp3")

	cmd := &AddCmd{
		Args: []string{fixture.DBPath, f1, f2},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}

	if err := cmd.Run(nil); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	// Verify items added
	dbConn := fixture.GetDB()
	defer dbConn.Close()

	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM media").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 items in database, got %d", count)
	}
}
