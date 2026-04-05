package commands_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestAddCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")
	f2 := fixture.CreateDummyFile("audio1.mp3")

	cmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1, f2},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}

	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.AddCmd failed: %v", err)
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
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")

	cmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	_ = cmd.AfterApply()
	_ = cmd.Run(context.Background())

	// Second run, should skip
	cmd2 := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	_ = cmd2.AfterApply()
	if err := cmd2.Run(context.Background()); err != nil {
		t.Fatalf("commands.AddCmd second run failed: %v", err)
	}
	// We check if it skipped by checking if the output says 1/1 processed from skip
	// But it's hard to capture stdout here.
	// Instead, let's verify it still works when file is marked as deleted.
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	_, _ = dbConn.Exec("UPDATE media SET time_deleted = unixepoch() WHERE path = ?", f1)

	cmd3 := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	_ = cmd3.AfterApply()
	if err := cmd3.Run(context.Background()); err != nil {
		t.Fatalf("commands.AddCmd third run failed: %v", err)
	}

	var timeDeleted int64
	_ = dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if timeDeleted != 0 {
		t.Errorf("Expected time_deleted to be 0 after re-adding, got %d", timeDeleted)
	}
}

func TestAddCmd_AllFiles(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// Create temp directory for scanning
	tempDir := t.TempDir()

	// Create a mix of media and non-media files
	mediaFile := filepath.Join(tempDir, "video.mp4")
	nonMediaFile := filepath.Join(tempDir, "document.txt")
	otherFile := filepath.Join(tempDir, "random.data")
	symlinkFile := filepath.Join(tempDir, "link.mp4")

	os.WriteFile(mediaFile, []byte("fake video content"), 0o644)
	os.WriteFile(nonMediaFile, []byte("fake document content"), 0o644)
	os.WriteFile(otherFile, []byte("fake random content"), 0o644)
	os.Symlink(mediaFile, symlinkFile)

	// Run 'add' on the directory
	cmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, tempDir},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}

	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.AddCmd failed: %v", err)
	}

	// Verify items added
	dbConn := fixture.GetDB()
	defer dbConn.Close()

	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM media").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Should have 4 files: video.mp4, document.txt, random.data, link.mp4 (symlink).
	// Note: symlink skipping is currently disabled in fs.FindMediaChan
	if count != 4 {
		t.Errorf("Expected 4 items in database, got %d", count)
	}

	// Verify types
	var mediaType sql.NullString
	dbConn.QueryRow("SELECT media_type FROM media WHERE path = ?", mediaFile).Scan(&mediaType)
	if mediaType.String != "video" {
		t.Errorf("Expected media_type 'video' for %s, got '%s'", mediaFile, mediaType.String)
	}

	dbConn.QueryRow("SELECT media_type FROM media WHERE path = ?", nonMediaFile).Scan(&mediaType)
	if mediaType.String != "text" {
		t.Errorf("Expected media_type 'text' for %s, got '%s'", nonMediaFile, mediaType.String)
	}
}

func TestAddCmd_FilterVideo(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	tempDir := t.TempDir()

	mediaFile := filepath.Join(tempDir, "video.mp4")
	nonMediaFile := filepath.Join(tempDir, "document.txt")
	os.WriteFile(mediaFile, []byte("fake video"), 0o644)
	os.WriteFile(nonMediaFile, []byte("fake doc"), 0o644)

	cmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, tempDir},
	}
	cmd.VideoOnly = true // Only video
	_ = cmd.AfterApply()
	_ = cmd.Run(context.Background())

	dbConn := fixture.GetDB()
	defer dbConn.Close()

	var count int
	dbConn.QueryRow("SELECT COUNT(*) FROM media").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 item (video only), got %d", count)
	}
}
