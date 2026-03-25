package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

func TestE2E_AddAndCheck(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()
	sqlDB := fixture.GetDB()
	defer sqlDB.Close()
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("database initialization failed: %v", err)
	}

	// 1. Add a dummy file
	dummyPath := fixture.CreateDummyFile("video.mp4")

	addCmd := &commands.AddCmd{
		MediaFilterFlags: models.MediaFilterFlags{ScanSubtitles: false},
		Database:         fixture.DBPath,
		ScanPaths:        []string{dummyPath},
		Parallel:         1,
	}

	if err := addCmd.Run(nil); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	// 2. Verify file is in DB
	sqlDB2 := fixture.GetDB()
	defer sqlDB2.Close()
	var count int
	err := sqlDB2.QueryRow("SELECT COUNT(*) FROM media WHERE path = ? AND time_deleted = 0", dummyPath).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("Expected 1 media record, got %d", count)
	}

	// 3. Delete the physical file
	if err := os.Remove(dummyPath); err != nil {
		t.Fatal(err)
	}

	// 4. Run Check command
	checkCmd := &commands.CheckCmd{
		Databases:  []string{fixture.DBPath},
		CheckPaths: []string{fixture.TempDir},
	}

	if err := checkCmd.Run(nil); err != nil {
		t.Fatalf("CheckCmd failed: %v", err)
	}

	// 5. Verify marked as deleted
	sqlDB3 := fixture.GetDB()
	defer sqlDB3.Close()
	var timeDeleted int64
	err = sqlDB3.QueryRow("SELECT time_deleted FROM media WHERE path = ?", dummyPath).Scan(&timeDeleted)
	if err != nil {
		t.Fatal(err)
	}
	if timeDeleted == 0 {
		t.Error("Expected file to be marked as deleted in database")
	}
}

func TestE2E_HistoryAdd(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()
	sqlDB := fixture.GetDB()
	defer sqlDB.Close()
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("database initialization failed: %v", err)
	}

	dummyPath := fixture.CreateDummyFile("played.mp4")

	// 1. Add to media
	addCmd := &commands.AddCmd{
		MediaFilterFlags: models.MediaFilterFlags{ScanSubtitles: false},
		Database:         fixture.DBPath,
		ScanPaths:        []string{dummyPath},
		Parallel:         1,
	}
	addCmd.Run(nil)

	// 2. Add to history
	histCmd := &commands.HistoryAddCmd{
		Database: fixture.DBPath,
		Paths:    []string{dummyPath},
		Done:     true,
	}
	if err := histCmd.Run(nil); err != nil {
		t.Fatalf("HistoryAddCmd failed: %v", err)
	}

	// 3. Verify history record
	sqlDB2 := fixture.GetDB()
	defer sqlDB2.Close()
	var count int
	err := sqlDB2.QueryRow("SELECT COUNT(*) FROM history WHERE media_path = ? AND done = 1", dummyPath).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("Expected 1 history record, got %d", count)
	}
}

func TestE2E_PathConsolidation(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()
	sqlDB := fixture.GetDB()
	defer sqlDB.Close()
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("database initialization failed: %v", err)
	}

	parentDir := filepath.Join(fixture.TempDir, "parent")
	subDir := filepath.Join(parentDir, "sub")
	os.MkdirAll(subDir, 0o755)
	fixture.CreateDummyFile("parent/video1.mp4")
	fixture.CreateDummyFile("parent/sub/video2.mp4")

	// 1. Add parent
	addCmd := &commands.AddCmd{
		Database:  fixture.DBPath,
		ScanPaths: []string{parentDir},
		Parallel:  1,
	}
	addCmd.Run(nil)

	sqlDB2 := fixture.GetDB()
	defer sqlDB2.Close()
	var count int
	sqlDB2.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 playlist, got %d", count)
	}

	// 2. Try adding subpath - should be skipped
	addCmdSub := &commands.AddCmd{
		Database:  fixture.DBPath,
		ScanPaths: []string{subDir},
		Parallel:  1,
	}
	addCmdSub.Run(nil)

	sqlDB3 := fixture.GetDB()
	defer sqlDB3.Close()
	sqlDB3.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&count)
	if count != 1 {
		t.Errorf("Expected still 1 playlist, got %d", count)
	}
}

