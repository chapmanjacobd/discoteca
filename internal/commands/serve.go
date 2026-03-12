package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
	"github.com/chapmanjacobd/discoteca/web"
)

// writeJSON writes a JSON response with proper headers and error handling
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: message})
}

func init() {
	_ = mime.AddExtensionType(".js", "text/javascript")
	_ = mime.AddExtensionType(".mjs", "text/javascript")
	_ = mime.AddExtensionType(".ts", "text/javascript")
}

// LsEntry represents a directory listing entry
type LsEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Type  string `json:"type,omitempty"`
	Local bool   `json:"local"`
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

	Databases            []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
	Port                 int      `short:"p" default:"5555" help:"Port to listen on"`
	PublicDir            string   `help:"Override embedded web assets with local directory"`
	Dev                  bool     `help:"Enable development mode (auto-reload)"`
	Trashcan             bool     `help:"Enable trash/recycle page and empty bin functionality"`
	ReadOnly             bool     `help:"Disable server-side progress tracking and playlist modifications"`
	ApplicationStartTime int64    `kong:"-"`
	APIToken             string   `kong:"-"`
	thumbnailCache       sync.Map `kong:"-"`
	dbCache              sync.Map `kong:"-"`
	hasFfmpeg            bool     `kong:"-"`
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

