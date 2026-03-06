package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discotheque/internal/aggregate"
	database "github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/query"
	"github.com/chapmanjacobd/discotheque/internal/utils"
	"github.com/chapmanjacobd/discotheque/web"
)

type LsEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Type  string `json:"type,omitempty"`
	Local bool   `json:"local"`
}

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

func (c *ServeCmd) isPathBlacklisted(path string) bool {
	p := strings.ToLower(path)
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

func (c *ServeCmd) Mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/databases", c.authMiddleware(c.handleDatabases))
	mux.HandleFunc("/api/categories", c.authMiddleware(c.handleCategories))
	mux.HandleFunc("/api/genres", c.authMiddleware(c.handleGenres))
	mux.HandleFunc("/api/ratings", c.authMiddleware(c.handleRatings))
	mux.HandleFunc("/api/query", c.authMiddleware(c.handleQuery))
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

	mux.HandleFunc("/api/zim/view", c.authMiddleware(c.handleZimView))
	mux.HandleFunc("/api/zim/proxy/{port}/{rest...}", c.authMiddleware(c.handleZimProxy))

	mux.HandleFunc("/api/rsvp", c.authMiddleware(c.handleRSVP))

	mux.HandleFunc("/api/hls/playlist", c.authMiddleware(c.handleHLSPlaylist))
	mux.HandleFunc("/api/hls/segment", c.authMiddleware(c.handleHLSSegment))
	mux.HandleFunc("/api/subtitles", c.authMiddleware(c.handleSubtitles))
	mux.HandleFunc("/api/thumbnail", c.authMiddleware(c.handleThumbnail))
	mux.HandleFunc("/opds", c.authMiddleware(c.handleOPDS))

	if c.Trashcan {
		mux.HandleFunc("/api/trash", c.authMiddleware(c.handleTrash))
		mux.HandleFunc("/api/empty-bin", c.authMiddleware(c.handleEmptyBin))
	}

	// Serve static files
	fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set cookie on every load so the frontend has access to it
		http.SetCookie(w, &http.Cookie{
			Name:     "disco_token",
			Value:    c.APIToken,
			Path:     "/",
			HttpOnly: false, // Frontend needs to read it for CSRF/Auth headers
			SameSite: http.SameSiteStrictMode,
		})

		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
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
			sqlDB, err = database.Connect(dbPath)
			if err != nil {
				// Connect error might be corruption too (e.g. invalid header)
				if database.IsCorruptionError(err) && i < maxRetries {
					slog.Warn("Database corruption detected on connect, attempting repair", "db", dbPath)
					if repErr := database.Repair(dbPath); repErr != nil {
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
			if database.IsCorruptionError(err) && i < maxRetries {
				c.dbCache.Delete(dbPath)
				sqlDB.Close()

				slog.Warn("Database corruption detected on query, attempting repair", "db", dbPath)
				if repErr := database.Repair(dbPath); repErr != nil {
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

func (c *ServeCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	c.ApplicationStartTime = time.Now().UnixNano()
	c.APIToken = utils.RandomString(32)

	for _, dbPath := range c.Databases {
		sqlDB, err := database.Connect(dbPath)
		if err != nil {
			slog.Error("Failed to connect to database on startup", "db", dbPath, "error", err)
			continue
		}
		if err := InitDB(sqlDB); err != nil {
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

func (c *ServeCmd) handleDatabases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := models.DatabaseInfo{
		Databases: c.Databases,
		Trashcan:  c.Trashcan,
		ReadOnly:  c.ReadOnly,
		Dev:       c.Dev,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode databases response", "error", err)
	}
}

// handleCategories returns a list of categories and their media counts.
// GET /api/categories
func (c *ServeCmd) handleCategories(w http.ResponseWriter, r *http.Request) {
	counts := make(map[string]int64)
	isCustom := make(map[string]bool)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)

			// 1. Get categories already assigned to media
			rows, err := queries.GetUsedCategories(r.Context())
			if err != nil {
				return err
			}
			for _, row := range rows {
				if row.Categories.Valid {
					trimmed := strings.Trim(row.Categories.String, ";")
					if trimmed == "" {
						continue
					}
					cats := strings.SplitSeq(trimmed, ";")
					for cat := range cats {
						if cat != "" {
							counts[cat] += row.Count
						}
					}
				}
			}

			// 2. Get categories from custom keywords
			customCats, err := queries.GetCustomCategories(r.Context())
			if err == nil {
				for _, cat := range customCats {
					isCustom[cat] = true
					if _, ok := counts[cat]; !ok {
						counts[cat] = 0
					}
				}
			}

			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch categories", "db", dbPath, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Failed to fetch categories"})
			return
		}
	}

	// 3. Add Uncategorized count
	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			var count int64
			err := sqlDB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM media WHERE time_deleted = 0 AND (categories IS NULL OR categories = '')").Scan(&count)
			if err == nil {
				counts["Uncategorized"] += count
			}
			return nil
		})
	}

	var res []models.CatStat
	for k, v := range counts {
		res = append(res, models.CatStat{Category: k, Count: v})
	}

	sort.Slice(res, func(i, j int) bool {
		if res[i].Category == "Uncategorized" {
			return false
		}
		if res[j].Category == "Uncategorized" {
			return true
		}
		if res[i].Count != res[j].Count {
			return res[i].Count > res[j].Count
		}
		return res[i].Category < res[j].Category
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// handleRatings returns rating statistics.
// GET /api/ratings
func (c *ServeCmd) handleRatings(w http.ResponseWriter, r *http.Request) {
	counts := make(map[int64]int64)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			stats, err := queries.GetRatingStats(r.Context())
			if err != nil {
				return err
			}
			for _, s := range stats {
				counts[s.Rating] = counts[s.Rating] + s.Count
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch ratings", "db", dbPath, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Failed to fetch ratings"})
			return
		}
	}

	var res []models.RatStat
	res = make([]models.RatStat, 0)
	for k, v := range counts {
		res = append(res, models.RatStat{Rating: k, Count: v})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Rating > res[j].Rating
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

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
	return flags
}

// handleQuery handles media searching and filtering.
// GET /api/query?search=...&category=...&rating=...&sort=...&limit=...&offset=...
func (c *ServeCmd) handleQuery(w http.ResponseWriter, r *http.Request) {
	flags := c.parseFlags(r)
	q := r.URL.Query()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Pre-resolve percentiles so Count matches Query results
	resolvedFlags, err := query.ResolvePercentileFlags(ctx, c.Databases, flags)
	if err == nil {
		flags = resolvedFlags
	}

	if q.Get("view") == "captions" || q.Get("captions") == "true" {
		var media []models.MediaWithDB
		queryStr := strings.Join(flags.Search, " ")
		limit := flags.Limit
		if limit <= 0 {
			limit = 100
		}
		if flags.All {
			limit = 1000000
		}

		// Check if aggregation is requested
		aggregate := q.Get("aggregate") == "true"

		for _, dbPath := range c.Databases {
			err := c.execDB(ctx, dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				var rows []database.SearchCaptionsRow
				var err error

				if queryStr != "" {
					rows, err = queries.SearchCaptions(ctx, database.SearchCaptionsParams{
						Query: queryStr,
						Limit: int64(limit),
					})
				} else {
					var rawRows []database.GetAllCaptionsRow
					rawRows, err = queries.GetAllCaptions(ctx, int64(limit))
					for _, r := range rawRows {
						rows = append(rows, database.SearchCaptionsRow(r))
					}
				}

				if err != nil {
					return err
				}

				if aggregate {
					// Aggregate captions by media path
					aggregated := make(map[string]*models.MediaWithDB)
					for _, row := range rows {
						path := row.MediaPath
						if _, ok := aggregated[path]; !ok {
							aggregated[path] = &models.MediaWithDB{
								Media: models.Media{
									Path:     path,
									Type:     models.NullStringPtr(row.Type),
									Title:    models.NullStringPtr(row.Title),
									Size:     models.NullInt64Ptr(row.Size),
									Duration: models.NullInt64Ptr(row.Duration),
								},
								DB: dbPath,
							}
						}
						// Accumulate caption data
						stat := aggregated[path]
						if stat.CaptionText == "" {
							stat.CaptionText = row.Text.String
						}
						if stat.CaptionTime == 0 {
							stat.CaptionTime = row.Time.Float64
						}
						// Count captions
						stat.CaptionCount++
						// Accumulate caption duration (in seconds, stored as int64)
						if row.Time.Valid {
							stat.CaptionDuration += int64(row.Time.Float64)
						}
					}
					for _, m := range aggregated {
						media = append(media, *m)
					}
				} else {
					// Return individual caption rows (legacy behavior)
					for _, row := range rows {
						m := models.MediaWithDB{
							Media: models.Media{
								Path:  row.MediaPath,
								Type:  models.NullStringPtr(row.Type),
								Title: models.NullStringPtr(row.Title),
							},
							DB:          dbPath,
							CaptionText: row.Text.String,
							CaptionTime: row.Time.Float64,
						}
						media = append(media, m)
					}
				}
				return nil
			})
			if err != nil {
				slog.Error("Caption fetch failed", "db", dbPath, "error", err)
			}
		}

		totalCount := len(media)

		// Pagination for captions (since we fetched them all or up to limit per DB)
		if !flags.All && flags.Limit > 0 && !aggregate {
			start := flags.Offset
			if start > len(media) {
				media = []models.MediaWithDB{}
			} else {
				end := min(start+flags.Limit, len(media))
				media = media[start:end]
			}
		}

		w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(media)
		return
	}

	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		slog.Error("Query failed", "dbs", c.Databases, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Query failed: " + err.Error()})
		return
	}

	// Check if filter counts are requested
	includeCounts := q.Get("include_counts") == "true"
	var filterCounts *models.FilterBinsResponse
	if includeCounts {
		filterCounts = c.calculateFilterCounts(ctx, flags)
	}

	// Caption enrichment for main media grid
	if flags.WithCaptions && len(flags.Search) > 0 {
		queryStr := strings.Join(flags.Search, " ")
		for _, dbPath := range c.Databases {
			err := c.execDB(ctx, dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				// Enrich existing results with matching caption segments
				rows, err := queries.SearchCaptions(ctx, database.SearchCaptionsParams{
					Query: queryStr,
					Limit: 5, // Just get a few per DB for enrichment
				})
				if err != nil {
					return err
				}

				mediaMap := make(map[string]int)
				for i, m := range media {
					mediaMap[m.Path] = i
				}

				for _, row := range rows {
					if idx, ok := mediaMap[row.MediaPath]; ok {
						if media[idx].CaptionText == "" {
							media[idx].CaptionText = row.Text.String
							media[idx].CaptionTime = row.Time.Float64
						}
					}
				}
				return nil
			})
			if err != nil {
				slog.Error("Caption enrichment failed", "db", dbPath, "error", err)
			}
		}
	}

	totalCount, err := query.MediaQueryCount(ctx, c.Databases, flags)
	if err != nil {
		slog.Error("Count query failed", "dbs", c.Databases, "error", err)
		// Don't fail the whole request just for count
	}

	if c.hasFfmpeg {
		for i := range media {
			media[i].Transcode = utils.GetTranscodeStrategy(media[i].Media).NeedsTranscode
		}
	}

	query.SortMedia(media, flags)

	w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// Return counts with media if requested
	if includeCounts && filterCounts != nil {
		response := map[string]any{
			"items":  media,
			"counts": filterCounts,
		}
		json.NewEncoder(w).Encode(response)
	} else {
		json.NewEncoder(w).Encode(media)
	}
}

