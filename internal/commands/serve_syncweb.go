//go:build syncweb

package commands

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discotheque/internal/syncweb"
	"github.com/chapmanjacobd/discotheque/internal/utils"
)

var (
	swInstance *syncweb.Syncweb
	swMu       sync.Mutex
)

func (c *ServeCmd) setupSyncweb() {
	swMu.Lock()
	defer swMu.Unlock()
	sw, err := syncweb.NewSyncweb(c.SyncwebHome, "disco-syncweb", "")
	if err != nil {
		slog.Warn("Failed to initialize Syncweb instance", "error", err)
	} else {
		swInstance = sw
		if err := sw.Start(); err != nil {
			slog.Error("Failed to start Syncweb instance", "error", err)
		} else {
			slog.Info("Syncweb instance started", "myID", sw.Node.MyID())
		}
	}
}

func (c *ServeCmd) addSyncwebRoots(resultsMap map[string]LsEntry, counts map[string]int, path string) {
	swMu.Lock()
	defer swMu.Unlock()
	if swInstance != nil && (path == "/" || path == "") && swInstance.IsRunning() {
		for _, folder := range swInstance.GetFolders() {
			id := folder.ID
			entryPath := fmt.Sprintf("syncweb://%s/", id)
			name := id
			if localPath, ok := swInstance.GetFolderPath(id); ok {
				name += " (" + filepath.Base(localPath) + ")"
			}
			resultsMap[entryPath] = LsEntry{
				Name:  name,
				Path:  entryPath,
				IsDir: true,
			}
			counts[entryPath] = 1000 // High priority for roots
		}
	}
}

func (c *ServeCmd) resolveSyncwebPath(path string) (string, string, error) {
	swMu.Lock()
	defer swMu.Unlock()
	if swInstance == nil {
		return "", "", fmt.Errorf("syncweb not configured")
	}
	return swInstance.ResolveLocalPath(path)
}

func (c *ServeCmd) serveSyncwebContent(w http.ResponseWriter, r *http.Request, folderID, path, localPath string) {
	swMu.Lock()
	if swInstance == nil || !swInstance.IsRunning() {
		swMu.Unlock()
		http.Error(w, "Syncweb not configured or offline", http.StatusServiceUnavailable)
		return
	}
	sw := swInstance
	swMu.Unlock()

	slog.Info("Serving remote Syncweb file via block pulling", "path", path)
	rs, err := sw.NewReadSeeker(r.Context(), folderID, strings.TrimPrefix(path, "syncweb://"+folderID+"/"))
	if err != nil {
		slog.Error("Failed to create SyncwebReadSeeker", "path", path, "error", err)
		http.Error(w, "Failed to stream remote file", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, filepath.Base(localPath), time.Now(), rs)
}

func (c *ServeCmd) handleSyncwebFolders(w http.ResponseWriter, r *http.Request) {
	swMu.Lock()
	defer swMu.Unlock()
	if swInstance == nil || !swInstance.IsRunning() {
		http.Error(w, "Syncweb not configured or offline", http.StatusServiceUnavailable)
		return
	}

	folders := swInstance.GetFolders()
	for i := range folders {
		folders[i].Path = "" // Hide local path from API
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

func (c *ServeCmd) handleSyncwebLs(w http.ResponseWriter, r *http.Request) {
	swMu.Lock()
	defer swMu.Unlock()
	if swInstance == nil || !swInstance.IsRunning() {
		http.Error(w, "Syncweb not configured or offline", http.StatusServiceUnavailable)
		return
	}

	folderID := r.URL.Query().Get("folder")
	prefix := r.URL.Query().Get("prefix")

	// Security check: ensure the folder is one we actually have
	configuredFolders := swInstance.GetFolders()
	found := false
	for _, f := range configuredFolders {
		if f.ID == folderID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Folder not found or not configured", http.StatusNotFound)
		return
	}

	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	seq, cancel := swInstance.Node.App.Internals.AllGlobalFiles(folderID)
	defer cancel()

	resultsMap := make(map[string]LsEntry)
	for meta := range seq {
		name := meta.Name
		if !strings.HasPrefix(name, prefix) || name == prefix {
			continue
		}

		rel := strings.TrimPrefix(name, prefix)
		parts := strings.Split(rel, "/")
		entryName := parts[0]
		isDir := len(parts) > 1

		fullSyncwebPath := fmt.Sprintf("syncweb://%s/%s", folderID, filepath.Join(prefix, entryName))
		if _, ok := resultsMap[fullSyncwebPath]; ok {
			continue
		}

		localPath, _, _ := swInstance.ResolveLocalPath(fullSyncwebPath)
		isLocal := utils.FileExists(localPath)

		entry := LsEntry{
			Name:  entryName,
			Path:  fullSyncwebPath,
			IsDir: isDir,
			Local: isLocal,
		}
		if !isDir {
			entry.Type = utils.DetectMimeType(entryName)
		}
		resultsMap[fullSyncwebPath] = entry
	}

	results := make([]LsEntry, 0, len(resultsMap))
	for _, entry := range resultsMap {
		results = append(results, entry)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].IsDir != results[j].IsDir {
			return results[i].IsDir
		}
		return results[i].Name < results[j].Name
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (c *ServeCmd) handleSyncwebDownload(w http.ResponseWriter, r *http.Request) {
	swMu.Lock()
	defer swMu.Unlock()
	if swInstance == nil || !swInstance.IsRunning() {
		http.Error(w, "Syncweb not configured or offline", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	// Try to decode from JSON body first
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Fallback to query param for compatibility if body is empty or malformed
		req.Path = r.URL.Query().Get("path")
	}

	if req.Path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	localPath, folderID, err := swInstance.ResolveLocalPath(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if c.isPathBlacklisted(localPath) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	folderPath, ok := swInstance.GetFolderPath(folderID)
	if !ok {
		http.Error(w, "Folder root not found", http.StatusInternalServerError)
		return
	}
	relativePath, _ := filepath.Rel(folderPath, localPath)

	if err := swInstance.Unignore(folderID, relativePath); err != nil {
		slog.Error("Syncweb download trigger failed", "path", req.Path, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "Download triggered")
}

func (c *ServeCmd) handleSyncwebToggle(w http.ResponseWriter, r *http.Request) {
	swMu.Lock()
	defer swMu.Unlock()

	if swInstance == nil {
		http.Error(w, "Syncweb not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Offline bool `json:"offline"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Offline {
		if swInstance.IsRunning() {
			slog.Info("Stopping Syncweb backend (Offline Mode)")
			swInstance.Stop()
		}
	} else {
		if !swInstance.IsRunning() {
			slog.Info("Starting Syncweb backend (Online Mode)")
			if err := swInstance.Start(); err != nil {
				slog.Error("Failed to restart Syncweb", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"offline": !swInstance.IsRunning()})
}

func (c *ServeCmd) handleSyncwebStatus(w http.ResponseWriter, r *http.Request) {
	swMu.Lock()
	defer swMu.Unlock()

	status := "Not Configured"
	offline := true
	if swInstance != nil {
		if swInstance.IsRunning() {
			status = "Online"
			offline = false
		} else {
			status = "Offline"
			offline = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  status,
		"offline": offline,
	})
}
