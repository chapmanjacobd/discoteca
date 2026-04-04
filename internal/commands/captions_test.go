package commands

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestHandleDU_CaptionsView(t *testing.T) {
	// Create temporary test database
	tmpDB, err := os.CreateTemp(t.TempDir(), "disco_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	defer os.Remove(tmpDB.Name())
	tmpDB.Close()

	db, err := sql.Open("sqlite3", tmpDB.Name())
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE media (
			path TEXT PRIMARY KEY,
			title TEXT,
			media_type TEXT,
			size INTEGER,
			duration INTEGER,
			time_deleted INTEGER DEFAULT 0
		);
		CREATE TABLE captions (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			media_path TEXT,
			time REAL,
			text TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Insert test media
	_, err = db.Exec(`
		INSERT INTO media (path, title, media_type, size, duration) VALUES
			(?, 'Test Video 1', 'video/mp4', 1024, 120),
			(?, 'Test Video 2', 'video/mp4', 2048, 180),
			(?, 'Test Video 3', 'video/mp4', 512, 90)`,
		filepath.FromSlash("/videos/test1.mp4"),
		filepath.FromSlash("/videos/test2.mp4"),
		filepath.FromSlash("/videos/test3.mp4"))
	if err != nil {
		t.Fatalf("Failed to insert media: %v", err)
	}

	// Insert test captions
	_, err = db.Exec(`
		INSERT INTO captions (media_path, time, text) VALUES
			(?, 10.5, 'Hello world from video 1'),
			(?, 20.3, 'Another caption in video 1'),
			(?, 15.0, 'Caption from video 2'),
			(?, 5.0, 'First caption in video 3'),
			(?, 25.0, 'Second caption in video 3')`,
		filepath.FromSlash("/videos/test1.mp4"),
		filepath.FromSlash("/videos/test1.mp4"),
		filepath.FromSlash("/videos/test2.mp4"),
		filepath.FromSlash("/videos/test3.mp4"),
		filepath.FromSlash("/videos/test3.mp4"))
	if err != nil {
		t.Fatalf("Failed to insert captions: %v", err)
	}

	db.Close()

	// Create ServeCmd with test database
	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
	}
	defer cmd.Close()

	t.Run("GetAllCaptions returns all captions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query?captions=true&limit=100", nil)
		w := httptest.NewRecorder()

		cmd.HandleQuery(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var media []models.MediaWithDB
		if err := json.Unmarshal(w.Body.Bytes(), &media); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(media) != 5 {
			t.Errorf("Expected 5 captions, got %d", len(media))
		}

		// Verify caption data structure
		for _, m := range media {
			if m.Path == "" {
				t.Error("Expected path to be set")
			}
			if m.CaptionText == "" {
				t.Error("Expected caption text to be set")
			}
			if m.CaptionTime <= 0 {
				t.Error("Expected caption time to be positive")
			}
		}
	})

	t.Run("GetAllCaptions respects limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query?captions=true&limit=2", nil)
		w := httptest.NewRecorder()

		cmd.HandleQuery(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var media []models.MediaWithDB
		if err := json.Unmarshal(w.Body.Bytes(), &media); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(media) != 2 {
			t.Errorf("Expected 2 captions with limit=2, got %d", len(media))
		}
	})

	t.Run("GetAllCaptions returns X-Total-Count header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query?captions=true&limit=2", nil)
		w := httptest.NewRecorder()

		cmd.HandleQuery(w, req)

		totalCount := w.Header().Get("X-Total-Count")
		if totalCount == "" {
			t.Error("Expected X-Total-Count header to be set")
		}
	})

	t.Run("GetAllCaptions with all flag returns all captions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query?captions=true&all=true", nil)
		w := httptest.NewRecorder()

		cmd.HandleQuery(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var media []models.MediaWithDB
		if err := json.Unmarshal(w.Body.Bytes(), &media); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should return all 5 captions
		if len(media) != 5 {
			t.Errorf("Expected 5 captions with all=true, got %d", len(media))
		}
	})
}