// handlePlay triggers local playback of a media file via mpv.
// POST /api/play
// Body: {"path": "..."}
func (c *ServeCmd) handlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.PlayResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	if !strings.HasPrefix(req.Path, "http") && !utils.FileExists(req.Path) {
		slog.Warn("File not found, marking as deleted in databases", "path", req.Path)
		c.markDeletedInAllDBs(r.Context(), req.Path, true)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "File not found"})
		return
	}

	// Trigger local playback
	slog.Info("Playing", "path", req.Path)
	cmd := exec.Command("mpv", req.Path)
	// We run it in background and don't wait for it
	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start mpv", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Failed to start playback: " + err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (c *ServeCmd) markDeletedInAllDBs(ctx context.Context, path string, deleted bool) {
	if c.ReadOnly {
		return
	}
	var deleteTime int64 = 0
	if deleted {
		deleteTime = time.Now().Unix()
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(ctx, dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			return queries.MarkDeleted(ctx, database.MarkDeletedParams{
				Path:        path,
				TimeDeleted: sql.NullInt64{Int64: deleteTime, Valid: deleted},
			})
		})
		if err != nil {
			slog.Error("Failed to mark file as deleted", "db", dbPath, "path", path, "error", err)
		}
	}
}

// handleDelete marks a file as deleted or restores it in all databases.
// POST /api/delete
// Body: {"path": "...", "restore": bool}
func (c *ServeCmd) handleDelete(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Read-only mode"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	c.markDeletedInAllDBs(r.Context(), req.Path, !req.Restore)
	w.WriteHeader(http.StatusOK)
}

