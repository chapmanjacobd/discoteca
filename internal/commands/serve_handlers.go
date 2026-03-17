package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
	"github.com/chapmanjacobd/discoteca/internal/utils/pathutil"
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
	dbs, err := c.getDBs(flags)
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

		// Pagination for captions
		// When aggregate=true, we still need pagination but it's applied after aggregation
		if !flags.All && flags.Limit > 0 {
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
		sendError(w, http.StatusInternalServerError, "Query failed: "+err.Error())
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

	// Only do in-memory sorting for expansion-based sorting or multi-DB queries
	// Single DB queries are already sorted by SQL with proper LIMIT/OFFSET
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
	} else if len(dbs) > 1 {
		// Multi-DB queries need in-memory sorting to merge results from multiple databases
		query.SortMedia(media, flags)
	}
	// For single DB queries, SQL already sorted with LIMIT/OFFSET, so skip in-memory sorting

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
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.PlayResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !strings.HasPrefix(req.Path, "http") && !utils.FileExists(req.Path) {
		slog.Warn("File not found, marking as deleted in databases", "path", req.Path)
		c.markDeletedInAllDBs(r.Context(), req.Path, true)
		sendError(w, http.StatusNotFound, "File not found")
		return
	}

	// Trigger local playback
	slog.Info("Playing", "path", req.Path)
	cmd := exec.Command("mpv", req.Path)
	// We run it in background and don't wait for it
	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start mpv", "error", err)
		sendError(w, http.StatusInternalServerError, "Failed to start playback: "+err.Error())
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
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
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
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.ProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
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
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
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
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
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
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path  string  `json:"path"`
		Score float64 `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
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
	includeCounts := r.URL.Query().Get("include_counts") == "true"

	// Convert URL path to filesystem path
	// On Windows: forward slashes become backslashes
	// On Linux: forward slashes stay as-is
	cleanPath := pathutil.FromURL(path)
	if cleanPath == "." || cleanPath == "/" || cleanPath == "\\" {
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
		// Count non-empty parts (for absolute paths like /home/xk, count both "home" and "xk")
		for _, p := range parts {
			if p != "" {
				currentDepth++
			}
		}
	}
	targetDepth := currentDepth + 1 // We want to show children at this depth

	// Validate and filter databases
	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}

	// Use DU aggregation with filter support
	folderResults, err := query.AggregateDUByPathMultiDBWithFilters(r.Context(), c.Databases, cleanPath, targetDepth, currentDepth, flags)
	if err != nil {
		slog.Error("Failed to fetch DU folders", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	// Fetch direct files at target depth with filter support
	directFiles, err := query.FetchDUDirectFilesWithFilters(r.Context(), c.Databases, cleanPath, targetDepth, flags)
	if err != nil {
		slog.Error("Failed to fetch DU files", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	// Convert folder results to FolderStats
	folders := make([]models.FolderStats, 0, len(folderResults))
	for _, r := range folderResults {
		folders = append(folders, models.FolderStats{
			Path:          r.Path,
			Count:         r.Count,
			TotalSize:     r.TotalSize,
			TotalDuration: r.TotalDuration,
		})
	}

	// Sort folders
	sortBy := r.URL.Query().Get("sort")
	reverse := r.URL.Query().Get("reverse") == "true"

	if sortBy == "" {
		sortBy = "size"
		reverse = true
	}

	query.SortFolders(folders, sortBy, reverse)

	// Sort files using the same sort parameters
	sort.Slice(directFiles, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "size":
			iSize := int64(0)
			jSize := int64(0)
			if directFiles[i].Size != nil {
				iSize = *directFiles[i].Size
			}
			if directFiles[j].Size != nil {
				jSize = *directFiles[j].Size
			}
			less = iSize < jSize
		case "duration":
			iDur := int64(0)
			jDur := int64(0)
			if directFiles[i].Duration != nil {
				iDur = *directFiles[i].Duration
			}
			if directFiles[j].Duration != nil {
				jDur = *directFiles[j].Duration
			}
			less = iDur < jDur
		case "path", "name":
			less = directFiles[i].Path < directFiles[j].Path
		case "title":
			iTitle := ""
			jTitle := ""
			if directFiles[i].Title != nil {
				iTitle = *directFiles[i].Title
			}
			if directFiles[j].Title != nil {
				jTitle = *directFiles[j].Title
			}
			less = iTitle < jTitle
		default:
			less = directFiles[i].Path < directFiles[j].Path
		}
		if reverse {
			return !less
		}
		return less
	})

	// Apply pagination
	totalCount := len(folders) + len(directFiles)
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100 // Default page size
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Apply pagination to folders and files separately
	// Folders come first, then files
	if offset >= len(folders) {
		// Offset is beyond folders, apply to files
		fileStart := offset - len(folders)
		fileEnd := fileStart + limit
		if fileStart >= len(directFiles) {
			// Offset is beyond all items
			folders = folders[len(folders):]
			directFiles = directFiles[len(directFiles):]
		} else {
			folders = folders[len(folders):]
			if fileEnd > len(directFiles) {
				fileEnd = len(directFiles)
			}
			directFiles = directFiles[fileStart:fileEnd]
		}
	} else {
		// Offset is within folders
		folderEnd := offset + limit
		if folderEnd > len(folders) {
			// Pagination spans folders and files
			fileEnd := min(folderEnd-len(folders), len(directFiles))
			folders = folders[offset:]
			directFiles = directFiles[0:fileEnd]
		} else {
			// Pagination is within folders only
			folders = folders[offset:folderEnd]
		}
	}

	// Build response
	response := models.DUResponse{
		Folders:     folders,
		Files:       directFiles,
		FolderCount: len(folders),
		FileCount:   len(directFiles),
		TotalCount:  totalCount,
	}

	// Calculate filter bins if requested
	if includeCounts {
		response.Counts = c.calculateFilterCounts(r.Context(), flags, dbs)
	}

	// Set total count header for pagination
	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
