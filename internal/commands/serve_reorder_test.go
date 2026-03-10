package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func TestServeReorder_Playlist(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_reorder.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	// Setup: 1 playlist with 3 items
	res := sqlDB.QueryRow(`INSERT INTO playlists (path, title) VALUES ('/plist', 'Test Playlist') RETURNING id`)
	var pid int64
	res.Scan(&pid)

	for i := 1; i <= 3; i++ {
		path := fmt.Sprintf("/media%d.mp4", i)
		sqlDB.Exec(`INSERT INTO media (path, type, time_deleted) VALUES (?, 'video', 0)`, path)
		sqlDB.Exec(`INSERT INTO playlist_items (playlist_id, media_path, track_number) VALUES (?, ?, ?)`, pid, path, i)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	// Reorder: Move item 3 to track 1
	reorderReq := map[string]any{
		"playlist_title": "Test Playlist",
		"media_path":     "/media3.mp4",
		"new_index":      0, // 0-based index means track_number 1
	}
	body, _ := json.Marshal(reorderReq)
	req := httptest.NewRequest("POST", "/api/playlists/reorder", bytes.NewBuffer(body))
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify track numbers in DB
	sqlDB, _ = sql.Open("sqlite3", dbPath)
	rows, _ := sqlDB.Query(`SELECT media_path, track_number FROM playlist_items WHERE playlist_id = ? ORDER BY track_number`, pid)
	defer rows.Close()

	expected := map[string]int64{
		"/media3.mp4": 1,
		"/media1.mp4": 2,
		"/media2.mp4": 3,
	}

	for rows.Next() {
		var path string
		var track int64
		rows.Scan(&path, &track)
		if expected[path] != track {
			t.Errorf("Expected path %s to have track %d, got %d", path, expected[path], track)
		}
	}
}

func TestServeReorder_Security(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_reorder_sec.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	t.Run("UnauthorizedDatabase", func(t *testing.T) {
		// handlePlaylistReorder doesn't take a 'db' param, it checks ALL configured DBs
		// but it requires a playlist title
		reorderReq := map[string]any{
			"playlist_title": "Nonexistent",
			"media_path":     "/some.mp4",
			"new_index":      0,
		}
		body, _ := json.Marshal(reorderReq)
		req := httptest.NewRequest("POST", "/api/playlists/reorder", bytes.NewBuffer(body))
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
		db.InitDB(db2)
		// Add a playlist to the second DB
		db2.Exec(`INSERT INTO playlists (path, title) VALUES ('/plist2', 'Other Playlist')`)
		db2.Close()

		cmd2 := &ServeCmd{
			Databases: []string{dbPath, dbPath2},
		}
		mux2 := cmd2.Mux()

		// Request for second DB playlist should work
		reorderReq := map[string]any{
			"playlist_title": "Other Playlist",
			"media_path":     "/some.mp4",
			"new_index":      0,
		}
		body, _ := json.Marshal(reorderReq)
		req := httptest.NewRequest("POST", "/api/playlists/reorder", bytes.NewBuffer(body))
		req.Header.Set("X-Disco-Token", cmd2.APIToken)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		// 404 because item not in playlist, but not 401/403/400
		if w.Code == http.StatusBadRequest || w.Code == http.StatusUnauthorized {
			t.Errorf("Expected 404 for item not in playlist, got %d", w.Code)
		}
	})
}
