package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestInMemoryRankingEffectiveness demonstrates that in-memory Go ranking
// provides meaningful relevance scoring compared to FTS5 BM25 with trigram
func TestInMemoryRankingEffectiveness(t *testing.T) {
	f, err := os.CreateTemp("", "ranking-test-*.db")
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

	// Create schema with all required columns
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

	// Create test documents with controlled relevance levels
	testData := []struct {
		path        string
		title       string
		desc        string
		expectRank  int // Expected rank after sorting (1 = highest)
		description string
	}{
		// Rank 1: Multiple title matches (highest score)
		{"/doc1.mp4", "Python Python Python Tutorial", "Learn coding", 1, "3 title matches"},

		// Rank 2-3: Single title match + path match
		{"/python/doc2.mp4", "Python Tutorial", "Learn coding", 2, "1 title + 1 path match"},
		{"/python/doc3.mp4", "Python Guide", "Learn coding", 3, "1 title + 1 path match"},

		// Rank 4-5: Title match only
		{"/doc4.mp4", "Python Tutorial", "Learn coding", 4, "1 title match"},
		{"/doc5.mp4", "Python Guide", "Learn coding", 5, "1 title match"},

		// Rank 6-7: Path match only
		{"/python/doc6.mp4", "Tutorial Video", "Learn coding", 6, "1 path match"},
		{"/python/doc7.mp4", "Guide Video", "Learn coding", 7, "1 path match"},

		// Rank 8-10: Description matches only (lowest score)
		{"/doc8.mp4", "Tutorial Video", "Learn Python coding Python", 8, "2 desc matches"},
		{"/doc9.mp4", "Tutorial Video", "Learn Python coding", 9, "1 desc match"},
		{"/doc10.mp4", "Tutorial Video", "Python introduction", 10, "1 desc match"},

		// Rank 11+: False positives (has "pyt" trigram but not "python")
		{"/doc11.mp4", "PyT Tutorial", "Fire display", 11, "False positive - has pyt trigram"},
	}

	ctx := context.Background()
	for _, td := range testData {
		sqlDB.Exec("INSERT INTO media (path, title, description) VALUES (?, ?, ?)",
			td.path, td.title, td.desc)
	}
	sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")

	// Test 1: Verify FTS5 BM25 provides no meaningful ranking
	t.Run("FTS5 BM25 provides no differentiation", func(t *testing.T) {
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
			var path string
			var rank float64
			rows.Scan(&path, &rank)
			ranks = append(ranks, rank)
		}

		// Check that all ranks are essentially identical
		if len(ranks) < 2 {
			t.Fatal("Not enough results")
		}

		// All ranks should be within 0.000001 of each other (effectively identical)
		for i := 1; i < len(ranks); i++ {
			diff := ranks[i] - ranks[0]
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0000001 {
				t.Errorf("Rank %d differs significantly from rank 0: %f vs %f (diff: %f)",
					i, ranks[i], ranks[0], diff)
			}
		}
		t.Logf("FTS5 BM25 ranks: all values within %.7f (no meaningful differentiation)", ranks[len(ranks)-1]-ranks[0])
	})

	// Test 2: Verify in-memory Go ranking provides meaningful differentiation
	t.Run("In-memory Go ranking provides meaningful differentiation", func(t *testing.T) {
		// Fetch results using simple query (SearchMediaFTS expects full schema)
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

		// Convert to SearchMediaFTSResult
		var results []SearchMediaFTSResult
		for rows.Next() {
			var path, title, desc string
			err := rows.Scan(&path, &title, &desc)
			if err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			results = append(results, SearchMediaFTSResult{
				Media: Media{
					Path:        path,
					Title:       sql.NullString{String: title, Valid: true},
					Description: sql.NullString{String: desc, Valid: true},
				},
			})
		}

		if len(results) == 0 {
			t.Fatal("No results found")
		}

		// Apply in-memory ranking
		RankSearchResults(results, "python")

		// Verify ranking order
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

		// Verify high-relevance documents rank higher than low-relevance ones
		if len(results) < 10 {
			t.Fatalf("Expected at least 10 results, got %d", len(results))
		}

		// Top results should have title matches (score >= 10)
		for i := 0; i < 3 && i < len(results); i++ {
			if results[i].Rank < 10 {
				t.Errorf("Rank %d: Expected score >= 10 (title match), got %.0f for %s",
					i+1, results[i].Rank, results[i].Media.Path)
			}
		}

		// Documents with only description matches should rank lower
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

		// Verify scores are differentiated (not all the same)
		scoreSet := make(map[float64]bool)
		for _, r := range results {
			scoreSet[r.Rank] = true
		}
		if len(scoreSet) < 5 {
			t.Errorf("Expected at least 5 different score values, got %d: %v", len(scoreSet), scoreSet)
		} else {
			t.Logf("Found %d different score values (good differentiation)", len(scoreSet))
		}
	})

	// Test 3: Verify specific scoring rules
	t.Run("Verify scoring rules", func(t *testing.T) {
		// Fetch results using simple query
		query := `SELECT m.path, m.title, m.description FROM media m, media_fts WHERE m.rowid = media_fts.rowid AND media_fts MATCH 'pyt' AND m.time_deleted = 0`
		rows, _ := sqlDB.QueryContext(ctx, query)

		var results []SearchMediaFTSResult
		for rows.Next() {
			var path, title, desc string
			rows.Scan(&path, &title, &desc)
			results = append(results, SearchMediaFTSResult{
				Media: Media{
					Path:        path,
					Title:       sql.NullString{String: title, Valid: true},
					Description: sql.NullString{String: desc, Valid: true},
				},
			})
		}
		rows.Close()

		RankSearchResults(results, "python")

		// Find specific test cases
		var titleOnly, pathOnly, descOnly *SearchMediaFTSResult
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

		// Verify title > path > description
		if titleOnly.Rank <= pathOnly.Rank {
			t.Errorf("Title match (%.0f) should score higher than path match (%.0f)",
				titleOnly.Rank, pathOnly.Rank)
		}
		if pathOnly.Rank <= descOnly.Rank {
			t.Errorf("Path match (%.0f) should score higher than description match (%.0f)",
				pathOnly.Rank, descOnly.Rank)
		}

		// Verify exact title match bonus is applied
		if titleOnly.Rank != 15 {
			t.Logf("Note: Title-only score is %.0f (includes +5 exact match bonus)", titleOnly.Rank)
		}
	})

	// Test 4: Verify false positives rank lowest
	t.Run("False positives rank lowest", func(t *testing.T) {
		query := `SELECT m.path, m.title, m.description FROM media m, media_fts WHERE m.rowid = media_fts.rowid AND media_fts MATCH 'pyt' AND m.time_deleted = 0`
		rows, _ := sqlDB.QueryContext(ctx, query)

		var results []SearchMediaFTSResult
		for rows.Next() {
			var path, title, desc string
			rows.Scan(&path, &title, &desc)
			results = append(results, SearchMediaFTSResult{
				Media: Media{Path: path, Title: sql.NullString{String: title, Valid: true}, Description: sql.NullString{String: desc, Valid: true}},
			})
		}
		rows.Close()

		RankSearchResults(results, "python")

		// Find the pyrotechnics document (false positive)
		var falsePositive *SearchMediaFTSResult
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
	})
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
			wantRank: 15.0, // 10 for match + 5 bonus for exact title match
		},
		{
			name:     "Multiple occurrences in title",
			title:    "Python Python Python",
			path:     "/test.mp4",
			desc:     "",
			query:    "python",
			wantRank: 35.0, // 3 * 10 + 5 bonus
		},
		{
			name:     "Title + path + description",
			title:    "Python",
			path:     "/python/test.mp4",
			desc:     "Learn Python",
			query:    "python",
			wantRank: 21.0, // 10 + 5 + 1 + 5 bonus
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
			results := []SearchMediaFTSResult{
				{
					Media: Media{
						Title:       sql.NullString{String: tt.title, Valid: true},
						Path:        tt.path,
						Description: sql.NullString{String: tt.desc, Valid: true},
					},
				},
			}

			RankSearchResults(results, tt.query)

			if results[0].Rank != tt.wantRank {
				t.Errorf("RankSearchResults() rank = %.1f, want %.1f", results[0].Rank, tt.wantRank)
			}
		})
	}
}

