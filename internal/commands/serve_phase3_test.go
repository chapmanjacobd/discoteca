package commands_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func newServePhase3DB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "serve-phase3.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitDB(context.Background(), conn); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return dbPath, conn
}

func TestQueryGroupedTotalCountMatchesItems(t *testing.T) {
	dbPath, conn := newServePhase3DB(t)
	defer conn.Close()
	if _, err := conn.Exec(`INSERT INTO media (path) VALUES
		('/show/one/e1.mkv'), ('/show/one/e2.mkv'),
		('/show/two/e1.mkv'), ('/show/two/e2.mkv'), ('/show/two/e3.mkv')`); err != nil {
		t.Fatal(err)
	}

	cmd := SetupTestServeCmd(dbPath)
	defer cmd.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/query?group_by=parent&all=true", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	cmd.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var items []models.MediaWithDB
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	total, err := strconv.Atoi(w.Header().Get("X-Total-Count"))
	if err != nil {
		t.Fatalf("missing total header: %v", err)
	}
	if total != len(items) || total != 2 {
		t.Errorf("X-Total-Count = %d, items = %d, want 2", total, len(items))
	}
}

func TestCaptionsViewTotalCountIsPrePagination(t *testing.T) {
	dbPath, conn := newServePhase3DB(t)
	defer conn.Close()
	if _, err := conn.Exec(`INSERT INTO media (path, title) VALUES
		('/video/one.mp4', 'One'), ('/video/two.mp4', 'Two')`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO captions (media_path, time, text) VALUES
		('/video/one.mp4', 1.0, 'alpha one'),
		('/video/one.mp4', 2.0, 'alpha two'),
		('/video/two.mp4', 3.0, 'alpha three')`); err != nil {
		t.Fatal(err)
	}

	cmd := SetupTestServeCmd(dbPath)
	defer cmd.Close()

	do := func(url string) ([]models.MediaWithDB, int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.Mux().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		var items []models.MediaWithDB
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		total, err := strconv.Atoi(w.Header().Get("X-Total-Count"))
		if err != nil {
			t.Fatalf("missing total header: %v", err)
		}
		return items, total
	}

	allItems, allTotal := do("/api/query?view=captions&search=alpha&all=true")
	pageItems, pageTotal := do("/api/query?view=captions&search=alpha&limit=1")

	if allTotal != len(allItems) || allTotal == 0 {
		t.Errorf("all view total = %d, items = %d", allTotal, len(allItems))
	}
	if len(pageItems) != 1 {
		t.Errorf("page size = %d, want 1", len(pageItems))
	}
	if pageTotal != allTotal {
		t.Errorf("page X-Total-Count = %d, want full count %d", pageTotal, allTotal)
	}
}

func TestDUTotalCountIsPrePagination(t *testing.T) {
	dbPath, conn := newServePhase3DB(t)
	defer conn.Close()
	if _, err := conn.Exec(`INSERT INTO media (path, size) VALUES
		('/videos/movie1.mp4', 1000),
		('/videos/movie2.mp4', 1000),
		('/audio/song1.mp3', 1000)`); err != nil {
		t.Fatal(err)
	}

	cmd := SetupTestServeCmd(dbPath)
	defer cmd.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/du?path=&limit=1", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	cmd.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.TotalCount < 2 {
		t.Errorf("total = %d, want at least 2", resp.TotalCount)
	}
	if len(resp.Folders) > 1 || len(resp.Files) > 1 {
		t.Errorf("pagination not applied: %d folders, %d files", len(resp.Folders), len(resp.Files))
	}
}
