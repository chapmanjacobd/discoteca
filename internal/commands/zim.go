package commands

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

const (
	KiwixBin       = "kiwix-serve"
	KiwixPortStart = 8181
)

type OpdsEntry struct {
	Title string `xml:"title"`
	Name  string `xml:"name"`
	Link  []struct {
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
}

type OpdsFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []OpdsEntry `xml:"entry"`
}

type KiwixInstance struct {
	Process  *exec.Cmd
	Port     int
	ZimPath  string
	LastUsed time.Time
}

type KiwixManager struct {
	Instances map[string]*KiwixInstance // zimPath -> instance
	Mutex     sync.Mutex
	UsedPorts map[int]bool
}

var ZimManager = &KiwixManager{
	Instances: make(map[string]*KiwixInstance),
	UsedPorts: make(map[int]bool),
}

// StartZimManager starts the ZIM manager's background cleanup goroutine.
// This should be called during application startup.
func StartZimManager() {
	go ZimManager.cleanupOldInstances()
}

func (c *ServeCmd) HandleZimProxy(w http.ResponseWriter, r *http.Request) {
	port := r.PathValue("port")
	if port == "" {
		http.Error(w, "Missing port", http.StatusBadRequest)
		return
	}

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%s", port))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	// We need to strip the prefix /api/zim/proxy/{port} before sending to target
	// but kiwix-serve was started with --urlRootLocation=/api/zim/proxy/{port}/
	// so it might actually expect the full path.
	// The filestash plugin didn't seem to strip it in its ZimProxyHandler.
	proxy.ServeHTTP(w, r)
}

func (c *ServeCmd) HandleZimView(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	localPath := path

	if !strings.HasSuffix(strings.ToLower(localPath), ".zim") {
		http.Error(w, "Not a .zim file", http.StatusBadRequest)
		return
	}

	if !utils.FileExists(localPath) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	port, err := ZimManager.EnsureKiwixServing(r.Context(), localPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err2 := WaitForKiwixReady(r.Context(), port, 10*time.Second); err2 != nil {
		http.Error(
			w,
			fmt.Sprintf("Kiwix server did not start in time: %s", err2.Error()),
			http.StatusServiceUnavailable,
		)
		return
	}

	contentURL, err := getKiwixContentURL(r.Context(), port)
	if err != nil {
		models.Log.Warn("Could not parse ZIM catalog, using root URL", "error", err)
		contentURL = fmt.Sprintf("/api/zim/proxy/%d/", port)
	}

	zimName := filepath.Base(localPath)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <style>
        body, html { margin: 0; padding: 0; height: 100%%; overflow: hidden; background: #000; }
        iframe { width: 100%%; height: 100%%; border: none; }
    </style>
</head>
<body>
    <iframe src="%s" allowfullscreen></iframe>
</body>
</html>`, zimName, contentURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (m *KiwixManager) EnsureKiwixServing(ctx context.Context, zimPath string) (int, error) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if instance, exists := m.Instances[zimPath]; exists {
		instance.LastUsed = time.Now()
		return instance.Port, nil
	}

	port := m.FindAvailablePort(ctx)
	if port == 0 {
		return 0, errors.New("no available ports for kiwix-serve")
	}

	cmd := exec.CommandContext(
		ctx,
		KiwixBin,
		"--nolibrarybutton",
		"-p", strconv.Itoa(port),
		fmt.Sprintf("--urlRootLocation=/api/zim/proxy/%d/", port),
		zimPath,
	)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start kiwix-serve: %w", err)
	}

	m.Instances[zimPath] = &KiwixInstance{
		Process:  cmd,
		Port:     port,
		ZimPath:  zimPath,
		LastUsed: time.Now(),
	}
	m.UsedPorts[port] = true

	models.Log.Info("Started kiwix-serve", "port", port, "path", zimPath)
	return port, nil
}

func (m *KiwixManager) FindAvailablePort(ctx context.Context) int {
	for i := range 100 {
		port := KiwixPortStart + i
		if !m.UsedPorts[port] && IsPortAvailable(ctx, port) {
			return port
		}
	}
	return 0
}

func (m *KiwixManager) cleanupOldInstances() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.Mutex.Lock()
		cutoff := time.Now().Add(-30 * time.Minute)
		for path, instance := range m.Instances {
			if instance.LastUsed.Before(cutoff) {
				models.Log.Info("Cleaning up unused kiwix-serve instance", "port", instance.Port, "path", path)
				if instance.Process.Process != nil {
					if err := instance.Process.Process.Kill(); err != nil {
						models.Log.Warn("Failed to kill kiwix-serve process", "port", instance.Port, "error", err)
					}
				}
				_ = instance.Process.Wait()
				delete(m.UsedPorts, instance.Port)
				delete(m.Instances, path)
			}
		}
		m.Mutex.Unlock()
	}
}

func IsPortAvailable(ctx context.Context, port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

var WaitForKiwixReady = DefaultWaitForKiwixReady

func DefaultWaitForKiwixReady(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	urlRoot := fmt.Sprintf("/api/zim/proxy/%d/", port)
	checkURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, urlRoot)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, checkURL, nil)
		if err == nil {
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for kiwix-serve on port %d", port)
}

func getKiwixContentURL(ctx context.Context, port int) (string, error) {
	urlRoot := fmt.Sprintf("/api/zim/proxy/%d", port)
	catalogURL := fmt.Sprintf("http://127.0.0.1:%d%s/catalog/v2/entries", port, urlRoot)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("catalog returned status %d", resp.StatusCode)
	}

	var feed OpdsFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return "", err
	}

	if len(feed.Entries) == 1 {
		for _, link := range feed.Entries[0].Link {
			if link.Type == "text/html" {
				contentPath := strings.TrimPrefix(link.Href, urlRoot+"/content/")
				return fmt.Sprintf("%s/viewer#%s", urlRoot, contentPath), nil
			}
		}
	}

	return urlRoot + "/", nil
}
