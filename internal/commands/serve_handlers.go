package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// HandleHealth returns OK if the server is running
func (c *ServeCmd) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// HandleQuery handles media searching and filtering.
// GET /api/query?search=...&category=...&rating=...&sort=...&limit=...&offset=...
func (c *ServeCmd) HandleQuery(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	q := r.URL.Query()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}

	resolvedFlags, err := query.ResolvePercentileFlags(ctx, dbs, flags)
	if err == nil {
		flags = resolvedFlags
	}

	if q.Get("view") == "captions" || q.Get("captions") == "true" {
		c.handleQueryCaptionsView(ctx, w, r, queryViewParams{
			Flags: flags,
			DBs:   dbs,
		})
		return
	}

	c.handleQueryMediaGrid(ctx, w, r, flags, dbs)
}

type queryViewParams struct {
	Flags models.GlobalFlags
	DBs   []string
}

// handleQueryCaptionsView handles the captions view mode of HandleQuery
func (c *ServeCmd) handleQueryCaptionsView(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	params queryViewParams,
) {
	q := r.URL.Query()
	flags := params.Flags
	dbs := params.DBs
	media := []models.MediaWithDB{}
	queryStr := strings.Join(flags.Search, " ")
	limit := flags.Limit
	if limit <= 0 {
		limit = 100
	}
	if flags.All {
		limit = 1000000
	}

	aggregate := q.Get("aggregate") == "true"

	for _, dbPath := range dbs {
		err2 := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			var rows []database.SearchCaptionsRow
			var captionErr error

			if queryStr != "" {
				rows, captionErr = c.getCaptionsWithContext(
					ctx,
					queries,
					CaptionsQueryParams{
						QueryStr:  queryStr,
						Limit:     int64(limit),
						VideoOnly: flags.VideoOnly,
						AudioOnly: flags.AudioOnly,
						ImageOnly: flags.ImageOnly,
						TextOnly:  flags.TextOnly,
					},
				)
			} else {
				captionsParams := database.GetAllCaptionsOrderedParams{
					VideoOnly: utils.BoolToInt64(flags.VideoOnly),
					AudioOnly: utils.BoolToInt64(flags.AudioOnly),
					ImageOnly: utils.BoolToInt64(flags.ImageOnly),
					TextOnly:  utils.BoolToInt64(flags.TextOnly),
					Limit:     int64(limit),
				}
				rawRows, err3 := queries.GetAllCaptionsOrdered(ctx, captionsParams)
				if err3 != nil {
					return err3
				}
				models.Log.Info(
					"Fetched captions",
					"count",
					len(rawRows),
					"video_only",
					captionsParams.VideoOnly,
					"audio_only",
					captionsParams.AudioOnly,
				)
				for _, row := range rawRows {
					rows = append(rows, database.SearchCaptionsRow{
						MediaPath: row.MediaPath,
						Time:      row.Time,
						Text:      row.Text,
						Title:     row.Title,
						MediaType: row.MediaType,
						Size:      row.Size,
						Duration:  row.Duration,
						Rank:      0,
					})
				}
			}

			if captionErr != nil {
				return captionErr
			}

			c.appendCaptionRows(&media, rows, dbPath, aggregate)
			return nil
		})
		if err2 != nil {
			models.Log.Error("Caption fetch failed", "db", dbPath, "error", err2)
		}
	}

	totalCount := len(media)

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

	includeCounts := q.Get("include_counts") == "true"
	var filterCounts *models.FilterBinsResponse
	if includeCounts {
		filterCounts = c.calculateFilterCounts(ctx, flags, dbs)
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))
	w.Header().Set("Content-Type", "application/json")
	c.encodeQueryResponse(w, media, filterCounts, includeCounts)
}

