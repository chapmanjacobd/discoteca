package server_test

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/server"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func serverPort(t *testing.T, server *httptest.Server) int {
	t.Helper()
	port, err := strconv.Atoi(strings.TrimPrefix(server.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatalf("failed to parse mock server port: %v", err)
	}
	return port
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func insertZimMedia(t *testing.T, dbPath, zimPath string) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO media (path, media_type, time_deleted) VALUES (?, 'text', 0)`,
		zimPath,
	); err != nil {
		t.Fatal(err)
	}
}

func TestHandleZimProxyUsesConfiguredPortOnly(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock kiwix response"))
	}))
	defer targetServer.Close()
	port := serverPort(t, targetServer)

	cmd := &server.Server{
		Databases: []string{"/tmp/unused.db"},
		KiwixHost: "127.0.0.1",
		KiwixPort: port,
	}
	defer cmd.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/zim/proxy/{rest...}", cmd.HandleZimProxy)

	// A request-supplied port must not redirect the proxy to another target.
	req := httptest.NewRequest(http.MethodGet, "/api/zim/proxy/9999/content", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from the configured target, got %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != "mock kiwix response" {
		t.Fatalf("unexpected proxy response %q", w.Body.String())
	}
}

func TestHandleZimProxyDisabled(t *testing.T) {
	cmd := &server.Server{Databases: []string{"/tmp/unused.db"}}
	req := httptest.NewRequest(http.MethodGet, "/api/zim/proxy/content", nil)
	w := httptest.NewRecorder()
	cmd.HandleZimProxy(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when ZIM is disabled, got %d", w.Code)
	}
}

func TestHandleZimView(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	zimPath := filepath.Join(fixture.TempDir, "test.zim")
	if err := os.WriteFile(zimPath, []byte("dummy zim content"), 0o644); err != nil {
		t.Fatal(err)
	}
	insertZimMedia(t, fixture.DBPath, zimPath)

	catalogXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Test ZIM</title>
    <name>test-zim</name>
    <link rel="alternate" href="/api/zim/proxy/content/test-zim" type="text/html"/>
  </entry>
</feed>`

	kiwixServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/catalog/v2/entries") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(catalogXML))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer kiwixServer.Close()

	cmd := &server.Server{
		Databases: []string{fixture.DBPath},
		KiwixHost: "127.0.0.1",
		KiwixPort: serverPort(t, kiwixServer),
	}
	defer cmd.Close()

	t.Run("valid listed ZIM returns HTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+zimPath, nil)
		w := httptest.NewRecorder()
		cmd.HandleZimView(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d - %s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, "<iframe") {
			t.Error("expected an iframe")
		}
		if !strings.Contains(body, "test.zim") {
			t.Error("expected the ZIM file name")
		}
		if !strings.Contains(body, "/api/zim/proxy/viewer#test-zim") {
			t.Errorf("expected the configured proxy viewer URL, got %s", body)
		}
	})

	t.Run("missing path returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view", nil)
		w := httptest.NewRecorder()
		cmd.HandleZimView(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("non-zim path returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path=/tmp/test.mp4", nil)
		w := httptest.NewRecorder()
		cmd.HandleZimView(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unlisted ZIM returns 403", func(t *testing.T) {
		other := filepath.Join(fixture.TempDir, "other.zim")
		if err := os.WriteFile(other, []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+other, nil)
		w := httptest.NewRecorder()
		cmd.HandleZimView(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("unreachable kiwix returns 503", func(t *testing.T) {
		unreachable := &server.Server{
			Databases: []string{fixture.DBPath},
			KiwixHost: "127.0.0.1",
			KiwixPort: freePort(t),
		}
		defer unreachable.Close()
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+zimPath, nil)
		w := httptest.NewRecorder()
		unreachable.HandleZimView(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "not reachable") {
			t.Errorf("expected a clear unreachable message, got %q", w.Body.String())
		}
	})

	t.Run("disabled mode returns 503", func(t *testing.T) {
		disabled := &server.Server{Databases: []string{fixture.DBPath}}
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+zimPath, nil)
		w := httptest.NewRecorder()
		disabled.HandleZimView(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", w.Code)
		}
	})
}

