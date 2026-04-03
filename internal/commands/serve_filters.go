package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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

// computeFilterBinsData queries specified databases and collects size, duration, and parent count data
// This is the single source of truth for filter bins data collection
func (c *ServeCmd) computeFilterBinsData(ctx context.Context, flags models.GlobalFlags, filterToIgnore string, dbs []string) filterBinsData {
	var mu sync.Mutex
	// Track sizes and durations with their parent paths for filtering
	type itemWithParent struct {
		size       int64
		duration   int64
		modified   int64
		created    int64
		downloaded int64
		parentDir  string
		mediaType  string
	}
	var allItems []itemWithParent
	allParentCounts := make(map[string]int64)
	allTypeCounts := make(map[string]int64)

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
	} else if filterToIgnore == "media_type" {
		tempFlags.VideoOnly = false
		tempFlags.AudioOnly = false
		tempFlags.ImageOnly = false
		tempFlags.TextOnly = false
	} else if filterToIgnore == "modified" {
		tempFlags.ModifiedAfter = ""
		tempFlags.ModifiedBefore = ""
	} else if filterToIgnore == "created" {
		tempFlags.CreatedAfter = ""
		tempFlags.CreatedBefore = ""
	} else if filterToIgnore == "downloaded" {
		tempFlags.DownloadedAfter = ""
		tempFlags.DownloadedBefore = ""
	}

	fb := query.NewFilterBuilder(tempFlags)
	sqlQuery, args := fb.BuildSelect("path, size, duration, media_type, time_modified, time_created, time_downloaded")

	var wg sync.WaitGroup
	errChan := make(chan error, len(dbs))

	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			err := c.execDB(ctx, path, func(sqlDB *sql.DB) error {
				rows, err := sqlDB.QueryContext(ctx, sqlQuery, args...)
				if err != nil {
					return err
				}
				defer rows.Close()

				var localItems []itemWithParent
				localParentCounts := make(map[string]int64)
				localTypeCounts := make(map[string]int64)

				for rows.Next() {
					var p string
					var s, d, tm, tc, td sql.NullInt64
					var t sql.NullString
					if err := rows.Scan(&p, &s, &d, &t, &tm, &tc, &td); err == nil {
						parent := filepath.Dir(p)
						localParentCounts[parent]++

						mediaType := "unknown"
						if t.Valid && t.String != "" {
							mediaType = t.String
						}
						localTypeCounts[mediaType]++

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
						localItems = append(localItems, itemWithParent{
							size:       sizeVal,
							duration:   durVal,
							modified:   modVal,
							created:    creVal,
							downloaded: dlVal,
							parentDir:  parent,
							mediaType:  mediaType,
						})
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
				return nil
			})
			if err != nil {
				errChan <- fmt.Errorf("db %s: %w", path, err)
			}
		}(dbPath)
	}
	wg.Wait()
	close(errChan)

	// Collect and log any errors from database queries
	var queryErrors []error
	for err := range errChan {
		queryErrors = append(queryErrors, err)
	}
	if len(queryErrors) > 0 {
		for _, qErr := range queryErrors {
			slog.Warn("Database query failed in filter bins", "error", qErr)
		}
	}

	// Apply FileCounts (episodes) filter in post-processing if not being ignored
	// This matches the logic in MediaQuery for consistent filtering
	if filterToIgnore != "episodes" && flags.FileCounts != "" {
		r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		})
		if err == nil {
			// Filter parent counts to only those matching the file count range
			filteredParentCounts := make(map[string]int64)
			for parent, count := range allParentCounts {
				if r.Matches(count) {
					filteredParentCounts[parent] = count
				}
			}
			allParentCounts = filteredParentCounts

			// Filter items to only those from matching parents
			filteredItems := make([]itemWithParent, 0, len(allItems))
			newTypeCounts := make(map[string]int64)
			for _, item := range allItems {
				if _, ok := allParentCounts[item.parentDir]; ok {
					filteredItems = append(filteredItems, item)
					newTypeCounts[item.mediaType]++
				}
			}
			allItems = filteredItems
			allTypeCounts = newTypeCounts
		}
	}

	// Extract sizes and durations from filtered items
	allSizes := make([]int64, 0, len(allItems))
	allDurations := make([]int64, 0, len(allItems))
	allModified := make([]int64, 0, len(allItems))
	allCreated := make([]int64, 0, len(allItems))
	allDownloaded := make([]int64, 0, len(allItems))

	for _, item := range allItems {
		if item.size > 0 {
			allSizes = append(allSizes, item.size)
		}
		if item.duration > 0 {
			allDurations = append(allDurations, item.duration)
		}
		if item.modified > 0 {
			allModified = append(allModified, item.modified)
		}
		if item.created > 0 {
			allCreated = append(allCreated, item.created)
		}
		if item.downloaded > 0 {
			allDownloaded = append(allDownloaded, item.downloaded)
		}
	}

	return filterBinsData{
		sizes:        allSizes,
		durations:    allDurations,
		modified:     allModified,
		created:      allCreated,
		downloaded:   allDownloaded,
		parentCounts: allParentCounts,
		typeCounts:   allTypeCounts,
	}
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
	var bins []models.FilterBin
	var keys []string
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