// handleProgress updates the playback progress for a media file.
// POST /api/progress
// Body: {"path": "...", "playhead": int64, "completed": bool}
func (c *ServeCmd) handleProgress(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Read-only mode"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.ProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	now := time.Now().Unix()
	increment := 0
	if req.Completed {
		increment = 1
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			// Use raw SQL to update progress to avoid complex sqlc param mapping if not existing
			// We want to increment play_count only once per session ideally, but for now we follow simple logic
			if _, err := sqlDB.ExecContext(r.Context(), `
			UPDATE media
			SET time_last_played = ?,
			    time_first_played = COALESCE(time_first_played, ?),
			    playhead = ?,
			    play_count = COALESCE(play_count, 0) + ?
			WHERE path = ?`,
				now, now, req.Playhead, increment, req.Path); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to update progress", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleMarkUnplayed resets play count and progress for a media file.
// POST /api/mark-unplayed
// Body: {"path": "..."}
func (c *ServeCmd) handleMarkUnplayed(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Read-only mode"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			if _, err := sqlDB.ExecContext(r.Context(), `
			UPDATE media
			SET play_count = 0,
			    playhead = 0,
			    time_last_played = 0
			WHERE path = ?`,
				req.Path); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to mark as unplayed", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleMarkPlayed increments play count and resets progress for a media file.
// POST /api/mark-played
// Body: {"path": "..."}
func (c *ServeCmd) handleMarkPlayed(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Read-only mode"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	now := time.Now().Unix()
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			if _, err := sqlDB.ExecContext(r.Context(), `
			UPDATE media
			SET time_last_played = ?,
			    time_first_played = COALESCE(time_first_played, ?),
			    play_count = COALESCE(play_count, 0) + 1,
			    playhead = 0
			WHERE path = ?`,
				now, now, req.Path); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to mark as played", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleRate updates the rating for a media file.
// POST /api/rate
// Body: {"path": "...", "score": float64}
func (c *ServeCmd) handleRate(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Read-only mode"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req struct {
		Path  string  `json:"path"`
		Score float64 `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			if _, err := sqlDB.ExecContext(r.Context(), "UPDATE media SET score = ? WHERE path = ?", req.Score, req.Path); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to update rating", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (c *ServeCmd) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial comment to establish connection
	fmt.Fprintf(w, ": keep-alive\n\n")
	flusher.Flush()

	if c.Dev {
		fmt.Fprintf(w, "data: %d\n\n", c.ApplicationStartTime)
		flusher.Flush()
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (c *ServeCmd) handleLs(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	isPartial := false
	if strings.HasPrefix(path, "./") {
		isPartial = true
		path = strings.TrimPrefix(path, "./")
	} else if !strings.HasPrefix(path, "/") {
		// If it doesn't start with / or ./, treat as partial from current context
		isPartial = true
	}

	// Split into dir and base for better contextual suggestions
	searchDir := ""
	searchBase := path
	if isPartial {
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash != -1 {
			searchDir = path[:lastSlash+1]
			searchBase = path[lastSlash+1:]
		}
	} else {
		searchDir = path
		searchBase = ""
		if !strings.HasSuffix(path, "/") {
			lastSlash := strings.LastIndex(path, "/")
			if lastSlash != -1 {
				searchDir = path[:lastSlash+1]
				searchBase = path[lastSlash+1:]
			} else {
				searchDir = "/"
				searchBase = path[1:]
			}
		}
	}

	resultsMap := make(map[string]LsEntry)
	counts := make(map[string]int)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			var rows *sql.Rows
			var err error

			if isPartial {
				if searchDir == "" {
					rows, err = sqlDB.QueryContext(r.Context(), `
						SELECT path, type FROM media
						WHERE time_deleted = 0
						  AND path LIKE '%' || ? || '%'
						LIMIT 500`, searchBase)
				} else {
					rows, err = sqlDB.QueryContext(r.Context(), `
						SELECT path, type FROM media
						WHERE time_deleted = 0
						  AND path LIKE '%' || ? || '%' || ? || '%'
						LIMIT 500`, searchDir, searchBase)
				}
			} else {
				if searchBase == "" {
					rows, err = sqlDB.QueryContext(r.Context(), `
						SELECT path, type FROM media
						WHERE time_deleted = 0
						  AND path LIKE ? || '%'
						LIMIT 500`, searchDir)
				} else {
					rows, err = sqlDB.QueryContext(r.Context(), `
						SELECT path, type FROM media
						WHERE time_deleted = 0
						  AND path LIKE ? || '%'
						  AND path LIKE '%' || ? || '%'
						LIMIT 500`, searchDir, searchBase)
				}
			}

			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var p, t sql.NullString
				if err := rows.Scan(&p, &t); err == nil && p.Valid {
					fullPath := p.String

					if isPartial && path == "" {
						// Special case: empty partial search (./)
						segments := strings.Split(strings.Trim(fullPath, "/"), "/")
						current := "/"
						for _, seg := range segments {
							if seg == "" {
								continue
							}
							entryName := seg
							entryPath := current + seg + "/"
							if !strings.HasSuffix(fullPath, "/") && seg == segments[len(segments)-1] {
								entryPath = current + seg
								counts[entryPath]++
								if _, ok := resultsMap[entryPath]; !ok {
									resultsMap[entryPath] = LsEntry{Name: entryName, Path: entryPath, IsDir: false, Type: t.String}
								}
								break
							}
							counts[entryPath]++
							if _, ok := resultsMap[entryPath]; !ok {
								resultsMap[entryPath] = LsEntry{Name: entryName, Path: entryPath, IsDir: true}
							}
							current = entryPath
						}
						continue
					}

					var entryName string
					var entryPath string
					var isDir bool

					if isPartial {
						matchStr := path
						if searchDir != "" {
							matchStr = searchDir
						}

						idx := strings.Index(fullPath, matchStr)
						if idx == -1 {
							continue
						}

						var prefix string
						var remaining string

						if strings.HasSuffix(matchStr, "/") {
							// Suggest contents of the matched directory
							prefix = fullPath[:idx+len(matchStr)]
							remaining = fullPath[idx+len(matchStr):]
						} else {
							// Suggest the segment containing the match
							lastSlash := strings.LastIndex(fullPath[:idx], "/")
							if lastSlash == -1 {
								lastSlash = 0
							}
							prefix = fullPath[:lastSlash+1]
							remaining = fullPath[lastSlash+1:]
						}

						if remaining == "" {
							continue
						}

						if before, _, ok := strings.Cut(remaining, "/"); ok {
							entryName = before
							isDir = true
							entryPath = prefix + entryName + "/"
						} else {
							entryName = remaining
							isDir = false
							entryPath = prefix + entryName
						}
					} else {
						// Absolute path
						if !strings.HasPrefix(fullPath, searchDir) {
							continue
						}
						suffix := strings.TrimPrefix(fullPath, searchDir)
						if suffix == "" {
							continue
						}
						if before, _, ok := strings.Cut(suffix, "/"); ok {
							entryName = before
							isDir = true
							entryPath = searchDir + entryName + "/"
						} else {
							entryName = suffix
							isDir = false
							entryPath = searchDir + entryName
						}
					}

					if entryName == "" {
						continue
					}

					counts[entryPath]++
					if existing, ok := resultsMap[entryPath]; ok {
						if !existing.IsDir && isDir {
							resultsMap[entryPath] = LsEntry{
								Name:  entryName,
								Path:  entryPath,
								IsDir: true,
							}
						}
					} else {
						resultsMap[entryPath] = LsEntry{
							Name:  entryName,
							Path:  entryPath,
							IsDir: isDir,
							Type:  t.String,
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			slog.Error("handleLs DB query failed", "db", dbPath, "error", err)
		}
	}

	var results []LsEntry
	for _, entry := range resultsMap {
		results = append(results, entry)
	}

	sort.Slice(results, func(i, j int) bool {
		countI := counts[results[i].Path]
		countJ := counts[results[j].Path]
		if countI != countJ {
			return countI > countJ // Best matches first
		}
		if results[i].IsDir != results[j].IsDir {
			return results[i].IsDir
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	if len(results) > 20 {
		results = results[:20]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (c *ServeCmd) handleDU(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	// Clean and normalize the path
	cleanPath := filepath.Clean(path)
	if cleanPath == "." {
		cleanPath = ""
	}

	// Calculate the depth of current path (number of path components)
	// "" or "/" = depth 0
	// "/media" = depth 1
	// "/media/videos" = depth 2
	currentDepth := 0
	if cleanPath != "" && cleanPath != "/" {
		parts := strings.Split(cleanPath, string(filepath.Separator))
		if len(parts) > 0 && parts[0] == "" {
			parts = parts[1:]
		}
		currentDepth = len(parts)
	}
	targetDepth := currentDepth + 1 // We want to show children at this depth

	// Fetch files that are children of the current path
	// We aggregate in Go to avoid complex SQL
	querySQL := `
		SELECT path, size, duration, type
		FROM media
		WHERE time_deleted = 0
		  AND (path LIKE ? OR ? = '')
		  AND (length(path) - length(replace(path, '/', ''))) >= ?
	`

	slog.Debug("DU query", "cleanPath", cleanPath, "targetDepth", targetDepth, "sql", querySQL)

	// Separate maps for direct files and folders
	directFiles := make(map[string]*models.MediaWithDB)
	folders := make(map[string]*models.FolderStats)

	// Parse type filter
	typeFilter := r.URL.Query().Get("type")

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			rows, err := sqlDB.QueryContext(r.Context(), querySQL,
				cleanPath+"%", // ? - path LIKE pattern
				cleanPath,     // ? - empty path check
				targetDepth,   // ? - minimum depth (children only)
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var filePath string
				var size, duration sql.NullInt64
				var mediaType sql.NullString
				if err := rows.Scan(&filePath, &size, &duration, &mediaType); err != nil {
					return err
				}

				// Apply type filter
				if typeFilter != "" && mediaType.Valid {
					if !strings.HasPrefix(mediaType.String, typeFilter) {
						continue
					}
				}

				// Calculate the file's depth (number of path components)
				parts := strings.Split(filePath, string(filepath.Separator))
				isAbsolute := len(parts) > 0 && parts[0] == ""

				// Remove empty first element if path starts with /
				if isAbsolute {
					parts = parts[1:]
				}
				fileDepth := len(parts)

				if fileDepth == targetDepth {
					// This is a direct child file - add to files list
					media := &models.MediaWithDB{
						Media: models.Media{
							Path:     filePath,
							Size:     &size.Int64,
							Duration: &duration.Int64,
							Type:     &mediaType.String,
						},
						DB: dbPath,
					}
					directFiles[filePath] = media
				} else if fileDepth > targetDepth {
					// This file is in a subfolder - group by subfolder
					var parent string
					if currentDepth == 0 {
						// Root level: parent is first component
						if len(parts) > 0 {
							parent = parts[0]
							if isAbsolute {
								parent = string(filepath.Separator) + parent
							}
						}
					} else {
						// Subdirectory: parent is path up to targetDepth components
						if len(parts) >= targetDepth {
							parent = strings.Join(parts[:targetDepth], string(filepath.Separator))
							if isAbsolute {
								parent = string(filepath.Separator) + parent
							}
						} else {
							parent = filePath
						}
					}

					if parent == "" {
						continue
					}

					if _, ok := folders[parent]; !ok {
						folders[parent] = &models.FolderStats{Path: parent}
					}
					stat := folders[parent]
					stat.Count++
					if size.Valid {
						stat.TotalSize += size.Int64
					}
					if duration.Valid {
						stat.TotalDuration += duration.Int64
					}
				}
			}
			return rows.Err()
		})
		if err != nil {
			slog.Error("Failed to fetch DU stats", "db", dbPath, "error", err)
		}
	}

	// Combine folders and files into results
	var results []models.FolderStats

	// Add folders
	for _, stat := range folders {
		results = append(results, *stat)
	}

	// Add direct files as single-file "folders"
	for _, file := range directFiles {
		var fileSize, fileDuration int64
		if file.Size != nil {
			fileSize = *file.Size
		}
		if file.Duration != nil {
			fileDuration = *file.Duration
		}
		results = append(results, models.FolderStats{
			Path:          file.Path,
			Count:         0,
			TotalSize:     fileSize,
			TotalDuration: fileDuration,
			ExistsCount:   0,
			Files:         []models.MediaWithDB{*file},
		})
	}

	// Sort results
	sortBy := r.URL.Query().Get("sort")
	reverse := r.URL.Query().Get("reverse") == "true"

	if sortBy == "" {
		sortBy = "size"
		reverse = true
	}

	query.SortFolders(results, sortBy, reverse)

	// Set total count header for pagination
	w.Header().Set("X-Total-Count", strconv.Itoa(len(results)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (c *ServeCmd) handleEpisodes(w http.ResponseWriter, r *http.Request) {
	flags := c.parseFlags(r)
	if flags.Limit <= 0 {
		flags.All = true
		flags.Limit = 1000000
	}

	allMedia, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		slog.Error("Failed to fetch media for episodes", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	results := aggregate.GroupByParent(allMedia)

	// Set total count header for pagination
	w.Header().Set("X-Total-Count", strconv.Itoa(len(results)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// filterBinsData holds the raw collected data for building filter bins
type filterBinsData struct {
	sizes        []int64
	durations    []int64
	parentCounts map[string]int64
}

// computeFilterBinsData queries all databases and collects size, duration, and parent count data
// This is the single source of truth for filter bins data collection
func (c *ServeCmd) computeFilterBinsData(ctx context.Context, flags models.GlobalFlags, filterToIgnore string) filterBinsData {
	var mu sync.Mutex
	var allSizes []int64
	var allDurations []int64
	allParentCounts := make(map[string]int64)

	// Build flags for this query, ignoring the specified filter
	tempFlags := flags
	tempFlags.Where = append([]string{}, flags.Where...)
	tempFlags.All = true
	tempFlags.Limit = 0

	if filterToIgnore == "size" {
		tempFlags.Size = nil
	} else if filterToIgnore == "duration" {
		tempFlags.Duration = nil
	} else if filterToIgnore == "episodes" {
		tempFlags.FileCounts = ""
	}

	qb := query.NewQueryBuilder(tempFlags)
	sqlQuery, args := qb.BuildSelect("path, size, duration")

	var wg sync.WaitGroup
	for _, dbPath := range c.Databases {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			c.execDB(ctx, path, func(sqlDB *sql.DB) error {
				rows, err := sqlDB.QueryContext(ctx, sqlQuery, args...)
				if err != nil {
					return err
				}
				defer rows.Close()

				var localSizes []int64
				var localDurations []int64
				localParentCounts := make(map[string]int64)

				for rows.Next() {
					var p string
					var s, d sql.NullInt64
					if err := rows.Scan(&p, &s, &d); err == nil {
						// Validate size: between 1 byte and 100TB
						if s.Valid && s.Int64 > 0 && s.Int64 < 100*1024*1024*1024*1024 {
							localSizes = append(localSizes, s.Int64)
						}
						// Validate duration: between 1 second and 31 days
						if d.Valid && d.Int64 > 0 && d.Int64 < 2678400 {
							localDurations = append(localDurations, d.Int64)
						}
						parent := filepath.Dir(p)
						localParentCounts[parent]++
					}
				}

				mu.Lock()
				allSizes = append(allSizes, localSizes...)
				allDurations = append(allDurations, localDurations...)
				for k, v := range localParentCounts {
					allParentCounts[k] += v
				}
				mu.Unlock()
				return nil
			})
		}(dbPath)
	}
	wg.Wait()

	return filterBinsData{
		sizes:        allSizes,
		durations:    allDurations,
		parentCounts: allParentCounts,
	}
}

// buildSizeBins creates size filter bins from raw size data
func buildSizeBins(sizes []int64) (minVal, maxVal int64, bins []models.FilterBin, percentiles []int64) {
	if len(sizes) == 0 {
		return 0, 0, nil, nil
	}

	minVal = slices.Min(sizes)
	maxVal = slices.Max(sizes)
	percentiles = utils.CalculatePercentiles(sizes)

	p16 := int64(utils.Percentile(sizes, 16.6))
	p33 := int64(utils.Percentile(sizes, 33.3))
	p50 := int64(utils.Percentile(sizes, 50.0))
	p66 := int64(utils.Percentile(sizes, 66.6))
	p83 := int64(utils.Percentile(sizes, 83.3))

	sbins := []int64{0, p16, p33, p50, p66, p83, maxVal}
	for i := 0; i < len(sbins)-1; i++ {
		minS := sbins[i]
		maxS := sbins[i+1]
		if i == 0 {
			bins = append(bins, models.FilterBin{Label: "less than " + utils.FormatSize(maxS), Max: maxS})
		} else if i == len(sbins)-2 {
			bins = append(bins, models.FilterBin{Label: utils.FormatSize(minS) + "+", Min: minS})
		} else {
			bins = append(bins, models.FilterBin{Label: utils.FormatSize(minS) + " - " + utils.FormatSize(maxS), Min: minS, Max: maxS})
		}
	}

	return minVal, maxVal, bins, percentiles
}

// buildDurationBins creates duration filter bins from raw duration data
func buildDurationBins(durations []int64) (minVal, maxVal int64, bins []models.FilterBin, percentiles []int64) {
	if len(durations) == 0 {
		return 0, 0, nil, nil
	}

	minVal = slices.Min(durations)
	maxVal = slices.Max(durations)
	percentiles = utils.CalculatePercentiles(durations)

	p16 := int64(utils.Percentile(durations, 16.6))
	p33 := int64(utils.Percentile(durations, 33.3))
	p50 := int64(utils.Percentile(durations, 50.0))
	p66 := int64(utils.Percentile(durations, 66.6))
	p83 := int64(utils.Percentile(durations, 83.3))

	dbins := []int64{0, p16, p33, p50, p66, p83, maxVal}
	for i := 0; i < len(dbins)-1; i++ {
		minD := dbins[i]
		maxD := dbins[i+1]
		if i == 0 {
			bins = append(bins, models.FilterBin{Label: "under " + utils.FormatDuration(int(maxD)), Max: maxD})
		} else if i == len(dbins)-2 {
			bins = append(bins, models.FilterBin{Label: utils.FormatDuration(int(minD)) + "+", Min: minD})
		} else {
			bins = append(bins, models.FilterBin{Label: utils.FormatDuration(int(minD)) + " - " + utils.FormatDuration(int(maxD)), Min: minD, Max: maxD})
		}
	}

	return minVal, maxVal, bins, percentiles
}

// buildEpisodeBins creates episode count filter bins from parent counts
func buildEpisodeBins(parentCounts map[string]int64) (minVal, maxVal int64, bins []models.FilterBin, percentiles []int64) {
	if len(parentCounts) == 0 {
		return 0, 0, nil, nil
	}

	var allCounts []int64
	var countsGT1 []int64
	for _, count := range parentCounts {
		allCounts = append(allCounts, count)
		if count > 1 {
			countsGT1 = append(countsGT1, count)
		}
	}

	minVal = slices.Min(allCounts)
	maxVal = slices.Max(allCounts)
	percentiles = utils.CalculatePercentiles(allCounts)

	// Always include "Specials" bin for single files
	bins = append(bins, models.FilterBin{Label: "Specials", Value: 1})

	if len(countsGT1) > 0 {
		q1 := int64(utils.Percentile(countsGT1, 16.6))
		q2 := int64(utils.Percentile(countsGT1, 33.3))
		q3 := int64(utils.Percentile(countsGT1, 50.0))
		q4 := int64(utils.Percentile(countsGT1, 66.6))
		q5 := int64(utils.Percentile(countsGT1, 83.3))
		maxCount := int64(utils.Percentile(countsGT1, 100))

		rawBins := []int64{2, q1, q2, q3, q4, q5, maxCount}
		slices.Sort(rawBins)

		// Remove duplicates
		uniqueBins := []int64{rawBins[0]}
		for i := 1; i < len(rawBins); i++ {
			if rawBins[i] > uniqueBins[len(uniqueBins)-1] {
				uniqueBins = append(uniqueBins, rawBins[i])
			}
		}

		// Build range bins
		for i := 0; i < len(uniqueBins)-1; i++ {
			minE := uniqueBins[i]
			maxE := uniqueBins[i+1]
			displayMin := minE
			if i > 0 {
				displayMin = uniqueBins[i] + 1
			}
			if displayMin > maxE {
				continue
			}
			if displayMin == maxE {
				bins = append(bins, models.FilterBin{Label: fmt.Sprintf("%d", displayMin), Value: displayMin})
			} else {
				bins = append(bins, models.FilterBin{Label: fmt.Sprintf("%d-%d", displayMin, maxE), Min: displayMin, Max: maxE})
			}
		}

		// Add final "X+" bin if not already covered
		lastMax := uniqueBins[len(uniqueBins)-1]
		alreadyAdded := false
		if len(bins) > 0 {
			lastBin := bins[len(bins)-1]
			if lastBin.Max == lastMax || lastBin.Value == lastMax {
				alreadyAdded = true
			}
		}
		if !alreadyAdded {
			bins = append(bins, models.FilterBin{Label: fmt.Sprintf("%d+", lastMax), Min: lastMax})
		}
	}

	return minVal, maxVal, bins, percentiles
}

// handleFilterBins handles the /api/filter-bins endpoint
func (c *ServeCmd) handleFilterBins(w http.ResponseWriter, r *http.Request) {
	flags := c.parseFlags(r)
	q := r.URL.Query()

	// Set flags to get all data for bin calculation
	flags.All = true
	flags.Limit = 0

	resp := models.FilterBinsResponse{}

	// Collect data for each filter type, ignoring that filter to get full distribution
	episodesOnly := q.Has("episodes")
	sizeOnly := q.Has("size") || q.Has("min_size") || q.Has("max_size")
	durationOnly := q.Has("duration") || q.Has("min_duration") || q.Has("max_duration")

	// Get episode data
	epData := c.computeFilterBinsData(r.Context(), flags, "episodes")
	resp.EpisodesMin, resp.EpisodesMax, resp.Episodes, resp.EpisodesPercentiles = buildEpisodeBins(epData.parentCounts)

	// Get size data
	sizeData := c.computeFilterBinsData(r.Context(), flags, "size")
	resp.SizeMin, resp.SizeMax, resp.Size, resp.SizePercentiles = buildSizeBins(sizeData.sizes)

	// Get duration data
	durData := c.computeFilterBinsData(r.Context(), flags, "duration")
	resp.DurationMin, resp.DurationMax, resp.Duration, resp.DurationPercentiles = buildDurationBins(durData.durations)

	// Log query info for debugging
	slog.Info("FilterBins computed",
		"episodesOnly", episodesOnly,
		"sizeOnly", sizeOnly,
		"durationOnly", durationOnly,
		"sizeCount", len(sizeData.sizes),
		"durationCount", len(durData.durations),
		"parentCount", len(epData.parentCounts))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// calculateFilterCounts computes filter bin counts for the current query
// This is used with include_counts to provide filter UI data alongside query results
func (c *ServeCmd) calculateFilterCounts(ctx context.Context, flags models.GlobalFlags) *models.FilterBinsResponse {
	resp := &models.FilterBinsResponse{}

	// Collect data for each filter type
	data := c.computeFilterBinsData(ctx, flags, "")

	// Build all bin types from the same data source
	resp.SizeMin, resp.SizeMax, resp.Size, resp.SizePercentiles = buildSizeBins(data.sizes)
	resp.DurationMin, resp.DurationMax, resp.Duration, resp.DurationPercentiles = buildDurationBins(data.durations)
	resp.EpisodesMin, resp.EpisodesMax, resp.Episodes, resp.EpisodesPercentiles = buildEpisodeBins(data.parentCounts)

	return resp
}

func (c *ServeCmd) handleCategorizeKeywords(w http.ResponseWriter, r *http.Request) {
	type catKeywords struct {
		Category string   `json:"category"`
		Keywords []string `json:"keywords"`
	}

	data := make(map[string]map[string]bool)

	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			rows, err := sqlDB.QueryContext(r.Context(), "SELECT category, keyword FROM custom_keywords")
			if err != nil {
				return nil
			}
			defer rows.Close()
			for rows.Next() {
				var cat, kw string
				if err := rows.Scan(&cat, &kw); err == nil {
					if _, ok := data[cat]; !ok {
						data[cat] = make(map[string]bool)
					}
					data[cat][kw] = true
				}
			}
			return nil
		})
	}

	var results []catKeywords
	for cat, kwSet := range data {
		var kws []string
		for kw := range kwSet {
			kws = append(kws, kw)
		}
		sort.Strings(kws)
		results = append(results, catKeywords{
			Category: cat,
			Keywords: kws,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Category < results[j].Category
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (c *ServeCmd) handleCategorizeDefaults(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count := 0
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			tx, err := sqlDB.Begin()
			if err != nil {
				return err
			}
			stmt, err := tx.PrepareContext(r.Context(), "INSERT OR IGNORE INTO custom_keywords (category, keyword) VALUES (?, ?)")
			if err != nil {
				tx.Rollback()
				return err
			}
			defer stmt.Close()

			for cat, keywords := range models.DefaultCategories {
				for _, kw := range keywords {
					_, err := stmt.ExecContext(r.Context(), cat, kw)
					if err == nil {
						count++
					}
				}
			}
			return tx.Commit()
		})
		if err != nil {
			slog.Error("Failed to insert default categories", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (c *ServeCmd) handleCategorizeDeleteCategory(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := r.URL.Query().Get("category")
	if category == "" {
		http.Error(w, "Category required", http.StatusBadRequest)
		return
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			_, err := sqlDB.ExecContext(r.Context(), "DELETE FROM custom_keywords WHERE category = ?", category)
			return err
		})
		if err != nil {
			slog.Error("Failed to delete category", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (c *ServeCmd) handleCategorizeKeyword(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodDelete {
		var req struct {
			Category string `json:"category"`
			Keyword  string `json:"keyword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				_, err := sqlDB.ExecContext(r.Context(), "DELETE FROM custom_keywords WHERE category = ? AND keyword = ?", req.Category, req.Keyword)
				return err
			})
			if err != nil {
				slog.Error("Failed to delete keyword", "db", dbPath, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	var req struct {
		Category string `json:"category"`
		Keyword  string `json:"keyword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Category == "" || req.Keyword == "" {
		http.Error(w, "Category and Keyword are required", http.StatusBadRequest)
		return
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			_, err := sqlDB.ExecContext(r.Context(), "INSERT OR IGNORE INTO custom_keywords (category, keyword) VALUES (?, ?)", req.Category, req.Keyword)
			return err
		})
		if err != nil {
			slog.Error("Failed to save custom keyword", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (c *ServeCmd) handleRandomClip(w http.ResponseWriter, r *http.Request) {
	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMedia(r.Context(), 1000000)
			if err != nil {
				return err
			}
			for _, m := range dbMedia {
				allMedia = append(allMedia, models.MediaWithDB{
					Media: models.FromDB(m),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch media for random clip", "error", err)
		}
	}

	if len(allMedia) == 0 {
		http.Error(w, "No media found", http.StatusNotFound)
		return
	}

	// Filter for video/audio only
	var playable []models.MediaWithDB
	targetType := r.URL.Query().Get("type")

	for _, m := range allMedia {
		if m.Type == nil {
			continue
		}

		if targetType != "" {
			if strings.HasPrefix(*m.Type, targetType) {
				playable = append(playable, m)
			}
		} else {
			// Default behavior: video or audio
			if strings.HasPrefix(*m.Type, "video") || strings.HasPrefix(*m.Type, "audio") || *m.Type == "audiobook" {
				playable = append(playable, m)
			}
		}
	}

	if len(playable) == 0 {
		http.Error(w, "No playable media found", http.StatusNotFound)
		return
	}

	item := playable[utils.RandomInt(0, len(playable)-1)]

	duration := 0
	if item.Duration != nil {
		duration = int(*item.Duration)
	}

	cableDuration := 15 // Default 15s clips
	if q := r.URL.Query().Get("duration"); q != "" {
		if d, err := strconv.Atoi(q); err == nil {
			cableDuration = d
		}
	}

	// If 0, play the whole thing (start at 0, end at duration)
	start := 0
	end := duration

	if cableDuration > 0 {
		if duration > cableDuration {
			start = utils.RandomInt(0, duration-cableDuration)
		}
		end = start + cableDuration
	}

	// Support fields parameter to limit response size
	fieldsParam := r.URL.Query().Get("fields")

	type clipResponse struct {
		models.MediaWithDB
		Start int `json:"start"`
		End   int `json:"end"`
	}

	response := clipResponse{
		MediaWithDB: item,
		Start:       start,
		End:         end,
	}

	// If fields parameter is provided, only include specified fields
	if fieldsParam != "" {
		requestedFields := strings.Split(fieldsParam, ",")
		fieldSet := make(map[string]bool)
		for _, f := range requestedFields {
			fieldSet[strings.TrimSpace(f)] = true
		}

		// Clear all fields first
		cleared := models.MediaWithDB{
			DB: item.DB,
		}

		// Only include requested fields
		if fieldSet["path"] {
			cleared.Path = item.Path
		}
		if fieldSet["type"] {
			cleared.Type = item.Type
		}
		if fieldSet["duration"] {
			cleared.Duration = item.Duration
		}
		if fieldSet["start"] || fieldSet["end"] {
			// Always include start/end if either is requested since they're part of the response wrapper
			response.Start = start
			response.End = end
		}
		if fieldSet["db"] {
			cleared.DB = item.DB
		}

		response.MediaWithDB = cleared
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *ServeCmd) handleCategorizeSuggest(w http.ResponseWriter, r *http.Request) {
	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMedia(r.Context(), 1000000)
			if err != nil {
				return err
			}
			for _, m := range dbMedia {
				allMedia = append(allMedia, models.MediaWithDB{
					Media: models.FromDB(m),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch media for categorize suggest", "error", err)
		}
	}

	// Fetch existing keywords to filter them out
	existingKeywords := make(map[string]bool)
	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			rows, err := sqlDB.QueryContext(r.Context(), "SELECT DISTINCT keyword FROM custom_keywords")
			if err != nil {
				return nil
			}
			defer rows.Close()
			for rows.Next() {
				var kw string
				if err := rows.Scan(&kw); err == nil {
					existingKeywords[strings.ToLower(kw)] = true
				}
			}
			return nil
		})
	}

	cmd := CategorizeCmd{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		PostActionFlags:  c.PostActionFlags,
		Databases:        c.Databases,
	}
	// Note: mineCategories and applyCategories need to be exported or called through a wrapper
	// Since I'm in the same package 'commands', I can call them directly.

	// We need to compile regexes first
	compiled := cmd.CompileRegexes()

	wordCounts := make(map[string]int)
	for _, m := range allMedia {
		// Skip files that already have categories assigned
		if m.Categories != nil && *m.Categories != "" {
			continue
		}

		matched := false
		pathAndTitle := m.Path
		if m.Title != nil {
			pathAndTitle += " " + *m.Title
		}

		for _, res := range compiled {
			for _, re := range res {
				if re.MatchString(pathAndTitle) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if !matched {
			// Use a map to count each word only once per file
			uniqueWords := make(map[string]bool)

			words := utils.ExtractWords(utils.PathToSentence(m.Path))
			if m.Title != nil {
				words = append(words, utils.ExtractWords(*m.Title)...)
			}

			for _, word := range words {
				if len(word) < 4 {
					continue
				}
				// Filter out already-assigned keywords
				if existingKeywords[strings.ToLower(word)] {
					continue
				}
				// Only count each word once per file
				if !uniqueWords[word] {
					uniqueWords[word] = true
					wordCounts[word]++
				}
			}
		}
	}

	type wordFreq struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}
	var freqs []wordFreq
	for w, c := range wordCounts {
		if c > 1 {
			freqs = append(freqs, wordFreq{Word: w, Count: c})
		}
	}

	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].Count > freqs[j].Count
	})

	limit := min(len(freqs), 100)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(freqs[:limit])
}

func (c *ServeCmd) handleCategorizeApply(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMedia(r.Context(), 1000000)
			if err != nil {
				return err
			}
			for _, m := range dbMedia {
				allMedia = append(allMedia, models.MediaWithDB{
					Media: models.FromDB(m),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch media for categorize apply", "error", err)
		}
	}

	cmd := CategorizeCmd{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		PostActionFlags:  c.PostActionFlags,
		Databases:        c.Databases,
	}
	compiled := cmd.CompileRegexes()

	if len(compiled) == 0 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"count": 0}`)
		return
	}

	count := 0
	for _, m := range allMedia {
		foundCategories := []string{}
		pathAndTitle := m.Path
		if m.Title != nil {
			pathAndTitle += " " + *m.Title
		}

		for cat, res := range compiled {
			for _, re := range res {
				if re.MatchString(pathAndTitle) {
					foundCategories = append(foundCategories, cat)
					break
				}
			}
		}

		if len(foundCategories) > 0 {
			merged := make(map[string]bool)
			if m.Categories != nil && *m.Categories != "" {
				existing := strings.SplitSeq(strings.Trim(*m.Categories, ";"), ";")
				for e := range existing {
					if e != "" {
						merged[strings.TrimSpace(e)] = true
					}
				}
			}
			for _, f := range foundCategories {
				merged[f] = true
			}
			combined := []string{}
			for k := range merged {
				combined = append(combined, k)
			}
			sort.Strings(combined)
			newCategories := ";" + strings.Join(combined, ";") + ";"

			err := c.execDB(r.Context(), m.DB, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				return queries.UpdateMediaCategories(r.Context(), database.UpdateMediaCategoriesParams{
					Categories: utils.ToNullString(newCategories),
					Path:       m.Path,
				})
			})
			if err == nil {
				count++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"count": %d}`, count)
}

func (c *ServeCmd) handleRaw(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	if c.isPathBlacklisted(path) {
		slog.Warn("Access denied: path is blacklisted", "path", path)
		http.Error(w, "Access denied: sensitive path", http.StatusForbidden)
		return
	}

	slog.Debug("handleRaw request", "path", path)

	var m models.Media
	found := false

	localPath := path

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				m = models.FromDB(dbMedia)
				found = true
			}
			return err
		})
		if found {
			break
		}
		if err != nil && err != sql.ErrNoRows {
			slog.Error("Database error in handleRaw", "db", dbPath, "error", err)
		}
	}

	if !found {
		slog.Warn("Access denied: file not in database", "path", path)
		http.Error(w, "Access denied: file not in database", http.StatusForbidden)
		return
	}

	isLocal := utils.FileExists(localPath)
	if !isLocal {
		slog.Warn("File not found on disk, marking as deleted in databases", "path", path)
		c.markDeletedInAllDBs(r.Context(), path, true)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	strategy := utils.GetTranscodeStrategy(m)
	slog.Debug("handleRaw strategy", "path", path, "needs_transcode", strategy.NeedsTranscode, "vcopy", strategy.VideoCopy, "acopy", strategy.AudioCopy)

	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(localPath)))

	if strategy.NeedsTranscode {
		if c.hasFfmpeg {
			c.handleTranscode(w, r, localPath, m, strategy)
			return
		} else {
			slog.Error("ffmpeg not found in PATH, skipping transcoding", "path", path)
		}
	}

	slog.Debug("Serving local file", "path", localPath)
	http.ServeFile(w, r, localPath)
}

func (c *ServeCmd) handleTranscode(w http.ResponseWriter, r *http.Request, path string, m models.Media, strategy utils.TranscodeStrategy) {
	w.Header().Set("Content-Type", strategy.TargetMime)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(path)))

	start := r.URL.Query().Get("start")

	// Add flags to help with piped streaming duration and timestamp issues
	var args []string

	if start != "" {
		args = append(args, "-ss", start)
	}

	args = append(args, "-fflags", "+genpts", "-i", path)

	// If we have duration in metadata, tell ffmpeg so it can write it to headers
	if m.Duration != nil && *m.Duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", *m.Duration))
	}

	if strategy.VideoCopy {
		args = append(args, "-c:v", "copy")
	} else {
		if strategy.TargetMime == "video/mp4" {
			args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-crf", "28")
		} else {
			// WebM
			args = append(args, "-c:v", "libvpx-vp9", "-deadline", "realtime", "-cpu-used", "8", "-crf", "30", "-b:v", "0")
		}
	}

	if strategy.AudioCopy {
		args = append(args, "-c:a", "copy")
	} else {
		if strategy.TargetMime == "video/mp4" {
			args = append(args, "-c:a", "aac", "-b:a", "128k", "-ac", "2")
		} else {
			// WebM supports Opus
			args = append(args, "-c:a", "libopus", "-b:a", "128k", "-ac", "2")
		}
	}

	args = append(args, "-avoid_negative_ts", "make_zero", "-map_metadata", "-1", "-sn")

	if strategy.TargetMime == "video/mp4" {
		// frag_keyframe+empty_moov+default_base_moof+global_sidx is the standard for fragmented streaming
		args = append(args, "-f", "mp4", "-movflags", "frag_keyframe+empty_moov+default_base_moof+global_sidx", "pipe:1")
	} else {
		// Matroska with index space reserved and cluster limits can help browsers determine duration
		args = append(args, "-f", "matroska", "-live", "1", "-reserve_index_space", "1024k", "-cluster_size_limit", "2M", "-cluster_time_limit", "5100", "pipe:1")
	}

	ffmpegArgs := append([]string{"-hide_banner", "-loglevel", "error"}, args...)
	slog.Info("Streaming with transcode", "path", path, "strategy", strategy, "args", strings.Join(ffmpegArgs, " "))

	cmd := exec.CommandContext(r.Context(), "ffmpeg", ffmpegArgs...)
	cmd.Stdout = w

	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start ffmpeg", "path", path, "error", err)
		http.Error(w, "Unplayable: transcoding failed", http.StatusUnsupportedMediaType)
		return
	}

	if err := cmd.Wait(); err != nil {
		if r.Context().Err() == nil {
			slog.Error("ffmpeg failed", "path", path, "error", err)
		} else {
			slog.Debug("ffmpeg finished (client disconnected)", "path", path)
		}
	}
}

func (c *ServeCmd) handleSubtitles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	slog.Debug("handleSubtitles request", "path", path, "index", r.URL.Query().Get("index"))

	// Verify path or siblings
	found := false
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			_, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				found = true
				return nil
			}

			// If path doesn't exist, it might be an external subtitle file next to a media file
			// We'll check if any media in the database shares the same directory and base name
			dir := filepath.Dir(path)
			filename := filepath.Base(path)
			base := strings.TrimSuffix(filename, filepath.Ext(filename))
			// Handle cases like movie.en.srt by stripping one more extension if it exists
			if secondExt := filepath.Ext(base); secondExt != "" {
				base = strings.TrimSuffix(base, secondExt)
			}

			// Simple check: does this directory contain ANY media we know with the same base name?
			mediaInDir, _ := queries.GetMedia(r.Context(), 1000)
			for _, m := range mediaInDir {
				if filepath.Dir(m.Path) == dir {
					mBase := strings.TrimSuffix(filepath.Base(m.Path), filepath.Ext(m.Path))
					if mBase == base {
						found = true
						break
					}
				}
			}
			return nil
		})
		if found {
			break
		}
		if err != nil && err != sql.ErrNoRows {
			slog.Error("Database error in handleSubtitles", "db", dbPath, "error", err)
		}
	}

	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if !utils.FileExists(path) {
		slog.Warn("File not found on disk, marking as deleted in databases", "path", path)
		c.markDeletedInAllDBs(r.Context(), path, true)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	streamIndex := r.URL.Query().Get("index")

	// If it's a media container but no index is specified, we should try to find an external sidecar
	if streamIndex == "" && (ext == ".mkv" || ext == ".mp4" || ext == ".m4v" || ext == ".mov" || ext == ".webm") {
		// Try to find a sibling subtitle file
		sidecars := utils.GetExternalSubtitles(path)
		if len(sidecars) > 0 {
			// Serve the first found sidecar
			path = sidecars[0]
			ext = strings.ToLower(filepath.Ext(path))
			slog.Debug("Found sidecar for media file", "media", r.URL.Query().Get("path"), "sidecar", path)
		} else {
			http.Error(w, "No index specified and no sidecar found", http.StatusNotFound)
			return
		}
	}

	if ext == ".idx" {
		subPath := strings.TrimSuffix(path, ".idx") + ".sub"
		if !utils.FileExists(subPath) {
			slog.Warn("VobSub conversion requested but .sub file is missing", "idx", path)
			http.Error(w, "Corresponding .sub file not found", http.StatusNotFound)
			return
		}
	}

	if ext == ".vtt" {
		w.Header().Set("Content-Type", "text/vtt")
		http.ServeFile(w, r, path)
		return
	}

	var args []string
	isImageSub := ext == ".idx" || ext == ".sub" || ext == ".sup"

	if streamIndex != "" {
		// Embedded tracks
		args = append(args, "-i", path, "-map", "0:s:"+streamIndex, "-f", "webvtt", "pipe:1")
	} else {
		// Standalone file (srt, lrc, ass, etc.)
		args = append(args, "-i", path, "-f", "webvtt", "pipe:1")
	}

	ffmpegArgs := append([]string{"-hide_banner", "-loglevel", "error"}, args...)
	slog.Debug("subtitle ffmpeg command", "args", strings.Join(ffmpegArgs, " "))

	cmd := exec.CommandContext(r.Context(), "ffmpeg", ffmpegArgs...)

	// We don't set Content-Type yet to allow http.Error if ffmpeg fails immediately
	output, err := cmd.CombinedOutput()
	if err != nil {
		if r.Context().Err() == nil {
			msg := "Failed to convert subtitles"
			if isImageSub || streamIndex != "" {
				msg = "Failed to convert subtitles (image-based formats require OCR which is not yet supported for direct VTT streaming)"
			}
			slog.Error(msg, "path", path, "error", err, "output", string(output))
			http.Error(w, "Unplayable: subtitle conversion failed", http.StatusUnsupportedMediaType)
		}
		return
	}

	w.Header().Set("Content-Type", "text/vtt")
	w.Write(output)
}

func (c *ServeCmd) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Verify path exists in database to prevent arbitrary file access
	found := false
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			_, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				found = true
			}
			return err
		})
		if found {
			break
		}
		if err != nil && err != sql.ErrNoRows {
			slog.Error("Database error in handleThumbnail", "db", dbPath, "error", err)
		}
	}

	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check cache
	if data, ok := c.thumbnailCache.Load(path); ok {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Write(data.([]byte))
		return
	}

	// Generate thumbnail
	mime := utils.DetectMimeType(path)

	if strings.HasPrefix(mime, "image/") {
		if info, err := os.Stat(path); err == nil && info.Size() < 500*1024 {
			data, err := os.ReadFile(path)
			if err == nil {
				w.Header().Set("Content-Type", mime)
				w.Header().Set("Cache-Control", "public, max-age=31536000")
				w.Write(data)
				return
			}
		}
	}

	var args []string
	if strings.HasPrefix(mime, "video/") {
		args = []string{"-ss", "25", "-i", path, "-frames:v", "1", "-q:v", "4", "-vf", "scale=320:-1", "-f", "image2", "pipe:1"}
	} else if strings.HasPrefix(mime, "image/") {
		args = []string{"-i", path, "-vf", "scale=320:-1", "-f", "image2", "pipe:1"}
	} else if strings.HasPrefix(mime, "audio/") {
		args = []string{"-i", path, "-an", "-vcodec", "copy", "-f", "image2", "pipe:1"}
	} else {
		http.Error(w, "Unsupported type", http.StatusUnsupportedMediaType)
		return
	}

	cmd := exec.CommandContext(r.Context(), "ffmpeg", append([]string{"-hide_banner", "-loglevel", "error"}, args...)...)
	thumb, err := cmd.Output()
	if err != nil {
		// slog.Debug("Thumbnail generation failed", "path", path, "error", err)
		http.Error(w, "Failed to generate thumbnail", http.StatusInternalServerError)
		return
	}

	// Cache it
	c.thumbnailCache.Store(path, thumb)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Write(thumb)
}

