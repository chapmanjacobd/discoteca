package query

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
	"github.com/chapmanjacobd/discoteca/internal/utils/pathutil"
)

// DUQueryResult holds the result of a DU aggregation query
type DUQueryResult struct {
	Path          string
	Count         int
	TotalSize     int64
	TotalDuration int64
}

// hasActiveFilters checks if any filters are active that require slow-path processing
// Returns true if FileCounts (episodes) filter is active, which requires folder backfiltering
func hasActiveFilters(flags models.GlobalFlags) bool {
	// FileCounts requires folder-level aggregation before filtering
	if flags.FileCounts != "" {
		return true
	}
	// FolderCounts also requires folder-level filtering
	if flags.FolderCounts != "" {
		return true
	}
	// FolderSizes requires post-aggregation filtering
	if len(flags.FolderSizes) > 0 {
		return true
	}
	return false
}

// hasBasicFilters checks if only basic SQL-level filters are active (size, duration, media_type, etc.)
// These can be applied directly in SQL without folder backfiltering
func hasBasicFilters(flags models.GlobalFlags) bool {
	if flags.VideoOnly || flags.AudioOnly || flags.ImageOnly || flags.TextOnly {
		return true
	}
	if len(flags.Size) > 0 || len(flags.Duration) > 0 {
		return true
	}
	if flags.PlayCountMin > 0 || flags.PlayCountMax > 0 {
		return true
	}
	if flags.Genre != "" || len(flags.Language) > 0 || len(flags.Ext) > 0 {
		return true
	}
	if flags.ModifiedAfter != "" || flags.ModifiedBefore != "" {
		return true
	}
	if flags.CreatedAfter != "" || flags.CreatedBefore != "" {
		return true
	}
	if flags.DownloadedAfter != "" || flags.DownloadedBefore != "" {
		return true
	}
	if flags.OnlyDeleted || flags.HideDeleted {
		return true
	}
	if flags.OnlineMediaOnly || flags.LocalMediaOnly {
		return true
	}
	if len(flags.Search) > 0 || len(flags.Include) > 0 || len(flags.PathContains) > 0 {
		return true
	}
	if len(flags.Category) > 0 {
		return true
	}
	if flags.Watched != nil || flags.Unfinished || flags.InProgress || flags.Completed {
		return true
	}
	if flags.Partial != "" {
		return true
	}
	return false
}