func TestHandleDU_CaptionsView_EmptyDatabase(t *testing.T) {
	// Create temporary test database
	tmpDB, err := os.CreateTemp(t.TempDir(), "disco_test_empty_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	defer os.Remove(tmpDB.Name())
	tmpDB.Close()

	db, err := sql.Open("sqlite3", tmpDB.Name())
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE media (
			path TEXT PRIMARY KEY,
			title TEXT,
			media_type TEXT,
			size INTEGER,
			duration INTEGER,
			time_deleted INTEGER DEFAULT 0
		);
		CREATE TABLE captions (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			media_path TEXT,
			time REAL,
			text TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	db.Close()

	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
	}
	defer cmd.Close()

	t.Run("GetAllCaptions with no captions returns empty array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query?captions=true&limit=100", nil)
		w := httptest.NewRecorder()

		cmd.HandleQuery(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var media []models.MediaWithDB
		if err := json.Unmarshal(w.Body.Bytes(), &media); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(media) != 0 {
			t.Errorf("Expected 0 captions, got %d", len(media))
		}
	})
}

func TestHandleDU_CaptionsView_MultipleDatabases(t *testing.T) {
	// Create two temporary test databases
	tmpDB1, err := os.CreateTemp(t.TempDir(), "disco_test_multi1_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db1: %v", err)
	}
	defer os.Remove(tmpDB1.Name())
	tmpDB1.Close()

	tmpDB2, err := os.CreateTemp(t.TempDir(), "disco_test_multi2_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db2: %v", err)
	}
	defer os.Remove(tmpDB2.Name())
	tmpDB2.Close()

	// Setup first database
	db1, err := sql.Open("sqlite3", tmpDB1.Name())
	if err != nil {
		t.Fatalf("Failed to open db1: %v", err)
	}
	_, err = db1.Exec(`
		CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT, media_type TEXT, size INTEGER, duration INTEGER, time_deleted INTEGER DEFAULT 0);
		CREATE TABLE captions (rowid INTEGER PRIMARY KEY AUTOINCREMENT, media_path TEXT, time REAL, text TEXT);
		INSERT INTO media (path, title, media_type, size, duration) VALUES (?, 'DB1 Video', 'video/mp4', 1024, 120);
		INSERT INTO captions (media_path, time, text) VALUES (?, 10.0, 'Caption from DB1');
	`, filepath.FromSlash("/db1/video1.mp4"), filepath.FromSlash("/db1/video1.mp4"))
	if err != nil {
		t.Fatalf("Failed to setup db1: %v", err)
	}
	db1.Close()

	// Setup second database
	db2, err := sql.Open("sqlite3", tmpDB2.Name())
	if err != nil {
		t.Fatalf("Failed to open db2: %v", err)
	}
	_, err = db2.Exec(`
		CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT, media_type TEXT, size INTEGER, duration INTEGER, time_deleted INTEGER DEFAULT 0);
		CREATE TABLE captions (rowid INTEGER PRIMARY KEY AUTOINCREMENT, media_path TEXT, time REAL, text TEXT);
		INSERT INTO media (path, title, media_type, size, duration) VALUES (?, 'DB2 Video', 'video/mp4', 2048, 180);
		INSERT INTO captions (media_path, time, text) VALUES (?, 20.0, 'Caption from DB2');
	`, filepath.FromSlash("/db2/video1.mp4"), filepath.FromSlash("/db2/video1.mp4"))
	if err != nil {
		t.Fatalf("Failed to setup db2: %v", err)
	}
	db2.Close()

	cmd := &ServeCmd{
		Databases: []string{tmpDB1.Name(), tmpDB2.Name()},
	}
	defer cmd.Close()

	t.Run("GetAllCaptions merges results from multiple databases", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query?captions=true&limit=100", nil)
		w := httptest.NewRecorder()

		cmd.HandleQuery(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var media []models.MediaWithDB
		if err := json.Unmarshal(w.Body.Bytes(), &media); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have captions from both databases
		if len(media) != 2 {
			t.Errorf("Expected 2 captions from 2 databases, got %d", len(media))
		}

		// Verify captions from both DBs are present
		hasDB1 := false
		hasDB2 := false
		for _, m := range media {
			if m.Path == filepath.FromSlash("/db1/video1.mp4") {
				hasDB1 = true
			}
			if m.Path == filepath.FromSlash("/db2/video1.mp4") {
				hasDB2 = true
			}
		}

		if !hasDB1 {
			t.Error("Expected captions from DB1")
		}
		if !hasDB2 {
			t.Error("Expected captions from DB2")
		}
	})
}
