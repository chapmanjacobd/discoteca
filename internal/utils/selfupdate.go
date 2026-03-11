package utils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/ulikunitz/xz"
)

// MaybeUpdate will check for an update and install it immediately.
// Returns true if an update was successfully installed and the process should restart.
func MaybeUpdate() bool {
	url := checkUpdate()
	if url == "" {
		return false
	}

	return doUpdate(url)
}

// AutoUpdate will periodically check for an update and install it.
func AutoUpdate() {
	if os.Getenv("DISCO_DISABLE_SELFUPDATE") != "" {
		return
	}

	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsTerminal(os.Stdin.Fd()) {
		return
	}

	go func() {
		// "don't check on startup" - wait 15 minutes before the first potential check
		// and add some jitter to further distribute load.
		time.Sleep(15*time.Minute + time.Duration(rand.Int63n(int64(15*time.Minute))))

		for {
			if shouldCheckProbabilistically() {
				if MaybeUpdate() {
					os.Exit(0)
				}
			}
			// Only attempt to check once every 24 hours.
			time.Sleep(24 * time.Hour)
		}
	}()
}

func shouldCheckProbabilistically() bool {
	statePath := filepath.Join(GetConfigDir(), "update_state.json")

	var state struct {
		LastCheck time.Time `json:"last_check"`
	}

	data, err := os.ReadFile(statePath)
	if err == nil {
		json.Unmarshal(data, &state)
	}

	// Ensure we don't check more than once every 24 hours even if the process restarts.
	if time.Since(state.LastCheck) < 24*time.Hour {
		return false
	}

	// 1. Average of twice per month (2/30 ≈ 0.066).
	// 2. No more than 10% of users on any given day (0.066 < 0.1).
	if rand.Float64() >= 0.066 {
		return false
	}

	// Update the last check time before performing the check to ensure we don't
	// retry immediately on failure and stay within the daily quota.
	state.LastCheck = time.Now()
	os.MkdirAll(filepath.Dir(statePath), 0o755)
	if newData, err := json.Marshal(state); err == nil {
		os.WriteFile(statePath, newData, 0o644)
	}

	return true
}

func whichFilename() string {
	switch {
	case runtime.GOARCH == "amd64" && runtime.GOOS == "linux":
		return "disco.xz"
	case runtime.GOARCH == "arm64" && runtime.GOOS == "linux":
		return "disco.arm64.xz"
	case runtime.GOARCH == "amd64" && runtime.GOOS == "windows":
		return "disco.exe.xz"
	default:
		return ""
	}
}

func doUpdate(url string) bool {
	curp, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't get os.Executable:", err)
		return false
	}
	if doUpdateAt(curp, url) {
		fmt.Fprintln(os.Stderr, "new version downloaded, exiting to get restarted")
		return true
	}
	return false
}

func verifyChecksum(ctx context.Context, url string, data []byte) error {
	// Try downloading checksum
	checksumUrl := url + ".sha256"
	req, err := http.NewRequestWithContext(ctx, "GET", checksumUrl, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If checksum file not found, we'll just skip verification for now
		// unless we want it to be mandatory.
		return nil
	}

	expectedHex, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	actualHash := sha256.Sum256(data)
	actualHex := fmt.Sprintf("%x", actualHash)

	if strings.TrimSpace(string(expectedHex)) != actualHex {
		return fmt.Errorf("expected %s, got %s", string(expectedHex), actualHex)
	}

	return nil
}

func doUpdateAt(curp, url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	newp := curp + ".new"
	oldp := curp + ".old"

	f, err := os.Create(newp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't make file to update:", err)
		return false
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating request:", err)
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't download update:", err)
		return false
	}
	defer resp.Body.Close()

	// 1. Read the update into a buffer so we can checksum it
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't read update:", err)
		return false
	}

	// 2. Download checksum if available
	if err := verifyChecksum(ctx, url, data); err != nil {
		fmt.Fprintln(os.Stderr, "checksum verification failed:", err)
		return false
	}

	// 3. Decompress and write
	xzr, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't decompress update:", err)
		return false
	}

	if _, err = io.Copy(f, xzr); err != nil {
		fmt.Fprintln(os.Stderr, "couldn't write update:", err)
		return false
	}

	if err := os.Chmod(newp, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "couldn't chmod update:", err)
		return false
	}

	if err := os.Rename(curp, oldp); err != nil {
		fmt.Fprintln(os.Stderr, "couldn't rename original file:", err)
		return false
	}

	if err := os.Rename(newp, curp); err != nil {
		fmt.Fprintln(os.Stderr, "couldn't rename new file:", err)
		os.Rename(oldp, curp) // Try to rollback
		return false
	}

	return true
}

var githubApiUrl = "https://api.github.com/repos/chapmanjacobd/discoteca/releases/latest"

func checkUpdate() string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", githubApiUrl, nil)
	if err != nil {
		return ""
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var found struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string
			BrowserDownloadURL string `json:"browser_download_url"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&found); err != nil {
		return ""
	}

	if found.TagName == "v"+Version || found.TagName == Version {
		return ""
	}

	for _, a := range found.Assets {
		if whichFilename() == a.Name {
			return a.BrowserDownloadURL
		}
	}

	return ""
}
