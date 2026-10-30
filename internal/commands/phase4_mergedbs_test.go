package commands_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func openTempDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func initTargetDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target.db")
	conn := openTempDB(t, path)
	defer conn.Close()
	if err := db.InitDB(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	return path
}

func createMediaSourceDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.db")
	conn := openTempDB(t, path)
	defer conn.Close()
	if _, err := conn.Exec("CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec("INSERT INTO media VALUES ('/a.mp4', 'A'), ('/b.mp4', 'B')"); err != nil {
		t.Fatal(err)
	}
	return path
}

func countTargetRows(t *testing.T, path, query string, args ...any) int {
	t.Helper()
	conn := openTempDB(t, path)
	defer conn.Close()
	var count int
	if err := conn.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestMergeDBsCmd_WhereAppliesToMediaOnly(t *testing.T) {
	source := createMediaSourceDB(t)
	target := initTargetDB(t)

	cmd := &commands.MergeDBsCmd{
		TargetDB:  target,
		SourceDBs: []string{source},
		MergeFlags: models.MergeFlags{
			OnlyTables: []string{"media"},
			WhereMedia: []string{"path = '/a.mp4'"},
		},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("MergeDBsCmd failed: %v", err)
	}

	if got := countTargetRows(t, target, "SELECT COUNT(*) FROM media"); got != 1 {
		t.Errorf("--where copied %d rows, want 1", got)
	}
	if got := countTargetRows(t, target, "SELECT COUNT(*) FROM media WHERE path = '/a.mp4'"); got != 1 {
		t.Errorf("--where did not copy the matching row")
	}
}

func TestMergeDBsCmd_SimulateWritesNothing(t *testing.T) {
	source := createMediaSourceDB(t)
	target := initTargetDB(t)

	cmd := &commands.MergeDBsCmd{
		TargetDB:   target,
		SourceDBs:  []string{source},
		CoreFlags:  models.CoreFlags{Simulate: true},
		MergeFlags: models.MergeFlags{OnlyTables: []string{"media"}},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("MergeDBsCmd failed: %v", err)
	}

	if got := countTargetRows(t, target, "SELECT COUNT(*) FROM media"); got != 0 {
		t.Errorf("--simulate wrote %d rows", got)
	}
}

func TestMergeDBsCmd_WhereTableForPlaylists(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "source.db")
	sourceConn := openTempDB(t, sourcePath)
	if err := db.InitDB(context.Background(), sourceConn); err != nil {
		sourceConn.Close()
		t.Fatal(err)
	}
	if _, err := sourceConn.Exec("INSERT INTO playlists (path) VALUES ('/keep'), ('/skip')"); err != nil {
		sourceConn.Close()
		t.Fatal(err)
	}
	sourceConn.Close()

	target := initTargetDB(t)
	cmd := &commands.MergeDBsCmd{
		TargetDB:  target,
		SourceDBs: []string{sourcePath},
		MergeFlags: models.MergeFlags{
			OnlyTables: []string{"playlists"},
			WhereTable: []string{"playlists=path = '/keep'"},
		},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("MergeDBsCmd failed: %v", err)
	}

	if got := countTargetRows(t, target, "SELECT COUNT(*) FROM playlists"); got != 1 {
		t.Errorf("--where-table copied %d playlists, want 1", got)
	}
}

func TestMergeDBsCmd_FailedTableReturnsError(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "source.db")
	sourceConn := openTempDB(t, sourcePath)
	if _, err := sourceConn.Exec("CREATE TABLE orphan (id INTEGER PRIMARY KEY)"); err != nil {
		sourceConn.Close()
		t.Fatal(err)
	}
	if _, err := sourceConn.Exec("INSERT INTO orphan (id) VALUES (1)"); err != nil {
		sourceConn.Close()
		t.Fatal(err)
	}
	sourceConn.Close()

	target := initTargetDB(t)
	cmd := &commands.MergeDBsCmd{
		TargetDB:   target,
		SourceDBs:  []string{sourcePath},
		MergeFlags: models.MergeFlags{OnlyTables: []string{"orphan"}},
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected an error when a table cannot be merged")
	}
}

func TestMergeDBsCmd_UpsertWithoutPrimaryKeyFailsBeforeCopy(t *testing.T) {
	source := createMediaSourceDB(t)
	target := initTargetDB(t)

	cmd := &commands.MergeDBsCmd{
		TargetDB:  target,
		SourceDBs: []string{source},
		MergeFlags: models.MergeFlags{
			OnlyTables:  []string{"media"},
			Upsert:      true,
			SkipColumns: []string{"path"},
		},
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected --upsert with a skipped primary key to fail")
	}
	if got := countTargetRows(t, target, "SELECT COUNT(*) FROM media"); got != 0 {
		t.Errorf("upsert failure still copied %d rows", got)
	}
}
