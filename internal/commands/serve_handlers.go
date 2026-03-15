package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// handleHealth returns OK if the server is running
func (c *ServeCmd) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleQuery handles media searching and filtering.
// GET /api/query?search=...&category=...&rating=...&sort=...&limit=...&offset=...
func (c *ServeCmd) handleQuery(w http.ResponseWriter, r *http.Request) {
	flags := c.parseFlags(r)
	q := r.URL.Query()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Validate and filter databases
	dbs, err := c.getDatabasesForQuery(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}

	// Pre-resolve percentiles so Count matches Query results
	resolvedFlags, err := query.ResolvePercentileFlags(ctx, dbs, flags)
	if err == nil {
		flags = resolvedFlags
	}

	if q.Get("view") == "captions" || q.Get("captions") == "true" {
		media := []models.MediaWithDB{}
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

		for _, dbPath := range dbs {
			err := c.execDB(ctx, dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				var rows []database.SearchCaptionsRow
				var err error

				if queryStr != "" {
					// Search with context - get all captions for matched media
					rows, err = c.getCaptionsWithContext(ctx, queries, queryStr, int64(limit), flags.VideoOnly, flags.AudioOnly, flags.ImageOnly, flags.TextOnly)
				} else {
					// No search - return captions ordered by path and time for captions view
					// Apply media type filters
					params := database.GetAllCaptionsOrderedParams{
						VideoOnly: utils.BoolToInt64(flags.VideoOnly),
						AudioOnly: utils.BoolToInt64(flags.AudioOnly),
						ImageOnly: utils.BoolToInt64(flags.ImageOnly),
						TextOnly:  utils.BoolToInt64(flags.TextOnly),
						Limit:     int64(limit),
					}
					rawRows, err2 := queries.GetAllCaptionsOrdered(ctx, params)
					if err2 != nil {
						return err2
					}
					slog.Info("Fetched captions", "count", len(rawRows), "video_only", params.VideoOnly, "audio_only", params.AudioOnly)
					// Convert GetAllCaptionsOrderedRow to SearchCaptionsRow (rank=0 for non-search)
					for _, r := range rawRows {
						rows = append(rows, database.SearchCaptionsRow{
							MediaPath: r.MediaPath,
							Time:      r.Time,
							Text:      r.Text,
							Title:     r.Title,
							Type:      r.Type,
							Size:      r.Size,
							Duration:  r.Duration,
							Rank:      0, // No ranking for non-search queries
						})
					}
				}

				if err != nil {
					return err
				}

				if aggregate {
					// Aggregate captions by media path to get counts
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

					// For captions view, we want to return ALL individual caption rows
					// but with the aggregated count attached to each row
					for _, row := range rows {
						path := row.MediaPath
						stat := aggregated[path]
						m := models.MediaWithDB{
							Media: models.Media{
								Path:     path,
								Type:     models.NullStringPtr(row.Type),
								Title:    models.NullStringPtr(row.Title),
								Size:     models.NullInt64Ptr(row.Size),
								Duration: models.NullInt64Ptr(row.Duration),
							},
							DB:              dbPath,
							CaptionText:     row.Text.String,
							CaptionTime:     row.Time.Float64,
							CaptionCount:    stat.CaptionCount,
							CaptionDuration: stat.CaptionDuration,
						}
						media = append(media, m)
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

		if media == nil {
			media = []models.MediaWithDB{}
		}

		// Check if filter counts are requested
		includeCounts := q.Get("include_counts") == "true"
		var filterCounts *models.FilterBinsResponse
		if includeCounts {
			filterCounts = c.calculateFilterCounts(ctx, flags, dbs)
		}

		w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))
		w.Header().Set("Content-Type", "application/json")

		// Ensure media is not nil for JSON encoding
		items := media
		if items == nil {
			items = []models.MediaWithDB{}
		}

		if includeCounts && filterCounts != nil {
			response := map[string]any{
				"items":  items,
				"counts": filterCounts,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			json.NewEncoder(w).Encode(items)
		}
		return
	}

	media, err := query.MediaQuery(ctx, dbs, flags)
	if err != nil {
		slog.Error("Query failed", "dbs", dbs, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Query failed: " + err.Error()})
		return
	}
	if media == nil {
		media = []models.MediaWithDB{}
	}

	// Check if filter counts are requested
	includeCounts := q.Get("include_counts") == "true"
	var filterCounts *models.FilterBinsResponse
	if includeCounts {
		filterCounts = c.calculateFilterCounts(ctx, flags, dbs)
	}

	// Caption enrichment for main media grid
	if flags.WithCaptions && len(flags.Search) > 0 {
		queryStr := strings.Join(flags.Search, " ")
		for _, dbPath := range dbs {
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

				// Apply in-memory ranking for better relevance
				database.RankCaptionsResults(rows, queryStr)

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

	totalCount, err := query.MediaQueryCount(ctx, dbs, flags)
	if err != nil {
		slog.Error("Count query failed", "dbs", dbs, "error", err)
		// Don't fail the whole request just for count
	}

	if c.hasFfmpeg {
		for i := range media {
			media[i].Transcode = utils.GetTranscodeStrategy(media[i].Media).NeedsTranscode
		}
	}

	// Check if sort config contains expansion markers (like _related_media)
	sortConfig := flags.PlayInOrder
	if sortConfig == "" {
		sortConfig = flags.SortBy
	}

	if strings.Contains(sortConfig, "_related_media") && len(dbs) > 0 {
		// Use expansion-aware sorting with first database
		err := c.execDB(ctx, dbs[0], func(sqlDB *sql.DB) error {
			query.SortMediaWithExpansion(ctx, sqlDB, &media, flags)
			return nil
		})
		if err != nil {
			slog.Warn("SortMediaWithExpansion failed", "error", err)
			// Fall back to regular sorting
			query.SortMedia(media, flags)
		}
	} else {
		// Use regular sorting
		query.SortMedia(media, flags)
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// Ensure media is not nil for JSON encoding
	items := media
	if items == nil {
		items = []models.MediaWithDB{}
	}

	// Return counts with media if requested
	if includeCounts && filterCounts != nil {
		response := map[string]any{
			"items":  items,
			"counts": filterCounts,
		}
		json.NewEncoder(w).Encode(response)
	} else {
		json.NewEncoder(w).Encode(items)
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
	if !c.Trashcan {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "Server not started with --trashcan"})
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
			// Use raw SQL to update progress
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
	flags := c.parseFlags(r)
	path := r.URL.Query().Get("path")

	// Clean the path
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "/" {
		cleanPath = ""
	}

	// Calculate the depth of current path (number of path components)
	// "" = depth 0
	// "/media" = depth 1
	// "/media/videos" = depth 2
	currentDepth := 0
	if cleanPath != "" {
		parts := strings.FieldsFunc(cleanPath, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		currentDepth = len(parts)
	}
	targetDepth := currentDepth + 1 // We want to show children at this depth

	// Use MediaQuery to get all media with all filters applied
	// We need to override the path filter to only get children of current path
	originalWhere := flags.Where
	if cleanPath != "" {
		// Add path prefix filter (handle both separators)
		escapedPath := strings.ReplaceAll(cleanPath, "'", "''")
		flags.Where = append(flags.Where, "(path LIKE '"+escapedPath+"/%' OR path LIKE '"+escapedPath+"\\%')")
	} else {
		// At root level - no path filter needed
	}

	// Ensure we get all media (no limit for aggregation)
	flags.All = true
	flags.Limit = 0

	allMedia, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		slog.Error("Failed to fetch media for DU", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	// Restore original where clause for future use
	flags.Where = originalWhere

	// Separate maps for direct files and folders
	directFiles := make(map[string]*models.MediaWithDB)
	folders := make(map[string]*models.FolderStats)

	for i := range allMedia {
		media := &allMedia[i]
		// Use path as-is
		filePath := media.Path

		// Calculate the file's depth (number of path components)
		parts := strings.FieldsFunc(filePath, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		isAbsolute := len(filePath) > 0 && (filePath[0] == '/' || filePath[0] == '\\')
		sep := string(filepath.Separator)
		fileDepth := len(parts)

		if fileDepth == targetDepth {
			// This is a direct child file - add to files list
			directFiles[filePath] = media
		} else if fileDepth > targetDepth {
			// This file is in a subfolder - group by subfolder
			var parent string
			if currentDepth == 0 {
				// Root level: parent is first component
				if len(parts) > 0 {
					parent = parts[0]
					if isAbsolute {
						parent = sep + parent
					}
				}
			} else {
				// Subdirectory: parent is path up to targetDepth components
				if len(parts) >= targetDepth {
					parent = filepath.Join(parts[:targetDepth]...)
					if isAbsolute {
						parent = sep + parent
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
			if media.Size != nil {
				stat.TotalSize += *media.Size
			}
			if media.Duration != nil {
				stat.TotalDuration += *media.Duration
			}
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
