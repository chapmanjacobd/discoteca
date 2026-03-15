//go:build bleve

package bleve

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

// ComparisonBenchmarkConfig holds configuration for the benchmark
type ComparisonBenchmarkConfig struct {
	MediaCount   int
	CaptionCount int
}

// Generate data for benchmarks
func generateComparisonData(mediaCount, captionCount int) ([]*MediaDocument, []*CaptionDocument) {
	media := make([]*MediaDocument, mediaCount)
	captions := make([]*CaptionDocument, captionCount)

	types := []string{"video", "audio", "image", "text"}
	genres := []string{"Action", "Comedy", "Drama", "Music", "News", "Sci-Fi", "Horror", "Documentary"}

	// Pre-generate some long text for descriptions to simulate real content
	lorem := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

	for i := range mediaCount {
		mediaType := types[i%len(types)]
		genre := genres[i%len(genres)]
		path := fmt.Sprintf("/mnt/media/%s/file_%d.%s", mediaType, i, getExt(mediaType))

		media[i] = &MediaDocument{
			ID:             path,
			Path:           path,
			PathTokenized:  fmt.Sprintf("mnt media %s file_%d %s", mediaType, i, getExt(mediaType)),
			Title:          fmt.Sprintf("Title %d - %s Movie", i, genre),
			Description:    fmt.Sprintf("Description for media %d. %s", i, lorem),
			Type:           mediaType,
			Size:           int64(rand.Intn(1000000000) + 1000),
			Duration:       int64(rand.Intn(7200) + 60),
			TimeCreated:    time.Now().Unix(),
			TimeModified:   time.Now().Unix(),
			TimeDownloaded: time.Now().Unix(),
			PlayCount:      int64(rand.Intn(100)),
			Genre:          genre,
			Score:          rand.Float64() * 5.0,
		}
	}

	for i := range captionCount {
		mediaIdx := i % mediaCount
		captions[i] = &CaptionDocument{
			MediaPath: media[mediaIdx].Path,
			Time:      float64(i * 10),
			Text:      fmt.Sprintf("Caption text number %d containing search terms like apple banana cherry date elderberry fig grape", i),
		}
	}

	return media, captions
}

func getExt(t string) string {
	switch t {
	case "video":
		return "mp4"
	case "audio":
		return "mp3"
	case "image":
		return "jpg"
	default:
		return "txt"
	}
}

