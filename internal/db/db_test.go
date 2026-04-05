package db_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

func TestConnect(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "db-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	sqlDB, err := db.Connect(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestIsCorruptionError(t *testing.T) {
	if db.IsCorruptionError(nil) {
		t.Error("nil should not be corruption error")
	}
	if !db.IsCorruptionError(errors.New("database disk image is malformed")) {
		t.Error("Expected corruption error")
	}
	if db.IsCorruptionError(errors.New("other error")) {
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

	if db.IsHealthy(context.Background(), unhealthyPath) {
		t.Error("Garbage file should not be healthy DB")
	}

	// Healthy test
	f2, _ := os.CreateTemp(t.TempDir(), "healthy-test-*.db")
	healthyPath := f2.Name()
	f2.Close()
	defer os.Remove(healthyPath)

	sqlDB, _ := sql.Open("sqlite3", healthyPath)
	sqlDB.Exec("CREATE TABLE t(id INT)")
	sqlDB.Close()

	if !db.IsHealthy(context.Background(), healthyPath) {
		t.Error("Valid DB should be healthy")
	}

	if db.IsHealthy(context.Background(), "/non/existent/path") {
		t.Error("Non-existent path should not be healthy")
	}
}
