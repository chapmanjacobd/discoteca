package server_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func setupDUSortEnv(t *testing.T) (*duTestEnv, string) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "du_sort.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err = db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}

	// Five sibling folders with increasing total size.
	_, err = sqlDB.Exec(`INSERT INTO media (path, title, media_type, size, duration, time_deleted) VALUES
		('/folders/one/a.mp4', 'A', 'video', 100, 10, 0),
		('/folders/two/b.mp4', 'B', 'video', 200, 20, 0),
		('/folders/three/c.mp4', 'C', 'video', 300, 30, 0),
		('/folders/four/d.mp4', 'D', 'video', 400, 40, 0),
		('/folders/five/e.mp4', 'E', 'video', 500, 50, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	cmd := SetupTestServer(dbPath)
	env := &duTestEnv{cmd: cmd, mux: cmd.Mux()}
	return env, dbPath
}

func getDU(t *testing.T, env *duTestEnv, query string) models.DUResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/du"+query, nil)
	req.Header.Set("X-Disco-Token", env.cmd.APIToken)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
	}

	var resp models.DUResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	return resp
}

func folderSizes(resp models.DUResponse) []int64 {
	sizes := make([]int64, 0, len(resp.Folders))
	for _, folder := range resp.Folders {
		sizes = append(sizes, folder.TotalSize)
	}
	return sizes
}

func TestHandleDU_AllDisablesPagination(t *testing.T) {
	env, _ := setupDUSortEnv(t)
	defer env.cmd.Close()

	limited := getDU(t, env, "?path=/folders&limit=2")
	if len(limited.Folders) != 2 {
		t.Fatalf("Expected 2 folders with limit=2, got %d", len(limited.Folders))
	}
	if limited.TotalCount != 5 {
		t.Fatalf("Expected total_count=5, got %d", limited.TotalCount)
	}

	all := getDU(t, env, "?path=/folders&all=true")
	if len(all.Folders) != 5 {
		t.Fatalf("Expected all 5 folders, got %d", len(all.Folders))
	}
	if all.TotalCount != 5 {
		t.Fatalf("Expected total_count=5, got %d", all.TotalCount)
	}
}

func TestHandleDU_SortAndReverse(t *testing.T) {
	env, _ := setupDUSortEnv(t)
	defer env.cmd.Close()

	ascending := getDU(t, env, "?path=/folders&sort=size")
	gotAsc := folderSizes(ascending)
	wantAsc := []int64{100, 200, 300, 400, 500}
	for i := range wantAsc {
		if gotAsc[i] != wantAsc[i] {
			t.Fatalf("sort=size ascending = %v, want %v", gotAsc, wantAsc)
		}
	}

	descending := getDU(t, env, "?path=/folders&sort=size&reverse=true")
	gotDesc := folderSizes(descending)
	wantDesc := []int64{500, 400, 300, 200, 100}
	for i := range wantDesc {
		if gotDesc[i] != wantDesc[i] {
			t.Fatalf("sort=size reverse = %v, want %v", gotDesc, wantDesc)
		}
	}
}

func TestHandleDU_DefaultSortHonorsReverse(t *testing.T) {
	env, _ := setupDUSortEnv(t)
	defer env.cmd.Close()

	forward := folderSizes(getDU(t, env, "?path=/folders"))
	reverse := folderSizes(getDU(t, env, "?path=/folders&reverse=true"))

	if len(forward) != len(reverse) {
		t.Fatalf("Forward/reverse folder counts differ: %d vs %d", len(forward), len(reverse))
	}
	for i := range forward {
		if forward[i] != reverse[len(reverse)-1-i] {
			t.Fatalf("Default reverse is not the reverse of the default order: %v vs %v", forward, reverse)
		}
	}
}