func (c *ServeCmd) handleTrash(w http.ResponseWriter, r *http.Request) {
	flags := c.GetGlobalFlags()
	flags.OnlyDeleted = true
	flags.HideDeleted = false
	flags.All = true
	flags.SortBy = "time_deleted"
	flags.Reverse = true

	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		slog.Error("Trash query failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(len(media)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

func (c *ServeCmd) handleEmptyBin(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var media []models.MediaWithDB
	if len(req.Paths) > 0 {
		// Only delete the requested paths
		for _, p := range req.Paths {
			media = append(media, models.MediaWithDB{Media: models.Media{Path: p}})
		}
	} else {
		// Fallback: Delete everything in trash if no paths provided
		flags := c.GetGlobalFlags()
		flags.OnlyDeleted = true
		flags.HideDeleted = false
		flags.All = true

		var err error
		media, err = query.MediaQuery(context.Background(), c.Databases, flags)
		if err != nil {
			slog.Error("Trash query failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	count := 0
	for _, m := range media {
		if utils.FileExists(m.Path) {
			if err := os.Remove(m.Path); err != nil {
				slog.Error("Failed to delete file", "path", m.Path, "error", err)
				continue
			}
		}

		// Remove from DB
		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				result, err := sqlDB.Exec("DELETE FROM media WHERE path = ?", m.Path)
				if err != nil {
					return err
				}
				rows, _ := result.RowsAffected()
				if rows > 0 {
					count++
				}
				return nil
			})
			if err != nil {
				slog.Error("Failed to delete from DB", "db", dbPath, "error", err)
			}
		}
	}

	slog.Info("Bin emptied", "files_removed", count)
	fmt.Fprintf(w, "Deleted %d files", count)
}

func (c *ServeCmd) handleOPDS(w http.ResponseWriter, r *http.Request) {
	flags := c.GetGlobalFlags()
	flags.TextOnly = true
	flags.All = true

	media, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		slog.Error("OPDS query failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml;charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opds="http://opds-spec.org/2010/catalog">
  <id>discotheque-text</id>
  <title>Discotheque Text</title>
  <updated>`+time.Now().Format(time.RFC3339)+`</updated>
  <author><name>Discotheque</name></author>
`)

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	for _, m := range media {
		title := m.Stem()
		if m.Title != nil && *m.Title != "" {
			title = *m.Title
		}

		author := "Unknown"
		if m.Artist != nil && *m.Artist != "" {
			author = *m.Artist
		}

		mime := "application/octet-stream"
		if m.Type != nil {
			mime = *m.Type
		}

		fmt.Fprintf(w, `
  <entry>
    <title>%s</title>
    <id>%s</id>
    <updated>%s</updated>
    <author><name>%s</name></author>
    <content type="text">%s</content>
    <link rel="http://opds-spec.org/acquisition" href="%s://%s/api/raw?path=%s" type="%s"/>
  </entry>`,
			utils.EscapeXML(title),
			utils.EscapeXML(m.Path),
			time.Now().Format(time.RFC3339), // Ideally use modification time
			utils.EscapeXML(author),
			utils.EscapeXML(m.Path),
			scheme, host, strings.ReplaceAll(url.QueryEscape(m.Path), "+", "%20"),
			mime,
		)
	}

	fmt.Fprint(w, "\n</feed>")
}

func (c *ServeCmd) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		titles := make(map[string]bool)
		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				pls, err := queries.GetPlaylists(r.Context())
				if err != nil {
					return err
				}
				for _, p := range pls {
					if p.Title.Valid {
						titles[p.Title.String] = true
					}
				}
				return nil
			})
			if err != nil {
				slog.Error("Failed to fetch playlists", "db", dbPath, "error", err)
			}
		}

		uniqueTitles := make(models.PlaylistResponse, 0, len(titles))
		for t := range titles {
			uniqueTitles = append(uniqueTitles, t)
		}
		sort.Strings(uniqueTitles)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uniqueTitles)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			http.Error(w, "Title required", http.StatusBadRequest)
			return
		}

		playlistPath := "custom:" + utils.RandomString(12)

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				_, err := queries.InsertPlaylist(r.Context(), database.InsertPlaylistParams{
					Title: sql.NullString{String: req.Title, Valid: true},
					Path:  sql.NullString{String: playlistPath, Valid: true},
				})
				return err
			})
			if err != nil {
				slog.Error("Failed to insert playlist", "db", dbPath, "title", req.Title, "error", err)
			}
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

	if r.Method == http.MethodDelete {
		title := r.URL.Query().Get("title")
		if title == "" {
			http.Error(w, "Title required", http.StatusBadRequest)
			return
		}

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				// We need to find the ID by title first because DeletePlaylist takes ID
				pls, err := queries.GetPlaylists(r.Context())
				if err != nil {
					return err
				}
				for _, p := range pls {
					if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
						err = queries.DeletePlaylist(r.Context(), database.DeletePlaylistParams{
							ID:          p.ID,
							TimeDeleted: sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
						})
						if err != nil {
							return err
						}
					}
				}
				return nil
			})
			if err != nil {
				slog.Error("Failed to delete playlist", "db", dbPath, "title", title, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (c *ServeCmd) handlePlaylistItems(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		title := r.URL.Query().Get("title")
		if title == "" {
			http.Error(w, "Title required", http.StatusBadRequest)
			return
		}

		allMedia := make([]models.MediaWithDB, 0)
		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				pls, err := queries.GetPlaylists(r.Context())
				if err != nil {
					return err
				}

				var playlistID int64 = -1
				for _, p := range pls {
					if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
						playlistID = p.ID
						break
					}
				}

				if playlistID == -1 {
					return nil
				}

				items, err := queries.GetPlaylistItems(r.Context(), playlistID)
				if err != nil {
					return err
				}

				for _, item := range items {
					m := models.FromDB(database.Media{
						Path:            item.Path,
						Title:           item.Title,
						Duration:        item.Duration,
						Size:            item.Size,
						TimeCreated:     item.TimeCreated,
						TimeModified:    item.TimeModified,
						TimeDeleted:     item.TimeDeleted,
						TimeFirstPlayed: item.TimeFirstPlayed,
						TimeLastPlayed:  item.TimeLastPlayed,
						PlayCount:       item.PlayCount,
						Playhead:        item.Playhead,
						Type:            item.Type,
						Width:           item.Width,
						Height:          item.Height,
						Fps:             item.Fps,
						VideoCodecs:     item.VideoCodecs,
						AudioCodecs:     item.AudioCodecs,
						SubtitleCodecs:  item.SubtitleCodecs,
						VideoCount:      item.VideoCount,
						AudioCount:      item.AudioCount,
						SubtitleCount:   item.SubtitleCount,
						Album:           item.Album,
						Artist:          item.Artist,
						Genre:           item.Genre,
						Mood:            item.Mood,
						Bpm:             item.Bpm,
						Key:             item.Key,
						Decade:          item.Decade,
						Categories:      item.Categories,
						City:            item.City,
						Country:         item.Country,
						Description:     item.Description,
						Language:        item.Language,
						Webpath:         item.Webpath,
						Uploader:        item.Uploader,
						TimeUploaded:    item.TimeUploaded,
						TimeDownloaded:  item.TimeDownloaded,
						ViewCount:       item.ViewCount,
						NumComments:     item.NumComments,
						FavoriteCount:   item.FavoriteCount,
						Score:           item.Score,
						UpvoteRatio:     item.UpvoteRatio,
						Latitude:        item.Latitude,
						Longitude:       item.Longitude,
					})
					m.TrackNumber = models.NullInt64Ptr(item.TrackNumber)
					mw := models.MediaWithDB{
						Media: m,
						DB:    dbPath,
					}
					if c.hasFfmpeg {
						mw.Transcode = utils.GetTranscodeStrategy(m).NeedsTranscode
					}
					allMedia = append(allMedia, mw)
				}
				return nil
			})
			if err != nil {
				slog.Error("Failed to fetch playlist items", "db", dbPath, "title", title, "error", err)
			}
		}

		// Sort to match reordering logic: TrackNumber, then Path
		sort.Slice(allMedia, func(i, j int) bool {
			tnA := int64(0)
			if allMedia[i].Media.TrackNumber != nil {
				tnA = *allMedia[i].Media.TrackNumber
			}
			tnB := int64(0)
			if allMedia[j].Media.TrackNumber != nil {
				tnB = *allMedia[j].Media.TrackNumber
			}

			if tnA != tnB {
				return tnA < tnB
			}
			return allMedia[i].Media.Path < allMedia[j].Media.Path
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allMedia)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			PlaylistTitle string `json:"playlist_title"`
			MediaPath     string `json:"media_path"`
			TrackNumber   int64  `json:"track_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 1. Find which DB the media belongs to
		var mediaDB string
		found := false
		for _, dbPath := range c.Databases {
			_ = c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				_, err := queries.GetMediaByPathExact(r.Context(), req.MediaPath)
				if err == nil {
					mediaDB = dbPath
					found = true
				}
				return err
			})
			if found {
				break
			}
		}

		if !found {
			http.Error(w, "Media not found", http.StatusNotFound)
			return
		}

		// 2. Ensure playlist exists in that DB and add item
		err := c.execDB(r.Context(), mediaDB, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			pls, err := queries.GetPlaylists(r.Context())
			if err != nil {
				return err
			}

			var playlistID int64 = -1
			for _, p := range pls {
				if p.Title.Valid && strings.EqualFold(p.Title.String, req.PlaylistTitle) {
					playlistID = p.ID
					break
				}
			}

			if playlistID == -1 {
				// Create it if missing in this DB
				playlistPath := "custom:" + utils.RandomString(12)
				playlistID, err = queries.InsertPlaylist(r.Context(), database.InsertPlaylistParams{
					Title: sql.NullString{String: req.PlaylistTitle, Valid: true},
					Path:  sql.NullString{String: playlistPath, Valid: true},
				})
				if err != nil {
					return err
				}
			}

			return queries.AddPlaylistItem(r.Context(), database.AddPlaylistItemParams{
				PlaylistID:  playlistID,
				MediaPath:   req.MediaPath,
				TrackNumber: sql.NullInt64{Int64: req.TrackNumber, Valid: req.TrackNumber != 0},
			})
		})
		if err != nil {
			slog.Error("Failed to add playlist item", "title", req.PlaylistTitle, "path", req.MediaPath, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodDelete {
		var req struct {
			PlaylistTitle string `json:"playlist_title"`
			MediaPath     string `json:"media_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				pls, err := queries.GetPlaylists(r.Context())
				if err != nil {
					return err
				}

				var playlistID int64 = -1
				for _, p := range pls {
					if p.Title.Valid && strings.EqualFold(p.Title.String, req.PlaylistTitle) {
						playlistID = p.ID
						break
					}
				}

				if playlistID == -1 {
					return nil
				}

				return queries.RemovePlaylistItem(r.Context(), database.RemovePlaylistItemParams{
					PlaylistID: playlistID,
					MediaPath:  req.MediaPath,
				})
			})
			if err != nil {
				slog.Error("Failed to remove playlist item", "db", dbPath, "title", req.PlaylistTitle, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (c *ServeCmd) handleGenres(w http.ResponseWriter, r *http.Request) {
	counts := make(map[string]int64)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			stats, err := queries.GetGenreStats(r.Context())
			if err != nil {
				return err
			}
			for _, s := range stats {
				if s.Genre.Valid {
					counts[s.Genre.String] = counts[s.Genre.String] + s.Count
				}
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch genres", "db", dbPath, "error", err)
		}
	}

	type genreStat struct {
		Genre string `json:"genre"`
		Count int64  `json:"count"`
	}
	res := make([]genreStat, 0)
	for k, v := range counts {
		if v > 0 {
			res = append(res, genreStat{Genre: k, Count: v})
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Count > res[j].Count
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	json.NewEncoder(w).Encode(res)
}

const HLS_SEGMENT_DURATION = 6

func (c *ServeCmd) handleHLSPlaylist(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Fetch media to get duration
	var m models.Media
	found := false
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				m = models.FromDB(dbMedia)
				found = true
			}
			return err
		})
		if found {
			break
		}
		if err != nil && err != sql.ErrNoRows {
			slog.Error("Database error in handleHLSPlaylist", "db", dbPath, "error", err)
		}
	}

	if !found || m.Duration == nil {
		http.Error(w, "Media not found or no duration", http.StatusNotFound)
		return
	}

	duration := float64(*m.Duration)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")

	playlist := utils.GenerateHLSPlaylist(path, duration, HLS_SEGMENT_DURATION)
	fmt.Fprint(w, playlist)
}

func (c *ServeCmd) handleHLSSegment(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	indexStr := r.URL.Query().Get("index")
	if path == "" || indexStr == "" {
		http.Error(w, "Path and index required", http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "Invalid index", http.StatusBadRequest)
		return
	}

	if !utils.FileExists(path) {
		slog.Warn("File not found on disk, marking as deleted in databases", "path", path)
		c.markDeletedInAllDBs(r.Context(), path, true)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	startTime := float64(index * HLS_SEGMENT_DURATION)

	// Check if we have ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		http.Error(w, "ffmpeg not found", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "video/MP2T")

	// Fetch media to get codec info
	var m models.Media
	found := false
	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				m = models.FromDB(dbMedia)
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	strategy := utils.GetTranscodeStrategy(m)
	slog.Debug("HLS Segment request", "index", index, "start", startTime, "strategy", strategy, "path", path)

	args := utils.GetHLSSegmentArgs(path, startTime, HLS_SEGMENT_DURATION, strategy)

	// Skip logging for segments to avoid spam
	// slog.Debug("HLS Segment", "index", index, "start", startTime)

	cmd := exec.CommandContext(r.Context(), "ffmpeg", append([]string{"-hide_banner", "-loglevel", "error"}, args...)...)
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		if r.Context().Err() != nil {
			slog.Debug("Client disconnected during HLS transcoding", "path", path, "index", index)
		} else {
			slog.Error("HLS transcoding failed", "path", path, "index", index, "error", err)
		}
	}
}

func (c *ServeCmd) handleRSVP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Verify path in database
	found := false
	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			_, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	wpm := 220
	if wpmStr := r.URL.Query().Get("wpm"); wpmStr != "" {
		if val, err := strconv.Atoi(wpmStr); err == nil && val > 0 {
			wpm = val
		}
	}

	text, err := utils.ExtractText(path)
	if err != nil {
		slog.Error("Text extraction failed", "path", path, "error", err)
		http.Error(w, fmt.Sprintf("Text extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	assContent, duration := utils.GenerateRSVPAss(text, wpm)
	if duration <= 0 {
		http.Error(w, "Empty text content", http.StatusBadRequest)
		return
	}

	// Create temp directory for ASS and WAV
	tmpDir, err := os.MkdirTemp("", "disco-rsvp-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	assPath := filepath.Join(tmpDir, "subtitles.ass")
	if err := os.WriteFile(assPath, []byte(assContent), 0o644); err != nil {
		http.Error(w, "Failed to write subtitles", http.StatusInternalServerError)
		return
	}

	wavPath := filepath.Join(tmpDir, "audio.wav")
	if err := utils.GenerateTTS(text, wavPath, wpm); err != nil {
		slog.Warn("TTS generation failed", "error", err)
		wavPath = ""
	}

	w.Header().Set("Content-Type", "video/webm")
	w.Header().Set("Accept-Ranges", "bytes")

	// FFmpeg: black background + audio + RSVP subtitles
	var args []string
	args = append(args, "-hide_banner", "-loglevel", "error")
	args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("color=c=black:s=1280x720:d=%f", duration))

	if wavPath != "" {
		args = append(args, "-i", wavPath)
	}

	// Escape path for ffmpeg filter (simple Linux paths should be fine, but let's be safe)
	escapedAssPath := strings.ReplaceAll(assPath, "\\", "/")
	escapedAssPath = strings.ReplaceAll(escapedAssPath, ":", "\\:")

	args = append(args,
		"-vf", fmt.Sprintf("ass='%s'", escapedAssPath),
		"-c:v", "libvpx-vp9",
		"-deadline", "realtime",
		"-cpu-used", "8",
		"-crf", "30",
		"-b:v", "0",
	)

	if wavPath != "" {
		args = append(args, "-c:a", "libopus", "-b:a", "64k")
	}

	args = append(args, "-f", "webm", "pipe:1")

	slog.Info("Starting RSVP stream", "path", path, "wpm", wpm, "duration", duration)

	cmd := exec.CommandContext(r.Context(), "ffmpeg", args...)
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		if r.Context().Err() == nil {
			slog.Error("FFmpeg RSVP streaming failed", "error", err)
		}
	}
}
