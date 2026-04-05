package query_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestTrigramBM25WithMoreData tests BM25 ranking with a larger dataset
func TestTrigramBM25WithMoreData(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "trigram-large-test-*.db")
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

	// Create table
	schema := `
	CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT, description TEXT);
	CREATE VIRTUAL TABLE media_fts USING fts5(path, title, description, content='media', content_rowid='rowid', tokenize='trigram', detail='none');
	CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN INSERT INTO media_fts(rowid, path, title, description) VALUES (new.rowid, new.path, new.title, new.description); END;
	`
	sqlDB.Exec(schema)

	// Create larger dataset with varying relevance
	testData := []struct {
		path  string
		title string
		desc  string
	}{
		// High relevance: "python" appears 5+ times
		{"/a", "Python Tutorial", "Python Python Python Python Python"},
		{"/b", "Learn Python", "Python programming Python course Python tutorial"},

		// Medium relevance: "python" appears 2-3 times
		{"/c", "Python Intro", "Introduction to Python programming"},
		{"/d", "Python Basics", "Learn Python basics Python"},
		{"/e", "Advanced Python", "Advanced Python topics"},

		// Low relevance: "python" appears 1 time
		{"/f", "Programming", "Covers Python language"},
		{"/g", "Tutorial Video", "Python content here"},
		{"/h", "Course", "Study Python"},

		// False positives: has "pyt" trigram but not "python"
		{"/i", "Pyrotechnics", "Fire display show"},
		{"/j", "Python History", "About the Python language created by Guido van Rossum"},

		// More varied content
		{"/k", "Python vs Go", "Comparing Python with Go programming languages"},
		{"/l", "Machine Learning", "Python for ML Python data science Python AI"},
		{"/m", "Web Development", "Python Django Flask web Python"},
		{"/n", "Data Science", "Python pandas numpy Python data Python analysis"},
		{"/o", "Automation", "Python scripting automation Python"},
	}

	for _, td := range testData {
		sqlDB.Exec("INSERT INTO media (path, title, description) VALUES (?, ?, ?)", td.path, td.title, td.desc)
	}
	sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")

	ctx := context.Background()

	// Test different query strategies
	strategies := []struct {
		name  string
		query string
	}{
		{"Single trigram (pyt)", "pyt"},
		{"Two trigrams AND (pyt AND yth)", "pyt AND yth"},
		{"Three trigrams AND", "pyt AND yth AND tho"},
		{"Two trigrams OR (pyt OR yth)", "pyt OR yth"},
		{"All trigrams OR", "pyt OR yth OR tho OR hon OR on_"},
	}

	t.Run("BM25 Ranking", func(t *testing.T) {
		for _, tc := range strategies {
			func() {
				sql := `
		SELECT m.path, m.title, media_fts.rank,
			(LENGTH(m.title) + LENGTH(m.description) - LENGTH(REPLACE(LOWER(m.title || ' ' || m.description), 'python', ''))) / 6 as python_count
		FROM media m, media_fts
		WHERE m.rowid = media_fts.rowid
		AND media_fts MATCH ?
		ORDER BY media_fts.rank DESC
		LIMIT 10
		`
				rows, err := sqlDB.QueryContext(ctx, sql, tc.query)
				if err != nil {
					t.Errorf("%s: ERROR - %v", tc.name, err)
					return
				}
				defer rows.Close()

				var results []string
				rank := 0
				for rows.Next() {
					var path, title string
					var bm25 float64
					var pythonCount int
					if err := rows.Scan(&path, &title, &bm25, &pythonCount); err != nil {
						t.Errorf("%s: Scan error: %v", tc.name, err)
						continue
					}
					rank++
					results = append(
						results,
						fmt.Sprintf("%d. %s (python#=%d, score=%.6f)", rank, path, pythonCount, bm25),
					)
				}
				if err := rows.Err(); err != nil {
					t.Errorf("%s: rows error: %v", tc.name, err)
				}

				if len(results) == 0 {
					t.Errorf("%s: no results returned", tc.name)
				}
			}()
		}
	})

	// Test correlation between term frequency and BM25 rank
	t.Run("Term Frequency vs BM25 Rank Correlation", func(t *testing.T) {
		sql := `
	SELECT m.path, media_fts.rank,
		(LENGTH(m.title) + LENGTH(m.description) - LENGTH(REPLACE(LOWER(m.title || ' ' || m.description), 'python', ''))) / 6 as python_count
	FROM media m, media_fts
	WHERE m.rowid = media_fts.rowid
	AND media_fts MATCH 'pyt'
	ORDER BY media_fts.rank DESC
	`
		rows, err := sqlDB.QueryContext(ctx, sql)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		totalDocs := 0
		correctOrder := 0
		lastCount := 999
		for rows.Next() {
			var path string
			var bm25 float64
			var pythonCount int
			if err := rows.Scan(&path, &bm25, &pythonCount); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			totalDocs++

			if pythonCount <= lastCount {
				correctOrder++
			}
			lastCount = pythonCount
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows error: %v", err)
		}

		correlation := float64(correctOrder) / float64(totalDocs) * 100
		if correlation < 70.0 {
			t.Errorf("Low correlation: %d/%d documents in term-frequency order (%.1f%%)",
				correctOrder, totalDocs, correlation)
		}
	})
}

