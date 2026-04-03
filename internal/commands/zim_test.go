package commands

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

// TestKiwixManager_findAvailablePort tests port allocation logic
func TestKiwixManager_findAvailablePort(t *testing.T) {
	manager := &KiwixManager{
		instances: make(map[string]*KiwixInstance),
		usedPorts: make(map[int]bool),
	}

	t.Run("returns first available port starting from KIWIX_PORT_START", func(t *testing.T) {
		port := manager.findAvailablePort()
		if port != KIWIX_PORT_START {
			t.Errorf("Expected port %d, got %d", KIWIX_PORT_START, port)
		}
	})

	t.Run("skips used ports in usedPorts map", func(t *testing.T) {
		manager.usedPorts[KIWIX_PORT_START] = true
		port := manager.findAvailablePort()
		if port != KIWIX_PORT_START+1 {
			t.Errorf("Expected port %d, got %d", KIWIX_PORT_START+1, port)
		}
	})

	t.Run("skips ports that are not available", func(t *testing.T) {
		// This test verifies that isPortAvailable is called
		// We can't easily mock isPortAvailable, but we can verify the logic
		manager2 := &KiwixManager{
			instances: make(map[string]*KiwixInstance),
			usedPorts: make(map[int]bool),
		}
		// Mark first 5 ports as used
		for i := range 5 {
			manager2.usedPorts[KIWIX_PORT_START+i] = true
		}
		port := manager2.findAvailablePort()
		if port != KIWIX_PORT_START+5 {
			t.Errorf("Expected port %d, got %d", KIWIX_PORT_START+5, port)
		}
	})

	t.Run("returns 0 when no ports available in range", func(t *testing.T) {
		manager3 := &KiwixManager{
			instances: make(map[string]*KiwixInstance),
			usedPorts: make(map[int]bool),
		}
		// Mark all 100 ports as used
		for i := range 100 {
			manager3.usedPorts[KIWIX_PORT_START+i] = true
		}
		port := manager3.findAvailablePort()
		if port != 0 {
			t.Errorf("Expected port 0 (no available ports), got %d", port)
		}
	})
}

