package commands_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestOptimizeCmd_SimulateLeavesDatabaseUnchanged(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbConn := fixture.GetDB()
	if err := testutils.InitTestDB(t, dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	if _, err := dbConn.Exec("INSERT INTO media (path, media_type) VALUES ('/movie.mp4', NULL)"); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	dbConn.Close()

	cmd := &commands.OptimizeCmd{
		Databases: []string{fixture.DBPath},
		CoreFlags: models.CoreFlags{Simulate: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("OptimizeCmd failed: %v", err)
	}

	check := fixture.GetDB()
	defer check.Close()
	var mediaType sql.NullString
	if err := check.QueryRow("SELECT media_type FROM media WHERE path = '/movie.mp4'").Scan(&mediaType); err != nil {
		t.Fatal(err)
	}
	if mediaType.Valid {
		t.Errorf("--simulate backfilled media_type to %q", mediaType.String)
	}
}

func TestOptimizeCmd_BackfillOnlyFillsNullMediaTypes(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbConn := fixture.GetDB()
	if err := testutils.InitTestDB(t, dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	if _, err := dbConn.Exec(`
		INSERT INTO media (path, media_type) VALUES
			('/null.mp4', NULL),
			('/existing.mp4', 'audio'),
			('/text.epub', NULL)
	`); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}

	cmd := &commands.OptimizeCmd{}
	if err := cmd.BackfillMediaTypes(context.Background(), dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}

	var nullType, existingType, textType string
	if err := dbConn.QueryRow("SELECT media_type FROM media WHERE path = '/null.mp4'").Scan(&nullType); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	if err := dbConn.QueryRow("SELECT media_type FROM media WHERE path = '/existing.mp4'").Scan(&existingType); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	if err := dbConn.QueryRow("SELECT media_type FROM media WHERE path = '/text.epub'").Scan(&textType); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	dbConn.Close()

	if nullType != "video" {
		t.Errorf("NULL mp4 media_type = %q, want video", nullType)
	}
	if existingType != "audio" {
		t.Errorf("existing media_type was overwritten: %q", existingType)
	}
	if textType != "text" {
		t.Errorf("NULL epub media_type = %q, want text", textType)
	}
}
