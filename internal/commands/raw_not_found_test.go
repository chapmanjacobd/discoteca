package commands

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func TestRawNotFound(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_notfound.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	t.Run("MediaNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/raw?db="+dbPath+"&path=/nonexistent.mp4", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("DatabaseNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/raw?db=missing.db&path=/some.mp4", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should be 400 Bad Request if DB is not in allowed list
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}
