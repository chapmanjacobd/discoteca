package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
)

// zimProxyRoot is the discoteca path that fronts the external ZIM server.
const zimProxyRoot = "/api/zim/proxy"

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

// zimanticSource is the subset of a zimantic library record needed to resolve
// a local ZIM file to its source key.
type zimanticSource struct {
	Key        string `json:"key"`
	Title      string `json:"title"`
	ProviderID string `json:"provider_id"`
}

// validateKiwixConfig checks the externally managed kiwix-serve configuration.
// discoteca never starts, stops, or waits for that process.
func (c *Server) validateKiwixConfig() error {
	if c.KiwixPort < 0 || c.KiwixPort > 65535 {
		return fmt.Errorf("invalid --kiwix-port %d: must be between 0 and 65535", c.KiwixPort)
	}
	return nil
}

// validateZimanticConfig checks the externally managed zimantic configuration.
// discoteca never starts, stops, or waits for that process.
func (c *Server) validateZimanticConfig() error {
	if c.ZimanticPort < 0 || c.ZimanticPort > 65535 {
		return fmt.Errorf("invalid --zimantic-port %d: must be between 0 and 65535", c.ZimanticPort)
	}
	return nil
}

func (c *Server) kiwixHost() string {
	if strings.TrimSpace(c.KiwixHost) == "" {
		return "127.0.0.1"
	}
	return c.KiwixHost
}

func (c *Server) zimanticHost() string {
	if strings.TrimSpace(c.ZimanticHost) == "" {
		return "127.0.0.1"
	}
	return c.ZimanticHost
}

// zimBackend returns the configured external ZIM server. zimantic takes
// precedence so it can replace kiwix-serve without clearing the kiwix flags.
func (c *Server) zimBackend() string {
	switch {
	case c.ZimanticPort > 0:
		return "zimantic"
	case c.KiwixPort > 0:
		return "kiwix-serve"
	default:
		return ""
	}
}

func (c *Server) zimAddress() string {
	if c.zimBackend() == "zimantic" {
		return net.JoinHostPort(c.zimanticHost(), strconv.Itoa(c.ZimanticPort))
	}
	return net.JoinHostPort(c.kiwixHost(), strconv.Itoa(c.KiwixPort))
}

func zimProxyRootPath(basePath string) string {
	return basePath + zimProxyRoot
}

// HandleZimProxy forwards to exactly the configured external ZIM server host
// and port. The request path is never used to choose a proxy target.
func (c *Server) HandleZimProxy(w http.ResponseWriter, r *http.Request) {
	backend := c.zimBackend()
	if backend == "" {
		http.Error(w, "ZIM support disabled", http.StatusServiceUnavailable)
		return
	}

	target, err := url.Parse("http://" + c.zimAddress())
	if err != nil {
		http.Error(w, "Invalid "+backend+" configuration", http.StatusInternalServerError)
		return
	}

	switch backend {
	case "kiwix-serve":
		// kiwix-serve is started with --urlRootLocation set to the external
		// path (including any base path), so restore the prefix that the
		// base-path middleware stripped before forwarding.
		if basePath := BasePathFromContext(r.Context()); basePath != "" {
			r = cloneRequestWithPath(r, basePath+r.URL.Path)
		}
	case "zimantic":
		// zimantic generates links with its configured web_base_path and
		// serves app-relative routes. Strip the discoteca proxy prefix.
		forwardPath := strings.TrimPrefix(r.URL.Path, zimProxyRoot)
		if forwardPath == "" {
			forwardPath = "/"
		}
		r = cloneRequestWithPath(r, forwardPath)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		logging.Log.Warn(backend+" is not reachable", "error", err)
		http.Error(w, backend+" is not reachable", http.StatusServiceUnavailable)
	}
	proxy.ServeHTTP(w, r)
}

