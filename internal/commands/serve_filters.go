package commands

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"sync"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// filterBinsData holds the raw collected data for building filter bins
type filterBinsData struct {
	sizes        []int64
	durations    []int64
	modified     []int64
	created      []int64
	downloaded   []int64
	parentCounts map[string]int64
	typeCounts   map[string]int64
}

func (c *ServeCmd) prepareFilterFlags(flags models.GlobalFlags, filterToIgnore string) models.GlobalFlags {
	tempFlags := flags
	tempFlags.Where = append([]string{}, flags.Where...)
	tempFlags.All = true
	tempFlags.Limit = 0

	switch filterToIgnore {
	case "size":
		tempFlags.Size = nil
	case "duration":
		tempFlags.Duration = nil
	case "episodes":
		tempFlags.FileCounts = ""
	case "media_type":
		tempFlags.VideoOnly = false
		tempFlags.AudioOnly = false
		tempFlags.ImageOnly = false
		tempFlags.TextOnly = false
	case "modified":
		tempFlags.ModifiedAfter = ""
		tempFlags.ModifiedBefore = ""
	case "created":
		tempFlags.CreatedAfter = ""
		tempFlags.CreatedBefore = ""
	case "downloaded":
		tempFlags.DownloadedAfter = ""
		tempFlags.DownloadedBefore = ""
	}
	return tempFlags
}

type itemWithParent struct {
	size       int64
	duration   int64
	modified   int64
	created    int64
	downloaded int64
	parentDir  string
	mediaType  string
}

func (c *ServeCmd) scanFilterRow(rows *sql.Rows) (itemWithParent, error) {
	var p string
	var s, d, tm, tc, td sql.NullInt64
	var t sql.NullString
	if err := rows.Scan(&p, &s, &d, &t, &tm, &tc, &td); err != nil {
		return itemWithParent{}, err
	}
	parent := filepath.Dir(p)
	mediaType := "unknown"
	if t.Valid && t.String != "" {
		mediaType = t.String
	}

	var sizeVal, durVal, modVal, creVal, dlVal int64
	if s.Valid && s.Int64 > 0 && s.Int64 < 100*1024*1024*1024*1024 {
		sizeVal = s.Int64
	}
	if d.Valid && d.Int64 > 0 && d.Int64 < 2678400 {
		durVal = d.Int64
	}
	if tm.Valid {
		modVal = tm.Int64
	}
	if tc.Valid {
		creVal = tc.Int64
	}
	if td.Valid {
		dlVal = td.Int64
	}
	return itemWithParent{
		size:       sizeVal,
		duration:   durVal,
		modified:   modVal,
		created:    creVal,
		downloaded: dlVal,
		parentDir:  parent,
		mediaType:  mediaType,
	}, nil
}

func (c *ServeCmd) collectRawItems(
	ctx context.Context,
	sqlQuery string,
	args []any,
	dbs []string,
) (allItems []itemWithParent, allParentCounts, allTypeCounts map[string]int64) {
	var mu sync.Mutex
	allParentCounts = make(map[string]int64)
	allTypeCounts = make(map[string]int64)
	var wg sync.WaitGroup

	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			_ = c.execDB(ctx, path, func(ctx context.Context, sqlDB *sql.DB) error {
				rows, err := sqlDB.QueryContext(ctx, sqlQuery, args...)
				if err != nil {
					return err
				}
				defer rows.Close()

				var localItems []itemWithParent
				localParentCounts := make(map[string]int64)
				localTypeCounts := make(map[string]int64)

				for rows.Next() {
					if item, err := c.scanFilterRow(rows); err == nil {
						localParentCounts[item.parentDir]++
						localTypeCounts[item.mediaType]++
						localItems = append(localItems, item)
					}
				}

				mu.Lock()
				allItems = append(allItems, localItems...)
				for k, v := range localParentCounts {
					allParentCounts[k] += v
				}
				for k, v := range localTypeCounts {
					allTypeCounts[k] += v
				}
				mu.Unlock()
				return rows.Err()
			})
		}(dbPath)
	}
	wg.Wait()
	return allItems, allParentCounts, allTypeCounts
}