// AggregateDUByPath performs SQL-level aggregation for DU mode
// This is much faster than fetching all media and aggregating in Go
func AggregateDUByPath(
	ctx context.Context,
	dbPath, pathPrefix string,
	targetDepth int,
) ([]DUQueryResult, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	// Build SQL query for aggregation
	// We use SUBSTR and INSTR to extract the path at target depth
	// For cross-platform compatibility, we normalize separators in the SQL query
	// by replacing backslashes with forward slashes for comparison
	var query string
	var args []any

	if pathPrefix == "" {
		// Root level: aggregate by first path component
		// Normalize paths by replacing backslashes with forward slashes for consistent grouping
		query = `
			SELECT
				CASE
					WHEN substr(replace(path, '\\', '/'), 1, 1) = '/' THEN
						substr(replace(path, '\\', '/'), 1, 
							CASE 
								WHEN instr(substr(replace(path, '\\', '/'), 2), '/') > 0 
								THEN instr(substr(replace(path, '\\', '/'), 2), '/')
								ELSE length(replace(path, '\\', '/'))
							END
						)
					ELSE
						CASE
							WHEN instr(replace(path, '\\', '/'), '/') > 0 
							THEN substr(replace(path, '\\', '/'), 1, instr(replace(path, '\\', '/'), '/') - 1)
							ELSE replace(path, '\\', '/')
						END
				END as agg_path,
				COUNT(*) as count,
				COALESCE(SUM(size), 0) as total_size,
				COALESCE(SUM(duration), 0) as total_duration
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
			GROUP BY agg_path
			ORDER BY total_size DESC
		`
	} else {
		// Subdirectory level: aggregate by path at target depth
		// Normalize both the prefix and stored paths to forward slashes for matching
		// This ensures /videos/movies matches both /videos/movies/file and \videos\movies\file
		normalizedPrefix := strings.ReplaceAll(pathPrefix, "\\", "/")
		escapedPrefix := strings.ReplaceAll(normalizedPrefix, "'", "''")

		query = `
			SELECT
				CASE
					WHEN substr(?, 1, 1) = '/' THEN
						? || substr(
							substr(replace(path, '\\', '/'), length(?) + 1),
							1,
							CASE
								WHEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') > 0
								THEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/')
								ELSE length(substr(replace(path, '\\', '/'), length(?) + 1))
							END
						)
					ELSE
						? || CASE
							WHEN instr(substr(replace(path, '\\', '/'), length(?) + 1), '/') > 0
							THEN substr(substr(replace(path, '\\', '/'), length(?) + 1), 1, instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') - 1)
							ELSE substr(replace(path, '\\', '/'), length(?) + 1)
						END
				END as agg_path,
				COUNT(*) as count,
				COALESCE(SUM(size), 0) as total_size,
				COALESCE(SUM(duration), 0) as total_duration
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
				AND replace(path, '\\', '/') LIKE ? || '/%'
				AND (length(replace(path, '\\', '/')) - length(replace(replace(path, '\\', '/'), '/', ''))) > ?
			GROUP BY agg_path
			ORDER BY total_size DESC
		`
		// Add args for the placeholders (12 total)
		args = append(args,
			normalizedPrefix, // 1: substr(?, 1, 1)
			normalizedPrefix, // 2: ? || substr(
			normalizedPrefix, // 3: length(?) + 1
			normalizedPrefix, // 4: length(?) + 2
			normalizedPrefix, // 5: length(?) + 2
			normalizedPrefix, // 6: length(?) + 1
			normalizedPrefix, // 7: ? || CASE
			normalizedPrefix, // 8: length(?) + 1
			normalizedPrefix, // 9: length(?) + 1
			normalizedPrefix, // 10: length(?) + 2
			normalizedPrefix, // 11: length(?) + 1
			escapedPrefix,    // 12: LIKE ? || '/%'
			targetDepth,      // 13: separator count > targetDepth
		)
	}

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DUQueryResult
	for rows.Next() {
		var r DUQueryResult
		if err := rows.Scan(&r.Path, &r.Count, &r.TotalSize, &r.TotalDuration); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// AggregateDUByPathMultiDB performs SQL-level aggregation across multiple databases
func AggregateDUByPathMultiDB(
	ctx context.Context,
	dbPaths []string,
	pathPrefix string,
	targetDepth int,
) ([]DUQueryResult, error) {
	allResults := make([]DUQueryResult, 0)

	for _, dbPath := range dbPaths {
		results, err := AggregateDUByPath(ctx, dbPath, pathPrefix, targetDepth)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, results...)
	}

	// Aggregate results from multiple databases by path
	pathMap := make(map[string]*DUQueryResult)
	for _, r := range allResults {
		if _, ok := pathMap[r.Path]; !ok {
			pathMap[r.Path] = &DUQueryResult{Path: r.Path}
		}
		pathMap[r.Path].Count += r.Count
		pathMap[r.Path].TotalSize += r.TotalSize
		pathMap[r.Path].TotalDuration += r.TotalDuration
	}

	// Convert map to slice
	finalResults := make([]DUQueryResult, 0, len(pathMap))
	for _, r := range pathMap {
		finalResults = append(finalResults, *r)
	}

	return finalResults, nil
}

// FetchDUDirectFiles fetches files at the target depth for DU mode
func FetchDUDirectFiles(
	ctx context.Context,
	dbPaths []string,
	pathPrefix string,
	targetDepth int,
) ([]models.MediaWithDB, error) {
	allFiles := make([]models.MediaWithDB, 0)

	for _, dbPath := range dbPaths {
		sqlDB, err := db.Connect(dbPath)
		if err != nil {
			return nil, err
		}

		var query string
		var args []any

		if pathPrefix == "" {
			// Root level: files with no directory separator
			// Normalize paths to forward slashes for consistent matching
			query = `
				SELECT path, title, duration, size, time_created, time_modified,
				       time_deleted, time_first_played, time_last_played, play_count,
				       playhead, album, artist, genre, categories, description,
				       language, time_downloaded, score, video_codecs, audio_codecs,
				       subtitle_codecs, width, height, media_type
				FROM media
				WHERE COALESCE(time_deleted, 0) = 0
				  AND instr(replace(path, '\\', '/'), '/') = 0
			`
		} else {
			// Subdirectory: files at exactly target depth
			// Normalize prefix to forward slashes for cross-platform matching
			normalizedPrefix := strings.ReplaceAll(pathPrefix, "\\", "/")
			escapedPrefix := strings.ReplaceAll(normalizedPrefix, "'", "''")

			// Check if pathPrefix is absolute (after normalization)
			isAbs := len(normalizedPrefix) > 0 && normalizedPrefix[0] == '/'
			separatorCount := targetDepth
			if !isAbs {
				separatorCount = targetDepth - 1
			}

			// Normalize paths to forward slashes for consistent matching
			query = `
				SELECT path, title, duration, size, time_created, time_modified,
				       time_deleted, time_first_played, time_last_played, play_count,
				       playhead, album, artist, genre, categories, description,
				       language, time_downloaded, score, video_codecs, audio_codecs,
				       subtitle_codecs, width, height, media_type
				FROM media
				WHERE COALESCE(time_deleted, 0) = 0
				  AND replace(path, '\\', '/') LIKE ? || '/%'
				  AND (
					length(replace(path, '\\', '/')) - length(replace(replace(path, '\\', '/'), '/', ''))
				  ) = ?
			`
			args = append(args, escapedPrefix, separatorCount)
		}

		rows, err := sqlDB.QueryContext(ctx, query, args...)
		if err != nil {
			sqlDB.Close()
			return nil, err
		}

		files, err := ScanMedia(rows, dbPath)
		rows.Close()
		sqlDB.Close()
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}

// AggregateMedia media using the specified aggregation mode
func AggregateMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.FolderStats {
	return AggregateMediaWithMode(media, flags, false)
}

// AggregateMediaWithMode aggregates media with optional fast mode
// fastMode: skip expensive calculations (median, file storage) for faster navigation
func AggregateMediaWithMode(media []models.MediaWithDB, flags models.GlobalFlags, fastMode bool) []models.FolderStats {
	var stats []models.FolderStats
	if flags.GroupByExtensions {
		stats = AggregateExtensions(media, fastMode)
	} else if flags.GroupBySize {
		stats = AggregateSizeBuckets(media, fastMode)
	} else {
		stats = AggregateByDepth(media, flags, fastMode)
	}

	// Post-aggregation filtering (unchanged)
	if flags.FoldersOnly || flags.FilesOnly || flags.FolderSizes != nil || flags.FileCounts != "" ||
		flags.FolderCounts != "" {

		var filtered []models.FolderStats
		for _, f := range stats {
			keep := true
			if flags.FoldersOnly && f.Count == 0 {
				keep = false
			} else if flags.FilesOnly && f.Count > 0 {
				keep = false
			}

			if keep && len(flags.FolderSizes) > 0 {
				for _, fs := range flags.FolderSizes {
					if r, err := utils.ParseRange(fs, utils.HumanToBytes); err == nil {
						if !r.Matches(f.TotalSize) {
							keep = false
							break
						}
					}
				}
			}
			if keep && flags.FileCounts != "" {
				if r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
					return strconv.ParseInt(s, 10, 64)
				}); err == nil {
					// In Python, file_counts applies to 'exists' or 'count'
					if !r.Matches(int64(utils.Max(f.ExistsCount, f.Count))) {
						keep = false
					}
				}
			}
			if keep && flags.FolderCounts != "" {
				if r, err := utils.ParseRange(flags.FolderCounts, func(s string) (int64, error) {
					return strconv.ParseInt(s, 10, 64)
				}); err == nil {
					if !r.Matches(int64(f.FolderCount)) {
						keep = false
					}
				}
			}
			if keep {
				filtered = append(filtered, f)
			}
		}
		stats = filtered
	}

	return stats
}

