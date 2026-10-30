package query

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// MediaQuery executes a query against multiple databases concurrently
func MediaQuery(ctx context.Context, dbs []string, flags models.GlobalFlags) ([]models.MediaWithDB, error) {
	executor := NewQueryExecutor(flags)
	return executor.MediaQuery(ctx, dbs)
}

// MediaQueryCount executes a count query against multiple databases concurrently
func MediaQueryCount(ctx context.Context, dbs []string, flags models.GlobalFlags) (int64, error) {
	executor := NewQueryExecutor(flags)
	return executor.MediaQueryCount(ctx, dbs)
}

// FilterMedia applies all filters to media list in memory
func FilterMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	fb := NewFilterBuilder(flags)
	return fb.FilterMedia(media)
}

// SortMedia sorts media using the unified SortBuilder and applies --re-rank.
func SortMedia(media []models.MediaWithDB, flags models.GlobalFlags) {
	NewSortBuilder(flags).Sort(media)
}

// SortMediaWithExpansion sorts media with optional result expansion (siblings, related)
// This should be used when sort config includes expansion markers
func SortMediaWithExpansion(ctx context.Context, sqlDB *sql.DB, media *[]models.MediaWithDB, flags models.GlobalFlags) {
	// Check if sort config contains expansion markers
	sortConfig := flags.PlayInOrder
	if sortConfig == "" && flags.SortBy != "" {
		sortConfig = flags.SortBy
	}

	// Expand related media if requested
	if strings.Contains(sortConfig, "_related_media") {
		if err := ExpandRelatedMedia(ctx, sqlDB, media, flags); err != nil {
			models.Log.Warn("Related media expansion failed", "error", err)
		}
	}

	// Sort with the full config (including expansion markers)
	NewSortBuilder(flags).SortAdvanced(*media, sortConfig)
}

// ApplyReRank applies --re-rank to an already sorted slice.
func ApplyReRank(media []models.MediaWithDB, flags models.GlobalFlags) {
	applyReRank(media, flags)
}

func applyReRank(media []models.MediaWithDB, flags models.GlobalFlags) {
	if flags.ReRank == "" {
		return
	}
	reranked := ReRankMedia(media, flags)
	copy(media, reranked)
}

// FetchSiblings fetches sibling files for the given media (Re-exported for tests)
func FetchSiblings(
	ctx context.Context,
	media []models.MediaWithDB,
	flags models.GlobalFlags,
) ([]models.MediaWithDB, error) {
	executor := NewQueryExecutor(flags)
	return executor.FetchSiblings(ctx, media, flags)
}

// ResolvePercentileFlags resolves percentile-based filters (Re-exported for tests)
func ResolvePercentileFlags(ctx context.Context, dbs []string, flags models.GlobalFlags) (models.GlobalFlags, error) {
	executor := NewQueryExecutor(flags)
	return executor.ResolvePercentileFlags(ctx, dbs, flags)
}

// ReRankMedia implements MCDA-like re-ranking
func ReRankMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	if flags.ReRank == "" {
		return media
	}

	weights := make(map[string]float64)
	for p := range strings.FieldsSeq(flags.ReRank) {
		kv := strings.Split(p, "=")
		weight := 1.0
		if len(kv) == 2 {
			if w, err := strconv.ParseFloat(kv[1], 64); err == nil {
				weight = w
			}
		}
		weights[kv[0]] = weight
	}

	if len(weights) == 0 {
		return media
	}

	type rankedItem struct {
		media models.MediaWithDB
		score float64
	}

	n := len(media)
	items := make([]rankedItem, n)
	for i := range media {
		items[i].media = media[i]
	}

	for col, weight := range weights {
		direction := 1.0
		cleanCol := col
		if strings.HasPrefix(col, "-") {
			direction = -1.0
			cleanCol = col[1:]
		}

		sort.SliceStable(items, func(i, j int) bool {
			valI := getMediaValueFloat(items[i].media, cleanCol)
			valJ := getMediaValueFloat(items[j].media, cleanCol)
			if direction > 0 {
				return valI < valJ
			}
			return valI > valJ
		})

		for i := range n {
			items[i].score += float64(i) * weight
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].score < items[j].score
	})

	result := make([]models.MediaWithDB, n)
	for i := range items {
		result[i] = items[i].media
	}
	return result
}

