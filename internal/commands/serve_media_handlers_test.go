package commands_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
)

// TestHandleRate tests the rate endpoint
func TestHandleRate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_rate.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, score, time_deleted) VALUES 
		(?, 'Test1', 'video', 0, 0)`, filepath.FromSlash("/tmp/test1.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
		ReadOnly:  false,
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("ValidRate", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":  filepath.FromSlash("/tmp/test1.mp4"),
			"score": 4.5,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/rate", bytes.NewBuffer(reqBody))
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
		sqlDB.QueryRow("SELECT score FROM media WHERE path = ?", filepath.FromSlash("/tmp/test1.mp4")).Scan(&score)
		if score != 4.5 {
			t.Errorf("Expected score 4.5, got %f", score)
		}
	})

	t.Run("ReadOnlyMode", func(t *testing.T) {
		cmd.ReadOnly = true
		reqBody, _ := json.Marshal(map[string]any{
			"path":  filepath.FromSlash("/tmp/test1.mp4"),
			"score": 3.0,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/rate", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
		cmd.ReadOnly = false
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/rate", nil)
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
	db.InitDB(context.Background(), sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, time_deleted) VALUES 
		(?, 'Test1', 'video', 0)`, filepath.FromSlash("/tmp/test1.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
		ReadOnly:  false,
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("MarkAsDeleted", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":    filepath.FromSlash("/tmp/test1.mp4"),
			"restore": false,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/delete", bytes.NewBuffer(reqBody))
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
		sqlDB.QueryRow("SELECT time_deleted FROM media WHERE path = ?", filepath.FromSlash("/tmp/test1.mp4")).
			Scan(&timeDeleted)
		if timeDeleted == 0 {
			t.Error("Expected time_deleted to be set")
		}
	})

	t.Run("Restore", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":    filepath.FromSlash("/tmp/test1.mp4"),
			"restore": true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/delete", bytes.NewBuffer(reqBody))
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
		sqlDB.QueryRow("SELECT time_deleted FROM media WHERE path = ?", filepath.FromSlash("/tmp/test1.mp4")).
			Scan(&timeDeleted)
		if timeDeleted != 0 {
			t.Error("Expected time_deleted to be 0 after restore")
		}
	})

	t.Run("ReadOnlyMode", func(t *testing.T) {
		cmd.ReadOnly = true
		reqBody, _ := json.Marshal(map[string]any{
			"path":    filepath.FromSlash("/tmp/test1.mp4"),
			"restore": false,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/delete", bytes.NewBuffer(reqBody))
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
	db.InitDB(context.Background(), sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, playhead, play_count, time_deleted) VALUES 
		(?, 'Test1', 'video', 0, 0, 0)`, filepath.FromSlash("/tmp/test1.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
		ReadOnly:  false,
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("UpdateProgress", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":      filepath.FromSlash("/tmp/test1.mp4"),
			"playhead":  120,
			"completed": false,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/progress", bytes.NewBuffer(reqBody))
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
		sqlDB.QueryRow("SELECT playhead, play_count FROM media WHERE path = ?", filepath.FromSlash("/tmp/test1.mp4")).
			Scan(&playhead, &playCount)
		if playhead != 120 {
			t.Errorf("Expected playhead 120, got %d", playhead)
		}
		if playCount != 0 {
			t.Errorf("Expected play_count 0, got %d", playCount)
		}
	})

	t.Run("CompletePlayback", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"path":      filepath.FromSlash("/tmp/test1.mp4"),
			"playhead":  600,
			"completed": true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/progress", bytes.NewBuffer(reqBody))
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
		sqlDB.QueryRow("SELECT playhead, play_count FROM media WHERE path = ?", filepath.FromSlash("/tmp/test1.mp4")).
			Scan(&playhead, &playCount)
		if playCount != 1 {
			t.Errorf("Expected play_count 1 after completion, got %d", playCount)
		}
	})

	t.Run("ReadOnlyMode", func(t *testing.T) {
		cmd.ReadOnly = true
		reqBody, _ := json.Marshal(map[string]any{
			"path":      filepath.FromSlash("/tmp/test1.mp4"),
			"playhead":  50,
			"completed": false,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/progress", bytes.NewBuffer(reqBody))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
		cmd.ReadOnly = false
	})
}
