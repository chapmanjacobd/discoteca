package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/testutils"
)

func TestServeCmd_ExtendedHandlers(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	db := fixture.GetDB()
	InitDB(db)

	// Insert comprehensive test data for all sub-tests
	_, err := db.Exec(`
		INSERT INTO media (path, title, type, size, duration, genre, categories, time_last_played, time_deleted)
		VALUES 
		('/media/video1.mp4', 'Movie 1', 'video', 1000000, 3600, 'Action', ';action;', 1708700000, 0),
		('/media/video2.mp4', 'Movie 2', 'video', 1000050, 3605, 'Comedy', ';action;', 1708800000, 0),
		('/media/music/audio1.mp3', 'Song 1', 'audio', 5000000, 300, 'Rock', ';music;', 1708900000, 0),
		('/media/music/audio2.mp3', 'Song 2', 'audio', 5000000, 300, 'Jazz', ';music;', 0, 0),
		('/media/other/doc.pdf', 'Document', 'application/pdf', 5000, 0, 'Edu', ';edu;', 0, 0),
		('/media/deleted.mp4', 'Deleted', 'video', 1000, 10, 'Horror', '', 0, 123456789)
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert captions for search test
	_, err = db.Exec(`
		INSERT INTO captions (media_path, time, text)
		VALUES 
		('/media/video1.mp4', 10.5, 'I will be back'),
		('/media/video1.mp4', 20.0, 'Hello world')
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Manual FTS rebuild since triggers might not work across different connections/transfers in some sqlite versions or tests
	_, err = db.Exec("INSERT INTO captions_fts(captions_fts) VALUES('rebuild')")
	if err != nil {
		t.Logf("FTS rebuild failed (likely not using FTS5): %v", err)
	}

	db.Close()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
		Trashcan:  true,
	}
	cmd.HideDeleted = true
	handler := cmd.Mux()

	t.Run("HandleDatabases", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp struct {
			Databases []string `json:"databases"`
			Trashcan  bool     `json:"trashcan"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Databases) != 1 || resp.Databases[0] != fixture.DBPath {
			t.Errorf("Unexpected databases in response: %v", resp.Databases)
		}
		if !resp.Trashcan {
			t.Error("Expected Trashcan to be true")
		}
	})

	t.Run("HandleGenres", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/genres", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp []struct {
			Genre string `json:"genre"`
			Count int    `json:"count"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		// Action, Comedy, Rock, Jazz, Edu
		if len(resp) != 5 {
			t.Errorf("Expected 5 genres, got %d", len(resp))
		}
	})

	t.Run("HandleQuery_Offset", func(t *testing.T) {
		// Natural sort by path (asc):
		// /media/music/audio1.mp3 (0)
		// /media/music/audio2.mp3 (1)
		// /media/other/doc.pdf (2)
		// /media/video1.mp4 (3)
		// /media/video2.mp4 (4)

		req := httptest.NewRequest(http.MethodGet, "/api/query?limit=2&offset=3&sort=path", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp []struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if len(resp) != 2 {
			t.Errorf("Expected 2 results, got %d", len(resp))
		}
		if resp[0].Path != "/media/video1.mp4" {
			t.Errorf("Expected /media/video1.mp4 at offset 3, got %s", resp[0].Path)
		}
	})

	t.Run("HandlePlaylists", func(t *testing.T) {
		// POST create playlist
		payload := `{"title": "My Playlist"}`
		req := httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(payload))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", w.Code)
		}

		// GET playlists
		req = httptest.NewRequest(http.MethodGet, "/api/playlists", nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var listResp []string
		if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
			t.Fatal(err)
		}
		found := slices.Contains(listResp, "My Playlist")
		if !found {
			t.Errorf("Playlist 'My Playlist' not found in %v", listResp)
		}

		// POST add item to playlist
		addItemPayload := `{"playlist_title": "My Playlist", "media_path": "/media/video1.mp4"}`
		req = httptest.NewRequest(http.MethodPost, "/api/playlists/items", strings.NewReader(addItemPayload))
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 adding item, got %d", w.Code)
		}

		// GET playlist items
		req = httptest.NewRequest(http.MethodGet, "/api/playlists/items?title=My%20Playlist", nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		var itemsResp []any
		if err := json.NewDecoder(w.Body).Decode(&itemsResp); err != nil {
			t.Fatal(err)
		}
		if len(itemsResp) != 1 {
			t.Errorf("Expected 1 item in playlist, got %d", len(itemsResp))
		}

		// DELETE playlist item
		deleteItemPayload := `{"playlist_title": "My Playlist", "media_path": "/media/video1.mp4"}`
		req = httptest.NewRequest(http.MethodDelete, "/api/playlists/items", strings.NewReader(deleteItemPayload))
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 deleting item, got %d", w.Code)
		}

		// DELETE playlist
		req = httptest.NewRequest(http.MethodDelete, "/api/playlists?title=My%20Playlist", nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 deleting playlist, got %d", w.Code)
		}
	})

	t.Run("HandleTrash", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/trash", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp []any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if len(resp) != 1 {
			t.Errorf("Expected 1 deleted item, got %d", len(resp))
		}
	})

	t.Run("HandleHLSPlaylist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/hls/playlist?path=/media/video1.mp4", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "#EXTM3U") {
			t.Error("Response body did not contain #EXTM3U")
		}
	})

	t.Run("HandleEmptyBin", func(t *testing.T) {
		// Create a file, mark it deleted, then empty bin
		dummyPath := fixture.CreateDummyFile("to_be_permanently_deleted.mp4")
		dbConn := fixture.GetDB()
		dbConn.Exec("INSERT INTO media (path, type, time_deleted) VALUES (?, 'video', 12345)", dummyPath)
		dbConn.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/empty-bin", strings.NewReader("{}"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Verify file is gone from disk
		if _, err := os.Stat(dummyPath); !os.IsNotExist(err) {
			t.Error("Expected file to be deleted from disk")
		}

		// Verify row is gone from DB
		dbConn = fixture.GetDB()
		defer dbConn.Close()
		var count int
		dbConn.QueryRow("SELECT count(*) FROM media WHERE path = ?", dummyPath).Scan(&count)
		if count != 0 {
			t.Error("Expected row to be deleted from DB")
		}
	})

	t.Run("HandleDU", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/du?path=/media", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp []models.FolderStats
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if len(resp) == 0 {
			t.Error("Expected DU results, got none")
		}

		// We expect:
		// - Direct files: /media/video1.mp4, /media/video2.mp4 (shown as individual items)
		// - Folders: /media/music (2 files), /media/other (1 file)

		foundMusic := false
		foundOther := false
		foundVideo1 := false
		foundVideo2 := false

		for _, g := range resp {
			if g.Path == "/media/music" {
				foundMusic = true
				if g.Count != 2 {
					t.Errorf("Expected 2 items in /media/music, got %d", g.Count)
				}
			}
			if g.Path == "/media/other" {
				foundOther = true
				if g.Count != 1 {
					t.Errorf("Expected 1 item in /media/other, got %d", g.Count)
				}
			}
			if g.Path == "/media/video1.mp4" {
				foundVideo1 = true
			}
			if g.Path == "/media/video2.mp4" {
				foundVideo2 = true
			}
		}

		if !foundMusic {
			t.Error("Expected to find /media/music folder")
		}
		if !foundOther {
			t.Error("Expected to find /media/other folder")
		}
		if !foundVideo1 {
			t.Error("Expected to find /media/video1.mp4 file")
		}
		if !foundVideo2 {
			t.Error("Expected to find /media/video2.mp4 file")
		}
	})

	t.Run("HandleEpisodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/episodes", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp []models.FolderStats
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		// We expect groups for:
		// /media (video1, video2, deleted)
		// /media/music (audio1, audio2)
		// /media/other (doc.pdf)

		foundMusic := false
		foundRoot := false

		for _, g := range resp {
			if g.Path == "/media/music" {
				foundMusic = true
				if g.Count != 2 {
					t.Errorf("Expected 2 items in /media/music, got %d", g.Count)
				}
			}
			if g.Path == "/media" {
				foundRoot = true
				// deleted.mp4 is hidden by default in this test setup (HideDeleted=true)
				// video1.mp4, video2.mp4
				if g.Count != 2 {
					t.Errorf("Expected 2 items in /media, got %d", g.Count)
				}
			}
		}

		if !foundMusic {
			t.Error("Expected to find /media/music group")
		}
		if !foundRoot {
			t.Error("Expected to find /media group")
		}
	})

	t.Run("HandleRandomClip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/random-clip", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp struct {
			models.MediaWithDB
			Start int `json:"start"`
			End   int `json:"end"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if resp.Path == "" {
			t.Error("Expected a media path in random clip")
		}
	})

	t.Run("HandleCategorizeSuggest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/categorize/suggest", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp []any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("HandleCategorizeApply", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/apply", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp struct {
			Count int `json:"count"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("HandleQueryWithCaptions", func(t *testing.T) {
		// Skip if FTS5 is not available
		dbConn := fixture.GetDB()
		var name string
		err := dbConn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='captions_fts'").Scan(&name)
		dbConn.Close()
		if err != nil {
			t.Skip("FTS5 not available, skipping search captions tests")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/query?search=back&view=captions", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp []models.MediaWithDB
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		foundCaption := false
		for _, m := range resp {
			if strings.Contains(m.CaptionText, "back") {
				foundCaption = true
				if m.CaptionTime != 10.5 {
					t.Errorf("Expected caption time 10.5, got %f", m.CaptionTime)
				}
			}
		}
		if !foundCaption {
			t.Error("Expected to find media via caption search")
		}
	})
}
