package history_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/history"
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

func TestUpdateHistorySimple(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	sqlDB.Close() // history.UpdateHistorySimple opens its own connection
	defer os.Remove(dbPath)

	// Re-open to insert test data
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	path := "/test/audio.mp3"
	dbConn.Exec("INSERT INTO media (path) VALUES (?)", path)
	dbConn.Close()

	if err2 := history.UpdateHistorySimple(context.Background(), dbPath, []string{path}, 50, true); err2 != nil {
		t.Fatalf("history.UpdateHistorySimple failed: %v", err2)
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
	if err2 := history.UpdateHistoryWithTime(
		context.Background(),
		dbPath,
		[]string{path},
		history.HistoryEntry{
			Playhead:   10,
			TimePlayed: customTime,
			MarkDone:   false,
		},
	); err2 != nil {
		t.Fatal(err2)
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

func TestUpdateHistoryWithTime_DoesNotRegressTimestamps(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	sqlDB.Close()
	defer os.Remove(dbPath)

	path := "imported.mp3"
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	dbConn.Exec("INSERT INTO media (path) VALUES (?)", path)
	dbConn.Close()

	update := func(ts int64) {
		t.Helper()
		if updateErr := history.UpdateHistoryWithTime(
			context.Background(),
			dbPath,
			[]string{path},
			history.HistoryEntry{Playhead: 1, TimePlayed: ts},
		); updateErr != nil {
			t.Fatal(updateErr)
		}
	}

	newer := int64(2_000_000_000)
	older := int64(1_000_000_000)
	update(newer)
	update(older)
	update(newer)

	dbVerify, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbVerify.Close()

	var first, last int64
	if err := dbVerify.QueryRow(
		"SELECT time_first_played, time_last_played FROM media WHERE path = ?",
		path,
	).Scan(&first, &last); err != nil {
		t.Fatal(err)
	}
	if first != older {
		t.Errorf("Expected time_first_played to stay at %d, got %d", older, first)
	}
	if last != newer {
		t.Errorf("Expected time_last_played to stay at %d, got %d", newer, last)
	}
}

func TestUpdatePlayHistory_RewriteCanForceTimestamp(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	defer sqlDB.Close()
	defer os.Remove(dbPath)

	path := "rewrite.mp4"
	sqlDB.Exec("INSERT INTO media (path, time_first_played, time_last_played) VALUES (?, ?, ?)",
		path, 2_000_000_000, 2_000_000_000)

	queries := db.New(sqlDB)
	if err := queries.RewritePlayHistory(context.Background(), db.UpdatePlayHistoryParams{
		Path:            path,
		TimeLastPlayed:  sql.NullInt64{Int64: 1_000_000_000, Valid: true},
		TimeFirstPlayed: sql.NullInt64{Int64: 1_000_000_000, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}

	var first, last int64
	if err := sqlDB.QueryRow(
		"SELECT time_first_played, time_last_played FROM media WHERE path = ?",
		path,
	).Scan(&first, &last); err != nil {
		t.Fatal(err)
	}
	if first != 1_000_000_000 || last != 1_000_000_000 {
		t.Errorf("Expected explicit rewrite to force timestamps, got first=%d last=%d", first, last)
	}
}

func TestDeleteHistoryByPaths(t *testing.T) {
	sqlDB, dbPath := setupTestDB(t)
	defer os.Remove(dbPath)
	defer sqlDB.Close()

	path := "todelete.mp4"
	sqlDB.Exec("INSERT INTO media (path, play_count) VALUES (?, 5)", path)
	sqlDB.Exec("INSERT INTO history (media_path, playhead) VALUES (?, 100)", path)

	if err := history.DeleteHistoryByPaths(context.Background(), dbPath, []string{path}); err != nil {
		t.Fatal(err)
	}

	var count int
	sqlDB.QueryRow("SELECT COUNT(*) FROM history WHERE media_path = ?", path).Scan(&count)
	if count != 0 {
		t.Error("history.History record should be deleted")
	}

	var playCount int
	sqlDB.QueryRow("SELECT play_count FROM media WHERE path = ?", path).Scan(&playCount)
	if playCount != 0 {
		t.Error("Play count should be reset")
	}
}
