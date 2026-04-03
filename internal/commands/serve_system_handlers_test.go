package commands

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

func TestServeHandlers_Health(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_health.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
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
	t.Parallel()
	cmd := &ServeCmd{}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404 even if we don't have a real favicon
	if w.Code == http.StatusNotFound {
		t.Error("Favicon endpoint should be handled")
	}
}
