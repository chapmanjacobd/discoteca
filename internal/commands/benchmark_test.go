package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a test database with sample data
func setupTestDB(b *testing.B, count int) (*sql.DB, string) {
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

	// Insert sample data
	queries := db.New(sqlDB)
	ctx := context.Background()
	for i := range count {
		err := queries.UpsertMedia(ctx, db.UpsertMediaParams{
			Path:     fmt.Sprintf("/media/video_%d.mp4", i),
			Title:    sql.NullString{String: fmt.Sprintf("Sample Video Title %d", i), Valid: true},
			Type:     sql.NullString{String: "video", Valid: true},
			Size:     sql.NullInt64{Int64: int64(1000000 * (i % 100)), Valid: true},
			Duration: sql.NullInt64{Int64: int64(i % 3600), Valid: true},
		})
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}

	return sqlDB, dbPath
}

// BenchmarkSearch queries the database with various search patterns
func BenchmarkSearch(b *testing.B) {
	sqlDB, _ := setupTestDB(b, 100)
	defer sqlDB.Close()

	queries := db.New(sqlDB)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queries.SearchMediaFTS(ctx, db.SearchMediaFTSParams{
			Query: "video",
			Limit: 10,
		})
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// BenchmarkAddMedia measures performance of adding media to database
func BenchmarkAddMedia(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()

	if err := testutils.InitTestDB(b, sqlDB); err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}

	queries := db.New(sqlDB)
	ctx := context.Background()

	// Create test media files
	mediaDir := filepath.Join(tmpDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		b.Fatalf("Failed to create media directory: %v", err)
	}

	// Create dummy media files
	for i := range 10 {
		path := filepath.Join(mediaDir, fmt.Sprintf("video_%d.mp4", i))
		if err := os.WriteFile(path, []byte("dummy content"), 0o644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate adding media
		err := queries.UpsertMedia(ctx, db.UpsertMediaParams{
			Path:  filepath.Join(mediaDir, fmt.Sprintf("video_%d.mp4", i%10)),
			Title: sql.NullString{String: fmt.Sprintf("Video %d", i%10), Valid: true},
			Type:  sql.NullString{String: "video", Valid: true},
		})
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

// BenchmarkFTSSearch measures full-text search performance
func BenchmarkFTSSearch(b *testing.B) {
	sqlDB, _ := setupTestDB(b, 100)
	defer sqlDB.Close()

	queries := db.New(sqlDB)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queries.SearchMediaFTS(ctx, db.SearchMediaFTSParams{
			Query: "Sample",
			Limit: 10,
		})
		if err != nil {
			b.Fatalf("FTS search failed: %v", err)
		}
	}
}

// BenchmarkAggregateStats measures aggregation query performance
func BenchmarkAggregateStats(b *testing.B) {
	sqlDB, _ := setupTestDB(b, 1000)
	defer sqlDB.Close()

	queries := db.New(sqlDB)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queries.GetStats(ctx)
		if err != nil {
			b.Fatalf("GetStats failed: %v", err)
		}
	}
}

// BenchmarkHistoryQueries measures history-related query performance
func BenchmarkHistoryQueries(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()

	if err := testutils.InitTestDB(b, sqlDB); err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}

	queries := db.New(sqlDB)
	ctx := context.Background()

	// Insert sample data with history
	for i := range 500 {
		err := queries.UpsertMedia(ctx, db.UpsertMediaParams{
			Path:  fmt.Sprintf("/media/video_%d.mp4", i),
			Title: sql.NullString{String: fmt.Sprintf("Video %d", i), Valid: true},
			Type:  sql.NullString{String: "video", Valid: true},
		})
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}

		// Add play history
		err = queries.InsertHistory(ctx, db.InsertHistoryParams{
			MediaPath: fmt.Sprintf("/media/video_%d.mp4", i),
			Playhead:  sql.NullInt64{Int64: int64(i % 1000), Valid: true},
		})
		if err != nil {
			b.Fatalf("InsertPlayHistory failed: %v", err)
		}
	}

	b.Run("GetUnfinished", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := queries.GetUnfinishedMedia(ctx, 10)
			if err != nil {
				b.Fatalf("GetUnfinishedMedia failed: %v", err)
			}
		}
	})

	b.Run("GetUnwatched", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := queries.GetUnwatchedMedia(ctx, 10)
			if err != nil {
				b.Fatalf("GetUnwatchedMedia failed: %v", err)
			}
		}
	})

	b.Run("GetWatched", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := queries.GetWatchedMedia(ctx, 10)
			if err != nil {
				b.Fatalf("GetWatchedMedia failed: %v", err)
			}
		}
	})
}