// Setup SQLite for comparison
func setupSQLiteComparison(b *testing.B, media []*MediaDocument, captions []*CaptionDocument) (*sql.DB, *db.Queries) {
	// Create local temp dir in repo root
	os.MkdirAll(".tmp", 0o755)
	tmpDir, err := os.MkdirTemp(".tmp", "sqlite_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		b.Fatalf("Failed to connect to SQLite: %v", err)
	}

	if err := testutils.InitTestDB(b, sqlDB); err != nil {
		b.Fatalf("Failed to init DB schema: %v", err)
	}

	queries := db.New(sqlDB)
	ctx := context.Background()

	// Batch insert media
	tx, err := sqlDB.Begin()
	if err != nil {
		b.Fatalf("Failed to begin transaction: %v", err)
	}
	qTx := queries.WithTx(tx)

	for _, m := range media {
		err := qTx.UpsertMedia(ctx, db.UpsertMediaParams{
			Path:           m.Path,
			PathTokenized:  sql.NullString{String: m.PathTokenized, Valid: true},
			Title:          sql.NullString{String: m.Title, Valid: true},
			Description:    sql.NullString{String: m.Description, Valid: true},
			Type:           sql.NullString{String: m.Type, Valid: true},
			Size:           sql.NullInt64{Int64: m.Size, Valid: true},
			Duration:       sql.NullInt64{Int64: m.Duration, Valid: true},
			TimeCreated:    sql.NullInt64{Int64: m.TimeCreated, Valid: true},
			TimeModified:   sql.NullInt64{Int64: m.TimeModified, Valid: true},
			TimeDownloaded: sql.NullInt64{Int64: m.TimeDownloaded, Valid: true},
			Genre:          sql.NullString{String: m.Genre, Valid: true},
			Score:          sql.NullFloat64{Float64: m.Score, Valid: true},
		})
		if err != nil {
			b.Fatalf("Failed to insert media: %v", err)
		}
	}

	// Insert captions
	for _, c := range captions {
		err := qTx.InsertCaption(ctx, db.InsertCaptionParams{
			MediaPath: c.MediaPath,
			Time:      sql.NullFloat64{Float64: c.Time, Valid: true},
			Text:      sql.NullString{String: c.Text, Valid: true},
		})
		if err != nil {
			b.Fatalf("Failed to insert caption: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		b.Fatalf("Failed to commit transaction: %v", err)
	}

	// For the benchmark, ensure FTS tables are fully populated
	// Especially since captions_ai trigger now joins with media
	var mCount, cCount int
	sqlDB.QueryRow("SELECT COUNT(*) FROM media").Scan(&mCount)
	sqlDB.QueryRow("SELECT COUNT(*) FROM captions").Scan(&cCount)

	var mPath, cPath string
	sqlDB.QueryRow("SELECT path FROM media LIMIT 1").Scan(&mPath)
	sqlDB.QueryRow("SELECT media_path FROM captions LIMIT 1").Scan(&cPath)
	fmt.Printf("DEBUG SETUP: media count=%d (%s), captions count=%d (%s)\n", mCount, mPath, cCount, cPath)

	_, err = sqlDB.Exec("INSERT INTO media_fts(rowid, path, path_tokenized, title, description, time_deleted) SELECT rowid, path, path_tokenized, title, description, time_deleted FROM media")
	if err != nil {
		fmt.Printf("DEBUG SETUP: Failed to populate media_fts: %v\n", err)
		b.Fatalf("Failed to populate media_fts: %v. Media count in DB: %d", err, mCount)
	}
	_, err = sqlDB.Exec("INSERT INTO captions_fts(rowid, media_path, text) SELECT rowid, media_path, text FROM captions")
	if err != nil {
		fmt.Printf("DEBUG SETUP: Failed to populate captions_fts: %v\n", err)
		b.Fatalf("Failed to populate captions_fts: %v. Media count: %d (%s), Captions count: %d (%s)", err, mCount, mPath, cCount, cPath)
	}

	var ftsCCount int
	sqlDB.QueryRow("SELECT COUNT(*) FROM captions_fts").Scan(&ftsCCount)
	fmt.Printf("DEBUG SETUP: captions_fts count after populate=%d\n", ftsCCount)

	return sqlDB, queries
}

// Setup Bleve for comparison
func setupBleveComparison(b *testing.B, media []*MediaDocument, captions []*CaptionDocument) (string, func()) {
	// Create local temp dir in repo root
	os.MkdirAll(".tmp", 0o755)
	tmpDir, err := os.MkdirTemp(".tmp", "bleve_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	// InitIndex expects a DB path and creates a sibling .bleve directory
	err = InitIndex(dbPath)
	if err != nil {
		b.Fatalf("Failed to init Bleve index: %v", err)
	}

	// Batch index media
	if err = BatchIndexDocuments(media, 1000); err != nil {
		b.Fatalf("Failed to batch index media: %v", err)
	}

	// Batch index captions
	// We need to implement manual batching for captions as there's no helper
	batchSize := 1000
	totalCaptions := len(captions)
	idx := GetIndex()

	for i := 0; i < totalCaptions; i += batchSize {
		batch := idx.NewBatch()
		end := min(i+batchSize, totalCaptions)

		for j := i; j < end; j++ {
			c := captions[j]
			docID := fmt.Sprintf("%s:%.3f", c.MediaPath, c.Time)
			batch.Index(docID, c)
		}

		if err = idx.Batch(batch); err != nil {
			b.Fatalf("Failed to batch index captions: %v", err)
		}
	}

	return dbPath, func() {
		CloseIndex()
		os.RemoveAll(tmpDir)
	}
}

func BenchmarkComparison(b *testing.B) {
	configs := []ComparisonBenchmarkConfig{
		{MediaCount: 80000, CaptionCount: 160000},
	}

	for _, config := range configs {
		name := fmt.Sprintf("M%d_C%d", config.MediaCount, config.CaptionCount)
		b.Run(name, func(b *testing.B) {
			media, captions := generateComparisonData(config.MediaCount, config.CaptionCount)

			// --- Setup Environments ---
			// We set them up ONCE per configuration to save time,
			// but for strict benchmarking we might want to include setup time or use ResetTimer.
			// However, since we want to benchmark READ operations, we set up once.

			// SQLite Setup
			sqliteDB, sqliteQueries := setupSQLiteComparison(b, media, captions)
			defer sqliteDB.Close()

			// Verify SQLite Data
			var mediaCount int
			sqliteDB.QueryRow("SELECT COUNT(*) FROM media").Scan(&mediaCount)
			if mediaCount != config.MediaCount {
				b.Fatalf("SQLite media count mismatch: expected %d, got %d", config.MediaCount, mediaCount)
			}

			var captionCount int
			sqliteDB.QueryRow("SELECT COUNT(*) FROM captions").Scan(&captionCount)
			if captionCount != config.CaptionCount {
				b.Fatalf("SQLite captions count mismatch: expected %d, got %d", config.CaptionCount, captionCount)
			}

			var ftsCount int
			sqliteDB.QueryRow("SELECT COUNT(*) FROM captions_fts").Scan(&ftsCount)
			if ftsCount != config.CaptionCount {
				b.Fatalf("SQLite captions_fts count mismatch: expected %d, got %d", config.CaptionCount, ftsCount)
			}

			// Bleve Setup
			_, bleveCleanup := setupBleveComparison(b, media, captions)
			defer bleveCleanup()

			// --- SEARCH BENCHMARKS ---

			// 1. Full Text Search (Common Term)
			term := "apple"           // Present in captions
			pathTerm := "media"       // Present in path_tokenized (/mnt/media/...)
			descTerm := "Description" // Present in description

			b.Run("Search_Path_FTS_SQLite", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					res, err := sqliteQueries.SearchMediaFTS(context.Background(), db.SearchMediaFTSParams{
						Query: pathTerm,
						Limit: 1000,
					})
					if err != nil {
						b.Fatal(err)
					}
					if i == 0 {
						b.ReportMetric(float64(len(res)), "results")
					}
					if len(res) == 0 && i == 0 {
						b.Fatal("Search_Path_FTS_SQLite returned 0 results")
					}
				}
			})

			b.Run("Search_Path_FTS_Bleve", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					ids, total, err := SearchPath(pathTerm, 1000)
					if err != nil {
						b.Fatal(err)
					}
					if i == 0 {
						b.ReportMetric(float64(len(ids)), "results")
						b.ReportMetric(float64(total), "total_hits")
					}
					if len(ids) == 0 && i == 0 {
						b.Fatal("Search_Path_FTS_Bleve returned 0 results")
					}
				}
			})

			b.Run("Search_Desc_FTS_SQLite", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					res, err := sqliteQueries.SearchMediaFTS(context.Background(), db.SearchMediaFTSParams{
						Query: descTerm,
						Limit: 1000,
					})
					if err != nil {
						b.Fatal(err)
					}
					if i == 0 {
						b.ReportMetric(float64(len(res)), "results")
					}
					if len(res) == 0 && i == 0 {
						b.Fatal("Search_Desc_FTS_SQLite returned 0 results")
					}
				}
			})

			b.Run("Search_Desc_FTS_Bleve", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					ids, total, err := Search(descTerm, 1000)
					if err != nil {
						b.Fatal(err)
					}
					if i == 0 {
						b.ReportMetric(float64(len(ids)), "results")
						b.ReportMetric(float64(total), "total_hits")
					}
					if len(ids) == 0 && i == 0 {
						b.Fatal("Search_Desc_FTS_Bleve returned 0 results")
					}
				}
			})

			// 2. Caption Search
			b.Run("Search_Captions_SQLite", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					// SQLite FTS on Captions using helper that handles tokenization
					res, err := sqliteQueries.SearchCaptions(context.Background(), db.SearchCaptionsParams{
						Query: term,
						Limit: 20,
					})
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 && i == 0 {
						b.Fatal("Search_Captions_SQLite returned 0 results")
					}
				}
			})

			b.Run("Search_Captions_Bleve", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					captions, _, err := SearchCaptions(term, 20)
					if err != nil {
						b.Fatal(err)
					}
					if len(captions) == 0 && i == 0 {
						b.Fatal("Search_Captions_Bleve returned 0 results")
					}
				}
			})

			// 3. Complex Query: Filter + Sort + Limit
			// Filter by Type='video', Sort by Size DESC, Limit 20
			b.Run("Complex_FilterSort_SQLite", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rows, err := sqliteDB.Query(`SELECT path FROM media WHERE type = 'video' AND time_deleted = 0 ORDER BY size DESC LIMIT 20`)
					if err != nil {
						b.Fatal(err)
					}
					count := 0
					for rows.Next() {
						count++
					}
					rows.Close()
					if count == 0 && i == 0 {
						b.Fatal("Complex_FilterSort_SQLite returned 0 results")
					}
				}
			})

			b.Run("Complex_FilterSort_Bleve", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					// Construct Bleve query
					// Type filter
					q := bleve.NewMatchQuery("video")
					q.SetField("type")

					req := bleve.NewSearchRequest(q)
					req.Size = 20
					req.Sort = search.ParseSortOrderStrings([]string{"-size"})

					idx := GetIndex()
					res, err := idx.Search(req)
					if err != nil {
						b.Fatal(err)
					}
					if res.Total == 0 && i == 0 {
						b.Fatal("Complex_FilterSort_Bleve returned 0 results")
					}
				}
			})

			// 6. Stats Aggregation (e.g. Count by Type)
			b.Run("Stats_Agg_SQLite", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					rows, err := sqliteDB.Query(`SELECT type, COUNT(*) FROM media GROUP BY type`)
					if err != nil {
						b.Fatal(err)
					}
					count := 0
					for rows.Next() {
						count++
					}
					rows.Close()
					if count == 0 && i == 0 {
						b.Fatal("Stats_Agg_SQLite returned 0 results")
					}
				}
			})

			b.Run("Stats_Agg_Bleve", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					counts, err := GetTermFacetCounts("type", 10)
					if err != nil {
						b.Fatal(err)
					}
					if len(counts) == 0 && i == 0 {
						b.Fatal("Stats_Agg_Bleve returned 0 results")
					}
				}
			})

			// 7. Group By Parent (Disk Usage Mode)
			b.Run("Group_By_Parent_SQLite", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					// Simulate aggregating by parent directory
					// Extract parent using string manipulation:
					// length(path) - length(replace(path, '/', '')) gives count of slashes
					// We want to group by everything before the last slash
					// Note: This logic is approximate for benchmark but forces string ops
					rows, err := sqliteDB.Query(`
						SELECT 
							substr(path, 1, length(path) - length(replace(path, '/', '')) + 10) as parent,
							COUNT(*),
							SUM(size)
						FROM media 
						WHERE path LIKE '/mnt/media/video/%'
						GROUP BY parent
						LIMIT 50
					`)
					if err != nil {
						b.Fatal(err)
					}
					count := 0
					for rows.Next() {
						count++
					}
					rows.Close()
					// We might get 0 results if the string logic is weird or empty, but let's allow it for perf test
					// actually we should expect results for video path
					if count == 0 && i == 0 {
						// debug print
						var p string
						sqliteDB.QueryRow("SELECT path FROM media WHERE path LIKE '/mnt/media/video/%' LIMIT 1").Scan(&p)
						b.Fatalf("Group_By_Parent_SQLite returned 0 results. Sample path: %s", p)
					}
				}
			})

			b.Run("Group_By_Parent_Bleve", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					// Bleve disk usage with prefix filter
					// Use a large limit to aggregate all matching docs, making it fair compared to SQLite
					stats, err := DiskUsageByDirectory("/mnt/media/video/", 1000000)
					if err != nil {
						b.Fatal(err)
					}
					if len(stats) == 0 && i == 0 {
						b.Fatal("Group_By_Parent_Bleve returned 0 results")
					}
				}
			})
		})
	}
}
