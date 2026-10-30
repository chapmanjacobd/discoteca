package db_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

func newInitializedDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "maintenance.db")
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	return sqlDB, dbPath
}

func insertMedia(t *testing.T, sqlDB *sql.DB, path string, size, duration int64) {
	t.Helper()
	if _, err := sqlDB.Exec(
		"INSERT INTO media (path, size, duration) VALUES (?, ?, ?)",
		path, size, duration,
	); err != nil {
		t.Fatal(err)
	}
}

func folderStat(t *testing.T, sqlDB *sql.DB, parent string) (count, size, duration int64, found bool) {
	t.Helper()
	err := sqlDB.QueryRow(
		"SELECT file_count, total_size, total_duration FROM folder_stats WHERE parent = ?",
		parent,
	).Scan(&count, &size, &duration)
	if err == sql.ErrNoRows {
		return 0, 0, 0, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return count, size, duration, true
}

func dirtyBit(t *testing.T, sqlDB *sql.DB) int {
	t.Helper()
	var dirty int
	if err := sqlDB.QueryRow("SELECT dirty FROM folder_stats_meta WHERE id = 1").Scan(&dirty); err != nil {
		t.Fatal(err)
	}
	return dirty
}

func TestFolderStatsDirtyBitAndRefresh(t *testing.T) {
	ctx := context.Background()
	sqlDB, dbPath := newInitializedDB(t)

	insertMedia(t, sqlDB, "/media/a/one.mkv", 100, 10)
	insertMedia(t, sqlDB, "/media/a/two.mkv", 200, 20)
	insertMedia(t, sqlDB, "/media/b/three.mp3", 50, 5)

	if dirtyBit(t, sqlDB) != 1 {
		t.Fatal("dirty bit should be set after media inserts")
	}

	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		t.Fatalf("RefreshFolderStats failed: %v", err)
	}
	if dirtyBit(t, sqlDB) != 0 {
		t.Error("dirty bit should be cleared after a successful refresh")
	}

	count, size, duration, found := folderStat(t, sqlDB, "/media/a")
	if !found || count != 2 || size != 300 || duration != 30 {
		t.Errorf("/media/a = (%d, %d, %d, %v), want (2, 300, 30, true)", count, size, duration, found)
	}

	// Once FTS is current too, a clean folder_stats view should not request a
	// refresh.
	if err := db.RebuildFTS(ctx, sqlDB, dbPath); err != nil {
		t.Fatalf("RebuildFTS failed: %v", err)
	}
	needs, err := db.NeedsRefresh(ctx, sqlDB, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if needs {
		t.Error("NeedsRefresh = true for a clean database")
	}

	// A no-op refresh must not change rows.
	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	countAgain, sizeAgain, durationAgain, foundAgain := folderStat(t, sqlDB, "/media/a")
	if !foundAgain || countAgain != count || sizeAgain != size || durationAgain != duration {
		t.Errorf("no-op refresh changed /media/a: got (%d, %d, %d)", countAgain, sizeAgain, durationAgain)
	}

	// Adding a folder sets the dirty bit and a refresh must pick it up.
	insertMedia(t, sqlDB, "/media/c/four.mkv", 10, 1)
	if dirtyBit(t, sqlDB) != 1 {
		t.Error("dirty bit should be set after adding a folder")
	}
	needs, err = db.NeedsRefresh(ctx, sqlDB, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !needs {
		t.Error("NeedsRefresh = false after a media insert")
	}
	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if count, _, _, found := folderStat(t, sqlDB, "/media/c"); !found || count != 1 {
		t.Errorf("/media/c = (%d, found=%v), want (1, true)", count, found)
	}

	// Renaming a folder drops the stale parent and creates the new one.
	insertMedia(t, sqlDB, "/media/d/five.mkv", 7, 1)
	if _, err := sqlDB.Exec(
		"UPDATE media SET path = '/media/renamed/one.mkv' WHERE path = '/media/a/one.mkv'",
	); err != nil {
		t.Fatal(err)
	}
	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if count, _, _, found := folderStat(t, sqlDB, "/media/a"); !found || count != 1 {
		t.Errorf("/media/a after rename = (%d, found=%v), want (1, true)", count, found)
	}
	if count, _, _, found := folderStat(t, sqlDB, "/media/renamed"); !found || count != 1 {
		t.Errorf("/media/renamed = (%d, found=%v), want (1, true)", count, found)
	}

	// Deleting all media must remove all stale folder rows.
	if _, err := sqlDB.Exec("DELETE FROM media"); err != nil {
		t.Fatal(err)
	}
	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	var countRows int
	if err := sqlDB.QueryRow("SELECT count(*) FROM folder_stats").Scan(&countRows); err != nil {
		t.Fatal(err)
	}
	if countRows != 0 {
		t.Errorf("folder_stats has %d rows after deleting all media, want 0", countRows)
	}
}

func TestMigrateDoesNotRepopulateFolderStats(t *testing.T) {
	ctx := context.Background()
	sqlDB, _ := newInitializedDB(t)
	insertMedia(t, sqlDB, "/media/a/one.mkv", 100, 10)

	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	// Sentinel stands in for a row produced by a previous refresh.
	if _, err := sqlDB.Exec(
		"INSERT INTO folder_stats (parent, file_count, total_size, total_duration) VALUES ('/sentinel', 99, 0, 0)",
	); err != nil {
		t.Fatal(err)
	}

	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if _, _, _, found := folderStat(t, sqlDB, "/sentinel"); !found {
		t.Error("Migrate repopulated folder_stats even though it was not empty")
	}

	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if _, _, _, found := folderStat(t, sqlDB, "/sentinel"); found {
		t.Error("RefreshFolderStats did not delete stale folder rows")
	}
}
