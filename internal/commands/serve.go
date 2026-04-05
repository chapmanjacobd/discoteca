package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
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

func init() {
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
	Local     bool   `json:"local"`
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

	Databases            []string `help:"SQLite database files"                                                           required:"" arg:"" type:"existingfile"`
	Port                 int      `help:"Port to listen on"                                                                                                      default:"5555" short:"p"`
	PublicDir            string   `help:"Override embedded web assets with local directory"`
	Dev                  bool     `help:"Enable development mode (auto-reload)"`
	ReadOnly             bool     `help:"Disable write operations (progress tracking, playlist modifications, deletions)"`
	NoBrowser            bool     `help:"Don't open browser on startup"`
	ApplicationStartTime int64    `                                                                                                                                                       kong:"-"`
	APIToken             string   `                                                                                                                                                       kong:"-"`
	thumbnailCache       sync.Map `                                                                                                                                                       kong:"-"`
	dbCache              sync.Map `                                                                                                                                                       kong:"-"`
	hasFfmpeg            bool     `                                                                                                                                                       kong:"-"`
}

// authMiddleware validates API token for authenticated endpoints
func (c *ServeCmd) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Disco-Token")
		if token == "" {
			// Also check cookie for same-origin convenience
			if cookie, err := r.Cookie("disco_token"); err == nil {
				token = cookie.Value
			}
		}

		if token != c.APIToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
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

