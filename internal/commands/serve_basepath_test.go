package commands_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

func TestMuxBasePath(t *testing.T) {
	cmd := &commands.ServeCmd{BasePath: "/disco"}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("RedirectsBarePrefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disco", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Fatalf("expected 301, got %d", w.Code)
		}
		if got := w.Header().Get("Location"); got != "/disco/" {
			t.Errorf("expected Location /disco/, got %q", got)
		}
	})

	t.Run("RedirectsRootToPrefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Fatalf("expected 301, got %d", w.Code)
		}
		if got := w.Header().Get("Location"); got != "/disco/?foo=bar" {
			t.Errorf("expected Location /disco/?foo=bar, got %q", got)
		}
	})

	t.Run("ServesAPIUnderPrefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disco/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if strings.TrimSpace(w.Body.String()) != "OK" {
			t.Errorf("expected OK body, got %q", w.Body.String())
		}
	})

	t.Run("HidesAPIAtRoot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 outside prefix, got %d", w.Code)
		}
	})

	t.Run("InjectsBasePathIntoIndex", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disco/", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, `window.__DISCO_BASE_PATH__="/disco"`) {
			t.Error("index.html is missing the injected base path")
		}
		if !strings.Contains(body, `src="./index.js"`) {
			t.Error("index.html should use relative asset URLs")
		}
	})

	t.Run("ScopesAuthCookieToPrefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disco/", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		cookies := w.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected a cookie")
		}
		if got := cookies[0].Path; got != "/disco/" {
			t.Errorf("expected cookie path /disco/, got %q", got)
		}
	})
}

func TestMuxForwardedPrefix(t *testing.T) {
	cmd := &commands.ServeCmd{}
	defer cmd.Close()
	mux := cmd.Mux()

	t.Run("PrefixStillInPath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disco/health", nil)
		req.Header.Set("X-Forwarded-Prefix", "/disco")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("PrefixStrippedByProxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Forwarded-Prefix", "/disco")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("IndexUsesForwardedPrefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Prefix", "/disco")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `window.__DISCO_BASE_PATH__="/disco"`) {
			t.Error("index.html is missing the forwarded base path")
		}
	})

	t.Run("MalformedHeaderIgnored", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Forwarded-Prefix", "//evil.example")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestServeCmdClampsRequestedLimit(t *testing.T) {
	cmd := &commands.ServeCmd{MaxLimit: 50}

	req := httptest.NewRequest(http.MethodGet, "/api/query?limit=5000&offset=-3", nil)
	flags := cmd.ParseFlags(req)

	if flags.Limit != 50 {
		t.Errorf("expected limit to be clamped to 50, got %d", flags.Limit)
	}
	if flags.Offset != 0 {
		t.Errorf("expected negative offset to be ignored, got %d", flags.Offset)
	}

	unclamped := &commands.ServeCmd{}
	flags = unclamped.ParseFlags(httptest.NewRequest(http.MethodGet, "/api/query?limit=5000", nil))
	if flags.Limit != 5000 {
		t.Errorf("expected no clamp without MaxLimit, got %d", flags.Limit)
	}
}
