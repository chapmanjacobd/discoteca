package commands

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
	"github.com/chapmanjacobd/discoteca/web"
)

// sendJSON writes a JSON response with proper headers and error handling
func sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		models.Log.Error("Failed to encode JSON response", "error", err)
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

// ServeCmd is the HTTP server command
type ServeCmd struct {
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
	Host                  string        `help:"Address to listen on (loopback by default)"                                                default:"127.0.0.1"`
	Port                  int           `help:"Port to listen on"                                                                                                          default:"5555" short:"p"`
	KiwixHost             string        `help:"Host of the externally managed kiwix-serve"                                                  default:"127.0.0.1"`
	KiwixPort             int           `help:"Port of the externally managed kiwix-serve (0 disables ZIM support)"                         default:"8181"`
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
	ApplicationStartTime  int64         `                                                                                                                                                           kong:"-"`
	APIToken              string        `                                                                                                                                                           kong:"-"`
	thumbnailCache        sync.Map
	dbCache               sync.Map
	searchSem             chan struct{}
	searchSemOnce         sync.Once
	hasFfmpeg             bool
}

// authMiddleware validates API token for authenticated endpoints. Scripts may
// send the token in X-Disco-Token or as a Bearer token; browsers use the
// HttpOnly session cookie established by POST /api/session.
func (c *ServeCmd) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

// tokenMatches compares tokens in constant time and rejects empty tokens when
// the server has a configured token.
func (c *ServeCmd) tokenMatches(token string) bool {
	if c.APIToken == "" {
		return token == ""
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(c.APIToken)) == 1
}

