package commands

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestSearchDBCmd_Run(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "sdb-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	dbConn, _ := sql.Open("sqlite3", dbPath)
	dbConn.Exec("CREATE TABLE test (name TEXT, val TEXT)")
	dbConn.Exec("INSERT INTO test VALUES ('apple', 'fruit'), ('carrot', 'vegetable')")
	dbConn.Close()

	t.Run("FuzzyTableMatching", func(t *testing.T) {
		cmd := &SearchDBCmd{
			DisplayFlags: models.DisplayFlags{JSON: true},
			Database:     dbPath,
			Table:        "tes", // fuzzy match for 'test'
			Search:       []string{"apple"},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchDBCmd failed: %v", err)
		}
	})

	t.Run("DeleteRows", func(t *testing.T) {
		cmd := &SearchDBCmd{
			PostActionFlags: models.PostActionFlags{
				DeleteRows: true,
			},
			Database: dbPath,
			Table:    "test",
			Search:   []string{"carrot"},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchDBCmd failed: %v", err)
		}

		// Verify deletion
		dbConn, _ = sql.Open("sqlite3", dbPath)
		defer dbConn.Close()
		var count int
		dbConn.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 row left, got %d", count)
		}
	})

	t.Run("MarkDeletedRows", func(t *testing.T) {
		dbConn, _ := sql.Open("sqlite3", dbPath)
		dbConn.Exec("ALTER TABLE test ADD COLUMN time_deleted INTEGER")
		dbConn.Exec("INSERT INTO test (name, val) VALUES ('banana', 'fruit')")
		dbConn.Close()

		cmd := &SearchDBCmd{
			PostActionFlags: models.PostActionFlags{
				MarkDeleted: true,
			},
			Database: dbPath,
			Table:    "test",
			Search:   []string{"banana"},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("SearchDBCmd failed: %v", err)
		}

		dbConn, _ = sql.Open("sqlite3", dbPath)
		defer dbConn.Close()
		var timeDeleted sql.NullInt64
		dbConn.QueryRow("SELECT time_deleted FROM test WHERE name = 'banana'").Scan(&timeDeleted)
		if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
			t.Error("Expected row to be marked as deleted")
		}
	})
}
