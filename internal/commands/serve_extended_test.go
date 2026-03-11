package commands

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func TestServeExtended_Filters(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_filters.db")

	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)
	// Add diverse test data
	sqlDB.Exec(`INSERT INTO media (path, type, size, time_deleted) VALUES ('/v1.mp4', 'video', 100, 0)`)
	sqlDB.Exec(`INSERT INTO media (path, type, size, time_deleted) VALUES ('/a1.mp3', 'audio', 200, 0)`)
	sqlDB.Close()

	cmd := &ServeCmd{
		Databases: []string{dbPath},
	}
	mux := cmd.Mux()

	req := httptest.NewRequest("GET", "/api/filter-bins?db="+dbPath, nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var bins map[string]any
	json.NewDecoder(w.Body).Decode(&bins)

	if bins["type"] == nil {
		t.Error("Expected type bins to be present")
	}
}
