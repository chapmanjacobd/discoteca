package commands

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestCheckCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	// Delete file from FS
	os.Remove(f1)

	cmd := &CheckCmd{
		Args: []string{fixture.DBPath},
	}
	cmd.AfterApply()
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("CheckCmd failed: %v", err)
	}

	// Verify it was marked as deleted
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	err := dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
		t.Errorf("Expected file to be marked as deleted")
	}
}
