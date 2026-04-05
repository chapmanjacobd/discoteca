package commands_test

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

func TestHandleCategorizeKeyword(t *testing.T) {
	tmpDB, err := os.CreateTemp(t.TempDir(), "disco_test_keyword_*.db")
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

	cmd := &commands.ServeCmd{
		Databases: []string{tmpDB.Name()},
		ReadOnly:  false,
	}

	t.Run("AddKeyword inserts new keyword", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"category": "Genre", "keyword": "Rock"}`))
		req := httptest.NewRequest(http.MethodPost, "/api/categorize/keyword", body)
		w := httptest.NewRecorder()

		cmd.HandleCategorizeKeyword(w, req)

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
		err = db.QueryRow("SELECT COUNT(*) FROM custom_keywords WHERE category = 'Genre' AND keyword = 'Rock'").
			Scan(&count)
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

		cmd.HandleCategorizeKeyword(w, req)

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

		cmd.HandleCategorizeKeyword(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify keyword was deleted
		db, err = sql.Open("sqlite3", tmpDB.Name())
		if err != nil {
			t.Fatalf("Failed to reopen_db: %v", err)
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM custom_keywords WHERE category = 'Mood' AND keyword = 'Happy'").
			Scan(&count)
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

		cmd.HandleCategorizeKeyword(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 in read-only mode, got %d", w.Code)
		}
	})
}
