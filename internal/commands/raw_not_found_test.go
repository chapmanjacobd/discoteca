package commands_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
)

func TestRawNotFound(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_notfound.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("MediaNotFound", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/raw?db="+dbPath+"&path="+filepath.FromSlash("/nonexistent.mp4"),
			nil,
		)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("DatabaseNotFound", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/raw?db=missing.db&path="+filepath.FromSlash("/some.mp4"), nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should be 400 Bad Request if DB is not in allowed list
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}