func getMediaValueFloat(m models.MediaWithDB, col string) float64 {
	switch col {
	case "size":
		return float64(utils.Int64Value(m.Size))
	case "duration":
		return float64(utils.Int64Value(m.Duration))
	case "play_count":
		return float64(utils.Int64Value(m.PlayCount))
	case "time_last_played":
		return float64(utils.Int64Value(m.TimeLastPlayed))
	case "time_created":
		return float64(utils.Int64Value(m.TimeCreated))
	case "time_modified":
		return float64(utils.Int64Value(m.TimeModified))
	case "playhead":
		return float64(utils.Int64Value(m.Playhead))
	case "bitrate":
		d := utils.Int64Value(m.Duration)
		if d == 0 {
			return 0
		}
		return float64(utils.Int64Value(m.Size)) / float64(d)
	default:
		return 0
	}
}

// SortHistory applies specialized sorting for playback history. It returns a
// new slice when the partial filter drops rows; callers must use the result.
func SortHistory(media []models.MediaWithDB, partial string, reverse bool) []models.MediaWithDB {
	if strings.Contains(partial, "s") {
		filtered := make([]models.MediaWithDB, 0, len(media))
		for _, m := range media {
			if m.TimeFirstPlayed == nil || *m.TimeFirstPlayed == 0 {
				filtered = append(filtered, m)
			}
		}
		media = filtered
	}

	mpvProgress := func(m models.MediaWithDB) float64 {
		playhead := utils.Int64Value(m.Playhead)
		duration := utils.Int64Value(m.Duration)
		if playhead <= 0 || duration <= 0 {
			return -math.MaxFloat64
		}

		if strings.Contains(partial, "p") && strings.Contains(partial, "t") {
			return (float64(duration) / float64(playhead)) * -float64(duration-playhead)
		} else if strings.Contains(partial, "t") {
			return -float64(duration - playhead)
		} else {
			return float64(playhead) / float64(duration)
		}
	}

	less := func(i, j int) bool {
		var valI, valJ float64

		if strings.Contains(partial, "f") {
			valI = float64(utils.Int64Value(media[i].TimeFirstPlayed))
			valJ = float64(utils.Int64Value(media[j].TimeFirstPlayed))
		} else if strings.Contains(partial, "p") || strings.Contains(partial, "t") {
			valI = mpvProgress(media[i])
			valJ = mpvProgress(media[j])
		} else {
			valI = float64(utils.Int64Value(media[i].TimeLastPlayed))
			if valI == 0 {
				valI = float64(utils.Int64Value(media[i].TimeFirstPlayed))
			}
			valJ = float64(utils.Int64Value(media[j].TimeLastPlayed))
			if valJ == 0 {
				valJ = float64(utils.Int64Value(media[j].TimeFirstPlayed))
			}
		}

		if reverse {
			return valI > valJ
		}
		return valI < valJ
	}

	sort.Slice(media, less)
	return media
}

// RegexSortMedia sorts media using the text processor
func RegexSortMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	if len(media) == 0 {
		return media
	}

	sentenceStrings := make([]string, len(media))
	mapping := make(map[string][]models.MediaWithDB)

	for i, m := range media {
		parts := []string{m.Path}
		if m.Title != nil {
			parts = append(parts, *m.Title)
		}
		sentence := utils.PathToSentence(strings.Join(parts, " "))
		sentenceStrings[i] = sentence
		mapping[sentence] = append(mapping[sentence], m)
	}

	sortedSentences := utils.TextProcessor(flags, sentenceStrings)

	result := make([]models.MediaWithDB, 0, len(media))
	seenCount := make(map[string]int)
	for _, s := range sortedSentences {
		idx := seenCount[s]
		if idx < len(mapping[s]) {
			result = append(result, mapping[s][idx])
			seenCount[s]++
		}
	}

	return result
}

