package commands

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestHandleCategorizeDeleteCategory(t *testing.T) {
	tmpDB, err := os.CreateTemp(t.TempDir(), "disco_test_delete_*.db")
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
