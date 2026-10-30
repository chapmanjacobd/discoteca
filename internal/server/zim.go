package server

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
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

// validateKiwixConfig checks the externally managed kiwix-serve configuration.
// discoteca never starts, stops, or waits for that process.
func (c *Server) validateKiwixConfig() error {
	if c.KiwixPort < 0 || c.KiwixPort > 65535 {
		return fmt.Errorf("invalid --kiwix-port %d: must be between 0 and 65535", c.KiwixPort)
	}
	return nil
}

func (c *Server) kiwixHost() string {
	if strings.TrimSpace(c.KiwixHost) == "" {
		return "127.0.0.1"
	}
	return c.KiwixHost
}

func (c *Server) kiwixAddress() string {
	return net.JoinHostPort(c.kiwixHost(), strconv.Itoa(c.KiwixPort))
}

func (c *Server) kiwixConfigured() bool {
	return c.KiwixPort > 0
}

func kiwixRootPath(basePath string) string {
	return basePath + "/api/zim/proxy"
}

// HandleZimProxy forwards to exactly the configured kiwix-serve host and port.
// The request path is never used to choose a proxy target.
func (c *Server) HandleZimProxy(w http.ResponseWriter, r *http.Request) {
	if !c.kiwixConfigured() {
		http.Error(w, "ZIM support disabled", http.StatusServiceUnavailable)
		return
	}

	target, err := url.Parse("http://" + c.kiwixAddress())
	if err != nil {
		http.Error(w, "Invalid kiwix-serve configuration", http.StatusInternalServerError)
		return
	}

	// kiwix-serve is started with --urlRootLocation set to the external path
	// (including any base path), so restore the prefix that the base-path
	// middleware stripped before forwarding.
	if basePath := BasePathFromContext(r.Context()); basePath != "" {
		r = cloneRequestWithPath(r, basePath+r.URL.Path)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		logging.Log.Warn("kiwix-serve is not reachable", "error", err)
		http.Error(w, "kiwix-serve is not reachable", http.StatusServiceUnavailable)
	}
	proxy.ServeHTTP(w, r)
}

func (c *Server) HandleZimView(w http.ResponseWriter, r *http.Request) {
	if !c.kiwixConfigured() {
		http.Error(w, "ZIM support disabled", http.StatusServiceUnavailable)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	if !strings.HasSuffix(strings.ToLower(path), ".zim") {
		http.Error(w, "Not a .zim file", http.StatusBadRequest)
		return
	}
	if c.isPathBlocklisted(path) {
		http.Error(w, "Access denied: sensitive path", http.StatusForbidden)
		return
	}
	if !fileutil.FileExists(path) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if found, _ := c.getMediaTypeFromDB(r.Context(), path); !found {
		http.Error(w, "ZIM file not found in database", http.StatusForbidden)
		return
	}

	basePath := BasePathFromContext(r.Context())
	contentURL, err := c.getKiwixContentURL(r.Context(), basePath, path)
	if err != nil {
		logging.Log.Warn("kiwix-serve is not reachable", "error", err)
		http.Error(w, "kiwix-serve is not reachable", http.StatusServiceUnavailable)
		return
	}

	zimName := filepath.Base(path)
	htmlDocument := fmt.Sprintf(`<!DOCTYPE html>
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
</html>`, html.EscapeString(zimName), html.EscapeString(contentURL))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(htmlDocument))
}

// getKiwixContentURL reads the external kiwix-serve catalog and returns the
// viewer URL for the requested ZIM. If the catalog does not expose a matching
// title or path, it falls back to the first usable entry.
func (c *Server) getKiwixContentURL(ctx context.Context, basePath, zimPath string) (string, error) {
	urlRoot := kiwixRootPath(basePath)
	catalogURL := fmt.Sprintf("http://%s%s/catalog/v2/entries", c.kiwixAddress(), urlRoot)

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

	target := strings.TrimSuffix(strings.ToLower(filepath.Base(zimPath)), ".zim")
	fallback := ""
	for _, entry := range feed.Entries {
		for _, link := range entry.Link {
			if link.Type != "text/html" {
				continue
			}
			contentPath := strings.TrimPrefix(link.Href, urlRoot+"/content/")
			candidate := fmt.Sprintf("%s/viewer#%s", urlRoot, contentPath)
			if fallback == "" {
				fallback = candidate
			}
			if strings.Contains(strings.ToLower(entry.Title), target) ||
				strings.Contains(strings.ToLower(contentPath), target) {
				return candidate, nil
			}
		}
	}

	if fallback != "" {
		return fallback, nil
	}
	return urlRoot + "/", nil
}