// appendCaptionRows appends caption rows to the media slice, optionally aggregating
func (c *ServeCmd) appendCaptionRows(
	media *[]models.MediaWithDB,
	rows []database.SearchCaptionsRow,
	dbPath string,
	aggregate bool,
) {
	if aggregate {
		aggregated := make(map[string]*models.MediaWithDB)
		for _, row := range rows {
			path := row.MediaPath
			if _, ok := aggregated[path]; !ok {
				aggregated[path] = &models.MediaWithDB{
					Media: models.Media{
						Path:      path,
						MediaType: models.NullStringPtr(row.MediaType),
						Title:     models.NullStringPtr(row.Title),
						Size:      models.NullInt64Ptr(row.Size),
						Duration:  models.NullInt64Ptr(row.Duration),
					},
					DB: dbPath,
				}
			}
			stat := aggregated[path]
			if stat.CaptionText == "" {
				stat.CaptionText = row.Text.String
			}
			if stat.CaptionTime == 0 {
				stat.CaptionTime = row.Time.Float64
			}
			stat.CaptionCount++
			if row.Time.Valid {
				stat.CaptionDuration += int64(row.Time.Float64)
			}
		}

		for _, row := range rows {
			path := row.MediaPath
			stat := aggregated[path]
			m := models.MediaWithDB{
				Media: models.Media{
					Path:      path,
					MediaType: models.NullStringPtr(row.MediaType),
					Title:     models.NullStringPtr(row.Title),
					Size:      models.NullInt64Ptr(row.Size),
					Duration:  models.NullInt64Ptr(row.Duration),
				},
				DB:              dbPath,
				CaptionText:     row.Text.String,
				CaptionTime:     row.Time.Float64,
				CaptionCount:    stat.CaptionCount,
				CaptionDuration: stat.CaptionDuration,
			}
			*media = append(*media, m)
		}
	} else {
		for _, row := range rows {
			m := models.MediaWithDB{
				Media: models.Media{
					Path:      row.MediaPath,
					MediaType: models.NullStringPtr(row.MediaType),
					Title:     models.NullStringPtr(row.Title),
				},
				DB:          dbPath,
				CaptionText: row.Text.String,
				CaptionTime: row.Time.Float64,
			}
			*media = append(*media, m)
		}
	}
}

// handleQueryMediaGrid handles the main media grid mode of HandleQuery
func (c *ServeCmd) handleQueryMediaGrid(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	flags models.GlobalFlags,
	dbs []string,
) {
	q := r.URL.Query()
	media, err := query.MediaQuery(ctx, dbs, flags)
	if err != nil {
		models.Log.Error("Query failed", "dbs", dbs, "error", err)
		sendError(w, http.StatusInternalServerError, "Query failed: "+err.Error())
		return
	}
	if media == nil {
		media = []models.MediaWithDB{}
	}

	includeCounts := q.Get("include_counts") == "true"
	var filterCounts *models.FilterBinsResponse
	if includeCounts {
		filterCounts = c.calculateFilterCounts(ctx, flags, dbs)
	}

	if flags.WithCaptions && len(flags.Search) > 0 {
		c.enrichMediaWithCaptions(ctx, media, flags, dbs)
	}

	totalCount, err := query.MediaQueryCount(ctx, dbs, flags)
	if err != nil {
		models.Log.Error("Count query failed", "dbs", dbs, "error", err)
	}

	if c.hasFfmpeg {
		for i := range media {
			media[i].Transcode = utils.GetTranscodeStrategy(media[i].Media).NeedsTranscode
		}
	}

	c.sortMediaIfNeeded(ctx, media, flags, dbs)

	w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.encodeQueryResponse(w, media, filterCounts, includeCounts)
}

// enrichMediaWithCaptions adds caption snippets to media results
func (c *ServeCmd) enrichMediaWithCaptions(
	ctx context.Context,
	media []models.MediaWithDB,
	flags models.GlobalFlags,
	dbs []string,
) {
	queryStr := strings.Join(flags.Search, " ")
	for _, dbPath := range dbs {
		err2 := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			rows, captionErr := queries.SearchCaptions(ctx, database.SearchCaptionsParams{
				Query: queryStr,
				Limit: 5,
			})
			if captionErr != nil {
				return captionErr
			}

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
		if err2 != nil {
			models.Log.Error("Caption enrichment failed", "db", dbPath, "error", err2)
		}
	}
}

// sortMediaIfNeeded sorts media results based on sort configuration
func (c *ServeCmd) sortMediaIfNeeded(
	ctx context.Context,
	media []models.MediaWithDB,
	flags models.GlobalFlags,
	dbs []string,
) {
	sortConfig := flags.PlayInOrder
	if sortConfig == "" {
		sortConfig = flags.SortBy
	}

	if strings.Contains(sortConfig, "_related_media") && len(dbs) > 0 {
		err := c.execDB(ctx, dbs[0], func(ctx context.Context, sqlDB *sql.DB) error {
			query.SortMediaWithExpansion(ctx, sqlDB, &media, flags)
			return nil
		})
		if err != nil {
			models.Log.Warn("SortMediaWithExpansion failed", "error", err)
			query.SortMedia(media, flags)
		}
	} else if len(dbs) > 1 {
		query.SortMedia(media, flags)
	}
}

