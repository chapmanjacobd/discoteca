package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/testutils"
)

func TestServeCmd_HandleLs(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// 1. Add files to DB to create a directory structure
	paths := []string{
		"/home/user/music/rock/song1.mp3",
		"/home/user/music/rock/song2.mp3",
		"/home/user/music/pop/tune1.mp3",
		"/home/user/videos/movie1.mp4",
		"/home/user/xk/sync/audio/file1.mp3",
		"/home/user/xk/sync/audio/file2.mp3",
	}

	for _, p := range paths {
		fixture.CreateDummyFile(p)
	}

	dbConn := fixture.GetDB()
	if err := db.InitDB(dbConn); err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}
	for _, p := range paths {
		_, err := dbConn.Exec("INSERT INTO media (path, type, time_deleted) VALUES (?, ?, 0)", p, "audio/mpeg")
		if err != nil {
			t.Fatalf("Failed to insert path %s: %v", p, err)
		}
	}

	// Verify DB has data
	var dbCount int
	dbConn.QueryRow("SELECT COUNT(*) FROM media").Scan(&dbCount)
	if dbCount == 0 {
		t.Fatalf("DB is empty after manual insert")
	}

	cmd := setupTestServeCmd(fixture.DBPath)

	t.Run("Absolute Path - Root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/ls?path=/", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.handleLs(w, req)

		var resp []any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should show "home/"
		if len(resp) != 1 {
			// Print all paths in DB for debugging if it fails
			rows, _ := dbConn.Query("SELECT path FROM media")
			var allPaths []string
			for rows.Next() {
				var p string
				rows.Scan(&p)
				allPaths = append(allPaths, p)
			}
			t.Errorf("Expected 1 result for root, got %d. Paths in DB: %v", len(resp), allPaths)
			return
		}
		entry := resp[0].(map[string]any)
		if entry["name"] != "home" || entry["is_dir"] != true {
			t.Errorf("Unexpected root entry: %+v", entry)
		}
	})

	t.Run("Absolute Path - Directory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/ls?path=/home/user/music/", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.handleLs(w, req)

		var resp []any
		json.NewDecoder(w.Body).Decode(&resp)

		// Should show "pop/" and "rock/"
		if len(resp) != 2 {
			t.Errorf("Expected 2 results, got %d", len(resp))
		}
	})

	t.Run("Partial Search - ./home/", func(t *testing.T) {
		// Searching for ./home/ should suggest contents of /home/
		req := httptest.NewRequest(http.MethodGet, "/api/ls?path=./home/", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.handleLs(w, req)

		var resp []any
		json.NewDecoder(w.Body).Decode(&resp)

		// Should show "user/"
		found := false
		for _, r := range resp {
			entry := r.(map[string]any)
			if entry["name"] == "user" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'user' folder to be suggested for ./home/")
		}
	})

	t.Run("Partial Search - Deep context - ./home/user/xk/sync/au", func(t *testing.T) {
		// This should suggest 'audio/' because it's under 'sync/' and contains 'au'
		req := httptest.NewRequest(http.MethodGet, "/api/ls?path=./home/user/xk/sync/au", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.handleLs(w, req)

		var resp []any
		json.NewDecoder(w.Body).Decode(&resp)

		found := false
		for _, r := range resp {
			entry := r.(map[string]any)
			if entry["name"] == "audio" {
				found = true
				if entry["path"] != "/home/user/xk/sync/audio/" {
					t.Errorf("Unexpected path for audio: %v", entry["path"])
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected 'audio' to be suggested for ./home/user/xk/sync/au")
		}
	})

	t.Run("Frequency Ranking", func(t *testing.T) {
		// rock has 2 songs, pop has 1. rock should be ranked higher if both match.
		// Searching for "./music/"
		req := httptest.NewRequest(http.MethodGet, "/api/ls?path=./music/", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.handleLs(w, req)

		var resp []any
		json.NewDecoder(w.Body).Decode(&resp)

		if len(resp) < 2 {
			t.Fatalf("Expected at least 2 results, got %d", len(resp))
		}

		// rock should be first because it has more matches (2)
		first := resp[0].(map[string]any)
		if first["name"] != "rock" {
			t.Errorf("Expected 'rock' to be first due to frequency, got '%v'", first["name"])
		}
	})
}
