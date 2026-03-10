package commands

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/db"
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

	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, time_deleted) VALUES (?, 'Test', 'video', 0)`, subPath)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	t.Run("ValidVTTSubtitle", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/subtitles?path="+subPath, nil)
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
		req := httptest.NewRequest("GET", "/api/subtitles", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("UnauthorizedPath", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/subtitles?path=/etc/passwd", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for unauthorized path, got %d", w.Code)
		}
	})
}

// TestHandleThumbnail tests the thumbnail endpoint
func TestHandleThumbnail(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_thumbnail.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Create a test image file with valid JPEG header
	// Minimal JPEG header (SOI marker + JFIF)
	imgData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9}
	imgPath := filepath.Join(tempDir, "test.jpg")
	if err := os.WriteFile(imgPath, imgData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, size, time_deleted) VALUES (?, 'Test', 'image', 1024, 0)`, imgPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ValidImageThumbnail", func(t *testing.T) {
		cmd := &ServeCmd{
			Databases: []string{dbPath},
		}
		mux := cmd.Mux()

		req := httptest.NewRequest("GET", "/api/thumbnail?path="+imgPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		// Should serve the image directly (small images)
		if w.Header().Get("Content-Type") != "image/jpeg" {
			t.Errorf("Expected Content-Type image/jpeg, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("MissingPath", func(t *testing.T) {
		cmd := &ServeCmd{
			Databases: []string{dbPath},
		}
		mux := cmd.Mux()

		req := httptest.NewRequest("GET", "/api/thumbnail", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("UnauthorizedPath", func(t *testing.T) {
		cmd := &ServeCmd{
			Databases: []string{dbPath},
		}
		mux := cmd.Mux()

		req := httptest.NewRequest("GET", "/api/thumbnail?path=/etc/passwd", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for unauthorized path, got %d", w.Code)
		}
	})

	t.Run("PlaceholderForUnknownType", func(t *testing.T) {
		// Create a text file
		txtPath := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(txtPath, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, size, time_deleted) VALUES (?, 'Test', 'text', 10, 0)`, txtPath)
		if err != nil {
			t.Fatal(err)
		}

		cmd := &ServeCmd{
			Databases: []string{dbPath},
		}
		mux := cmd.Mux()

		req := httptest.NewRequest("GET", "/api/thumbnail?path="+txtPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		// Should return SVG placeholder
		if w.Header().Get("Content-Type") != "image/svg+xml" {
			t.Errorf("Expected Content-Type image/svg+xml for placeholder, got %s", w.Header().Get("Content-Type"))
		}
	})

	sqlDB.Close()
}

// TestHandleDU tests the disk usage endpoint
func TestHandleDU(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_du.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Create test media in different directories
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, size, duration, time_deleted) VALUES 
		('/videos/movies/movie1.mp4', 'Movie1', 'video', 1073741824, 7200, 0),
		('/videos/movies/movie2.mp4', 'Movie2', 'video', 536870912, 3600, 0),
		('/videos/music/song1.mp4', 'Song1', 'video', 268435456, 300, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	t.Run("RootLevel", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du", nil)
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
	})

	t.Run("SpecificPath", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=/videos", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
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
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, time_deleted) VALUES 
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
	mux := cmd.Mux()

	t.Run("GroupEpisodes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/episodes", nil)
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
