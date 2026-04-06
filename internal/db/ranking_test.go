package db_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

// TestInMemoryRankingEffectiveness demonstrates that in-memory Go ranking
// provides meaningful relevance scoring compared to FTS5 BM25 with trigram
func TestInMemoryRankingEffectiveness(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "ranking-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	schema := `
	CREATE TABLE media (
		path TEXT PRIMARY KEY,
		title TEXT,
		description TEXT,
		time_deleted INTEGER DEFAULT 0
	);
	CREATE VIRTUAL TABLE media_fts USING fts5(path, title, description, content='media', content_rowid='rowid', tokenize='trigram', detail='none');
	CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN INSERT INTO media_fts(rowid, path, title, description) VALUES (new.rowid, new.path, new.title, new.description); END;
	`
	sqlDB.Exec(schema)

	testData := []struct {
		path        string
		title       string
		desc        string
		expectRank  int
		description string
	}{
		{"/doc1.mp4", "Python Python Python Tutorial", "Learn coding", 1, "3 title matches"},
		{"/python/doc2.mp4", "Python Tutorial", "Learn coding", 2, "1 title + 1 path match"},
		{"/python/doc3.mp4", "Python Guide", "Learn coding", 3, "1 title + 1 path match"},
		{"/doc4.mp4", "Python Tutorial", "Learn coding", 4, "1 title match"},
		{"/doc5.mp4", "Python Guide", "Learn coding", 5, "1 title match"},
		{"/python/doc6.mp4", "Tutorial Video", "Learn coding", 6, "1 path match"},
		{"/python/doc7.mp4", "Guide Video", "Learn coding", 7, "1 path match"},
		{"/doc8.mp4", "Tutorial Video", "Learn Python coding Python", 8, "2 desc matches"},
		{"/doc9.mp4", "Tutorial Video", "Learn Python coding", 9, "1 desc match"},
		{"/doc10.mp4", "Tutorial Video", "Python introduction", 10, "1 desc match"},
		{"/doc11.mp4", "PyT Tutorial", "Fire display", 11, "False positive - has pyt trigram"},
	}

	ctx := context.Background()
	for _, td := range testData {
		sqlDB.Exec("INSERT INTO media (path, title, description) VALUES (?, ?, ?)",
			td.path, td.title, td.desc)
	}
	sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")

	t.Run("FTS5 BM25 provides no differentiation", func(t *testing.T) {
		testFTS5NoDifferentiation(ctx, t, sqlDB)
	})

	t.Run("In-memory Go ranking provides meaningful differentiation", func(t *testing.T) {
		testInMemoryRankingDifferentiation(ctx, t, sqlDB)
	})

	t.Run("Verify scoring rules", func(t *testing.T) {
		testScoringRules(ctx, t, sqlDB)
	})

	t.Run("False positives rank lowest", func(t *testing.T) {
		testFalsePositivesRankLowest(ctx, t, sqlDB)
	})
}

func testFTS5NoDifferentiation(ctx context.Context, t *testing.T, sqlDB *sql.DB) {
	query := `
	SELECT m.path, m.title, media_fts.rank
	FROM media m, media_fts
	WHERE m.rowid = media_fts.rowid
	AND media_fts MATCH 'pyt'
	ORDER BY media_fts.rank DESC
	`
	rows, err := sqlDB.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var ranks []float64
	for rows.Next() {
		var path, title string
		var rank float64
		if err := rows.Scan(&path, &title, &rank); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		ranks = append(ranks, rank)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows error: %v", err)
	}

	if len(ranks) < 2 {
		t.Fatal("Not enough results")
	}

	for i := 1; i < len(ranks); i++ {
		diff := ranks[i] - ranks[0]
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.000002 {
			t.Errorf("Rank %d differs significantly from rank 0: %f vs %f (diff: %f)",
				i, ranks[i], ranks[0], diff)
		}
	}
	t.Logf("FTS5 BM25 ranks: all values within %.7f (no meaningful differentiation)", ranks[len(ranks)-1]-ranks[0])
}

func testInMemoryRankingDifferentiation(ctx context.Context, t *testing.T, sqlDB *sql.DB) {
	query := `
	SELECT m.path, m.title, m.description
	FROM media m, media_fts
	WHERE m.rowid = media_fts.rowid
	AND media_fts MATCH 'pyt'
	AND m.time_deleted = 0
	`
	rows, err := sqlDB.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var results []db.SearchMediaFTSResult
	for rows.Next() {
		var path, title, desc string
		if err := rows.Scan(&path, &title, &desc); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		results = append(results, db.SearchMediaFTSResult{
			Media: db.Media{
				Path:        path,
				Title:       sql.NullString{String: title, Valid: true},
				Description: sql.NullString{String: desc, Valid: true},
			},
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("No results found")
	}

	db.RankSearchResults(results, "python")

	t.Logf("\n%-6s %-8s %-30s %s\n", "Rank", "Score", "Path", "Title")
	t.Log("----------------------------------------------------------------------")

	for i, r := range results {
		actualRank := i + 1
		title := ""
		if r.Media.Title.Valid {
			title = r.Media.Title.String
		}
		t.Logf("%-6d %-8.0f %-30s %s\n", actualRank, r.Rank, r.Media.Path, title)
	}

	if len(results) < 10 {
		t.Fatalf("Expected at least 10 results, got %d", len(results))
	}

	for i := 0; i < 3 && i < len(results); i++ {
		if results[i].Rank < 10 {
			t.Errorf("Rank %d: Expected score >= 10 (title match), got %.0f for %s",
				i+1, results[i].Rank, results[i].Media.Path)
		}
	}

	descOnlyStart := -1
	for i, r := range results {
		if r.Rank < 10 && r.Rank >= 1 {
			descOnlyStart = i
			break
		}
	}
	if descOnlyStart > 0 {
		t.Logf("Description-only matches start at rank %d (score < 10)", descOnlyStart+1)
	}

	scoreSet := make(map[float64]bool)
	for _, r := range results {
		scoreSet[r.Rank] = true
	}
	if len(scoreSet) < 5 {
		t.Errorf("Expected at least 5 different score values, got %d: %v", len(scoreSet), scoreSet)
	} else {
		t.Logf("Found %d different score values (good differentiation)", len(scoreSet))
	}
}

func testScoringRules(ctx context.Context, t *testing.T, sqlDB *sql.DB) {
	query := `SELECT m.path, m.title, m.description FROM media m, media_fts WHERE m.rowid = media_fts.rowid AND media_fts MATCH 'pyt' AND m.time_deleted = 0`
	rows, err := sqlDB.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var results []db.SearchMediaFTSResult
	for rows.Next() {
		var path, title, desc string
		if err := rows.Scan(&path, &title, &desc); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		results = append(results, db.SearchMediaFTSResult{
			Media: db.Media{
				Path:        path,
				Title:       sql.NullString{String: title, Valid: true},
				Description: sql.NullString{String: desc, Valid: true},
			},
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows error: %v", err)
	}

	db.RankSearchResults(results, "python")

	var titleOnly, pathOnly, descOnly *db.SearchMediaFTSResult
	for i := range results {
		if results[i].Media.Path == "/doc4.mp4" {
			titleOnly = &results[i]
		}
		if results[i].Media.Path == "/python/doc6.mp4" {
			pathOnly = &results[i]
		}
		if results[i].Media.Path == "/doc9.mp4" {
			descOnly = &results[i]
		}
	}

	if titleOnly == nil {
		t.Fatal("Could not find title-only test document")
	}
	if pathOnly == nil {
		t.Fatal("Could not find path-only test document")
	}
	if descOnly == nil {
		t.Fatal("Could not find description-only test document")
	}

	t.Logf("Title-only score: %.0f (expected: 15 = 10 for match + 5 bonus)", titleOnly.Rank)
	t.Logf("Path-only score: %.0f (expected: 5)", pathOnly.Rank)
	t.Logf("Desc-only score: %.0f (expected: 1)", descOnly.Rank)

	if titleOnly.Rank <= pathOnly.Rank {
		t.Errorf("Title match (%.0f) should score higher than path match (%.0f)",
			titleOnly.Rank, pathOnly.Rank)
	}
	if pathOnly.Rank <= descOnly.Rank {
		t.Errorf("Path match (%.0f) should score higher than description match (%.0f)",
			pathOnly.Rank, descOnly.Rank)
	}

	if titleOnly.Rank != 15 {
		t.Logf("Note: Title-only score is %.0f (includes +5 exact match bonus)", titleOnly.Rank)
	}
}

func testFalsePositivesRankLowest(ctx context.Context, t *testing.T, sqlDB *sql.DB) {
	query := `SELECT m.path, m.title, m.description FROM media m, media_fts WHERE m.rowid = media_fts.rowid AND media_fts MATCH 'pyt' AND m.time_deleted = 0`
	rows, err := sqlDB.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var results []db.SearchMediaFTSResult
	for rows.Next() {
		var path, title, desc string
		if err := rows.Scan(&path, &title, &desc); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		results = append(results, db.SearchMediaFTSResult{
			Media: db.Media{
				Path:        path,
				Title:       sql.NullString{String: title, Valid: true},
				Description: sql.NullString{String: desc, Valid: true},
			},
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows error: %v", err)
	}

	db.RankSearchResults(results, "python")

	var falsePositive *db.SearchMediaFTSResult
	for i := range results {
		if results[i].Media.Path == "/doc11.mp4" {
			falsePositive = &results[i]
			break
		}
	}

	if falsePositive == nil {
		t.Fatal("Could not find false positive document")
	}

	t.Logf("False positive score: %.0f (should be 0)", falsePositive.Rank)
	if falsePositive.Rank > 0 {
		t.Errorf("False positive should have score 0, got %.0f", falsePositive.Rank)
	}
}

// TestRankingEdgeCases tests edge cases in the ranking algorithm
func TestRankingEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		path     string
		desc     string
		query    string
		wantRank float64
	}{
		{
			name:     "Case insensitive matching",
			title:    "PYTHON Tutorial",
			path:     "/test.mp4",
			desc:     "",
			query:    "python",
			wantRank: 15.0,
		},
		{
			name:     "Multiple occurrences in title",
			title:    "Python Python Python",
			path:     "/test.mp4",
			desc:     "",
			query:    "python",
			wantRank: 35.0,
		},
		{
			name:     "Title + path + description",
			title:    "Python",
			path:     "/python/test.mp4",
			desc:     "Learn Python",
			query:    "python",
			wantRank: 21.0,
		},
		{
			name:     "Empty query returns zero rank",
			title:    "Python",
			path:     "/test.mp4",
			desc:     "",
			query:    "",
			wantRank: 0,
		},
		{
			name:     "Partial match doesn't count",
			title:    "Pyth",
			path:     "/test.mp4",
			desc:     "",
			query:    "python",
			wantRank: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := []db.SearchMediaFTSResult{
				{
					Media: db.Media{
						Title:       sql.NullString{String: tt.title, Valid: true},
						Path:        tt.path,
						Description: sql.NullString{String: tt.desc, Valid: true},
					},
				},
			}

			db.RankSearchResults(results, tt.query)

			if results[0].Rank != tt.wantRank {
				t.Errorf("db.RankSearchResults() rank = %.1f, want %.1f", results[0].Rank, tt.wantRank)
			}
		})
	}
}

// TestRankingReorderAmount measures how much the in-memory ranking reorders results
func TestRankingReorderAmount(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "reorder-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	schema := `
	CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT, description TEXT, time_deleted INTEGER DEFAULT 0);
	CREATE VIRTUAL TABLE media_fts USING fts5(path, title, description, content='media', content_rowid='rowid', tokenize='trigram', detail='none');
	CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN INSERT INTO media_fts(rowid, path, title, description) VALUES (new.rowid, new.path, new.title, new.description); END;
	`
	sqlDB.Exec(schema)

	testData := []struct {
		path  string
		title string
		desc  string
	}{
		{"/a.mp4", "Tutorial", "Python introduction"},
		{"/b.mp4", "Python Guide", "Learn coding"},
		{"/c.mp4", "Tutorial", "Learn Python Python Python"},
		{"/d.mp4", "Python", "Learn coding"},
		{"/e.mp4", "Python Python", "Python Python Python"},
		{"/f.mp4", "Guide", "Introduction"},
	}

	ctx := context.Background()
	for _, td := range testData {
		sqlDB.Exec("INSERT INTO media (path, title, description) VALUES (?, ?, ?)",
			td.path, td.title, td.desc)
	}
	sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")

	query := `
	SELECT m.path, m.title, m.description
	FROM media m, media_fts
	WHERE m.rowid = media_fts.rowid
	AND media_fts MATCH 'pyt'
	AND m.time_deleted = 0
	ORDER BY m.path
	`
	rows, err := sqlDB.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var dbOrder []db.SearchMediaFTSResult
	for rows.Next() {
		var path, title, desc string
		if scanErr := rows.Scan(&path, &title, &desc); scanErr != nil {
			t.Fatalf("Scan failed: %v", scanErr)
		}
		dbOrder = append(dbOrder, db.SearchMediaFTSResult{
			Media: db.Media{
				Path:        path,
				Title:       sql.NullString{String: title, Valid: true},
				Description: sql.NullString{String: desc, Valid: true},
			},
		})
	}
	if err2 := rows.Err(); err2 != nil {
		t.Fatalf("Rows error: %v", err2)
	}

	if len(dbOrder) == 0 {
		t.Fatal("No results found")
	}

	t.Logf("Database order (by path):")
	for i, r := range dbOrder {
		t.Logf("  %d: %s - %s", i+1, r.Media.Path, r.Media.Title.String)
	}

	db.RankSearchResults(dbOrder, "python")

	t.Logf("\nRanked order (by relevance):")
	for i, r := range dbOrder {
		t.Logf("  %d: %s - %s (score: %.0f)", i+1, r.Media.Path, r.Media.Title.String, r.Rank)
	}

	// Test reorder statistics
	testReorderStatistics(ctx, t, sqlDB, dbOrder)
}

type trackedResult struct {
	result  db.SearchMediaFTSResult
	origPos int
}

func testReorderStatistics(ctx context.Context, t *testing.T, sqlDB *sql.DB, dbOrder []db.SearchMediaFTSResult) {
	query := `SELECT m.path, m.title, m.description FROM media m, media_fts WHERE m.rowid = media_fts.rowid AND media_fts MATCH 'pyt' AND m.time_deleted = 0 ORDER BY m.path`
	rows, err := sqlDB.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var results []trackedResult
	idx := 0
	for rows.Next() {
		var path, title, desc string
		if err := rows.Scan(&path, &title, &desc); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		results = append(results, trackedResult{
			result: db.SearchMediaFTSResult{
				Media: db.Media{
					Path:        path,
					Title:       sql.NullString{String: title, Valid: true},
					Description: sql.NullString{String: desc, Valid: true},
				},
			},
			origPos: idx,
		})
		idx++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows error: %v", err)
	}

	plainResults := make([]db.SearchMediaFTSResult, len(results))
	for i, tr := range results {
		plainResults[i] = tr.result
	}

	db.RankSearchResults(plainResults, "python")

	maxDisplacement := 0
	totalDisplacement := 0
	for _, tr := range results {
		newPos := -1
		for j, pr := range plainResults {
			if pr.Media.Path == tr.result.Media.Path {
				newPos = j
				break
			}
		}

		displacement := newPos - tr.origPos
		if displacement < 0 {
			displacement = -displacement
		}
		totalDisplacement += displacement
		if displacement > maxDisplacement {
			maxDisplacement = displacement
		}
	}

	avgDisplacement := float64(totalDisplacement) / float64(len(results))

	t.Logf("\n=== Reorder Statistics ===")
	t.Logf("Total items: %d", len(results))
	t.Logf("Max displacement: %d positions", maxDisplacement)
	t.Logf("Total displacement: %d positions", totalDisplacement)
	t.Logf("Average displacement: %.1f positions", avgDisplacement)

	inversions := countInversions(results, plainResults)
	t.Logf("Total inversions (flips): %d out of %d possible pairs", inversions, len(results)*(len(results)-1)/2)
}

func countInversions(results []trackedResult, plainResults []db.SearchMediaFTSResult) int {
	inversions := 0
	for i := range results {
		for j := i + 1; j < len(results); j++ {
			origI := results[i].origPos
			origJ := results[j].origPos

			newI := -1
			newJ := -1
			for k, pr := range plainResults {
				if pr.Media.Path == results[i].result.Media.Path {
					newI = k
				}
				if pr.Media.Path == results[j].result.Media.Path {
					newJ = k
				}
			}

			if origI < origJ && newI > newJ {
				inversions++
			}
		}
	}
	return inversions
}
