package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// FilterBuilder constructs SQL queries and in-memory filters from flags
// This is the single source of truth for all filter logic
type FilterBuilder struct {
	Flags models.GlobalFlags
}

// NewFilterBuilder creates a new FilterBuilder from global flags
func NewFilterBuilder(flags models.GlobalFlags) *FilterBuilder {
	return &FilterBuilder{Flags: flags}
}

// BuildWhereClauses builds WHERE clauses and arguments for SQL queries
// Order matters for performance: selective indexed filters come before expensive substring searches
// Based on benchmark: indexed equality (~277μs) << LIKE prefix (~500μs) << LIKE substring (~1ms)
func (fb *FilterBuilder) BuildWhereClauses(ctx context.Context) ([]string, []any) {
	var whereClauses []string
	var args []any

	// Selective indexed equality filters
	fb.buildBasicFilters(&whereClauses, &args)
	fb.buildRangeFilters(&whereClauses, &args)
	fb.buildTimeFilters(&whereClauses, &args)
	fb.buildStatusFilters(&whereClauses)

	// Substring and more expensive searches
	fb.buildSearchFilters(ctx, &whereClauses, &args)

	// Miscellaneous filters
	fb.buildMiscellaneousFilters(&whereClauses, &args)

	return whereClauses, args
}

func (fb *FilterBuilder) buildBasicFilters(whereClauses *[]string, args *[]any) {
	// Deleted status (indexed column)
	if fb.Flags.OnlyDeleted {
		*whereClauses = append(*whereClauses, fmt.Sprintf("COALESCE(%s, 0) > 0", fb.col("time_deleted")))
	} else if fb.Flags.HideDeleted {
		*whereClauses = append(*whereClauses, fmt.Sprintf("COALESCE(%s, 0) = 0", fb.col("time_deleted")))
	}

	// Content type filters (indexed column - should come before expensive searches)
	var typeClauses []string
	if fb.Flags.VideoOnly {
		typeClauses = append(typeClauses, fmt.Sprintf("%s = 'video'", fb.col("media_type")))
	}
	if fb.Flags.AudioOnly {
		typeClauses = append(
			typeClauses,
			fmt.Sprintf("%s = 'audio'", fb.col("media_type")),
			fmt.Sprintf("%s = 'audiobook'", fb.col("media_type")),
		)
	}
	if fb.Flags.ImageOnly {
		typeClauses = append(typeClauses, fmt.Sprintf("%s = 'image'", fb.col("media_type")))
	}
	if fb.Flags.TextOnly {
		typeClauses = append(typeClauses, fmt.Sprintf("%s = 'text'", fb.col("media_type")))
	}
	if len(typeClauses) > 0 {
		*whereClauses = append(*whereClauses, "("+strings.Join(typeClauses, " OR ")+")")
	}

	// Genre filter (equality match)
	if fb.Flags.Genre != "" {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s = ?", fb.col("genre")))
		*args = append(*args, fb.Flags.Genre)
	}

	// Language filter (equality match)
	if len(fb.Flags.Language) > 0 {
		langClauses := make([]string, 0, len(fb.Flags.Language))
		for _, lang := range fb.Flags.Language {
			langClauses = append(langClauses, fmt.Sprintf("%s = ?", fb.col("language")))
			*args = append(*args, lang)
		}
		if len(langClauses) > 0 {
			*whereClauses = append(*whereClauses, "("+strings.Join(langClauses, " OR ")+")")
		}
	}

	// Exact path filters (IN clause - very selective)
	if len(fb.Flags.Paths) > 0 {
		var inPaths []string
		for _, p := range fb.Flags.Paths {
			if strings.Contains(p, "%") {
				*whereClauses = append(*whereClauses, fmt.Sprintf("%s LIKE ?", fb.col("path")))
				*args = append(*args, p)
			} else {
				inPaths = append(inPaths, p)
			}
		}
		if len(inPaths) > 0 {
			placeholders := make([]string, len(inPaths))
			for i := range inPaths {
				placeholders[i] = "?"
				*args = append(*args, inPaths[i])
			}
			*whereClauses = append(
				*whereClauses,
				fmt.Sprintf("%s IN (%s)", fb.col("path"), strings.Join(placeholders, ", ")),
			)
		}
	}

	// Play count filters (indexed column)
	if fb.Flags.PlayCountMin > 0 {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("play_count")))
		*args = append(*args, fb.Flags.PlayCountMin)
	}
	if fb.Flags.PlayCountMax > 0 {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("play_count")))
		*args = append(*args, fb.Flags.PlayCountMax)
	}
}

func (fb *FilterBuilder) buildRangeFilters(whereClauses *[]string, args *[]any) {
	// Size filters (indexed column)
	for _, s := range fb.Flags.Size {
		if r, err := utils.ParseRange(s, utils.HumanToBytes); err == nil {
			if r.Value != nil {
				*whereClauses = append(*whereClauses, fmt.Sprintf("%s = ?", fb.col("size")))
				*args = append(*args, *r.Value)
			}
			if r.Min != nil {
				*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("size")))
				*args = append(*args, *r.Min)
			}
			if r.Max != nil {
				*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("size")))
				*args = append(*args, *r.Max)
			}
		}
	}

	// Duration filters (indexed column)
	for _, s := range fb.Flags.Duration {
		if r, err := utils.ParseRange(s, utils.HumanToSeconds); err == nil {
			if r.Value != nil {
				*whereClauses = append(*whereClauses, fmt.Sprintf("%s = ?", fb.col("duration")))
				*args = append(*args, *r.Value)
			}
			if r.Min != nil {
				*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("duration")))
				*args = append(*args, *r.Min)
			}
			if r.Max != nil {
				*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("duration")))
				*args = append(*args, *r.Max)
			}
		}
	}
}

func (fb *FilterBuilder) buildTimeFilters(whereClauses *[]string, args *[]any) {
	if fb.Flags.DeletedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.DeletedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_deleted")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.DeletedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.DeletedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_deleted")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.CreatedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.CreatedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_created")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.CreatedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.CreatedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_created")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.ModifiedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.ModifiedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_modified")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.ModifiedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.ModifiedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_modified")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.DownloadedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.DownloadedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_downloaded")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.DownloadedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.DownloadedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_downloaded")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.PlayedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.PlayedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_last_played")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.PlayedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.Flags.PlayedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_last_played")))
			*args = append(*args, ts)
		}
	}
}

func (fb *FilterBuilder) buildStatusFilters(whereClauses *[]string) {
	// Watched/unwatched status (indexed via time_last_played)
	if fb.Flags.Watched != nil {
		if *fb.Flags.Watched {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s > 0", fb.col("time_last_played")))
		} else {
			*whereClauses = append(*whereClauses, fmt.Sprintf("COALESCE(%s, 0) = 0", fb.col("time_last_played")))
		}
	}

	// Playhead/playback status
	if fb.Flags.Unfinished || fb.Flags.InProgress {
		*whereClauses = append(*whereClauses, fmt.Sprintf("COALESCE(%s, 0) > 0", fb.col("playhead")))
	}
	if fb.Flags.Partial != "" {
		if strings.Contains(fb.Flags.Partial, "s") {
			*whereClauses = append(*whereClauses, fmt.Sprintf("COALESCE(%s, 0) = 0", fb.col("time_first_played")))
		} else {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s > 0", fb.col("time_first_played")))
		}
	}
	if fb.Flags.Completed {
		*whereClauses = append(*whereClauses, fmt.Sprintf("COALESCE(%s, 0) > 0", fb.col("play_count")))
	}
}

