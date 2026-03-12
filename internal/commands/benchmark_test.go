package commands

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
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
	for i := 0; i < count; i++ {
		_, err := db.InsertMedia(sqlDB, db.Media{
			Path:     fmt.Sprintf("/media/video_%d.mp4", i),
			Title:    fmt.Sprintf("Sample Video Title %d", i),
			Type:     "video",
			Size:     int64(1000000 * (i%100)),
			Duration: float64(i % 3600),
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

	queries := []string{
		"test",
		"video",
		"mp4",
		"2024",
		"sample",
	}

	for _, query := range queries {
		b.Run(fmt.Sprintf("query_%s", query), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := db.SearchMedia(sqlDB, query, nil)
				if err != nil {
					b.Fatalf("Search failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkAddMedia measures performance of adding media to database
func BenchmarkAddMedia(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	testDB, err := testutils.CreateTestDatabase(dbPath)
	if err != nil {
		b.Fatalf("Failed to create test database: %v", err)
	}
	defer os.Remove(dbPath)

	// Create test media files
	mediaDir := filepath.Join(tmpDir, "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		b.Fatalf("Failed to create media directory: %v", err)
	}

	// Create dummy media files
	for i := 0; i < 10; i++ {
		path := filepath.Join(mediaDir, fmt.Sprintf("video_%d.mp4", i))
		if err := os.WriteFile(path, []byte("dummy content"), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate adding media
		_, err := db.InsertMedia(testDB.DB, db.Media{
			Path:  filepath.Join(mediaDir, fmt.Sprintf("video_%d.mp4", i%10)),
			Title: fmt.Sprintf("Video %d", i%10),
			Type:  "video",
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.FTSSearch(sqlDB, "Sample")
		if err != nil {
			b.Fatalf("FTS search failed: %v", err)
		}
	}
}

// BenchmarkAggregateStats measures aggregation query performance
func BenchmarkAggregateStats(b *testing.B) {
	sqlDB, _ := setupTestDB(b, 1000)
	defer sqlDB.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetStats(sqlDB)
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

	// Insert sample data with history
	for i := 0; i < 500; i++ {
		mediaID, err := db.InsertMedia(sqlDB, db.Media{
			Path:  fmt.Sprintf("/media/video_%d.mp4", i),
			Title: fmt.Sprintf("Video %d", i),
			Type:  "video",
		})
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}

		// Add play history
		_, err = db.InsertPlayHistory(sqlDB, db.PlayHistory{
			MediaID:  mediaID,
			Position: float64(i % 1000),
		})
		if err != nil {
			b.Fatalf("InsertPlayHistory failed: %v", err)
		}
	}

	b.Run("GetInProgress", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := db.GetInProgress(sqlDB, nil)
			if err != nil {
				b.Fatalf("GetInProgress failed: %v", err)
			}
		}
	})

	b.Run("GetUnplayed", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := db.GetUnplayed(sqlDB, nil)
			if err != nil {
				b.Fatalf("GetUnplayed failed: %v", err)
			}
		}
	})

	b.Run("GetCompleted", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := db.GetCompleted(sqlDB, nil)
			if err != nil {
				b.Fatalf("GetCompleted failed: %v", err)
			}
		}
	})
}

// BenchmarkMetadataExtraction measures metadata extraction performance
func BenchmarkMetadataExtraction(b *testing.B) {
	tmpDir := b.TempDir()
	mediaDir := filepath.Join(tmpDir, "media")

	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		b.Fatalf("Failed to create media directory: %v", err)
	}

	// Create dummy media files
	for i := 0; i < 10; i++ {
		path := filepath.Join(mediaDir, fmt.Sprintf("video_%d.mp4", i))
		if err := os.WriteFile(path, bytes.Repeat([]byte{0x00}, 1024), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(mediaDir, fmt.Sprintf("video_%d.mp4", i%10))
		_, err := extractMetadata(path)
		if err != nil && err != io.EOF {
			// Ignore EOF errors from dummy files
			b.Logf("Metadata extraction warning: %v", err)
		}
	}
}

// extractMetadata is a helper to extract metadata from a file
func extractMetadata(path string) (interface{}, error) {
	// This would call the actual metadata extraction logic
	// For now, just read the file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, 512)
	_, err = f.Read(buf)
	return buf, err
}
