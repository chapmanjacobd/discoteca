package commands_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func setupDedupe(t *testing.T) (*testutils.TestFixture, []string) {
	t.Helper()

	fixture := testutils.Setup(t)
	sqlDB, err := sql.Open("sqlite3", fixture.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		sqlDB.Close()
		t.Fatal(err)
	}

	f1 := fixture.CreateDummyFile("video1.mp4")
	f2 := fixture.CreateDummyFile("video2.mp4")
	if _, err := sqlDB.Exec(
		"INSERT INTO media (path, title, duration, size) VALUES (?, ?, ?, ?)",
		f1,
		"Same Title",
		100,
		1000,
	); err != nil {
		sqlDB.Close()
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		"INSERT INTO media (path, title, duration, size) VALUES (?, ?, ?, ?)",
		f2,
		"Same Title",
		100,
		1000,
	); err != nil {
		sqlDB.Close()
		t.Fatal(err)
	}
	sqlDB.Close()
	return fixture, []string{f1, f2}
}

func dedupeColumns(t *testing.T, dbPath, path string) (timeDeleted, isDeduped int64) {
	t.Helper()

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	if err := dbConn.QueryRow(
		"SELECT COALESCE(time_deleted, 0), COALESCE(is_deduped, 0) FROM media WHERE path = ?",
		path,
	).Scan(&timeDeleted, &isDeduped); err != nil {
		t.Fatal(err)
	}
	return timeDeleted, isDeduped
}

func TestDedupeCmd_Run(t *testing.T) {
	fixture, paths := setupDedupe(t)
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases: []string{fixture.DBPath},
		CoreFlags: models.CoreFlags{NoConfirm: true},
		DedupeFlags: models.DedupeFlags{
			TitleOnly: true,
		},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.DedupeCmd failed: %v", err)
	}

	remaining := 0
	deletedRows := int64(0)
	dedupedRows := int64(0)
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			remaining++
		}
		timeDeleted, isDeduped := dedupeColumns(t, fixture.DBPath, path)
		deletedRows += timeDeleted
		dedupedRows += isDeduped
	}

	if remaining != 1 {
		t.Errorf("Expected exactly one file to remain, got %d", remaining)
	}
	if deletedRows == 0 {
		t.Error("Expected the removed duplicate row to have time_deleted set")
	}
	if dedupedRows != 1 {
		t.Errorf("Expected exactly one keep candidate to be marked deduped, got %d", dedupedRows)
	}
}

func TestDedupeCmd_SimulateChangesNothing(t *testing.T) {
	fixture, paths := setupDedupe(t)
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases:   []string{fixture.DBPath},
		CoreFlags:   models.CoreFlags{Simulate: true},
		DedupeFlags: models.DedupeFlags{TitleOnly: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.DedupeCmd failed: %v", err)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("simulate removed %s", path)
		}
		timeDeleted, isDeduped := dedupeColumns(t, fixture.DBPath, path)
		if timeDeleted != 0 {
			t.Errorf("simulate changed time_deleted for %s to %d", path, timeDeleted)
		}
		if isDeduped != 0 {
			t.Errorf("simulate changed is_deduped for %s to %d", path, isDeduped)
		}
	}
}

func TestDedupeCmd_FailedRemoverKeepsRowActive(t *testing.T) {
	fixture, paths := setupDedupe(t)
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases:   []string{fixture.DBPath},
		CoreFlags:   models.CoreFlags{NoConfirm: true},
		DedupeFlags: models.DedupeFlags{TitleOnly: true, DedupeCmd: "false"},
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected a failing dedupe command to return an error")
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("failing remover removed %s", path)
		}
		timeDeleted, isDeduped := dedupeColumns(t, fixture.DBPath, path)
		if timeDeleted != 0 {
			t.Errorf("failing remover set time_deleted for %s", path)
		}
		if isDeduped != 0 {
			t.Errorf("failing remover set is_deduped for %s", path)
		}
	}
}

func TestDedupeCmd_SimulateFSCandidateCollectionDoesNotSetDeduped(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	sqlDB, err := sql.Open("sqlite3", fixture.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		sqlDB.Close()
		t.Fatal(err)
	}
	f1 := fixture.CreateDummyFile("same1.bin")
	f2 := fixture.CreateDummyFile("same2.bin")
	for _, path := range []string{f1, f2} {
		if _, err := sqlDB.Exec("INSERT INTO media (path, size) VALUES (?, ?)", path, 10); err != nil {
			sqlDB.Close()
			t.Fatal(err)
		}
	}
	sqlDB.Close()

	cmd := &commands.DedupeCmd{
		Databases:   []string{fixture.DBPath},
		CoreFlags:   models.CoreFlags{Simulate: true},
		DedupeFlags: models.DedupeFlags{Filesystem: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.DedupeCmd failed: %v", err)
	}

	for _, path := range []string{f1, f2} {
		_, isDeduped := dedupeColumns(t, fixture.DBPath, path)
		if isDeduped != 0 {
			t.Errorf("candidate collection set is_deduped for %s", path)
		}
	}
}

func TestDedupeCmd_TrashSimulateDoesNotInvokeTrash(t *testing.T) {
	fixture, _ := setupDedupe(t)
	defer fixture.Cleanup()

	binDir := t.TempDir()
	marker := filepath.Join(binDir, "trash-called")
	script := "#!/bin/sh\ntouch " + marker + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "trash-put"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cmd := &commands.DedupeCmd{
		Databases: []string{fixture.DBPath},
		CoreFlags: models.CoreFlags{Simulate: true},
		DedupeFlags: models.DedupeFlags{
			TitleOnly: true,
		},
		PostActionFlags: models.PostActionFlags{Trash: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.DedupeCmd failed: %v", err)
	}

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("trash command was invoked during a dry run")
	}
}