func (c *Server) HandleZimView(w http.ResponseWriter, r *http.Request) {
	backend := c.zimBackend()
	if backend == "" {
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
	var contentURL string
	var err error
	if backend == "zimantic" {
		var key string
		var found bool
		key, found, err = c.getZimanticSourceKey(r.Context(), path)
		switch {
		case err != nil:
			logging.Log.Warn("zimantic is not reachable", "error", err)
			http.Error(w, "zimantic is not reachable", http.StatusServiceUnavailable)
			return
		case !found:
			http.Error(w, "ZIM file is not listed by zimantic", http.StatusNotFound)
			return
		}
		contentURL = fmt.Sprintf("%s/content/%s/", zimProxyRootPath(basePath), escapeZimanticSegment(key))
	} else {
		contentURL, err = c.getKiwixContentURL(r.Context(), basePath, path)
		if err != nil {
			logging.Log.Warn("kiwix-serve is not reachable", "error", err)
			http.Error(w, "kiwix-serve is not reachable", http.StatusServiceUnavailable)
			return
		}
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

// escapeZimanticSegment percent-encodes a source key as one URL path segment.
// zimantic documents colon-bearing keys as percent-encoded (%3A).
func escapeZimanticSegment(segment string) string {
	return strings.ReplaceAll(url.PathEscape(segment), ":", "%3A")
}

// getZimanticSourceKey maps a local ZIM file to the source key that zimantic
// assigns to it. The ZIM UUID is the primary identity; title and key-stem
// matching cover archives that zimantic catalogued without reading metadata.
func (c *Server) getZimanticSourceKey(ctx context.Context, zimPath string) (string, bool, error) {
	entries, err := c.fetchZimanticLibrary(ctx)
	if err != nil {
		return "", false, err
	}

	if uuid := zimFileUUID(zimPath); uuid != "" {
		for _, entry := range entries {
			if strings.EqualFold(entry.ProviderID, uuid) {
				return entry.Key, true, nil
			}
		}
	}

	stem := strings.TrimSuffix(filepath.Base(zimPath), filepath.Ext(zimPath))
	perma := zimPermaRef(stem)
	keyStem := zimanticSlug(stem)
	for _, entry := range entries {
		title := strings.TrimSpace(entry.Title)
		if strings.EqualFold(title, stem) || strings.EqualFold(title, perma) {
			return entry.Key, true, nil
		}
		keyPart := entry.Key
		if idx := strings.LastIndex(keyPart, ":"); idx >= 0 {
			keyPart = keyPart[idx+1:]
		}
		if keyStem != "" && strings.HasPrefix(strings.ToLower(keyPart), keyStem+"-") {
			return entry.Key, true, nil
		}
	}

	return "", false, nil
}

// fetchZimanticLibrary returns every ZIM record that zimantic lists. It asks
// for ZIM sources only so a large Files or Kolibri library does not dominate
// the lookup.
func (c *Server) fetchZimanticLibrary(ctx context.Context) ([]zimanticSource, error) {
	const pageSize = 1000

	var entries []zimanticSource
	for offset := 0; ; offset += pageSize {
		libraryURL := fmt.Sprintf("http://%s/library.json?provider=zim&limit=%d&offset=%d",
			c.zimAddress(), pageSize, offset)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, libraryURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		var page []zimanticSource
		decodeErr := json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("library returned status %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}

		entries = append(entries, page...)
		total, totalErr := strconv.Atoi(resp.Header.Get("X-Total-Count"))
		if len(page) == 0 || (totalErr == nil && offset+len(page) >= total) || len(page) < pageSize {
			return entries, nil
		}
	}
}

// zimFileUUID reads the UUID from a ZIM header. An unreadable or malformed
// header returns "" so callers can fall back to title matching.
func zimFileUUID(zimPath string) string {
	file, err := os.Open(zimPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	var header [24]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return ""
	}
	if string(header[:4]) != "ZIM\x04" {
		return ""
	}
	raw := hex.EncodeToString(header[8:24])
	return raw[0:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20] + "-" + raw[20:32]
}

var zimDateSuffixRe = regexp.MustCompile(`_\d{4}-\d{2}(?:-\d{2})?$`)

// zimPermaRef strips the trailing date suffix that Kiwix uses in archive names.
func zimPermaRef(stem string) string {
	loc := zimDateSuffixRe.FindStringIndex(stem)
	if loc == nil {
		return stem
	}
	return stem[:loc[0]]
}

// zimanticSlug mirrors zimantic's source-key slug for a ZIM archive stem.
func zimanticSlug(stem string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(stem) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-._")
	if len(slug) > 48 {
		slug = strings.Trim(slug[:48], "-._")
	}
	if slug == "" {
		return "source"
	}
	return slug
}

// getKiwixContentURL reads the external kiwix-serve catalog and returns the
// viewer URL for the requested ZIM. If the catalog does not expose a matching
// title or path, it falls back to the first usable entry.
func (c *Server) getKiwixContentURL(ctx context.Context, basePath, zimPath string) (string, error) {
	urlRoot := zimProxyRootPath(basePath)
	catalogURL := fmt.Sprintf("http://%s%s/catalog/v2/entries", c.zimAddress(), urlRoot)

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
