package query_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestMediaTypeAndEpisodicConstraint(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "repro-episodic-*.db")
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	dbConn, _ := sql.Open("sqlite3", dbPath)
	if err := testutils.InitTestDBNoFTS(dbConn); err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}
	// Directory 1: 1 video, 1 image
	dbConn.Exec("INSERT INTO media (path, media_type) VALUES ('/dir1/v1.mp4', 'video')")
	dbConn.Exec("INSERT INTO media (path, media_type) VALUES ('/dir1/i1.jpg', 'image')")
	// Directory 2: 2 videos
	dbConn.Exec("INSERT INTO media (path, media_type) VALUES ('/dir2/v2.mp4', 'video')")
	dbConn.Exec("INSERT INTO media (path, media_type) VALUES ('/dir2/v3.mp4', 'video')")
	dbConn.Close()

	ctx := context.Background()
	dbs := []string{dbPath}

	t.Run("Unconstrained FileCounts=1", func(t *testing.T) {
		// Should return nothing because both dirs have 2 files (global count)
		got, err := query.MediaQuery(
			ctx,
			dbs,
			models.GlobalFlags{AggregateFlags: models.AggregateFlags{FileCounts: "1"}},
		)
		if err != nil {
			t.Fatalf("query.MediaQuery failed: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("Expected 0 results, got %d", len(got))
		}
	})

	t.Run("Constrained VideoOnly and FileCounts=1", func(t *testing.T) {
		got, err := query.MediaQuery(ctx, dbs, models.GlobalFlags{
			MediaFilterFlags: models.MediaFilterFlags{VideoOnly: true},
			AggregateFlags:   models.AggregateFlags{FileCounts: "1"},
		})
		if err != nil {
			t.Fatalf("query.MediaQuery failed: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("Expected 1 result (the video in /dir1), got %d", len(got))
		} else if got[0].Path != "/dir1/v1.mp4" {
			t.Errorf("Expected /dir1/v1.mp4, got %s", got[0].Path)
		}
	})
}
