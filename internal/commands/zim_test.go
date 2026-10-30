package commands_test

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
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

	cmd := &commands.ServeCmd{
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
	cmd := &commands.ServeCmd{Databases: []string{"/tmp/unused.db"}}
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

	cmd := &commands.ServeCmd{
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
		unreachable := &commands.ServeCmd{
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
		disabled := &commands.ServeCmd{Databases: []string{fixture.DBPath}}
		req := httptest.NewRequest(http.MethodGet, "/api/zim/view?path="+zimPath, nil)
		w := httptest.NewRecorder()
		disabled.HandleZimView(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", w.Code)
		}
	})
}

func TestZimProxyNeverSpawnsAProcess(t *testing.T) {
	// This is a regression guard for the managed-process design: the handler
	// is entirely HTTP-based, so a request to an unreachable upstream returns
	// 503 rather than trying to start kiwix-serve.
	cmd := &commands.ServeCmd{
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
