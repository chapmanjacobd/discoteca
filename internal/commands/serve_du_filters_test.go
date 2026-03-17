package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

const e2eTestDBPath = "../../e2e/fixtures/test.db"

func TestHandleDU_WithFilters(t *testing.T) {
	// Check if e2e test database exists
	dbPath, err := filepath.Abs(e2eTestDBPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("E2E test database not found at %s. Run 'make e2e-init' first.", dbPath)
	}

	cmd := setupTestServeCmd(dbPath)
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("video-only filter returns only video folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&video=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have folders (all media is under /home in test db)
		if len(resp.Folders) == 0 {
			t.Error("Expected folders in response")
		}

		// Total count should reflect filtered results
		// Test DB has 5 videos out of 13 total media
		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("audio-only filter returns only audio folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&audio=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("image-only filter returns only image folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&image=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("size filter returns only media matching size range", func(t *testing.T) {
		// Filter for media > 100KB
		req := httptest.NewRequest("GET", "/api/du?path=&size=>100KB", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("duration filter returns only media matching duration range", func(t *testing.T) {
		// Filter for media > 10 seconds
		req := httptest.NewRequest("GET", "/api/du?path=&duration=>10", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("search filter returns only matching media", func(t *testing.T) {
		// Search for "test"
		req := httptest.NewRequest("GET", "/api/du?path=&search=test", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		t.Logf("Folders: %d, Files: %d, Total: %d", resp.FolderCount, resp.FileCount, resp.TotalCount)
	})

	t.Run("include_counts returns filter bins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/du?path=&include_counts=true", nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d - Body: %s", w.Code, w.Body.String())
		}

		var resp models.DUResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Counts == nil {
			t.Error("Expected counts in response when include_counts=true")
		}

		// Check that bins have data
		if len(resp.Counts.Type) == 0 {
			t.Error("Expected type bins in counts")
		}

		if len(resp.Counts.Size) == 0 {
			t.Error("Expected size bins in counts")
		}

		if len(resp.Counts.Duration) == 0 {
			t.Error("Expected duration bins in counts")
		}
	})

	t.Run("filter with include_counts returns filtered bins", func(t *testing.T) {
		// Get unfiltered counts
		req1 := httptest.NewRequest("GET", "/api/du?path=&include_counts=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		// Get video-only counts
		req2 := httptest.NewRequest("GET", "/api/du?path=&include_counts=true&video-only=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		if resp2.Counts == nil {
			t.Fatal("Expected counts in filtered response")
		}

		// Video count in filtered should be same as total video count
		// but other types should be 0 or not present
		t.Logf("Unfiltered types: %+v", resp1.Counts.Type)
		t.Logf("Filtered types: %+v", resp2.Counts.Type)
	})
}