// TestFirstTrigramOnly tests if using just the first trigram is sufficient
func TestFirstTrigramOnly(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "first-tri-test-*.db")
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

	sqlDB.Exec(`
		CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT);
		CREATE VIRTUAL TABLE media_fts USING fts5(path, title, content='media', content_rowid='rowid', tokenize='trigram', detail='none');
		CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN INSERT INTO media_fts(rowid, path, title) VALUES (new.rowid, new.path, new.title); END;
	`)

	// Documents with different trigram coverage
	docs := []struct{ path, title string }{
		{"/a", "Video Tutorial"},  // vid ide deo o_t tut utu tor
		{"/b", "Video"},           // vid ide deo
		{"/c", "Tutorial"},        // tut utu tor
		{"/d", "Vid"},             // vid (too short, may not match)
		{"/e", "Tutorial Videos"}, // tut utu tor vid ide deo os_
	}

	for _, d := range docs {
		sqlDB.Exec("INSERT INTO media (path, title) VALUES (?, ?)", d.path, d.title)
	}
	sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")

	ctx := context.Background()

	tests := []struct {
		name  string
		query string
	}{
		{"First trigram of 'video' (vid)", "vid"},
		{"First trigram of 'tutorial' (tut)", "tut"},
		{"Both first trigrams (vid OR tut)", "vid OR tut"},
		{"All 'video' trigrams", "vid AND ide AND deo"},
		{"All 'tutorial' trigrams", "tut AND utu AND tor"},
	}

	t.Run("First Trigram vs All Trigrams", func(t *testing.T) {
		for _, tc := range tests {
			func() {
				sql := `SELECT m.path, m.title, media_fts.rank FROM media m, media_fts WHERE m.rowid = media_fts.rowid AND media_fts MATCH ? ORDER BY media_fts.rank DESC`
				rows, err := sqlDB.QueryContext(ctx, sql, tc.query)
				if err != nil {
					t.Errorf("%-35s ERROR: %v", tc.name, err)
					return
				}
				defer rows.Close()

				var results []string
				for rows.Next() {
					var path, title string
					var rank float64
					if err := rows.Scan(&path, &title, &rank); err != nil {
						t.Errorf("%s: Scan error: %v", tc.name, err)
						continue
					}
					results = append(results, fmt.Sprintf("%s(%.0f)", path, rank*1000000))
				}
				if err := rows.Err(); err != nil {
					t.Errorf("%s: rows error: %v", tc.name, err)
				}

				if len(results) == 0 {
					t.Errorf("%s: no results returned", tc.name)
				}
			}()
		}
	})
}
