package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestHandleCategorizeKeywords(t *testing.T) {
	// Create temporary test database
	tmpDB, err := os.CreateTemp("", "disco_test_cat_*.db")
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

	// Create custom_keywords table
	_, err = db.Exec(`
		CREATE TABLE custom_keywords (
			category TEXT,
			keyword TEXT,
			UNIQUE(category, keyword)
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO custom_keywords (category, keyword) VALUES
			('Genre', 'Rock'),
			('Genre', 'Jazz'),
			('Mood', 'Happy'),
			('Mood', 'Sad');
	`)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	db.Close()

	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
	}

	t.Run("GetKeywords returns all categories and keywords", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/categorize/keywords", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeKeywords(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var results []struct {
			Category string   `json:"category"`
			Keywords []string `json:"keywords"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 categories, got %d", len(results))
		}

		// Find Genre category
		var genreCat *struct {
			Category string   `json:"category"`
			Keywords []string `json:"keywords"`
		}
		for i := range results {
			if results[i].Category == "Genre" {
				genreCat = &results[i]
				break
			}
		}

		if genreCat == nil {
			t.Fatal("Expected to find Genre category")
		}

		if len(genreCat.Keywords) != 2 {
			t.Errorf("Expected 2 Genre keywords, got %d", len(genreCat.Keywords))
		}
	})

	t.Run("GetKeywords with empty database returns empty array", func(t *testing.T) {
		// Create empty database
		emptyDB, err := os.CreateTemp("", "disco_test_empty_*.db")
		if err != nil {
			t.Fatalf("Failed to create temp db: %v", err)
		}
		defer os.Remove(emptyDB.Name())
		emptyDB.Close()

		db, err := sql.Open("sqlite3", emptyDB.Name())
		if err != nil {
			t.Fatalf("Failed to open db: %v", err)
		}
		_, err = db.Exec(`CREATE TABLE custom_keywords (category TEXT, keyword TEXT, UNIQUE(category, keyword));`)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
		db.Close()

		emptyCmd := &ServeCmd{
			Databases: []string{emptyDB.Name()},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/categorize/keywords", nil)
		w := httptest.NewRecorder()

		emptyCmd.handleCategorizeKeywords(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var results []struct {
			Category string   `json:"category"`
			Keywords []string `json:"keywords"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 categories, got %d", len(results))
		}
	})
}

func TestHandleCategorizeDefaults(t *testing.T) {
	tmpDB, err := os.CreateTemp("", "disco_test_defaults_*.db")
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

	_, err = db.Exec(`CREATE TABLE custom_keywords (category TEXT, keyword TEXT, UNIQUE(category, keyword));`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	db.Close()

	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
		ReadOnly:  false,
	}

	t.Run("AddDefaults inserts default categories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/defaults", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeDefaults(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify defaults were inserted
		db, err := sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM custom_keywords").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count keywords: %v", err)
		}

		if count == 0 {
			t.Error("Expected default categories to be inserted")
		}
	})

	t.Run("AddDefaults respects read-only mode", func(t *testing.T) {
		cmd.ReadOnly = true
		defer func() { cmd.ReadOnly = false }()

		req := httptest.NewRequest(http.MethodPost, "/api/categorize/defaults", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeDefaults(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 in read-only mode, got %d", w.Code)
		}
	})

	t.Run("AddDefaults rejects non-POST method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/categorize/defaults", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeDefaults(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for GET, got %d", w.Code)
		}
	})
}

