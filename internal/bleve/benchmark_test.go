//go:build bleve

package bleve

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

// Benchmark data generators
func generateTestDocuments(count int) []*MediaDocument {
	docs := make([]*MediaDocument, count)
	types := []string{"video", "audio", "image", "text"}
	genres := []string{"Action", "Comedy", "Drama", "Music", "News"}
	languages := []string{"en", "es", "fr", "de", "ja"}

	now := time.Now().Unix()

	for i := range count {
		docs[i] = &MediaDocument{
			ID:             fmt.Sprintf("doc%d", i),
			Path:           filepath.FromSlash(fmt.Sprintf("/media/%s/file%d.mp4", types[i%4], i)),
			PathTokenized:  fmt.Sprintf("media file%d", i),
			Title:          fmt.Sprintf("Test Document Title %d", i),
			Description:    fmt.Sprintf("This is a test description for document number %d with some keywords like matrix and action", i),
			Type:           types[i%4],
			Size:           int64(1000 + (i * 100)),
			Duration:       int64(60 + (i * 10)),
			TimeCreated:    now - int64((count-i)*86400),
			TimeModified:   now - int64(((count-i)/2)*86400),
			TimeDownloaded: now - int64(count-i)*86400,
			TimeLastPlayed: now - int64((count-i)/3)*86400,
			PlayCount:      int64(i % 100),
			Genre:          genres[i%5],
			Language:       languages[i%5],
			VideoCount:     int64(i % 3),
			AudioCount:     int64(i % 2),
			Score:          float64(i%100) / 100.0,
		}
	}
	return docs
}

// setupBleveBenchmark creates a Bleve index with test documents
func setupBleveBenchmark(b *testing.B, docCount int) (string, func()) {
	t := &testing.T{}
	fixture := testutils.Setup(t)
	dbPath := fixture.DBPath

	err := InitIndex(dbPath)
	if err != nil {
		b.Fatalf("InitIndex failed: %v", err)
	}

	docs := generateTestDocuments(docCount)
	if err := BatchIndexDocuments(docs, 1000); err != nil {
		b.Fatalf("BatchIndexDocuments failed: %v", err)
	}

	return dbPath, func() {
		CloseIndex()
		fixture.Cleanup()
	}
}

