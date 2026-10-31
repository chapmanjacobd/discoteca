// Package server implements the discoteca HTTP API and web UI host.
package server

import (
	"crypto/subtle"
	"encoding/json"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/web"
)

// sendJSON writes a JSON response with proper headers and error handling
func sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logging.Log.Error("Failed to encode JSON response", "error", err)
	}
}

// sendError writes a JSON error response with the given message
func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, models.ErrorResponse{Error: message})
}

// initMimeTypes registers custom MIME types for JavaScript files.
// This should be called during application startup.
func initMimeTypes() {
	_ = mime.AddExtensionType(".js", "text/javascript")
	_ = mime.AddExtensionType(".mjs", "text/javascript")
	_ = mime.AddExtensionType(".ts", "text/javascript")
}

// LsEntry represents a directory listing entry
type LsEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	MediaType string `json:"media_type,omitempty"`
}

// Server is the HTTP server command
type Server struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.AggregateFlags   `embed:""`
	models.PlaybackFlags    `embed:""`
	models.PostActionFlags  `embed:""`
	models.FTSFlags         `embed:""`

	Databases             []string      `help:"SQLite database files"                                                           required:"true" arg:"" type:"existingfile"`
	Host                  string        `help:"Address to listen on (loopback by default)"                                                                                 default:"127.0.0.1"`
	Port                  int           `help:"Port to listen on"                                                                                                          default:"5555"      short:"p"`
	KiwixHost             string        `help:"Host of the externally managed kiwix-serve"                                                                                 default:"127.0.0.1"`
	KiwixPort             int           `help:"Port of the externally managed kiwix-serve (0 disables ZIM support)"                                                        default:"8181"`
	TrustForwardedPrefix  bool          `help:"Trust X-Forwarded-Prefix from non-loopback peers"`
	PublicDir             string        `help:"Override embedded web assets with local directory"`
	BasePath              string        `help:"URL prefix when served behind a reverse proxy (for example /disco)"                                                         default:""`
	Dev                   bool          `help:"Enable development mode (auto-reload)"`
	ReadOnly              bool          `help:"Disable write operations (progress tracking, playlist modifications, deletions)"`
	NoBrowser             bool          `help:"Don't open browser on startup"`
	SearchTimeout         time.Duration `help:"Timeout for a single /api/query search"                                                                                     default:"15s"`
	MaxConcurrentSearches int           `help:"Maximum concurrent /api/query searches (0 = unlimited)"                                                                     default:"4"`
	MaxLimit              int           `help:"Maximum number of results a client may request per database (0 = unlimited)"                                                default:"1000"`
	DBCacheMB             int           `help:"SQLite page cache size in MiB per connection"                                                                               default:"256"`
	ApplicationStartTime  int64         `                                                                                                                                                                kong:"-"`
	APIToken              string        `                                                                                                                                                                kong:"-"`
	thumbnailCache        sync.Map
	dbCache               sync.Map
	searchSem             chan struct{}
	searchSemOnce         sync.Once
	hasFfmpeg             bool
}

// authMiddleware validates the API token for authenticated endpoints. When no
// token is configured (the default for loopback-only servers) all requests are
// allowed. Scripts may send the token in X-Disco-Token or as a Bearer token;
// browsers use the HttpOnly session cookie established by POST /api/session.
func (c *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.APIToken == "" {
			next(w, r)
			return
		}

		token := r.Header.Get("X-Disco-Token")
		if token == "" {
			if bearer, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				token = strings.TrimSpace(bearer)
			}
		}
		if token == "" {
			if cookie, err := r.Cookie(sessionCookieName); err == nil {
				token = cookie.Value
			}
		}

		if !c.tokenMatches(token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

const sessionCookieName = "disco_token"

// tokenMatches compares tokens in constant time. An empty configured token
// disables authentication, so every supplied token matches.
func (c *Server) tokenMatches(token string) bool {
	if c.APIToken == "" {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(c.APIToken)) == 1
}

// HandleSession exchanges an API token for an HttpOnly session cookie. The
// token is accepted only in the Authorization header so it never appears in a
// URL that reaches a reverse proxy log.
func (c *Server) HandleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c.APIToken == "" {
		http.Error(w, "Session authentication is not configured", http.StatusServiceUnavailable)
		return
	}

	bearer, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || !c.tokenMatches(strings.TrimSpace(bearer)) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	prefix := BasePathFromContext(r.Context())
	cookiePath := "/"
	if prefix != "" {
		cookiePath = prefix + "/"
	}
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    c.APIToken,
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// isPathBlocklisted checks if a path should be denied access. It matches path
// components and well-known sensitive suffixes rather than arbitrary
// substrings, so a legitimate media tree named "root" is not rejected.
func (c *Server) isPathBlocklisted(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	parts := strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return false
	}

	for _, part := range parts {
		switch part {
		case ".ssh", ".aws", ".config", ".gnupg":
			return true
		}
	}

	base := parts[len(parts)-1]
	if base == "id_rsa" || base == "id_ed25519" {
		return true
	}

	joined := "/" + strings.Join(parts, "/")
	if joined == "/root" || strings.HasPrefix(joined, "/root/") {
		return true
	}

	sensitiveSuffixes := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/windows/system32/config/sam",
		"/windows/system32/config/security",
		"/windows/system32/config/software",
		"/windows/system32/config/system",
	}
	for _, suffix := range sensitiveSuffixes {
		if joined == suffix || strings.HasSuffix(joined, suffix) {
			return true
		}
	}
	return false
}

