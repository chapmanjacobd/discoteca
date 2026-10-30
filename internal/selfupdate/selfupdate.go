// Package selfupdate checks for and installs discoteca releases.
package selfupdate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/ulikunitz/xz"

	"github.com/chapmanjacobd/discoteca/internal/config"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
)

// MaybeUpdate will check for an update and install it immediately.
// Returns true if an update was successfully installed and the process should restart.
func MaybeUpdate(ctx context.Context) bool {
	updated, _ := MaybeUpdateE(ctx)
	return updated
}

// MaybeUpdateE is MaybeUpdate with the download/install error exposed.
func MaybeUpdateE(ctx context.Context) (bool, error) {
	url := checkUpdate(ctx)
	if url == "" {
		return false, nil
	}

	curp, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("could not locate current executable: %w", err)
	}
	if err := doUpdateAt(ctx, curp, url); err != nil {
		return false, err
	}
	fmt.Fprintln(fileutil.Stderr, "new version downloaded, exiting to get restarted")
	return true, nil
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
		time.Sleep(15*time.Minute + time.Duration(rand.Int64N(int64(15*time.Minute))))

		for {
			if shouldCheckProbabilistically() {
				if MaybeUpdate(context.Background()) {
					os.Exit(0)
				}
			}
			// Only attempt to check once every 24 hours.
			time.Sleep(24 * time.Hour)
		}
	}()
}

func shouldCheckProbabilistically() bool {
	statePath := filepath.Join(config.GetConfigDir(), "update_state.json")

	var state struct {
		LastCheck time.Time `json:"last_check"`
	}

	data, err := os.ReadFile(statePath)
	if err == nil {
		_ = json.Unmarshal(data, &state)
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
	_ = os.MkdirAll(filepath.Dir(statePath), 0o755)
	if newData, err := json.Marshal(state); err == nil {
		_ = os.WriteFile(statePath, newData, 0o644)
	}

	return true
}

func whichFilename() string {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "disco-linux-amd64"
		case "arm64":
			return "disco-linux-arm64"
		case "arm":
			return "disco-linux-armv7"
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "disco-darwin-amd64"
		case "arm64":
			return "disco-darwin-arm64"
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "disco-windows-amd64.exe"
		case "arm64":
			return "disco-windows-arm64.exe"
		}
	}
	return ""
}

func verifyChecksum(ctx context.Context, url string, data []byte) error {
	// Try downloading checksum
	checksumURL := url + ".sha256"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksum file unavailable: %s returned %s", checksumURL, resp.Status)
	}

	expectedHex, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	actualHash := sha256.Sum256(data)
	actualHex := hex.EncodeToString(actualHash[:])

	if strings.TrimSpace(string(expectedHex)) != actualHex {
		return fmt.Errorf("expected %s, got %s", string(expectedHex), actualHex)
	}

	return nil
}

func doUpdateAt(ctx context.Context, curp, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	newp := curp + ".new"
	oldp := curp + ".old"

	f, err := os.Create(newp)
	if err != nil {
		return fmt.Errorf("couldn't make file to update: %w", err)
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't download update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	// Read the update into a buffer so it can be checksummed before install.
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("couldn't read update: %w", err)
	}

	if checksumErr := verifyChecksum(ctx, url, data); checksumErr != nil {
		return fmt.Errorf("checksum verification failed: %w", checksumErr)
	}

	// Release assets are raw binaries; older/third-party assets may be xz
	// compressed. Accept both so self-update matches the actual release names.
	if strings.HasSuffix(strings.ToLower(url), ".xz") {
		xzr, readerErr := xz.NewReader(bytes.NewReader(data))
		if readerErr != nil {
			return fmt.Errorf("couldn't decompress update: %w", readerErr)
		}
		if _, copyErr := io.Copy(f, xzr); copyErr != nil {
			return fmt.Errorf("couldn't write update: %w", copyErr)
		}
	} else if _, writeErr := f.Write(data); writeErr != nil {
		return fmt.Errorf("couldn't write update: %w", writeErr)
	}

	if err := os.Chmod(newp, 0o755); err != nil {
		return fmt.Errorf("couldn't chmod update: %w", err)
	}

	if err := os.Rename(curp, oldp); err != nil {
		return fmt.Errorf("couldn't rename original file: %w", err)
	}

	if err := os.Rename(newp, curp); err != nil {
		if rbErr := os.Rename(oldp, curp); rbErr != nil {
			return fmt.Errorf("couldn't rename new file: %w; rollback failed: %w", err, rbErr)
		}
		return fmt.Errorf("couldn't rename new file: %w", err)
	}

	return nil
}

var githubAPIURL = "https://api.github.com/repos/chapmanjacobd/discoteca/releases/latest"

func checkUpdate(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, nil)
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
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
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
