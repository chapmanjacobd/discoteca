package commands

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestHandleCategorizeKeywords(t *testing.T) {
	t.Parallel()
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