// Mux creates the HTTP request multiplexer with all routes
func (c *ServeCmd) Mux() http.Handler {
	if c.APIToken == "" {
		c.APIToken = "test-token"
	}
	mux := http.NewServeMux()

	// Health and Static
	mux.HandleFunc("/health", c.HandleHealth)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.WriteHeader(http.StatusOK)
	})

	// API routes
	mux.HandleFunc("/api/databases", c.authMiddleware(c.HandleDatabases))
	mux.HandleFunc("/api/categories", c.authMiddleware(c.HandleCategories))
	mux.HandleFunc("/api/genres", c.authMiddleware(c.HandleGenres))
	mux.HandleFunc("/api/languages", c.authMiddleware(c.HandleLanguages))
	mux.HandleFunc("/api/ratings", c.authMiddleware(c.HandleRatings))
	mux.HandleFunc("/api/query", c.authMiddleware(c.HandleQuery))
	mux.HandleFunc("/api/metadata", c.authMiddleware(c.HandleMetadata))
	mux.HandleFunc("/api/play", c.authMiddleware(c.HandlePlay))
	mux.HandleFunc("/api/delete", c.authMiddleware(c.HandleDelete))
	mux.HandleFunc("/api/progress", c.authMiddleware(c.HandleProgress))
	mux.HandleFunc("/api/mark-played", c.authMiddleware(c.HandleMarkPlayed))
	mux.HandleFunc("/api/mark-unplayed", c.authMiddleware(c.HandleMarkUnplayed))
	mux.HandleFunc("/api/rate", c.authMiddleware(c.HandleRate))
	mux.HandleFunc("/api/playlists", c.authMiddleware(c.HandlePlaylists))
	mux.HandleFunc("/api/playlists/items", c.authMiddleware(c.HandlePlaylistItems))
	mux.HandleFunc("/api/playlists/reorder", c.authMiddleware(c.HandlePlaylistReorder))
	mux.HandleFunc("/api/events", c.authMiddleware(c.HandleEvents))
	mux.HandleFunc("/api/ls", c.authMiddleware(c.HandleLs))
	mux.HandleFunc("/api/du", c.authMiddleware(c.HandleDU))
	mux.HandleFunc("/api/episodes", c.authMiddleware(c.HandleEpisodes))
	mux.HandleFunc("/api/filter-bins", c.authMiddleware(c.HandleFilterBins))
	mux.HandleFunc("/api/random-clip", c.authMiddleware(c.HandleRandomClip))
	mux.HandleFunc("/api/categorize/suggest", c.authMiddleware(c.HandleCategorizeSuggest))
	mux.HandleFunc("/api/categorize/apply", c.authMiddleware(c.HandleCategorizeApply))
	mux.HandleFunc("/api/categorize/keywords", c.authMiddleware(c.HandleCategorizeKeywords))
	mux.HandleFunc("/api/categorize/category", c.authMiddleware(c.HandleCategorizeDeleteCategory))
	mux.HandleFunc("/api/categorize/keyword", c.authMiddleware(c.HandleCategorizeKeyword))
	mux.HandleFunc("/api/raw", c.authMiddleware(c.HandleRaw))

	// Query statistics / slow query dashboard
	mux.HandleFunc("/api/queries", c.authMiddleware(c.HandleQueries))

	// ZIM routes
	mux.HandleFunc("/api/zim/view", c.authMiddleware(c.HandleZimView))
	mux.HandleFunc("/api/zim/proxy/{port}/{rest...}", c.authMiddleware(c.HandleZimProxy))

	// Special features
	mux.HandleFunc("/api/rsvp", c.authMiddleware(c.HandleRSVP))
	mux.HandleFunc("/api/epub/{path...}", c.authMiddleware(c.HandleEpubConvert))

	// Streaming
	mux.HandleFunc("/api/hls/playlist", c.authMiddleware(c.HandleHLSPlaylist))
	mux.HandleFunc("/api/hls/segment", c.authMiddleware(c.HandleHLSSegment))
	mux.HandleFunc("/api/subtitles", c.authMiddleware(c.HandleSubtitles))
	mux.HandleFunc("/api/thumbnail", c.authMiddleware(c.HandleThumbnail))
	mux.HandleFunc("/opds", c.authMiddleware(c.HandleOPDS))

	// Trash endpoints (respects ReadOnly mode)
	mux.HandleFunc("/api/trash", c.authMiddleware(c.HandleTrash))
	mux.HandleFunc("/api/empty-bin", c.authMiddleware(c.HandleEmptyBin))

	// Static assets
	mux.HandleFunc("/lib/", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// Serve other static files
	fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set cookie on every load so the frontend has access to it
		// Note: HttpOnly is set to true for security (prevents XSS token theft)
		// The auth middleware checks both X-Disco-Token header and the cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "disco_token",
			Value:    c.APIToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		// Set Cache-Control headers for static assets (1 week cache)
		// Disable caching in dev mode for easier development
		if !c.Dev {
			if strings.HasSuffix(r.URL.Path, ".js") || strings.HasSuffix(r.URL.Path, ".mjs") ||
				strings.HasSuffix(r.URL.Path, ".ts") {

				w.Header().Set("Content-Type", "text/javascript")
				w.Header().Set("Cache-Control", "public, max-age=604800") // 1 week
			} else if strings.HasSuffix(
				r.URL.Path,
				".css",
			) {

				w.Header().Set("Cache-Control", "public, max-age=604800") // 1 week
			} else if strings.HasSuffix(
				r.URL.Path,
				".html",
			) {

				w.Header().Set("Cache-Control", "no-cache, must-revalidate") // HTML should not be cached
			} else if strings.HasPrefix(
				r.URL.Path,
				"/lib/",
			) {

				w.Header().Set("Cache-Control", "public, max-age=604800") // 1 week for library files
			}
		} else {
			// Dev mode: disable caching for all static assets
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		if c.PublicDir != "" {
			http.FileServer(http.Dir(c.PublicDir)).ServeHTTP(w, r)
		} else {
			http.FileServer(http.FS(web.FS)).ServeHTTP(w, r)
		}
	})

	mux.Handle("/", fileHandler)
	return mux
}

