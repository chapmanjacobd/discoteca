package commands

import (
	"database/sql"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/testutils"
)

func TestSimilarityCmds(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath
	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Create files that are similar in size/duration
	f1 := fixture.CreateDummyFile("video1.mp4")
	f2 := fixture.CreateDummyFile("video2.mp4")
	f3 := fixture.CreateDummyFile("video3.mp4")

	sqlDB.Exec("INSERT INTO media (path, size, duration) VALUES (?, ?, ?)", f1, 1000, 100)
	sqlDB.Exec("INSERT INTO media (path, size, duration) VALUES (?, ?, ?)", f2, 1005, 101)
	sqlDB.Exec("INSERT INTO media (path, size, duration) VALUES (?, ?, ?)", f3, 5000, 500)
	sqlDB.Close()

	t.Run("SimilarFilesCmd", func(t *testing.T) {
		cmd := &SimilarFilesCmd{
			Databases: []string{dbPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("SimilarFilesCmd failed: %v", err)
		}
	})

	t.Run("SimilarFoldersCmd", func(t *testing.T) {
		cmd := &SimilarFoldersCmd{
			Databases: []string{dbPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("SimilarFoldersCmd failed: %v", err)
		}
	})
}