func AggregateExtensions(media []models.MediaWithDB, fastMode ...bool) []models.FolderStats {
	fast := len(fastMode) > 0 && fastMode[0]
	groups := make(map[string]*models.FolderStats)
	for _, m := range media {
		ext := strings.ToLower(filepath.Ext(m.Path))
		if ext == "" {
			ext = "no extension"
		}
		if _, ok := groups[ext]; !ok {
			groups[ext] = &models.FolderStats{Path: ext}
		}
		updateStats(groups[ext], m, true)
	}
	return finalizeStatsWithOptions(groups, !fast, !fast)
}

func AggregateSizeBuckets(media []models.MediaWithDB, fastMode ...bool) []models.FolderStats {
	fast := len(fastMode) > 0 && fastMode[0]
	baseEdges := []int64{2, 5, 10}
	var multipliers []int64
	multipliers = append(multipliers, baseEdges...)
	for _, n := range baseEdges {
		multipliers = append(multipliers, n*10)
	}
	for _, n := range baseEdges {
		multipliers = append(multipliers, n*100)
	}

	unitMultiplier := int64(1024)
	units := []int64{
		1,
		unitMultiplier,
		unitMultiplier * unitMultiplier,
		unitMultiplier * unitMultiplier * unitMultiplier,
		unitMultiplier * unitMultiplier * unitMultiplier * unitMultiplier,
	}

	var binEdges []float64
	binEdges = append(binEdges, 0.0)
	for _, unit := range units {
		for _, mMult := range multipliers {
			binEdges = append(binEdges, float64(mMult*unit))
		}
	}
	binEdges = append(binEdges, math.Inf(1))

	groups := make(map[string]*models.FolderStats)
	for _, m := range media {
		size := float64(0)
		if m.Size != nil {
			size = float64(*m.Size)
		}

		var label string
		for i := range len(binEdges) - 1 {
			if size >= binEdges[i] && size < binEdges[i+1] {
				label = fmt.Sprintf(
					"%s-%s",
					utils.FormatSize(int64(binEdges[i])),
					utils.FormatSize(int64(binEdges[i+1])),
				)
				if binEdges[i+1] == math.Inf(1) {
					label = fmt.Sprintf(">%s", utils.FormatSize(int64(binEdges[i])))
				}
				break
			}
		}

		if _, ok := groups[label]; !ok {
			groups[label] = &models.FolderStats{Path: label}
		}
		updateStats(groups[label], m, true)
	}
	return finalizeStatsWithOptions(groups, !fast, !fast)
}

