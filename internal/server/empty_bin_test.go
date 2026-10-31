package server_test

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

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
)

func setupEmptyBinEnv(t *testing.T) (*duTestEnv, string) {
	t.Helper()

	logging.SetupLogging(0)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "empty_bin.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err = db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}

	_, err = sqlDB.Exec(`INSERT INTO media (path, media_type, size, time_deleted) VALUES
		('/trash/a.mp4', 'video', 100, 1),
		('/trash/b.mp4', 'video', 200, 1),
		('/keep/c.mp4', 'video', 300, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	cmd := SetupTestServer(dbPath)
	return &duTestEnv{cmd: cmd, mux: cmd.Mux()}, dbPath
}

func postEmptyBin(t *testing.T, env *duTestEnv, paths []string) string {
	t.Helper()

	body, err := json.Marshal(map[string][]string{"paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/empty-bin", bytes.NewReader(body))
	req.Header.Set("X-Disco-Token", env.cmd.APIToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}
	return w.Body.String()
}

func countRemaining(t *testing.T, dbPath string) map[string]int {
	t.Helper()

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	rows, err := sqlDB.Query("SELECT path FROM media")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	remaining := map[string]int{}
	for rows.Next() {
		var path string
		if err = rows.Scan(&path); err != nil {
			t.Fatal(err)
		}
		remaining[path]++
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	return remaining
}

func TestHandleEmptyBin_EmptyPathsEmptiesWholeBin(t *testing.T) {
	env, dbPath := setupEmptyBinEnv(t)
	defer env.cmd.Close()

	postEmptyBin(t, env, []string{})

	remaining := countRemaining(t, dbPath)
	if _, ok := remaining["/trash/a.mp4"]; ok {
		t.Error("expected /trash/a.mp4 to be removed")
	}
	if _, ok := remaining["/trash/b.mp4"]; ok {
		t.Error("expected /trash/b.mp4 to be removed")
	}
	if remaining["/keep/c.mp4"] != 1 {
		t.Error("expected non-trash media to remain")
	}
}

func TestHandleEmptyBin_ExplicitPathsRemoveOnlyThoseItems(t *testing.T) {
	env, dbPath := setupEmptyBinEnv(t)
	defer env.cmd.Close()

	postEmptyBin(t, env, []string{"/trash/a.mp4"})

	remaining := countRemaining(t, dbPath)
	if _, ok := remaining["/trash/a.mp4"]; ok {
		t.Error("expected /trash/a.mp4 to be removed")
	}
	if remaining["/trash/b.mp4"] != 1 {
		t.Error("expected /trash/b.mp4 to remain")
	}
	if remaining["/keep/c.mp4"] != 1 {
		t.Error("expected non-trash media to remain")
	}
}