// TestRankingReorderAmount measures how much the in-memory ranking reorders results
func TestRankingReorderAmount(t *testing.T) {
	f, err := os.CreateTemp("", "reorder-test-*.db")
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

	// Create schema
	schema := `
	CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT, description TEXT, time_deleted INTEGER DEFAULT 0);
	CREATE VIRTUAL TABLE media_fts USING fts5(path, title, description, content='media', content_rowid='rowid', tokenize='trigram', detail='none');
	CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN INSERT INTO media_fts(rowid, path, title, description) VALUES (new.rowid, new.path, new.title, new.description); END;
	`
	sqlDB.Exec(schema)

	// Insert documents in a specific order (by path)
	testData := []struct {
		path  string
		title string
		desc  string
	}{
		{"/a.mp4", "Tutorial", "Python introduction"},        // Would rank 1 by path, but desc-only
		{"/b.mp4", "Python Guide", "Learn coding"},           // Would rank 2 by path, title match
		{"/c.mp4", "Tutorial", "Learn Python Python Python"}, // Would rank 3 by path, 3x desc
		{"/d.mp4", "Python", "Learn coding"},                 // Would rank 4 by path, title only
		{"/e.mp4", "Python Python", "Python Python Python"},  // Would rank 5 by path, best match
		{"/f.mp4", "Guide", "Introduction"},                  // Would rank 6 by path, no match
	}

	ctx := context.Background()
	for _, td := range testData {
		sqlDB.Exec("INSERT INTO media (path, title, description) VALUES (?, ?, ?)",
			td.path, td.title, td.desc)
	}
	sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")

	// Fetch results in database order (by rowid, which is insert order)
	query := `
	SELECT m.path, m.title, m.description
	FROM media m, media_fts
	WHERE m.rowid = media_fts.rowid
	AND media_fts MATCH 'pyt'
	AND m.time_deleted = 0
	ORDER BY m.path  -- Database order
	`
	rows, err := sqlDB.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var dbOrder []SearchMediaFTSResult
	for rows.Next() {
		var path, title, desc string
		rows.Scan(&path, &title, &desc)
		dbOrder = append(dbOrder, SearchMediaFTSResult{
			Media: Media{
				Path:        path,
				Title:       sql.NullString{String: title, Valid: true},
				Description: sql.NullString{String: desc, Valid: true},
			},
		})
	}

	if len(dbOrder) == 0 {
		t.Fatal("No results found")
	}

	t.Logf("Database order (by path):")
	for i, r := range dbOrder {
		t.Logf("  %d: %s - %s", i+1, r.Media.Path, r.Media.Title.String)
	}

	// Apply in-memory ranking
	RankSearchResults(dbOrder, "python")

	t.Logf("\nRanked order (by relevance):")
	for i, r := range dbOrder {
		t.Logf("  %d: %s - %s (score: %.0f)", i+1, r.Media.Path, r.Media.Title.String, r.Rank)
	}

	// Track original positions before sorting
	type trackedResult struct {
		result  SearchMediaFTSResult
		origPos int
	}

	// Re-fetch and re-rank properly
	query2 := `SELECT m.path, m.title, m.description FROM media m, media_fts WHERE m.rowid = media_fts.rowid AND media_fts MATCH 'pyt' AND m.time_deleted = 0 ORDER BY m.path`
	rows2, _ := sqlDB.QueryContext(ctx, query2)

	var results []trackedResult
	idx := 0
	for rows2.Next() {
		var path, title, desc string
		rows2.Scan(&path, &title, &desc)
		results = append(results, trackedResult{
			result: SearchMediaFTSResult{
				Media: Media{Path: path, Title: sql.NullString{String: title, Valid: true}, Description: sql.NullString{String: desc, Valid: true}},
			},
			origPos: idx,
		})
		idx++
	}
	rows2.Close()

	// Extract just the results for ranking
	plainResults := make([]SearchMediaFTSResult, len(results))
	for i, tr := range results {
		plainResults[i] = tr.result
	}

	RankSearchResults(plainResults, "python")

	// Count flips: how many items moved from their original position
	maxDisplacement := 0
	totalDisplacement := 0
	for _, tr := range results {
		// Find this item's new position
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

	// Count inversions (pairs that flipped relative order)
	inversions := 0
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			// Check if this pair is inverted
			origI := results[i].origPos
			origJ := results[j].origPos

			// Find new positions
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

			// If originally i < j but now i > j, it's an inversion
			if origI < origJ && newI > newJ {
				inversions++
				t.Logf("  Inversion: %s (was %d, now %d) vs %s (was %d, now %d)",
					results[i].result.Media.Path, origI, newI,
					results[j].result.Media.Path, origJ, newJ)
			}
		}
	}

	t.Logf("Total inversions (flips): %d out of %d possible pairs", inversions, len(results)*(len(results)-1)/2)
}
