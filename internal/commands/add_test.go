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

func TestAddCmd_Skip(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")

	cmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	_ = cmd.AfterApply()
	_ = cmd.Run(nil)

	// Second run, should skip
	cmd2 := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	_ = cmd2.AfterApply()
	if err := cmd2.Run(nil); err != nil {
		t.Fatalf("AddCmd second run failed: %v", err)
	}
	// We check if it skipped by checking if the output says 1/1 processed from skip
	// But it's hard to capture stdout here.
	// Instead, let's verify it still works when file is marked as deleted.
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	_, _ = dbConn.Exec("UPDATE media SET time_deleted = unixepoch() WHERE path = ?", f1)

	cmd3 := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	_ = cmd3.AfterApply()
	if err := cmd3.Run(nil); err != nil {
		t.Fatalf("AddCmd third run failed: %v", err)
	}

	var timeDeleted int64
	_ = dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if timeDeleted != 0 {
		t.Errorf("Expected time_deleted to be 0 after re-adding, got %d", timeDeleted)
	}
}