func (fb *FilterBuilder) buildSearchFilters(ctx context.Context, whereClauses *[]string, args *[]any) {
	if fb.Flags.OnlineMediaOnly {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s LIKE 'http%%'", fb.col("path")))
	}
	if fb.Flags.LocalMediaOnly {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s NOT LIKE 'http%%'", fb.col("path")))
	}

	// Extension filters (EndsWith pattern - surprisingly fast ~660μs per benchmark)
	if len(fb.Flags.Ext) > 0 {
		extClauses := make([]string, 0, len(fb.Flags.Ext))
		for _, ext := range fb.Flags.Ext {
			extClauses = append(extClauses, fmt.Sprintf("%s LIKE ?", fb.col("path")))
			*args = append(*args, "%"+ext)
		}
		*whereClauses = append(*whereClauses, "("+strings.Join(extClauses, " OR ")+")")
	}

	// Category filter (LIKE with wildcards - expensive)
	if len(fb.Flags.Category) > 0 {
		var catClauses []string
		for _, cat := range fb.Flags.Category {
			if cat == "Uncategorized" {
				catClauses = append(
					catClauses,
					fmt.Sprintf("(%s IS NULL OR %s = '')", fb.col("categories"), fb.col("categories")),
				)
			} else {
				catClauses = append(catClauses, fmt.Sprintf("%s LIKE '%%' || ? || '%%'", fb.col("categories")))
				*args = append(*args, ";"+cat+";")
			}
		}
		if len(catClauses) > 0 {
			*whereClauses = append(*whereClauses, "("+strings.Join(catClauses, " OR ")+")")
		}
	}

	// Search terms (FTS or LIKE - most expensive operation)
	allInclude := append([]string{}, fb.Flags.Search...)
	allInclude = append(allInclude, fb.Flags.Include...)

	// Path contains filters
	pathContains := append([]string{}, fb.Flags.PathContains...)

	var filteredInclude []string
	for _, term := range allInclude {
		if strings.HasPrefix(term, "./") || strings.HasPrefix(term, ".\\") {
			pathContains = append(pathContains, term[2:])
		} else if strings.HasPrefix(term, "/") || strings.HasPrefix(term, "\\") {
			pathContains = append(pathContains, term)
		} else {
			filteredInclude = append(filteredInclude, term)
		}
	}
	allInclude = filteredInclude

	if len(allInclude) > 0 {
		joinOp := " AND "
		if fb.Flags.FlexibleSearch {
			joinOp = " OR "
		}

		// Determine search mode: --no-fts > --fts > auto-detect
		useFTS := fb.Flags.FTS
		noFTS := fb.Flags.NoFTS

		// Auto-detect if not explicitly set
		if !useFTS && !noFTS {
			mode := DetectSearchMode(ctx, nil)
			useFTS = (mode == SearchModeFTS5)
		}

		if noFTS {
			// Force substring search
			useFTS = false
		}

		if useFTS && !fb.Flags.Exact {
			// Hybrid FTS + LIKE search for phrase support with detail=none
			// Note: FTS with detail=none doesn't support exact matching, so we use LIKE for --exact
			// Combine all terms into a single query string for parsing
			queryStr := strings.Join(allInclude, " ")
			hybrid := utils.ParseHybridSearchQuery(queryStr)

			// FTS terms (works with detail=none)
			if hybrid.HasFTSTerms() {
				// Use trigram matching for fuzzy search
				ftsQuery := hybrid.BuildFTSQuery(joinOp)
				if ftsQuery != "" {
					*whereClauses = append(*whereClauses, fmt.Sprintf("%s MATCH ?", fb.getFTSTable()))
					*args = append(*args, ftsQuery)
				}
			}

			// Phrase searches via LIKE (trigram-optimized)
			for _, phrase := range hybrid.Phrases {
				*whereClauses = append(
					*whereClauses,
					fmt.Sprintf(
						"(%s LIKE ? OR %s LIKE ? OR %s LIKE ?)",
						fb.col("path"),
						fb.col("title"),
						fb.col("description"),
					),
				)
				pattern := "%" + phrase + "%"
				*args = append(*args, pattern, pattern, pattern)
			}
		} else {
			// Regular LIKE search (also used for --exact mode since FTS detail=none doesn't support exact)
			var searchParts []string
			for _, term := range allInclude {
				if fb.Flags.Exact {
					// For exact match, use raw path column with word boundary matching
					// Match basename containing the exact term followed by separator or extension
					// This ensures "exact" matches "exact.mp4" but not "exact_match.mp4"
					searchParts = append(searchParts, fmt.Sprintf(
						"(%s LIKE ? ESCAPE '\\' OR %s LIKE ? ESCAPE '\\')",
						fb.col("path"), fb.col("path"),
					))
					// Match: "%/exact.%" or "%/exact" (basename boundaries)
					// The % before matches any path prefix, then we match exact word boundary
					*args = append(*args,
						"%/"+term+".%", // path contains "/exact." (exact followed by extension)
						"%/"+term,      // path ends with "/exact" (exact as basename)
					)
				} else {
					searchParts = append(
						searchParts,
						fmt.Sprintf(
							"(%s LIKE ? OR %s LIKE ? OR %s LIKE ?)",
							fb.col("path"),
							fb.col("title"),
							fb.col("path_tokenized"),
						),
					)
					pattern := "%" + strings.ReplaceAll(term, " ", "%") + "%"
					*args = append(*args, pattern, pattern, pattern)
				}
			}
			*whereClauses = append(*whereClauses, "("+strings.Join(searchParts, joinOp)+")")
		}
	}

	// Exclude patterns (expensive NOT LIKE)
	for _, exc := range fb.Flags.Exclude {
		*whereClauses = append(
			*whereClauses,
			fmt.Sprintf("%s NOT LIKE ? AND %s NOT LIKE ?", fb.col("path"), fb.col("title")),
		)
		pattern := "%" + exc + "%"
		*args = append(*args, pattern, pattern)
	}

	// Regex filter (requires regex extension or post-filter)
	if fb.Flags.Regex != "" {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s REGEXP ?", fb.col("path")))
		*args = append(*args, fb.Flags.Regex)
	}

	// Path contains filters (substring search - expensive)
	for _, contain := range pathContains {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s LIKE ?", fb.col("path")))
		*args = append(*args, "%"+contain+"%")
	}
}

func (fb *FilterBuilder) buildMiscellaneousFilters(whereClauses *[]string, args *[]any) {
	if fb.Flags.Portrait {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s < %s", fb.col("width"), fb.col("height")))
	}

	if fb.Flags.WithCaptions {
		*whereClauses = append(
			*whereClauses,
			fmt.Sprintf("%s IN (SELECT DISTINCT media_path FROM captions)", fb.col("path")),
		)
	}

	// Custom WHERE clauses
	*whereClauses = append(*whereClauses, fb.Flags.Where...)

	if fb.Flags.DurationFromSize != "" {
		if r, err := utils.ParseRange(fb.Flags.DurationFromSize, utils.HumanToBytes); err == nil {
			var subWhere []string
			var subArgs []any
			if r.Value != nil {
				subWhere = append(subWhere, fmt.Sprintf("%s = ?", fb.col("size")))
				subArgs = append(subArgs, *r.Value)
			}
			if r.Min != nil {
				subWhere = append(subWhere, fmt.Sprintf("%s >= ?", fb.col("size")))
				subArgs = append(subArgs, *r.Min)
			}
			if r.Max != nil {
				subWhere = append(subWhere, fmt.Sprintf("%s <= ?", fb.col("size")))
				subArgs = append(subArgs, *r.Max)
			}

			if len(subWhere) > 0 {
				*whereClauses = append(
					*whereClauses,
					fmt.Sprintf(
						"%s IS NOT NULL AND %s IN (SELECT DISTINCT %s FROM media WHERE %s)",
						fb.col("size"),
						fb.col("duration"),
						fb.col("duration"),
						strings.Join(subWhere, " AND "),
					),
				)
				*args = append(*args, subArgs...)
			}
		}
	}
}

// BuildQuery constructs a complete SQL query with the given columns
func (fb *FilterBuilder) BuildQuery(ctx context.Context, columns string) (string, []any) {
	// If raw query provided, use it
	if fb.Flags.Query != "" {
		if columns == "COUNT(*)" {
			return "SELECT COUNT(*) FROM (" + fb.Flags.Query + ")", nil
		}
		return fb.Flags.Query, nil
	}

	whereClauses, args := fb.BuildWhereClauses(ctx)

	// Base table
	table := "media"
	useFTSJoin := fb.Flags.FTS && fb.hasSearchTerms()

	if useFTSJoin {
		table = fmt.Sprintf("media JOIN %s ON media.rowid = %s.rowid", fb.getFTSTable(), fb.getFTSTable())
		if columns == "*" {
			columns = "media.*"
		}
	}

	query := fmt.Sprintf("SELECT %s FROM %s", columns, table)

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if columns == "COUNT(*)" {
		return query, args
	}

	// Order by
	if !fb.Flags.Random && !fb.Flags.NatSort && fb.Flags.SortBy != "" {
		sortExpr := fb.OverrideSort(fb.Flags.SortBy)
		order := "ASC"
		if fb.Flags.Reverse {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", sortExpr, order)
	} else if fb.Flags.Random {
		// Optimization for large databases: select rowids randomly first
		if !fb.Flags.All && !fb.Flags.FTS && !fb.hasSearchTerms() && fb.Flags.Limit > 0 {
			whereNotDeleted := "WHERE COALESCE(time_deleted, 0) = 0"
			if fb.Flags.OnlyDeleted {
				whereNotDeleted = "WHERE COALESCE(time_deleted, 0) > 0"
			}
			// We use a larger pool for random selection then limit it in the outer query
			randomLimit := fb.Flags.Limit * 16

			randomSubquery := fmt.Sprintf(
				"rowid IN (SELECT rowid FROM media %s ORDER BY RANDOM() LIMIT %d)",
				whereNotDeleted,
				randomLimit,
			)
			if strings.Contains(query, " WHERE ") {
				query += " AND " + randomSubquery
			} else {
				query += " WHERE " + randomSubquery
			}
		}
		query += " ORDER BY RANDOM()"
	}

	// Limit and offset
	if !fb.Flags.All && fb.Flags.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", fb.Flags.Limit)
	}
	if fb.Flags.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", fb.Flags.Offset)
	}

	return query, args
}

// OverrideSort translates logical sort fields into SQL expressions
func (fb *FilterBuilder) OverrideSort(s string) string {
	yearMonthSQL := func(v string) string {
		return fmt.Sprintf("cast(strftime('%%Y%%m', datetime(%s, 'unixepoch')) as int)", fb.col(v))
	}
	yearMonthDaySQL := func(v string) string {
		return fmt.Sprintf("cast(strftime('%%Y%%m%%d', datetime(%s, 'unixepoch')) as int)", fb.col(v))
	}

	s = strings.ReplaceAll(s, "month_created", yearMonthSQL("time_created"))
	s = strings.ReplaceAll(s, "month_modified", yearMonthSQL("time_modified"))
	s = strings.ReplaceAll(s, "date_created", yearMonthDaySQL("time_created"))
	s = strings.ReplaceAll(s, "date_modified", yearMonthDaySQL("time_modified"))
	s = strings.ReplaceAll(s, "time_deleted", fmt.Sprintf("COALESCE(%s, 0)", fb.col("time_deleted")))

	progressExpr := fmt.Sprintf(
		"CAST(COALESCE(%s, 0) AS FLOAT) / CAST(COALESCE(%s, 1) AS FLOAT)",
		fb.col("playhead"),
		fb.col("duration"),
	)
	s = strings.ReplaceAll(s, "progress", fmt.Sprintf("(%s = 0), %s", progressExpr, progressExpr))

	s = strings.ReplaceAll(
		s,
		"play_count",
		fmt.Sprintf("(COALESCE(%s, 0) = 0), %s", fb.col("play_count"), fb.col("play_count")),
	)
	s = strings.ReplaceAll(
		s,
		"time_last_played",
		fmt.Sprintf("(COALESCE(%s, 0) = 0), %s", fb.col("time_last_played"), fb.col("time_last_played")),
	)

	s = strings.ReplaceAll(s, "media_type", fmt.Sprintf("LOWER(%s)", fb.col("media_type")))
	s = strings.ReplaceAll(s, "random()", "RANDOM()")
	s = strings.ReplaceAll(s, "random", "RANDOM()")
	s = strings.ReplaceAll(
		s,
		"default",
		fmt.Sprintf(
			"%s, %s DESC, %s, %s DESC, %s DESC, %s IS NOT NULL DESC, %s",
			fb.col("play_count"),
			fb.col("playhead"),
			fb.col("time_last_played"),
			fb.col("duration"),
			fb.col("size"),
			fb.col("title"),
			fb.col("path"),
		),
	)
	s = strings.ReplaceAll(
		s,
		"priorityfast",
		fmt.Sprintf("ntile(1000) over (order by %s) desc, %s", fb.col("size"), fb.col("duration")),
	)
	s = strings.ReplaceAll(
		s,
		"priority",
		fmt.Sprintf("ntile(1000) over (order by %s/%s) desc", fb.col("size"), fb.col("duration")),
	)
	s = strings.ReplaceAll(s, "bitrate", fmt.Sprintf("%s/%s", fb.col("size"), fb.col("duration")))
	s = strings.ReplaceAll(s, "time_scanned", fmt.Sprintf("COALESCE(%s, 0)", fb.col("time_downloaded")))
	s = strings.ReplaceAll(s, "time_downloaded", fmt.Sprintf("COALESCE(%s, 0)", fb.col("time_downloaded")))

	return s
}

