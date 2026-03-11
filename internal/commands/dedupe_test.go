package commands

import (
	"database/sql"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestDedupeCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath
	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	f1 := fixture.CreateDummyFile("video1.mp4")
	f2 := fixture.CreateDummyFile("video2.mp4")

	sqlDB.Exec("INSERT INTO media (path, title, duration, size) VALUES (?, ?, ?, ?)", f1, "Same Title", 100, 1000)
	sqlDB.Exec("INSERT INTO media (path, title, duration, size) VALUES (?, ?, ?, ?)", f2, "Same Title", 100, 1000)
	sqlDB.Close()

	t.Run("TitleDedupe", func(t *testing.T) {
		cmd := &DedupeCmd{
			Databases: []string{dbPath},
			CoreFlags: models.CoreFlags{NoConfirm: true},
			DedupeFlags: models.DedupeFlags{
				TitleOnly: true,
			},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("DedupeCmd failed: %v", err)
		}
	})

	t.Run("DurationDedupe", func(t *testing.T) {
		cmd := &DedupeCmd{
			Databases: []string{dbPath},
			CoreFlags: models.CoreFlags{NoConfirm: true},
			DedupeFlags: models.DedupeFlags{
				DurationOnly: true,
			},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("DedupeCmd failed: %v", err)
		}
	})
}
