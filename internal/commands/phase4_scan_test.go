package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func runPrint(t *testing.T, args ...string) error {
	t.Helper()
	cmd := &commands.PrintCmd{Args: args}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}
	return cmd.Run(context.Background())
}

func TestPrintCmd_FileAndDirectoryScanPaths(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	file := fixture.CreateDummyFile("movie.mp4")
	if err := runPrint(t, file); err != nil {
		t.Fatalf("print file failed: %v", err)
	}

	dir := filepath.Join(fixture.TempDir, "library")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "episode.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runPrint(t, dir); err != nil {
		t.Fatalf("print directory failed: %v", err)
	}
}

func TestPrintCmd_EmptyScanPath(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	empty := filepath.Join(fixture.TempDir, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runPrint(t, empty); err != nil {
		t.Fatalf("print empty directory failed: %v", err)
	}
}

func TestPrintCmd_NonexistentScanPath(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	missing := filepath.Join(fixture.TempDir, "missing")
	if err := runPrint(t, missing); err == nil {
		t.Fatal("expected an error for a nonexistent scan path")
	}
}

func TestDiskUsageCmd_DirectoryScanPath(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dir := filepath.Join(fixture.TempDir, "library")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "movie.mp4"), []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := &commands.DiskUsageCmd{
		PrintCmd: commands.PrintCmd{Args: []string{dir}},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("du directory failed: %v", err)
	}
}
