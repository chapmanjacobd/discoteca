package commands

import (
	"context"
	"database/sql"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestSearchCaptionsCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath
	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Skip if FTS5 is not available
	var name string
	err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='captions_fts'").Scan(&name)
	if err != nil {
		t.Skip("FTS5 not available, skipping search captions tests")
	}

	// Insert test media with different types
	_, err = sqlDB.Exec(`
		INSERT INTO media (path, title, media_type, time_deleted)
		VALUES
		('/path/video1.mp4', 'Video 1', 'video', 0),
		('/path/video2.mp4', 'Video 2', 'video', 0),
		('/path/audio1.mp3', 'Audio 1', 'audio', 0),
		('/path/image1.jpg', 'Image 1', 'image', 0)
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert captions for different media types
	_, err = sqlDB.Exec(`
		INSERT INTO captions (media_path, time, text)
		VALUES
		('/path/video1.mp4', 10.0, 'hello world'),
		('/path/video1.mp4', 12.0, 'this is overlapping'),
		('/path/video1.mp4', 40.0, 'different world'),
		('/path/video2.mp4', 15.0, 'video caption world'),
		('/path/audio1.mp3', 15.0, 'audio caption world'),
		('/path/image1.jpg', 5.0, 'image caption world')
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Manual FTS rebuild to ensure captions are searchable
	_, err = sqlDB.Exec("INSERT INTO captions_fts(captions_fts) VALUES('rebuild')")
	if err != nil {
		t.Logf("FTS rebuild failed: %v", err)
	}
	sqlDB.Close()

	t.Run("BasicSearch", func(t *testing.T) {
		cmd := &SearchCaptionsCmd{
			Database: dbPath,
			Search:   []string{"world"},
		}

		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchCaptionsCmd failed: %v", err)
		}
	})

	t.Run("OverlapMerging", func(t *testing.T) {
		cmd := &SearchCaptionsCmd{
			Database: dbPath,
			Search:   []string{"world", "overlapping"},
			Overlap:  10,
		}

		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchCaptionsCmd failed: %v", err)
		}
	})

	t.Run("VideoOnlyFilter", func(t *testing.T) {
		cmd := &SearchCaptionsCmd{
			Database: dbPath,
			Search:   []string{"world"},
			MediaFilterFlags: models.MediaFilterFlags{
				VideoOnly: true,
			},
		}

		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchCaptionsCmd with VideoOnly failed: %v", err)
		}
	})

	t.Run("AudioOnlyFilter", func(t *testing.T) {
		cmd := &SearchCaptionsCmd{
			Database: dbPath,
			Search:   []string{"world"},
			MediaFilterFlags: models.MediaFilterFlags{
				AudioOnly: true,
			},
		}

		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchCaptionsCmd with AudioOnly failed: %v", err)
		}
	})

	t.Run("ImageOnlyFilter", func(t *testing.T) {
		cmd := &SearchCaptionsCmd{
			Database: dbPath,
			Search:   []string{"world"},
			MediaFilterFlags: models.MediaFilterFlags{
				ImageOnly: true,
			},
		}

		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchCaptionsCmd with ImageOnly failed: %v", err)
		}
	})
}