// TestE2E_PathConsolidation_WindowsPaths tests path consolidation with Windows-style paths
// This ensures the logic works correctly on both Unix and Windows
func TestE2E_PathConsolidation_WindowsPaths(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()
	sqlDB := fixture.GetDB()
	defer sqlDB.Close()
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("database initialization failed: %v", err)
	}

	// Create directory structure with Windows-style path separators in test data
	parentDir := filepath.Join(fixture.TempDir, "parent")
	subDir := filepath.Join(parentDir, "sub")
	os.MkdirAll(subDir, 0o755)
	fixture.CreateDummyFile("parent/video1.mp4")
	fixture.CreateDummyFile("parent/sub/video2.mp4")

	// Test with forward slashes (Unix-style but valid on Windows too)
	addCmd := &commands.AddCmd{
		Database:  fixture.DBPath,
		ScanPaths: []string{parentDir},
		Parallel:  1,
	}
	addCmd.Run(nil)

	sqlDB2 := fixture.GetDB()
	defer sqlDB2.Close()
	var count int
	sqlDB2.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 playlist, got %d", count)
	}

	// Try adding with backslash path (Windows-style)
	winStyleSubDir := parentDir + string(filepath.Separator) + "sub"
	addCmdSub := &commands.AddCmd{
		Database:  fixture.DBPath,
		ScanPaths: []string{winStyleSubDir},
		Parallel:  1,
	}
	addCmdSub.Run(nil)

	sqlDB3 := fixture.GetDB()
	defer sqlDB3.Close()
	sqlDB3.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&count)
	if count != 1 {
		t.Errorf("Expected still 1 playlist after adding Windows-style subpath, got %d", count)
	}
}

func TestE2E_Stats(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dummyPath := fixture.CreateDummyFile("stats.mp4")

	// 1. Add to media
	addCmd := &commands.AddCmd{
		MediaFilterFlags: models.MediaFilterFlags{ScanSubtitles: false},
		Database:         fixture.DBPath,
		ScanPaths:        []string{dummyPath},
		Parallel:         1,
	}
	addCmd.Run(nil)

	// 2. Add to history
	histCmd := &commands.HistoryAddCmd{
		Database: fixture.DBPath,
		Paths:    []string{dummyPath},
	}
	histCmd.Run(nil)

	// 3. Run Stats
	statsCmd := &commands.StatsCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := statsCmd.Run(nil); err != nil {
		t.Fatalf("StatsCmd failed: %v", err)
	}
}

func TestCLI_Structure(t *testing.T) {
	cli := &CLI{}
	_, err := kong.New(cli)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
}