// BuildSelect is an alias for BuildQuery for backward compatibility
func (fb *FilterBuilder) BuildSelect(ctx context.Context, columns string) (string, []any) {
	return fb.BuildQuery(ctx, columns)
}

// BuildCount builds a count query
func (fb *FilterBuilder) BuildCount(ctx context.Context) (string, []any) {
	return fb.BuildQuery(ctx, "COUNT(*)")
}

// hasSearchTerms checks if there are any search/include terms
func (fb *FilterBuilder) hasSearchTerms() bool {
	allInclude := append([]string{}, fb.Flags.Search...)
	allInclude = append(allInclude, fb.Flags.Include...)
	for _, term := range allInclude {
		if strings.HasPrefix(term, "./") || strings.HasPrefix(term, "/") {
			continue
		}
		return true
	}
	return false
}

// getFTSTable returns the FTS table name
func (fb *FilterBuilder) getFTSTable() string {
	if fb.Flags.FTSTable != "" {
		return fb.Flags.FTSTable
	}
	return "media_fts"
}

// usesFTSJoin returns true if the query will join media with FTS table
func (fb *FilterBuilder) usesFTSJoin() bool {
	return fb.Flags.FTS && fb.hasSearchTerms()
}

// col qualifies a column name with media. prefix if using FTS join
func (fb *FilterBuilder) col(name string) string {
	if fb.usesFTSJoin() {
		return "media." + name
	}
	return name
}

func (fb *FilterBuilder) parseSizeRanges() []utils.Range {
	var sizeRanges []utils.Range
	for _, s := range fb.Flags.Size {
		if r, err := utils.ParseRange(s, utils.HumanToBytes); err == nil {
			sizeRanges = append(sizeRanges, r)
		}
	}
	return sizeRanges
}

func (fb *FilterBuilder) parseDurationRanges() []utils.Range {
	var durationRanges []utils.Range
	for _, s := range fb.Flags.Duration {
		if r, err := utils.ParseRange(s, utils.HumanToSeconds); err == nil {
			durationRanges = append(durationRanges, r)
		}
	}
	return durationRanges
}

func (fb *FilterBuilder) matchesRangeFilters(m models.MediaWithDB, sizeRanges, durationRanges []utils.Range) bool {
	// Size filters
	for _, r := range sizeRanges {
		if m.Size == nil || !r.Matches(*m.Size) {
			return false
		}
	}

	// Duration filters
	for _, r := range durationRanges {
		if m.Duration == nil || !r.Matches(*m.Duration) {
			return false
		}
	}
	return true
}

