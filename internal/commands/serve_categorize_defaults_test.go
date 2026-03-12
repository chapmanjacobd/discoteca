package commands

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestHandleCategorizeDefaults(t *testing.T) {
	t.Parallel()
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