func (c *ServeCmd) applyFileCountsFilter(
	flags models.GlobalFlags,
	allItemsIn []itemWithParent,
	allParentCountsIn map[string]int64,
) (allItems []itemWithParent, allParentCounts, allTypeCounts map[string]int64) {
	r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
		return strconv.ParseInt(s, 10, 64)
	})
	if err != nil {
		return allItemsIn, allParentCountsIn, nil
	}

	allParentCounts = make(map[string]int64)
	for parent, count := range allParentCountsIn {
		if r.Matches(count) {
			allParentCounts[parent] = count
		}
	}

	allItems = make([]itemWithParent, 0, len(allItemsIn))
	allTypeCounts = make(map[string]int64)
	for _, item := range allItemsIn {
		if _, ok := allParentCounts[item.parentDir]; ok {
			allItems = append(allItems, item)
			allTypeCounts[item.mediaType]++
		}
	}
	return allItems, allParentCounts, allTypeCounts
}

func (c *ServeCmd) computeFilterBinsData(
	ctx context.Context,
	flags models.GlobalFlags,
	filterToIgnore string,
	dbs []string,
) filterBinsData {
	tempFlags := c.prepareFilterFlags(flags, filterToIgnore)
	fb := query.NewFilterBuilder(tempFlags)
	sqlQuery, args := fb.BuildSelect(
		ctx,
		"path, size, duration, media_type, time_modified, time_created, time_downloaded",
	)

	allItems, allParentCounts, allTypeCounts := c.collectRawItems(ctx, sqlQuery, args, dbs)

	if filterToIgnore != "episodes" && flags.FileCounts != "" {
		allItems, allParentCounts, allTypeCounts = c.applyFileCountsFilter(flags, allItems, allParentCounts)
	}

	res := filterBinsData{
		parentCounts: allParentCounts,
		typeCounts:   allTypeCounts,
	}
	for _, item := range allItems {
		if item.size > 0 {
			res.sizes = append(res.sizes, item.size)
		}
		if item.duration > 0 {
			res.durations = append(res.durations, item.duration)
		}
		if item.modified > 0 {
			res.modified = append(res.modified, item.modified)
		}
		if item.created > 0 {
			res.created = append(res.created, item.created)
		}
		if item.downloaded > 0 {
			res.downloaded = append(res.downloaded, item.downloaded)
		}
	}
	return res
}

// buildSizeBins creates size filter bins from raw size data
// Returns only percentiles for the slider (frontend uses percentiles for range slider)
func buildSizeBins(sizes []int64) (minVal, maxVal int64, percentiles []int64) {
	if len(sizes) == 0 {
		return 0, 0, nil
	}

	minVal = slices.Min(sizes)
	maxVal = slices.Max(sizes)
	percentiles = utils.CalculatePercentiles(sizes)

	return minVal, maxVal, percentiles
}

// buildDurationBins creates duration filter bins from raw duration data
// Returns only percentiles for the slider (frontend uses percentiles for range slider)
func buildDurationBins(durations []int64) (minVal, maxVal int64, percentiles []int64) {
	if len(durations) == 0 {
		return 0, 0, nil
	}

	minVal = slices.Min(durations)
	maxVal = slices.Max(durations)
	percentiles = utils.CalculatePercentiles(durations)

	return minVal, maxVal, percentiles
}

// buildEpisodeBins creates episode count filter bins from parent counts
// Returns only percentiles for the slider (frontend uses percentiles for range slider)
func buildEpisodeBins(parentCounts map[string]int64) (minVal, maxVal int64, percentiles []int64) {
	if len(parentCounts) == 0 {
		return 0, 0, nil
	}

	var allCounts []int64
	for _, count := range parentCounts {
		allCounts = append(allCounts, count)
	}

	minVal = slices.Min(allCounts)
	maxVal = slices.Max(allCounts)
	percentiles = utils.CalculatePercentiles(allCounts)

	return minVal, maxVal, percentiles
}

// buildTypeBins creates type filter bins from type counts
func buildTypeBins(typeCounts map[string]int64) []models.FilterBin {
	bins := make([]models.FilterBin, 0, len(typeCounts))
	keys := make([]string, 0, len(typeCounts))
	for k := range typeCounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		bins = append(bins, models.FilterBin{
			Label: k,
			Value: typeCounts[k],
		})
	}
	return bins
}

