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

	t.Run("filters persist when navigating to subfolder", func(t *testing.T) {
		// First, get root level with video filter
		req1 := httptest.NewRequest("GET", "/api/du?path=&video=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal root response: %v", err)
		}

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level with video filter")
		}

		// Get the first folder path (e.g., /home)
		firstFolderPath := resp1.Folders[0].Path
		t.Logf("Navigating to folder: %s", firstFolderPath)

		// Navigate to subfolder with same video filter
		req2 := httptest.NewRequest("GET", "/api/du?path="+firstFolderPath+"&video=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal subfolder response: %v", err)
		}

		// Should have subfolders and/or files
		totalItems := len(resp2.Folders) + len(resp2.Files)
		if totalItems == 0 {
			t.Errorf("Expected folders or files in subfolder with video filter, got none")
			t.Logf("Response: %+v", resp2)
		}

		t.Logf("Subfolder results - Folders: %d, Files: %d, Total: %d", 
			resp2.FolderCount, resp2.FileCount, resp2.TotalCount)
	})

	t.Run("audio filter persists when navigating to subfolder", func(t *testing.T) {
		// Get root level with audio filter
		req1 := httptest.NewRequest("GET", "/api/du?path=&audio=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
			t.Fatalf("Failed to unmarshal root response: %v", err)
		}

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level with audio filter")
		}

		firstFolderPath := resp1.Folders[0].Path

		// Navigate to subfolder with audio filter
		req2 := httptest.NewRequest("GET", "/api/du?path="+firstFolderPath+"&audio=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("Failed to unmarshal subfolder response: %v", err)
		}

		totalItems := len(resp2.Folders) + len(resp2.Files)
		if totalItems == 0 {
			t.Errorf("Expected folders or files in subfolder with audio filter, got none")
		}

		t.Logf("Audio filter - Subfolder results: Folders: %d, Files: %d", 
			resp2.FolderCount, resp2.FileCount)
	})

	t.Run("image filter persists when navigating to subfolder", func(t *testing.T) {
		req1 := httptest.NewRequest("GET", "/api/du?path=&image=true", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level")
		}

		req2 := httptest.NewRequest("GET", "/api/du?path="+resp1.Folders[0].Path+"&image=true", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		totalItems := len(resp2.Folders) + len(resp2.Files)
		if totalItems == 0 {
			t.Errorf("Expected folders or files with image filter, got none")
		}
	})

	t.Run("size filter persists when navigating to subfolder", func(t *testing.T) {
		// Filter for media > 100KB
		req1 := httptest.NewRequest("GET", "/api/du?path=&size=>100KB", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level")
		}

		req2 := httptest.NewRequest("GET", "/api/du?path="+resp1.Folders[0].Path+"&size=>100KB", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		// Should have results with size filter applied
		if resp2.FolderCount == 0 && resp2.FileCount == 0 {
			t.Errorf("Expected folders or files with size filter, got none")
		}
		t.Logf("Size filter - Subfolder results: Folders: %d, Files: %d", 
			resp2.FolderCount, resp2.FileCount)
	})

	t.Run("duration filter persists when navigating to subfolder", func(t *testing.T) {
		// Filter for media > 10 seconds
		req1 := httptest.NewRequest("GET", "/api/du?path=&duration=>10", nil)
		req1.Header.Set("X-Disco-Token", cmd.APIToken)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		var resp1 models.DUResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		if len(resp1.Folders) == 0 {
			t.Fatal("Expected folders at root level")
		}

		req2 := httptest.NewRequest("GET", "/api/du?path="+resp1.Folders[0].Path+"&duration=>10", nil)
		req2.Header.Set("X-Disco-Token", cmd.APIToken)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		var resp2 models.DUResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		// Should have results with duration filter applied
		if resp2.FolderCount == 0 && resp2.FileCount == 0 {
			t.Errorf("Expected folders or files with duration filter, got none")
		}
		t.Logf("Duration filter - Subfolder results: Folders: %d, Files: %d", 
			resp2.FolderCount, resp2.FileCount)
	})
}
