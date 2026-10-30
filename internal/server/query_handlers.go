package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	mediautil "github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// HandleQuery handles media searching and filtering.
// GET /api/query?search=...&category=...&rating=...&sort=...&limit=...&offset=...
func (c *Server) HandleQuery(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	if rejectInvalidFlags(w, flags) {
		return
	}
	// The media grid should not show sidecar subtitle files as playable cards,
	// unless the request explicitly asks for a path or extension.
	flags.HideSubtitles = true
	q := r.URL.Query()

	ctx, cancel := context.WithTimeout(r.Context(), c.searchTimeout())
	defer cancel()

	release, err := c.acquireSearchSlot(ctx)
	if err != nil {
		http.Error(w, "Server busy: search concurrency limit reached", http.StatusServiceUnavailable)
		return
	}
	defer release()

	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}
	c.applySearchMode(ctx, &flags, dbs)

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

type fetchCaptionsParams struct {
	dbPath    string
	queryStr  string
	flags     models.GlobalFlags
	limit     int
	aggregate bool
	media     *[]models.MediaWithDB
}

func (c *Server) fetchCaptionsFromDB(ctx context.Context, params fetchCaptionsParams) error {
	return c.execDB(ctx, params.dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
		queries := database.New(sqlDB)
		var rows []database.SearchCaptionsRow
		var captionErr error

		if params.queryStr != "" {
			rows, captionErr = c.getCaptionsWithContext(
				ctx,
				queries,
				CaptionsQueryParams{
					QueryStr:  params.queryStr,
					Limit:     int64(params.limit),
					VideoOnly: params.flags.VideoOnly,
					AudioOnly: params.flags.AudioOnly,
					ImageOnly: params.flags.ImageOnly,
					TextOnly:  params.flags.TextOnly,
					UseFTS:    params.flags.FTS && !params.flags.NoFTS,
				},
			)
		} else {
			captionsParams := database.GetAllCaptionsOrderedParams{
				VideoOnly: utils.BoolToInt64(params.flags.VideoOnly),
				AudioOnly: utils.BoolToInt64(params.flags.AudioOnly),
				ImageOnly: utils.BoolToInt64(params.flags.ImageOnly),
				TextOnly:  utils.BoolToInt64(params.flags.TextOnly),
				Limit:     int64(params.limit),
			}
			rawRows, err := queries.GetAllCaptionsOrdered(ctx, captionsParams)
			if err != nil {
				return err
			}
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

		c.appendCaptionRows(params.media, rows, params.dbPath, params.aggregate)
		return nil
	})
}

func (c *Server) applyPagination(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	if !flags.All && flags.Limit > 0 {
		start := flags.Offset
		if start > len(media) {
			return []models.MediaWithDB{}
		}
		end := min(start+flags.Limit, len(media))
		return media[start:end]
	}
	return media
}

// handleQueryCaptionsView handles the captions view mode of HandleQuery
func (c *Server) handleQueryCaptionsView(
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
	// Fetch every matching caption row so X-Total-Count describes the full
	// result set; the requested page is sliced afterwards.
	fetchLimit := 1000000

	aggregate := q.Get("aggregate") == "true"

	var allowedOrder map[string]int
	if filterFlags, active := captionMediaFilterFlags(flags); active {
		mediaFlags := filterFlags
		mediaFlags.All = true
		mediaFlags.Limit = 0
		mediaFlags.Offset = 0
		mediaFlags.FTS = false
		mediaFlags.NoFTS = true
		allowedMedia, err := query.MediaQuery(ctx, dbs, mediaFlags)
		if err == nil {
			allowedOrder = make(map[string]int, len(allowedMedia))
			for i, item := range allowedMedia {
				allowedOrder[item.Path] = i
			}
		} else {
			logging.Log.Warn("Caption media filters failed; showing unfiltered captions", "error", err)
		}
	}

	for _, dbPath := range dbs {
		params := fetchCaptionsParams{
			dbPath:    dbPath,
			queryStr:  queryStr,
			flags:     flags,
			limit:     fetchLimit,
			aggregate: aggregate,
			media:     &media,
		}
		if err := c.fetchCaptionsFromDB(ctx, params); err != nil {
			logging.Log.Error("Caption fetch failed", "db", dbPath, "error", err)
		}
	}

	if allowedOrder != nil {
		filtered := make([]models.MediaWithDB, 0, len(media))
		for _, item := range media {
			if _, ok := allowedOrder[item.Path]; ok {
				filtered = append(filtered, item)
			}
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			left, right := allowedOrder[filtered[i].Path], allowedOrder[filtered[j].Path]
			if left != right {
				return left < right
			}
			return filtered[i].CaptionTime < filtered[j].CaptionTime
		})
		media = filtered
	}

	totalCount := len(media)
	media = c.applyPagination(media, flags)

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

