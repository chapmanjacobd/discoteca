package commands_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
)

func TestServeReorder_Playlist(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_reorder.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)

	// Setup: 1 playlist with 3 items
	res := sqlDB.QueryRow(
		`INSERT INTO playlists (path, title) VALUES (?, 'Test Playlist') RETURNING id`,
		filepath.FromSlash("/plist"),
	)
	var pid int64
	res.Scan(&pid)

	for i := 1; i <= 3; i++ {
		path := filepath.FromSlash(fmt.Sprintf("/media%d.mp4", i))
		sqlDB.Exec(`INSERT INTO media (path, media_type, time_deleted) VALUES (?, 'video', 0)`, path)
		sqlDB.Exec(`INSERT INTO playlist_items (playlist_id, media_path, track_number) VALUES (?, ?, ?)`, pid, path, i)
	}
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	// Reorder: Move item 3 to track 1
	reorderReq := map[string]any{
		"playlist_title": "Test Playlist",
		"media_path":     filepath.FromSlash("/media3.mp4"),
		"new_index":      0, // 0-based index means track_number 1
	}
	body, err := json.Marshal(reorderReq)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/playlists/reorder", bytes.NewBuffer(body))
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify track numbers in DB
	sqlDB, _ = sql.Open("sqlite3", dbPath)
	defer sqlDB.Close()
	rows, _ := sqlDB.Query(
		`SELECT media_path, track_number FROM playlist_items WHERE playlist_id = ? ORDER BY track_number`,
		pid,
	)
	defer rows.Close()

	expected := map[string]int64{
		filepath.FromSlash("/media3.mp4"): 1,
		filepath.FromSlash("/media1.mp4"): 2,
		filepath.FromSlash("/media2.mp4"): 3,
	}

	for rows.Next() {
		var path string
		var track int64
		if err := rows.Scan(&path, &track); err != nil {
			t.Errorf("Scan failed: %v", err)
		}
		if expected[path] != track {
			t.Errorf("Expected path %s to have track %d, got %d", path, expected[path], track)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows error: %v", err)
	}
}

func TestServeReorder_Security(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_reorder_sec.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(context.Background(), sqlDB)
	sqlDB.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("UnauthorizedDatabase", func(t *testing.T) {
		// handlePlaylistReorder doesn't take a 'db' param, it checks ALL configured DBs
		// but it requires a playlist title
		reorderReq := map[string]any{
			"playlist_title": "Nonexistent",
			"media_path":     filepath.FromSlash("/some.mp4"),
			"new_index":      0,
		}
		body, _ := json.Marshal(reorderReq)
		req := httptest.NewRequest(http.MethodPost, "/api/playlists/reorder", bytes.NewBuffer(body))
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should be 404 because playlist not found
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for nonexistent playlist, got %d", w.Code)
		}
	})

	t.Run("MultipleDatabasesAllowed", func(t *testing.T) {
		dbPath2 := filepath.Join(tempDir, "test_reorder_sec2.db")
		db2, _ := sql.Open("sqlite3", dbPath2)
		db.InitDB(context.Background(), db2)
		// Add a playlist to the second DB
		db2.Exec(`INSERT INTO playlists (path, title) VALUES (?, 'Other Playlist')`, filepath.FromSlash("/plist2"))
		db2.Close()

		cmd2 := &commands.ServeCmd{
			Databases: []string{dbPath, dbPath2},
		}
		defer cmd2.Close()
		mux2 := cmd2.Mux()

		// Request for second DB playlist should work
		reorderReq := map[string]any{
			"playlist_title": "Other Playlist",
			"media_path":     filepath.FromSlash("/some.mp4"),
			"new_index":      0,
		}
		body, _ := json.Marshal(reorderReq)
		req := httptest.NewRequest(http.MethodPost, "/api/playlists/reorder", bytes.NewBuffer(body))
		req.Header.Set("X-Disco-Token", cmd2.APIToken)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		// 404 because item not in playlist, but not 401/403/400
		if w.Code == http.StatusBadRequest || w.Code == http.StatusUnauthorized {
			t.Errorf("Expected 404 for item not in playlist, got %d", w.Code)
		}
	})
}