// buildTimeBins creates time filter bins from raw time data
// Returns only percentiles for the slider (frontend uses percentiles for range slider)
func buildTimeBins(times []int64) (minVal, maxVal int64, percentiles []int64) {
	if len(times) == 0 {
		return 0, 0, nil
	}

	minVal = slices.Min(times)
	maxVal = slices.Max(times)
	percentiles = utils.CalculatePercentiles(times)

	return minVal, maxVal, percentiles
}

// HandleFilterBins handles the /api/filter-bins endpoint
func (c *ServeCmd) HandleFilterBins(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	if rejectInvalidFlags(w, flags) {
		return
	}
	// Filter bins describe the media grid, which does not include sidecar
	// subtitle files.
	flags.HideSubtitles = true
	q := r.URL.Query()

	// Validate and filter databases
	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}

	// Set flags to get all data for bin calculation
	flags.All = true
	flags.Limit = 0

	resp := models.FilterBinsResponse{}

	// Collect data for each filter type, ignoring that filter to get full distribution
	episodesOnly := q.Has("episodes")
	sizeOnly := q.Has("size") || q.Has("min_size") || q.Has("max_size")
	durationOnly := q.Has("duration") || q.Has("min_duration") || q.Has("max_duration")
	modifiedOnly := q.Has("min_modified") || q.Has("max_modified")
	createdOnly := q.Has("min_created") || q.Has("max_created")
	downloadedOnly := q.Has("min_downloaded") || q.Has("max_downloaded")

	// Get episode data - store min/max and percentiles
	epData := c.computeFilterBinsData(r.Context(), flags, "episodes", dbs)
	resp.EpisodesMinVal, resp.EpisodesMaxVal, resp.EpisodesPercentiles = buildEpisodeBins(epData.parentCounts)

	// Get size data - store min/max and percentiles
	sizeData := c.computeFilterBinsData(r.Context(), flags, "size", dbs)
	resp.SizeMinVal, resp.SizeMaxVal, resp.SizePercentiles = buildSizeBins(sizeData.sizes)

	// Get duration data - store min/max and percentiles
	durData := c.computeFilterBinsData(r.Context(), flags, "duration", dbs)
	resp.DurationMinVal, resp.DurationMaxVal, resp.DurationPercentiles = buildDurationBins(durData.durations)

	// Get modified data - store min/max and percentiles
	modData := c.computeFilterBinsData(r.Context(), flags, "modified", dbs)
	resp.ModifiedMinVal, resp.ModifiedMaxVal, resp.ModifiedPercentiles = buildTimeBins(modData.modified)

	// Get created data - store min/max and percentiles
	creData := c.computeFilterBinsData(r.Context(), flags, "created", dbs)
	resp.CreatedMinVal, resp.CreatedMaxVal, resp.CreatedPercentiles = buildTimeBins(creData.created)

	// Get downloaded data - store min/max and percentiles
	dlData := c.computeFilterBinsData(r.Context(), flags, "downloaded", dbs)
	resp.DownloadedMinVal, resp.DownloadedMaxVal, resp.DownloadedPercentiles = buildTimeBins(dlData.downloaded)

	// Get type data - keep as bins (special case, not percentile-based)
	typeData := c.computeFilterBinsData(r.Context(), flags, "media_type", dbs)
	resp.MediaType = buildTypeBins(typeData.typeCounts)

	// Log query info for debugging
	models.Log.Info("FilterBins computed",
		"episodesOnly", episodesOnly,
		"sizeOnly", sizeOnly,
		"durationOnly", durationOnly,
		"modifiedOnly", modifiedOnly,
		"createdOnly", createdOnly,
		"downloadedOnly", downloadedOnly,
		"databases", len(dbs),
		"sizeCount", len(sizeData.sizes),
		"durationCount", len(durData.durations),
		"parentCount", len(epData.parentCounts))

	sendJSON(w, http.StatusOK, resp)
}

// calculateFilterCounts computes filter bin counts for the current query
// This is used with include_counts to provide filter UI data alongside query results
// Each filter dimension is calculated independently (ignoring that filter) to avoid recursive constraints
func (c *ServeCmd) calculateFilterCounts(
	ctx context.Context,
	flags models.GlobalFlags,
	dbs []string,
) *models.FilterBinsResponse {
	// Use optimized version that uses SQL aggregation instead of fetching all rows
	return c.calculateFilterCountsOptimized(ctx, flags, dbs)
}

