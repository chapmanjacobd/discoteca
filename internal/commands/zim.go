package commands

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
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

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

const (
	KIWIX_BIN        = "kiwix-serve"
	KIWIX_PORT_START = 8181
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
	instances map[string]*KiwixInstance // zimPath -> instance
	mutex     sync.Mutex
	usedPorts map[int]bool
}

var zimManager = &KiwixManager{
	instances: make(map[string]*KiwixInstance),
	usedPorts: make(map[int]bool),
}

func init() {
	go zimManager.cleanupOldInstances()
}

func (c *ServeCmd) handleZimProxy(w http.ResponseWriter, r *http.Request) {
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

func (c *ServeCmd) handleZimView(w http.ResponseWriter, r *http.Request) {
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

	port, err := zimManager.ensureKiwixServing(localPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := waitForKiwixReady(r.Context(), port, 10*time.Second); err != nil {
		http.Error(w, fmt.Sprintf("Kiwix server did not start in time: %s", err.Error()), http.StatusServiceUnavailable)
		return
	}

	contentURL, err := getKiwixContentURL(r.Context(), port)
	if err != nil {
		slog.Warn("Could not parse ZIM catalog, using root URL", "error", err)
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
	w.Write([]byte(html))
}

func (m *KiwixManager) ensureKiwixServing(zimPath string) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if instance, exists := m.instances[zimPath]; exists {
		instance.LastUsed = time.Now()
		return instance.Port, nil
	}

	port := m.findAvailablePort()
	if port == 0 {
		return 0, errors.New("no available ports for kiwix-serve")
	}

	cmd := exec.Command(
		KIWIX_BIN,
		"--nolibrarybutton",
		"-p", strconv.Itoa(port),
		fmt.Sprintf("--urlRootLocation=/api/zim/proxy/%d/", port),
		zimPath,
	)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start kiwix-serve: %w", err)
	}

	m.instances[zimPath] = &KiwixInstance{
		Process:  cmd,
		Port:     port,
		ZimPath:  zimPath,
		LastUsed: time.Now(),
	}
	m.usedPorts[port] = true

	slog.Info("Started kiwix-serve", "port", port, "path", zimPath)
	return port, nil
}

func (m *KiwixManager) findAvailablePort() int {
	for i := range 100 {
		port := KIWIX_PORT_START + i
		if !m.usedPorts[port] && isPortAvailable(port) {
			return port
		}
	}
	return 0
}

func (m *KiwixManager) cleanupOldInstances() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.mutex.Lock()
		cutoff := time.Now().Add(-30 * time.Minute)
		for path, instance := range m.instances {
			if instance.LastUsed.Before(cutoff) {
				slog.Info("Cleaning up unused kiwix-serve instance", "port", instance.Port, "path", path)
				if instance.Process.Process != nil {
					instance.Process.Process.Kill()
				}
				instance.Process.Wait()
				delete(m.usedPorts, instance.Port)
				delete(m.instances, path)
			}
		}
		m.mutex.Unlock()
	}
}

func isPortAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

var waitForKiwixReady = defaultWaitForKiwixReady

func defaultWaitForKiwixReady(ctx context.Context, port int, timeout time.Duration) error {
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