func (fb *FilterBuilder) matchesPathFilters(m models.MediaWithDB, regex *regexp.Regexp) bool {
	// Check existence
	if fb.Flags.Exists && !utils.FileExists(m.Path) {
		return false
	}

	// Include/exclude patterns
	if len(fb.Flags.Include) > 0 && !utils.MatchesAny(m.Path, fb.Flags.Include) {
		return false
	}
	if len(fb.Flags.Exclude) > 0 && utils.MatchesAny(m.Path, fb.Flags.Exclude) {
		return false
	}

	// Path contains
	for _, contain := range fb.Flags.PathContains {
		if !strings.Contains(m.Path, contain) {
			return false
		}
	}

	// Regex filter
	if regex != nil && !regex.MatchString(m.Path) {
		return false
	}

	// Extension filters
	if len(fb.Flags.Ext) > 0 {
		fileExt := strings.ToLower(filepath.Ext(m.Path))
		matched := false
		for _, ext := range fb.Flags.Ext {
			if fileExt == strings.ToLower(ext) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// CreateInMemoryFilter creates a function that can filter media in memory
func (fb *FilterBuilder) CreateInMemoryFilter() func(models.MediaWithDB) bool {
	// Pre-compile regex if needed
	var regex *regexp.Regexp
	if fb.Flags.Regex != "" {
		regex = regexp.MustCompile(fb.Flags.Regex)
	}

	sizeRanges := fb.parseSizeRanges()
	durationRanges := fb.parseDurationRanges()

	return func(m models.MediaWithDB) bool {
		if !fb.matchesPathFilters(m, regex) {
			return false
		}

		if !fb.matchesRangeFilters(m, sizeRanges, durationRanges) {
			return false
		}

		return true
	}
}

// FilterMedia applies in-memory filtering to a slice of media
func (fb *FilterBuilder) FilterMedia(media []models.MediaWithDB) []models.MediaWithDB {
	filter := fb.CreateInMemoryFilter()
	filtered := make([]models.MediaWithDB, 0, len(media))
	for _, m := range media {
		if filter(m) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// SortBuilder handles both SQL and in-memory sorting
type SortBuilder struct {
	Flags models.GlobalFlags
}

func NewSortBuilder(flags models.GlobalFlags) *SortBuilder {
	return &SortBuilder{Flags: flags}
}

func (sb *SortBuilder) Sort(media []models.MediaWithDB) {
	if sb.Flags.Random {
		rand.Shuffle(len(media), func(i, j int) {
			media[i], media[j] = media[j], media[i]
		})
		return
	}

	if sb.Flags.NoPlayInOrder {
		sb.SortBasic(media)
		return
	}

	// If the user explicitly requested a specific sort field (and it's not the default "path" or "default"),
	// respect it and use basic sorting - this takes precedence over PlayInOrder
	if sb.Flags.SortBy != "" && sb.Flags.SortBy != "path" && sb.Flags.SortBy != "default" {
		sb.SortBasic(media)
		return
	}

	// If the user explicitly requested "default", use xklb sorting (with optional reverse)
	if sb.Flags.SortBy == "default" {
		if sb.Flags.Reverse {
			sb.SortAdvanced(media, "reverse_xklb")
		} else {
			sb.SortAdvanced(media, "xklb")
		}
		return
	}

	// If PlayInOrder is explicitly set (and SortBy is default), use it
	if sb.Flags.PlayInOrder != "" {
		if sb.Flags.Reverse {
			// Prepend "reverse_" to the PlayInOrder config
			sb.SortAdvanced(media, "reverse_"+sb.Flags.PlayInOrder)
		} else {
			sb.SortAdvanced(media, sb.Flags.PlayInOrder)
		}
		return
	}

	// If SortBy is "path" (the default) with Reverse or NatSort flags, use basic sorting
	if sb.Flags.Reverse || sb.Flags.NatSort {
		sb.SortBasic(media)
		return
	}

	// Fall back to xklb default sorting when SortBy is "path" (the default)
	// This provides xklb-style sorting as the default behavior
	if sb.Flags.SortBy == "path" || sb.Flags.SortBy == "" {
		sb.SortAdvanced(media, "xklb")
		return
	}

	sb.SortBasic(media)
}

func (sb *SortBuilder) sortBasicNumeric(media []models.MediaWithDB, sortBy string, reverse bool) {
	sort.SliceStable(media, func(i, j int) bool {
		var vI, vJ float64

		switch sortBy {
		case "play_count":
			vI = float64(utils.Int64Value(media[i].PlayCount))
			vJ = float64(utils.Int64Value(media[j].PlayCount))
		case "time_last_played":
			vI = float64(utils.Int64Value(media[i].TimeLastPlayed))
			vJ = float64(utils.Int64Value(media[j].TimeLastPlayed))
		case "progress":
			dI := float64(utils.Int64Value(media[i].Duration))
			if dI > 0 {
				vI = float64(utils.Int64Value(media[i].Playhead)) / dI
			}
			dJ := float64(utils.Int64Value(media[j].Duration))
			if dJ > 0 {
				vJ = float64(utils.Int64Value(media[j].Playhead)) / dJ
			}
		}

		// Zero check: zeros always last (greater index)
		if vI == 0 && vJ != 0 {
			return false
		}
		if vI != 0 && vJ == 0 {
			return true
		}
		if vI == 0 && vJ == 0 {
			return false
		}

		if reverse {
			return vI > vJ
		}
		return vI < vJ
	})
}

func (sb *SortBuilder) getBasicLessFunc(
	media []models.MediaWithDB,
	sortBy string,
	reverse, natSort bool,
) func(i, j int) bool {
	return func(i, j int) bool {
		switch sortBy {
		case "path":
			if natSort {
				return utils.NaturalLess(media[i].Path, media[j].Path)
			}
			return media[i].Path < media[j].Path
		case "title":
			return utils.StringValue(media[i].Title) < utils.StringValue(media[j].Title)
		case "duration":
			return sb.lessDuration(media[i], media[j], reverse)
		case "size":
			return utils.Int64Value(media[i].Size) < utils.Int64Value(media[j].Size)
		case "bitrate", "priority":
			return sb.lessBitrate(media[i], media[j])
		case "priorityfast":
			return sb.lessPriorityFast(media[i], media[j])
		case "time_created", "date_created", "month_created":
			return utils.Int64Value(media[i].TimeCreated) < utils.Int64Value(media[j].TimeCreated)
		case "time_modified", "date_modified", "month_modified":
			return utils.Int64Value(media[i].TimeModified) < utils.Int64Value(media[j].TimeModified)
		case "time_last_played":
			return utils.Int64Value(media[i].TimeLastPlayed) < utils.Int64Value(media[j].TimeLastPlayed)
		case "play_count":
			return utils.Int64Value(media[i].PlayCount) < utils.Int64Value(media[j].PlayCount)
		case "time_deleted":
			return utils.Int64Value(media[i].TimeDeleted) < utils.Int64Value(media[j].TimeDeleted)
		case "time_downloaded", "time_scanned":
			return utils.Int64Value(media[i].TimeDownloaded) < utils.Int64Value(media[j].TimeDownloaded)
		case "media_type":
			return sb.lessMediaType(media[i], media[j], reverse)
		case "extension":
			return strings.ToLower(filepath.Ext(media[i].Path)) < strings.ToLower(filepath.Ext(media[j].Path))
		default:
			return utils.NaturalLess(media[i].Path, media[j].Path)
		}
	}
}

func (sb *SortBuilder) SortBasic(media []models.MediaWithDB) {
	sortBy := sb.Flags.SortBy
	reverse := sb.Flags.Reverse

	if sortBy == "play_count" || sortBy == "time_last_played" || sortBy == "progress" {
		sb.sortBasicNumeric(media, sortBy, reverse)
		return
	}

	less := sb.getBasicLessFunc(media, sortBy, reverse, sb.Flags.NatSort)
	if reverse {
		sort.SliceStable(media, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.SliceStable(media, less)
	}
}

func (sb *SortBuilder) lessDuration(m1, m2 models.MediaWithDB, reverse bool) bool {
	v1 := utils.Int64Value(m1.Duration)
	v2 := utils.Int64Value(m2.Duration)

	if v1 == 0 && v2 != 0 {
		return reverse
	}
	if v1 != 0 && v2 == 0 {
		return !reverse
	}
	if v1 == 0 && v2 == 0 {
		return false
	}

	return v1 < v2
}

func (sb *SortBuilder) lessBitrate(m1, m2 models.MediaWithDB) bool {
	size1 := float64(utils.Int64Value(m1.Size))
	dur1 := float64(utils.Int64Value(m1.Duration))
	size2 := float64(utils.Int64Value(m2.Size))
	dur2 := float64(utils.Int64Value(m2.Duration))

	var br1, br2 float64
	if dur1 > 0 {
		br1 = size1 / dur1
	}
	if dur2 > 0 {
		br2 = size2 / dur2
	}

	if br1 == 0 && br2 != 0 {
		return false
	}
	if br1 != 0 && br2 == 0 {
		return true
	}
	return br1 < br2
}

func (sb *SortBuilder) lessPriorityFast(m1, m2 models.MediaWithDB) bool {
	s1 := utils.Int64Value(m1.Size)
	s2 := utils.Int64Value(m2.Size)

	if s1 != s2 {
		return s1 > s2
	}

	return utils.Int64Value(m1.Duration) < utils.Int64Value(m2.Duration)
}

func (sb *SortBuilder) lessMediaType(m1, m2 models.MediaWithDB, reverse bool) bool {
	v1 := strings.ToLower(utils.StringValue(m1.MediaType))
	v2 := strings.ToLower(utils.StringValue(m2.MediaType))

	if v1 == "" && v2 != "" {
		return reverse
	}
	if v1 != "" && v2 == "" {
		return !reverse
	}
	if v1 == "" && v2 == "" {
		return false
	}

	return v1 < v2
}

// SortField represents a single field in a multi-field sort
type SortField struct {
	Field   string
	Reverse bool
}

// SortGroup represents a group of fields with a specific sorting algorithm
type SortGroup struct {
	Fields []SortField
	Alg    string // "weighted", "natural", "related", or empty for standard
}

// ParseSortConfig parses a sort configuration string into SortField slices
// Supports:
//   - Simple: "path", "title"
//   - Prefixed: "natural_path", "python_title"
//   - Reversed: "-path", "reverse_path"
//   - Multi-field: "video_count desc,audio_count desc,path asc"
//   - Complex: "natural_path,title desc"
//   - Array notation: "field1,field2,field3" (comma-separated)
//   - Meta-field markers: "_weighted_rerank", "_natural_order" (apply to fields below)
func ParseSortConfig(config string) ([]SortField, string) {
	if config == "" {
		return []SortField{{Field: "ps", Reverse: false}}, "natural"
	}

	// Check if this is a simple algorithm-only config (not field names)
	knownAlgs := map[string]bool{
		"natural": true, "ignorecase": true,
		"lowercase": true, "human": true, "locale": true,
		"signed": true, "os": true, "python": true,
	}

	// If it's a known algorithm without underscores, use default sort key
	if knownAlgs[config] {
		return []SortField{{Field: "ps", Reverse: false}}, config
	}

	var sortFields []SortField
	alg := "natural"

	// Split by comma for multi-field sorting
	parts := strings.SplitSeq(config, ",")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		reverse := false
		field := part

		// Check for "desc" / "asc" suffix
		if strings.HasSuffix(part, " desc") {
			reverse = true
			field = strings.TrimSpace(strings.TrimSuffix(part, " desc"))
		} else if before, ok := strings.CutSuffix(part, " asc"); ok {
			field = strings.TrimSpace(before)
		}

		// Check for "reverse_" prefix
		if after, ok := strings.CutPrefix(field, "reverse_"); ok {
			reverse = !reverse
			field = after
		}

		// Check for "-" prefix
		if strings.HasPrefix(field, "-") {
			reverse = !reverse
			field = field[1:]
		}

		// Check for algorithm prefix (alg_field)
		if strings.Contains(field, "_") {
			potentialParts := strings.SplitN(field, "_", 2)
			if knownAlgs[potentialParts[0]] {
				alg = potentialParts[0]
				field = potentialParts[1]
			}
		}

		sortFields = append(sortFields, SortField{Field: field, Reverse: reverse})
	}

	if len(sortFields) == 0 {
		return []SortField{{Field: "ps", Reverse: false}}, alg
	}

	return sortFields, alg
}

// ParseSortConfigWithGroups parses a sort configuration string into SortGroup slices
// Meta-field markers (_weighted_rerank, _natural_order) create new groups that apply to fields below them
func (sb *SortBuilder) handleMetaMarkers(part string, currentGroup *SortGroup, groups *[]SortGroup) bool {
	var alg string
	switch part {
	case "_weighted_rerank":
		alg = "weighted"
	case "_natural_order":
		alg = "natural"
	case "_related_media":
		alg = "related"
	default:
		return false
	}

	if len(currentGroup.Fields) > 0 {
		*groups = append(*groups, *currentGroup)
		*currentGroup = SortGroup{}
	}
	currentGroup.Alg = alg
	return true
}

func parseSortField(part string) SortField {
	reverse := false
	field := part

	if strings.HasSuffix(part, " desc") {
		reverse = true
		field = strings.TrimSpace(strings.TrimSuffix(part, " desc"))
	} else if before, ok := strings.CutSuffix(part, " asc"); ok {
		field = strings.TrimSpace(before)
	}

	if after, ok := strings.CutPrefix(field, "reverse_"); ok {
		reverse = !reverse
		field = after
	}

	if strings.HasPrefix(field, "-") {
		reverse = !reverse
		field = field[1:]
	}

	knownAlgs := map[string]bool{
		"natural": true, "ignorecase": true,
		"lowercase": true, "human": true, "locale": true,
		"signed": true, "os": true, "python": true,
	}

	if strings.Contains(field, "_") {
		potentialParts := strings.SplitN(field, "_", 2)
		if knownAlgs[potentialParts[0]] {
			field = potentialParts[1]
		}
	}

	return SortField{Field: field, Reverse: reverse}
}

func ParseSortConfigWithGroups(config string) []SortGroup {
	if config == "" {
		return []SortGroup{{Fields: []SortField{{Field: "ps", Reverse: false}}, Alg: "natural"}}
	}

	var groups []SortGroup
	var currentGroup SortGroup
	currentAlg := "natural"

	parts := strings.SplitSeq(config, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if sb := (&SortBuilder{}); sb.handleMetaMarkers(part, &currentGroup, &groups) {
			continue
		}

		field := parseSortField(part)
		if currentGroup.Alg == "" {
			currentGroup.Alg = currentAlg
		}
		currentGroup.Fields = append(currentGroup.Fields, field)
	}

	if len(currentGroup.Fields) > 0 {
		if currentGroup.Alg == "" {
			currentGroup.Alg = currentAlg
		}
		groups = append(groups, currentGroup)
	}

	if len(groups) == 0 {
		return []SortGroup{{Fields: []SortField{{Field: "ps", Reverse: false}}, Alg: "natural"}}
	}

	return groups
}

// GetSortValueFloat64 returns a numeric sort value for a field
func GetSortValueFloat64(m models.MediaWithDB, field string) float64 {
	switch field {
	case "video_count":
		return float64(utils.Int64Value(m.VideoCount))
	case "audio_count":
		return float64(utils.Int64Value(m.AudioCount))
	case "subtitle_count":
		return float64(utils.Int64Value(m.SubtitleCount))
	case "play_count":
		return float64(utils.Int64Value(m.PlayCount))
	case "playhead":
		return float64(utils.Int64Value(m.Playhead))
	case "time_last_played":
		return float64(utils.Int64Value(m.TimeLastPlayed))
	case "time_created":
		return float64(utils.Int64Value(m.TimeCreated))
	case "time_modified":
		return float64(utils.Int64Value(m.TimeModified))
	case "time_downloaded":
		return float64(utils.Int64Value(m.TimeDownloaded))
	case "time_deleted":
		return float64(utils.Int64Value(m.TimeDeleted))
	case "duration":
		return float64(utils.Int64Value(m.Duration))
	case "size":
		return float64(utils.Int64Value(m.Size))
	case "width":
		return float64(utils.Int64Value(m.Width))
	case "height":
		return float64(utils.Int64Value(m.Height))
	case "fps":
		return utils.Float64Value(m.Fps)
	case "score":
		return utils.Float64Value(m.Score)
	case "track_number":
		return float64(utils.Int64Value(m.TrackNumber))
	case "count":
		// For folder stats compatibility
		return float64(utils.Int64Value(m.Size))
	default:
		return 0
	}
}

// GetSortValueString returns a string sort value for a field
func GetSortValueString(m models.MediaWithDB, field string) string {
	switch field {
	case "path":
		return m.Path
	case "title":
		return utils.StringValue(m.Title)
	case "parent":
		return m.Parent()
	case "stem":
		return m.Stem()
	case "ps":
		return m.Parent() + " " + m.Stem()
	case "pts":
		return m.Parent() + " " + utils.StringValue(m.Title) + " " + m.Stem()
	case "media_type":
		return utils.StringValue(m.MediaType)
	case "genre":
		return utils.StringValue(m.Genre)
	case "artist":
		return utils.StringValue(m.Artist)
	case "album":
		return utils.StringValue(m.Album)
	case "language":
		return utils.StringValue(m.Language)
	case "categories":
		return utils.StringValue(m.Categories)
	case "video_codecs":
		return utils.StringValue(m.VideoCodecs)
	case "audio_codecs":
		return utils.StringValue(m.AudioCodecs)
	case "extension":
		return strings.ToLower(filepath.Ext(m.Path))
	default:
		return m.Path
	}
}

// IsNumericField checks if a field is numeric (float64) or string
func IsNumericField(field string) bool {
	numericFields := map[string]bool{
		"video_count": true, "audio_count": true, "subtitle_count": true,
		"play_count": true, "playhead": true, "time_last_played": true,
		"time_created": true, "time_modified": true, "time_downloaded": true,
		"time_deleted": true, "duration": true, "size": true,
		"width": true, "height": true, "fps": true, "score": true,
		"track_number": true, "count": true,
	}
	return numericFields[field]
}

// XklbDefaultSort returns the xklb-style default sort fields
// This matches the lambda sorting from xklb:
// - video_count desc (videos before audio-only)
// - audio_count desc (files with audio before silent ones)
// - path like "http%" asc (local files before remote URLs)
// - subtitle_count desc
// - play_count asc (unplayed/least-played first)
// - playhead desc (furthest along first)
// - time_last_played asc (least-recently played first)
// - title is not null desc (titled entries before untitled)
// - path asc (alphabetical tiebreak)
func XklbDefaultSort() []SortField {
	return []SortField{
		{Field: "video_count", Reverse: true},
		{Field: "audio_count", Reverse: true},
		{Field: "path_is_remote", Reverse: false}, // local before remote
		{Field: "subtitle_count", Reverse: true},
		{Field: "play_count", Reverse: false},
		{Field: "playhead", Reverse: true},
		{Field: "time_last_played", Reverse: false},
		{Field: "title_is_null", Reverse: false}, // titled before untitled
		{Field: "path", Reverse: false},
	}
}

// DuDefaultSort returns the default sort for DU mode
// lambda x: ((size/count), size, count, folders, reverse(path))
func DuDefaultSort() []SortField {
	return []SortField{
		{Field: "size_per_count", Reverse: true}, // size/count desc
		{Field: "size", Reverse: true},
		{Field: "count", Reverse: true},
		{Field: "folders", Reverse: true},
		{Field: "path", Reverse: true}, // reverse path
	}
}

func compareNumericField(i, j models.MediaWithDB, field string) int {
	var valI, valJ float64

	switch field {
	case "path_is_remote":
		if strings.HasPrefix(i.Path, "http") {
			valI = 1
		}
		if strings.HasPrefix(j.Path, "http") {
			valJ = 1
		}
	case "title_is_null":
		if i.Title == nil || *i.Title == "" {
			valI = 1
		}
		if j.Title == nil || *j.Title == "" {
			valJ = 1
		}
	case "size_per_count":
		sizeI := float64(utils.Int64Value(i.Size))
		sizeJ := float64(utils.Int64Value(j.Size))
		// For media items, just use size
		valI = sizeI
		valJ = sizeJ
	default:
		valI = GetSortValueFloat64(i, field)
		valJ = GetSortValueFloat64(j, field)
	}

	if valI < valJ {
		return -1
	} else if valI > valJ {
		return 1
	}
	return 0
}

func compareStringField(i, j models.MediaWithDB, field, alg string) int {
	valI := GetSortValueString(i, field)
	valJ := GetSortValueString(j, field)

	if valI == valJ {
		return 0
	}

	var less bool
	if alg == "python" {
		less = valI < valJ
	} else {
		less = utils.NaturalLess(valI, valJ)
	}

	if less {
		return -1
	}
	return 1
}

// CompareSortFields compares two media items using multiple sort fields
func CompareSortFields(media []models.MediaWithDB, i, j int, sortFields []SortField, alg string) int {
	for _, sf := range sortFields {
		var cmp int
		if IsNumericField(sf.Field) {
			cmp = compareNumericField(media[i], media[j], sf.Field)
		} else {
			cmp = compareStringField(media[i], media[j], sf.Field, alg)
		}

		if cmp != 0 {
			if sf.Reverse {
				return -cmp
			}
			return cmp
		}
	}
	return 0
}

func (sb *SortBuilder) SortAdvanced(media []models.MediaWithDB, config string) {
	if config == "" {
		sb.SortBasic(media)
		return
	}

	// Special keywords for preset sorts
	switch config {
	case "xklb", "xklb_default":
		config = "video_count desc,audio_count desc,path_is_remote asc,subtitle_count desc,play_count asc,playhead desc,time_last_played asc,title_is_null asc,path asc"
	case "reverse_xklb", "reverse_xklb_default":
		// Reverse of xklb: prioritize audio, remote URLs, no subtitles, most played, etc.
		config = "video_count asc,audio_count asc,path_is_remote desc,subtitle_count asc,play_count desc,playhead asc,time_last_played desc,title_is_null desc,path desc"
	case "du", "du_default":
		config = "size_per_count desc,size desc,count desc,folders desc,path desc"
	case "reverse_du", "reverse_du_default":
		// Reverse of du: smallest folders first, alphabetical path
		config = "size_per_count asc,size asc,count asc,folders asc,path asc"
	}

	// Check for meta-field markers and use grouped sorting
	groups := ParseSortConfigWithGroups(config)

	if len(groups) > 1 {
		// Multiple groups - apply each group's sorting in sequence
		for _, group := range groups {
			switch group.Alg {
			case "weighted":
				// Apply weighted re-ranking for this group
				ApplyWeightedRerank(media, group.Fields)
			case "natural":
				// Apply natural sorting as tiebreaker for this group
				ApplyNaturalSort(media, group.Fields)
			case "related":
				// Related media - apply standard natural sort
				// Note: Media expansion happens earlier in SortMediaWithExpansion
				applyStandardSort(media, group.Fields, "natural")
			default:
				// Standard multi-field sorting
				applyStandardSort(media, group.Fields, group.Alg)
			}
		}
		return
	}

	// Single group - use original logic
	sortFields, alg := ParseSortConfig(config)

	if len(sortFields) == 1 && sortFields[0].Field == "ps" {
		// Legacy single-field sorting
		reverse := sortFields[0].Reverse
		less := func(i, j int) bool {
			valI := GetSortValueString(media[i], "ps")
			valJ := GetSortValueString(media[j], "ps")

			var res bool
			if alg == "python" {
				res = valI < valJ
			} else {
				res = utils.NaturalLess(valI, valJ)
			}

			if reverse {
				return !res
			}
			return res
		}
		sort.Slice(media, less)
		return
	}

	// Multi-field sorting
	sort.SliceStable(media, func(i, j int) bool {
		cmp := CompareSortFields(media, i, j, sortFields, alg)
		return cmp < 0
	})
}

// applyStandardSort applies standard multi-field sorting to media
func applyStandardSort(media []models.MediaWithDB, fields []SortField, alg string) {
	if len(fields) == 0 {
		return
	}
	sort.SliceStable(media, func(i, j int) bool {
		cmp := CompareSortFields(media, i, j, fields, alg)
		return cmp < 0
	})
}

// ApplyNaturalSort applies natural sorting to media using the specified fields as tiebreakers
func ApplyNaturalSort(media []models.MediaWithDB, fields []SortField) {
	if len(fields) == 0 {
		return
	}
	// Use natural algorithm for string comparisons
	applyStandardSort(media, fields, "natural")
}

// ApplyWeightedRerank applies MCDA-style weighted re-ranking based on field positions
// Fields earlier in the list have higher weights (position-based weighting)
func ApplyWeightedRerank(media []models.MediaWithDB, fields []SortField) {
	if len(fields) == 0 || len(media) == 0 {
		return
	}

	// Calculate ranks for each field and combine with weights
	// Weight decreases with position: first field gets highest weight
	type rankedMedia struct {
		totalScore float64
	}

	ranked := make([]rankedMedia, len(media))

	// For each field, calculate ranks and apply weights
	for fieldIdx, field := range fields {
		// Weight based on position (higher weight for earlier fields)
		// Using inverse position weighting: weight = 1 / (position + 1)
		weight := 1.0 / float64(fieldIdx+1)

		// Create sortable slice with field values
		type fieldValue struct {
			originalIndex int
			value         float64
			strValue      string
			isNumeric     bool
		}

		values := make([]fieldValue, len(media))
		for i, m := range media {
			values[i] = fieldValue{originalIndex: i}
			if IsNumericField(field.Field) {
				values[i].value = GetSortValueFloat64(m, field.Field)
				values[i].isNumeric = true
			} else {
				values[i].strValue = GetSortValueString(m, field.Field)
				values[i].isNumeric = false
			}
		}

		// Sort values to determine ranks
		sort.Slice(values, func(i, j int) bool {
			if values[i].isNumeric {
				if field.Reverse {
					return values[i].value > values[j].value
				}
				return values[i].value < values[j].value
			}
			// Use natural comparison for strings
			if field.Reverse {
				return utils.NaturalLess(values[j].strValue, values[i].strValue)
			}
			return utils.NaturalLess(values[i].strValue, values[j].strValue)
		})

		// Assign ranks (normalized to 0-1 range)
		for rank, fv := range values {
			// Normalize rank to 0-1 (best = 1, worst = 0)
			normalizedRank := 1.0 - (float64(rank) / float64(len(values)-1))
			if len(values) == 1 {
				normalizedRank = 1.0
			}
			ranked[fv.originalIndex].totalScore += normalizedRank * weight
		}
	}

	// Sort media by total weighted score
	sort.SliceStable(media, func(i, j int) bool {
		return ranked[i].totalScore > ranked[j].totalScore
	})
}

// ExpandRelatedMedia expands the result set with media related to the first item
// based on shared search terms (title, path words) using FTS rank
func ExpandRelatedMedia(
	ctx context.Context,
	sqlDB *sql.DB,
	media *[]models.MediaWithDB,
	flags models.GlobalFlags,
) error {
	if len(*media) == 0 {
		return nil
	}

	// Get the first media item to find related content
	first := (*media)[0]

	// Extract search terms from flags first (if available)
	var words []string
	if len(flags.Search) > 0 {
		// Use the original search query terms
		queryStr := strings.Join(flags.Search, " ")
		hybrid := utils.ParseHybridSearchQuery(queryStr)

		// Use FTS terms from the search
		words = append(words, hybrid.FTSTerms...)

		// Also include phrases as individual words
		for _, phrase := range hybrid.Phrases {
			phraseWords := strings.FieldsSeq(phrase)
			for w := range phraseWords {
				if len(w) > 2 {
					words = append(words, w)
				}
			}
		}
	}

	// If no search terms from flags, extract from media item
	if len(words) == 0 {
		words = extractSearchWords(first)
	}

	if len(words) == 0 {
		return nil
	}

	// Sort words by length (longer = more specific) and take top words
	sort.Slice(words, func(i, j int) bool {
		return len(words[i]) > len(words[j])
	})

	maxWords := 50
	if len(words) > maxWords {
		words = words[:maxWords]
	}

	// Build FTS search query using trigram-compatible format for detail=none
	hybrid := &utils.HybridSearchQuery{FTSTerms: words}
	queryStr := hybrid.BuildFTSQuery("OR")

	// Detect FTS table
	ftsTable := detectFTSTable(ctx, sqlDB)
	if ftsTable == "" {
		// No FTS available, skip expansion
		return nil
	}

	// Query for related media using FTS with rank
	relatedRows, err := queryRelatedMediaWithRank(ctx, sqlDB, RelatedSearchParams{
		FTSTable:    ftsTable,
		ExcludePath: first.Path,
		QueryStr:    queryStr,
		Limit:       20,
	})
	if err != nil {
		return err
	}

	// Convert rows to media items and append
	seenPaths := make(map[string]bool)
	for _, m := range *media {
		seenPaths[m.Path] = true
	}

	for _, row := range relatedRows {
		if !seenPaths[row.Path] {
			*media = append(*media, models.MediaWithDB{
				Media: models.Media{
					Path:           row.Path,
					Title:          models.NullStringPtr(row.Title),
					Size:           models.NullInt64Ptr(row.Size),
					Duration:       models.NullInt64Ptr(row.Duration),
					VideoCount:     models.NullInt64Ptr(row.VideoCount),
					AudioCount:     models.NullInt64Ptr(row.AudioCount),
					SubtitleCount:  models.NullInt64Ptr(row.SubtitleCount),
					PlayCount:      models.NullInt64Ptr(row.PlayCount),
					Playhead:       models.NullInt64Ptr(row.Playhead),
					TimeCreated:    models.NullInt64Ptr(row.TimeCreated),
					TimeModified:   models.NullInt64Ptr(row.TimeModified),
					TimeDownloaded: models.NullInt64Ptr(row.TimeDownloaded),
					TimeLastPlayed: models.NullInt64Ptr(row.TimeLastPlayed),
					Score:          models.NullFloat64Ptr(row.Score),
					MediaType:      models.NullStringPtr(row.MediaType),
				},
				DB: row.DB,
			})
			seenPaths[row.Path] = true
		}
	}

	return nil
}

// extractSearchWords extracts searchable words from a media item
func extractSearchWords(m models.MediaWithDB) []string {
	words := make(map[string]bool)

	// Extract from path
	pathWords := strings.FieldsFunc(m.Path, func(r rune) bool {
		return r == '/' || r == '\\' || r == '.' || r == '-' || r == '_' || r == ' '
	})
	for _, w := range pathWords {
		w = strings.TrimSpace(w)
		if len(w) > 2 {
			words[strings.ToLower(w)] = true
		}
	}

	// Extract from title if available
	if m.Title != nil && *m.Title != "" {
		titleWords := strings.FieldsFunc(*m.Title, func(r rune) bool {
			return r == ' ' || r == '.' || r == '-' || r == '_' || r == ',' || r == ':' || r == '(' || r == ')'
		})
		for _, w := range titleWords {
			w = strings.TrimSpace(w)
			if len(w) > 2 {
				words[strings.ToLower(w)] = true
			}
		}
	}

	// Extract from stem (filename without extension)
	stem := m.Stem()
	stemWords := strings.FieldsFunc(stem, func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || r == ' '
	})
	for _, w := range stemWords {
		w = strings.TrimSpace(w)
		if len(w) > 2 {
			words[strings.ToLower(w)] = true
		}
	}

	result := make([]string, 0, len(words))
	for w := range words {
		result = append(result, w)
	}
	return result
}

// detectFTSTable detects the available FTS table for media
func detectFTSTable(ctx context.Context, sqlDB *sql.DB) string {
	tables := []string{"media_fts", "media_fts5", "media_search"}
	for _, table := range tables {
		var exists int
		err := sqlDB.QueryRowContext(ctx,
			"SELECT 1 FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&exists)
		if err == nil && exists == 1 {
			return table
		}
	}
	return ""
}

// relatedMediaRow represents a row from related media query
type relatedMediaRow struct {
	Path           string
	Title          sql.NullString
	Size           sql.NullInt64
	Duration       sql.NullInt64
	VideoCount     sql.NullInt64
	AudioCount     sql.NullInt64
	SubtitleCount  sql.NullInt64
	PlayCount      sql.NullInt64
	Playhead       sql.NullInt64
	TimeCreated    sql.NullInt64
	TimeModified   sql.NullInt64
	TimeDownloaded sql.NullInt64
	TimeLastPlayed sql.NullInt64
	Score          sql.NullFloat64
	MediaType      sql.NullString
	DB             string
	Rank           float64
}

// RelatedSearchParams represents parameters for related media search
type RelatedSearchParams struct {
	FTSTable    string
	ExcludePath string
	QueryStr    string
	Limit       int64
}

// queryRelatedMediaWithRank queries for related media using FTS, ordered by bm25 rank
func queryRelatedMediaWithRank(
	ctx context.Context,
	sqlDB *sql.DB,
	params RelatedSearchParams,
) ([]relatedMediaRow, error) {
	// FTS query with bm25() ranking - works with detail=none + trigram
	// Note: bm25() requires the table name, not an alias
	query := fmt.Sprintf(`
		SELECT
			m.path, m.title, m.size, m.duration,
			m.video_count, m.audio_count, m.subtitle_count,
			m.play_count, m.playhead,
			m.time_created, m.time_modified, m.time_downloaded, m.time_last_played,
			m.score, m.media_type,
			bm25(%s) as rank
		FROM %s, media m
		WHERE m.rowid = %s.rowid
		AND %s MATCH ?
			AND m.path != ?
		ORDER BY bm25(%s) DESC, m.path
		LIMIT ?
	`, params.FTSTable, params.FTSTable, params.FTSTable, params.FTSTable, params.FTSTable)

	rows, err := sqlDB.QueryContext(ctx, query, params.QueryStr, params.ExcludePath, params.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []relatedMediaRow
	for rows.Next() {
		var r relatedMediaRow
		err := rows.Scan(
			&r.Path, &r.Title, &r.Size, &r.Duration,
			&r.VideoCount, &r.AudioCount, &r.SubtitleCount,
			&r.PlayCount, &r.Playhead,
			&r.TimeCreated, &r.TimeModified, &r.TimeDownloaded, &r.TimeLastPlayed,
			&r.Score, &r.MediaType, &r.Rank,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// QueryExecutor executes queries against databases
type QueryExecutor struct {
	filterBuilder *FilterBuilder
}

// NewQueryExecutor creates a new QueryExecutor
func NewQueryExecutor(flags models.GlobalFlags) *QueryExecutor {
	return &QueryExecutor{
		filterBuilder: NewFilterBuilder(flags),
	}
}

// ScanMedia maps SQL rows to MediaWithDB structs
func ScanMedia(rows *sql.Rows, dbPath string) ([]models.MediaWithDB, error) {
	cols, _ := rows.Columns()
	allMedia := []models.MediaWithDB{}

	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		m := db.Media{}
		for i, col := range cols {
			if values[i] == nil {
				continue
			}

			val := values[i]
			lowerCol := strings.ToLower(col)
			if mapStringField(&m, lowerCol, val) {
				continue
			}
			if mapInt64Field(&m, lowerCol, val) {
				continue
			}
			if mapFloat64Field(&m, lowerCol, val) {
				continue
			}
		}

		allMedia = append(allMedia, models.MediaWithDB{
			Media: models.FromDB(m),
			DB:    dbPath,
		})
	}

	return allMedia, rows.Err()
}

func mapStringField(m *db.Media, col string, val any) bool {
	s := utils.GetString(val)
	switch col {
	case "path":
		m.Path = s
	case "title":
		m.Title = sql.NullString{String: s, Valid: true}
	case "album":
		m.Album = sql.NullString{String: s, Valid: true}
	case "artist":
		m.Artist = sql.NullString{String: s, Valid: true}
	case "genre":
		m.Genre = sql.NullString{String: s, Valid: true}
	case "categories":
		m.Categories = sql.NullString{String: s, Valid: true}
	case "description":
		m.Description = sql.NullString{String: s, Valid: true}
	case "language":
		m.Language = sql.NullString{String: s, Valid: true}
	case "video_codecs":
		m.VideoCodecs = sql.NullString{String: s, Valid: true}
	case "audio_codecs":
		m.AudioCodecs = sql.NullString{String: s, Valid: true}
	case "subtitle_codecs":
		m.SubtitleCodecs = sql.NullString{String: s, Valid: true}
	case "media_type":
		m.MediaType = sql.NullString{String: s, Valid: true}
	default:
		return false
	}
	return true
}

func mapInt64Field(m *db.Media, col string, val any) bool {
	i := utils.GetInt64(val)
	switch col {
	case "duration":
		m.Duration = sql.NullInt64{Int64: i, Valid: true}
	case "size":
		m.Size = sql.NullInt64{Int64: i, Valid: true}
	case "time_created":
		m.TimeCreated = sql.NullInt64{Int64: i, Valid: true}
	case "time_modified":
		m.TimeModified = sql.NullInt64{Int64: i, Valid: true}
	case "time_deleted":
		m.TimeDeleted = sql.NullInt64{Int64: i, Valid: true}
	case "time_first_played":
		m.TimeFirstPlayed = sql.NullInt64{Int64: i, Valid: true}
	case "time_last_played":
		m.TimeLastPlayed = sql.NullInt64{Int64: i, Valid: true}
	case "play_count":
		m.PlayCount = sql.NullInt64{Int64: i, Valid: true}
	case "playhead":
		m.Playhead = sql.NullInt64{Int64: i, Valid: true}
	case "time_downloaded":
		m.TimeDownloaded = sql.NullInt64{Int64: i, Valid: true}
	case "width":
		m.Width = sql.NullInt64{Int64: i, Valid: true}
	case "height":
		m.Height = sql.NullInt64{Int64: i, Valid: true}
	default:
		return false
	}
	return true
}

func mapFloat64Field(m *db.Media, col string, val any) bool {
	switch col {
	case "score":
		m.Score = sql.NullFloat64{Float64: utils.GetFloat64(val), Valid: true}
	default:
		return false
	}
	return true
}

// QueryDatabase executes a query against a single database
func QueryDatabase(ctx context.Context, dbPath, query string, args []any) ([]models.MediaWithDB, error) {
	sqlDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ScanMedia(rows, dbPath)
}

// executeMultiDB executes queries against multiple databases concurrently
func (qe *QueryExecutor) executeMultiDB(
	ctx context.Context,
	dbs []string,
	query string,
	args []any,
) ([]models.MediaWithDB, []error) {
	var wg sync.WaitGroup
	results := make(chan []models.MediaWithDB, len(dbs))
	errorsChan := make(chan error, len(dbs))

	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			media, err := QueryDatabase(ctx, path, query, args)
			if err != nil {
				errorsChan <- fmt.Errorf("%s: %w", path, err)
				return
			}
			results <- media
		}(dbPath)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errorsChan)
	}()

	allMedia := []models.MediaWithDB{}
	for media := range results {
		allMedia = append(allMedia, media...)
	}

	var errs []error
	for err := range errorsChan {
		errs = append(errs, err)
	}

	return allMedia, errs
}

func (qe *QueryExecutor) prepareMediaQuery(
	dbs []string,
) (flags models.GlobalFlags, isEpisodic, isMultiDB bool, origLimit, origOffset int) {
	flags = qe.filterBuilder.Flags
	origLimit = flags.Limit
	origOffset = flags.Offset
	isEpisodic = flags.FileCounts != ""
	isMultiDB = len(dbs) > 1

	if isEpisodic {
		// Fetch everything matching other filters so we can count directories accurately
		flags.All = true
		flags.Limit = 0
		flags.Offset = 0
	}

	// For multiple databases, we need to fetch more results from each DB
	if isMultiDB && !flags.All && flags.Limit > 0 {
		flags.Limit += flags.Offset
		flags.Offset = 0
	}

	return flags, isEpisodic, isMultiDB, origLimit, origOffset
}

type postProcessOptions struct {
	media      []models.MediaWithDB
	flags      models.GlobalFlags
	origLimit  int
	origOffset int
	isEpisodic bool
	isMultiDB  bool
}

func (qe *QueryExecutor) postProcessMediaQuery(
	ctx context.Context,
	opts postProcessOptions,
) ([]models.MediaWithDB, error) {
	allMedia := opts.media
	if opts.isEpisodic {
		counts := make(map[string]int64)
		for _, m := range allMedia {
			counts[m.Parent()]++
		}

		r, err := utils.ParseRange(opts.flags.FileCounts, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		})

		if err == nil {
			var filtered []models.MediaWithDB
			for _, m := range allMedia {
				if r.Matches(counts[m.Parent()]) {
					filtered = append(filtered, m)
				}
			}
			allMedia = filtered
		}

		// Apply sorting
		NewSortBuilder(opts.flags).Sort(allMedia)

		// Apply original limit/offset
		if opts.origOffset > 0 {
			if opts.origOffset >= len(allMedia) {
				return []models.MediaWithDB{}, nil
			}
			allMedia = allMedia[opts.origOffset:]
		}
		if opts.origLimit > 0 && len(allMedia) > opts.origLimit {
			allMedia = allMedia[:opts.origLimit]
		}
	}

	// Group by parent directory
	if opts.flags.GroupByParent {
		allMedia = qe.GroupByParent(allMedia, opts.flags)
	}

	// Fetch siblings
	if opts.flags.FetchSiblings != "" {
		var err error
		allMedia, err = qe.FetchSiblings(ctx, allMedia, opts.flags)
		if err != nil {
			return allMedia, err
		}
	}

	// For multiple databases, apply limit/offset after merging and sorting
	if opts.isMultiDB && !opts.isEpisodic && !opts.flags.GroupByParent && !opts.flags.All && opts.origLimit > 0 {
		NewSortBuilder(opts.flags).Sort(allMedia)

		if opts.origOffset > 0 {
			if opts.origOffset >= len(allMedia) {
				return []models.MediaWithDB{}, nil
			}
			allMedia = allMedia[opts.origOffset:]
		}
		if len(allMedia) > opts.origLimit {
			allMedia = allMedia[:opts.origLimit]
		}
	}

	return allMedia, nil
}

// MediaQuery executes a query against multiple databases concurrently
func (qe *QueryExecutor) MediaQuery(ctx context.Context, dbs []string) ([]models.MediaWithDB, error) {
	flags, isEpisodic, isMultiDB, origLimit, origOffset := qe.prepareMediaQuery(dbs)

	resolvedFlags, err := qe.ResolvePercentileFlags(ctx, dbs, flags)
	if err == nil {
		flags = resolvedFlags
	}

	// Rebuild filter builder with resolved flags
	fb := NewFilterBuilder(flags)
	query, args := fb.BuildQuery(ctx, "*")

	allMedia, errs := qe.executeMultiDB(ctx, dbs, query, args)
	if len(errs) > 0 {
		return allMedia, errors.Join(errs...)
	}

	return qe.postProcessMediaQuery(ctx, postProcessOptions{
		media:      allMedia,
		flags:      flags,
		origLimit:  origLimit,
		origOffset: origOffset,
		isEpisodic: isEpisodic,
		isMultiDB:  isMultiDB,
	})
}

// MediaQueryCount executes a count query against multiple databases concurrently
func (qe *QueryExecutor) MediaQueryCount(ctx context.Context, dbs []string) (int64, error) {
	flags := qe.filterBuilder.Flags

	if flags.FileCounts != "" {
		tempFlags := flags
		tempFlags.All = true
		tempFlags.Limit = 0
		tempFlags.Offset = 0

		tempExecutor := NewQueryExecutor(tempFlags)
		allMedia, err := tempExecutor.MediaQuery(ctx, dbs)
		if err != nil {
			return 0, err
		}
		return int64(len(allMedia)), nil
	}

	query, args := qe.filterBuilder.BuildCount(ctx)

	var wg sync.WaitGroup
	results := make(chan int64, len(dbs))
	errorsChan := make(chan error, len(dbs))

	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sqlDB, err := db.Connect(ctx, path)
			if err != nil {
				errorsChan <- err
				return
			}
			defer sqlDB.Close()

			var count int64
			err = sqlDB.QueryRowContext(ctx, query, args...).Scan(&count)
			if err != nil {
				errorsChan <- err
				return
			}
			results <- count
		}(dbPath)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errorsChan)
	}()

	var total int64
	for count := range results {
		total += count
	}

	var errs []error
	for err := range errorsChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return total, errors.Join(errs...)
	}

	return total, nil
}

