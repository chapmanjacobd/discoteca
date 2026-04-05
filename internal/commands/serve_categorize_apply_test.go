package commands_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

func TestHandleCategorizeApply(t *testing.T) {
	tmpDB, err := os.CreateTemp(t.TempDir(), "disco_test_apply_*.db")
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

	// Insert test media
	_, err = db.Exec(`
		INSERT INTO media (path, title, media_type, size, duration) VALUES
			('/videos/rock_concert.mp4', 'Rock Concert', 'video/mp4', 1024, 120),
			('/videos/jazz_performance.mp4', 'Jazz Performance', 'video/mp4', 2048, 180),
			('/videos/uncategorized.mp4', 'Random Video', 'video/mp4', 512, 90);
	`)
	if err != nil {
		t.Fatalf("Failed to insert media: %v", err)
	}

	// Insert keywords that should match
	_, err = db.Exec(`
		INSERT INTO custom_keywords (category, keyword) VALUES
			('Genre', 'Rock'),
			('Genre', 'Jazz'),
			('Type', 'Concert'),
			('Type', 'Performance');
	`)
	if err != nil {
		t.Fatalf("Failed to insert keywords: %v", err)
	}

	db.Close()

	cmd := &commands.ServeCmd{
		Databases: []string{tmpDB.Name()},
		ReadOnly:  false,
	}

	t.Run("ApplyCategorization updates media categories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/apply", nil)
		w := httptest.NewRecorder()

		cmd.HandleCategorizeApply(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Check response structure
		var resp struct {
			Count int `json:"count"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Count == 0 {
			t.Fatal("Expected at least 1 media to be categorized")
		}

		// Verify categories were applied
		db, err := sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen db: %v", err)
		}
		defer db.Close()

		// Check rock_concert has Genre and Type categories
		var categories sql.NullString
		err = db.QueryRow("SELECT categories FROM media WHERE path = '/videos/rock_concert.mp4'").Scan(&categories)
		if err != nil {
			t.Fatalf("Failed to query categories: %v", err)
		}

		if !categories.Valid || categories.String == "" {
			t.Fatal("Expected rock_concert to have categories")
		}

		// Verify category format: should contain both Genre and Type
		if !strings.Contains(categories.String, ";Genre;") {
			t.Errorf("Expected categories to contain ';Genre;', got '%s'", categories.String)
		}
		if !strings.Contains(categories.String, ";Type;") {
			t.Errorf("Expected categories to contain ';Type;', got '%s'", categories.String)
		}

		// Check jazz_performance has Genre category
		err = db.QueryRow("SELECT categories FROM media WHERE path = '/videos/jazz_performance.mp4'").Scan(&categories)
		if err != nil {
			t.Fatalf("Failed to query categories: %v", err)
		}

		if !categories.Valid || categories.String == "" {
			t.Error("Expected jazz_performance to have categories")
		} else if !strings.Contains(categories.String, ";Genre;") {
			t.Errorf("Expected jazz_performance categories to contain ';Genre;', got '%s'", categories.String)
		}

		// Check uncategorized file remains uncategorized (no keyword matches)
		err = db.QueryRow("SELECT categories FROM media WHERE path = '/videos/uncategorized.mp4'").Scan(&categories)
		if err != nil {
			t.Fatalf("Failed to query categories: %v", err)
		}

		if categories.Valid && categories.String != "" {
			t.Errorf("Expected uncategorized to remain empty, got '%s'", categories.String)
		}
	})

	t.Run("ApplyCategorization with no matches returns count 0", func(t *testing.T) {
		// Create database with media that won't match any keywords
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
			INSERT INTO media (path, title, media_type, size, duration) VALUES ('/videos/abc123.mp4', 'XYZ', 'video/mp4', 1024, 120);
		`)
		if err != nil {
			t.Fatalf("Failed to setup db: %v", err)
		}
		db.Close()

		emptyCmd := &commands.ServeCmd{
			Databases: []string{emptyDB.Name()},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/categorize/apply", nil)
		w := httptest.NewRecorder()

		emptyCmd.HandleCategorizeApply(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Count int `json:"count"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Count != 0 {
			t.Errorf("Expected count 0, got %d", resp.Count)
		}
	})
}