// calculateFilterCountsOptimized computes filter bin counts using optimized SQL queries
// This is MUCH faster than the original version for large libraries
func (c *ServeCmd) calculateFilterCountsOptimized(
	ctx context.Context,
	flags models.GlobalFlags,
	dbs []string,
) *models.FilterBinsResponse {
	resp := &models.FilterBinsResponse{}

	// Collect data for each filter type, ignoring that filter to get full distribution
	// This prevents recursive constraints where filtering by duration would shrink the duration range itself
	epData := c.computeFilterBinsDataOptimized(ctx, flags, "episodes", dbs)
	resp.EpisodesMinVal, resp.EpisodesMaxVal, resp.EpisodesPercentiles = buildEpisodeBins(epData.parentCounts)

	sizeData := c.computeFilterBinsDataOptimized(ctx, flags, "size", dbs)
	resp.SizeMinVal, resp.SizeMaxVal, resp.SizePercentiles = buildSizeBins(sizeData.sizes)

	durData := c.computeFilterBinsDataOptimized(ctx, flags, "duration", dbs)
	resp.DurationMinVal, resp.DurationMaxVal, resp.DurationPercentiles = buildDurationBins(durData.durations)

	modData := c.computeFilterBinsDataOptimized(ctx, flags, "modified", dbs)
	resp.ModifiedMinVal, resp.ModifiedMaxVal, resp.ModifiedPercentiles = buildTimeBins(modData.modified)

	creData := c.computeFilterBinsDataOptimized(ctx, flags, "created", dbs)
	resp.CreatedMinVal, resp.CreatedMaxVal, resp.CreatedPercentiles = buildTimeBins(creData.created)

	dlData := c.computeFilterBinsDataOptimized(ctx, flags, "downloaded", dbs)
	resp.DownloadedMinVal, resp.DownloadedMaxVal, resp.DownloadedPercentiles = buildTimeBins(dlData.downloaded)

	typeData := c.computeFilterBinsDataOptimized(ctx, flags, "media_type", dbs)
	resp.MediaType = buildTypeBins(typeData.typeCounts)

	return resp
}

func (c *ServeCmd) fetchParentCounts(
	ctx context.Context,
	sqlDB *sql.DB,
	allParentCounts map[string]int64,
	mu *sync.Mutex,
) {
	parentCountQuery := `
		SELECT parent, file_count
		FROM folder_stats
	`
	rows, err := sqlDB.QueryContext(ctx, parentCountQuery)
	hasData := false
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			hasData = true
			var parent string
			var cnt int64
			if scanErr := rows.Scan(&parent, &cnt); scanErr == nil {
				mu.Lock()
				allParentCounts[parent] += cnt
				mu.Unlock()
			}
		}
		if err := rows.Err(); err != nil {
			models.Log.Debug("Parent count query error", "error", err)
		}
	}

	if !hasData {
		c.fetchParentCountsFallback(ctx, sqlDB, allParentCounts, mu)
	}
}

func (c *ServeCmd) fetchParentCountsFallback(
	ctx context.Context,
	sqlDB *sql.DB,
	allParentCounts map[string]int64,
	mu *sync.Mutex,
) {
	parentCountQuery := `
		SELECT path
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0
	`
	rows, err := sqlDB.QueryContext(ctx, parentCountQuery)
	if err == nil {
		defer rows.Close()
		localParentCounts := make(map[string]int64)
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err == nil {
				parent := filepath.Dir(path)
				localParentCounts[parent]++
			}
		}
		if err := rows.Err(); err != nil {
			models.Log.Debug("Fallback parent count query error", "error", err)
		}
		mu.Lock()
		for p, cnt := range localParentCounts {
			allParentCounts[p] += cnt
		}
		mu.Unlock()
	}
}