// handleFilterBins handles the /api/filter-bins endpoint
func (c *ServeCmd) handleFilterBins(w http.ResponseWriter, r *http.Request) {
	flags := c.parseFlags(r)
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
	slog.Info("FilterBins computed",
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
func (c *ServeCmd) calculateFilterCounts(ctx context.Context, flags models.GlobalFlags, dbs []string) *models.FilterBinsResponse {
	// Use optimized version that uses SQL aggregation instead of fetching all rows
	return c.calculateFilterCountsOptimized(ctx, flags, dbs)
}

// calculateFilterCountsOptimized computes filter bin counts using optimized SQL queries
// This is MUCH faster than the original version for large libraries
func (c *ServeCmd) calculateFilterCountsOptimized(ctx context.Context, flags models.GlobalFlags, dbs []string) *models.FilterBinsResponse {
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

// computeFilterBinsDataOptimized is an optimized version that uses SQL aggregation
// instead of fetching all rows. This is MUCH faster for large libraries.
func (c *ServeCmd) computeFilterBinsDataOptimized(ctx context.Context, flags models.GlobalFlags, filterToIgnore string, dbs []string) filterBinsData {
	var mu sync.Mutex
	allParentCounts := make(map[string]int64)
	allTypeCounts := make(map[string]int64)

	// Collect histogram data for efficient percentile calculation
	type histogramData struct {
		sizes      []int64
		durations  []int64
		modified   []int64
		created    []int64
		downloaded []int64
	}
	var allHistogram histogramData

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
	} else if filterToIgnore == "media_type" {
		tempFlags.VideoOnly = false
		tempFlags.AudioOnly = false
		tempFlags.ImageOnly = false
		tempFlags.TextOnly = false
	} else if filterToIgnore == "modified" {
		tempFlags.ModifiedAfter = ""
		tempFlags.ModifiedBefore = ""
	} else if filterToIgnore == "created" {
		tempFlags.CreatedAfter = ""
		tempFlags.CreatedBefore = ""
	} else if filterToIgnore == "downloaded" {
		tempFlags.DownloadedAfter = ""
		tempFlags.DownloadedBefore = ""
	}

	var wg sync.WaitGroup
	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			c.execDB(ctx, path, func(sqlDB *sql.DB) error {
				// OPTIMIZATION: Use SQL aggregation instead of fetching all rows
				// This is much faster for large datasets

				// 1. Get parent counts from folder_stats materialized view (for episode filtering)
				// Fetch parent counts when computing episodes filter (filterToIgnore == "episodes")
				if filterToIgnore == "episodes" {
					parentCountQuery := `
						SELECT parent, file_count
						FROM folder_stats
					`
					rows, err := sqlDB.QueryContext(ctx, parentCountQuery)
					hasData := false
					folderStatsCount := 0
					if err != nil {
						slog.Debug("Parent count query failed (folder_stats may not exist)", "error", err)
					} else {
						defer rows.Close()
						for rows.Next() {
							hasData = true
							folderStatsCount++
							var parent string
							var cnt int64
							if err := rows.Scan(&parent, &cnt); err == nil {
								mu.Lock()
								allParentCounts[parent] += cnt
								mu.Unlock()
							}
						}
					}

					// If folder_stats was empty or had errors, fall back to counting from media table
					if !hasData {
						slog.Debug("folder_stats empty, using fallback to media table")
						parentCountQuery = `
							SELECT path
							FROM media
							WHERE COALESCE(time_deleted, 0) = 0
						`
						rows, err = sqlDB.QueryContext(ctx, parentCountQuery)
						if err != nil {
							slog.Debug("Fallback parent count query failed", "error", err)
						} else {
							defer rows.Close()
							localParentCounts := make(map[string]int64)
							fallbackCount := 0
							for rows.Next() {
								fallbackCount++
								var path string
								if err := rows.Scan(&path); err == nil {
									parent := filepath.Dir(path)
									localParentCounts[parent]++
								}
							}
							slog.Debug("Fallback parent count", "paths_scanned", fallbackCount, "unique_parents", len(localParentCounts))
							mu.Lock()
							for p, cnt := range localParentCounts {
								allParentCounts[p] += cnt
							}
							mu.Unlock()
						}
					} else {
						slog.Debug("folder_stats data", "rows", folderStatsCount, "unique_parents", len(allParentCounts))
					}
				}

				// 2. Get type counts
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
						if err := rows.Scan(&t, &cnt); err == nil {
							mu.Lock()
							allTypeCounts[t] += cnt
							mu.Unlock()
						}
					}
				}

				// 3. For percentile calculation, fetch a SAMPLE of actual values
				// This is MUCH faster than fetching all rows while still providing
				// accurate percentile estimates
				sampleQuery := `
					SELECT size, duration, time_modified, time_created, time_downloaded
					FROM media
					WHERE COALESCE(time_deleted, 0) = 0
					ORDER BY random()
					LIMIT 1000
				`
				rows, err = sqlDB.QueryContext(ctx, sampleQuery)
				if err == nil {
					defer rows.Close()
					var localSizes, localDurs, localMods, localCres, localDls []int64
					for rows.Next() {
						var s, d, tm, tc, td sql.NullInt64
						if err := rows.Scan(&s, &d, &tm, &tc, &td); err == nil {
							if s.Valid && s.Int64 > 0 && s.Int64 < 100*1024*1024*1024*1024 {
								localSizes = append(localSizes, s.Int64)
							}
							if d.Valid && d.Int64 > 0 && d.Int64 < 2678400 {
								localDurs = append(localDurs, d.Int64)
							}
							if tm.Valid && tm.Int64 > 0 {
								localMods = append(localMods, tm.Int64)
							}
							if tc.Valid && tc.Int64 > 0 {
								localCres = append(localCres, tc.Int64)
							}
							if td.Valid && td.Int64 > 0 {
								localDls = append(localDls, td.Int64)
							}
						}
					}
					mu.Lock()
					allHistogram.sizes = append(allHistogram.sizes, localSizes...)
					allHistogram.durations = append(allHistogram.durations, localDurs...)
					allHistogram.modified = append(allHistogram.modified, localMods...)
					allHistogram.created = append(allHistogram.created, localCres...)
					allHistogram.downloaded = append(allHistogram.downloaded, localDls...)
					mu.Unlock()
				}

				return nil
			})
		}(dbPath)
	}
	wg.Wait()

	// Apply FileCounts (episodes) filter in post-processing if not being ignored
	if filterToIgnore != "episodes" && flags.FileCounts != "" {
		r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		})
		if err == nil {
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