func TestHandleCategorizeDeleteCategory(t *testing.T) {
	tmpDB, err := os.CreateTemp("", "disco_test_delete_*.db")
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

	_, err = db.Exec(`CREATE TABLE custom_keywords (category TEXT, keyword TEXT, UNIQUE(category, keyword));`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO custom_keywords (category, keyword) VALUES
			('Genre', 'Rock'),
			('Genre', 'Jazz'),
			('Mood', 'Happy');
	`)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	db.Close()

	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
		ReadOnly:  false,
	}

	t.Run("DeleteCategory removes all keywords in category", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/categorize/category?category=Genre", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeDeleteCategory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify category was deleted
		db, err := sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM custom_keywords WHERE category = 'Genre'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count keywords: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 Genre keywords, got %d", count)
		}

		// Verify other categories remain
		err = db.QueryRow("SELECT COUNT(*) FROM custom_keywords WHERE category = 'Mood'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count keywords: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 Mood keyword, got %d", count)
		}
	})

	t.Run("DeleteCategory requires category parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/categorize/category", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeDeleteCategory(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("DeleteCategory respects read-only mode", func(t *testing.T) {
		cmd.ReadOnly = true
		defer func() { cmd.ReadOnly = false }()

		req := httptest.NewRequest(http.MethodDelete, "/api/categorize/category?category=Genre", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeDeleteCategory(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 in read-only mode, got %d", w.Code)
		}
	})

	t.Run("DeleteCategory rejects non-DELETE method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/category?category=Genre", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeDeleteCategory(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for POST, got %d", w.Code)
		}
	})
}

func TestHandleCategorizeKeyword(t *testing.T) {
	tmpDB, err := os.CreateTemp("", "disco_test_keyword_*.db")
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

	_, err = db.Exec(`CREATE TABLE custom_keywords (category TEXT, keyword TEXT, UNIQUE(category, keyword));`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	db.Close()

	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
		ReadOnly:  false,
	}

	t.Run("AddKeyword inserts new keyword", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"category": "Genre", "keyword": "Rock"}`))
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/keyword", body)
		w := httptest.NewRecorder()

		cmd.handleCategorizeKeyword(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify keyword was inserted
		db, err := sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM custom_keywords WHERE category = 'Genre' AND keyword = 'Rock'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count keywords: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 keyword, got %d", count)
		}
	})

	t.Run("AddKeyword requires category and keyword", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"category": "Genre"}`))
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/keyword", body)
		w := httptest.NewRecorder()

		cmd.handleCategorizeKeyword(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("DeleteKeyword removes keyword", func(t *testing.T) {
		// First add a keyword
		db, err := sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen db: %v", err)
		}
		_, err = db.Exec("INSERT INTO custom_keywords (category, keyword) VALUES ('Mood', 'Happy')")
		if err != nil {
			t.Fatalf("Failed to insert keyword: %v", err)
		}
		db.Close()

		body := bytes.NewReader([]byte(`{"category": "Mood", "keyword": "Happy"}`))
		req := httptest.NewRequest(http.MethodDelete, "/api/categorize/keyword", body)
		w := httptest.NewRecorder()

		cmd.handleCategorizeKeyword(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify keyword was deleted
		db, err = sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM custom_keywords WHERE category = 'Mood' AND keyword = 'Happy'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count keywords: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 keywords after delete, got %d", count)
		}
	})

	t.Run("AddKeyword respects read-only mode", func(t *testing.T) {
		cmd.ReadOnly = true
		defer func() { cmd.ReadOnly = false }()

		body := bytes.NewReader([]byte(`{"category": "Test", "keyword": "Test"}`))
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/keyword", body)
		w := httptest.NewRecorder()

		cmd.handleCategorizeKeyword(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 in read-only mode, got %d", w.Code)
		}
	})
}

func TestHandleCategorizeSuggest(t *testing.T) {
	tmpDB, err := os.CreateTemp("", "disco_test_suggest_*.db")
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
			title TEXT,
			type TEXT,
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
		INSERT INTO media (path, title, type, size, duration) VALUES
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
			t.Error("Expected some keyword suggestions")
		}

		// "Rock" should not be in suggestions (already categorized)
		for _, r := range results {
			if r.Word == "Rock" {
				t.Error("Should not suggest 'Rock' as it's already a keyword")
			}
		}
	})

	t.Run("SuggestKeywords with empty database returns empty array", func(t *testing.T) {
		emptyDB, err := os.CreateTemp("", "disco_test_empty_*.db")
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
				title TEXT,
				type TEXT,
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

func TestHandleCategorizeApply(t *testing.T) {
	tmpDB, err := os.CreateTemp("", "disco_test_apply_*.db")
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
			title TEXT,
			type TEXT,
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
		INSERT INTO media (path, title, type, size, duration) VALUES
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

	cmd := &ServeCmd{
		Databases: []string{tmpDB.Name()},
		ReadOnly:  false,
	}

	t.Run("ApplyCategorization updates media categories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/apply", nil)
		w := httptest.NewRecorder()

		cmd.handleCategorizeApply(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify categories were applied
		db, err := sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen db: %v", err)
		}
		defer db.Close()

		var categories sql.NullString
		err = db.QueryRow("SELECT categories FROM media WHERE path = '/videos/rock_concert.mp4'").Scan(&categories)
		if err != nil {
			t.Fatalf("Failed to query categories: %v", err)
		}

		if !categories.Valid || categories.String == "" {
			t.Error("Expected rock_concert to have categories")
		} else {
			t.Logf("rock_concert categories: %s", categories.String)
		}

		// Check response count
		var resp struct {
			Count int `json:"count"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Count == 0 {
			t.Error("Expected at least 1 media to be categorized")
		}
	})

	t.Run("ApplyCategorization with no matches returns count 0", func(t *testing.T) {
	// Create database with media that won't match any keywords
		emptyDB, err := os.CreateTemp("", "disco_test_empty_*.db")
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
				title TEXT,
				type TEXT,
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
			INSERT INTO media (path, title, type, size, duration) VALUES ('/videos/abc123.mp4', 'XYZ', 'video/mp4', 1024, 120);
		`)
		if err != nil {
			t.Fatalf("Failed to setup db: %v", err)
		}
		db.Close()

		emptyCmd := &ServeCmd{
			Databases: []string{emptyDB.Name()},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/categorize/apply", nil)
		w := httptest.NewRecorder()

		emptyCmd.handleCategorizeApply(w, req)

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
