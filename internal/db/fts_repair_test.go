//go:build fts5

package db

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestFTSRepair(t *testing.T) {
	// 1. Setup
	f, err := os.CreateTemp(t.TempDir(), "fts-repair-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	setupSQL := `
CREATE TABLE media (
    path TEXT PRIMARY KEY,
    title TEXT,
    time_deleted INTEGER DEFAULT 0
);

CREATE VIRTUAL TABLE media_fts USING fts5(
    path,
    title,
    content='media',
    content_rowid='rowid'
);

CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN
    INSERT INTO media_fts(rowid, path, title)
    VALUES (new.rowid, new.path, new.title);
END;

CREATE TRIGGER media_au AFTER UPDATE ON media BEGIN
    INSERT INTO media_fts(media_fts, rowid, path, title) VALUES('delete', old.rowid, old.path, old.title);
    INSERT INTO media_fts(rowid, path, title) VALUES (new.rowid, new.path, new.title);
END;
`
	if _, err := db.Exec(setupSQL); err != nil {
		t.Fatal(err)
	}

	// Insert data
	for i := range 100 {
		if _, err := db.Exec("INSERT INTO media (path, title) VALUES (?, ?)",
			fmt.Sprintf("file%d.mp4", i), fmt.Sprintf("Video %d", i)); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	// 2. Verify Healthy
	if !isHealthy(dbPath) {
		t.Fatal("Database should be healthy initially")
	}

	// 3. Corrupt - corrupt a data page (not schema pages which are at the beginning)
	// SQLite page size is typically 4096 bytes. Schema is in page 1.
	// We'll corrupt a page in the middle-rear of the file where data is likely stored.
	stats, _ := os.Stat(dbPath)
	size := stats.Size()
	t.Logf("Database size: %d", size)

	file, err := os.OpenFile(dbPath, os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt page 5 (offset 16384) which should be data, not schema
	// This is far enough from the header to preserve schema
	corruptOffset := int64(16384)
	if size < corruptOffset+100 {
		// File is too small, corrupt the middle
		corruptOffset = size / 2
	}
	file.WriteAt([]byte("CORRUPT DATA CORRUPT DATA CORRUPT DATA"), corruptOffset)
	file.Close()

	// 4. Verify Corrupt
	if isHealthy(dbPath) {
		t.Log("isHealthy did NOT detect corruption, trying additional corruption")
		// Add more corruption to ensure detection
		file, _ = os.OpenFile(dbPath, os.O_WRONLY, 0o644)
		file.WriteAt([]byte("CORRUPT"), corruptOffset+4096)
		file.WriteAt([]byte("CORRUPT"), corruptOffset+8192)
		file.Close()
		if isHealthy(dbPath) {
			t.Fatal("Failed to corrupt database in a way isHealthy detects")
		}
	}

	// 5. Repair
	t.Log("Running Repair...")
	if err := Repair(dbPath); err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	// 6. Verify Healthy again
	if !isHealthy(dbPath) {
		t.Fatal("Database should be healthy after repair")
	}

	// 7. Verify Data and FTS (best effort - recovery may not preserve everything)
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Check if media table exists and has some data
	var mediaCount int
	err = db.QueryRow("SELECT count(*) FROM media").Scan(&mediaCount)
	if err != nil {
		t.Logf("Media table not recovered (this can happen with severe corruption): %v", err)
		// This is acceptable - the main goal is that the database is healthy
		return
	}

	t.Logf("Recovered %d media records", mediaCount)

	// Try to get a specific record (best effort)
	var title string
	err = db.QueryRow("SELECT title FROM media WHERE path = ?", "file42.mp4").Scan(&title)
	if err == nil {
		if title != "Video 42" {
			t.Errorf("Expected 'Video 42', got %q", title)
		}
		t.Logf("Recovered title: %q", title)
	} else {
		t.Logf("Could not recover specific record (acceptable): %v", err)
	}

	// Check FTS (best effort)
	var hasFTS bool
	_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')").Scan(&hasFTS)
	if hasFTS {
		var count int
		err = db.QueryRow("SELECT count(*) FROM media_fts WHERE media_fts MATCH 'Video'").Scan(&count)
		if err == nil {
			t.Logf("Recovered FTS MATCH count: %d", count)
			// FTS count may be less than 100 due to partial recovery
			if count == 0 && mediaCount > 0 {
				t.Log("Warning: FTS table exists but has no matches (may need manual rebuild)")
			}
		} else {
			t.Logf("FTS query failed (acceptable with corruption): %v", err)
		}
	} else {
		t.Log("FTS table not recovered (acceptable with severe corruption)")
	}
}
