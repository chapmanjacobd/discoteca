package commands

import (
	"bytes"
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

func TestServeHandlers_Health(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_health.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected 'OK', got %s", w.Body.String())
	}
}

func TestServeHandlers_Favicon(t *testing.T) {
	cmd := &ServeCmd{}
	mux := cmd.Mux()

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404 even if we don't have a real favicon
	if w.Code == http.StatusNotFound {
		t.Error("Favicon endpoint should be handled")
	}
}

// TestHandleCategories tests the categories endpoint
func TestHandleCategories(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_categories.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, categories, time_deleted) VALUES 
		('/tmp/test1.mp4', 'Test1', 'video', 'comedy;action', 0),
		('/tmp/test2.mp4', 'Test2', 'video', 'comedy', 0),
		('/tmp/test3.mp4', 'Test3', 'video', '', 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	req := httptest.NewRequest("GET", "/api/categories", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var categories []models.CatStat
	if err := json.NewDecoder(w.Body).Decode(&categories); err != nil {
		t.Fatal(err)
	}

	if len(categories) == 0 {
		t.Error("Expected at least one category")
	}

	foundComedy := false
	for _, cat := range categories {
		if cat.Category == "comedy" && cat.Count == 2 {
			foundComedy = true
		}
	}
	if !foundComedy {
		t.Error("Expected comedy category with count 2")
	}
}

// TestHandleGenres tests the genres endpoint
func TestHandleGenres(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_genres.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, genre, time_deleted) VALUES 
		('/tmp/test1.mp4', 'Test1', 'video', 'Action', 0),
		('/tmp/test2.mp4', 'Test2', 'video', 'Action', 0),
		('/tmp/test3.mp4', 'Test3', 'video', 'Comedy', 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	req := httptest.NewRequest("GET", "/api/genres", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var genres []models.CatStat
	if err := json.NewDecoder(w.Body).Decode(&genres); err != nil {
		t.Fatal(err)
	}

	if len(genres) == 0 {
		t.Error("Expected at least one genre")
	}

	foundAction := false
	for _, g := range genres {
		if g.Category == "Action" && g.Count == 2 {
			foundAction = true
		}
	}
	if !foundAction {
		t.Error("Expected Action genre with count 2")
	}
}

// TestHandleRatings tests the ratings endpoint
func TestHandleRatings(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_ratings.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, score, time_deleted) VALUES 
		('/tmp/test1.mp4', 'Test1', 'video', 5.0, 0),
		('/tmp/test2.mp4', 'Test2', 'video', 5.0, 0),
		('/tmp/test3.mp4', 'Test3', 'video', 3.0, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	req := httptest.NewRequest("GET", "/api/ratings", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var ratings []models.RatStat
	if err := json.NewDecoder(w.Body).Decode(&ratings); err != nil {
		t.Fatal(err)
	}

	if len(ratings) == 0 {
		t.Error("Expected at least one rating")
	}

	found5Star := false
	for _, r := range ratings {
		if r.Rating == 5 && r.Count == 2 {
			found5Star = true
		}
	}
	if !found5Star {
		t.Error("Expected 5-star rating with count 2")
	}
}

// TestHandleRate tests the rate endpoint
func TestHandleRate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_rate.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, score, time_deleted) VALUES 
		('/tmp/test1.mp4', 'Test1', 'video', 0, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
		ReadOnly:  false,
	}
	mux := cmd.Mux()

	t.Run("ValidRate", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":  "/tmp/test1.mp4",
			"score": 4.5,
		})
		req := httptest.NewRequest("POST", "/api/rate", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Verify database update
		sqlDB, _ := sql.Open("sqlite3", dbPath)
		defer sqlDB.Close()
		var score float64
		sqlDB.QueryRow("SELECT score FROM media WHERE path = ?", "/tmp/test1.mp4").Scan(&score)
		if score != 4.5 {
			t.Errorf("Expected score 4.5, got %f", score)
		}
	})

	t.Run("ReadOnlyMode", func(t *testing.T) {
		cmd.ReadOnly = true
		reqBody, _ := json.Marshal(map[string]any{
			"path":  "/tmp/test1.mp4",
			"score": 3.0,
		})
		req := httptest.NewRequest("POST", "/api/rate", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
		cmd.ReadOnly = false
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/rate", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})
}

// TestHandleDelete tests the delete endpoint
func TestHandleDelete(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_delete.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, time_deleted) VALUES 
		('/tmp/test1.mp4', 'Test1', 'video', 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
		ReadOnly:  false,
	}
	mux := cmd.Mux()

	t.Run("MarkAsDeleted", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":    "/tmp/test1.mp4",
			"restore": false,
		})
		req := httptest.NewRequest("POST", "/api/delete", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Verify database update
		sqlDB, _ := sql.Open("sqlite3", dbPath)
		defer sqlDB.Close()
		var timeDeleted int64
		sqlDB.QueryRow("SELECT time_deleted FROM media WHERE path = ?", "/tmp/test1.mp4").Scan(&timeDeleted)
		if timeDeleted == 0 {
			t.Error("Expected time_deleted to be set")
		}
	})

	t.Run("Restore", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":    "/tmp/test1.mp4",
			"restore": true,
		})
		req := httptest.NewRequest("POST", "/api/delete", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Verify database update
		sqlDB, _ := sql.Open("sqlite3", dbPath)
		defer sqlDB.Close()
		var timeDeleted int64
		sqlDB.QueryRow("SELECT time_deleted FROM media WHERE path = ?", "/tmp/test1.mp4").Scan(&timeDeleted)
		if timeDeleted != 0 {
			t.Error("Expected time_deleted to be 0 after restore")
		}
	})

	t.Run("ReadOnlyMode", func(t *testing.T) {
		cmd.ReadOnly = true
		reqBody, _ := json.Marshal(map[string]any{
			"path":    "/tmp/test1.mp4",
			"restore": false,
		})
		req := httptest.NewRequest("POST", "/api/delete", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
		cmd.ReadOnly = false
	})
}