// folderLess reports whether a sorts before b for the given sort key.
func folderLess(a, b models.FolderStats, sortBy string) bool {
	switch sortBy {
	case "count":
		return a.Count < b.Count
	case "size":
		return a.TotalSize < b.TotalSize
	case "duration":
		return a.TotalDuration < b.TotalDuration
	case "priority", "size_per_count", "size_per_file":
		return lessDiskUsage(a, b)
	case "folders", "folder_count":
		if a.FolderCount != b.FolderCount {
			return a.FolderCount < b.FolderCount
		}
		return a.Path < b.Path
	case "path":
		return a.Path < b.Path
	default:
		// Default to DU-style sorting: (size/count), size, count, folders, reverse(path)
		return lessDiskUsage(a, b)
	}
}

// lessDiskUsage orders folders by average file size, then size, count,
// folder count, and finally reverse path.
func lessDiskUsage(a, b models.FolderStats) bool {
	aPerFile := float64(a.TotalSize) / float64(utils.Max(1, a.Count))
	bPerFile := float64(b.TotalSize) / float64(utils.Max(1, b.Count))
	if aPerFile != bPerFile {
		return aPerFile < bPerFile
	}
	if a.TotalSize != b.TotalSize {
		return a.TotalSize < b.TotalSize
	}
	if a.Count != b.Count {
		return a.Count < b.Count
	}
	if a.FolderCount != b.FolderCount {
		return a.FolderCount < b.FolderCount
	}
	return a.Path > b.Path // reverse path
}

// SortFolders sorts folder stats
func SortFolders(folders []models.FolderStats, sortBy string, reverse bool) {
	less := func(i, j int) bool {
		return folderLess(folders[i], folders[j], sortBy)
	}

	if reverse {
		sort.SliceStable(folders, func(i, j int) bool { return less(j, i) })
		return
	}
	sort.SliceStable(folders, less)
}

type FrequencyStats struct {
	Label         string `json:"label"`
	Count         int    `json:"count"`
	TotalSize     int64  `json:"total_size"`
	TotalDuration int64  `json:"total_duration"`
}

func SummarizeMedia(media []models.MediaWithDB) []FrequencyStats {
	if len(media) == 0 {
		return nil
	}

	sizes := make([]int64, 0, len(media))
	durations := make([]int64, 0, len(media))

	for _, m := range media {
		if m.Size != nil {
			sizes = append(sizes, *m.Size)
		}
		if m.Duration != nil {
			durations = append(durations, *m.Duration)
		}
	}

	return []FrequencyStats{
		{
			Label:         "Total",
			Count:         len(media),
			TotalSize:     utils.SafeSum(sizes),
			TotalDuration: utils.SafeSum(durations),
		},
		{
			Label:         "Median",
			Count:         len(media),
			TotalSize:     int64(utils.SafeMedian(sizes)),
			TotalDuration: int64(utils.SafeMedian(durations)),
		},
	}
}

func HistoricalUsage(ctx context.Context, dbPath, freq, timeColumn string) ([]FrequencyStats, error) {
	return HistoricalUsageFiltered(ctx, dbPath, freq, timeColumn, models.GlobalFlags{})
}