// execDB connects to the database and executes fn. If a corruption error occurs,
// it attempts to repair the database and retries the operation once.
func (c *ServeCmd) execDB(ctx context.Context, dbPath string, fn func(ctx context.Context, sqlDB *sql.DB) error) error {
	const maxRetries = 1
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		var sqlDB *sql.DB
		if val, ok := c.dbCache.Load(dbPath); ok {
			sqlDB = val.(*sql.DB)
		} else {
			// Create a new connection
			newDB, err := db.Connect(ctx, dbPath)
			if err != nil {
				// Connect error might be corruption too (e.g. invalid header)
				if db.IsCorruptionError(err) && i < maxRetries {
					models.Log.Warn("Database corruption detected on connect, attempting repair", "db", dbPath)
					if repErr := db.Repair(ctx, dbPath); repErr != nil {
						return fmt.Errorf("repair failed: %w (original error: %w)", repErr, err)
					}
					models.Log.Info("Database repaired, retrying connect", "db", dbPath)
					continue
				}
				return err
			}

			// Use LoadOrStore to avoid race condition where multiple goroutines
			// create duplicate connections. If we lose the race, close our connection.
			sqlDB = newDB
			if loaded, ok := c.dbCache.LoadOrStore(dbPath, sqlDB); ok {
				// Another goroutine stored a connection first; use theirs and close ours
				sqlDB = loaded.(*sql.DB)
				newDB.Close()
			}
		}

		err := fn(ctx, sqlDB)
		if err != nil {
			if db.IsCorruptionError(err) && i < maxRetries {
				c.dbCache.Delete(dbPath)
				sqlDB.Close()

				models.Log.Warn("Database corruption detected on query, attempting repair", "db", dbPath)
				if repErr := db.Repair(ctx, dbPath); repErr != nil {
					models.Log.Error("Database repair failed", "db", dbPath, "error", repErr)
					return err // Return original error if repair fails
				}
				models.Log.Info("Database repaired, retrying operation", "db", dbPath)
				continue
			}
			if i > 0 {
				models.Log.Error("Operation failed even after database repair", "db", dbPath, "error", err)
			}
			return err
		}
		return nil
	}
	return lastErr
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

// Run starts the HTTP server
func (c *ServeCmd) Run(ctx context.Context) error {
	defer c.Close()
	models.SetupLogging(c.Verbose)
	db.SetFtsEnabled(true)

	for _, dbPath := range c.Databases {
		sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
		if err == nil {
			sqlDB.Close()
		}
	}

	c.ApplicationStartTime = time.Now().UnixNano()

	if envToken := os.Getenv("DISCO_API_TOKEN"); envToken != "" {
		c.APIToken = envToken
	} else {
		c.APIToken = utils.RandomString(32)
	}

	for _, dbPath := range c.Databases {
		sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
		if err != nil {
			models.Log.Error("Failed to connect to database on startup", "db", dbPath, "error", err)
			continue
		}
		c.dbCache.Store(dbPath, sqlDB)
	}

	// Run maintenance on all databases (refresh folder_stats and FTS if needed)
	// This runs asynchronously so it doesn't block server startup
	go func() {
		config := db.DefaultMaintenanceConfig()
		for _, dbPath := range c.Databases {
			if sqlDB, ok := c.dbCache.Load(dbPath); ok {
				if err := db.RunMaintenance(ctx, sqlDB.(*sql.DB), config, dbPath); err != nil {
					models.Log.Error("Maintenance failed", "db", dbPath, "error", err)
				}
			}
		}
	}()

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		models.Log.Warn("ffmpeg not found in PATH, on-the-fly transcoding will be unavailable")
		c.hasFfmpeg = false
	} else {
		c.hasFfmpeg = true
	}

	handler := c.Mux()

	addr := fmt.Sprintf(":%d", c.Port)
	baseURL := fmt.Sprintf("http://localhost:%d", c.Port)
	models.Log.Info("Server starting", "addr", baseURL)

	// Open browser unless --no-browser is passed
	if !c.NoBrowser {
		go func() {
			// Give server time to start
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
				cmd := exec.CommandContext(context.Background(), openCmd, openArgs...)
				if err := cmd.Start(); err != nil {
					models.Log.Debug("Failed to open browser", "error", err)
				}
			}
		}()
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

