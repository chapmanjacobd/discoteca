package server

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

func (c *Server) prepareFilterFlags(flags models.GlobalFlags, filterToIgnore string) models.GlobalFlags {
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
		tempFlags.FolderSizes = nil
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

func (c *Server) scanFilterRow(rows *sql.Rows) (itemWithParent, error) {
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

func (c *Server) collectRawItems(
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

func (c *Server) applyFileCountsFilter(
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

func (c *Server) computeFilterBinsData(
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
func (c *Server) HandleFilterBins(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	if rejectInvalidFlags(w, flags) {
		return
	}
	// Filter bins describe the media grid, which does not include sidecar
	// subtitle files.
	flags.HideSubtitles = true

	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}
	c.applySearchMode(r.Context(), &flags, dbs)

	sendJSON(w, http.StatusOK, c.buildFilterBinsResponse(r.Context(), flags, dbs))
}

// buildFilterBinsResponse computes each filter dimension independently so a
// selected value in one dimension does not shrink its own available range.
func (c *Server) buildFilterBinsResponse(
	ctx context.Context,
	flags models.GlobalFlags,
	dbs []string,
) *models.FilterBinsResponse {
	resp := &models.FilterBinsResponse{}

	epData := c.computeFilterBinsData(ctx, flags, "episodes", dbs)
	resp.EpisodesMinVal, resp.EpisodesMaxVal, resp.EpisodesPercentiles = buildEpisodeBins(epData.parentCounts)

	sizeData := c.computeFilterBinsData(ctx, flags, "size", dbs)
	resp.SizeMinVal, resp.SizeMaxVal, resp.SizePercentiles = buildSizeBins(sizeData.sizes)

	durData := c.computeFilterBinsData(ctx, flags, "duration", dbs)
	resp.DurationMinVal, resp.DurationMaxVal, resp.DurationPercentiles = buildDurationBins(durData.durations)

	modData := c.computeFilterBinsData(ctx, flags, "modified", dbs)
	resp.ModifiedMinVal, resp.ModifiedMaxVal, resp.ModifiedPercentiles = buildTimeBins(modData.modified)

	creData := c.computeFilterBinsData(ctx, flags, "created", dbs)
	resp.CreatedMinVal, resp.CreatedMaxVal, resp.CreatedPercentiles = buildTimeBins(creData.created)

	dlData := c.computeFilterBinsData(ctx, flags, "downloaded", dbs)
	resp.DownloadedMinVal, resp.DownloadedMaxVal, resp.DownloadedPercentiles = buildTimeBins(dlData.downloaded)

	typeData := c.computeFilterBinsData(ctx, flags, "media_type", dbs)
	resp.MediaType = buildTypeBins(typeData.typeCounts)

	return resp
}

// calculateFilterCounts computes filter bin counts for the current query
// This is used with include_counts to provide filter UI data alongside query results
// Each filter dimension is calculated independently (ignoring that filter) to avoid recursive constraints
func (c *Server) calculateFilterCounts(
	ctx context.Context,
	flags models.GlobalFlags,
	dbs []string,
) *models.FilterBinsResponse {
	return c.buildFilterBinsResponse(ctx, flags, dbs)
}