func (qe *QueryExecutor) GroupByParent(allMedia []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	type GroupedMedia struct {
		ParentPath              string  `json:"parent_path"`
		EpisodeCount            int64   `json:"episode_count"`
		TotalSize               int64   `json:"total_size"`
		TotalDuration           int64   `json:"total_duration"`
		LatestEpisodeTime       *string `json:"latest_episode_time,omitempty"`
		RepresentativePath      string  `json:"representative_path"`
		RepresentativeMediaType *string `json:"media_type,omitempty"`
	}

	groups := make(map[string]*GroupedMedia)
	for _, m := range allMedia {
		parent := m.Parent()
		if _, ok := groups[parent]; !ok {
			groups[parent] = &GroupedMedia{
				ParentPath:              parent,
				EpisodeCount:            0,
				TotalSize:               0,
				TotalDuration:           0,
				RepresentativePath:      m.Path,
				RepresentativeMediaType: m.MediaType,
			}
		}
		g := groups[parent]
		g.EpisodeCount++
		if m.Size != nil {
			g.TotalSize += *m.Size
		}
		if m.Duration != nil {
			g.TotalDuration += *m.Duration
		}
		if g.LatestEpisodeTime == nil || m.Path > *g.LatestEpisodeTime {
			g.LatestEpisodeTime = &m.Path
		}
	}

	result := make([]models.MediaWithDB, 0, len(groups))
	for _, g := range groups {
		m := models.MediaWithDB{
			Media: models.Media{
				Path:      g.RepresentativePath,
				MediaType: g.RepresentativeMediaType,
				Size:      &g.TotalSize,
				Duration:  &g.TotalDuration,
				Title:     &g.ParentPath,
			},
			DB:            allMedia[0].DB,
			EpisodeCount:  g.EpisodeCount,
			TotalSize:     g.TotalSize,
			TotalDuration: g.TotalDuration,
		}
		result = append(result, m)
	}

	NewSortBuilder(flags).Sort(result)
	return result
}

