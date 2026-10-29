package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

func TestSetCacheSizeMB(t *testing.T) {
	t.Cleanup(func() { db.SetCacheSizeMB(256) })
	db.SetCacheSizeMB(32)

	if got := db.CacheSizeMB(); got != 32 {
		t.Fatalf("CacheSizeMB() = %d, want 32", got)
	}

	conn, err := db.Connect(context.Background(), filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var cacheSize int
	if err := conn.QueryRowContext(context.Background(), "PRAGMA cache_size").Scan(&cacheSize); err != nil {
		t.Fatal(err)
	}
	// SQLite reports a negative value as a KiB page count.
	if want := -32 * 1024; cacheSize != want {
		t.Errorf("PRAGMA cache_size = %d, want %d", cacheSize, want)
	}
}

func TestSetCacheSizeMBResetsInvalidValues(t *testing.T) {
	t.Cleanup(func() { db.SetCacheSizeMB(256) })
	db.SetCacheSizeMB(0)

	if got := db.CacheSizeMB(); got != 256 {
		t.Errorf("CacheSizeMB() = %d, want the default 256", got)
	}
}