func AggregateByDepth(media []models.MediaWithDB, flags models.GlobalFlags, fastMode ...bool) []models.FolderStats {
	fast := len(fastMode) > 0 && fastMode[0]
	groups := make(map[string]*models.FolderStats)

	for _, m := range media {
		path := filepath.Clean(m.Path)
		// Use pathutil for proper cross-platform path handling
		parts, isAbs := pathutil.Split(path)

		// 1. Add the file itself if it matches depth
		if flags.Depth > 0 && len(parts) == flags.Depth {
			if _, ok := groups[path]; !ok {
				groups[path] = &models.FolderStats{Path: path}
			}
			updateStats(groups[path], m, false)
		}

		// 2. Add folders
		if flags.Parents {
			// MinDepth refers to the number of path components, so MinDepth=1 means start from first component
			start := max(flags.MinDepth-1, 0)
			for d := start; d < len(parts); d++ {
				if flags.MaxDepth > 0 && d+1 > flags.MaxDepth {
					break
				}
				parent := pathutil.Join(parts[:d+1], isAbs)
				if _, ok := groups[parent]; !ok {
					groups[parent] = &models.FolderStats{Path: parent}
				}
				updateStats(groups[parent], m, true)
			}
		} else if flags.BigDirs {
			// BigDirs: group by immediate parent directory
			parent := m.Parent()
			if _, ok := groups[parent]; !ok {
				groups[parent] = &models.FolderStats{Path: parent}
			}
			updateStats(groups[parent], m, true)
		} else if flags.Depth > 0 && len(parts) > flags.Depth {
			// Group at depth (e.g., depth=1 -> "/media" or "media", depth=2 -> "/media/video")
			var parent string
			if flags.Depth < len(parts) {
				parent = pathutil.Join(parts[:flags.Depth], isAbs)
			} else {
				parent = pathutil.Join(parts, isAbs)
			}
			if _, ok := groups[parent]; !ok {
				groups[parent] = &models.FolderStats{Path: parent}
			}
			updateStats(groups[parent], m, true)
		} else {
			// Default to immediate parent (when Depth=0 or file is at/below target depth)
			parent := m.Parent()
			if _, ok := groups[parent]; !ok {
				groups[parent] = &models.FolderStats{Path: parent}
			}
			updateStats(groups[parent], m, true)
		}
	}

	return finalizeStatsWithOptions(groups, !fast, !fast)
}

func updateStats(f *models.FolderStats, m models.MediaWithDB, isFolder bool) {
	if isFolder {
		f.Count++
	}
	isDeleted := m.TimeDeleted != nil && *m.TimeDeleted > 0
	if isDeleted {
		f.DeletedCount++
	} else {
		f.ExistsCount++
		if m.Size != nil {
			f.TotalSize += *m.Size
		}
		if m.Duration != nil {
			f.TotalDuration += *m.Duration
		}
	}

	if m.PlayCount != nil && *m.PlayCount > 0 {
		f.PlayedCount++
	}
	f.Files = append(f.Files, m)
}

// finalizeStatsWithOptions finalizes stats with optional expensive calculations
// calculateMedians: whether to calculate median size/duration (expensive)
// storeFiles: whether to store file references (memory intensive)
func finalizeStatsWithOptions(
	groups map[string]*models.FolderStats,
	calculateMedians bool,
	storeFiles ...bool,
) []models.FolderStats {
	shouldStoreFiles := len(storeFiles) > 0 && storeFiles[0]

	// Identify parents to count subdirectories
	for path := range groups {
		parts, isAbs := pathutil.Split(path)
		if len(parts) > 1 {
			for i := len(parts) - 1; i > 0; i-- {
				p := pathutil.Join(parts[:i], isAbs)
				if _, ok := groups[p]; ok {
					groups[p].FolderCount++
				}
			}
		}
		// Special case for root (isAbs with no components, or drive letter root)
		if isAbs {
			root := pathutil.Join(nil, true)
			if path != root {
				if _, ok := groups[root]; ok {
					groups[root].FolderCount++
				}
			}
		}
	}

	var result []models.FolderStats
	for _, f := range groups {
		if f.ExistsCount > 0 {
			f.AvgSize = f.TotalSize / int64(f.ExistsCount)
			f.AvgDuration = f.TotalDuration / int64(f.ExistsCount)

			// Skip expensive median calculation and file storage for fast mode
			if calculateMedians {
				sizes := make([]int64, 0, f.ExistsCount)
				durations := make([]int64, 0, f.ExistsCount)
				for _, m := range f.Files {
					isDeleted := m.TimeDeleted != nil && *m.TimeDeleted > 0
					if !isDeleted {
						if m.Size != nil {
							sizes = append(sizes, *m.Size)
						}
						if m.Duration != nil {
							durations = append(durations, *m.Duration)
						}
					}
				}
				f.MedianSize = int64(utils.SafeMedian(sizes))
				f.MedianDuration = int64(utils.SafeMedian(durations))
			}

			// Clear files slice if not needed (saves memory)
			if !shouldStoreFiles {
				f.Files = nil
			}
		}
		result = append(result, *f)
	}
	return result
}

// AggregateDUByPathMultiDBWithFilters performs SQL-level aggregation with filter support
func AggregateDUByPathMultiDBWithFilters(
	ctx context.Context,
	dbPaths []string,
	pathPrefix string,
	targetDepth int,
	flags models.GlobalFlags,
) ([]DUQueryResult, error) {
	allResults := make([]DUQueryResult, 0)

	for _, dbPath := range dbPaths {
		results, err := AggregateDUByPathWithFilters(ctx, dbPath, pathPrefix, targetDepth, flags)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, results...)
	}

	// Aggregate results from multiple databases by path
	pathMap := make(map[string]*DUQueryResult)
	for _, r := range allResults {
		if _, ok := pathMap[r.Path]; !ok {
			pathMap[r.Path] = &DUQueryResult{Path: r.Path}
		}
		pathMap[r.Path].Count += r.Count
		pathMap[r.Path].TotalSize += r.TotalSize
		pathMap[r.Path].TotalDuration += r.TotalDuration
	}

	// Convert map to slice
	finalResults := make([]DUQueryResult, 0, len(pathMap))
	for _, r := range pathMap {
		finalResults = append(finalResults, *r)
	}

	return finalResults, nil
}

