package db

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestConnect(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "db-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	db, err := Connect(dbPath)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestIsCorruptionError(t *testing.T) {
	if IsCorruptionError(nil) {
		t.Error("nil should not be corruption error")
	}
	if !IsCorruptionError(errors.New("database disk image is malformed")) {
		t.Error("Expected corruption error")
	}
	if IsCorruptionError(errors.New("other error")) {
		t.Error("other error should not be corruption error")
	}
}

func TestIsHealthy(t *testing.T) {
	// Unhealthy test
	f, _ := os.CreateTemp(t.TempDir(), "unhealthy-test-*.db")
	unhealthyPath := f.Name()
	f.WriteString("Not a SQLite database")
	f.Close()
	defer os.Remove(unhealthyPath)

	if isHealthy(unhealthyPath) {
		t.Error("Garbage file should not be healthy DB")
	}

	// Healthy test
	f2, _ := os.CreateTemp(t.TempDir(), "healthy-test-*.db")
	healthyPath := f2.Name()
	f2.Close()
	defer os.Remove(healthyPath)

	db, _ := sql.Open("sqlite3", healthyPath)
	db.Exec("CREATE TABLE t(id INT)")
	db.Close()

	if !isHealthy(healthyPath) {
		t.Error("Valid DB should be healthy")
	}

	if isHealthy("/non/existent/path") {
		t.Error("Non-existent path should not be healthy")
	}
}
