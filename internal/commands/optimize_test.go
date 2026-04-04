package commands

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestOptimizeCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath},
	}
	addCmd.AfterApply() // Will fail if no paths, but we just want to init DB
	// Manually init DB
	dbConn := fixture.GetDB()
	db.InitDB(context.Background(), dbConn)
	dbConn.Close()

	cmd := &OptimizeCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("OptimizeCmd failed: %v", err)
	}
}