// getMatchingParentDirs queries the database to find parent directories that match the given filters
// Returns a map of parent directory paths to their file counts
func getMatchingParentDirs(
	ctx context.Context,
	dbPath, pathPrefix string,
	targetDepth int,
	flags models.GlobalFlags,
) (map[string]int64, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	// Build query to extract parent directory at target depth and count files
	var query string
	var args []any

	if pathPrefix == "" {
		// Root level: extract first path component
		query = `
			SELECT
				CASE
					WHEN substr(replace(path, '\\', '/'), 1, 1) = '/' THEN
						substr(replace(path, '\\', '/'), 1,
							CASE
								WHEN instr(substr(replace(path, '\\', '/'), 2), '/') > 0
								THEN instr(substr(replace(path, '\\', '/'), 2), '/')
								ELSE length(replace(path, '\\', '/'))
							END
						)
					ELSE
						CASE
							WHEN instr(replace(path, '\\', '/'), '/') > 0
							THEN substr(replace(path, '\\', '/'), 1, instr(replace(path, '\\', '/'), '/') - 1)
							ELSE replace(path, '\\', '/')
						END
				END as parent_dir,
				COUNT(*) as file_count
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
		`
	} else {
		normalizedPrefix := strings.ReplaceAll(pathPrefix, "\\", "/")
		escapedPrefix := strings.ReplaceAll(normalizedPrefix, "'", "''")

		query = `
			SELECT
				CASE
					WHEN substr(?, 1, 1) = '/' THEN
						? || substr(
							substr(replace(path, '\\', '/'), length(?) + 1),
							1,
							CASE
								WHEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') > 0
								THEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/')
								ELSE length(substr(replace(path, '\\', '/'), length(?) + 1))
							END
						)
					ELSE
						? || CASE
							WHEN instr(substr(replace(path, '\\', '/'), length(?) + 1), '/') > 0
							THEN substr(substr(replace(path, '\\', '/'), length(?) + 1), 1, instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') - 1)
							ELSE substr(replace(path, '\\', '/'), length(?) + 1)
						END
				END as parent_dir,
				COUNT(*) as file_count
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
				AND replace(path, '\\', '/') LIKE ? || '/%'
				AND (length(replace(path, '\\', '/')) - length(replace(replace(path, '\\', '/'), '/', ''))) > ?
		`
		args = append(args,
			normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
			normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
			normalizedPrefix, escapedPrefix, targetDepth,
		)
	}

	// Add basic filters (size, duration, media_type, etc.) - but NOT FileCounts/FolderCounts
	basicFlags := flags
	basicFlags.FileCounts = ""
	basicFlags.FolderCounts = ""
	basicFlags.FolderSizes = nil

	fb := NewFilterBuilder(basicFlags)
	filterClauses, filterArgs := fb.BuildWhereClauses()

	if len(filterClauses) > 0 {
		query += " AND " + strings.Join(filterClauses, " AND ")
		args = append(args, filterArgs...)
	}

	query += " GROUP BY parent_dir"

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		models.Log.Error("Parent directory query failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	parentCounts := make(map[string]int64)
	for rows.Next() {
		var parentDir string
		var count int64
		if err := rows.Scan(&parentDir, &count); err != nil {
			return nil, err
		}
		parentCounts[parentDir] = count
	}

	return parentCounts, rows.Err()
}

// filterParentsByCounts filters parent directories based on FileCounts/FolderCounts/FolderSizes
func filterParentsByCounts(parentCounts map[string]int64, flags models.GlobalFlags) map[string]struct{} {
	matchingParents := make(map[string]struct{})

	for parent, count := range parentCounts {
		keep := true

		// Apply FileCounts filter (episode count per folder)
		if flags.FileCounts != "" {
			if r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			}); err == nil {
				if !r.Matches(count) {
					keep = false
				}
			}
		}

		// Apply FolderCounts filter
		if keep && flags.FolderCounts != "" {
			// FolderCounts applies to folder count within a parent, not file count
			// For now, we'll use the same logic - this may need adjustment based on exact semantics
			if _, err := utils.ParseRange(flags.FolderCounts, func(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			}); err != nil {
				// Invalid range filter, skip
			}
			// We don't have folder count here, so skip this filter at this level
			// It will be applied post-aggregation
		}

		// Apply FolderSizes filter
		if keep && len(flags.FolderSizes) > 0 {
			// FolderSizes applies to total size, which we don't have at this stage
			// Will be applied post-aggregation
		}

		if keep {
			matchingParents[parent] = struct{}{}
		}
	}

	return matchingParents
}