// HandleSession exchanges an API token for an HttpOnly session cookie. The
// token is accepted only in the Authorization header so it never appears in a
// URL that reaches a reverse proxy log.
func (c *ServeCmd) HandleSession(w http.ResponseWriter, r *http.Request) {
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

// isPathBlocklisted checks if a path should be denied access
func (c *ServeCmd) isPathBlocklisted(path string) bool {
	// Normalize path separators to forward slashes for consistent matching
	p := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	blocked := []string{
		"etc/passwd",
		"etc/shadow",
		".ssh/",
		".aws/",
		".config/",
		".gnupg/",
		"root/",
		"id_rsa",
		"id_ed25519",
		// Windows-specific sensitive paths
		"windows/system32/config/sam",
		"windows/system32/config/security",
		"windows/system32/config/software",
		"windows/system32/config/system",
	}
	for _, b := range blocked {
		if strings.Contains(p, b) {
			return true
		}
	}
	return false
}

// registerAPIRoutes registers all API routes with the mux
func (c *ServeCmd) registerAPIRoutes(mux *http.ServeMux) {
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
func (c *ServeCmd) newLibHandler() http.HandlerFunc {
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
func (c *ServeCmd) newStaticHandler() http.HandlerFunc {
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
func (c *ServeCmd) setCacheControlHeaders(w http.ResponseWriter, r *http.Request) {
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
func (c *ServeCmd) Mux() http.Handler {
	mux := http.NewServeMux()

	c.registerAPIRoutes(mux)
	mux.HandleFunc("/lib/", c.newLibHandler())
	mux.Handle("/", c.newStaticHandler())

	return c.withBasePath(mux)
}

// getOrCreateDBConn retrieves a cached database connection or creates a new one
func (c *ServeCmd) getOrCreateDBConn(ctx context.Context, dbPath string) (*sql.DB, error) {
	if val, ok := c.dbCache.Load(dbPath); ok {
		if dbConn, ok := val.(*sql.DB); ok {
			return dbConn, nil
		}
	}

	newDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return nil, err
	}

	sqlDB := newDB
	if loaded, ok := c.dbCache.LoadOrStore(dbPath, sqlDB); ok {
		if dbConn, ok := loaded.(*sql.DB); ok {
			sqlDB = dbConn
		}
		_ = newDB.Close()
	}
	return sqlDB, nil
}

// handleCorruptionError attempts to repair a corrupted database and returns whether to retry
func (c *ServeCmd) handleCorruptionError(
	ctx context.Context,
	dbPath string,
	sqlDB *sql.DB,
	err error,
	isConnectError bool,
) (shouldRetry bool, wrappedErr error) {
	if !db.IsCorruptionError(err) {
		return false, nil
	}

	if isConnectError {
		if repErr := db.Repair(ctx, dbPath); repErr != nil {
			return false, fmt.Errorf("repair failed: %w (original error: %w)", repErr, err)
		}
		models.Log.Info("Database repaired, retrying connect", "db", dbPath)
		return true, nil
	}

	// Query error: delete from cache and close connection
	c.dbCache.Delete(dbPath)
	_ = sqlDB.Close()

	if repErr := db.Repair(ctx, dbPath); repErr != nil {
		models.Log.Error("Database repair failed", "db", dbPath, "error", repErr)
		return false, nil // Return original error
	}
	models.Log.Info("Database repaired, retrying operation", "db", dbPath)
	return true, nil
}

// execDB connects to the database and executes fn. If a corruption error occurs,
// it attempts to repair the database and retries the operation once.
func (c *ServeCmd) execDB(ctx context.Context, dbPath string, fn func(ctx context.Context, sqlDB *sql.DB) error) error {
	const maxRetries = 1
	for i := 0; i <= maxRetries; i++ {
		sqlDB, err := c.getOrCreateDBConn(ctx, dbPath)
		if err != nil {
			if shouldRetry, wrappedErr := c.handleCorruptionError(ctx, dbPath, nil, err, true); shouldRetry {
				continue
			} else if wrappedErr != nil {
				return wrappedErr
			}
			return err
		}

		err = fn(ctx, sqlDB)
		if err != nil {
			if shouldRetry, _ := c.handleCorruptionError(ctx, dbPath, sqlDB, err, false); shouldRetry {
				continue
			}
			if i > 0 {
				models.Log.Error("Operation failed even after database repair", "db", dbPath, "error", err)
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}

// Close closes all cached database connections
func (c *ServeCmd) Close() error {
	var errs []error
	c.dbCache.Range(func(key, value any) bool {
		if sqlDB, ok := value.(*sql.DB); ok {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		c.dbCache.Delete(key)
		return true
	})

	if len(errs) > 0 {
		return fmt.Errorf("failed to close some resources: %v", errs)
	}
	return nil
}

// initDatabases connects to all databases and caches the connections
func (c *ServeCmd) initDatabases(ctx context.Context) {
	for _, dbPath := range c.Databases {
		sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

// setupAPIToken sets the API token from the environment or generates one. A
// generated token is printed once to stderr; it is never embedded in a static
// response.
func (c *ServeCmd) setupAPIToken() {
	if envToken := os.Getenv("DISCO_API_TOKEN"); envToken != "" {
		c.APIToken = envToken
		return
	}
	c.APIToken = utils.RandomString(32)
	fmt.Fprintf(
		os.Stderr,
		"DISCO_API_TOKEN is not set. Generated API token: %s\n"+
			"Use this token to authenticate (POST /api/session with an Authorization: Bearer header).\n",
		c.APIToken,
	)
}

// listenAddress returns the configured host and port as a net address.
func (c *ServeCmd) listenAddress() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(c.Port))
}

// isLoopbackHost reports whether host refers to the local machine only.
func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	switch strings.ToLower(host) {
	case "", "localhost":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// validateListenConfig refuses an unauthenticated bind on a non-loopback
// interface.
func (c *ServeCmd) validateListenConfig() error {
	if !isLoopbackHost(c.Host) && os.Getenv("DISCO_API_TOKEN") == "" {
		return fmt.Errorf(
			"refusing to listen on non-loopback address %q without DISCO_API_TOKEN; "+
				"set DISCO_API_TOKEN or bind to 127.0.0.1 and use a reverse proxy",
			c.Host,
		)
	}
	return nil
}

// cacheDatabases connects to all databases and stores them in the cache
func (c *ServeCmd) cacheDatabases(ctx context.Context) {
	for _, dbPath := range c.Databases {
		sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
		if err != nil {
			models.Log.Error("Failed to connect to database on startup", "db", dbPath, "error", err)
			continue
		}
		c.dbCache.Store(dbPath, sqlDB)
	}
}

// runMaintenance runs maintenance on all databases asynchronously
func (c *ServeCmd) runMaintenance(ctx context.Context) {
	go func() {
		config := db.DefaultMaintenanceConfig()
		for _, dbPath := range c.Databases {
			if val, ok := c.dbCache.Load(dbPath); ok {
				if sqlDB, ok := val.(*sql.DB); ok {
					if err := db.RunMaintenance(ctx, sqlDB, config, dbPath); err != nil {
						models.Log.Error("Maintenance failed", "db", dbPath, "error", err)
					}
				}
			}
		}
	}()
}

// checkFfmpeg checks if ffmpeg is available in PATH
func (c *ServeCmd) checkFfmpeg() {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		models.Log.Warn("ffmpeg not found in PATH, on-the-fly transcoding will be unavailable")
		c.hasFfmpeg = false
	} else {
		c.hasFfmpeg = true
	}
}

// openBrowser opens the default browser to the server URL
func (c *ServeCmd) openBrowser(ctx context.Context, baseURL string) {
	go func() {
		time.Sleep(500 * time.Millisecond)
		var openCmd string
		var openArgs []string

		switch runtime.GOOS {
		case "linux":
			openCmd = "xdg-open"
		case "darwin":
			openCmd = "open"
		case "windows":
			openCmd = "cmd"
			openArgs = []string{"/c", "start"}
		}

		if openCmd != "" {
			openArgs = append(openArgs, baseURL)
			browserCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(browserCtx, openCmd, openArgs...)
			if err := cmd.Start(); err != nil {
				models.Log.Debug("Failed to open browser", "error", err)
			}
		}
	}()
}

// Run starts the HTTP server
func (c *ServeCmd) Run(ctx context.Context) error {
	defer c.Close()
	models.SetupLogging(c.Verbose)
	db.InitFtsConfig()
	db.SetCacheSizeMB(c.DBCacheMB)

	if err := c.validateListenConfig(); err != nil {
		return err
	}
	if err := c.validateKiwixConfig(); err != nil {
		return err
	}

	// Initialize MIME types
	initMimeTypes()

	// Pre-warm database connections
	c.initDatabases(ctx)

	c.ApplicationStartTime = time.Now().UnixNano()

	// Setup API token
	c.setupAPIToken()

	// Connect and cache all databases
	c.cacheDatabases(ctx)

	// Run maintenance on all databases (refresh folder_stats and FTS if needed)
	// This runs asynchronously so it doesn't block server startup
	c.runMaintenance(ctx)

	// Check for ffmpeg
	c.checkFfmpeg()

	handler := c.Mux()

	addr := c.listenAddress()
	baseURL := fmt.Sprintf("http://%s", addr)
	models.Log.Info("Server starting", "addr", baseURL)

	// Open browser unless --no-browser is passed
	if !c.NoBrowser {
		c.openBrowser(ctx, baseURL)
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // Streaming responses (HLS, Raw files) need no write timeout or a very large one
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}

// GetGlobalFlags returns all embedded flag structs
func (c *ServeCmd) GetGlobalFlags() models.GlobalFlags {
	return models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		QueryFlags:       c.QueryFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		SortFlags:        c.SortFlags,
		DisplayFlags:     c.DisplayFlags,
		AggregateFlags:   c.AggregateFlags,
		PlaybackFlags:    c.PlaybackFlags,
		PostActionFlags:  c.PostActionFlags,
		FTSFlags:         c.FTSFlags,
	}
}

// parseSearchFlags extracts search-related flags from query parameters
func (c *ServeCmd) parseSearchFlags(flags *models.GlobalFlags, q url.Values) {
	if search := q.Get("search"); search != "" {
		flags.Search = strings.Fields(search)
	}
	if searchType := q.Get("search_type"); searchType == "substring" {
		flags.FTS = false
	} else if searchType == "fts" {
		flags.FTS = true
	}
}

// parseCategoryFlags extracts category and genre flags
func (c *ServeCmd) parseCategoryFlags(flags *models.GlobalFlags, q url.Values) {
	if categories := q["category"]; len(categories) > 0 {
		flags.Category = categories
	}
	if genre := q.Get("genre"); genre != "" {
		flags.Genre = genre
	}
}

// parseLanguageFlags extracts language filter flags
func (c *ServeCmd) parseLanguageFlags(flags *models.GlobalFlags, q url.Values) {
	if languages := q["language"]; len(languages) > 0 {
		flags.Language = languages
	}
}

// parsePathFlags extracts path-related flags
func (c *ServeCmd) parsePathFlags(flags *models.GlobalFlags, q url.Values) {
	if paths := q.Get("paths"); paths != "" {
		flags.Paths = strings.Split(paths, ",")
	}
}

// parseRatingFlags extracts rating filter flags
func (c *ServeCmd) parseRatingFlags(flags *models.GlobalFlags, q url.Values) {
	if ratings := q["rating"]; len(ratings) > 0 {
		var clauses []string
		for _, rating := range ratings {
			if r, err := strconv.Atoi(rating); err == nil {
				if r == 0 {
					clauses = append(clauses, "(score IS NULL OR score = 0)")
				} else {
					clauses = append(clauses, fmt.Sprintf("score = %d", r))
				}
			}
		}
		if len(clauses) > 0 {
			if len(clauses) == 1 {
				flags.Where = append(flags.Where, clauses[0])
			} else {
				flags.Where = append(flags.Where, "("+strings.Join(clauses, " OR ")+")")
			}
		}
	}
}

// rejectInvalidFlags writes a 400 when ParseFlags found an unsafe parameter.
func rejectInvalidFlags(w http.ResponseWriter, flags models.GlobalFlags) bool {
	if flags.ValidationError == "" {
		return false
	}
	sendError(w, http.StatusBadRequest, flags.ValidationError)
	return true
}

func setFlagValidationError(flags *models.GlobalFlags, err error) {
	if flags.ValidationError == "" {
		flags.ValidationError = err.Error()
	}
}

// parseScoreFlags extracts min/max score and unplayed flags
func (c *ServeCmd) parseScoreFlags(flags *models.GlobalFlags, q url.Values) {
	parseScore := func(name, operator string) {
		raw := q.Get(name)
		if raw == "" {
			return
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			setFlagValidationError(flags, fmt.Errorf("invalid %s value %q", name, raw))
			return
		}
		flags.Where = append(flags.Where, operator+" "+strconv.FormatFloat(value, 'g', -1, 64))
	}
	parseScore("min_score", "score >=")
	parseScore("max_score", "score <=")
	if unplayed := q.Get("unplayed"); unplayed == "true" {
		flags.Where = append(flags.Where, "COALESCE(play_count, 0) = 0 AND COALESCE(playhead, 0) = 0")
	}
}

// parseSortFlags extracts sorting-related flags
func (c *ServeCmd) parseSortFlags(flags *models.GlobalFlags, q url.Values) {
	if seed := strings.TrimSpace(q.Get("random_seed")); seed != "" {
		parsed, err := strconv.ParseUint(seed, 10, 64)
		if err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid random_seed: %w", err))
		} else {
			flags.RandomSeed = parsed
		}
	}

	if sortBy := strings.TrimSpace(q.Get("sort")); sortBy != "" {
		if err := query.ValidateSortConfig(sortBy); err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid sort: %w", err))
		} else {
			flags.SortBy = sortBy
			if sortBy == "random" || sortBy == "random()" ||
				sortBy == "RANDOM" || sortBy == "RANDOM()" {
				flags.Random = true
			}
		}
	}
	// Support complex sorting with sort_fields array (JSON) or comma-separated string
	if sortFields := q.Get("sort_fields"); sortFields != "" {
		config := sortFields
		if strings.HasPrefix(sortFields, "[") {
			var fieldList []string
			if err := json.Unmarshal([]byte(sortFields), &fieldList); err != nil {
				setFlagValidationError(flags, fmt.Errorf("invalid sort_fields: %w", err))
				return
			}
			config = strings.Join(fieldList, ",")
		}
		if err := query.ValidateSortConfig(config); err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid sort_fields: %w", err))
		} else {
			flags.PlayInOrder = config
		}
	}
	// Also support sort_order for explicit direction
	if sortDesc := q.Get("sort_desc"); sortDesc != "" {
		descFields := make(map[string]bool)
		for f := range strings.SplitSeq(sortDesc, ",") {
			f = strings.TrimSpace(f)
			if err := query.ValidateSortConfig(f); err != nil {
				setFlagValidationError(flags, fmt.Errorf("invalid sort_desc: %w", err))
				return
			}
			descFields[f] = true
		}
		if flags.PlayInOrder != "" {
			var newOrder []string
			for f := range strings.SplitSeq(flags.PlayInOrder, ",") {
				f = strings.TrimSpace(f)
				f = strings.TrimPrefix(f, "-")
				f = strings.TrimSuffix(f, " desc")
				f = strings.TrimSuffix(f, " asc")
				if descFields[f] {
					newOrder = append(newOrder, "-"+f)
				} else {
					newOrder = append(newOrder, f)
				}
			}
			flags.PlayInOrder = strings.Join(newOrder, ",")
		}
	}
	if reverse := q.Get("reverse"); reverse == "true" {
		flags.Reverse = true
	}
}

// parsePaginationFlags extracts limit and offset flags
func (c *ServeCmd) parsePaginationFlags(flags *models.GlobalFlags, q url.Values) {
	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l >= 0 {
			if c.MaxLimit > 0 && l > c.MaxLimit {
				l = c.MaxLimit
			}
			flags.Limit = l
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			flags.Offset = o
		}
	}
}

// parseSizeFlags extracts size filter flags
func (c *ServeCmd) parseSizeFlags(flags *models.GlobalFlags, q url.Values) {
	if minSize := q.Get("min_size"); minSize != "" {
		flags.Size = append(flags.Size, ">"+minSize+"MB")
	}
	if maxSize := q.Get("max_size"); maxSize != "" {
		flags.Size = append(flags.Size, "<"+maxSize+"MB")
	}
	if sizes := q["size"]; len(sizes) > 0 {
		flags.Size = append(flags.Size, sizes...)
	}
}

// parseDurationFlags extracts duration filter flags
func (c *ServeCmd) parseDurationFlags(flags *models.GlobalFlags, q url.Values) {
	if minDuration := q.Get("min_duration"); minDuration != "" {
		flags.Duration = append(flags.Duration, ">"+minDuration+"min")
	}
	if maxDuration := q.Get("max_duration"); maxDuration != "" {
		flags.Duration = append(flags.Duration, "<"+maxDuration+"min")
	}
	if durations := q["duration"]; len(durations) > 0 {
		flags.Duration = append(flags.Duration, durations...)
	}
}

// parseTimeFlags extracts time-based filter flags
func (c *ServeCmd) parseTimeFlags(flags *models.GlobalFlags, q url.Values) {
	if minModified := q.Get("min_modified"); minModified != "" {
		flags.ModifiedAfter = minModified
	}
	if maxModified := q.Get("max_modified"); maxModified != "" {
		flags.ModifiedBefore = maxModified
	}
	if minCreated := q.Get("min_created"); minCreated != "" {
		flags.CreatedAfter = minCreated
	}
	if maxCreated := q.Get("max_created"); maxCreated != "" {
		flags.CreatedBefore = maxCreated
	}
	if minDownloaded := q.Get("min_downloaded"); minDownloaded != "" {
		flags.DownloadedAfter = minDownloaded
	}
	if maxDownloaded := q.Get("max_downloaded"); maxDownloaded != "" {
		flags.DownloadedBefore = maxDownloaded
	}

	if modified := q["modified"]; len(modified) > 0 {
		flags.Modified = append(flags.Modified, modified...)
	}
	if created := q["created"]; len(created) > 0 {
		flags.Created = append(flags.Created, created...)
	}
	if downloaded := q["downloaded"]; len(downloaded) > 0 {
		flags.Downloaded = append(flags.Downloaded, downloaded...)
	}

	for _, m := range q["modified"] {
		for part := range strings.SplitSeq(m, ",") {
			if strings.HasPrefix(part, "+") {
				flags.ModifiedAfter = part[1:]
			} else if strings.HasPrefix(part, "-") {
				flags.ModifiedBefore = part[1:]
			}
		}
	}
	for _, m := range q["created"] {
		for part := range strings.SplitSeq(m, ",") {
			if strings.HasPrefix(part, "+") {
				flags.CreatedAfter = part[1:]
			} else if strings.HasPrefix(part, "-") {
				flags.CreatedBefore = part[1:]
			}
		}
	}
	for _, m := range q["downloaded"] {
		for part := range strings.SplitSeq(m, ",") {
			if strings.HasPrefix(part, "+") {
				flags.DownloadedAfter = part[1:]
			} else if strings.HasPrefix(part, "-") {
				flags.DownloadedBefore = part[1:]
			}
		}
	}
}

// parseMediaFlags extracts media type filter flags
func (c *ServeCmd) parseMediaFlags(flags *models.GlobalFlags, q url.Values) {
	mediaTypes := q["media_type"]
	if len(mediaTypes) == 0 {
		mediaTypes = q["type"]
	}
	for _, t := range mediaTypes {
		switch t {
		case "video":
			flags.VideoOnly = true
		case "audio":
			flags.AudioOnly = true
		case "image":
			flags.ImageOnly = true
		case "text":
			flags.TextOnly = true
		}
	}

	if video := q.Get("video"); video == "true" {
		flags.VideoOnly = true
	}
	if audio := q.Get("audio"); audio == "true" {
		flags.AudioOnly = true
	}
	if image := q.Get("image"); image == "true" {
		flags.ImageOnly = true
	}
	if text := q.Get("text"); text == "true" {
		flags.TextOnly = true
	}
	if q.Get("captions") == "true" || q.Get("view") == "captions" {
		flags.WithCaptions = true
	}
	if watched := q.Get("watched"); watched == "true" {
		w := true
		flags.Watched = &w
	}
	if unfinished := q.Get("unfinished"); unfinished == "true" {
		flags.Unfinished = true
	}
	if completed := q.Get("completed"); completed == "true" {
		flags.Completed = true
	}
}

// parseStatusFlags extracts status-related flags (trash, all, episodes, group_by)
func (c *ServeCmd) parseStatusFlags(flags *models.GlobalFlags, q url.Values) {
	if all := q.Get("all"); all == "true" {
		flags.All = true
	}
	if q.Get("trash") == "true" {
		flags.OnlyDeleted = true
		flags.HideDeleted = false
	}
	if episodes := q["episodes"]; len(episodes) > 0 {
		flags.FileCounts = strings.Join(episodes, ",")
	}
	if groupBy := q.Get("group_by"); groupBy == "parent" {
		flags.GroupByParent = true
	}
}

// parseDatabaseFlags extracts database filter flags
func (c *ServeCmd) parseDatabaseFlags(flags *models.GlobalFlags, q url.Values) {
	if dbs := q["db"]; len(dbs) > 0 {
		flags.Databases = dbs
	}
}

// ParseFlags extracts query parameters into GlobalFlags
func (c *ServeCmd) ParseFlags(r *http.Request) models.GlobalFlags {
	flags := c.GetGlobalFlags()
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		flags.ValidationError = "invalid query string: " + err.Error()
		return flags
	}

	c.parseSearchFlags(&flags, q)
	c.parseCategoryFlags(&flags, q)
	c.parseLanguageFlags(&flags, q)
	c.parsePathFlags(&flags, q)
	c.parseRatingFlags(&flags, q)
	c.parseScoreFlags(&flags, q)
	c.parseSortFlags(&flags, q)
	c.parsePaginationFlags(&flags, q)
	c.parseSizeFlags(&flags, q)
	c.parseDurationFlags(&flags, q)
	c.parseTimeFlags(&flags, q)
	c.parseMediaFlags(&flags, q)
	c.parseStatusFlags(&flags, q)
	c.parseDatabaseFlags(&flags, q)

	return flags
}

// filterDatabases validates and filters the requested databases against the server's allowed list
// Returns an error if any requested database is not in the allowed list
func (c *ServeCmd) filterDatabases(requested []string) ([]string, error) {
	// If no specific databases requested, use all configured databases
	if len(requested) == 0 {
		return c.Databases, nil
	}

	// Build a set of allowed databases for quick lookup
	allowedSet := make(map[string]bool, len(c.Databases))
	for _, db := range c.Databases {
		allowedSet[db] = true
	}

	// Validate each requested database
	var filtered []string
	for _, db := range requested {
		if !allowedSet[db] {
			return nil, fmt.Errorf("database not in allowed list: %s", db)
		}
		filtered = append(filtered, db)
	}

	return filtered, nil
}

// getDBs returns the list of databases to query based on request flags
// It validates that all requested databases are in the server's allowed list
func (c *ServeCmd) getDBs(flags models.GlobalFlags) ([]string, error) {
	return c.filterDatabases(flags.Databases)
}

// execWriteAcrossDBs runs fn against every configured database and returns a
// write outcome. Handlers use it so a failure is not reported as success.
func (c *ServeCmd) execWriteAcrossDBs(
	ctx context.Context,
	fn func(context.Context, *sql.DB) (int64, error),
) (affected int64, succeeded, failed int, errs []string) {
	for _, dbPath := range c.Databases {
		var rows int64
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			var err error
			rows, err = fn(ctx, sqlDB)
			return err
		})
		if err != nil {
			failed++
			errs = append(errs, fmt.Sprintf("%s: %v", dbPath, err))
			models.Log.Error("Database write failed", "db", dbPath, "error", err)
			continue
		}
		succeeded++
		affected += rows
	}
	return affected, succeeded, failed, errs
}

// sendWriteResult maps a multi-database write outcome to an HTTP response.
// notFoundMessage, when set, turns a zero-row success into a 404.
func sendWriteResult(
	w http.ResponseWriter,
	successStatus int,
	affected int64,
	succeeded, failed int,
	errs []string,
	notFoundMessage string,
) {
	if failed == 0 {
		if affected == 0 && notFoundMessage != "" {
			sendError(w, http.StatusNotFound, notFoundMessage)
			return
		}
		w.WriteHeader(successStatus)
		return
	}
	if succeeded == 0 {
		sendJSON(w, http.StatusInternalServerError, map[string]any{
			"error":    "all database writes failed",
			"affected": affected,
			"failed":   failed,
			"errors":   errs,
		})
		return
	}
	sendJSON(w, http.StatusMultiStatus, map[string]any{
		"status":   "partial",
		"affected": affected,
		"failed":   failed,
		"errors":   errs,
	})
}

// searchTimeout returns the configured per-search timeout.
func (c *ServeCmd) searchTimeout() time.Duration {
	if c.SearchTimeout <= 0 {
		return 15 * time.Second
	}
	return c.SearchTimeout
}

// acquireSearchSlot blocks until a search slot is free or ctx is done. The
// returned function must be called to release the slot. A limit of 0 disables
// the check.
func (c *ServeCmd) acquireSearchSlot(ctx context.Context) (func(), error) {
	if c.MaxConcurrentSearches <= 0 {
		return func() {}, nil
	}
	c.searchSemOnce.Do(func() {
		c.searchSem = make(chan struct{}, c.MaxConcurrentSearches)
	})
	select {
	case c.searchSem <- struct{}{}:
		return func() { <-c.searchSem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
