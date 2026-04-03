package query

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

// TestMultiDBPagination verifies that limit and offset are applied correctly
// when querying multiple databases. The issue was that each database would
// return LIMIT results, causing the merged result to have LIMIT*num_dbs results
// instead of LIMIT total.
func TestMultiDBPagination(t *testing.T) {
	// Create two test databases
	dbPath1, err := createTestDB("multidb-test1-*.db")
	if err != nil {
		t.Fatalf("Failed to create test DB 1: %v", err)
	}
	defer os.Remove(dbPath1)

	dbPath2, err := createTestDB("multidb-test2-*.db")
	if err != nil {
		t.Fatalf("Failed to create test DB 2: %v", err)
	}
	defer os.Remove(dbPath2)

	ctx := context.Background()
	dbs := []string{dbPath1, dbPath2}

	t.Run("Limit applied globally across multiple DBs", func(t *testing.T) {
		// Each DB has 5 items, total 10 items
		// With limit=3, should get exactly 3 results, not 6 (3 per DB)
		flags := models.GlobalFlags{
			QueryFlags: models.QueryFlags{Limit: 3},
			SortFlags:  models.SortFlags{SortBy: "path"},
		}

		results, err := MediaQuery(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("MediaQuery failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results with limit=3, got %d", len(results))
		}
	})

	t.Run("Offset works correctly with multiple DBs", func(t *testing.T) {
		// Each DB has 5 items, total 10 items
		// With limit=3, offset=2, should get items 3-5 (0-indexed: 2,3,4)
		flags := models.GlobalFlags{
			QueryFlags: models.QueryFlags{Limit: 3, Offset: 2},
			SortFlags:  models.SortFlags{SortBy: "path"},
		}

		results, err := MediaQuery(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("MediaQuery failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results with limit=3 offset=2, got %d", len(results))
		}

		// Verify we're getting the correct items (not the first 3)
		// First DB items: /db1/item1, /db1/item2, /db1/item3, /db1/item4, /db1/item5
		// Second DB items: /db2/item1, /db2/item2, /db2/item3, /db2/item4, /db2/item5
		// Sorted: /db1/item1, /db1/item2, /db1/item3, /db1/item4, /db1/item5, /db2/item1, ...
		// With offset=2, limit=3: /db1/item3, /db1/item4, /db1/item5
		if len(results) > 0 && results[0].Path == "/db1/item1" {
			t.Error("Expected offset to skip first items, but got first item")
		}
	})

	t.Run("No limit returns all results", func(t *testing.T) {
		flags := models.GlobalFlags{
			SortFlags: models.SortFlags{SortBy: "path"},
		}

		results, err := MediaQuery(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("MediaQuery failed: %v", err)
		}

		// Each DB has 5 items
		expected := 10
		if len(results) != expected {
			t.Errorf("Expected %d results with no limit, got %d", expected, len(results))
		}
	})

	t.Run("All flag returns all results", func(t *testing.T) {
		flags := models.GlobalFlags{
			QueryFlags: models.QueryFlags{All: true},
			SortFlags:  models.SortFlags{SortBy: "path"},
		}

		results, err := MediaQuery(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("MediaQuery failed: %v", err)
		}

		// Each DB has 5 items
		expected := 10
		if len(results) != expected {
			t.Errorf("Expected %d results with all=true, got %d", expected, len(results))
		}
	})

	t.Run("Large offset returns empty", func(t *testing.T) {
		flags := models.GlobalFlags{
			QueryFlags: models.QueryFlags{Limit: 3, Offset: 100}, // More than total items
			SortFlags:  models.SortFlags{SortBy: "path"},
		}

		results, err := MediaQuery(ctx, dbs, flags)
		if err != nil {
			t.Fatalf("MediaQuery failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results with offset=100, got %d", len(results))
		}
	})

	t.Run("Single DB pagination unchanged", func(t *testing.T) {
		// Verify single DB behavior is not affected by the fix
		singleDB := []string{dbPath1}
		flags := models.GlobalFlags{
			QueryFlags: models.QueryFlags{Limit: 3, Offset: 1},
			SortFlags:  models.SortFlags{SortBy: "path"},
		}

		results, err := MediaQuery(ctx, singleDB, flags)
		if err != nil {
			t.Fatalf("MediaQuery failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results with limit=3, got %d", len(results))
		}
	})
}

// createTestDB creates a test database with 5 sample items
func createTestDB(pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	dbPath := f.Name()
	f.Close()

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.Remove(dbPath)
		return "", err
	}
	defer dbConn.Close()

	// Create schema using central schema (without FTS)
	if err := testutils.InitTestDBNoFTS(dbConn); err != nil {
		os.Remove(dbPath)
		return "", err
	}

	// Insert 5 items
	dbName := dbPath[len(dbPath)-6:] // Get unique part of filename
	for i := 1; i <= 5; i++ {
		path := "/" + dbName + "/item" + string(rune('0'+i))
		_, err := dbConn.Exec(
			"INSERT INTO media (path, title, media_type, size, duration) VALUES (?, ?, ?, ?, ?)",
			path, "Item "+string(rune('0'+i)), "video", 1000*i, 100*i,
		)
		if err != nil {
			os.Remove(dbPath)
			return "", err
		}
	}

	return dbPath, nil
}