// encodeQueryResponse encodes the query response as JSON
func (c *ServeCmd) encodeQueryResponse(
	w http.ResponseWriter,
	media []models.MediaWithDB,
	filterCounts *models.FilterBinsResponse,
	includeCounts bool,
) {
	items := media
	if items == nil {
		items = []models.MediaWithDB{}
	}

	if includeCounts && filterCounts != nil {
		response := map[string]any{
			"items":  items,
			"counts": filterCounts,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			models.Log.Warn("Failed to encode response", "error", err)
		}
	} else {
		if err := json.NewEncoder(w).Encode(items); err != nil {
			models.Log.Warn("Failed to encode response", "error", err)
		}
	}
}

// HandlePlay triggers local playback of a media file via mpv.
// POST /api/play
// Body: {"path": "..."}
func (c *ServeCmd) HandlePlay(w http.ResponseWriter, r *http.Request) {
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
		models.Log.Warn("File not found, marking as deleted in databases", "path", req.Path)
		c.markDeletedInAllDBs(r.Context(), req.Path, true)
		sendError(w, http.StatusNotFound, "File not found")
		return
	}

	// Trigger local playback
	models.Log.Info("Playing", "path", req.Path)
	cmd := exec.CommandContext(r.Context(), "mpv", req.Path)
	// We run it in background and don't wait for it
	if err := cmd.Start(); err != nil {
		models.Log.Error("Failed to start mpv", "error", err)
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
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			return queries.MarkDeleted(ctx, database.MarkDeletedParams{
				Path:        path,
				TimeDeleted: sql.NullInt64{Int64: deleteTime, Valid: deleted},
			})
		})
		if err != nil {
			models.Log.Error("Failed to mark file as deleted", "db", dbPath, "path", path, "error", err)
		}
	}
}