// captionMediaFilterFlags removes filters already handled by the caption SQL
// (search text, deleted rows, media type) and reports whether any remaining
// media-column filter needs a path lookup.
func captionMediaFilterFlags(flags models.GlobalFlags) (models.GlobalFlags, bool) {
	filtered := flags
	filtered.Search = nil
	filtered.VideoOnly = false
	filtered.AudioOnly = false
	filtered.ImageOnly = false
	filtered.TextOnly = false
	filtered.HideDeleted = false
	filtered.OnlyDeleted = false
	filtered.WithCaptions = false
	filtered.HideSubtitles = false
	filtered.FTS = false
	filtered.NoFTS = true
	return filtered, query.HasAnyFilter(filtered)
}

// appendCaptionRows appends caption rows to the media slice, optionally aggregating
func (c *Server) appendCaptionRows(
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
func (c *Server) handleQueryMediaGrid(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	flags models.GlobalFlags,
	dbs []string,
) {
	q := r.URL.Query()
	media, err := query.MediaQuery(ctx, dbs, flags)
	if err != nil {
		logging.Log.Error("Query failed", "dbs", dbs, "error", err)
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
		if isClientDisconnect(err) {
			logging.Log.Debug("Count query canceled", "dbs", dbs, "error", err)
		} else {
			logging.Log.Error("Count query failed", "dbs", dbs, "error", err)
		}
		totalCount = int64(len(media))
	}

	if c.hasFfmpeg {
		for i := range media {
			media[i].Transcode = mediautil.GetTranscodeStrategy(media[i].Media).NeedsTranscode
		}
	}

	c.sortMediaIfNeeded(ctx, media, flags, dbs)

	w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.encodeQueryResponse(w, media, filterCounts, includeCounts)
}

// enrichMediaWithCaptions adds caption snippets to media results
func (c *Server) enrichMediaWithCaptions(
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
			logging.Log.Error("Caption enrichment failed", "db", dbPath, "error", err2)
		}
	}
}

// sortMediaIfNeeded sorts media results based on sort configuration
func (c *Server) sortMediaIfNeeded(
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
			logging.Log.Warn("SortMediaWithExpansion failed", "error", err)
			query.SortMedia(media, flags)
		}
	} else if len(dbs) > 1 {
		query.SortMedia(media, flags)
	}
}

// encodeQueryResponse encodes the query response as JSON
func (c *Server) encodeQueryResponse(
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
			if isClientDisconnect(err) {
				logging.Log.Debug("Client disconnected while encoding response", "error", err)
			} else {
				logging.Log.Warn("Failed to encode response", "error", err)
			}
		}
	} else {
		if err := json.NewEncoder(w).Encode(items); err != nil {
			if isClientDisconnect(err) {
				logging.Log.Debug("Client disconnected while encoding response", "error", err)
			} else {
				logging.Log.Warn("Failed to encode response", "error", err)
			}
		}
	}
}

// isClientDisconnect reports whether an error is just an aborted HTTP request
// (user navigated, typed another search character, closed a tab, ...). Those
// are normal and should not be logged as server errors.
func isClientDisconnect(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context canceled") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "client disconnected")
}