// AggregateDUByPathWithFilters performs SQL-level aggregation with filter support
// Uses two-path approach:
// - Fast path: no FileCounts/FolderCounts/FolderSizes filters - direct SQL aggregation
// - Slow path: folder backfiltering for FileCounts/FolderCounts/FolderSizes
func AggregateDUByPathWithFilters(
	ctx context.Context,
	dbPath, pathPrefix string,
	targetDepth int,
	flags models.GlobalFlags,
) ([]DUQueryResult, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	// Check if we need folder backfiltering (slow path)
	needsBackfiltering := hasActiveFilters(flags)

	if !needsBackfiltering {
		// FAST PATH: Direct SQL aggregation with basic filters
		return aggregateDUWithBasicFilters(ctx, sqlDB, pathPrefix, targetDepth, flags)
	}

	// SLOW PATH: Folder backfiltering
	// Phase 1: Get parent directories with file counts, applying basic filters
	parentCounts, err := getMatchingParentDirs(ctx, dbPath, pathPrefix, targetDepth, flags)
	if err != nil {
		return nil, err
	}

	// Phase 2: Filter parents by FileCounts/FolderCounts/FolderSizes
	matchingParents := filterParentsByCounts(parentCounts, flags)

	if len(matchingParents) == 0 {
		// No matching parents, return empty result
		return []DUQueryResult{}, nil
	}

	// Phase 3: Aggregate only matching parent directories
	return aggregateDUWithParentFilter(ctx, sqlDB, pathPrefix, targetDepth, flags, matchingParents)
}