// HandleDelete marks a file as deleted or restores it in all databases.
// POST /api/delete
// Body: {"path": "...", "restore": bool}
func (c *ServeCmd) HandleDelete(w http.ResponseWriter, r *http.Request) {
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

// HandleProgress updates the playback progress for a media file.
// POST /api/progress
// Body: {"path": "...", "playhead": int64, "completed": bool}
func (c *ServeCmd) HandleProgress(w http.ResponseWriter, r *http.Request) {
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
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			// Use raw SQL to update progress
			// We want to increment play_count only once per session ideally, but for now we follow simple logic
			if _, err := sqlDB.ExecContext(ctx, `
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
			models.Log.Error("Failed to update progress", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// HandleMarkUnplayed resets play count and progress for a media file.
// POST /api/mark-unplayed
// Body: {"path": "..."}
func (c *ServeCmd) HandleMarkUnplayed(w http.ResponseWriter, r *http.Request) {
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
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			if _, err := sqlDB.ExecContext(ctx, `
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
			models.Log.Error("Failed to mark as unplayed", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// HandleMarkPlayed increments play count and resets progress for a media file.
// POST /api/mark-played
// Body: {"path": "..."}
func (c *ServeCmd) HandleMarkPlayed(w http.ResponseWriter, r *http.Request) {
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
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			if _, err := sqlDB.ExecContext(ctx, `
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
			models.Log.Error("Failed to mark as played", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// HandleRate updates the rating for a media file.
// POST /api/rate
// Body: {"path": "...", "score": float64}
func (c *ServeCmd) HandleRate(w http.ResponseWriter, r *http.Request) {
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
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			if _, err := sqlDB.ExecContext(
				ctx,
				"UPDATE media SET score = ? WHERE path = ?",
				req.Score,
				req.Path,
			); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			models.Log.Error("Failed to update rating", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (c *ServeCmd) HandleEvents(w http.ResponseWriter, r *http.Request) {
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

func (c *ServeCmd) HandleLs(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	isPartial, searchDir, searchBase := c.parseLsPath(path)

	resultsMap := make(map[string]LsEntry)
	counts := make(map[string]int)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			rows, err := c.queryLsRows(ctx, lsQueryParams{
				SQLDB:      sqlDB,
				IsPartial:  isPartial,
				SearchDir:  searchDir,
				SearchBase: searchBase,
			})
			if err != nil {
				return err
			}
			defer rows.Close()

			c.processLsRows(lsRowProcessParams{
				Rows:       rows,
				ResultsMap: &resultsMap,
				Counts:     counts,
				IsPartial:  isPartial,
				OrigPath:   path,
				SearchDir:  searchDir,
			})
			return rows.Err()
		})
		if err != nil {
			models.Log.Error("handleLs DB query failed", "db", dbPath, "error", err)
		}
	}

	results := c.buildLsResults(resultsMap, counts)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		models.Log.Warn("Failed to encode results", "error", err)
	}
}

// parseLsPath parses the input path and returns isPartial flag, searchDir, and searchBase
func (c *ServeCmd) parseLsPath(path string) (isPartial bool, searchDir, searchBase string) {
	if strings.HasPrefix(path, "./") {
		isPartial = true
		path = strings.TrimPrefix(path, "./")
	} else if !strings.HasPrefix(path, "/") {
		isPartial = true
	}

	if isPartial {
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash != -1 {
			searchDir = path[:lastSlash+1]
			searchBase = path[lastSlash+1:]
		} else {
			searchBase = path
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
	return isPartial, searchDir, searchBase
}

type lsQueryParams struct {
	SQLDB      *sql.DB
	IsPartial  bool
	SearchDir  string
	SearchBase string
}

// queryLsRows executes the appropriate SQL query based on search mode
func (c *ServeCmd) queryLsRows(ctx context.Context, params lsQueryParams) (*sql.Rows, error) {
	if params.IsPartial {
		if params.SearchDir == "" {
			return params.SQLDB.QueryContext(ctx, `
				SELECT path, media_type FROM media
				WHERE time_deleted = 0
				  AND path LIKE '%' || ? || '%'
				LIMIT 500`, params.SearchBase)
		}
		return params.SQLDB.QueryContext(ctx, `
			SELECT path, media_type FROM media
			WHERE time_deleted = 0
			  AND path LIKE '%' || ? || '%' || ? || '%'
			LIMIT 500`, params.SearchDir, params.SearchBase)
	}

	if params.SearchBase == "" {
		return params.SQLDB.QueryContext(ctx, `
			SELECT path, media_type FROM media
			WHERE time_deleted = 0
			  AND path LIKE ? || '%'
			LIMIT 500`, params.SearchDir)
	}
	return params.SQLDB.QueryContext(ctx, `
		SELECT path, media_type FROM media
		WHERE time_deleted = 0
		  AND path LIKE ? || '%'
		  AND path LIKE '%' || ? || '%'
		LIMIT 500`, params.SearchDir, params.SearchBase)
}

type lsRowProcessParams struct {
	Rows       *sql.Rows
	ResultsMap *map[string]LsEntry
	Counts     map[string]int
	IsPartial  bool
	OrigPath   string
	SearchDir  string
}

// processLsRows processes SQL rows and populates resultsMap and counts
func (c *ServeCmd) processLsRows(params lsRowProcessParams) {
	for params.Rows.Next() {
		var p, t sql.NullString
		if err := params.Rows.Scan(&p, &t); err != nil || !p.Valid {
			continue
		}
		fullPath := p.String

		if params.IsPartial && params.OrigPath == "" {
			c.processLsEmptySearchPath(fullPath, t.String, params.ResultsMap, params.Counts)
			continue
		}

		entryName, entryPath, isDir := c.extractLsEntry(fullPath, params.IsPartial, params.OrigPath, params.SearchDir)
		if entryName == "" {
			continue
		}

		c.mergeLsEntry(lsMergeParams{
			ResultsMap: params.ResultsMap,
			Counts:     params.Counts,
			EntryName:  entryName,
			EntryPath:  entryPath,
			IsDir:      isDir,
			MediaType:  t.String,
		})
	}
}

// processLsEmptySearchPath handles the special case of empty partial search (./)
func (c *ServeCmd) processLsEmptySearchPath(
	fullPath string,
	mediaType string,
	resultsMap *map[string]LsEntry,
	counts map[string]int,
) {
	segments := strings.Split(strings.Trim(fullPath, "/"), "/")
	current := "/"
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		entryPath := current + seg + "/"
		if !strings.HasSuffix(fullPath, "/") && seg == segments[len(segments)-1] {
			entryPath = current + seg
			counts[entryPath]++
			if _, ok := (*resultsMap)[entryPath]; !ok {
				(*resultsMap)[entryPath] = LsEntry{
					Name:      seg,
					Path:      entryPath,
					IsDir:     false,
					MediaType: mediaType,
				}
			}
			break
		}
		counts[entryPath]++
		if _, ok := (*resultsMap)[entryPath]; !ok {
			(*resultsMap)[entryPath] = LsEntry{Name: seg, Path: entryPath, IsDir: true}
		}
		current = entryPath
	}
}

// extractLsEntry extracts the entry name, path, and isDir flag from a full path
func (c *ServeCmd) extractLsEntry(
	fullPath string,
	isPartial bool,
	origPath string,
	searchDir string,
) (entryName, entryPath string, isDir bool) {
	if isPartial {
		matchStr := origPath
		if searchDir != "" {
			matchStr = searchDir
		}

		idx := strings.Index(fullPath, matchStr)
		if idx == -1 {
			return "", "", false
		}

		var prefix string
		var remaining string

		if strings.HasSuffix(matchStr, "/") {
			prefix = fullPath[:idx+len(matchStr)]
			remaining = fullPath[idx+len(matchStr):]
		} else {
			lastSlash := strings.LastIndex(fullPath[:idx], "/")
			if lastSlash == -1 {
				lastSlash = 0
			}
			prefix = fullPath[:lastSlash+1]
			remaining = fullPath[lastSlash+1:]
		}

		if remaining == "" {
			return "", "", false
		}

		if before, _, ok := strings.Cut(remaining, "/"); ok {
			return before, prefix + before + "/", true
		}
		return remaining, prefix + remaining, false
	}

	// Absolute path
	if !strings.HasPrefix(fullPath, searchDir) {
		return "", "", false
	}
	suffix := strings.TrimPrefix(fullPath, searchDir)
	if suffix == "" {
		return "", "", false
	}
	if before, _, ok := strings.Cut(suffix, "/"); ok {
		return before, searchDir + before + "/", true
	}
	return suffix, searchDir + suffix, false
}

type lsMergeParams struct {
	ResultsMap *map[string]LsEntry
	Counts     map[string]int
	EntryName  string
	EntryPath  string
	IsDir      bool
	MediaType  string
}

// mergeLsEntry adds or updates an entry in resultsMap
func (c *ServeCmd) mergeLsEntry(params lsMergeParams) {
	params.Counts[params.EntryPath]++
	if existing, ok := (*params.ResultsMap)[params.EntryPath]; ok {
		if !existing.IsDir && params.IsDir {
			(*params.ResultsMap)[params.EntryPath] = LsEntry{
				Name:  params.EntryName,
				Path:  params.EntryPath,
				IsDir: true,
			}
		}
	} else {
		(*params.ResultsMap)[params.EntryPath] = LsEntry{
			Name:      params.EntryName,
			Path:      params.EntryPath,
			IsDir:     params.IsDir,
			MediaType: params.MediaType,
		}
	}
}

// buildLsResults sorts and truncates the results
func (c *ServeCmd) buildLsResults(
	resultsMap map[string]LsEntry,
	counts map[string]int,
) []LsEntry {
	results := make([]LsEntry, 0, len(resultsMap))
	for _, entry := range resultsMap {
		results = append(results, entry)
	}

	sort.Slice(results, func(i, j int) bool {
		countI := counts[results[i].Path]
		countJ := counts[results[j].Path]
		if countI != countJ {
			return countI > countJ
		}
		if results[i].IsDir != results[j].IsDir {
			return results[i].IsDir
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	if len(results) > 20 {
		results = results[:20]
	}
	return results
}

func (c *ServeCmd) HandleDU(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	path := r.URL.Query().Get("path")
	includeCounts := r.URL.Query().Get("include_counts") == "true"

	cleanPath := c.normalizeDUPath(path)
	targetDepth := c.calculateDUTargetDepth(cleanPath)

	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}

	resolvedFlags, err := query.ResolvePercentileFlags(r.Context(), dbs, flags)
	if err != nil {
		models.Log.Warn("Failed to resolve percentile filters", "error", err)
		resolvedFlags = flags
	}

	folderResults, err := query.AggregateDUByPathMultiDBWithFilters(
		r.Context(),
		c.Databases,
		cleanPath,
		targetDepth,
		resolvedFlags,
	)
	if err != nil {
		models.Log.Error("Failed to fetch DU folders", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	directFiles, err := query.FetchDUDirectFilesWithFilters(
		r.Context(),
		c.Databases,
		cleanPath,
		targetDepth,
		resolvedFlags,
	)
	if err != nil {
		models.Log.Error("Failed to fetch DU files", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	folders := c.convertDUFolderResults(folderResults)

	sortBy, reverse := c.parseDUSortParams(r)
	query.SortFolders(folders, sortBy, reverse)
	c.sortDUFiles(directFiles, sortBy, reverse)

	limit, offset := c.parseDUPagination(r)
	totalCount := len(folders) + len(directFiles)
	folders, directFiles = c.applyDUPagination(folders, directFiles, offset, limit)

	response := models.DUResponse{
		Folders:     folders,
		Files:       directFiles,
		FolderCount: len(folders),
		FileCount:   len(directFiles),
		TotalCount:  totalCount,
	}

	if includeCounts {
		response.Counts = c.calculateFilterCounts(r.Context(), flags, dbs)
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		models.Log.Warn("Failed to encode response", "error", err)
	}
}

// normalizeDUPath converts URL path to filesystem path
func (c *ServeCmd) normalizeDUPath(path string) string {
	cleanPath := pathutil.FromURL(path)
	if cleanPath == "." || cleanPath == "/" || cleanPath == "\\" {
		cleanPath = ""
	}
	return cleanPath
}

// calculateDUTargetDepth calculates the depth of the path and returns targetDepth (currentDepth + 1)
func (c *ServeCmd) calculateDUTargetDepth(cleanPath string) int {
	currentDepth := 0
	if cleanPath != "" {
		parts := strings.FieldsFunc(cleanPath, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		for _, p := range parts {
			if p != "" {
				currentDepth++
			}
		}
	}
	return currentDepth + 1
}

// convertDUFolderResults converts folder query results to FolderStats slice
func (c *ServeCmd) convertDUFolderResults(folderResults []query.DUQueryResult) []models.FolderStats {
	folders := make([]models.FolderStats, 0, len(folderResults))
	for _, r := range folderResults {
		folders = append(folders, models.FolderStats{
			Path:          r.Path,
			Count:         r.Count,
			TotalSize:     r.TotalSize,
			TotalDuration: r.TotalDuration,
		})
	}
	return folders
}

// parseDUSortParams extracts sort and reverse parameters from request
func (c *ServeCmd) parseDUSortParams(r *http.Request) (sortBy string, reverse bool) {
	sortBy = r.URL.Query().Get("sort")
	reverse = r.URL.Query().Get("reverse") == "true"

	if sortBy == "" {
		sortBy = "size"
		reverse = true
	}
	return sortBy, reverse
}

// sortDUFiles sorts direct files using the given sort parameters
func (c *ServeCmd) sortDUFiles(directFiles []models.MediaWithDB, sortBy string, reverse bool) {
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
}

// parseDUPagination extracts limit and offset from request
func (c *ServeCmd) parseDUPagination(r *http.Request) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit = 100
	offset = 0

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
	return limit, offset
}

// applyDUPagination applies pagination to folders and files slices
func (c *ServeCmd) applyDUPagination(
	folders []models.FolderStats,
	directFiles []models.MediaWithDB,
	offset int,
	limit int,
) ([]models.FolderStats, []models.MediaWithDB) {
	if offset >= len(folders) {
		fileStart := offset - len(folders)
		fileEnd := fileStart + limit
		if fileStart >= len(directFiles) {
			return folders[len(folders):], directFiles[len(directFiles):]
		}
		if fileEnd > len(directFiles) {
			fileEnd = len(directFiles)
		}
		return folders[len(folders):], directFiles[fileStart:fileEnd]
	}

	folderEnd := offset + limit
	if folderEnd > len(folders) {
		fileEnd := min(folderEnd-len(folders), len(directFiles))
		return folders[offset:], directFiles[0:fileEnd]
	}
	return folders[offset:folderEnd], directFiles
}

func (c *ServeCmd) HandleEpisodes(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	if flags.Limit <= 0 {
		flags.All = true
		flags.Limit = 1000000
	}

	allMedia, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		models.Log.Error("Failed to fetch media for episodes", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	results := aggregate.GroupByParent(allMedia)

	// Set total count header for pagination
	w.Header().Set("X-Total-Count", strconv.Itoa(len(results)))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		models.Log.Warn("Failed to encode results", "error", err)
	}
}