func (qe *QueryExecutor) determineSiblingLimit(filesInDir []models.MediaWithDB, flags models.GlobalFlags) (int, bool) {
	limit := flags.FetchSiblingsMax
	switch flags.FetchSiblings {
	case "all", "always":
		return 2000, true
	case "each":
		if limit <= 0 {
			limit = len(filesInDir)
		}
		return limit, true
	case "if-audiobook":
		isAudiobook := false
		for _, f := range filesInDir {
			if strings.Contains(strings.ToLower(f.Path), "audiobook") {
				isAudiobook = true
				break
			}
		}
		if !isAudiobook {
			return 0, false
		}
		if limit <= 0 {
			limit = 2000
		}
		return limit, true
	default:
		if utils.IsDigit(flags.FetchSiblings) {
			if l, err := strconv.Atoi(flags.FetchSiblings); err == nil {
				return l, true
			}
		}
		return 0, false
	}
}

func (qe *QueryExecutor) querySiblings(
	ctx context.Context,
	dbPath, dir string,
	limit int,
) ([]models.MediaWithDB, error) {
	query := "SELECT path, path_tokenized, title, duration, size, time_created, time_modified, time_deleted, time_first_played, time_last_played, play_count, playhead, media_type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs, video_count, audio_count, subtitle_count, album, artist, genre, categories, description, language, time_downloaded, score, fasthash, sha256, is_deduped FROM media WHERE time_deleted = 0 AND path LIKE ? ORDER BY path LIMIT ?"
	pattern := dir + "%"
	return QueryDatabase(ctx, dbPath, query, []any{pattern, limit})
}

