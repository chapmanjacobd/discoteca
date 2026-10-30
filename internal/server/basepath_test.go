package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/server"
)

func TestMuxBasePath(t *testing.T) {
	cmd := &server.Server{BasePath: "/disco"}
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

	t.Run("DoesNotSetTokenCookieOnStaticRequest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disco/", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == "disco_token" {
				t.Fatalf("static request must not set an authentication token cookie")
			}
		}
	})
}

func TestMuxForwardedPrefix(t *testing.T) {
	cmd := &server.Server{}
	defer cmd.Close()
	mux := cmd.Mux()

	newForwardedRequest := func(method, target, prefix string) *http.Request {
		req := httptest.NewRequest(method, target, nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("X-Forwarded-Prefix", prefix)
		return req
	}

	t.Run("PrefixStillInPath", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, newForwardedRequest(http.MethodGet, "/disco/health", "/disco"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("PrefixStrippedByProxy", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, newForwardedRequest(http.MethodGet, "/health", "/disco"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("IndexUsesForwardedPrefix", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, newForwardedRequest(http.MethodGet, "/", "/disco"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `window.__DISCO_BASE_PATH__="/disco"`) {
			t.Error("index.html is missing the forwarded base path")
		}
	})

	t.Run("MalformedHeaderIgnored", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, newForwardedRequest(http.MethodGet, "/health", "//evil.example"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestForwardedPrefixTrust(t *testing.T) {
	t.Run("UntrustedNonLoopbackPeer", func(t *testing.T) {
		cmd := &server.Server{}
		defer cmd.Close()
		mux := cmd.Mux()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.10:4567"
		req.Header.Set("X-Forwarded-Prefix", "/disco")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if strings.Contains(w.Body.String(), `window.__DISCO_BASE_PATH__="/disco"`) {
			t.Error("untrusted forwarded prefix must not be injected")
		}
	})

	t.Run("TrustedFlagAllowsRemotePeer", func(t *testing.T) {
		cmd := &server.Server{TrustForwardedPrefix: true}
		defer cmd.Close()
		mux := cmd.Mux()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.10:4567"
		req.Header.Set("X-Forwarded-Prefix", "/disco")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if !strings.Contains(w.Body.String(), `window.__DISCO_BASE_PATH__="/disco"`) {
			t.Error("trusted forwarded prefix should be injected")
		}
	})

	t.Run("RejectsSchemeAndDotDot", func(t *testing.T) {
		cmd := &server.Server{TrustForwardedPrefix: true}
		defer cmd.Close()
		mux := cmd.Mux()

		for _, prefix := range []string{"http://evil.example", "../", "/a/../b", `\evil`} {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "203.0.113.10:4567"
			req.Header.Set("X-Forwarded-Prefix", prefix)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if strings.Contains(w.Body.String(), "window.__DISCO_BASE_PATH__") {
				t.Errorf("prefix %q should have been rejected", prefix)
			}
		}
	})
}

func TestServerClampsRequestedLimit(t *testing.T) {
	cmd := &server.Server{MaxLimit: 50}

	req := httptest.NewRequest(http.MethodGet, "/api/query?limit=5000&offset=-3", nil)
	flags := cmd.ParseFlags(req)

	if flags.Limit != 50 {
		t.Errorf("expected limit to be clamped to 50, got %d", flags.Limit)
	}
	if flags.Offset != 0 {
		t.Errorf("expected negative offset to be ignored, got %d", flags.Offset)
	}

	unclamped := &server.Server{}
	flags = unclamped.ParseFlags(httptest.NewRequest(http.MethodGet, "/api/query?limit=5000", nil))
	if flags.Limit != 5000 {
		t.Errorf("expected no clamp without MaxLimit, got %d", flags.Limit)
	}
}