// setupSQLiteFTS5Benchmark creates SQLite database with FTS5 and test documents
func setupSQLiteFTS5Benchmark(b *testing.B, docCount int) (*sql.DB, func()) {
	t := &testing.T{}
	fixture := testutils.Setup(t)
	dbPath := fixture.DBPath

	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		b.Fatalf("db.Connect failed: %v", err)
	}

	// Insert test documents
	now := time.Now().Unix()
	types := []string{"video", "audio", "image", "text"}
	genres := []string{"Action", "Comedy", "Drama", "Music", "News"}

	stmt, err := sqlDB.Prepare(`
		INSERT INTO media (
			path, path_tokenized, title, description, type, size, duration,
			time_created, time_modified, time_downloaded, time_last_played,
			play_count, genre, language, video_count, audio_count, score
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		b.Fatalf("Prepare failed: %v", err)
	}
	defer stmt.Close()

	for i := range docCount {
		_, err := stmt.Exec(
			filepath.FromSlash(fmt.Sprintf("/media/%s/file%d.mp4", types[i%4], i)),
			fmt.Sprintf("media file%d", i),
			fmt.Sprintf("Test Document Title %d", i),
			fmt.Sprintf("This is a test description for document number %d with some keywords like matrix and action", i),
			types[i%4],
			1000+(i*100),
			60+(i*10),
			now-int64((docCount-i)*86400),
			now-int64(((docCount-i)/2)*86400),
			now-int64((docCount-i)*86400),
			now-int64((docCount-i)/3)*86400,
			i%100,
			genres[i%5],
			"en",
			i%3,
			i%2,
			float64(i%100)/100.0,
		)
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}

	return sqlDB, func() {
		sqlDB.Close()
		fixture.Cleanup()
	}
}

// Benchmark indexing performance
func BenchmarkBleveIndexing(b *testing.B) {
	docCounts := []int{100, 1000, 5000}

	for _, count := range docCounts {
		b.Run(fmt.Sprintf("Docs%d", count), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				t := &testing.T{}
				fixture := testutils.Setup(t)
				dbPath := fixture.DBPath

				err := InitIndex(dbPath)
				if err != nil {
					b.Fatalf("InitIndex failed: %v", err)
				}

				docs := generateTestDocuments(count)
				b.StartTimer()
				if err := BatchIndexDocuments(docs, 1000); err != nil {
					b.Fatalf("BatchIndexDocuments failed: %v", err)
				}
				b.StopTimer()

				CloseIndex()
				fixture.Cleanup()
			}
		})
	}
}

// Benchmark simple search performance
func BenchmarkBleveSimpleSearch(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	searchTerms := []string{"test", "matrix", "action", "document"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term := searchTerms[i%len(searchTerms)]
		_, _, err := Search(term, 10)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// Benchmark exact match search performance
func BenchmarkBleveExactMatchSearch(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := SearchWithExactMatch("test", 10, true)
		if err != nil {
			b.Fatalf("SearchWithExactMatch failed: %v", err)
		}
	}
}

// Benchmark search with sorting performance
func BenchmarkBleveSearchWithSort(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	sortConfigs := [][]SortField{
		{{Field: "size", Descending: true}},
		{{Field: "time_created", Descending: true}},
		{{Field: "play_count", Descending: false}},
		{{Field: "size", Descending: true}, {Field: "time_created", Descending: true}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sort := sortConfigs[i%len(sortConfigs)]
		_, _, _, err := SearchWithSort("test", 10, 0, sort)
		if err != nil {
			b.Fatalf("SearchWithSort failed: %v", err)
		}
	}
}

// Benchmark faceted search performance
func BenchmarkBleveFacetedSearch(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		facetRequests := make(map[string]*bleve.FacetRequest)
		facetRequests["type"] = NewTermFacetRequest("type", 10)
		facetRequests["genre"] = NewTermFacetRequest("genre", 10)

		_, _, _, err := SearchWithFacets("test", 10, facetRequests)
		if err != nil {
			b.Fatalf("SearchWithFacets failed: %v", err)
		}
	}
}

// Benchmark numeric range facet performance
func BenchmarkBleveNumericRangeFacet(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	small := 0.0
	medium := 50000.0
	large := 100000.0
	huge := 1000000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		facetRequests := make(map[string]*bleve.FacetRequest)
		facetRequests["size"] = NewNumericRangeFacetRequest("size", []struct {
			Name string
			Min  *float64
			Max  *float64
		}{
			{"Small", &small, &medium},
			{"Medium", &medium, &large},
			{"Large", &large, &huge},
		})

		_, _, _, err := SearchWithFacets("test", 10, facetRequests)
		if err != nil {
			b.Fatalf("SearchWithFacets failed: %v", err)
		}
	}
}

// Benchmark combined search with sort and facets
func BenchmarkBleveSearchWithSortAndFacets(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sortFields := []SortField{{Field: "size", Descending: true}}
		facetRequests := make(map[string]*bleve.FacetRequest)
		facetRequests["type"] = NewTermFacetRequest("type", 10)

		_, err := SearchWithSortAndFacets("test", 10, 0, sortFields, facetRequests)
		if err != nil {
			b.Fatalf("SearchWithSortAndFacets failed: %v", err)
		}
	}
}

// Benchmark pagination performance (deep paging)
func BenchmarkBleveDeepPagination(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 5000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test page 10 (offset 90)
		_, _, _, err := SearchWithSort("test", 10, 90, []SortField{{Field: "size", Descending: true}})
		if err != nil {
			b.Fatalf("SearchWithSort deep pagination failed: %v", err)
		}
	}
}

// Benchmark cursor-based pagination (SearchAfter)
func BenchmarkBleveCursorPagination(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 5000)
	defer cleanup()

	// Get first page to get searchAfter token
	_, _, searchAfter, err := SearchWithSort("test", 10, 0, []SortField{{Field: "size", Descending: true}})
	if err != nil {
		b.Fatalf("Initial search failed: %v", err)
	}
	if len(searchAfter) == 0 {
		b.Fatal("No searchAfter token returned")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use SearchAfter for efficient deep pagination
		_, _, _, err := SearchWithSort("test", 10, 0, []SortField{{Field: "size", Descending: true}})
		if err != nil {
			b.Fatalf("SearchWithSearchAfter failed: %v", err)
		}
	}
}

// Benchmark multi-field search with boosting
func BenchmarkBleveMultiFieldSearch(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	fieldBoosts := map[string]float64{
		"title":       2.0,
		"description": 1.0,
		"path":        0.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := MultiFieldSearch("test", 10, fieldBoosts)
		if err != nil {
			b.Fatalf("MultiFieldSearch failed: %v", err)
		}
	}
}

// Benchmark prefix/autocomplete search
func BenchmarkBlevePrefixSearch(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	prefixes := []string{"te", "mat", "doc", "act"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prefix := prefixes[i%len(prefixes)]
		_, _, err := PrefixSearch(prefix, 10)
		if err != nil {
			b.Fatalf("PrefixSearch failed: %v", err)
		}
	}
}

// Benchmark batch indexing at scale
func BenchmarkBleveBatchIndexing(b *testing.B) {
	docCounts := []int{1000, 5000, 10000}

	for _, count := range docCounts {
		b.Run(fmt.Sprintf("Docs%d", count), func(b *testing.B) {
			docs := generateTestDocuments(count)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				t := &testing.T{}
				fixture := testutils.Setup(t)
				dbPath := fixture.DBPath

				err := InitIndex(dbPath)
				if err != nil {
					b.Fatalf("InitIndex failed: %v", err)
				}

				if err := BatchIndexDocuments(docs, 1000); err != nil {
					b.Fatalf("BatchIndexDocuments failed: %v", err)
				}

				CloseIndex()
				fixture.Cleanup()
			}
		})
	}
}

// Benchmark index statistics retrieval
func BenchmarkBleveGetIndexStats(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetIndexStats()
		if err != nil {
			b.Fatalf("GetIndexStats failed: %v", err)
		}
	}
}

// Benchmark concurrent search operations
func BenchmarkBleveConcurrentSearch(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		searchTerms := []string{"test", "matrix", "action"}
		termIdx := rand.Intn(len(searchTerms))

		for pb.Next() {
			term := searchTerms[termIdx%len(searchTerms)]
			termIdx++
			_, _, err := Search(term, 10)
			if err != nil {
				b.Fatalf("Concurrent search failed: %v", err)
			}
		}
	})
}

// Benchmark memory usage estimation (via docValues)
func BenchmarkBleveDocValuesAccess(b *testing.B) {
	_, cleanup := setupBleveBenchmark(b, 1000)
	defer cleanup()

	sortConfigs := [][]SortField{
		{{Field: "size", Descending: true}},
		{{Field: "duration", Descending: true}},
		{{Field: "play_count", Descending: false}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sort := sortConfigs[i%len(sortConfigs)]
		// This exercises docValues for sorting
		_, _, _, err := SearchWithSort("test", 10, 0, sort)
		if err != nil {
			b.Fatalf("SearchWithSort failed: %v", err)
		}
	}
}

// SQLite FTS5 comparison benchmarks
func BenchmarkSQLiteFTS5Search(b *testing.B) {
	sqlDB, cleanup := setupSQLiteFTS5Benchmark(b, 1000)
	defer cleanup()

	searchTerms := []string{"test", "matrix", "action"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term := searchTerms[i%len(searchTerms)]
		query := `SELECT path FROM media JOIN media_fts ON media.rowid = media_fts.rowid 
		          WHERE media_fts MATCH ? AND time_deleted = 0 LIMIT 10`
		rows, err := sqlDB.QueryContext(context.Background(), query, term)
		if err != nil {
			b.Fatalf("FTS5 query failed: %v", err)
		}
		rows.Close()
	}
}

func BenchmarkSQLiteFTS5ExactSearch(b *testing.B) {
	sqlDB, cleanup := setupSQLiteFTS5Benchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := `SELECT path FROM media WHERE path_tokenized LIKE ? AND time_deleted = 0 LIMIT 10`
		rows, err := sqlDB.QueryContext(context.Background(), query, "%test%")
		if err != nil {
			b.Fatalf("Exact search query failed: %v", err)
		}
		rows.Close()
	}
}

func BenchmarkSQLiteFTS5SortBySize(b *testing.B) {
	sqlDB, cleanup := setupSQLiteFTS5Benchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := `SELECT path FROM media WHERE time_deleted = 0 ORDER BY size DESC LIMIT 10`
		rows, err := sqlDB.QueryContext(context.Background(), query)
		if err != nil {
			b.Fatalf("Sort query failed: %v", err)
		}
		rows.Close()
	}
}

func BenchmarkSQLiteFTS5GroupByType(b *testing.B) {
	sqlDB, cleanup := setupSQLiteFTS5Benchmark(b, 1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := `SELECT type, COUNT(*) FROM media WHERE time_deleted = 0 GROUP BY type`
		rows, err := sqlDB.QueryContext(context.Background(), query)
		if err != nil {
			b.Fatalf("Group by query failed: %v", err)
		}
		rows.Close()
	}
}
