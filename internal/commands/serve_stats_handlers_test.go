package commands

import (
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

// TestHandleCategories tests the categories endpoint
func TestHandleCategories(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_categories.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, categories, time_deleted) VALUES 
		(?, 'Test1', 'video', 'comedy;action', 0),
		(?, 'Test2', 'video', 'comedy', 0),
		(?, 'Test3', 'video', '', 0)`,
		filepath.FromSlash("/tmp/test1.mp4"),
		filepath.FromSlash("/tmp/test2.mp4"),
		filepath.FromSlash("/tmp/test3.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var categories []models.CatStat
	if err := json.NewDecoder(w.Body).Decode(&categories); err != nil {
		t.Fatal(err)
	}

	if len(categories) == 0 {
		t.Error("Expected at least one category")
	}

	foundComedy := false
	for _, cat := range categories {
		if cat.Category == "comedy" && cat.Count == 2 {
			foundComedy = true
		}
	}
	if !foundComedy {
		t.Error("Expected comedy category with count 2")
	}
}

// TestHandleGenres tests the genres endpoint
func TestHandleGenres(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_genres.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, genre, time_deleted) VALUES 
		(?, 'Test1', 'video', 'Action', 0),
		(?, 'Test2', 'video', 'Action', 0),
		(?, 'Test3', 'video', 'Comedy', 0)`,
		filepath.FromSlash("/tmp/test1.mp4"),
		filepath.FromSlash("/tmp/test2.mp4"),
		filepath.FromSlash("/tmp/test3.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/api/genres", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var genres []models.CatStat
	if err := json.NewDecoder(w.Body).Decode(&genres); err != nil {
		t.Fatal(err)
	}

	if len(genres) == 0 {
		t.Error("Expected at least one genre")
	}

	foundAction := false
	for _, g := range genres {
		if g.Category == "Action" && g.Count == 2 {
			foundAction = true
		}
	}
	if !foundAction {
		t.Error("Expected Action genre with count 2")
	}
}

// TestHandleRatings tests the ratings endpoint
func TestHandleRatings(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_ratings.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	_, err := sqlDB.Exec(`INSERT INTO media (path, title, media_type, score, time_deleted) VALUES 
		(?, 'Test1', 'video', 5.0, 0),
		(?, 'Test2', 'video', 5.0, 0),
		(?, 'Test3', 'video', 3.0, 0)`,
		filepath.FromSlash("/tmp/test1.mp4"),
		filepath.FromSlash("/tmp/test2.mp4"),
		filepath.FromSlash("/tmp/test3.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	defer cmd.Close()
	mux := cmd.Mux()

	req := httptest.NewRequest(http.MethodGet, "/api/ratings", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var ratings []models.RatStat
	if err := json.NewDecoder(w.Body).Decode(&ratings); err != nil {
		t.Fatal(err)
	}

	if len(ratings) == 0 {
		t.Error("Expected at least one rating")
	}

	found5Star := false
	for _, r := range ratings {
		if r.Rating == 5 && r.Count == 2 {
			found5Star = true
		}
	}
	if !found5Star {
		t.Error("Expected 5-star rating with count 2")
	}
}
