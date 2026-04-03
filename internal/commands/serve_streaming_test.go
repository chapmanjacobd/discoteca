package commands

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// TestHandleSubtitles tests the subtitles endpoint
func TestHandleSubtitles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_subtitles.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Create a test subtitle file
	subPath := filepath.Join(tempDir, "test.vtt")
	subContent := `WEBVTT

00:00:01.000 --> 00:00:04.000
Test subtitle
`
	if err := os.WriteFile(subPath, []byte(subContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, time_deleted) VALUES (?, 'Test', 'video', 0)`,
		subPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("ValidVTTSubtitle", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path="+subPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		if w.Header().Get("Content-Type") != "text/vtt" {
			t.Errorf("Expected Content-Type text/vtt, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("MissingPath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("UnauthorizedPath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles?path=/etc/passwd", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for unauthorized path, got %d", w.Code)
		}
	})
}

// TestHandleDU tests the disk usage endpoint
func TestHandleDU(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_du.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize database schema
	if err := db.InitDB(sqlDB); err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}

	// Create test media in different directories
	// Include both Unix-style and Windows-style paths to test cross-platform compatibility
	_, err = sqlDB.Exec(`INSERT INTO media (path, title, media_type, size, duration, time_deleted) VALUES
		('/videos/movies/movie1.mp4', 'Movie1', 'video', 1073741824, 7200, 0),
		('/videos/movies/movie2.mp4', 'Movie2', 'video', 536870912, 3600, 0),
		('/videos/music/song1.mp4', 'Song1', 'video', 268435456, 300, 0),
		-- Windows-style paths (stored as-is in database)
		('\\videos\\movies\\movie3.mp4', 'Movie3', 'video', 800000000, 5400, 0),
		('\\videos\\music\\song2.mp4', 'Song2', 'video', 150000000, 240, 0),
		-- Mixed separator paths
		('/videos\\tv\\show1.mp4', 'Show1', 'video', 400000000, 1800, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("RootLevel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/du", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		// Should return folder statistics
		if w.Header().Get("X-Total-Count") == "" {
			t.Error("Expected X-Total-Count header")
		}

		// Verify new response format: {folders?: [], files?: []}
		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have folders array (not a flat array)
		if resp.Folders == nil {
			t.Error("Expected folders array in response")
		}

		// Verify folders have expected fields
		for _, folder := range resp.Folders {
			if folder.Path == "" {
				t.Error("Expected folder to have path")
			}
		}
	})

	t.Run("SpecificPath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		// Verify new response format
		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have folders and/or files arrays
		if resp.Folders == nil {
			t.Error("Expected folders array in response")
		}
		if resp.Files == nil {
			t.Error("Expected files array in response (can be empty)")
		}
	})

	t.Run("DirectFiles", func(t *testing.T) {
		// Test that direct files are returned in the files array, not as fake folders
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos/movies", nil)
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

		// Should have files in the files array (movie1.mp4, movie2.mp4)
		// Note: movie3.mp4 has Windows-style path, so it won't match /videos/movies
		if len(resp.Files) == 0 && len(resp.Folders) == 0 {
			t.Error("Expected files or folders in response")
		}

		// Verify files are not duplicated as fake folders
		for _, folder := range resp.Folders {
			if folder.Count == 0 && len(folder.Files) == 1 {
				t.Error("Files should be in the files array, not wrapped as fake folders")
			}
		}
	})

	t.Run("WindowsStylePath", func(t *testing.T) {
		// Test that Windows-style backslash paths work correctly
		// This ensures cross-platform compatibility
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=\\videos\\movies", nil)
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

		// Should find movie3.mp4 which has Windows-style path in database
		// After normalization, \\videos\\movies should match both /videos/movies and \\videos\\movies
		if len(resp.Files) == 0 && len(resp.Folders) == 0 {
			t.Error("Expected files or folders for Windows-style path")
		}
	})

	t.Run("MixedStylePath", func(t *testing.T) {
		// Test that mixed separator paths work correctly
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos\\movies", nil)
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

		// Should find files after normalizing mixed separators
		if len(resp.Files) == 0 && len(resp.Folders) == 0 {
			t.Error("Expected files or folders for mixed-style path")
		}
	})

	t.Run("WindowsAbsolutePath", func(t *testing.T) {
		// Test Windows absolute path with drive letter
		// This simulates how Windows clients would query paths
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=C:\\videos\\movies", nil)
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

		// Path normalization should handle Windows drive letters
		// Even though our test data uses Unix paths, the query should not fail
		if resp.Folders == nil {
			t.Error("Expected folders array (can be empty)")
		}
	})

	t.Run("WindowsRootPath", func(t *testing.T) {
		// Test Windows root path (drive letter only)
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=C:\\", nil)
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

		if resp.Folders == nil {
			t.Error("Expected folders array (can be empty)")
		}
	})

	t.Run("UNCPath", func(t *testing.T) {
		// Test UNC path (network share)
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=\\\\server\\share\\videos", nil)
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

		if resp.Folders == nil {
			t.Error("Expected folders array (can be empty)")
		}
	})

	t.Run("PathWithDotComponents", func(t *testing.T) {
		// Test path with . and .. components
		// FromURL normalizes the path: /videos/./movies/../music -> /videos/music
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos/./movies/../music", nil)
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

		// Should normalize to /videos/music and find the song
		if len(resp.Files) == 0 && len(resp.Folders) == 0 {
			t.Error("Expected files or folders for normalized path")
		}
	})

	t.Run("PathWithDoubleSeparators", func(t *testing.T) {
		// Test path with double separators
		// FromURL normalizes: /videos//movies -> /videos/movies
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=/videos//movies", nil)
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

		// Should normalize to /videos/movies and find the movies
		if len(resp.Files) == 0 && len(resp.Folders) == 0 {
			t.Error("Expected files or folders for normalized path")
		}
	})

	t.Run("WindowsPathWithDotDot", func(t *testing.T) {
		// Test Windows path with .. components
		// FromURL normalizes: \videos\movies\..\music -> /videos/music (on Linux)
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=\\videos\\movies\\..\\music", nil)
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

		// Should normalize to /videos/music and find the song
		if len(resp.Files) == 0 && len(resp.Folders) == 0 {
			t.Error("Expected files or folders for normalized path")
		}
	})
}

// TestHandleEpisodes tests the episodes endpoint
func TestHandleEpisodes(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_episodes.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Create test TV show episodes with same parent path
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, time_deleted) VALUES 
		('/shows/MyShow/MyShow.S01E01.mp4', 'Episode 1', 'video', 0),
		('/shows/MyShow/MyShow.S01E02.mp4', 'Episode 2', 'video', 0),
		('/shows/MyShow/MyShow.S01E03.mp4', 'Episode 3', 'video', 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("GroupEpisodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/episodes", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		// Should return grouped episodes
		if w.Header().Get("X-Total-Count") == "" {
			t.Error("Expected X-Total-Count header")
		}
	})
}
