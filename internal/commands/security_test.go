//go:build syncweb

package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/testutils"
	"github.com/syncthing/syncthing/lib/config"
)

func TestSecurity_SyncwebPathTraversal(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// 1. Create a "secret" file outside any syncweb folder
	secretContent := "top-secret-data"
	secretFile := filepath.Join(fixture.TempDir, "secret.txt")
	os.WriteFile(secretFile, []byte(secretContent), 0o600)

	// 2. Setup Syncweb with a dummy folder
	syncDir := filepath.Join(fixture.TempDir, "sync")
	os.MkdirAll(syncDir, 0o700)

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	cmd.APIToken = "test-token"
	cmd.SyncwebHome = fixture.TempDir
	cmd.setupSyncweb()
	if swInstance == nil {
		t.Fatal("Syncweb instance failed to initialize")
	}
	defer swInstance.Stop()

	folderID := "test-folder"
	err := swInstance.AddFolder(folderID, "Test", syncDir, config.FolderTypeSendReceive)
	if err != nil {
		t.Fatalf("Failed to add folder: %v", err)
	}

	// 3. Attempt path traversal via /api/raw
	// The traversal should go from syncDir back to fixture.TempDir/secret.txt
	// syncDir is fixture.TempDir/sync
	// relative path: ../secret.txt
	traversalPath := fmt.Sprintf("syncweb://%s/../secret.txt", folderID)

	req := httptest.NewRequest(http.MethodGet, "/api/raw?path="+traversalPath, nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()

	cmd.handleRaw(w, req)

	resp := w.Result()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("Security Vulnerability: Path traversal allowed! Got status 200")
		// Read body to confirm
		body := w.Body.String()
		if body == secretContent {
			t.Errorf("Security Vulnerability: Successfully read secret file content!")
		}
	} else if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Logf("Got status %d, which might be acceptable if it blocked the access", resp.StatusCode)
	}
}

func TestSecurity_Blacklist(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	cmd.APIToken = "test-token"

	testCases := []struct {
		path     string
		expected int
	}{
		{"/etc/passwd", http.StatusForbidden},
		{"/home/user/.ssh/id_rsa", http.StatusForbidden},
		{"/media/video.mp4", http.StatusForbidden}, // Returns 403 when not in DB
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/api/raw?path="+tc.path, nil)
		req.Header.Set("X-Disco-Token", cmd.APIToken)
		w := httptest.NewRecorder()
		cmd.handleRaw(w, req)
		if w.Code != tc.expected {
			t.Errorf("Path %s: expected status %d, got %d", tc.path, tc.expected, w.Code)
		}
	}
}

func TestSecurity_SyncwebInfoDisclosure(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	syncDir := filepath.Join(fixture.TempDir, "sync")
	os.MkdirAll(syncDir, 0o700)

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	cmd.APIToken = "test-token"
	cmd.SyncwebHome = fixture.TempDir
	cmd.setupSyncweb()
	if swInstance == nil {
		t.Fatal("Syncweb instance failed to initialize")
	}
	defer swInstance.Stop()

	folderID := "test-folder"
	swInstance.AddFolder(folderID, "Test", syncDir, config.FolderTypeSendReceive)

	// 4. Check /api/syncweb/folders for local path disclosure
	req := httptest.NewRequest(http.MethodGet, "/api/syncweb/folders", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()

	cmd.handleSyncwebFolders(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var folders []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&folders); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	for _, f := range folders {
		if path, ok := f["path"].(string); ok && path != "" {
			t.Errorf("Security Issue: Local path disclosure in /api/syncweb/folders: %s", path)
		}
	}
}

func TestSecurity_SyncwebFolderID(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
	}
	cmd.APIToken = "test-token"
	cmd.SyncwebHome = fixture.TempDir
	cmd.setupSyncweb()
	if swInstance == nil {
		t.Fatal("Syncweb instance failed to initialize")
	}
	defer swInstance.Stop()

	// Try to list a folder that doesn't exist
	req := httptest.NewRequest(http.MethodGet, "/api/syncweb/ls?folder=invalid-folder", nil)
	req.Header.Set("X-Disco-Token", cmd.APIToken)
	w := httptest.NewRecorder()

	cmd.handleSyncwebLs(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid folder, got %d", resp.StatusCode)
	}
}