// parseFlags extracts query parameters into GlobalFlags
func (c *ServeCmd) ParseFlags(r *http.Request) models.GlobalFlags {
	flags := c.GetGlobalFlags()
	q := r.URL.Query()
	if search := q.Get("search"); search != "" {
		flags.Search = strings.Fields(search)
	}
	if categories := q["category"]; len(categories) > 0 {
		flags.Category = categories
	} else if category := q.Get("category"); category != "" {
		flags.Category = []string{category}
	}
	if genre := q.Get("genre"); genre != "" {
		flags.Genre = genre
	}
	if languages := q["language"]; len(languages) > 0 {
		flags.Language = languages
	}
	if paths := q.Get("paths"); paths != "" {
		flags.Paths = strings.Split(paths, ",")
	}
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
	if sortBy := q.Get("sort"); sortBy != "" {
		flags.SortBy = sortBy
		if sortBy == "random" {
			flags.Random = true
		}
	}
	// Support complex sorting with sort_fields array (JSON) or comma-separated string
	if sortFields := q.Get("sort_fields"); sortFields != "" {
		// If it starts with '[', treat as JSON array
		if strings.HasPrefix(sortFields, "[") {
			var fieldList []string
			if err := json.Unmarshal([]byte(sortFields), &fieldList); err == nil {
				// Convert array to comma-separated format
				flags.PlayInOrder = strings.Join(fieldList, ",")
			}
		} else {
			// Already comma-separated or single field
			flags.PlayInOrder = sortFields
		}
	}
	// Also support sort_order for explicit direction (overrides individual field directions)
	if sortDesc := q.Get("sort_desc"); sortDesc != "" {
		// Comma-separated list of fields to sort descending
		descFields := make(map[string]bool)
		for f := range strings.SplitSeq(sortDesc, ",") {
			descFields[strings.TrimSpace(f)] = true
		}
		// If PlayInOrder is set, prepend '-' to descending fields
		if flags.PlayInOrder != "" {
			var newOrder []string
			for f := range strings.SplitSeq(flags.PlayInOrder, ",") {
				f = strings.TrimSpace(f)
				// Remove existing direction prefix
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
	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			flags.Limit = l
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			flags.Offset = o
		}
	}
	if minSize := q.Get("min_size"); minSize != "" {
		flags.Size = append(flags.Size, ">"+minSize+"MB")
	}
	if maxSize := q.Get("max_size"); maxSize != "" {
		flags.Size = append(flags.Size, "<"+maxSize+"MB")
	}
	if sizes := q["size"]; len(sizes) > 0 {
		flags.Size = append(flags.Size, sizes...)
	}
	if minDuration := q.Get("min_duration"); minDuration != "" {
		flags.Duration = append(flags.Duration, ">"+minDuration+"min")
	}
	if maxDuration := q.Get("max_duration"); maxDuration != "" {
		flags.Duration = append(flags.Duration, "<"+maxDuration+"min")
	}
	if durations := q["duration"]; len(durations) > 0 {
		flags.Duration = append(flags.Duration, durations...)
	}

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

	if episodes := q.Get("episodes"); episodes != "" {
		flags.FileCounts = episodes
	}
	if minScore := q.Get("min_score"); minScore != "" {
		flags.Where = append(flags.Where, "score >= "+minScore)
	}
	if maxScore := q.Get("max_score"); maxScore != "" {
		flags.Where = append(flags.Where, "score <= "+maxScore)
	}
	if unplayed := q.Get("unplayed"); unplayed == "true" {
		flags.Where = append(flags.Where, "COALESCE(play_count, 0) = 0 AND COALESCE(playhead, 0) = 0")
	}
	if all := q.Get("all"); all == "true" {
		flags.All = true
	}

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
	if q.Get("trash") == "true" {
		flags.OnlyDeleted = true
		flags.HideDeleted = false
	}
	if episodes := q["episodes"]; len(episodes) > 0 {
		flags.FileCounts = strings.Join(episodes, ",")
	} else if episodes := q.Get("episodes"); episodes != "" {
		flags.FileCounts = episodes
	}
	if groupBy := q.Get("group_by"); groupBy == "parent" {
		flags.GroupByParent = true
	}

	// Parse search type (FTS vs substring)
	if searchType := q.Get("search_type"); searchType == "substring" {
		flags.FTS = false
	} else if searchType == "fts" {
		flags.FTS = true
	}

	// Parse database filter from request
	if dbs := q["db"]; len(dbs) > 0 {
		flags.Databases = dbs
	}

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

	// If all databases were filtered out, return empty list (no results)
	if len(filtered) == 0 {
		return []string{}, nil
	}

	return filtered, nil
}

// getDBs returns the list of databases to query based on request flags
// It validates that all requested databases are in the server's allowed list
func (c *ServeCmd) getDBs(flags models.GlobalFlags) ([]string, error) {
	return c.filterDatabases(flags.Databases)
}