// registerAPIRoutes registers all API routes with the mux
func (c *Server) registerAPIRoutes(mux *http.ServeMux) {
	// Health and favicon
	mux.HandleFunc("/health", c.HandleHealth)
	// Session exchange is the one API route that validates the token itself;
	// it must be reachable before a session cookie exists.
	mux.HandleFunc("/api/session", c.HandleSession)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		favicon, err := web.FsRaw.ReadFile("dist/favicon.svg")
		if err != nil {
			http.NotFound(w, nil)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(favicon)
	})

	// Core API routes
	apiRoutes := []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"/api/databases", c.HandleDatabases},
		{"/api/categories", c.HandleCategories},
		{"/api/genres", c.HandleGenres},
		{"/api/languages", c.HandleLanguages},
		{"/api/ratings", c.HandleRatings},
		{"/api/query", c.HandleQuery},
		{"/api/metadata", c.HandleMetadata},
		{"/api/play", c.HandlePlay},
		{"/api/delete", c.HandleDelete},
		{"/api/progress", c.HandleProgress},
		{"/api/document-progress", c.HandleDocumentProgress},
		{"/api/mark-played", c.HandleMarkPlayed},
		{"/api/mark-unplayed", c.HandleMarkUnplayed},
		{"/api/rate", c.HandleRate},
		{"/api/playlists", c.HandlePlaylists},
		{"/api/playlists/items", c.HandlePlaylistItems},
		{"/api/playlists/reorder", c.HandlePlaylistReorder},
		{"/api/events", c.HandleEvents},
		{"/api/ls", c.HandleLs},
		{"/api/du", c.HandleDU},
		{"/api/episodes", c.HandleEpisodes},
		{"/api/filter-bins", c.HandleFilterBins},
		{"/api/random-clip", c.HandleRandomClip},
		{"/api/categorize/suggest", c.HandleCategorizeSuggest},
		{"/api/categorize/apply", c.HandleCategorizeApply},
		{"/api/categorize/keywords", c.HandleCategorizeKeywords},
		{"/api/categorize/category", c.HandleCategorizeDeleteCategory},
		{"/api/categorize/keyword", c.HandleCategorizeKeyword},
		{"/api/raw", c.HandleRaw},
		{"/api/zim/view", c.HandleZimView},
		{"/api/zim/proxy/{rest...}", c.HandleZimProxy},
		{"/api/rsvp", c.HandleRSVP},
		{"/api/epub/{path...}", c.HandleEpubConvert},
		{"/api/hls/playlist", c.HandleHLSPlaylist},
		{"/api/hls/segment", c.HandleHLSSegment},
		{"/api/subtitles", c.HandleSubtitles},
		{"/api/thumbnail", c.HandleThumbnail},
		{"/opds", c.HandleOPDS},
		{"/api/trash", c.HandleTrash},
		{"/api/empty-bin", c.HandleEmptyBin},
	}

	for _, route := range apiRoutes {
		mux.HandleFunc(route.pattern, c.authMiddleware(route.handler))
	}
}

// newLibHandler creates a handler for library static assets
func (c *Server) newLibHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		var f http.File
		var err error
		if c.PublicDir != "" {
			f, err = http.Dir(c.PublicDir).Open(path)
		} else {
			f, err = http.FS(web.FS).Open(path)
		}
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs") || strings.HasSuffix(path, ".ts") {
			w.Header().Set("Content-Type", "text/javascript")
		}
		stat, _ := f.Stat()
		http.ServeContent(w, r, path, stat.ModTime(), f)
	}
}

// newStaticHandler creates a handler for all other static files
func (c *Server) newStaticHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := BasePathFromContext(r.Context())

		c.setCacheControlHeaders(w, r)

		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			if c.serveIndexHTML(w, r, prefix) {
				return
			}
		}

		if c.PublicDir != "" {
			http.FileServer(http.Dir(c.PublicDir)).ServeHTTP(w, r)
		} else {
			http.FileServer(http.FS(web.FS)).ServeHTTP(w, r)
		}
	}
}

// setCacheControlHeaders sets appropriate cache headers based on file type
func (c *Server) setCacheControlHeaders(w http.ResponseWriter, r *http.Request) {
	if c.Dev {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return
	}

	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs") || strings.HasSuffix(path, ".ts"):
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "public, max-age=604800")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Cache-Control", "public, max-age=604800")
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	case strings.HasPrefix(path, "/lib/"):
		w.Header().Set("Cache-Control", "public, max-age=604800")
	}
}

// Mux creates the HTTP request multiplexer with all routes
func (c *Server) Mux() http.Handler {
	mux := http.NewServeMux()

	c.registerAPIRoutes(mux)
	mux.HandleFunc("/lib/", c.newLibHandler())
	mux.Handle("/", c.newStaticHandler())

	return c.withBasePath(mux)
}
