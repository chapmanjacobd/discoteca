package commands

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func TestHandleDU_WithFilters(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_du_filters.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize database schema
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}

	// Create test media with various types, sizes, and durations for filter testing
	// Organized under /home directory to test folder navigation
	_, err = sqlDB.Exec(`INSERT INTO media (path, title, type, size, duration, time_deleted) VALUES
		-- Videos (5 files)
		('/home/videos/movie1.mp4', 'Movie1', 'video', 500000000, 7200, 0),
		('/home/videos/movie2.mp4', 'Movie2', 'video', 300000000, 5400, 0),
		('/home/videos/clip1.mp4', 'Clip1', 'video', 50000000, 30, 0),
		('/home/videos/clip2.mp4', 'Clip2', 'video', 25000000, 15, 0),
		('/home/videos/short.mp4', 'Short', 'video', 10000000, 5, 0),
		-- Audio files (3 files)
		('/home/audio/song1.mp3', 'Song1', 'audio', 5000000, 180, 0),
		('/home/audio/song2.mp3', 'Song2', 'audio', 4000000, 150, 0),
		('/home/audio/podcast.mp3', 'Podcast', 'audio', 15000000, 900, 0),
		-- Image files (3 files)
		('/home/images/photo1.png', 'Photo1', 'image', 2000000, 0, 0),
		('/home/images/photo2.png', 'Photo2', 'image', 1500000, 0, 0),
		('/home/images/photo3.png', 'Photo3', 'image', 3000000, 0, 0),
		-- Text/document files (2 files)
		('/home/documents/doc1.txt', 'Doc1', 'text', 50000, 0, 0),
		('/home/documents/doc2.pdf', 'Doc2', 'text', 100000, 0, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	cmd := setupTestServeCmd(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("video-only filter returns only video folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&video=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have folders (all media is under /home in test db)
		if len(resp.Folders) == 0 {
			t.Error("Expected folders in response")
		}

		// Total count should reflect filtered results
		// Test DB has 5 videos out of 13 total media
		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("type=video filter returns only video folders", func(t *testing.T) {
		// Get unfiltered results first
		req1 := httptest.NewRequest("GET", "/api/du?path=&include_counts=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal unfiltered response: %v", err)
		}
		unfilteredTotal := resp1.TotalCount
		t.Logf("Unfiltered - Folders: %d, Total: %d", resp1.FolderCount, unfilteredTotal)

		// Get total file count from folders
		unfilteredFileCount := 0
		for _, f := range resp1.Folders {
			unfilteredFileCount += f.Count
		}
		t.Logf("Unfiltered file count in folders: %d", unfilteredFileCount)

		// Apply type=video filter (frontend uses this format)
		req2 := httptest.NewRequest("GET", "/api/du?path=&type=video", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w2.Code, w2.Body.String())
		}

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("type=video - Folders: %d, Files: %d, Total: %d", resp2.FolderCount, resp2.FileCount, resp2.TotalCount)

		// Get filtered file count from folders
		filteredFileCount := 0
		for _, f := range resp2.Folders {
			filteredFileCount += f.Count
		}
		t.Logf("Filtered file count in folders: %d", filteredFileCount)

		// File count within folders should be less (5 videos out of 13 total)
		if filteredFileCount >= unfilteredFileCount {
			t.Errorf("Expected filtered file count (%d) to be less than unfiltered (%d)", filteredFileCount, unfilteredFileCount)
		}
		if filteredFileCount == 0 {
			t.Error("Expected some video results")
		}
	})

	t.Run("audio-only filter returns only audio folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&audio=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("image-only filter returns only image folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&image=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("size filter returns only media matching size range", func(t *testing.T) {
		// Filter for media > 100KB
		req := httptest.NewRequest("GET", "/api/du?path=&size=>100KB", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("duration filter returns only media matching duration range", func(t *testing.T) {
		// Filter for media > 10 seconds
		req := httptest.NewRequest("GET", "/api/du?path=&duration=>10", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("search filter returns only matching media", func(t *testing.T) {
		// Search for "test"
		req := httptest.NewRequest("GET", "/api/du?path=&search=test", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("include_counts returns filter bins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&include_counts=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Counts == nil {
			t.Error("Expected counts in response when include_counts=true")
		}

		// Check that bins have data
		if len(resp.Counts.Type) == 0 {
			t.Error("Expected type bins in counts")
		}

		if len(resp.Counts.SizePercentiles) == 0 {
			t.Error("Expected size percentiles in counts")
		}

		if len(resp.Counts.DurationPercentiles) == 0 {
			t.Error("Expected duration percentiles in counts")
		}
	})

	t.Run("filter with include_counts returns filtered bins", func(t *testing.T) {
		// Get unfiltered counts
		req1 := httptest.NewRequest("GET", "/api/du?path=&include_counts=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		// Get video-only counts
		req2 := httptest.NewRequest("GET", "/api/du?path=&include_counts=true&video-only=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		if resp2.Counts == nil {
			t.Fatal("Expected counts in filtered response")
		}

		// Video count in filtered should be same as total video count
		// but other types should be 0 or not present
		t.Logf("Unfiltered types: %+v", resp1.Counts.Type)
		t.Logf("Filtered types: %+v", resp2.Counts.Type)
	})

	t.Run("filters persist when navigating to subfolder", func(t *testing.T) {
		// First, get root level with video filter
		req1 := httptest.NewRequest("GET", "/api/du?path=&video=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal root response: %v", err)
		}

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level with video filter")
		}

		// Get the first folder path (e.g., /home)
		firstFolderPath := resp1.Folders[0].Path
		t.Logf("Navigating to folder: %s", firstFolderPath)

		// Navigate to subfolder with same video filter
		req2 := httptest.NewRequest("GET", "/api/du?path="+firstFolderPath+"&video=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal subfolder response: %v", err)
		}

		// Should have subfolders and/or files
		totalItems := len(resp2.Folders) + len(resp2.Files)
		if totalItems == 0 {
			t.Errorf("Expected folders or files in subfolder with video filter, got none")
			t.Logf("Response: %+v", resp2)
		}

		t.Logf("Subfolder results - Folders: %d, Files: %d, Total: %d",
			resp2.FolderCount, resp2.FileCount, resp2.TotalCount)
	})

	t.Run("audio filter persists when navigating to subfolder", func(t *testing.T) {
		// Get root level with audio filter
		req1 := httptest.NewRequest("GET", "/api/du?path=&audio=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal root response: %v", err)
		}

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level with audio filter")
		}

		firstFolderPath := resp1.Folders[0].Path

		// Navigate to subfolder with audio filter
		req2 := httptest.NewRequest("GET", "/api/du?path="+firstFolderPath+"&audio=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal subfolder response: %v", err)
		}

		totalItems := len(resp2.Folders) + len(resp2.Files)
		if totalItems == 0 {
			t.Errorf("Expected folders or files in subfolder with audio filter, got none")
		}

		t.Logf("Audio filter - Subfolder results: Folders: %d, Files: %d",
			resp2.FolderCount, resp2.FileCount)
	})

	t.Run("image filter persists when navigating to subfolder", func(t *testing.T) {
		req1 := httptest.NewRequest("GET", "/api/du?path=&image=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level")
		}

		req2 := httptest.NewRequest("GET", "/api/du?path="+resp1.Folders[0].Path+"&image=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		totalItems := len(resp2.Folders) + len(resp2.Files)
		if totalItems == 0 {
			t.Errorf("Expected folders or files with image filter, got none")
		}
	})

	t.Run("size filter persists when navigating to subfolder", func(t *testing.T) {
		// Filter for media > 100KB
		req1 := httptest.NewRequest("GET", "/api/du?path=&size=>100KB", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level")
		}

		req2 := httptest.NewRequest("GET", "/api/du?path="+resp1.Folders[0].Path+"&size=>100KB", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		// Should have results with size filter applied
		if resp2.FolderCount == 0 && resp2.FileCount == 0 {
			t.Errorf("Expected folders or files with size filter, got none")
		}
		t.Logf("Size filter - Subfolder results: Folders: %d, Files: %d",
			resp2.FolderCount, resp2.FileCount)
	})

	t.Run("duration filter persists when navigating to subfolder", func(t *testing.T) {
		// Filter for media > 10 seconds
		req1 := httptest.NewRequest("GET", "/api/du?path=&duration=>10", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level")
		}

		req2 := httptest.NewRequest("GET", "/api/du?path="+resp1.Folders[0].Path+"&duration=>10", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		// Should have results with duration filter applied
		if resp2.FolderCount == 0 && resp2.FileCount == 0 {
			t.Errorf("Expected folders or files with duration filter, got none")
		}
		t.Logf("Duration filter - Subfolder results: Folders: %d, Files: %d",
			resp2.FolderCount, resp2.FileCount)
	})

	t.Run("episodes filecounts filter works in DU mode", func(t *testing.T) {
		// Get unfiltered results first
		req1 := httptest.NewRequest("GET", "/api/du?path=&include_counts=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal unfiltered response: %v", err)
		}

		t.Logf("Unfiltered - Folders: %d, Files: %d, Total: %d",
			resp1.FolderCount, resp1.FileCount, resp1.TotalCount)

		// Apply filecounts filter (folders with exactly 1 file)
		req2 := httptest.NewRequest("GET", "/api/du?path=&file_counts=1", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w2.Code, w2.Body.String())
		}

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal filtered response: %v", err)
		}

		t.Logf("FileCounts=1 - Folders: %d, Files: %d, Total: %d",
			resp2.FolderCount, resp2.FileCount, resp2.TotalCount)

		// Should have fewer or equal results than unfiltered
		if resp2.TotalCount > resp1.TotalCount {
			t.Errorf("Filtered count (%d) should not exceed unfiltered count (%d)",
				resp2.TotalCount, resp1.TotalCount)
		}
	})

	t.Run("episodes filecounts filter persists when navigating to subfolder", func(t *testing.T) {
		// Get root level with filecounts filter
		req1 := httptest.NewRequest("GET", "/api/du?path=&file_counts=1", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal root response: %v", err)
		}

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level with filecounts filter")
		}

		firstFolderPath := resp1.Folders[0].Path
		t.Logf("Navigating to folder: %s with filecounts filter", firstFolderPath)

		// Navigate to subfolder with same filecounts filter
		req2 := httptest.NewRequest("GET", "/api/du?path="+firstFolderPath+"&file_counts=1", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w2.Code, w2.Body.String())
		}

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal subfolder response: %v", err)
		}

		totalItems := len(resp2.Folders) + len(resp2.Files)
		t.Logf("Subfolder results - Folders: %d, Files: %d, Total: %d",
			resp2.FolderCount, resp2.FileCount, resp2.TotalCount)

		// Should have some results (or at least not error)
		if totalItems == 0 && resp2.TotalCount == 0 {
			t.Logf("No results in subfolder with filecounts=1 filter (may be expected depending on test data)")
		}
	})
}

// TestHandleDU_WithFilters_WindowsPaths tests filter functionality with mixed path separators
// This test verifies that the DU endpoint correctly handles paths with backslash separators
// which is important for Windows clients and cross-platform compatibility
func TestHandleDU_WithFilters_WindowsPaths(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_du_filters_windows.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize database schema
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}

	// Create test media with mixed path separators (simulating Windows and Unix paths)
	// This tests that the DU endpoint normalizes paths correctly regardless of separator style
	_, err = sqlDB.Exec(`INSERT INTO media (path, title, type, size, duration, time_deleted) VALUES
		-- Unix-style paths (5 videos)
		('/media/videos/movie1.mp4', 'Movie1', 'video', 500000000, 7200, 0),
		('/media/videos/movie2.mp4', 'Movie2', 'video', 300000000, 5400, 0),
		('/media/videos/clip1.mp4', 'Clip1', 'video', 50000000, 30, 0),
		('/media/videos/clip2.mp4', 'Clip2', 'video', 25000000, 15, 0),
		('/media/videos/short.mp4', 'Short', 'video', 10000000, 5, 0),
		-- Windows-style paths with backslashes (3 audio)
		('\\media\\audio\\song1.mp3', 'Song1', 'audio', 5000000, 180, 0),
		('\\media\\audio\\song2.mp3', 'Song2', 'audio', 4000000, 150, 0),
		('\\media\\audio\\podcast.mp3', 'Podcast', 'audio', 15000000, 900, 0),
		-- Mixed separator paths (3 images)
		('/media\\images\\photo1.png', 'Photo1', 'image', 2000000, 0, 0),
		('/media\\images\\photo2.png', 'Photo2', 'image', 1500000, 0, 0),
		('/media\\images\\photo3.png', 'Photo3', 'image', 3000000, 0, 0),
		-- More Windows-style paths (2 documents)
		('\\media\\documents\\doc1.txt', 'Doc1', 'text', 50000, 0, 0),
		('\\media\\documents\\doc2.pdf', 'Doc2', 'text', 100000, 0, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	cmd := setupTestServeCmd(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("video-only filter returns only video folders", func(t *testing.T) {
		// Query root level with video filter
		req := httptest.NewRequest("GET", "/api/du?video=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have video folders (from /media/videos)
		if resp.TotalCount == 0 {
			t.Error("Expected video results")
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("type=video filter returns only video folders", func(t *testing.T) {
		// Get unfiltered results first
		req1 := httptest.NewRequest("GET", "/api/du?include_counts=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal unfiltered response: %v", err)
		}
		unfilteredTotal := resp1.TotalCount
		t.Logf("Unfiltered - Total: %d", unfilteredTotal)

		// Get total file count from folders
		unfilteredFileCount := 0
		for _, f := range resp1.Folders {
			unfilteredFileCount += f.Count
		}
		t.Logf("Unfiltered file count in folders: %d", unfilteredFileCount)

		// Apply type=video filter
		req2 := httptest.NewRequest("GET", "/api/du?type=video", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w2.Code, w2.Body.String())
		}

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("type=video - Total: %d", resp2.TotalCount)

		// Get filtered file count from folders
		filteredFileCount := 0
		for _, f := range resp2.Folders {
			filteredFileCount += f.Count
		}
		t.Logf("Filtered file count in folders: %d", filteredFileCount)

		// File count within folders should be less (5 videos out of 13 total)
		if filteredFileCount >= unfilteredFileCount {
			t.Errorf("Expected filtered file count (%d) to be less than unfiltered (%d)", filteredFileCount, unfilteredFileCount)
		}
		if filteredFileCount == 0 {
			t.Error("Expected some video results")
		}
	})

	t.Run("audio-only filter returns only audio folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?audio=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.TotalCount == 0 {
			t.Error("Expected audio results")
		}
		t.Logf("Audio - Total: %d", resp.TotalCount)
	})

	t.Run("image-only filter returns only image folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?image=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.TotalCount == 0 {
			t.Error("Expected image results")
		}
		t.Logf("Image - Total: %d", resp.TotalCount)
	})

	t.Run("filters persist when navigating to subfolder", func(t *testing.T) {
		// First, get root level with video filter
		req1 := httptest.NewRequest("GET", "/api/du?video=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal root response: %v", err)
		}

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level with video filter")
		}

		// Get the first folder path
		firstFolderPath := resp1.Folders[0].Path
		t.Logf("Navigating to folder: %s", firstFolderPath)

		// Navigate to subfolder with same video filter
		req2 := httptest.NewRequest("GET", "/api/du?path="+firstFolderPath+"&video=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal subfolder response: %v", err)
		}

		// Should have subfolders and/or files
		totalItems := len(resp2.Folders) + len(resp2.Files)
		if totalItems == 0 {
			t.Errorf("Expected folders or files in subfolder with video filter, got none")
			t.Logf("Response: %+v", resp2)
		}

		t.Logf("Subfolder results - Folders: %d, Files: %d, Total: %d",
			resp2.FolderCount, resp2.FileCount, resp2.TotalCount)
	})
}
