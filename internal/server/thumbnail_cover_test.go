package server_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/server"
)

func TestHandleThumbnailServesSiblingCover(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "books.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}

	bookDir := filepath.Join(tempDir, "Alice", "Test Book (1)")
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bookPath := filepath.Join(bookDir, "Test Book - Alice.epub")
	if err := os.WriteFile(bookPath, []byte("dummy epub"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Not a real JPEG: the handler streams sidecar covers as-is, and the
	// test only asserts that bytes and content type come from cover.jpg.
	coverBytes := []byte("cover-image-bytes")
	if err := os.WriteFile(filepath.Join(bookDir, "cover.jpg"), coverBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, title, media_type, time_deleted) VALUES (?, 'Test Book', 'text', 0)`,
		bookPath,
	); err != nil {
		t.Fatal(err)
	}

	cmd := &server.Server{Databases: []string{dbPath}}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/api/thumbnail?path="+url.QueryEscape(bookPath), nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d - body: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", got)
	}
	if got := w.Body.String(); got != string(coverBytes) {
		t.Errorf("body = %q, want %q", got, coverBytes)
	}
}
