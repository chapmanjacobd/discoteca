package commands_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestCheckCmd_FiltersRowsBeforeMarkingDeleted(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbConn := fixture.GetDB()
	if err := testutils.InitTestDB(t, dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}

	missingVideo := filepath.Join(fixture.TempDir, "missing-video.mp4")
	missingAudio := filepath.Join(fixture.TempDir, "missing-audio.mp3")
	if _, err := dbConn.Exec(
		"INSERT INTO media (path, media_type) VALUES (?, 'video'), (?, 'audio')",
		missingVideo,
		missingAudio,
	); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	dbConn.Close()

	cmd := &commands.CheckCmd{
		Args:             []string{fixture.DBPath},
		MediaFilterFlags: models.MediaFilterFlags{VideoOnly: true},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("CheckCmd failed: %v", err)
	}

	if got := timeDeletedFor(t, fixture.DBPath, missingVideo); got == 0 {
		t.Error("filtered video row was not marked deleted")
	}
	if got := timeDeletedFor(t, fixture.DBPath, missingAudio); got != 0 {
		t.Errorf("audio row outside the filter was marked deleted: %d", got)
	}
}
