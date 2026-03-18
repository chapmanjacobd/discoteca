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
	} else if filterToIgnore == "type" {
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
	sqlQuery, args := fb.BuildSelect("path, size, duration, type, time_modified, time_created, time_downloaded")

	var wg sync.WaitGroup
	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			c.execDB(ctx, path, func(sqlDB *sql.DB) error {
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
		}(dbPath)
	}
	wg.Wait()

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
func buildTimeBins(times []int64) (minVal, maxVal int64, bins []models.FilterBin, percentiles []int64) {
	if len(times) == 0 {
		return 0, 0, nil, nil
	}

	minVal = slices.Min(times)
	maxVal = slices.Max(times)
	percentiles = utils.CalculatePercentiles(times)

	p16 := int64(utils.Percentile(times, 16.6))
	p33 := int64(utils.Percentile(times, 33.3))
	p50 := int64(utils.Percentile(times, 50.0))
	p66 := int64(utils.Percentile(times, 66.6))
	p83 := int64(utils.Percentile(times, 83.3))

	tbins := []int64{minVal, p16, p33, p50, p66, p83, maxVal}
	// Sort to handle potential duplicates or out of order due to small datasets
	slices.Sort(tbins)

	// Remove duplicates
	uniqueBins := []int64{tbins[0]}
	for i := 1; i < len(tbins); i++ {
		if tbins[i] > uniqueBins[len(uniqueBins)-1] {
			uniqueBins = append(uniqueBins, tbins[i])
		}
	}

	for i := 0; i < len(uniqueBins)-1; i++ {
		minT := uniqueBins[i]
		maxT := uniqueBins[i+1]
		bins = append(bins, models.FilterBin{Min: minT, Max: maxT})
	}

	return minVal, maxVal, bins, percentiles
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

	// Get episode data - only store percentiles
	epData := c.computeFilterBinsData(r.Context(), flags, "episodes", dbs)
	_, _, _, resp.EpisodesPercentiles = buildEpisodeBins(epData.parentCounts)

	// Get size data - only store percentiles
	sizeData := c.computeFilterBinsData(r.Context(), flags, "size", dbs)
	_, _, _, resp.SizePercentiles = buildSizeBins(sizeData.sizes)

	// Get duration data - only store percentiles
	durData := c.computeFilterBinsData(r.Context(), flags, "duration", dbs)
	_, _, _, resp.DurationPercentiles = buildDurationBins(durData.durations)

	// Get modified data - only store percentiles
	modData := c.computeFilterBinsData(r.Context(), flags, "modified", dbs)
	_, _, _, resp.ModifiedPercentiles = buildTimeBins(modData.modified)

	// Get created data - only store percentiles
	creData := c.computeFilterBinsData(r.Context(), flags, "created", dbs)
	_, _, _, resp.CreatedPercentiles = buildTimeBins(creData.created)

	// Get downloaded data - only store percentiles
	dlData := c.computeFilterBinsData(r.Context(), flags, "downloaded", dbs)
	_, _, _, resp.DownloadedPercentiles = buildTimeBins(dlData.downloaded)

	// Get type data - keep as bins (special case, not percentile-based)
	typeData := c.computeFilterBinsData(r.Context(), flags, "type", dbs)
	resp.Type = buildTypeBins(typeData.typeCounts)

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
	resp := &models.FilterBinsResponse{}

	// Collect data for each filter type, ignoring that filter to get full distribution
	// This prevents recursive constraints where filtering by duration would shrink the duration range itself
	epData := c.computeFilterBinsData(ctx, flags, "episodes", dbs)
	_, _, _, resp.EpisodesPercentiles = buildEpisodeBins(epData.parentCounts)

	sizeData := c.computeFilterBinsData(ctx, flags, "size", dbs)
	_, _, _, resp.SizePercentiles = buildSizeBins(sizeData.sizes)

	durData := c.computeFilterBinsData(ctx, flags, "duration", dbs)
	_, _, _, resp.DurationPercentiles = buildDurationBins(durData.durations)

	modData := c.computeFilterBinsData(ctx, flags, "modified", dbs)
	_, _, _, resp.ModifiedPercentiles = buildTimeBins(modData.modified)

	creData := c.computeFilterBinsData(ctx, flags, "created", dbs)
	_, _, _, resp.CreatedPercentiles = buildTimeBins(creData.created)

	dlData := c.computeFilterBinsData(ctx, flags, "downloaded", dbs)
	_, _, _, resp.DownloadedPercentiles = buildTimeBins(dlData.downloaded)

	typeData := c.computeFilterBinsData(ctx, flags, "type", dbs)
	resp.Type = buildTypeBins(typeData.typeCounts)

	return resp
}
