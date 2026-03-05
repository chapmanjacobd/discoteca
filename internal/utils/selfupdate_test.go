package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestCheckUpdate(t *testing.T) {
	filename := whichFilename()
	if filename == "" {
		t.Skip("No update filename for this platform")
	}

	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := struct {
			TagName string `json:"tag_name"`
			Assets  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}{
			TagName: "v99.9.9",
			Assets: []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{
				{
					Name:               filename,
					BrowserDownloadURL: "http://example.com/disco.xz",
				},
			},
		}
		json.NewEncoder(w).Encode(res)
	}))
	defer ts.Close()

	// Override URL
	oldUrl := githubApiUrl
	githubApiUrl = ts.URL
	defer func() { githubApiUrl = oldUrl }()

	// Save old Version
	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	url := checkUpdate()
	if url != "http://example.com/disco.xz" {
		t.Errorf("Expected http://example.com/disco.xz, got %s", url)
	}

	// Test same version
	Version = "99.9.9"
	url = checkUpdate()
	if url != "" {
		t.Errorf("Expected no update for same version, got %s", url)
	}
}

func TestDoUpdateAt(t *testing.T) {
	// 1. Create a dummy "executable"
	tmpDir, err := os.MkdirTemp("", "selfupdate_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	curp := filepath.Join(tmpDir, "disco")
	os.WriteFile(curp, []byte("old version"), 0o755)

	// 2. Prepare compressed "new version"
	newContent := "new version"
	var xzBuf bytes.Buffer
	xzw, err := xz.NewWriter(&xzBuf)
	if err != nil {
		t.Fatal(err)
	}
	xzw.Write([]byte(newContent))
	xzw.Close()

	// 3. Mock server to serve the compressed update and checksum
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			actualHash := sha256.Sum256(xzBuf.Bytes())
			fmt.Fprintf(w, "%x", actualHash)
			return
		}
		w.Write(xzBuf.Bytes())
	}))
	defer ts.Close()

	// 4. Run doUpdateAt
	success := doUpdateAt(curp, ts.URL+"/disco.xz")
	if !success {
		t.Error("doUpdateAt failed")
	}

	// 5. Verify update
	updatedContent, err := os.ReadFile(curp)
	if err != nil {
		t.Fatal(err)
	}
	if string(updatedContent) != newContent {
		t.Errorf("Expected %q, got %q", newContent, string(updatedContent))
	}

	// Verify old version exists as .old
	oldContent, err := os.ReadFile(curp + ".old")
	if err != nil {
		t.Fatal(err)
	}
	if string(oldContent) != "old version" {
		t.Errorf("Expected 'old version', got %q", string(oldContent))
	}
}
