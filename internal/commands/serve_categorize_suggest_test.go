package commands

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestHandleCategorizeSuggest(t *testing.T) {
	t.Parallel()
	tmpDB, err := os.CreateTemp(t.TempDir(), "disco_test_suggest_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	defer os.Remove(tmpDB.Name())
	tmpDB.Close()

	db, err := sql.Open("sqlite3", tmpDB.Name())
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE media (
			path TEXT PRIMARY KEY,
			path_tokenized TEXT,
			title TEXT,
			media_type TEXT,
			size INTEGER,
			duration INTEGER,
			time_deleted INTEGER DEFAULT 0,
			categories TEXT,
			time_created INTEGER,
			time_modified INTEGER,
			time_last_played INTEGER,
			time_first_played INTEGER,
			play_count INTEGER,
			playhead INTEGER,
			width INTEGER,
			height INTEGER,
			fps REAL,
			video_codecs TEXT,
			audio_codecs TEXT,
			subtitle_codecs TEXT,
			video_count INTEGER,
			audio_count INTEGER,
			subtitle_count INTEGER,
			album TEXT,
			artist TEXT,
			genre TEXT,
			mood TEXT,
			bpm INTEGER,
			"key" TEXT,
			decade TEXT,
			city TEXT,
			country TEXT,
			description TEXT,
			language TEXT,
			score REAL,
			webpath TEXT,
			uploader TEXT,
			time_uploaded INTEGER,
			time_downloaded INTEGER,
			view_count INTEGER,
			num_comments INTEGER,
			favorite_count INTEGER,
			upvote_ratio REAL,
			latitude REAL,
			longitude REAL
		);
		CREATE TABLE custom_keywords (
			category TEXT,
			keyword TEXT,
			UNIQUE(category, keyword)
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Insert test media without categories (uncategorized)
	// Include repeated words to test frequency counting
	_, err = db.Exec(`
		INSERT INTO media (path, title, media_type, size, duration) VALUES
			('/videos/rock_concert.mp4', 'Rock Concert', 'video/mp4', 1024, 120),
			('/videos/jazz_performance.mp4', 'Jazz Performance', 'video/mp4', 2048, 180),
			('/videos/rock_live.mp4', 'Rock Live', 'video/mp4', 512, 90),
			('/videos/pop_music.mp4', 'Pop Music Video', 'video/mp4', 1500, 200),
			('/videos/jazz_club.mp4', 'Jazz Club', 'video/mp4', 800, 150),
			('/videos/pop_concert.mp4', 'Pop Concert', 'video/mp4', 900, 160),
			('/videos/live_show.mp4', 'Live Show', 'video/mp4', 700, 140);
	`)
	if err != nil {
		t.Fatalf("Failed to insert media: %v", err)
	}

	// Insert some existing keywords (so they won't be suggested)
	_, err = db.Exec(`
		INSERT INTO custom_keywords (category, keyword) VALUES
			('Genre', 'Rock');
	`)
	if err != nil {
		t.Fatalf("Failed to insert keywords: %v", err)
	}

	db.Close()

	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
	}

	t.Run("SuggestKeywords returns uncategorized word frequencies", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/categorize/suggest", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeSuggest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var results []struct {
			Word  string `json:"word"`
			Count int    `json:"count"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have some suggestions
		if len(results) == 0 {
			t.Fatal("Expected some keyword suggestions")
		}

		// Validate response structure
		for i, r := range results {
			if r.Word == "" {
				t.Errorf("Result[%d]: Word should not be empty", i)
			}
			if r.Count <= 0 {
				t.Errorf("Result[%d]: Count should be positive, got %d", i, r.Count)
			}
		}

		// "Rock" should not be in suggestions (already categorized)
		for _, r := range results {
			if r.Word == "Rock" {
				t.Error("Should not suggest 'Rock' as it's already a keyword")
			}
		}

		// Expected words from titles: jazz, performance, pop, concert, live, show
		// Verify we get expected suggestions (case-insensitive)
		foundWords := make(map[string]bool)
		for _, r := range results {
			foundWords[strings.ToLower(r.Word)] = true
		}

		// At least some of these common words should appear
		expectedWords := []string{"jazz", "performance", "pop", "concert", "live", "show"}
		foundExpected := 0
		for _, expected := range expectedWords {
			if foundWords[expected] {
				foundExpected++
			}
		}
		if foundExpected < 1 {
			t.Errorf("Expected at least 1 common word from titles, found %d: %v", foundExpected, foundWords)
		}
	})

	t.Run("SuggestKeywords with empty database returns empty array", func(t *testing.T) {
		emptyDB, err := os.CreateTemp(t.TempDir(), "disco_test_empty_*.db")
		if err != nil {
			t.Fatalf("Failed to create temp db: %v", err)
		}
		defer os.Remove(emptyDB.Name())
		emptyDB.Close()

		db, err := sql.Open("sqlite3", emptyDB.Name())
		if err != nil {
			t.Fatalf("Failed to open db: %v", err)
		}
		_, err = db.Exec(`
			CREATE TABLE media (
				path TEXT PRIMARY KEY,
				path_tokenized TEXT,
				title TEXT,
				media_type TEXT,
				size INTEGER,
				duration INTEGER,
				time_deleted INTEGER DEFAULT 0,
				categories TEXT,
				time_created INTEGER,
				time_modified INTEGER,
				time_last_played INTEGER,
				time_first_played INTEGER,
				play_count INTEGER,
				playhead INTEGER,
				width INTEGER,
				height INTEGER,
				fps REAL,
				video_codecs TEXT,
				audio_codecs TEXT,
				subtitle_codecs TEXT,
				video_count INTEGER,
				audio_count INTEGER,
				subtitle_count INTEGER,
				album TEXT,
				artist TEXT,
				genre TEXT,
				mood TEXT,
				bpm INTEGER,
				"key" TEXT,
				decade TEXT,
				city TEXT,
				country TEXT,
				description TEXT,
				language TEXT,
				score REAL,
				webpath TEXT,
				uploader TEXT,
				time_uploaded INTEGER,
				time_downloaded INTEGER,
				view_count INTEGER,
				num_comments INTEGER,
				favorite_count INTEGER,
				upvote_ratio REAL,
				latitude REAL,
				longitude REAL
			);
			CREATE TABLE custom_keywords (category TEXT, keyword TEXT, UNIQUE(category, keyword));
		`)
		if err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}
		db.Close()

		emptyCmd := &ServeCmd{
			Databases: []string{emptyDB.Name()},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/categorize/suggest", nil)
		w := httptest.NewRecorder()

		emptyCmd.handleCategorizeSuggest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var results []struct {
			Word  string `json:"word"`
			Count int    `json:"count"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 suggestions, got %d", len(results))
		}
	})
}
