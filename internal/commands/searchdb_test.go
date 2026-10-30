package commands_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func newSearchDB(t *testing.T, schema string, inserts ...string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "sdb-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	if _, err := dbConn.Exec(schema); err != nil {
		t.Fatal(err)
	}
	for _, insert := range inserts {
		if _, err := dbConn.Exec(insert); err != nil {
			t.Fatal(err)
		}
	}
	return dbPath
}

func rowCount(t *testing.T, dbPath, table string) int {
	t.Helper()

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	var count int
	if err := dbConn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestSearchDBCmd_Run(t *testing.T) {
	dbPath := newSearchDB(
		t,
		"CREATE TABLE test (name TEXT, val TEXT)",
		"INSERT INTO test VALUES ('apple', 'fruit'), ('carrot', 'vegetable')",
	)

	t.Run("FuzzyTableMatching", func(t *testing.T) {
		cmd := &commands.SearchDBCmd{
			JSON:     true,
			Database: dbPath,
			Table:    "tes", // fuzzy match for 'test'
			Terms:    []string{"apple"},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.SearchDBCmd failed: %v", err)
		}
	})

	t.Run("DeleteRows", func(t *testing.T) {
		cmd := &commands.SearchDBCmd{
			DeleteRows: true,
			Database:   dbPath,
			Table:      "test",
			Terms:      []string{"carrot"},
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.SearchDBCmd failed: %v", err)
		}

		if count := rowCount(t, dbPath, "test"); count != 1 {
			t.Errorf("Expected 1 row left, got %d", count)
		}
	})
}

func TestSearchDBCmd_MarkDeletedRows(t *testing.T) {
	dbPath := newSearchDB(
		t,
		"CREATE TABLE test (name TEXT, val TEXT, time_deleted INTEGER)",
		"INSERT INTO test VALUES ('apple', 'fruit', 0), ('banana', 'fruit', 0)",
	)

	cmd := &commands.SearchDBCmd{
		MarkDeleted: true,
		Database:    dbPath,
		Table:       "test",
		Terms:       []string{"banana"},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.SearchDBCmd failed: %v", err)
	}

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	dbConn.QueryRow("SELECT time_deleted FROM test WHERE name = 'banana'").Scan(&timeDeleted)
	if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
		t.Error("Expected row to be marked as deleted")
	}
}

func TestSearchDBCmd_WhereLimitsDelete(t *testing.T) {
	dbPath := newSearchDB(
		t,
		"CREATE TABLE test (name TEXT, val TEXT)",
		"INSERT INTO test VALUES ('apple', 'fruit'), ('apple', 'vegetable'), ('carrot', 'vegetable')",
	)

	cmd := &commands.SearchDBCmd{
		DeleteRows: true,
		Where:      []string{"val = 'fruit'"},
		Database:   dbPath,
		Table:      "test",
		Terms:      []string{"apple"},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.SearchDBCmd failed: %v", err)
	}

	if count := rowCount(t, dbPath, "test"); count != 2 {
		t.Errorf("Expected 2 rows left, got %d", count)
	}
}

func TestSearchDBCmd_NoPredicateRefusesDelete(t *testing.T) {
	schema := "CREATE TABLE numbers (n INTEGER)"
	dbPath := newSearchDB(t, schema, "INSERT INTO numbers VALUES (1), (2)")

	cmd := &commands.SearchDBCmd{
		DeleteRows: true,
		Database:   dbPath,
		Table:      "numbers",
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected delete without predicate to be refused")
	}
	if count := rowCount(t, dbPath, "numbers"); count != 2 {
		t.Errorf("Expected rows to remain, got %d", count)
	}

	cmd.AllRows = true
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("explicit --all-rows delete failed: %v", err)
	}
	if count := rowCount(t, dbPath, "numbers"); count != 0 {
		t.Errorf("Expected all rows deleted, got %d", count)
	}
}

func TestSearchDBCmd_SimulateLeavesRows(t *testing.T) {
	dbPath := newSearchDB(
		t,
		"CREATE TABLE test (name TEXT, time_deleted INTEGER)",
		"INSERT INTO test VALUES ('apple', 0), ('carrot', 0)",
	)

	cmd := &commands.SearchDBCmd{
		CoreFlags:  models.CoreFlags{Simulate: true},
		DeleteRows: true,
		Database:   dbPath,
		Table:      "test",
		Terms:      []string{"apple"},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.SearchDBCmd failed: %v", err)
	}
	if count := rowCount(t, dbPath, "test"); count != 2 {
		t.Errorf("Expected simulate to leave both rows, got %d", count)
	}

	markCmd := &commands.SearchDBCmd{
		CoreFlags:   models.CoreFlags{Simulate: true},
		MarkDeleted: true,
		Database:    dbPath,
		Table:       "test",
		Terms:       []string{"apple"},
	}
	if err := markCmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.SearchDBCmd mark-deleted simulate failed: %v", err)
	}

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	dbConn.QueryRow("SELECT time_deleted FROM test WHERE name = 'apple'").Scan(&timeDeleted)
	if timeDeleted.Int64 != 0 {
		t.Error("Expected simulate to leave time_deleted unchanged")
	}
}

func TestSearchDBCmd_SearchFlagIsApplied(t *testing.T) {
	dbPath := newSearchDB(
		t,
		"CREATE TABLE test (name TEXT)",
		"INSERT INTO test VALUES ('apple'), ('carrot')",
	)

	cmd := &commands.SearchDBCmd{
		DeleteRows: true,
		Search:     []string{"apple"},
		Database:   dbPath,
		Table:      "test",
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("commands.SearchDBCmd failed: %v", err)
	}
	if count := rowCount(t, dbPath, "test"); count != 1 {
		t.Errorf("Expected --search to limit deletion to 1 row, got %d left", count)
	}
}