func writeZimFile(t *testing.T, path string) string {
	t.Helper()
	header := make([]byte, 24, 28)
	copy(header, "ZIM\x04")
	// UUID bytes must be 11111111-2222-3333-4444-555555555555.
	copy(header[8:], []byte{
		0x11, 0x11, 0x11, 0x11, 0x22, 0x22, 0x33, 0x33,
		0x44, 0x44, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
	})
	header = append(header, []byte("body")...)
	if err := os.WriteFile(path, header, 0o644); err != nil {
		t.Fatal(err)
	}
	return "11111111-2222-3333-4444-555555555555"
}

func TestHandleZimProxyZimanticStripsProxyPrefix(t *testing.T) {
	var gotPath string
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock zimantic response"))
	}))
	defer targetServer.Close()

	cmd := &server.Server{
		Databases:    []string{"/tmp/unused.db"},
		KiwixPort:    8181,
		ZimanticHost: "127.0.0.1",
		ZimanticPort: serverPort(t, targetServer),
	}
	defer cmd.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/zim/proxy/{rest...}", cmd.HandleZimProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/zim/proxy/content/zim%3Atest%3Abook/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from zimantic, got %d", w.Code)
	}
	if gotPath != "/content/zim:test:book/" {
		t.Fatalf("expected the proxy prefix stripped, got %q", gotPath)
	}
}

func TestHandleZimViewZimantic(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	zimPath := filepath.Join(fixture.TempDir, "testy_2024-01.zim")
	uuid := writeZimFile(t, zimPath)
	insertZimMedia(t, fixture.DBPath, zimPath)

	libraryJSON := fmt.Sprintf(`[
  {"key": "zim:library:testy-abcdef12", "provider": "zim", "title": "Testy", "provider_id": %q}
]`, uuid)

	zimanticServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/library.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Total-Count", "1")
		_, _ = w.Write([]byte(libraryJSON))
	}))
	defer zimanticServer.Close()

	cmd := &server.Server{
		Databases:    []string{fixture.DBPath},
		ZimanticHost: "127.0.0.1",
		ZimanticPort: serverPort(t, zimanticServer),
	}
	defer cmd.Close()

	t.Run("matching UUID returns the zimantic viewer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+zimPath, nil)
		w := httptest.NewRecorder()
		cmd.HandleZimView(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d - %s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, "<iframe") {
			t.Error("expected an iframe")
		}
		if !strings.Contains(body, "/api/zim/proxy/content/zim%3Alibrary%3Atesty-abcdef12/") {
			t.Errorf("expected the zimantic content URL, got %s", body)
		}
	})

	t.Run("unknown ZIM returns 404", func(t *testing.T) {
		libraryJSON = "[]"
		other := filepath.Join(fixture.TempDir, "other.zim")
		writeZimFile(t, other)
		insertZimMedia(t, fixture.DBPath, other)
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+other, nil)
		w := httptest.NewRecorder()
		cmd.HandleZimView(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d - %s", w.Code, w.Body.String())
		}
	})

	t.Run("unreachable zimantic returns 503", func(t *testing.T) {
		unreachable := &server.Server{
			Databases:    []string{fixture.DBPath},
			ZimanticHost: "127.0.0.1",
			ZimanticPort: freePort(t),
		}
		defer unreachable.Close()
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+zimPath, nil)
		w := httptest.NewRecorder()
		unreachable.HandleZimView(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "zimantic is not reachable") {
			t.Errorf("expected a clear unreachable message, got %q", w.Body.String())
		}
	})
}

func TestZimProxyNeverSpawnsAProcess(t *testing.T) {
	// This is a regression guard for the managed-process design: the handler
	// is entirely HTTP-based, so a request to an unreachable upstream returns
	// 503 rather than trying to start kiwix-serve.
	cmd := &server.Server{
		Databases: []string{"/tmp/unused.db"},
		KiwixHost: "127.0.0.1",
		KiwixPort: freePort(t),
	}
	defer cmd.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/zim/proxy/content", nil)
	w := httptest.NewRecorder()
	cmd.HandleZimProxy(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for unreachable upstream, got %d", w.Code)
	}
}