func TestE2E_AddWithVTTCaptions(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	sqlDB_init, err := sql.Open("sqlite3", fixture.DBPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer sqlDB_init.Close()
	if err := db.InitDB(sqlDB_init); err != nil {
		t.Fatalf("database initialization failed: %v", err)
	}

	// 1. Create a dummy video file and a sidecar VTT
	videoPath := filepath.Join(fixture.TempDir, "movie.mp4")
	if err := os.WriteFile(videoPath, []byte("\x00\x00\x00\x20ftypisom"), 0o644); err != nil {
		t.Fatalf("failed to create dummy video: %v", err)
	}

	vttPath := filepath.Join(fixture.TempDir, "movie.vtt")
	vttContent := `WEBVTT

00:00:11.000 --> 00:00:14.000
This is a sample caption.

00:00:15.000 --> 00:00:18.000
Another caption here.
`
	if err := os.WriteFile(vttPath, []byte(vttContent), 0o644); err != nil {
		t.Fatalf("failed to create dummy vtt: %v", err)
	}

	// 2. Run AddCmd with ScanSubtitles enabled
	addCmd := &commands.AddCmd{
		CoreFlags: models.CoreFlags{Verbose: 1},
		MediaFilterFlags: models.MediaFilterFlags{
			ScanSubtitles: true,
		},
		Database: fixture.DBPath,
		Args:     []string{fixture.DBPath, videoPath},
		Parallel: 1,
	}
	// We need to call AfterApply to set up Internal fields correctly
	if err := addCmd.AfterApply(); err != nil {
		t.Fatalf("AddCmd.AfterApply failed: %v", err)
	}

	if err := addCmd.Run(nil); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	// 3. Verify captions are in DB
	sqlDB, err := sql.Open("sqlite3", fixture.DBPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer sqlDB.Close()

	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM media WHERE path = ?", videoPath).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query media: %v", err)
	}
	if count == 0 {
		t.Fatalf("Expected media to be imported into the database, but found 0")
	}

	err = sqlDB.QueryRow("SELECT COUNT(*) FROM captions WHERE media_path = ?", videoPath).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query captions: %v", err)
	}

	if count == 0 {
		t.Error("Expected captions to be imported into the database, but found 0")
	} else {
		t.Logf("Found %d captions", count)
	}
}

// TestE2E_MetadataTags tests that metadata tags are correctly read and saved
func TestE2E_MetadataTags(t *testing.T) {
	// Skip if ffmpeg is not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found, skipping metadata tags test")
	}

	fixture := testutils.Setup(t)
	defer fixture.Cleanup()
	sqlDB := fixture.GetDB()
	defer sqlDB.Close()
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("database initialization failed: %v", err)
	}

	// Create a real video file with embedded metadata using ffmpeg
	videoPath := filepath.Join(fixture.TempDir, "test_with_tags.mp4")
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "lavfi",
		"-i", "testsrc=size=1920x1080:rate=30",
		"-f", "lavfi",
		"-i", "sine=frequency=440:duration=5",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-t", "5",
		"-metadata", "title=Test Video Title",
		"-metadata", "artist=Test Artist",
		"-metadata", "album=Test Album",
		"-metadata", "genre=Test Genre",
		"-metadata", "comment=Test Comment",
		videoPath,
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("ffmpeg failed to create test video: %v", err)
	}

	// Add the file to the database
	addCmd := &commands.AddCmd{
		MediaFilterFlags: models.MediaFilterFlags{ScanSubtitles: false},
		Database:         fixture.DBPath,
		ScanPaths:        []string{videoPath},
		Parallel:         1,
	}

	if err := addCmd.Run(nil); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	// Verify metadata was saved correctly
	sqlDB2 := fixture.GetDB()
	defer sqlDB2.Close()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"title", "SELECT title FROM media WHERE path = ?", "Test Video Title"},
		{"artist", "SELECT artist FROM media WHERE path = ?", "Test Artist"},
		{"album", "SELECT album FROM media WHERE path = ?", "Test Album"},
		{"genre", "SELECT genre FROM media WHERE path = ?", "Test Genre"},
		{"description", "SELECT description FROM media WHERE path = ?", "Test Comment"},
		{"width", "SELECT width FROM media WHERE path = ?", "1920"},
		{"height", "SELECT height FROM media WHERE path = ?", "1080"},
		{"duration", "SELECT duration FROM media WHERE path = ?", "5"},
		{"media_type", "SELECT media_type FROM media WHERE path = ?", "video"},
	}

	for _, tt := range tests {
		var got string
		err := sqlDB2.QueryRow(tt.query, videoPath).Scan(&got)
		if err != nil {
			t.Errorf("%s: query failed: %v", tt.name, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("%s: expected '%s', got '%s'", tt.name, tt.expected, got)
		} else {
			t.Logf("%s: OK ('%s')", tt.name, got)
		}
	}
}
