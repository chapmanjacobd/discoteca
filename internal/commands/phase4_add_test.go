package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func addMediaCount(t *testing.T, fixture *testutils.TestFixture) int {
	t.Helper()
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var count int
	if err := dbConn.QueryRow("SELECT COUNT(*) FROM media").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestAddCmd_ExtensionFilter(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	video := fixture.CreateDummyFile("video.mp4")
	audio := fixture.CreateDummyFile("audio.mp3")

	cmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, video, audio},
		MediaFilterFlags: models.MediaFilterFlags{
			Ext: []string{"mp4"},
		},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	if got := addMediaCount(t, fixture); got != 1 {
		t.Errorf("--ext should have added only the mp4, got %d rows", got)
	}
}

func TestAddCmd_SizeFilter(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	small := fixture.CreateDummyFile("small.mp4")
	large := filepath.Join(fixture.TempDir, "large.mp4")
	if err := os.WriteFile(large, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := &commands.AddCmd{
		Args:           []string{fixture.DBPath, small, large},
		AddFilterFlags: models.AddFilterFlags{Size: []string{">1KB"}},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	if got := addMediaCount(t, fixture); got != 1 {
		t.Errorf("--size should have added only the large file, got %d rows", got)
	}
}

func TestAddCmd_PathFilter(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	included := filepath.Join(fixture.TempDir, "included")
	if err := os.MkdirAll(included, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(included, "keep.mp4")
	skip := filepath.Join(fixture.TempDir, "skip.mp4")
	for _, path := range []string{keep, skip} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := &commands.AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
		PathFilterFlags: models.PathFilterFlags{
			Include: []string{included},
		},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	if got := addMediaCount(t, fixture); got != 1 {
		t.Errorf("--include should have added only the matching file, got %d rows", got)
	}
}
