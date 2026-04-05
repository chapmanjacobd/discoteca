package commands_test

import (
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

func TestServeAPI_Query(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_api.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.InitDB(context.Background(), sqlDB)

	// Add test data
	_, err = sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, size, time_deleted) VALUES ('/tmp/test.mp4', 'Test Video', 'video', 1024, 0)`,
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("ValidQuery", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query?db="+dbPath, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var results []map[string]any
		if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
			t.Fatal(err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
		req.Header.Set("X-Disco-Token", "wrong")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})
}

func TestServeAPI_Metadata(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_meta.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)
	sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, time_deleted) VALUES ('/tmp/meta.mp4', 'Meta Video', 'video', 0)`,
	)
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/api/metadata?db="+dbPath+"&path=/tmp/meta.mp4", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var meta map[string]any
	json.NewDecoder(w.Body).Decode(&meta)
	if meta["title"] != "Meta Video" {
		t.Errorf("Expected title 'Meta Video', got %v", meta["title"])
	}
}