func (qe *QueryExecutor) groupByParentDir(media []models.MediaWithDB) map[string][]models.MediaWithDB {
	parentToFiles := make(map[string][]models.MediaWithDB)
	for _, m := range media {
		dir := m.Parent()
		if !strings.HasSuffix(dir, "/") && !strings.HasSuffix(dir, "\\") {
			dir += string(filepath.Separator)
		}
		parentToFiles[dir] = append(parentToFiles[dir], m)
	}
	return parentToFiles
}

func (qe *QueryExecutor) fetchSiblingsForDir(
	ctx context.Context,
	dir string,
	filesInDir []models.MediaWithDB,
	flags models.GlobalFlags,
) ([]models.MediaWithDB, error) {
	limit, shouldQuery := qe.determineSiblingLimit(filesInDir, flags)
	if !shouldQuery {
		return filesInDir, nil
	}

	dbPath := filesInDir[0].DB
	return qe.querySiblings(ctx, dbPath, dir, limit)
}

func (qe *QueryExecutor) FetchSiblings(
	ctx context.Context,
	media []models.MediaWithDB,
	flags models.GlobalFlags,
) ([]models.MediaWithDB, error) {
	if len(media) == 0 {
		return media, nil
	}

	parentToFiles := qe.groupByParentDir(media)

	var allSiblings []models.MediaWithDB
	seenPaths := make(map[string]bool)

	for dir, filesInDir := range parentToFiles {
		siblings, err := qe.fetchSiblingsForDir(ctx, dir, filesInDir, flags)
		if err != nil {
			return nil, err
		}

		for _, s := range siblings {
			if !seenPaths[s.Path] {
				allSiblings = append(allSiblings, s)
				seenPaths[s.Path] = true
			}
		}
	}

	return allSiblings, nil
}

