package commands

import (
	"database/sql"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestCategorizeCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath
	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Create files that should match some categories
	f1 := fixture.CreateDummyFile("football_match.mp4")
	f2 := fixture.CreateDummyFile("coding_tutorial.mp4")
	f3 := fixture.CreateDummyFile("random_file.mp4")

	sqlDB.Exec("INSERT INTO media (path, title) VALUES (?, ?)", f1, "Football Match")
	sqlDB.Exec("INSERT INTO media (path, title) VALUES (?, ?)", f2, "Go Programming")
	sqlDB.Exec("INSERT INTO media (path, title) VALUES (?, ?)", f3, "Just a file")

	// Add custom keyword since default categories are now disabled by default
	sqlDB.Exec("INSERT INTO custom_keywords (category, keyword) VALUES (?, ?)", "sports", "football")
	sqlDB.Close()

	t.Run("ApplyCategories", func(t *testing.T) {
		cmd := &CategorizeCmd{
			Databases: []string{dbPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("CategorizeCmd failed: %v", err)
		}

		// Verify categorization
		sqlDB, _ = sql.Open("sqlite3", dbPath)
		defer sqlDB.Close()
		var cat sql.NullString
		err := sqlDB.QueryRow("SELECT categories FROM media WHERE path = ?", f1).Scan(&cat)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if !cat.Valid || cat.String == "" {
			t.Error("Expected categories for football_match.mp4")
		}
	})

	t.Run("MineCategories", func(t *testing.T) {
		cmd := &CategorizeCmd{
			Databases: []string{dbPath},
			Other:     true,
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("CategorizeCmd (Other) failed: %v", err)
		}
	})
}
