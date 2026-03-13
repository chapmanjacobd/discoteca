//go:build bleve

package bleve

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

// setupBenchmarkData creates test data for benchmarking
func setupBenchmarkData(b *testing.B, rowCount int) (*sql.DB, string) {
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

	b.Logf("Inserting %d rows...", rowCount)
	insertStart := time.Now()

	// Batch insert for better performance
	batchSize := 1000
	for i := 0; i < rowCount; i += batchSize {
		batchEnd := min(i+batchSize, rowCount)
		for j := i; j < batchEnd; j++ {
			path := filepath.Join("/home/user/videos", fmt.Sprintf("media_%d.mp4", j))
			title := fmt.Sprintf("Sample Video Title %d", j)
			description := "A sample video file with test content"
			mediaType := "video"

			err := queries.UpsertMedia(ctx, db.UpsertMediaParams{
				Path:        path,
				Title:       sql.NullString{String: title, Valid: true},
				Description: sql.NullString{String: description, Valid: true},
				Type:        sql.NullString{String: mediaType, Valid: true},
			})
			if err != nil {
				b.Fatalf("Insert failed: %v", err)
			}
		}
		if batchEnd%100000 == 0 {
			b.Logf("  Inserted %d/%d rows", batchEnd, rowCount)
		}
	}

	insertDuration := time.Since(insertStart)
	b.Logf("Insert completed in %v (%.2f rows/sec)", insertDuration, float64(rowCount)/insertDuration.Seconds())

	b.Logf("Indexing %d documents into Bleve...", rowCount)
	indexStart := time.Now()

	for i := 0; i < rowCount; i++ {
		path := filepath.Join("/home/user/videos", fmt.Sprintf("media_%d.mp4", i))
		title := fmt.Sprintf("Sample Video Title %d", i)
		description := "A sample video file with test content"
		mediaType := "video"

		doc := &MediaDocument{
			ID:            path,
			Path:          path,
			PathTokenized: "/home/user/videos",
			Title:         title,
			Description:   description,
			Type:          mediaType,
		}
		if i == 0 {
			if err := InitIndex(dbPath); err != nil {
				b.Fatalf("InitIndex failed: %v", err)
			}
		}
		if err := IndexDocument(doc); err != nil {
			b.Fatalf("IndexDocument failed: %v", err)
		}
		if (i+1)%100000 == 0 {
			b.Logf("  Indexed %d/%d documents", i+1, rowCount)
		}
	}

	indexDuration := time.Since(indexStart)
	b.Logf("Bleve indexing completed in %v (%.2f docs/sec)", indexDuration, float64(rowCount)/indexDuration.Seconds())

	count, _ := Count()
	b.Logf("Bleve index contains %d documents", count)

	return sqlDB, dbPath
}

func BenchmarkFTS5vsBleve_Scale(b *testing.B) {
	testSizes := []int{200, 20000, 200000, 2000000}
	testQuery := "sam"

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("Rows_%d", size), func(b *testing.B) {
			sqlDB, _ := setupBenchmarkData(b, size)
			defer sqlDB.Close()
			defer CloseIndex()

			queries := db.New(sqlDB)
			ctx := context.Background()

			// Verify both systems return results
			fts5Results, err := queries.SearchMediaFTS(ctx, db.SearchMediaFTSParams{
				Query: testQuery,
				Limit: 10,
			})
			if err != nil {
				b.Fatalf("FTS5 search failed: %v", err)
			}

			bleveIDs, _, err := Search(testQuery, 10)
			if err != nil {
				b.Fatalf("Bleve search failed: %v", err)
			}

			if len(fts5Results) == 0 || len(bleveIDs) == 0 {
				b.Fatal("One or both systems returned no results")
			}

			b.ResetTimer()

			b.Run("FTS5", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					results, err := queries.SearchMediaFTS(ctx, db.SearchMediaFTSParams{
						Query: testQuery,
						Limit: 10,
					})
					if err != nil {
						b.Fatalf("FTS5 search failed: %v", err)
					}
					_ = results
				}
			})

			b.Run("Bleve", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					ids, _, err := Search(testQuery, 10)
					if err != nil {
						b.Fatalf("Bleve search failed: %v", err)
					}
					_ = ids
				}
			})
		})
	}
}
