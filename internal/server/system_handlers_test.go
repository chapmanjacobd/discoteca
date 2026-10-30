package server_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/server"
)

func TestServeHandlers_Health(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_health.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)
	sqlDB.Close()

	cmd := &server.Server{
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
	cmd := &server.Server{}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Errorf("Expected image/svg+xml, got %q", got)
	}
	if w.Body.Len() == 0 {
		t.Error("Expected favicon response body")
	}
}