// TestHandleProgress tests the progress endpoint
func TestHandleProgress(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_progress.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, type, playhead, play_count, time_deleted) VALUES 
		('/tmp/test1.mp4', 'Test1', 'video', 0, 0, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
		ReadOnly:  false,
	}
	mux := cmd.Mux()

	t.Run("UpdateProgress", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":      "/tmp/test1.mp4",
			"playhead":  120,
			"completed": false,
		})
		req := httptest.NewRequest("POST", "/api/progress", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Verify database update
		sqlDB, _ := sql.Open("sqlite3", dbPath)
		defer sqlDB.Close()
		var playhead int64
		var playCount int64
		sqlDB.QueryRow("SELECT playhead, play_count FROM media WHERE path = ?", "/tmp/test1.mp4").Scan(&playhead, &playCount)
		if playhead != 120 {
			t.Errorf("Expected playhead 120, got %d", playhead)
		}
		if playCount != 0 {
			t.Errorf("Expected play_count 0, got %d", playCount)
		}
	})

	t.Run("CompletePlayback", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":      "/tmp/test1.mp4",
			"playhead":  600,
			"completed": true,
		})
		req := httptest.NewRequest("POST", "/api/progress", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Verify database update
		sqlDB, _ := sql.Open("sqlite3", dbPath)
		defer sqlDB.Close()
		var playhead int64
		var playCount int64
		sqlDB.QueryRow("SELECT playhead, play_count FROM media WHERE path = ?", "/tmp/test1.mp4").Scan(&playhead, &playCount)
		if playCount != 1 {
			t.Errorf("Expected play_count 1 after completion, got %d", playCount)
		}
	})

	t.Run("ReadOnlyMode", func(t *testing.T) {
		cmd.ReadOnly = true
		reqBody, _ := json.Marshal(map[string]any{
			"path":      "/tmp/test1.mp4",
			"playhead":  50,
			"completed": false,
		})
		req := httptest.NewRequest("POST", "/api/progress", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
		cmd.ReadOnly = false
	})
}
