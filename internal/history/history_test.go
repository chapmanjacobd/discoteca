package history

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, string) {
	f, err := os.CreateTemp(t.TempDir(), "history-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatal(err)
	}

	schema := `
	CREATE TABLE media (
		path TEXT PRIMARY KEY,
		time_first_played INTEGER,
		time_last_played INTEGER,
		playhead INTEGER,
		play_count INTEGER DEFAULT 0,
		time_deleted INTEGER DEFAULT 0
	);
	CREATE TABLE history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_path TEXT NOT NULL,
		time_played INTEGER,
		playhead INTEGER,
		done INTEGER
	);
	`
	if _, err := sqlDB.Exec(schema); err != nil {
		sqlDB.Close()
		os.Remove(dbPath)
		t.Fatal(err)
	}

	return sqlDB, dbPath
}

func TestTracker_UpdatePlayback(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	defer os.Remove(dbPath)
	defer sqlDB.Close()

	path := "/test/video.mp4"
	if _, err := sqlDB.Exec("INSERT INTO media (path) VALUES (?)", path); err != nil {
		t.Fatal(err)
	}

	tracker := NewTracker(sqlDB)
	if err := tracker.UpdatePlayback(context.Background(), path, 100); err != nil {
		t.Fatalf("UpdatePlayback failed: %v", err)
	}

	// Verify media update
	var lastPlayed, playhead int64
	err := sqlDB.QueryRow("SELECT time_last_played, playhead FROM media WHERE path = ?", path).
		Scan(&lastPlayed, &playhead)
	if err != nil {
		t.Fatalf("Failed to query media: %v", err)
	}
	if lastPlayed == 0 {
		t.Error("Expected time_last_played to be set")
	}
	if playhead != 100 {
		t.Errorf("Expected playhead 100, got %d", playhead)
	}

	// Verify history record
	var hPath string
	var hPlayhead int64
	err = sqlDB.QueryRow("SELECT media_path, playhead FROM history WHERE media_path = ?", path).Scan(&hPath, &hPlayhead)
	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}
	if filepath.ToSlash(hPath) != filepath.ToSlash(path) {
		t.Errorf("Expected history path %s, got %s", path, hPath)
	}
	if hPlayhead != 100 {
		t.Errorf("Expected history playhead 100, got %d", hPlayhead)
	}
}

func TestUpdateHistorySimple(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	sqlDB.Close() // UpdateHistorySimple opens its own connection
	defer os.Remove(dbPath)

	// Re-open to insert test data
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	path := "/test/audio.mp3"
	dbConn.Exec("INSERT INTO media (path) VALUES (?)", path)
	dbConn.Close()

	if err := UpdateHistorySimple(context.Background(), dbPath, []string{path}, 50, true); err != nil {
		t.Fatalf("UpdateHistorySimple failed: %v", err)
	}

	// Verify
	dbVerify, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbVerify.Close()

	var done int64
	err = dbVerify.QueryRow("SELECT done FROM history WHERE media_path = ?", path).Scan(&done)
	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}
	if done != 1 {
		t.Errorf("Expected done=1, got %d", done)
	}
}

func TestTracker_MarkDeleted(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	defer sqlDB.Close()
	defer os.Remove(dbPath)

	path := "deleted.mp4"
	sqlDB.Exec("INSERT INTO media (path) VALUES (?)", path)

	tracker := NewTracker(sqlDB)
	if err := tracker.MarkDeleted(context.Background(), path); err != nil {
		t.Fatal(err)
	}

	var timeDeleted int64
	err := sqlDB.QueryRow("SELECT time_deleted FROM media WHERE path = ?", path).Scan(&timeDeleted)
	if err != nil {
		t.Fatal(err)
	}
	if timeDeleted == 0 {
		t.Error("Expected time_deleted to be non-zero")
	}
}

func TestUpdateHistoryWithTime(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	sqlDB.Close()
	defer os.Remove(dbPath)

	path := "old.mp3"
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	dbConn.Exec("INSERT INTO media (path) VALUES (?)", path)
	dbConn.Close()

	customTime := int64(1000000000)
	if err := UpdateHistoryWithTime(context.Background(), dbPath, []string{path}, 10, customTime, false); err != nil {
		t.Fatal(err)
	}

	dbVerify, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbVerify.Close()
	var lastPlayed int64
	dbVerify.QueryRow("SELECT time_last_played FROM media WHERE path = ?", path).Scan(&lastPlayed)
	if lastPlayed != customTime {
		t.Errorf("Expected lastPlayed %d, got %d", customTime, lastPlayed)
	}
}

func TestDeleteHistoryByPaths(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	defer os.Remove(dbPath)
	defer sqlDB.Close()

	path := "todelete.mp4"
	sqlDB.Exec("INSERT INTO media (path, play_count) VALUES (?, 5)", path)
	sqlDB.Exec("INSERT INTO history (media_path, playhead) VALUES (?, 100)", path)

	if err := DeleteHistoryByPaths(dbPath, []string{path}); err != nil {
		t.Fatal(err)
	}

	var count int
	sqlDB.QueryRow("SELECT COUNT(*) FROM history WHERE media_path = ?", path).Scan(&count)
	if count != 0 {
		t.Error("History record should be deleted")
	}

	var playCount int
	sqlDB.QueryRow("SELECT play_count FROM media WHERE path = ?", path).Scan(&playCount)
	if playCount != 0 {
		t.Error("Play count should be reset")
	}
}
