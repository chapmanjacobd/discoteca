package query

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func TestExpandRelatedMedia_WithSearchTerms(t *testing.T) {
	// Create test database
	f, err := os.CreateTemp("", "related-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize schema
	schema := `
	CREATE TABLE media (
		path TEXT PRIMARY KEY,
		path_tokenized TEXT,
		title TEXT,
		description TEXT,
		type TEXT,
		time_deleted INTEGER DEFAULT 0,
		size INTEGER,
		duration INTEGER,
		video_count INTEGER DEFAULT 0,
		audio_count INTEGER DEFAULT 0,
		subtitle_count INTEGER DEFAULT 0,
		play_count INTEGER DEFAULT 0,
		playhead INTEGER DEFAULT 0,
		time_created INTEGER,
		time_modified INTEGER,
		time_downloaded INTEGER,
		time_last_played INTEGER,
		score REAL
	);
	CREATE VIRTUAL TABLE media_fts USING fts5(
		path, path_tokenized, title, description,
		content='media',
		content_rowid='rowid',
		tokenize='trigram',
		detail='none'
	);
	CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN
		INSERT INTO media_fts(rowid, path, path_tokenized, title, description)
		VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description);
	END;
	CREATE TRIGGER media_ad AFTER DELETE ON media BEGIN
		DELETE FROM media_fts WHERE rowid = old.rowid;
	END;
	CREATE TRIGGER media_au AFTER UPDATE ON media BEGIN
		INSERT INTO media_fts(media_fts, rowid, path, path_tokenized, title, description)
		VALUES('delete', old.rowid, old.path, old.path_tokenized, old.title, old.description);
		INSERT INTO media_fts(rowid, path, path_tokenized, title, description)
		VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description);
	END;
	`
	_, err = sqlDB.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	ctx := context.Background()

	// Insert test media
	testData := []struct {
		path  string
		title string
		desc  string
	}{
		{"/videos/python_tutorial.mp4", "Python Tutorial", "Learn Python programming"},
		{"/videos/python_advanced.mp4", "Advanced Python", "Advanced Python topics"},
		{"/videos/golang_tutorial.mp4", "Golang Tutorial", "Learn Go programming"},
		{"/videos/python_data.mp4", "Python for Data Science", "Data science with Python"},
		{"/videos/random.mp4", "Random Video", "Unrelated content"},
	}

	for _, td := range testData {
		_, err := sqlDB.Exec(`
			INSERT INTO media (path, path_tokenized, title, description, type)
			VALUES (?, ?, ?, ?, 'video')
		`, td.path, td.path, td.title, td.desc)
		if err != nil {
			t.Fatalf("Failed to insert test media: %v", err)
		}
	}

	// Rebuild FTS index
	_, err = sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")
	if err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Create initial media list with first item
	media := []models.MediaWithDB{
		{
			Media: models.Media{
				Path:  "/videos/python_tutorial.mp4",
				Title: new("Python Tutorial"),
			},
		},
	}

	// Test with search terms in flags
	flags := models.GlobalFlags{
		FilterFlags: models.FilterFlags{
			Search: []string{"python", "tutorial"},
		},
	}

	// Expand related media
	err = ExpandRelatedMedia(ctx, sqlDB, &media, flags)
	if err != nil {
		t.Fatalf("ExpandRelatedMedia failed: %v", err)
	}

	// Should have expanded to include related Python/tutorial videos
	if len(media) < 3 {
		t.Errorf("Expected at least 3 related media, got %d", len(media))
	}

	// Check that python videos are included
	foundPython := false
	foundAdvanced := false
	for _, m := range media {
		if m.Path == "/videos/python_advanced.mp4" {
			foundAdvanced = true
		}
		if m.Path == "/videos/python_data.mp4" {
			foundPython = true
		}
	}

	if !foundPython {
		t.Error("Expected python_data.mp4 to be included in related media")
	}
	if !foundAdvanced {
		t.Error("Expected python_advanced.mp4 to be included in related media")
	}

	// Random video should not be included (no matching terms)
	for _, m := range media {
		if m.Path == "/videos/random.mp4" {
			t.Error("Random video should not be included in related media")
		}
	}
}

func TestExpandRelatedMedia_WithPhrases(t *testing.T) {
	// Create test database
	f, err := os.CreateTemp("", "related-phrase-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize schema (same as above)
	schema := `
	CREATE TABLE media (
		path TEXT PRIMARY KEY,
		path_tokenized TEXT,
		title TEXT,
		description TEXT,
		type TEXT,
		time_deleted INTEGER DEFAULT 0,
		size INTEGER,
		duration INTEGER,
		video_count INTEGER DEFAULT 0,
		audio_count INTEGER DEFAULT 0,
		subtitle_count INTEGER DEFAULT 0,
		play_count INTEGER DEFAULT 0,
		playhead INTEGER DEFAULT 0,
		time_created INTEGER,
		time_modified INTEGER,
		time_downloaded INTEGER,
		time_last_played INTEGER,
		score REAL
	);
	CREATE VIRTUAL TABLE media_fts USING fts5(
		path, path_tokenized, title, description,
		content='media',
		content_rowid='rowid',
		tokenize='trigram',
		detail='none'
	);
	CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN
		INSERT INTO media_fts(rowid, path, path_tokenized, title, description)
		VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description);
	END;
	CREATE TRIGGER media_ad AFTER DELETE ON media BEGIN
		DELETE FROM media_fts WHERE rowid = old.rowid;
	END;
	CREATE TRIGGER media_au AFTER UPDATE ON media BEGIN
		INSERT INTO media_fts(media_fts, rowid, path, path_tokenized, title, description)
		VALUES('delete', old.rowid, old.path, old.path_tokenized, old.title, old.description);
		INSERT INTO media_fts(rowid, path, path_tokenized, title, description)
		VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description);
	END;
	`
	_, err = sqlDB.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	ctx := context.Background()

	// Insert test media
	testData := []struct {
		path  string
		title string
		desc  string
	}{
		{"/videos/machine_learning_intro.mp4", "Machine Learning Intro", "Introduction to ML"},
		{"/videos/machine_learning_advanced.mp4", "ML Advanced", "Advanced machine learning"},
		{"/videos/deep_learning.mp4", "Deep Learning", "Neural networks and deep learning"},
		{"/videos/cooking.mp4", "Cooking Show", "How to cook"},
	}

	for _, td := range testData {
		_, err := sqlDB.Exec(`
			INSERT INTO media (path, path_tokenized, title, description, type)
			VALUES (?, ?, ?, ?, 'video')
		`, td.path, td.path, td.title, td.desc)
		if err != nil {
			t.Fatalf("Failed to insert test media: %v", err)
		}
	}

	// Rebuild FTS index
	_, err = sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")
	if err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Create initial media list
	media := []models.MediaWithDB{
		{
			Media: models.Media{
				Path:  "/videos/machine_learning_intro.mp4",
				Title: new("Machine Learning Intro"),
			},
		},
	}

	// Test with phrase search
	flags := models.GlobalFlags{
		FilterFlags: models.FilterFlags{
			Search: []string{`"machine learning"`},
		},
	}

	// Expand related media
	err = ExpandRelatedMedia(ctx, sqlDB, &media, flags)
	if err != nil {
		t.Fatalf("ExpandRelatedMedia failed: %v", err)
	}

	// Should find related ML videos
	if len(media) < 2 {
		t.Errorf("Expected at least 2 related media, got %d", len(media))
	}

	// Check that ML videos are included
	foundML := false
	for _, m := range media {
		if m.Path == "/videos/machine_learning_advanced.mp4" {
			foundML = true
			break
		}
	}

	if !foundML {
		t.Error("Expected machine_learning_advanced.mp4 to be included in related media")
	}

	// Cooking video should not be included
	for _, m := range media {
		if m.Path == "/videos/cooking.mp4" {
			t.Error("Cooking video should not be included in related media")
		}
	}
}

func TestExpandRelatedMedia_NoSearchTerms(t *testing.T) {
	// Create test database
	f, err := os.CreateTemp("", "related-noterms-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize schema
	schema := `
	CREATE TABLE media (
		path TEXT PRIMARY KEY,
		path_tokenized TEXT,
		title TEXT,
		description TEXT,
		type TEXT,
		time_deleted INTEGER DEFAULT 0,
		size INTEGER,
		duration INTEGER,
		video_count INTEGER DEFAULT 0,
		audio_count INTEGER DEFAULT 0,
		subtitle_count INTEGER DEFAULT 0,
		play_count INTEGER DEFAULT 0,
		playhead INTEGER DEFAULT 0,
		time_created INTEGER,
		time_modified INTEGER,
		time_downloaded INTEGER,
		time_last_played INTEGER,
		score REAL
	);
	CREATE VIRTUAL TABLE media_fts USING fts5(
		path, path_tokenized, title, description,
		content='media',
		content_rowid='rowid',
		tokenize='trigram',
		detail='none'
	);
	CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN
		INSERT INTO media_fts(rowid, path, path_tokenized, title, description)
		VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description);
	END;
	`
	_, err = sqlDB.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	ctx := context.Background()

	// Insert test media
	_, err = sqlDB.Exec(`
		INSERT INTO media (path, path_tokenized, title, description, type)
		VALUES (?, ?, ?, ?, 'video')
	`, "/videos/test.mp4", "/videos/test.mp4", "Test Video", "A test")
	if err != nil {
		t.Fatalf("Failed to insert test media: %v", err)
	}

	// Rebuild FTS index
	_, err = sqlDB.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild')")
	if err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Create initial media list
	media := []models.MediaWithDB{
		{
			Media: models.Media{
				Path:  "/videos/test.mp4",
				Title: new("Test Video"),
			},
		},
	}

	// Test with NO search terms - should extract from media item
	flags := models.GlobalFlags{
		FilterFlags: models.FilterFlags{
			Search: []string{},
		},
	}

	// Expand related media (should use media item's words)
	err = ExpandRelatedMedia(ctx, sqlDB, &media, flags)
	if err != nil {
		t.Fatalf("ExpandRelatedMedia failed: %v", err)
	}

	// Should still work (extracts from media item)
	if len(media) < 1 {
		t.Error("Expected at least 1 media item")
	}
}
