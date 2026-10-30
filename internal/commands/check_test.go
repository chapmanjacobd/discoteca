package commands_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func insertMedia(t *testing.T, dbPath, path string) {
	t.Helper()

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()
	if _, err := dbConn.Exec("INSERT INTO media (path, path_tokenized) VALUES (?, ?)", path, path); err != nil {
		t.Fatal(err)
	}
}

func setupCheckDB(t *testing.T) *testutils.TestFixture {
	t.Helper()

	fixture := testutils.Setup(t)
	dbConn := fixture.GetDB()
	if err := testutils.InitTestDB(t, dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	dbConn.Close()
	return fixture
}

func timeDeletedFor(t *testing.T, dbPath, path string) int64 {
	t.Helper()

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	var timeDeleted sql.NullInt64
	if err := dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", path).Scan(&timeDeleted); err != nil {
		t.Fatal(err)
	}
	if !timeDeleted.Valid {
		return 0
	}
	return timeDeleted.Int64
}

func TestCheckCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	// Delete file from FS
	os.Remove(f1)

	cmd := &commands.CheckCmd{
		Args: []string{fixture.DBPath},
	}
	cmd.AfterApply()
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.CheckCmd failed: %v", err)
	}

	// Verify it was marked as deleted
	if timeDeletedFor(t, fixture.DBPath, f1) == 0 {
		t.Errorf("Expected file to be marked as deleted")
	}
}

func TestCheckCmd_SimulateLeavesRowsUnchanged(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		name := "--simulate"
		cmd := &commands.CheckCmd{}
		if dryRun {
			name = "--dry-run"
			cmd.DryRun = true
		} else {
			cmd.Simulate = true
		}

		t.Run(name, func(t *testing.T) {
			fixture := setupCheckDB(t)
			defer fixture.Cleanup()

			root := filepath.Join(fixture.TempDir, "library")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "present.mp4"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			missing := filepath.Join(root, "missing.mp4")
			insertMedia(t, fixture.DBPath, missing)

			cmd.Args = []string{fixture.DBPath, root}
			if err := cmd.AfterApply(); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Run(context.Background()); err != nil {
				t.Fatalf("commands.CheckCmd failed: %v", err)
			}

			// The root is empty: even a real run must fail before marking rows.
			if got := timeDeletedFor(t, fixture.DBPath, missing); got != 0 {
				t.Errorf("Expected time_deleted to remain 0, got %d", got)
			}
		})
	}
}

func TestCheckCmd_EmptyRootFailsClosed(t *testing.T) {
	fixture := setupCheckDB(t)
	defer fixture.Cleanup()

	root := filepath.Join(fixture.TempDir, "library")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing.mp4")
	insertMedia(t, fixture.DBPath, missing)

	cmd := &commands.CheckCmd{Args: []string{fixture.DBPath, root}}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected an empty scan root to return an error")
	}
	if got := timeDeletedFor(t, fixture.DBPath, missing); got != 0 {
		t.Errorf("Expected time_deleted to remain 0, got %d", got)
	}
}

func TestCheckCmd_RootBoundary(t *testing.T) {
	fixture := setupCheckDB(t)
	defer fixture.Cleanup()

	root := filepath.Join(fixture.TempDir, "media", "foo")
	sibling := filepath.Join(fixture.TempDir, "media", "foobar")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "present.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(sibling, "outside.mp4")
	insertMedia(t, fixture.DBPath, outside)

	cmd := &commands.CheckCmd{Args: []string{fixture.DBPath, root}}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.CheckCmd failed: %v", err)
	}

	if got := timeDeletedFor(t, fixture.DBPath, outside); got != 0 {
		t.Errorf("path under %s should not be governed by %s", sibling, root)
	}
}

func TestCheckCmd_UnreadableRootFailsClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks are not meaningful as root")
	}

	fixture := setupCheckDB(t)
	defer fixture.Cleanup()

	root := filepath.Join(fixture.TempDir, "library")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing.mp4")
	insertMedia(t, fixture.DBPath, missing)
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(root, 0o755) }()

	cmd := &commands.CheckCmd{Args: []string{fixture.DBPath, root}}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected unreadable scan root to return an error")
	}
	if got := timeDeletedFor(t, fixture.DBPath, missing); got != 0 {
		t.Errorf("Expected time_deleted to remain 0, got %d", got)
	}
}

func TestCheckCmd_SimulateAndDryRunMatch(t *testing.T) {
	results := make(map[string]int64, 2)
	for _, mode := range []string{"simulate", "dry-run"} {
		fixture := setupCheckDB(t)
		root := filepath.Join(fixture.TempDir, "library")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "present.mp4"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		missing := filepath.Join(root, "missing.mp4")
		insertMedia(t, fixture.DBPath, missing)

		cmd := &commands.CheckCmd{Args: []string{fixture.DBPath, root}}
		if mode == "simulate" {
			cmd.CoreFlags = models.CoreFlags{Simulate: true}
		} else {
			cmd.DryRun = true
		}
		if err := cmd.AfterApply(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("%s run failed: %v", mode, err)
		}
		results[mode] = timeDeletedFor(t, fixture.DBPath, missing)
		fixture.Cleanup()
	}

	if results["simulate"] != results["dry-run"] {
		t.Errorf("simulate result %d != dry-run result %d", results["simulate"], results["dry-run"])
	}
}