func (c *ServeCmd) fetchTypeCounts(ctx context.Context, sqlDB *sql.DB, allTypeCounts map[string]int64, mu *sync.Mutex) {
	typeCountQuery := `
		SELECT COALESCE(NULLIF(media_type, ''), 'unknown') as t, COUNT(*) as cnt
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0
		GROUP BY media_type
	`
	rows, err := sqlDB.QueryContext(ctx, typeCountQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var cnt int64
			if scanErr := rows.Scan(&t, &cnt); scanErr == nil {
				mu.Lock()
				allTypeCounts[t] += cnt
				mu.Unlock()
			}
		}
		if err := rows.Err(); err != nil {
			models.Log.Debug("Type count query error", "error", err)
		}
	}
}

type histogramData struct {
	sizes      []int64
	durations  []int64
	modified   []int64
	created    []int64
	downloaded []int64
}

func (c *ServeCmd) scanHistogramRow(rows *sql.Rows, h *histogramData) {
	var s, d, tm, tc, td sql.NullInt64
	if err := rows.Scan(&s, &d, &tm, &tc, &td); err == nil {
		if s.Valid && s.Int64 > 0 && s.Int64 < 100*1024*1024*1024*1024 {
			h.sizes = append(h.sizes, s.Int64)
		}
		if d.Valid && d.Int64 > 0 && d.Int64 < 2678400 {
			h.durations = append(h.durations, d.Int64)
		}
		if tm.Valid && tm.Int64 > 0 {
			h.modified = append(h.modified, tm.Int64)
		}
		if tc.Valid && tc.Int64 > 0 {
			h.created = append(h.created, tc.Int64)
		}
		if td.Valid && td.Int64 > 0 {
			h.downloaded = append(h.downloaded, td.Int64)
		}
	}
}

func (c *ServeCmd) fetchSampleHistogram(
	ctx context.Context,
	sqlDB *sql.DB,
	allHistogram *histogramData,
	mu *sync.Mutex,
) {
	sampleQuery := `
		SELECT size, duration, time_modified, time_created, time_downloaded
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0
		ORDER BY random()
		LIMIT 1000
	`
	rows, err := sqlDB.QueryContext(ctx, sampleQuery)
	if err == nil {
		defer rows.Close()
		var local histogramData
		for rows.Next() {
			c.scanHistogramRow(rows, &local)
		}
		if err := rows.Err(); err != nil {
			models.Log.Debug("Sample query error", "error", err)
		}
		mu.Lock()
		allHistogram.sizes = append(allHistogram.sizes, local.sizes...)
		allHistogram.durations = append(allHistogram.durations, local.durations...)
		allHistogram.modified = append(allHistogram.modified, local.modified...)
		allHistogram.created = append(allHistogram.created, local.created...)
		allHistogram.downloaded = append(allHistogram.downloaded, local.downloaded...)
		mu.Unlock()
	}
}

func (c *ServeCmd) computeFilterBinsDataOptimized(
	ctx context.Context,
	flags models.GlobalFlags,
	filterToIgnore string,
	dbs []string,
) filterBinsData {
	var mu sync.Mutex
	allParentCounts := make(map[string]int64)
	allTypeCounts := make(map[string]int64)
	var allHistogram histogramData

	var wg sync.WaitGroup
	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			_ = c.execDB(ctx, path, func(ctx context.Context, sqlDB *sql.DB) error {
				if filterToIgnore == "episodes" {
					c.fetchParentCounts(ctx, sqlDB, allParentCounts, &mu)
				}
				c.fetchTypeCounts(ctx, sqlDB, allTypeCounts, &mu)
				c.fetchSampleHistogram(ctx, sqlDB, &allHistogram, &mu)
				return nil
			})
		}(dbPath)
	}
	wg.Wait()

	if filterToIgnore != "episodes" && flags.FileCounts != "" {
		if r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		}); err == nil {
			filteredParentCounts := make(map[string]int64)
			for parent, count := range allParentCounts {
				if r.Matches(count) {
					filteredParentCounts[parent] = count
				}
			}
			allParentCounts = filteredParentCounts
		}
	}

	return filterBinsData{
		sizes:        allHistogram.sizes,
		durations:    allHistogram.durations,
		modified:     allHistogram.modified,
		created:      allHistogram.created,
		downloaded:   allHistogram.downloaded,
		parentCounts: allParentCounts,
		typeCounts:   allTypeCounts,
	}
}