// aggregateDUWithBasicFilters performs SQL aggregation with only basic filters (fast path)
func aggregateDUWithBasicFilters(
	ctx context.Context,
	sqlDB *sql.DB,
	pathPrefix string,
	targetDepth int,
	flags models.GlobalFlags,
) ([]DUQueryResult, error) {
	var query string
	var args []any

	if pathPrefix == "" {
		query = `
			SELECT
				CASE
					WHEN substr(replace(path, '\\', '/'), 1, 1) = '/' THEN
						substr(replace(path, '\\', '/'), 1,
							CASE
								WHEN instr(substr(replace(path, '\\', '/'), 2), '/') > 0
								THEN instr(substr(replace(path, '\\', '/'), 2), '/')
								ELSE length(replace(path, '\\', '/'))
							END
						)
					ELSE
						CASE
							WHEN instr(replace(path, '\\', '/'), '/') > 0
							THEN substr(replace(path, '\\', '/'), 1, instr(replace(path, '\\', '/'), '/') - 1)
							ELSE replace(path, '\\', '/')
						END
				END as agg_path,
				COUNT(*) as count,
				COALESCE(SUM(size), 0) as total_size,
				COALESCE(SUM(duration), 0) as total_duration
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
		`
	} else {
		normalizedPrefix := strings.ReplaceAll(pathPrefix, "\\", "/")
		escapedPrefix := strings.ReplaceAll(normalizedPrefix, "'", "''")

		query = `
			SELECT
				CASE
					WHEN substr(?, 1, 1) = '/' THEN
						? || substr(
							substr(replace(path, '\\', '/'), length(?) + 1),
							1,
							CASE
								WHEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') > 0
								THEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/')
								ELSE length(substr(replace(path, '\\', '/'), length(?) + 1))
							END
						)
					ELSE
						? || CASE
							WHEN instr(substr(replace(path, '\\', '/'), length(?) + 1), '/') > 0
							THEN substr(substr(replace(path, '\\', '/'), length(?) + 1), 1, instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') - 1)
							ELSE substr(replace(path, '\\', '/'), length(?) + 1)
						END
				END as agg_path,
				COUNT(*) as count,
				COALESCE(SUM(size), 0) as total_size,
				COALESCE(SUM(duration), 0) as total_duration
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
				AND replace(path, '\\', '/') LIKE ? || '/%'
				AND (length(replace(path, '\\', '/')) - length(replace(replace(path, '\\', '/'), '/', ''))) > ?
		`
		args = append(args,
			normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
			normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
			normalizedPrefix, escapedPrefix, targetDepth,
		)
	}

	// Add only basic filters (skip FileCounts, FolderCounts, FolderSizes which require post-aggregation)
	basicFlags := flags
	basicFlags.FileCounts = ""
	basicFlags.FolderCounts = ""
	basicFlags.FolderSizes = nil

	if hasBasicFilters(basicFlags) {
		fb := NewFilterBuilder(basicFlags)
		filterClauses, filterArgs := fb.BuildWhereClauses()
		if len(filterClauses) > 0 {
			query += " AND " + strings.Join(filterClauses, " AND ")
			args = append(args, filterArgs...)
		}
	}

	query += " GROUP BY agg_path ORDER BY total_size DESC"

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		models.Log.Error("DU aggregation query failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	var results []DUQueryResult
	for rows.Next() {
		var r DUQueryResult
		if err := rows.Scan(&r.Path, &r.Count, &r.TotalSize, &r.TotalDuration); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	// Apply post-aggregation filters (FolderCounts, FolderSizes)
	results = applyPostAggregationFilters(results, flags)

	return results, rows.Err()
}

// aggregateDUWithParentFilter performs SQL aggregation filtered by matching parent directories (slow path)
func aggregateDUWithParentFilter(
	ctx context.Context,
	sqlDB *sql.DB,
	pathPrefix string,
	targetDepth int,
	flags models.GlobalFlags,
	matchingParents map[string]struct{},
) ([]DUQueryResult, error) {
	var query string
	var args []any

	// Convert matching parents map to IN clause
	parentList := make([]string, 0, len(matchingParents))
	for parent := range matchingParents {
		parentList = append(parentList, parent)
	}

	// Sort for consistent ordering
	slices.Sort(parentList)

	if pathPrefix == "" {
		query = `
			SELECT
				CASE
					WHEN substr(replace(path, '\\', '/'), 1, 1) = '/' THEN
						substr(replace(path, '\\', '/'), 1,
							CASE
								WHEN instr(substr(replace(path, '\\', '/'), 2), '/') > 0
								THEN instr(substr(replace(path, '\\', '/'), 2), '/')
								ELSE length(replace(path, '\\', '/'))
							END
						)
					ELSE
						CASE
							WHEN instr(replace(path, '\\', '/'), '/') > 0
							THEN substr(replace(path, '\\', '/'), 1, instr(replace(path, '\\', '/'), '/') - 1)
							ELSE replace(path, '\\', '/')
						END
				END as agg_path,
				COUNT(*) as count,
				COALESCE(SUM(size), 0) as total_size,
				COALESCE(SUM(duration), 0) as total_duration
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
		`

		// Add parent filter
		if len(parentList) > 0 {
			placeholders := make([]string, len(parentList))
			for i, p := range parentList {
				placeholders[i] = "?"
				args = append(args, p)
			}
			query += fmt.Sprintf(
				" AND CASE WHEN substr(replace(path, '\\\\', '/'), 1, 1) = '/' THEN substr(replace(path, '\\\\', '/'), 1, CASE WHEN instr(substr(replace(path, '\\\\', '/'), 2), '/') > 0 THEN instr(substr(replace(path, '\\\\', '/'), 2), '/') ELSE length(replace(path, '\\\\', '/')) END) ELSE CASE WHEN instr(replace(path, '\\\\', '/'), '/') > 0 THEN substr(replace(path, '\\\\', '/'), 1, instr(replace(path, '\\\\', '/'), '/') - 1) ELSE replace(path, '\\\\', '/') END END IN (%s)",
				strings.Join(placeholders, ", "),
			)
		}
	} else {
		normalizedPrefix := strings.ReplaceAll(pathPrefix, "\\", "/")
		escapedPrefix := strings.ReplaceAll(normalizedPrefix, "'", "''")

		query = `
			SELECT
				CASE
					WHEN substr(?, 1, 1) = '/' THEN
						? || substr(
							substr(replace(path, '\\', '/'), length(?) + 1),
							1,
							CASE
								WHEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') > 0
								THEN instr(substr(replace(path, '\\', '/'), length(?) + 2), '/')
								ELSE length(substr(replace(path, '\\', '/'), length(?) + 1))
							END
						)
					ELSE
						? || CASE
							WHEN instr(substr(replace(path, '\\', '/'), length(?) + 1), '/') > 0
							THEN substr(substr(replace(path, '\\', '/'), length(?) + 1), 1, instr(substr(replace(path, '\\', '/'), length(?) + 2), '/') - 1)
							ELSE substr(replace(path, '\\', '/'), length(?) + 1)
						END
				END as agg_path,
				COUNT(*) as count,
				COALESCE(SUM(size), 0) as total_size,
				COALESCE(SUM(duration), 0) as total_duration
			FROM media
			WHERE COALESCE(time_deleted, 0) = 0
				AND replace(path, '\\', '/') LIKE ? || '/%'
				AND (length(replace(path, '\\', '/')) - length(replace(replace(path, '\\', '/'), '/', ''))) > ?
		`
		args = append(args,
			normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
			normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
			normalizedPrefix, escapedPrefix, targetDepth,
		)

		// Add parent filter
		if len(parentList) > 0 {
			placeholders := make([]string, len(parentList))
			for i, p := range parentList {
				placeholders[i] = "?"
				args = append(args, p)
			}
			// Reuse the same CASE expression for extracting parent path
			query += fmt.Sprintf(
				" AND CASE WHEN substr(?, 1, 1) = '/' THEN ? || substr(substr(replace(path, '\\\\', '/'), length(?) + 1), 1, CASE WHEN instr(substr(replace(path, '\\\\', '/'), length(?) + 2), '/') > 0 THEN instr(substr(replace(path, '\\\\', '/'), length(?) + 2), '/') ELSE length(substr(replace(path, '\\\\', '/'), length(?) + 1)) END) ELSE ? || CASE WHEN instr(substr(replace(path, '\\\\', '/'), length(?) + 1), '/') > 0 THEN substr(substr(replace(path, '\\\\', '/'), length(?) + 1), 1, instr(substr(replace(path, '\\\\', '/'), length(?) + 2), '/') - 1) ELSE substr(replace(path, '\\\\', '/'), length(?) + 1) END END IN (%s)",
				strings.Join(placeholders, ", "),
			)
			// Add the same args again for the CASE expression (11 total)
			args = append(args,
				normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
				normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix, normalizedPrefix,
				normalizedPrefix,
			)
		}
	}

	// Add basic filters
	basicFlags := flags
	basicFlags.FileCounts = ""
	basicFlags.FolderCounts = ""
	basicFlags.FolderSizes = nil

	if hasBasicFilters(basicFlags) {
		fb := NewFilterBuilder(basicFlags)
		filterClauses, filterArgs := fb.BuildWhereClauses()
		if len(filterClauses) > 0 {
			query += " AND " + strings.Join(filterClauses, " AND ")
			args = append(args, filterArgs...)
		}
	}

	query += " GROUP BY agg_path ORDER BY total_size DESC"

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		models.Log.Error("DU aggregation with parent filter failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	var results []DUQueryResult
	for rows.Next() {
		var r DUQueryResult
		if err := rows.Scan(&r.Path, &r.Count, &r.TotalSize, &r.TotalDuration); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	// Apply post-aggregation filters (FolderCounts, FolderSizes)
	results = applyPostAggregationFilters(results, flags)

	return results, rows.Err()
}

// applyPostAggregationFilters applies filters that require aggregated data (FolderCounts, FolderSizes)
func applyPostAggregationFilters(results []DUQueryResult, flags models.GlobalFlags) []DUQueryResult {
	if flags.FolderCounts == "" && len(flags.FolderSizes) == 0 {
		return results
	}

	var filtered []DUQueryResult
	for _, r := range results {
		keep := true

		// Apply FolderCounts filter
		if keep && flags.FolderCounts != "" {
			if rCount, err := utils.ParseRange(flags.FolderCounts, func(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			}); err == nil {
				if !rCount.Matches(int64(r.Count)) {
					keep = false
				}
			}
		}

		// Apply FolderSizes filter
		if keep && len(flags.FolderSizes) > 0 {
			for _, fs := range flags.FolderSizes {
				if rSize, err := utils.ParseRange(fs, utils.HumanToBytes); err == nil {
					if !rSize.Matches(r.TotalSize) {
						keep = false
						break
					}
				}
			}
		}

		if keep {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

// FetchDUDirectFilesWithFilters fetches files at the target depth with filter support
// Uses two-path approach:
// - Fast path: no filters - direct SQL query
// - Filter path: applies SQL-level filters
func FetchDUDirectFilesWithFilters(
	ctx context.Context,
	dbPaths []string,
	pathPrefix string,
	targetDepth int,
	flags models.GlobalFlags,
) ([]models.MediaWithDB, error) {
	allFiles := make([]models.MediaWithDB, 0)

	// Check if any filters are active
	hasFilters := hasBasicFilters(flags) || hasActiveFilters(flags)

	for _, dbPath := range dbPaths {
		sqlDB, err := db.Connect(dbPath)
		if err != nil {
			return nil, err
		}

		var query string
		var args []any

		if pathPrefix == "" {
			query = `
				SELECT path, title, duration, size, time_created, time_modified,
				       time_deleted, time_first_played, time_last_played, play_count,
				       playhead, album, artist, genre, categories, description,
				       language, time_downloaded, score, video_codecs, audio_codecs,
				       subtitle_codecs, width, height, media_type
				FROM media
				WHERE COALESCE(time_deleted, 0) = 0
				  AND instr(replace(path, '\\', '/'), '/') = 0
			`
		} else {
			normalizedPrefix := strings.ReplaceAll(pathPrefix, "\\", "/")
			escapedPrefix := strings.ReplaceAll(normalizedPrefix, "'", "''")

			isAbs := len(normalizedPrefix) > 0 && normalizedPrefix[0] == '/'
			separatorCount := targetDepth
			if !isAbs {
				separatorCount = targetDepth - 1
			}

			query = `
				SELECT path, title, duration, size, time_created, time_modified,
				       time_deleted, time_first_played, time_last_played, play_count,
				       playhead, album, artist, genre, categories, description,
				       language, time_downloaded, score, video_codecs, audio_codecs,
				       subtitle_codecs, width, height, media_type
				FROM media
				WHERE COALESCE(time_deleted, 0) = 0
				  AND replace(path, '\\', '/') LIKE ? || '/%'
				  AND (
					length(replace(path, '\\', '/')) - length(replace(replace(path, '\\', '/'), '/', ''))
				  ) = ?
			`
			args = append(args, escapedPrefix, separatorCount)
		}

		// Only add filter clauses if filters are active
		if hasFilters {
			fb := NewFilterBuilder(flags)
			filterClauses, filterArgs := fb.BuildWhereClauses()

			if len(filterClauses) > 0 {
				query += " AND " + strings.Join(filterClauses, " AND ")
				args = append(args, filterArgs...)
			}
		}

		rows, err := sqlDB.QueryContext(ctx, query, args...)
		if err != nil {
			sqlDB.Close()
			return nil, err
		}

		files, err := ScanMedia(rows, dbPath)
		rows.Close()
		sqlDB.Close()
		if err != nil {
			return nil, err
		}

		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}
