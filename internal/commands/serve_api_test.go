package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/testutils"
)

func TestCustomKeywordsCategorization(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	db := fixture.GetDB()
	InitDB(db)

	// Manual migration for custom_keywords since we are using fixture.GetDB() which might not have run it if we don't call runMigrations
	// Actually, ServeCmd.Run calls runMigrations, but here we are using Mux.
	// Let's ensure the table exists.
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS custom_keywords (
		category TEXT NOT NULL,
		keyword TEXT NOT NULL,
		PRIMARY KEY (category, keyword)
	)`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO media (path, title, type, time_deleted)
		VALUES ('/media/custom_test.mp4', 'Custom Test', 'video', 0)
	`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	handler := cmd.Mux()

	// 1. Save custom keyword
	kwReq := map[string]string{
		"category": "special",
		"keyword":  "custom",
	}
	body, _ := json.Marshal(kwReq)
	req := httptest.NewRequest(http.MethodPost, "/api/categorize/keyword", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for keyword save, got %d", w.Code)
	}

	// 2. Apply categorization
	req = httptest.NewRequest(http.MethodPost, "/api/categorize/apply", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for categorize apply, got %d", w.Code)
	}

	// 3. Verify categorization
	db = fixture.GetDB()
	var categories sql.NullString
	err = db.QueryRow("SELECT categories FROM media WHERE path = '/media/custom_test.mp4'").Scan(&categories)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}

	if !categories.Valid || categories.String != ";special;" {
		t.Errorf("Expected categories ';special;', got '%s' (valid: %v)", categories.String, categories.Valid)
	}
}

func TestDUServerSideFiltering(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	db := fixture.GetDB()
	InitDB(db)
	_, err := db.Exec(`
		INSERT INTO media (path, type, size, duration, time_deleted)
		VALUES
		('media/video/v1.mp4', 'video', 1000, 10, 0),
		('media/audio/a1.mp3', 'audio', 2000, 20, 0)
	`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	handler := cmd.Mux()

	// 1. Filter by video
	req := httptest.NewRequest(http.MethodGet, "/api/du?video=true", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp []struct {
		Path      string `json:"path"`
		TotalSize int64  `json:"total_size"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// Should find "media/video" folder (immediate parent of the video file)
	// The video filter should still apply, so we should see the media/video folder
	// with accurate size aggregation from video files
	foundVideoFolder := false
	for _, r := range resp {
		if r.Path == "media/video" || r.Path == "/media/video" {
			foundVideoFolder = true
			// Size should include video file (1000 bytes)
			if r.TotalSize < 1000 {
				t.Errorf("Expected media/video folder to include video size, got %d", r.TotalSize)
			}
		}
	}
	if !foundVideoFolder {
		t.Errorf("Did not find media/video folder in response: %v", resp)
	}
}