func (qe *QueryExecutor) cleanPercentileFlags(flags models.GlobalFlags) models.GlobalFlags {
	tempFlags := flags
	cleaner := func(ranges []string) []string {
		var clean []string
		for _, s := range ranges {
			if _, _, ok := utils.ParsePercentileRange(s); !ok {
				clean = append(clean, s)
			}
		}
		return clean
	}

	tempFlags.Size = cleaner(tempFlags.Size)
	tempFlags.Duration = cleaner(tempFlags.Duration)
	tempFlags.Modified = cleaner(tempFlags.Modified)
	tempFlags.Created = cleaner(tempFlags.Created)
	tempFlags.Downloaded = cleaner(tempFlags.Downloaded)

	if _, _, ok := utils.ParsePercentileRange(flags.FileCounts); ok {
		tempFlags.FileCounts = ""
	}
	tempFlags.All = true
	tempFlags.Limit = 0
	return tempFlags
}

func (qe *QueryExecutor) queryPercentileValues(
	ctx context.Context,
	dbPath, sqlQuery string,
	args []any,
	field string,
) ([]int64, error) {
	sqlDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	rows, err := sqlDB.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []int64
	if field == "episodes" {
		gCounts := make(map[string]int64)
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil {
				gCounts[filepath.Dir(p)]++
			}
		}
		if err := rows.Err(); err != nil {
			models.Log.Debug("Percentile query error", "error", err)
		}
		for _, c := range gCounts {
			values = append(values, c)
		}
	} else {
		for rows.Next() {
			var v sql.NullInt64
			if err := rows.Scan(&v); err == nil && v.Valid {
				values = append(values, v.Int64)
			}
		}
		if err := rows.Err(); err != nil {
			models.Log.Debug("Percentile query error", "error", err)
		}
	}
	return values, nil
}

func (qe *QueryExecutor) getPercentileValues(
	ctx context.Context,
	dbs []string,
	flags models.GlobalFlags,
	field string,
) []int64 {
	tempFlags := qe.cleanPercentileFlags(flags)

	fb := NewFilterBuilder(tempFlags)
	var sqlQuery string
	var args []any
	if field == "episodes" {
		sqlQuery, args = fb.BuildSelect(ctx, "path")
	} else {
		sqlQuery, args = fb.BuildSelect(ctx, field)
	}

	var values []int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			localValues, err := qe.queryPercentileValues(ctx, path, sqlQuery, args, field)
			if err != nil {
				return
			}
			mu.Lock()
			values = append(values, localValues...)
			mu.Unlock()
		}(dbPath)
	}
	wg.Wait()
	return values
}

func (qe *QueryExecutor) applyPercentileToRange(ranges []string, mapping []int64) []string {
	var newRanges []string
	for _, s := range ranges {
		if pmin, pmax, ok := utils.ParsePercentileRange(s); ok {
			minVal := mapping[int(pmin)]
			maxVal := mapping[int(pmax)]
			newRanges = append(newRanges, fmt.Sprintf("+%d", minVal))
			newRanges = append(newRanges, fmt.Sprintf("-%d", maxVal))
		} else {
			newRanges = append(newRanges, s)
		}
	}
	return newRanges
}

func (qe *QueryExecutor) resolveTimePercentiles(
	ctx context.Context,
	dbs []string,
	flags *models.GlobalFlags,
	field string,
	ranges []string,
	after, before *string,
) {
	values := qe.getPercentileValues(ctx, dbs, *flags, field)
	if len(values) > 0 {
		mapping := utils.CalculatePercentiles(values)
		for _, r := range ranges {
			if pmin, pmax, ok := utils.ParsePercentileRange(r); ok {
				*after = strconv.FormatInt(mapping[int(pmin)], 10)
				*before = strconv.FormatInt(mapping[int(pmax)], 10)
			}
		}
	}
}

func (qe *QueryExecutor) resolveSizePercentiles(ctx context.Context, dbs []string, flags *models.GlobalFlags) {
	values := qe.getPercentileValues(ctx, dbs, *flags, "size")
	if len(values) > 0 {
		mapping := utils.CalculatePercentiles(values)
		flags.Size = qe.applyPercentileToRange(flags.Size, mapping)
	}
}

func (qe *QueryExecutor) resolveDurationPercentiles(ctx context.Context, dbs []string, flags *models.GlobalFlags) {
	values := qe.getPercentileValues(ctx, dbs, *flags, "duration")
	if len(values) > 0 {
		mapping := utils.CalculatePercentiles(values)
		flags.Duration = qe.applyPercentileToRange(flags.Duration, mapping)
	}
}

func (qe *QueryExecutor) resolveEpisodePercentiles(ctx context.Context, dbs []string, flags *models.GlobalFlags) {
	values := qe.getPercentileValues(ctx, dbs, *flags, "episodes")
	if len(values) > 0 {
		mapping := utils.CalculatePercentiles(values)
		if pmin, pmax, ok := utils.ParsePercentileRange(flags.FileCounts); ok {
			minVal := mapping[int(pmin)]
			maxVal := mapping[int(pmax)]
			flags.FileCounts = fmt.Sprintf("+%d,-%d", minVal, maxVal)
		}
	}
}

func (qe *QueryExecutor) hasAnyPercentileFlag(flags models.GlobalFlags) bool {
	check := func(ranges []string) bool {
		for _, s := range ranges {
			if _, _, ok := utils.ParsePercentileRange(s); ok {
				return true
			}
		}
		return false
	}

	if check(flags.Size) || check(flags.Duration) || check(flags.Modified) ||
		check(flags.Created) || check(flags.Downloaded) {

		return true
	}

	if _, _, ok := utils.ParsePercentileRange(flags.FileCounts); ok {
		return true
	}

	return false
}

func (qe *QueryExecutor) resolveAllPercentiles(ctx context.Context, dbs []string, flags *models.GlobalFlags) {
	qe.resolveSizePercentiles(ctx, dbs, flags)
	qe.resolveDurationPercentiles(ctx, dbs, flags)

	qe.resolveTimePercentiles(
		ctx, dbs, flags, "time_modified",
		flags.Modified, &flags.ModifiedAfter, &flags.ModifiedBefore,
	)
	qe.resolveTimePercentiles(
		ctx, dbs, flags, "time_created",
		flags.Created, &flags.CreatedAfter, &flags.CreatedBefore,
	)
	qe.resolveTimePercentiles(
		ctx, dbs, flags, "time_downloaded",
		flags.Downloaded, &flags.DownloadedAfter, &flags.DownloadedBefore,
	)

	qe.resolveEpisodePercentiles(ctx, dbs, flags)
}

func (qe *QueryExecutor) ResolvePercentileFlags(
	ctx context.Context,
	dbs []string,
	flags models.GlobalFlags,
) (models.GlobalFlags, error) {
	if !qe.hasAnyPercentileFlag(flags) {
		return flags, nil
	}

	qe.resolveAllPercentiles(ctx, dbs, &flags)

	return flags, nil
}
