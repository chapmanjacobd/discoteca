//go:build bleve

package bleve

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

// Sample data for generating realistic test data
var (
	sampleTitles = []string{
		"Sample Video Title", "Amazing Movie", "Great Tutorial", "Epic Adventure",
		"Beautiful Documentary", "Funny Comedy", "Action Packed", "Drama Series",
		"Science Fiction", "Mystery Thriller", "Romance Story", "Horror Film",
		"Animation Classic", "Fantasy World", "Historical Epic", "Musical Performance",
		"Sports Highlights", "News Broadcast", "Educational Content", "Gaming Stream",
	}

	sampleDescriptions = []string{
		"A wonderful video about various topics",
		"An amazing collection of content",
		"Educational material for learning",
		"Entertainment for the whole family",
		"Professional quality production",
		"User generated content",
		"High definition video recording",
		"Classic movie from the archives",
		"Modern digital media",
		"Vintage film restoration",
	}

	samplePaths = []string{
		"/home/user/videos/movies",
		"/home/user/videos/shows",
		"/home/user/videos/tutorials",
		"/home/user/videos/music",
		"/media/storage/films",
		"/media/storage/documentaries",
		"/data/media/entertainment",
		"/data/media/educational",
		"/srv/media/shared",
		"/mnt/nas/media",
	}

	sampleExtensions = []string{".mp4", ".mkv", ".avi", ".mov", ".webm", ".mp3", ".flac", ".m4a"}
	sampleTypes      = []string{"video", "audio", "movie", "show", "music"}
)

// generateRandomMedia generates a random media item
func generateRandomMedia(i int) (db.UpsertMediaParams, *MediaDocument) {
	rand.Seed(time.Now().UnixNano() + int64(i))

	ext := sampleExtensions[rand.Intn(len(sampleExtensions))]
	pathDir := samplePaths[rand.Intn(len(samplePaths))]
	path := filepath.Join(pathDir, fmt.Sprintf("media_%d%s", i, ext))

	title := fmt.Sprintf("%s %d", sampleTitles[rand.Intn(len(sampleTitles))], i)
	description := sampleDescriptions[rand.Intn(len(sampleDescriptions))]
	mediaType := sampleTypes[rand.Intn(len(sampleTypes))]

	size := int64(rand.Intn(10_000_000_000)) // 0-10GB
	duration := int64(rand.Intn(14400))      // 0-4 hours

	dbParam := db.UpsertMediaParams{
		Path:        path,
		Title:       sql.NullString{String: title, Valid: true},
		Description: sql.NullString{String: description, Valid: true},
		Type:        sql.NullString{String: mediaType, Valid: true},
		Size:        sql.NullInt64{Int64: size, Valid: true},
		Duration:    sql.NullInt64{Int64: duration, Valid: true},
	}

	bleveDoc := &MediaDocument{
		ID:            path,
		Path:          path,
		PathTokenized: pathDir,
		Title:         title,
		Description:   description,
		Type:          mediaType,
		Size:          size,
		Duration:      duration,
	}

	return dbParam, bleveDoc
}

// setupLargeDataset creates a database with N rows and populates both FTS5 and Bleve
func setupLargeDataset(b *testing.B, count int) (*sql.DB, string) {
	b.Helper()

	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	if err := testutils.InitTestDB(b, sqlDB); err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}

	queries := db.New(sqlDB)
	ctx := context.Background()

	b.Logf("Inserting %d rows into database...", count)
	insertStart := time.Now()

	// Batch insert for performance
	batchSize := 1000
	for i := 0; i < count; i += batchSize {
		batchEnd := min(i+batchSize, count)
		for j := i; j < batchEnd; j++ {
			dbParam, _ := generateRandomMedia(j)
			err := queries.UpsertMedia(ctx, dbParam)
			if err != nil {
				b.Fatalf("Insert failed: %v", err)
			}
		}
		if (i+batchSize)%10000 == 0 {
			b.Logf("  Inserted %d/%d rows", i+batchSize, count)
		}
	}

	insertDuration := time.Since(insertStart)
	b.Logf("Insert completed in %v (%.2f rows/sec)", insertDuration, float64(count)/insertDuration.Seconds())

	// Initialize Bleve index
	b.Logf("Initializing Bleve index...")
	bleveStart := time.Now()
	if err := InitIndex(dbPath); err != nil {
		b.Fatalf("InitIndex failed: %v", err)
	}

	b.Logf("Indexing %d documents into Bleve...", count)
	for i := 0; i < count; i++ {
		_, bleveDoc := generateRandomMedia(i)
		if err := IndexDocument(bleveDoc); err != nil {
			b.Fatalf("IndexDocument failed: %v", err)
		}
		if (i+1)%10000 == 0 {
			b.Logf("  Indexed %d/%d documents", i+1, count)
		}
	}

	bleveDuration := time.Since(bleveStart)
	b.Logf("Bleve indexing completed in %v (%.2f docs/sec)", bleveDuration, float64(count)/bleveDuration.Seconds())

	countCheck, err := Count()
	if err != nil {
		b.Fatalf("Count failed: %v", err)
	}
	b.Logf("Bleve index contains %d documents", countCheck)

	return sqlDB, dbPath
}