// HistoricalUsageFiltered is HistoricalUsage with the standard filter set
// applied before the time buckets are computed.
func HistoricalUsageFiltered(
	ctx context.Context,
	dbPath, freq, timeColumn string,
	flags models.GlobalFlags,
) ([]FrequencyStats, error) {
	validTimeColumns := map[string]bool{
		"time_created":      true,
		"time_modified":     true,
		"time_deleted":      true,
		"time_first_played": true,
		"time_last_played":  true,
		"time_downloaded":   true,
	}
	if !validTimeColumns[timeColumn] {
		return nil, fmt.Errorf("invalid time facet: %s", timeColumn)
	}

	sqlDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	var freqSQL string
	switch freq {
	case "daily":
		freqSQL = fmt.Sprintf("strftime('%%Y-%%m-%%d', datetime(%s, 'unixepoch'))", timeColumn)
	case "weekly":
		freqSQL = fmt.Sprintf("strftime('%%Y-%%W', datetime(%s, 'unixepoch'))", timeColumn)
	case "monthly":
		freqSQL = fmt.Sprintf("strftime('%%Y-%%m', datetime(%s, 'unixepoch'))", timeColumn)
	case "quarterly":
		freqSQL = fmt.Sprintf(
			"strftime('%%Y', datetime(%s, 'unixepoch')) || '-Q' || ((CAST(strftime('%%m', datetime(%s, 'unixepoch')) AS INTEGER) - 1) / 3 + 1)",
			timeColumn,
			timeColumn,
		)
	case "yearly":
		freqSQL = fmt.Sprintf("strftime('%%Y', datetime(%s, 'unixepoch'))", timeColumn)
	case "decadally":
		freqSQL = fmt.Sprintf("(CAST(strftime('%%Y', datetime(%s, 'unixepoch')) AS INTEGER) / 10) * 10", timeColumn)
	case "hourly":
		freqSQL = fmt.Sprintf("strftime('%%Y-%%m-%%d %%Hh', datetime(%s, 'unixepoch'))", timeColumn)
	case "minutely":
		freqSQL = fmt.Sprintf("strftime('%%Y-%%m-%%d %%H:%%M', datetime(%s, 'unixepoch'))", timeColumn)
	default:
		return nil, fmt.Errorf("invalid frequency: %s", freq)
	}

	whereClauses, args := NewFilterBuilder(flags).BuildWhereClauses(ctx)
	whereClauses = append(whereClauses, fmt.Sprintf("%s > 0", timeColumn))
	if timeColumn != "time_deleted" && !flags.HideDeleted && !flags.OnlyDeleted {
		whereClauses = append(whereClauses, "COALESCE(time_deleted, 0) = 0")
	}

	query := fmt.Sprintf(`
		SELECT
			%s AS label,
			COUNT(*) AS count,
			SUM(size) AS total_size,
			SUM(duration) AS total_duration
		FROM media
		WHERE %s
		GROUP BY label
		ORDER BY label DESC
	`, freqSQL, strings.Join(whereClauses, " AND "))

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []FrequencyStats
	for rows.Next() {
		var s FrequencyStats
		var totalSize, totalDuration sql.NullInt64
		if err := rows.Scan(&s.Label, &s.Count, &totalSize, &totalDuration); err != nil {
			return nil, err
		}
		s.TotalSize = totalSize.Int64
		s.TotalDuration = totalDuration.Int64
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

// GetStatsFiltered returns the general statistics for the rows matched by the
// provided filters.
func GetStatsFiltered(ctx context.Context, dbPath string, flags models.GlobalFlags) (db.GetStatsRow, error) {
	sqlDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return db.GetStatsRow{}, err
	}
	defer sqlDB.Close()

	predicate, args := statsPredicate(ctx, flags)
	query := `SELECT COUNT(*) as total_count, SUM(size) as total_size, SUM(duration) as total_duration, ` +
		`COUNT(CASE WHEN COALESCE(time_last_played, 0) > 0 THEN 1 END) as watched_count, ` +
		`COUNT(CASE WHEN COALESCE(time_last_played, 0) = 0 THEN 1 END) as unwatched_count, ` +
		`SUM(COALESCE(play_count, 0) * COALESCE(duration, 0) + COALESCE(playhead, 0)) as total_watched_duration ` +
		`FROM media` + predicate

	var i db.GetStatsRow
	err = sqlDB.QueryRowContext(ctx, query, args...).Scan(
		&i.TotalCount,
		&i.TotalSize,
		&i.TotalDuration,
		&i.WatchedCount,
		&i.UnwatchedCount,
		&i.TotalWatchedDuration,
	)
	return i, err
}

// GetStatsByTypeFiltered returns statistics grouped by media type for the rows
// matched by the provided filters.
func GetStatsByTypeFiltered(
	ctx context.Context,
	dbPath string,
	flags models.GlobalFlags,
) ([]db.GetStatsByTypeRow, error) {
	sqlDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	predicate, args := statsPredicate(ctx, flags)
	query := `SELECT media_type, COUNT(*) as count, SUM(size) as total_size, SUM(duration) as total_duration ` +
		`FROM media` + predicate + ` GROUP BY media_type`

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []db.GetStatsByTypeRow
	for rows.Next() {
		var i db.GetStatsByTypeRow
		if err := rows.Scan(&i.MediaType, &i.Count, &i.TotalSize, &i.TotalDuration); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func statsPredicate(ctx context.Context, flags models.GlobalFlags) (string, []any) {
	whereClauses, args := NewFilterBuilder(flags).BuildWhereClauses(ctx)
	if len(whereClauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(whereClauses, " AND "), args
}
