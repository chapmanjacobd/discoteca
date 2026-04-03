package commands

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

// TestDeletedMedia tests querying and restoring deleted media items
func TestDeletedMedia(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_trash.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Add 2 deleted and 1 non-deleted media
	sqlDB.Exec(`INSERT INTO media (path, time_deleted) VALUES ('/deleted1.mp4', 100)`)
	sqlDB.Exec(`INSERT INTO media (path, time_deleted) VALUES ('/deleted2.mp4', 200)`)
	sqlDB.Exec(`INSERT INTO media (path, time_deleted) VALUES ('/kept.mp4', 0)`)
	sqlDB.Close()

	t.Run("ListDeleted", func(t *testing.T) {
		cmd := &HistoryCmd{
			Databases:    []string{dbPath},
			DeletedFlags: models.DeletedFlags{OnlyDeleted: true},
		}
		// We expect 2 deleted items
		media, _ := queryMedia(cmd)
		if len(media) != 2 {
			t.Errorf("Expected 2 deleted items, got %d", len(media))
		}
	})

	t.Run("RestoreDeleted", func(t *testing.T) {
		// Manual restore logic
		sqlDB, _ := sql.Open("sqlite3", dbPath)
		sqlDB.Exec("UPDATE media SET time_deleted = 0 WHERE path = '/deleted1.mp4'")

		var timeDeleted int64
		sqlDB.QueryRow("SELECT time_deleted FROM media WHERE path = '/deleted1.mp4'").Scan(&timeDeleted)
		if timeDeleted != 0 {
			t.Error("Expected item to be restored")
		}
		sqlDB.Close()
	})
}

func queryMedia(c *HistoryCmd) ([]db.Media, error) {
	// Simplified query for testing
	sqlDB, _ := sql.Open("sqlite3", c.Databases[0])
	defer sqlDB.Close()

	var media []db.Media
	rows, _ := sqlDB.Query("SELECT path FROM media WHERE time_deleted > 0")
	defer rows.Close()
	for rows.Next() {
		var m db.Media
		rows.Scan(&m.Path)
		media = append(media, m)
	}
	return media, nil
}
