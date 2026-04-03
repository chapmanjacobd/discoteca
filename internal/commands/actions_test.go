package commands

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestMarkDeletedItem(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
		DB:    fixture.DBPath,
	}

	if err := MarkDeletedItem(m); err != nil {
		t.Fatalf("MarkDeletedItem failed: %v", err)
	}

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

func TestMoveMediaItem(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	destDir := filepath.Join(fixture.TempDir, "moved")
	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
		DB:    fixture.DBPath,
	}

	if err := MoveMediaItem(destDir, m); err != nil {
		t.Fatalf("MoveMediaItem failed: %v", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(f1))
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("Expected file to exist at %s", destPath)
	}
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Errorf("Expected original file to be gone")
	}

	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var count int
	dbConn.QueryRow("SELECT COUNT(*) FROM media WHERE path = ?", destPath).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 row with new path, got %d", count)
	}
}

func TestCopyMediaItem(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
	}

	destDir := filepath.Join(fixture.TempDir, "copied")
	if err := CopyMediaItem(destDir, m); err != nil {
		t.Fatalf("CopyMediaItem failed: %v", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(f1))
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("Expected file to exist at %s", destPath)
	}
	if _, err := os.Stat(f1); err != nil {
		t.Errorf("Expected original file to still exist")
	}
}

func TestDeleteMediaItem(t *testing.T) {
	t.Parallel()
	f, err := os.CreateTemp(t.TempDir(), "delete-test")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	m := models.MediaWithDB{
		Media: models.Media{Path: f.Name()},
	}

	if err := DeleteMediaItem(m); err != nil {
		t.Fatalf("DeleteMediaItem failed: %v", err)
	}

	if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
		t.Error("File still exists after DeleteMediaItem")
	}
}

func TestExecutePostAction(t *testing.T) {
	t.Parallel()
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
		DB:    fixture.DBPath,
	}

	// Test mark-deleted
	flags := models.GlobalFlags{
		PostActionFlags: models.PostActionFlags{
			PostAction: "mark-deleted",
		},
	}
	// Manually init DB
	dbConn := fixture.GetDB()
	db.InitDB(dbConn)
	dbConn.Exec("INSERT INTO media (path) VALUES (?)", f1)
	dbConn.Close()

	if err := ExecutePostAction(flags, []models.MediaWithDB{m}); err != nil {
		t.Fatalf("ExecutePostAction mark-deleted failed: %v", err)
	}

	dbConn = fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted int64
	dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if timeDeleted == 0 {
		t.Error("Expected item to be marked as deleted")
	}
}
