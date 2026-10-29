package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/web"
)

// basePathKey stores the effective URL prefix for the current request.
type basePathKey struct{}

// BasePathFromContext returns the URL prefix (for example "/disco") that the
// current request was mounted under. It returns "" when discoteca is served at
// the root of the host.
func BasePathFromContext(ctx context.Context) string {
	prefix, _ := ctx.Value(basePathKey{}).(string)
	return prefix
}

func withBasePath(ctx context.Context, prefix string) context.Context {
	if prefix == "" {
		return ctx
	}
	return context.WithValue(ctx, basePathKey{}, prefix)
}

// normalizeBasePath turns user or proxy input into a clean prefix such as
// "/disco". Invalid values return "" (serve at the root).
func normalizeBasePath(raw string) string {
	// X-Forwarded-Prefix may contain a comma-separated chain. Use the first hop.
	raw = strings.TrimSpace(strings.Split(raw, ",")[0])
	if raw == "" || raw == "/" {
		return ""
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	if slices.Contains(strings.Split(raw, "/"), "..") {
		return ""
	}
	clean := path.Clean(raw)
	if clean == "/" || clean == "." || clean == ".." || strings.HasPrefix(clean, "..") {
		return ""
	}
	for _, r := range clean {
		// Reject control characters, backslashes, and quotes. These can only
		// come from a malformed proxy header and are unsafe in redirects.
		if r < 0x20 || r == 0x7f || r == '\\' || r == '"' || r == '\'' {
			return ""
		}
	}
	return clean
}

// effectiveBasePath returns the prefix to use for this request. An explicitly
// configured --base-path always wins. Otherwise X-Forwarded-Prefix is honored,
// which lets nginx either strip the prefix (proxy_pass with a trailing slash)
// or pass it through unchanged.
func (c *ServeCmd) effectiveBasePath(r *http.Request) (prefix string, fromForwardedHeader bool) {
	if configured := normalizeBasePath(c.BasePath); configured != "" {
		return configured, false
	}
	header := r.Header.Get("X-Forwarded-Prefix")
	if header == "" {
		return "", false
	}
	prefix = normalizeBasePath(header)
	return prefix, prefix != ""
}

// withBasePath mounts the mux below the configured (or forwarded) URL prefix.
func (c *ServeCmd) withBasePath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix, fromForwardedHeader := c.effectiveBasePath(r)
		if prefix == "" {
			next.ServeHTTP(w, r)
			return
		}

		requestPath := r.URL.Path

		switch {
		case requestPath == prefix:
			redirectBasePath(w, r, prefix+"/")
			return

		case strings.HasPrefix(requestPath, prefix+"/"):
			stripped := strings.TrimPrefix(requestPath, prefix)
			if stripped == "" {
				stripped = "/"
			}
			r = cloneRequestWithPath(r, stripped)
			r = r.WithContext(withBasePath(r.Context(), prefix))
			next.ServeHTTP(w, r)
			return

		case fromForwardedHeader:
			// The proxy already stripped the prefix before forwarding the
			// request. Keep the prefix for URL generation only.
			r = r.WithContext(withBasePath(r.Context(), prefix))
			next.ServeHTTP(w, r)
			return

		case requestPath == "/":
			redirectBasePath(w, r, prefix+"/")
			return

		default:
			http.NotFound(w, r)
			return
		}
	})
}

func redirectBasePath(w http.ResponseWriter, r *http.Request, target string) {
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	code := http.StatusPermanentRedirect
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		code = http.StatusMovedPermanently
	}
	http.Redirect(w, r, target, code)
}

func cloneRequestWithPath(r *http.Request, requestPath string) *http.Request {
	clone := r.Clone(r.Context())
	clone.URL.Path = requestPath
	if r.URL.RawPath != "" {
		// RawPath is only a hint. Dropping it keeps EscapedPath consistent
		// with the stripped Path.
		clone.URL.RawPath = ""
	}
	return clone
}

// injectBasePath adds the runtime prefix that the web client uses to build API
// and media URLs.
func injectBasePath(html []byte, prefix string) []byte {
	if prefix == "" {
		return html
	}
	encoded, err := json.Marshal(prefix)
	if err != nil {
		return html
	}
	tag := make([]byte, 0, len(encoded)+48)
	tag = append(tag, "<script>window.__DISCO_BASE_PATH__="...)
	tag = append(tag, encoded...)
	tag = append(tag, ";</script>"...)

	lower := bytes.ToLower(html)
	idx := bytes.Index(lower, []byte("</head>"))
	if idx < 0 {
		return append(tag, html...)
	}
	out := make([]byte, 0, len(html)+len(tag))
	out = append(out, html[:idx]...)
	out = append(out, tag...)
	out = append(out, html[idx:]...)
	return out
}

// serveIndexHTML writes index.html with the base path injected. It returns
// false when the index cannot be read, so the caller can fall back to the
// regular file server.
func (c *ServeCmd) serveIndexHTML(w http.ResponseWriter, r *http.Request, prefix string) bool {
	var data []byte
	var err error
	if c.PublicDir != "" {
		data, err = os.ReadFile(filepath.Join(c.PublicDir, "index.html"))
	} else {
		data, err = fs.ReadFile(web.FS, "index.html")
	}
	if err != nil {
		return false
	}

	data = injectBasePath(data, prefix)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(data))
	return true
}