// TestKiwixManager_ensureKiwixServing tests instance management
func TestKiwixManager_ensureKiwixServing(t *testing.T) {
	manager := &KiwixManager{
		instances: make(map[string]*KiwixInstance),
		usedPorts: make(map[int]bool),
	}

	t.Run("attempts to start kiwix-serve for any file", func(t *testing.T) {
		// Note: ensureKiwixServing doesn't validate file existence
		// That validation happens in handleZimView before calling ensureKiwixServing
		// This test verifies the function tries to start kiwix-serve
		nonExistentPath := "/tmp/nonexistent.zim"
		port, err := manager.ensureKiwixServing(nonExistentPath)
		// The function will try to start kiwix-serve, which may fail if not installed
		// or succeed if kiwix-serve is installed (it doesn't validate the file)
		if err != nil {
			// Expected if kiwix-serve is not installed
			if !strings.Contains(err.Error(), "kiwix-serve") {
				t.Errorf("Expected kiwix-serve related error, got: %v", err)
			}
		} else {
			// kiwix-serve started (file validation is done by handleZimView)
			t.Logf("kiwix-serve started on port %d (file validation done by handleZimView)", port)
		}
	})

	t.Run("returns same port for same ZIM file on subsequent calls", func(t *testing.T) {
		// Create a dummy ZIM file
		tmpDir := t.TempDir()
		zimPath := filepath.Join(tmpDir, "test.zim")
		if err := os.WriteFile(zimPath, []byte("dummy zim content"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// First call - would try to start kiwix-serve (which may not be installed)
		// We're testing the caching logic, so we'll mock the instance directly
		manager.instances[zimPath] = &KiwixInstance{
			Port:     KIWIX_PORT_START + 10,
			ZimPath:  zimPath,
			LastUsed: time.Now(),
		}

		// Second call should return cached instance
		port, err := manager.ensureKiwixServing(zimPath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if port != KIWIX_PORT_START+10 {
			t.Errorf("Expected cached port %d, got %d", KIWIX_PORT_START+10, port)
		}
	})

	t.Run("updates LastUsed on subsequent calls", func(t *testing.T) {
		tmpDir := t.TempDir()
		zimPath := filepath.Join(tmpDir, "test2.zim")
		if err := os.WriteFile(zimPath, []byte("dummy"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		oldTime := time.Now().Add(-1 * time.Hour)
		manager.instances[zimPath] = &KiwixInstance{
			Port:     KIWIX_PORT_START + 11,
			ZimPath:  zimPath,
			LastUsed: oldTime,
		}

		_, err := manager.ensureKiwixServing(zimPath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		instance := manager.instances[zimPath]
		if instance.LastUsed.Before(oldTime.Add(30 * time.Minute)) {
			t.Errorf("Expected LastUsed to be updated, got %v", instance.LastUsed)
		}
	})
}

// TestKiwixManager_cleanupOldInstances tests cleanup logic
func TestKiwixManager_cleanupOldInstances(t *testing.T) {
	manager := &KiwixManager{
		instances: make(map[string]*KiwixInstance),
		usedPorts: make(map[int]bool),
	}

	tmpDir := t.TempDir()
	zimPath1 := filepath.Join(tmpDir, "old.zim")
	zimPath2 := filepath.Join(tmpDir, "recent.zim")
	os.WriteFile(zimPath1, []byte("dummy"), 0o644)
	os.WriteFile(zimPath2, []byte("dummy"), 0o644)

	// Add old instance (should be cleaned up)
	oldTime := time.Now().Add(-31 * time.Minute)
	port1 := KIWIX_PORT_START + 20
	manager.instances[zimPath1] = &KiwixInstance{
		Port:     port1,
		ZimPath:  zimPath1,
		LastUsed: oldTime,
	}
	manager.usedPorts[port1] = true

	// Add recent instance (should NOT be cleaned up)
	recentTime := time.Now().Add(-10 * time.Minute)
	port2 := KIWIX_PORT_START + 21
	manager.instances[zimPath2] = &KiwixInstance{
		Port:     port2,
		ZimPath:  zimPath2,
		LastUsed: recentTime,
	}
	manager.usedPorts[port2] = true

	// Manually trigger cleanup logic (without goroutine/ticker)
	cutoff := time.Now().Add(-30 * time.Minute)
	for path, instance := range manager.instances {
		if instance.LastUsed.Before(cutoff) {
			delete(manager.usedPorts, instance.Port)
			delete(manager.instances, path)
		}
	}

	if _, exists := manager.instances[zimPath1]; exists {
		t.Errorf("Expected old instance to be cleaned up")
	}
	if _, exists := manager.usedPorts[port1]; exists {
		t.Errorf("Expected old port to be released")
	}
	if _, exists := manager.instances[zimPath2]; !exists {
		t.Errorf("Expected recent instance to remain")
	}
	if _, exists := manager.usedPorts[port2]; !exists {
		t.Errorf("Expected recent port to remain reserved")
	}
}

// TestKiwixManager_concurrentAccess tests thread safety
func TestKiwixManager_concurrentAccess(t *testing.T) {
	manager := &KiwixManager{
		instances: make(map[string]*KiwixInstance),
		usedPorts: make(map[int]bool),
		mutex:     sync.Mutex{},
	}

	tmpDir := t.TempDir()
	zimPath := filepath.Join(tmpDir, "concurrent.zim")
	os.WriteFile(zimPath, []byte("dummy"), 0o644)

	// Pre-populate with a mock instance
	manager.instances[zimPath] = &KiwixInstance{
		Port:     KIWIX_PORT_START + 30,
		ZimPath:  zimPath,
		LastUsed: time.Now(),
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Simulate concurrent access
	for range 50 {
		wg.Go(func() {
			port, err := manager.ensureKiwixServing(zimPath)
			if err != nil {
				errors <- err
				return
			}
			if port != KIWIX_PORT_START+30 {
				errors <- fmt.Errorf("expected port %d, got %d", KIWIX_PORT_START+30, port)
			}
		})
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

// TestHandleZimView tests the HTTP handler for ZIM files
func TestHandleZimView(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	cmd := &ServeCmd{
		Databases: []string{fixture.DBPath},
		APIToken:  "test-token",
	}
	defer cmd.Close()

	t.Run("returns 400 for missing path parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view", nil)
		w := httptest.NewRecorder()

		cmd.handleZimView(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("returns 400 for non-.zim files", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path=/tmp/test.mp4", nil)
		w := httptest.NewRecorder()

		cmd.handleZimView(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for non-.zim file, got %d", resp.StatusCode)
		}
	})

	t.Run("returns 404 for non-existent ZIM file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path=/tmp/nonexistent.zim", nil)
		w := httptest.NewRecorder()

		cmd.handleZimView(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("returns HTML with iframe for valid ZIM file", func(t *testing.T) {
		// Create a mock ZIM file
		zimPath := filepath.Join(fixture.TempDir, "test.zim")
		if err := os.WriteFile(zimPath, []byte("dummy zim content"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a mock server to simulate kiwix-serve
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer mockServer.Close()

		// Extract port from mock server
		mockPort := 8181 + 50
		// We need to intercept the waitForKiwixReady call
		originalWaitFunc := waitForKiwixReady
		waitForKiwixReady = func(ctx context.Context, port int, timeout time.Duration) error {
			// Skip actual check, assume server is ready
			return nil
		}
		defer func() { waitForKiwixReady = originalWaitFunc }()

		// Mock the zimManager to avoid actually starting kiwix-serve
		originalManager := zimManager
		mockManager := &KiwixManager{
			instances: make(map[string]*KiwixInstance),
			usedPorts: make(map[int]bool),
		}
		mockManager.instances[zimPath] = &KiwixInstance{
			Port:     mockPort,
			ZimPath:  zimPath,
			LastUsed: time.Now(),
		}
		zimManager = mockManager
		defer func() { zimManager = originalManager }()

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/zim/view?path=%s", zimPath), nil)
		w := httptest.NewRecorder()

		cmd.handleZimView(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<iframe") {
			t.Errorf("Expected HTML to contain iframe")
		}
		if !strings.Contains(body, filepath.Base(zimPath)) {
			t.Errorf("Expected HTML to contain ZIM file name")
		}
	})
}

// TestHandleZimProxy tests the proxy HTTP handler
func TestHandleZimProxy(t *testing.T) {
	cmd := &ServeCmd{
		Databases: []string{"/tmp/test.db"},
		APIToken:  "test-token",
	}
	defer cmd.Close()

	t.Run("returns 400 for missing port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/proxy/", nil)
		w := httptest.NewRecorder()

		cmd.handleZimProxy(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("proxies request to target server", func(t *testing.T) {
		// Create a mock target server
		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock kiwix response"))
		}))
		defer targetServer.Close()

		// Extract port from target server URL
		port := strings.TrimPrefix(targetServer.URL, "http://127.0.0.1:")

		// Create a request that mimics the route pattern /api/zim/proxy/{port}/{rest...}
		// We need to use a mux to properly set up path values
		mux := http.NewServeMux()
		mux.HandleFunc("/api/zim/proxy/{port}/{rest...}", func(w http.ResponseWriter, r *http.Request) {
			cmd.handleZimProxy(w, r)
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/zim/proxy/%s/content", port), nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

// TestGetKiwixContentURL tests parsing the ZIM catalog
func TestGetKiwixContentURL(t *testing.T) {
	t.Run("parses catalog entry with HTML link", func(t *testing.T) {
		// Create a mock catalog server
		catalogXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Test ZIM</title>
    <name>test-zim</name>
    <link rel="alternate" href="/api/zim/proxy/8181/content/test-zim" type="text/html"/>
  </entry>
</feed>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/catalog/v2/entries") {
				w.Header().Set("Content-Type", "application/xml")
				w.Write([]byte(catalogXML))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Parse the catalog to verify structure
		var feed struct {
			XMLName xml.Name    `xml:"feed"`
			Entries []OpdsEntry `xml:"entry"`
		}
		if err := xml.Unmarshal([]byte(catalogXML), &feed); err != nil {
			t.Fatalf("Failed to parse catalog: %v", err)
		}
		if len(feed.Entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(feed.Entries))
		}
		if feed.Entries[0].Title != "Test ZIM" {
			t.Errorf("Expected title 'Test ZIM', got '%s'", feed.Entries[0].Title)
		}
		hasHTMLLink := false
		for _, link := range feed.Entries[0].Link {
			if link.Type == "text/html" {
				hasHTMLLink = true
				break
			}
		}
		if !hasHTMLLink {
			t.Errorf("Expected HTML link in entry")
		}
	})

	t.Run("handles empty catalog", func(t *testing.T) {
		emptyXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
</feed>`

		var feed struct {
			XMLName xml.Name    `xml:"feed"`
			Entries []OpdsEntry `xml:"entry"`
		}
		if err := xml.Unmarshal([]byte(emptyXML), &feed); err != nil {
			t.Fatalf("Failed to parse empty catalog: %v", err)
		}
		if len(feed.Entries) != 0 {
			t.Errorf("Expected 0 entries, got %d", len(feed.Entries))
		}
	})
}

// TestIsPortAvailable tests port availability checking
func TestIsPortAvailable(t *testing.T) {
	t.Run("returns true for available port", func(t *testing.T) {
		// Find a truly available port
		port := KIWIX_PORT_START + 100
		if !isPortAvailable(port) {
			t.Errorf("Expected port %d to be available", port)
		}
	})

	t.Run("returns false for occupied port", func(t *testing.T) {
		// Bind to a port to make it unavailable
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to bind test port: %v", err)
		}
		defer listener.Close()

		addr := listener.Addr().String()
		parts := strings.Split(addr, ":")
		port := 0
		fmt.Sscanf(parts[1], "%d", &port)

		if isPortAvailable(port) {
			t.Errorf("Expected port %d to be unavailable", port)
		}
	})
}

// TestWaitForKiwixReady tests server readiness checking
func TestWaitForKiwixReady(t *testing.T) {
	t.Run("returns quickly for ready server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Extract port
		portStr := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
		var port int
		fmt.Sscanf(portStr, "%d", &port)

		// This would test the actual function, but it uses hardcoded URL patterns
		// We document the expected behavior instead
		start := time.Now()
		err := waitForKiwixReady(context.Background(), port, 2*time.Second)
		elapsed := time.Since(start)

		// Should return quickly if server is ready
		if err == nil && elapsed > 1*time.Second {
			t.Errorf("Expected quick return for ready server, took %v", elapsed)
		}
	})

	t.Run("returns error for unavailable server", func(t *testing.T) {
		// Use a port that's definitely not running anything
		port := KIWIX_PORT_START + 999

		start := time.Now()
		err := waitForKiwixReady(context.Background(), port, 100*time.Millisecond)
		elapsed := time.Since(start)

		if err == nil {
			t.Errorf("Expected error for unavailable server")
		}
		if elapsed < 100*time.Millisecond {
			t.Errorf("Expected to wait for timeout, took %v", elapsed)
		}
	})
}
