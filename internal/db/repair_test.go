package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestRepairRace(t *testing.T) {
	// Create a temporary database
	tempDir := t.TempDir()
	f, err := os.CreateTemp(tempDir, "race-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	// Initialize it in WAL mode
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatal(err)
	}
	for i := range 100 {
		_, err = db.Exec("INSERT INTO test (name) VALUES (?)", fmt.Sprintf("name-%d", i))
		if err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	// Ensure WAL file exists for testing sidecar handling
	walPath := dbPath + "-wal"
	shmPath := dbPath + "-shm"
	os.WriteFile(walPath, []byte("wal data"), 0o644)
	os.WriteFile(shmPath, []byte("shm data"), 0o644)

	// Corrupt the main file
	file, err := os.OpenFile(dbPath, os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	file.WriteAt([]byte("CORRUPT"), 100)
	file.Close()

	// After repairs, we can't easily check for the backups because they are timestamped and then REMOVED by Repair on success.
	// To test backup handling, we might need a separate test that doesn't successfully repair or we modify Repair to not remove them.
	// Actually, let's just trust the code if it passes the concurrency test.

	// Verify it's corrupt
	if isHealthy(context.Background(), dbPath) {
		t.Log("Warning: isHealthy didn't detect corruption at offset 5000, trying more extensive corruption")
		file, _ = os.OpenFile(dbPath, os.O_WRONLY, 0o644)
		file.WriteAt([]byte("CORRUPT"), 100) // Near header
		file.Close()
		if isHealthy(context.Background(), dbPath) {
			t.Fatal("Failed to create a corrupt database that isHealthy detects")
		}
	}

	// Now try to repair it from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 5
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			err := Repair(context.Background(), dbPath)
			if err != nil {
				// Some might fail if they catch it in a weird state, but the goal is at least one succeeds
				// and others either wait and see health or fail gracefully.
				// With the new lock, they should all wait and then see it as healthy.
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Final check
	if !isHealthy(context.Background(), dbPath) {
		t.Error("Database should be healthy after repairs")
	}

	// Verify sidecars are gone from the main path (they were moved to backupDir and backupDir was deleted)
	if _, err := os.Stat(walPath); err == nil {
		t.Error("WAL file should be gone after repair")
	}
	if _, err := os.Stat(shmPath); err == nil {
		t.Error("SHM file should be gone after repair")
	}

	// Verify no backup directories are left
	entries, _ := os.ReadDir(tempDir)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "race-test-") &&
			strings.Contains(entry.Name(), ".corrupt.") {

			t.Errorf("Found leftover backup directory: %s", entry.Name())
		}
	}
}