// BenchmarkFTS5vsBleve_200k performs comprehensive benchmark with 200K rows
func BenchmarkFTS5vsBleve_200k(b *testing.B) {
	const rowCount = 200000

	sqlDB, dbPath := setupLargeDataset(b, rowCount)
	defer sqlDB.Close()
	defer CloseIndex()

	queries := db.New(sqlDB)
	ctx := context.Background()

	// Test queries - using trigram-compatible terms for FTS5
	// Note: FTS5 with detail=none requires 3-char terms, no phrases
	testQueries := []struct {
		name  string
		query string
		limit int
	}{
		{"SingleTerm_Common", "vid", 100}, // Matches "video" type
		{"SingleTerm_Rare", "edu", 100},   // Matches "educational" in description
		{"MultiTerm", "sam vid", 100},     // Matches "sample" + "video"
		{"TitleSearch", "ama", 100},       // Matches "amazing" in title
		{"TypeSearch", "mov", 100},        // Matches "movie" type
		{"SampleTitle", "sam", 100},       // Matches "sample" in title/desc
	}

	b.Logf("\n=== Starting FTS5 vs Bleve Benchmark (%d rows) ===", rowCount)
	b.Logf("Test queries: %d", len(testQueries))

	// Benchmark each query type
	for _, tq := range testQueries {
		b.Run(fmt.Sprintf("FTS5_%s", tq.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				results, err := queries.SearchMediaFTS(ctx, db.SearchMediaFTSParams{
					Query: tq.query,
					Limit: int64(tq.limit),
				})
				if err != nil {
					b.Fatalf("FTS5 search failed for query '%s': %v", tq.query, err)
				}
				if len(results) == 0 {
					b.Logf("  Warning: FTS5 returned 0 results for query '%s'", tq.query)
				}
			}
		})

		b.Run(fmt.Sprintf("Bleve_%s", tq.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ids, total, err := Search(tq.query, tq.limit)
				if err != nil {
					b.Fatalf("Bleve search failed for query '%s': %v", tq.query, err)
				}
				if len(ids) == 0 {
					b.Logf("  Warning: Bleve returned 0 results for query '%s' (total: %d)", tq.query, total)
				}
				_ = ids // avoid unused variable
			}
		})
	}

	// Bulk insert benchmark
	b.Run("BulkInsert_FTS5", func(b *testing.B) {
		tmpDir := b.TempDir()
		fts5Path := filepath.Join(tmpDir, "fts5.db")
		fts5DB, err := sql.Open("sqlite3", fts5Path)
		if err != nil {
			b.Fatalf("Failed to open FTS5 database: %v", err)
		}
		defer fts5DB.Close()

		if err := testutils.InitTestDB(b, fts5DB); err != nil {
			b.Fatalf("Failed to initialize FTS5 database: %v", err)
		}

		fts5Queries := db.New(fts5DB)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			dbParam, _ := generateRandomMedia(i)
			if err := fts5Queries.UpsertMedia(ctx, dbParam); err != nil {
				b.Fatalf("FTS5 insert failed: %v", err)
			}
		}
	})

	b.Run("BulkInsert_Bleve", func(b *testing.B) {
		tmpDir := b.TempDir()
		blevePath := filepath.Join(tmpDir, "bleve.db")
		if err := InitIndex(blevePath); err != nil {
			b.Fatalf("Failed to initialize Bleve: %v", err)
		}
		defer CloseIndex()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, bleveDoc := generateRandomMedia(i)
			if err := IndexDocument(bleveDoc); err != nil {
				b.Fatalf("Bleve insert failed: %v", err)
			}
		}
	})

	// Index size comparison
	b.Run("IndexSize", func(b *testing.B) {
		b.StopTimer()

		// FTS5 size (approximate via database size)
		fts5Stats, err := os.Stat(dbPath)
		if err != nil {
			b.Fatalf("Failed to get database size: %v", err)
		}
		dbSize := fts5Stats.Size()

		// Bleve size
		bleveIndexPath := strings.TrimSuffix(dbPath, ".db") + ".bleve"
		bleveSize, err := getDirectorySize(bleveIndexPath)
		if err != nil {
			b.Fatalf("Failed to get Bleve size: %v", err)
		}

		b.Logf("\n=== Index Size Comparison (%d rows) ===", rowCount)
		b.Logf("FTS5 Database: %.2f MB (%.2f bytes/row)", float64(dbSize)/1024/1024, float64(dbSize)/float64(rowCount))
		b.Logf("Bleve Index:   %.2f MB (%.2f bytes/row)", float64(bleveSize)/1024/1024, float64(bleveSize)/float64(rowCount))
		b.Logf("Size Ratio:    Bleve/FTS5 = %.2fx", float64(bleveSize)/float64(dbSize))
		b.StartTimer()
	})
}

// BenchmarkFTS5vsBleve_Smaller runs a quicker benchmark with fewer rows
func BenchmarkFTS5vsBleve_Smaller(b *testing.B) {
	const rowCount = 50000

	sqlDB, _ := setupLargeDataset(b, rowCount)
	defer sqlDB.Close()
	defer CloseIndex()

	queries := db.New(sqlDB)
	ctx := context.Background()

	testQueries := []struct {
		name  string
		query string
		limit int
	}{
		{"SingleTerm", "vid", 100},
		{"MultiTerm", "sam vid", 100},
		{"SampleTitle", "sam", 100},
	}

	for _, tq := range testQueries {
		b.Run(fmt.Sprintf("FTS5_%s", tq.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				results, err := queries.SearchMediaFTS(ctx, db.SearchMediaFTSParams{
					Query: tq.query,
					Limit: int64(tq.limit),
				})
				if err != nil {
					b.Fatalf("FTS5 search failed: %v", err)
				}
				_ = results
			}
		})

		b.Run(fmt.Sprintf("Bleve_%s", tq.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ids, _, err := Search(tq.query, tq.limit)
				if err != nil {
					b.Fatalf("Bleve search failed: %v", err)
				}
				_ = ids
			}
		})
	}
}

// getDirectorySize calculates the total size of a directory
func getDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