// isPathBlacklisted checks if a path should be denied access
func (c *ServeCmd) isPathBlacklisted(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	blacklisted := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/.ssh/",
		"/.aws/",
		"/.config/",
		"/.gnupg/",
		"/root/",
		"id_rsa",
		"id_ed25519",
	}
	for _, b := range blacklisted {
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
	mux.HandleFunc("/health", c.handleHealth)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.WriteHeader(http.StatusOK)
	})

	// API routes
	mux.HandleFunc("/api/databases", c.authMiddleware(c.handleDatabases))
	mux.HandleFunc("/api/categories", c.authMiddleware(c.handleCategories))
	mux.HandleFunc("/api/genres", c.authMiddleware(c.handleGenres))
	mux.HandleFunc("/api/languages", c.authMiddleware(c.handleLanguages))
	mux.HandleFunc("/api/ratings", c.authMiddleware(c.handleRatings))
	mux.HandleFunc("/api/query", c.authMiddleware(c.handleQuery))
	mux.HandleFunc("/api/metadata", c.authMiddleware(c.handleMetadata))
	mux.HandleFunc("/api/play", c.authMiddleware(c.handlePlay))
	mux.HandleFunc("/api/delete", c.authMiddleware(c.handleDelete))
	mux.HandleFunc("/api/progress", c.authMiddleware(c.handleProgress))
	mux.HandleFunc("/api/mark-played", c.authMiddleware(c.handleMarkPlayed))
	mux.HandleFunc("/api/mark-unplayed", c.authMiddleware(c.handleMarkUnplayed))
	mux.HandleFunc("/api/rate", c.authMiddleware(c.handleRate))
	mux.HandleFunc("/api/playlists", c.authMiddleware(c.handlePlaylists))
	mux.HandleFunc("/api/playlists/items", c.authMiddleware(c.handlePlaylistItems))
	mux.HandleFunc("/api/playlists/reorder", c.authMiddleware(c.handlePlaylistReorder))
	mux.HandleFunc("/api/events", c.authMiddleware(c.handleEvents))
	mux.HandleFunc("/api/ls", c.authMiddleware(c.handleLs))
	mux.HandleFunc("/api/du", c.authMiddleware(c.handleDU))
	mux.HandleFunc("/api/episodes", c.authMiddleware(c.handleEpisodes))
	mux.HandleFunc("/api/filter-bins", c.authMiddleware(c.handleFilterBins))
	mux.HandleFunc("/api/random-clip", c.authMiddleware(c.handleRandomClip))
	mux.HandleFunc("/api/categorize/suggest", c.authMiddleware(c.handleCategorizeSuggest))
	mux.HandleFunc("/api/categorize/apply", c.authMiddleware(c.handleCategorizeApply))
	mux.HandleFunc("/api/categorize/keywords", c.authMiddleware(c.handleCategorizeKeywords))
	mux.HandleFunc("/api/categorize/defaults", c.authMiddleware(c.handleCategorizeDefaults))
	mux.HandleFunc("/api/categorize/category", c.authMiddleware(c.handleCategorizeDeleteCategory))
	mux.HandleFunc("/api/categorize/keyword", c.authMiddleware(c.handleCategorizeKeyword))
	mux.HandleFunc("/api/raw", c.authMiddleware(c.handleRaw))

	// ZIM routes
	mux.HandleFunc("/api/zim/view", c.authMiddleware(c.handleZimView))
	mux.HandleFunc("/api/zim/proxy/{port}/{rest...}", c.authMiddleware(c.handleZimProxy))

	// Special features
	mux.HandleFunc("/api/rsvp", c.authMiddleware(c.handleRSVP))
	mux.HandleFunc("/api/epub/{path...}", c.authMiddleware(c.handleEpubConvert))

	// Streaming
	mux.HandleFunc("/api/hls/playlist", c.authMiddleware(c.handleHLSPlaylist))
	mux.HandleFunc("/api/hls/segment", c.authMiddleware(c.handleHLSSegment))
	mux.HandleFunc("/api/subtitles", c.authMiddleware(c.handleSubtitles))
	mux.HandleFunc("/api/thumbnail", c.authMiddleware(c.handleThumbnail))
	mux.HandleFunc("/opds", c.authMiddleware(c.handleOPDS))

	// Trash (optional)
	if c.Trashcan {
		mux.HandleFunc("/api/trash", c.authMiddleware(c.handleTrash))
		mux.HandleFunc("/api/empty-bin", c.authMiddleware(c.handleEmptyBin))
	}

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
		http.SetCookie(w, &http.Cookie{
			Name:     "disco_token",
			Value:    c.APIToken,
			Path:     "/",
			HttpOnly: false, // Frontend needs to read it for CSRF/Auth headers
			SameSite: http.SameSiteStrictMode,
		})

		if strings.HasSuffix(r.URL.Path, ".js") || strings.HasSuffix(r.URL.Path, ".mjs") || strings.HasSuffix(r.URL.Path, ".ts") {
			w.Header().Set("Content-Type", "text/javascript")
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
func (c *ServeCmd) execDB(ctx context.Context, dbPath string, fn func(*sql.DB) error) error {
	const maxRetries = 1
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		var sqlDB *sql.DB
		if val, ok := c.dbCache.Load(dbPath); ok {
			sqlDB = val.(*sql.DB)
		} else {
			var err error
			sqlDB, err = db.Connect(dbPath)
			if err != nil {
				// Connect error might be corruption too (e.g. invalid header)
				if db.IsCorruptionError(err) && i < maxRetries {
					slog.Warn("Database corruption detected on connect, attempting repair", "db", dbPath)
					if repErr := db.Repair(dbPath); repErr != nil {
						return fmt.Errorf("repair failed: %w (original error: %v)", repErr, err)
					}
					slog.Info("Database repaired, retrying connect", "db", dbPath)
					continue
				}
				return err
			}
			c.dbCache.Store(dbPath, sqlDB)
		}

		err := fn(sqlDB)
		if err != nil {
			if db.IsCorruptionError(err) && i < maxRetries {
				c.dbCache.Delete(dbPath)
				sqlDB.Close()

				slog.Warn("Database corruption detected on query, attempting repair", "db", dbPath)
				if repErr := db.Repair(dbPath); repErr != nil {
					slog.Error("Database repair failed", "db", dbPath, "error", repErr)
					return err // Return original error if repair fails
				}
				slog.Info("Database repaired, retrying operation", "db", dbPath)
				continue
			}
			if i > 0 {
				slog.Error("Operation failed even after database repair", "db", dbPath, "error", err)
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
		return fmt.Errorf("failed to close some databases: %v", errs)
	}
	return nil
}

// Run starts the HTTP server
func (c *ServeCmd) Run(ctx *kong.Context) error {
	defer c.Close()
	models.SetupLogging(c.Verbose)
	c.ApplicationStartTime = time.Now().UnixNano()

	if envToken := os.Getenv("DISCO_API_TOKEN"); envToken != "" {
		c.APIToken = envToken
	} else {
		c.APIToken = utils.RandomString(32)
	}

	for _, dbPath := range c.Databases {
		sqlDB, err := db.Connect(dbPath)
		if err != nil {
			slog.Error("Failed to connect to database on startup", "db", dbPath, "error", err)
			continue
		}
		if err := db.InitDB(sqlDB); err != nil {
			slog.Error("Failed to initialize database", "db", dbPath, "error", err)
		}
		c.dbCache.Store(dbPath, sqlDB)
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		slog.Warn("ffmpeg not found in PATH, on-the-fly transcoding will be unavailable")
		c.hasFfmpeg = false
	} else {
		c.hasFfmpeg = true
	}

	handler := c.Mux()
	addr := fmt.Sprintf(":%d", c.Port)
	slog.Info("Server starting", "addr", addr)

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
	}
}

// parseFlags extracts query parameters into GlobalFlags
func (c *ServeCmd) parseFlags(r *http.Request) models.GlobalFlags {
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

	for _, t := range q["type"] {
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
	if q.Get("no-default-categories") == "true" {
		flags.NoDefaultCategories = true
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

// getDatabasesForQuery returns the list of databases to query based on request flags
// It validates that all requested databases are in the server's allowed list
func (c *ServeCmd) getDatabasesForQuery(flags models.GlobalFlags) ([]string, error) {
	return c.filterDatabases(flags.Databases)
}
