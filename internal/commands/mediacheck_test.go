package commands_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func addMediaForMediaCheck(t *testing.T, fixture *testutils.TestFixture, name string) string {
	t.Helper()

	f := fixture.CreateDummyFile(name)
	addCmd := &commands.AddCmd{Args: []string{fixture.DBPath, f}}
	if err := addCmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := addCmd.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	return f
}

func installFakeFFProbe(t *testing.T, output string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script mock is not supported on Windows")
	}

	binDir := t.TempDir()
	script := "#!/bin/sh\ncat <<'EOF'\n" + output + "\nEOF\n"
	if err := os.WriteFile(filepath.Join(binDir, "ffprobe"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func mediaTimeDeleted(t *testing.T, dbPath, path string) int64 {
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
	return timeDeleted.Int64
}

func TestMediaCheckCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")

	addCmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(context.Background())

	t.Run("QuickScan", func(t *testing.T) {
		cmd := &commands.MediaCheckCmd{
			Databases: []string{fixture.DBPath},
		}
		// This will likely fail because ffmpeg is not there or file is invalid
		// but we want to see if the code runs.
		cmd.Run(context.Background())
	})

	t.Run("FullScan", func(t *testing.T) {
		cmd := &commands.MediaCheckCmd{
			Databases: []string{fixture.DBPath},
			FullScan:  true,
		}
		cmd.Run(context.Background())
	})
}

func TestMediaCheckCmd_ToolFailureDoesNotDelete(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f := addMediaForMediaCheck(t, fixture, "invalid.mp4")
	installFakeFFProbe(t, "not json")

	cmd := &commands.MediaCheckCmd{
		Databases:     []string{fixture.DBPath},
		FullScan:      true,
		DeleteCorrupt: "0.1",
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.MediaCheckCmd failed: %v", err)
	}

	if _, err := os.Stat(f); err != nil {
		t.Error("tool failure deleted a file")
	}
	if got := mediaTimeDeleted(t, fixture.DBPath, f); got != 0 {
		t.Errorf("tool failure marked file deleted: %d", got)
	}
}

func TestMediaCheckCmd_AboveThresholdDeletes(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f := addMediaForMediaCheck(t, fixture, "corrupt.mp4")
	installFakeFFProbe(t, `{"streams":[{"r_frame_rate":"30/1","nb_read_frames":"1500"}],"format":{"duration":"100.0"}}`)

	cmd := &commands.MediaCheckCmd{
		Databases:     []string{fixture.DBPath},
		FullScan:      true,
		DeleteCorrupt: "0.4",
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.MediaCheckCmd failed: %v", err)
	}

	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("expected above-threshold file to be deleted")
	}
	if got := mediaTimeDeleted(t, fixture.DBPath, f); got == 0 {
		t.Error("expected above-threshold file to be marked deleted")
	}
}

func TestMediaCheckCmd_AboveThresholdSimulateKeepsFile(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f := addMediaForMediaCheck(t, fixture, "corrupt-sim.mp4")
	installFakeFFProbe(t, `{"streams":[{"r_frame_rate":"30/1","nb_read_frames":"1500"}],"format":{"duration":"100.0"}}`)

	cmd := &commands.MediaCheckCmd{
		CoreFlags:     models.CoreFlags{Simulate: true},
		Databases:     []string{fixture.DBPath},
		FullScan:      true,
		DeleteCorrupt: "0.4",
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.MediaCheckCmd failed: %v", err)
	}

	if _, err := os.Stat(f); err != nil {
		t.Error("simulated mediacheck deleted a file")
	}
	if got := mediaTimeDeleted(t, fixture.DBPath, f); got != 0 {
		t.Errorf("simulated mediacheck marked file deleted: %d", got)
	}
}
