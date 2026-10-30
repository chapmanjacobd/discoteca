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
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/formatutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/textutil"
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

	// Sidecar subtitles are captions, not media. The web grid opts into this;
	// explicit CLI path/extension filters still allow inspecting them.
	if fb.Flags.HideSubtitles && len(fb.Flags.Ext) == 0 && len(fb.Flags.Paths) == 0 {
		subtitleClauses := make([]string, 0, len(media.SubtitleExtensions))
		for _, ext := range media.SubtitleExtensions {
			subtitleClauses = append(
				subtitleClauses,
				fmt.Sprintf("LOWER(%s) NOT LIKE '%%%s'", fb.col("path"), ext),
			)
		}
		*whereClauses = append(*whereClauses, "("+strings.Join(subtitleClauses, " AND ")+")")
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

	// Play count filters (indexed column). Pointers distinguish "unset" from an
	// explicit zero, so --play-count-max 0 works.
	if fb.Flags.PlayCountMin != nil {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("play_count")))
		*args = append(*args, *fb.Flags.PlayCountMin)
	}
	if fb.Flags.PlayCountMax != nil {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("play_count")))
		*args = append(*args, *fb.Flags.PlayCountMax)
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
		if ts := formatutil.ParseDateOrRelative(fb.Flags.DeletedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_deleted")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.DeletedBefore != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.DeletedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_deleted")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.CreatedAfter != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.CreatedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_created")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.CreatedBefore != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.CreatedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_created")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.ModifiedAfter != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.ModifiedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_modified")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.ModifiedBefore != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.ModifiedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_modified")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.DownloadedAfter != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.DownloadedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_downloaded")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.DownloadedBefore != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.DownloadedBefore); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s <= ?", fb.col("time_downloaded")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.PlayedAfter != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.PlayedAfter); ts > 0 {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s >= ?", fb.col("time_last_played")))
			*args = append(*args, ts)
		}
	}
	if fb.Flags.PlayedBefore != "" {
		if ts := formatutil.ParseDateOrRelative(fb.Flags.PlayedBefore); ts > 0 {
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

	// Playhead/playback status. "Finished" is playhead >= 95% of duration.
	// Unfinished and InProgress both mean started but not finished, which is
	// the same concept the UI exposes under two names.
	if fb.Flags.Unfinished || fb.Flags.InProgress {
		playhead := fmt.Sprintf("COALESCE(%s, 0)", fb.col("playhead"))
		duration := fmt.Sprintf("COALESCE(%s, 0)", fb.col("duration"))
		finished := fmt.Sprintf("(%s > 0 AND %s >= %s * 0.95)", duration, playhead, duration)
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s > 0 AND NOT %s", playhead, finished))
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
	fb.buildMediaLocationFilters(whereClauses)
	fb.buildExtensionFilters(whereClauses, args)
	fb.buildCategoryFilters(whereClauses, args)

	allInclude, pathContains := fb.splitSearchTerms()

	if len(allInclude) > 0 {
		fb.buildSearchClauses(ctx, allInclude, whereClauses, args)
	}

	fb.buildIncludeFilters(whereClauses, args)
	fb.buildExcludeFilters(whereClauses, args)
	fb.buildRegexFilter(whereClauses, args)
	fb.buildPathContainsFilters(pathContains, whereClauses, args)
}

func (fb *FilterBuilder) buildMediaLocationFilters(whereClauses *[]string) {
	if fb.Flags.OnlineMediaOnly {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s LIKE 'http%%'", fb.col("path")))
	}
	if fb.Flags.LocalMediaOnly {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s NOT LIKE 'http%%'", fb.col("path")))
	}
}

func (fb *FilterBuilder) buildExtensionFilters(whereClauses *[]string, args *[]any) {
	if len(fb.Flags.Ext) == 0 {
		return
	}
	extClauses := make([]string, 0, len(fb.Flags.Ext))
	for _, ext := range fb.Flags.Ext {
		extClauses = append(extClauses, fmt.Sprintf("%s LIKE ?", fb.col("path")))
		*args = append(*args, "%"+ext)
	}
	*whereClauses = append(*whereClauses, "("+strings.Join(extClauses, " OR ")+")")
}

func (fb *FilterBuilder) buildCategoryFilters(whereClauses *[]string, args *[]any) {
	if len(fb.Flags.Category) == 0 {
		return
	}
	var catClauses []string
	for _, cat := range fb.Flags.Category {
		if cat == "Uncategorized" {
			catClauses = append(catClauses,
				fmt.Sprintf("(%s IS NULL OR %s = '')", fb.col("categories"), fb.col("categories")))
		} else {
			catClauses = append(catClauses, fmt.Sprintf("%s LIKE '%%' || ? || '%%'", fb.col("categories")))
			*args = append(*args, ";"+cat+";")
		}
	}
	if len(catClauses) > 0 {
		*whereClauses = append(*whereClauses, "("+strings.Join(catClauses, " OR ")+")")
	}
}

func (fb *FilterBuilder) splitSearchTerms() (allInclude, pathContains []string) {
	// --include is a path pattern, not a free-text search. It is handled by
	// buildIncludeFilters (SQL) and matchesPathFilters (in memory).
	allInclude = append([]string{}, fb.Flags.Search...)
	pathContains = append([]string{}, fb.Flags.PathContains...)

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
	return filteredInclude, pathContains
}

func (fb *FilterBuilder) buildSearchClauses(
	ctx context.Context,
	allInclude []string,
	whereClauses *[]string,
	args *[]any,
) {
	joinOp := " AND "
	if fb.Flags.FlexibleSearch {
		joinOp = " OR "
	}

	useFTS := fb.resolveSearchMode(ctx)

	if useFTS && !fb.Flags.Exact {
		fb.buildFTSSearchClauses(allInclude, joinOp, whereClauses, args)
	} else {
		fb.buildLikeSearchClauses(allInclude, joinOp, whereClauses, args)
	}
}

func (fb *FilterBuilder) resolveSearchMode(ctx context.Context) bool {
	_ = ctx
	// Runtime detection is deliberately not used here: the builder has no
	// per-database handle, and a process-global detection would make mixed
	// FTS/non-FTS database lists behave differently. Callers opt in per
	// database with --fts; --no-fts always wins.
	return fb.Flags.FTS && !fb.Flags.NoFTS
}

func (fb *FilterBuilder) buildFTSSearchClauses(
	allInclude []string,
	joinOp string,
	whereClauses *[]string,
	args *[]any,
) {
	queryStr := strings.Join(allInclude, " ")
	hybrid := ParseHybridSearchQuery(queryStr)

	if hybrid.HasFTSTerms() {
		ftsQuery := hybrid.BuildFTSQuery(joinOp)
		if ftsQuery != "" {
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s MATCH ?", fb.getFTSTable()))
			*args = append(*args, ftsQuery)
		}
	}

	for _, phrase := range hybrid.Phrases {
		*whereClauses = append(*whereClauses,
			fmt.Sprintf(
				"(%s LIKE ? OR %s LIKE ? OR %s LIKE ?)",
				fb.col("path"), fb.col("title"), fb.col("description"),
			),
		)
		pattern := "%" + phrase + "%"
		*args = append(*args, pattern, pattern, pattern)
	}
}

func (fb *FilterBuilder) buildLikeSearchClauses(
	allInclude []string,
	joinOp string,
	whereClauses *[]string,
	args *[]any,
) {
	var searchParts []string
	for _, term := range allInclude {
		if fb.Flags.Exact {
			searchParts = append(searchParts, fmt.Sprintf(
				"(%s LIKE ? ESCAPE '\\' OR %s LIKE ? ESCAPE '\\')",
				fb.col("path"), fb.col("path"),
			))
			*args = append(*args, "%/"+term+".%", "%/"+term)
		} else {
			searchParts = append(searchParts,
				fmt.Sprintf(
					"(%s LIKE ? ESCAPE '\\' OR %s LIKE ? ESCAPE '\\' OR %s LIKE ? ESCAPE '\\')",
					fb.col("path"), fb.col("title"), fb.col("path_tokenized"),
				),
			)
			literal := strings.Trim(term, `"'`)
			literal = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(literal)
			pattern := "%" + strings.ReplaceAll(literal, " ", "%") + "%"
			*args = append(*args, pattern, pattern, pattern)
		}
	}
	*whereClauses = append(*whereClauses, "("+strings.Join(searchParts, joinOp)+")")
}

// buildIncludeFilters applies path patterns using the same glob/prefix
// semantics as textutil.MatchesAny.
func (fb *FilterBuilder) buildIncludeFilters(whereClauses *[]string, args *[]any) {
	for _, inc := range fb.Flags.Include {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s LIKE ? ESCAPE '\\'", fb.col("path")))
		*args = append(*args, textutil.PathPatternLike(inc))
	}
}

// buildExcludeFilters applies path patterns using the same glob/prefix
// semantics as textutil.MatchesAny.
func (fb *FilterBuilder) buildExcludeFilters(whereClauses *[]string, args *[]any) {
	for _, exc := range fb.Flags.Exclude {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s NOT LIKE ? ESCAPE '\\'", fb.col("path")))
		*args = append(*args, textutil.PathPatternLike(exc))
	}
}

func (fb *FilterBuilder) buildRegexFilter(whereClauses *[]string, args *[]any) {
	if fb.Flags.Regex != "" {
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s REGEXP ?", fb.col("path")))
		*args = append(*args, fb.Flags.Regex)
	}
}

func (fb *FilterBuilder) buildPathContainsFilters(pathContains []string, whereClauses *[]string, args *[]any) {
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
	useFTSJoin := fb.usesFTSJoin()

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
	if orderExpr := fb.BuildQueryOrder(); orderExpr != "" {
		query += " ORDER BY " + orderExpr
	} else if fb.Flags.Random {
		query += " ORDER BY RANDOM()"
	}

	// Limit and offset
	if !fb.Flags.All && fb.Flags.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", fb.Flags.Limit)
		if fb.Flags.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", fb.Flags.Offset)
		}
	} else if fb.Flags.Offset > 0 {
		// SQLite requires a LIMIT when OFFSET is present.
		query += fmt.Sprintf(" LIMIT -1 OFFSET %d", fb.Flags.Offset)
	}

	return query, args
}

// OverrideSort translates logical sort fields into SQL expressions
func (fb *FilterBuilder) OverrideSort(s string) string {
	switch strings.TrimSpace(s) {
	case "default", "xklb", "xklb_default":
		return fb.xklbSortSQL(false)
	case "reverse_default", "reverse_xklb", "reverse_xklb_default":
		return fb.xklbSortSQL(true)
	}

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

// xklbSortSQL returns the SQL expression for the xklb default media order.
// It mirrors the in-memory XklbDefaultSort comparator so single-database and
// multi-database fetches start from the same order.
func (fb *FilterBuilder) xklbSortSQL(reverse bool) string {
	dir := func(asc bool) string {
		if reverse {
			asc = !asc
		}
		if asc {
			return "ASC"
		}
		return "DESC"
	}
	remote := fmt.Sprintf("CASE WHEN %s LIKE 'http%%' THEN 1 ELSE 0 END", fb.col("path"))
	untitled := fmt.Sprintf(
		"CASE WHEN %s IS NULL OR %s = '' THEN 1 ELSE 0 END",
		fb.col("title"),
		fb.col("title"),
	)
	const format = "COALESCE(%s, 0) %s, COALESCE(%s, 0) %s, %s %s, " +
		"COALESCE(%s, 0) %s, COALESCE(%s, 0) %s, COALESCE(%s, 0) %s, " +
		"COALESCE(%s, 0) %s, %s %s, %s %s"
	return fmt.Sprintf(
		format,
		fb.col("video_count"), dir(false),
		fb.col("audio_count"), dir(false),
		remote, dir(true),
		fb.col("subtitle_count"), dir(false),
		fb.col("play_count"), dir(true),
		fb.col("playhead"), dir(false),
		fb.col("time_last_played"), dir(true),
		untitled, dir(true),
		fb.col("path"), dir(true),
	)
}

// BuildQueryOrder returns the ORDER BY clause for the configured sort, or ""
// when the order must be applied in memory.
func (fb *FilterBuilder) BuildQueryOrder() string {
	if fb.Flags.Random || fb.Flags.NatSort || fb.Flags.SortBy == "" {
		return ""
	}
	sortExpr := fb.OverrideSort(fb.Flags.SortBy)
	if fb.Flags.SortBy == "default" || fb.Flags.SortBy == "xklb" ||
		fb.Flags.SortBy == "reverse_default" || fb.Flags.SortBy == "reverse_xklb" {
		return sortExpr
	}
	order := "ASC"
	if fb.Flags.Reverse {
		order = "DESC"
	}
	return fmt.Sprintf("%s %s", sortExpr, order)
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
	for _, term := range fb.Flags.Search {
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
	return fb.Flags.FTS && !fb.Flags.NoFTS && fb.hasSearchTerms()
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
	if fb.Flags.Exists && !fileutil.FileExists(m.Path) {
		return false
	}

	// Include/exclude patterns
	if len(fb.Flags.Include) > 0 && !textutil.MatchesAny(m.Path, fb.Flags.Include) {
		return false
	}
	if len(fb.Flags.Exclude) > 0 && textutil.MatchesAny(m.Path, fb.Flags.Exclude) {
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
	invalidRegex := false
	if fb.Flags.Regex != "" {
		compiled, err := regexp.Compile(fb.Flags.Regex)
		if err == nil {
			regex = compiled
		} else {
			// Keep in-memory filtering from panicking on user input. The SQL
			// path reports the same invalid pattern as a query error.
			invalidRegex = true
		}
	}

	sizeRanges := fb.parseSizeRanges()
	durationRanges := fb.parseDurationRanges()

	return func(m models.MediaWithDB) bool {
		if invalidRegex {
			return false
		}
		if !fb.matchesPathFilters(m, regex) {
			return false
		}

		if !fb.matchesRangeFilters(m, sizeRanges, durationRanges) {
			return false
		}

		return true
	}
}

// FilterMedia applies the in-memory-evaluable subset of filters (path patterns,
// extension, size, duration, and existence). SQL-backed predicates such as
// media type, categories, dates, and playback state must be applied by the
// query that produced media.
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
		seed := sb.Flags.RandomSeed
		if seed == 0 {
			// No seed supplied (CLI, one-shot requests): reshuffle per call.
			seed = rand.Uint64()
		}
		rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
		rng.Shuffle(len(media), func(i, j int) {
			media[i], media[j] = media[j], media[i]
		})
		applyReRank(media, sb.Flags)
		return
	}

	sortBy := strings.TrimSpace(sb.Flags.SortBy)
	if sortBy == "" {
		sortBy = "default"
	}

	if sb.Flags.NoPlayInOrder {
		sb.sortByConfig(media, sortBy, sb.Flags.Reverse)
		applyReRank(media, sb.Flags)
		return
	}

	if sortBy == "default" {
		config := "xklb"
		if sb.Flags.PlayInOrder != "" {
			config = sb.Flags.PlayInOrder
		}
		sb.sortAdvancedWithReverse(media, config, sb.Flags.Reverse)
		applyReRank(media, sb.Flags)
		return
	}

	sb.sortByConfig(media, sortBy, sb.Flags.Reverse)
	applyReRank(media, sb.Flags)
}

// SortBasic sorts by the configured SortBy using the unified sort contract.
func (sb *SortBuilder) SortBasic(media []models.MediaWithDB) {
	sb.sortByConfig(media, sb.Flags.SortBy, sb.Flags.Reverse)
	applyReRank(media, sb.Flags)
}

func (sb *SortBuilder) sortByConfig(media []models.MediaWithDB, config string, reverse bool) {
	config = strings.TrimSpace(config)
	if config == "" {
		config = "default"
	}

	if config == "path" {
		sb.sortPath(media, reverse)
		return
	}

	fields, _ := ParseSortConfig(config)
	if !allSortFieldsSupported(fields) {
		unsupported := firstUnsupportedSortField(fields)
		warnUnsupportedSortField(unsupported)
		sb.sortAdvancedWithReverse(media, "xklb", reverse)
		return
	}
	sb.sortAdvancedWithReverse(media, config, reverse)
}

func (sb *SortBuilder) sortPath(media []models.MediaWithDB, reverse bool) {
	less := func(i, j int) bool {
		if sb.Flags.NatSort {
			return textutil.NaturalLess(media[i].Path, media[j].Path)
		}
		return media[i].Path < media[j].Path
	}
	applyLess(media, less, reverse)
}

func applyLess(media []models.MediaWithDB, less func(i, j int) bool, reverse bool) {
	if reverse {
		sort.SliceStable(media, func(i, j int) bool { return less(j, i) })
		return
	}
	sort.SliceStable(media, less)
}

func warnUnsupportedSortField(field string) {
	if logging.Log != nil {
		logging.Log.Warn("Unsupported sort field; using default order", "field", field)
	}
}

func firstUnsupportedSortField(fields []SortField) string {
	for _, f := range fields {
		if !isSupportedSortField(f.Field) {
			return f.Field
		}
	}
	return ""
}

func allSortFieldsSupported(fields []SortField) bool {
	return firstUnsupportedSortField(fields) == ""
}

// isSupportedSortField reports whether a sort key has a defined in-memory
// comparator. Unknown keys are never silently treated as path.
func isSupportedSortField(field string) bool {
	switch field {
	case "ps", "pts", "parent", "stem", "extension", "path", "title",
		"media_type", "genre", "artist", "album", "language", "categories",
		"video_codecs", "audio_codecs", "path_tokenized", "fasthash", "sha256",
		"path_is_remote", "title_is_null", "size_per_count", "folders",
		"folder_count", "count", "priority", "priorityfast", "bitrate",
		"progress", "date_created", "date_modified", "month_created",
		"month_modified", "time_scanned":
		return true
	}
	return IsNumericField(field)
}

// ValidateSortConfig reports whether a sort configuration uses only known
// presets, algorithms, meta markers, and field names. It is the allowlist for
// user-supplied sort query parameters before they can reach SQL.
func ValidateSortConfig(config string) error {
	config = strings.TrimSpace(config)
	if config == "" {
		return nil
	}
	switch config {
	case "default", "xklb", "xklb_default", "reverse_default", "reverse_xklb",
		"reverse_xklb_default", "du", "du_default", "reverse_du", "reverse_du_default",
		"random", "random()", "RANDOM", "RANDOM()":
		return nil
	}

	knownAlgs := map[string]bool{
		"natural": true, "ignorecase": true, "lowercase": true, "human": true,
		"locale": true, "signed": true, "os": true, "python": true,
	}
	if knownAlgs[config] {
		return nil
	}

	for part := range strings.SplitSeq(config, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch part {
		case "_weighted_rerank", "_natural_order", "_related_media":
			continue
		}
		field := parseSortField(part).Field
		if field == "ps" || isSupportedSortField(field) {
			continue
		}
		return fmt.Errorf("unsupported sort field %q", field)
	}
	return nil
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
	case "is_deduped":
		return float64(utils.Int64Value(m.IsDeduped))
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
	case "path_tokenized":
		return utils.StringValue(m.PathTokenized)
	case "fasthash":
		return utils.StringValue(m.Fasthash)
	case "sha256":
		return utils.StringValue(m.Sha256)
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
		"track_number": true, "count": true, "is_deduped": true,
	}
	return numericFields[field]
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
		less = textutil.NaturalLess(valI, valJ)
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
	sb.sortAdvancedWithReverse(media, config, false)
	applyReRank(media, sb.Flags)
}

func (sb *SortBuilder) sortAdvancedWithReverse(media []models.MediaWithDB, config string, reverse bool) {
	if config == "" {
		sb.SortBasic(media)
		return
	}

	config = expandSortPreset(config)

	// Groups are applied as successive stable full sorts: each later group
	// reorders the entire slice, and stability preserves the earlier group's
	// order only for items that compare equal. Later groups therefore take
	// precedence over earlier groups.
	groups := ParseSortConfigWithGroups(config)

	if len(groups) > 1 {
		for _, group := range groups {
			switch group.Alg {
			case "weighted":
				weightedRerank(media, group.Fields, reverse)
			case "related":
				// Related media is expanded earlier in SortMediaWithExpansion;
				// ordering here is natural order over the group's fields.
				applyStandardSort(media, group.Fields, "natural", reverse)
			default:
				applyStandardSort(media, group.Fields, group.Alg, reverse)
			}
		}
		return
	}

	sortFields, alg := ParseSortConfig(config)

	if len(sortFields) == 1 && sortFields[0].Field == "ps" {
		less := func(i, j int) bool {
			valI := GetSortValueString(media[i], "ps")
			valJ := GetSortValueString(media[j], "ps")
			if alg == "python" {
				return valI < valJ
			}
			return textutil.NaturalLess(valI, valJ)
		}
		applyLess(media, less, sortFields[0].Reverse != reverse)
		return
	}

	applyStandardSort(media, sortFields, alg, reverse)
}

func expandSortPreset(config string) string {
	switch config {
	case "xklb", "xklb_default":
		return "video_count desc,audio_count desc,path_is_remote asc,subtitle_count desc,play_count asc,playhead desc,time_last_played asc,title_is_null asc,path asc"
	case "reverse_xklb", "reverse_xklb_default":
		return "video_count asc,audio_count asc,path_is_remote desc,subtitle_count asc,play_count desc,playhead asc,time_last_played desc,title_is_null desc,path desc"
	case "du", "du_default":
		return "size_per_count desc,size desc,count desc,folders desc,path desc"
	case "reverse_du", "reverse_du_default":
		return "size_per_count asc,size asc,count asc,folders asc,path asc"
	default:
		return config
	}
}

// applyStandardSort applies standard multi-field sorting to media.
func applyStandardSort(media []models.MediaWithDB, fields []SortField, alg string, reverse bool) {
	if len(fields) == 0 {
		return
	}
	sort.SliceStable(media, func(i, j int) bool {
		cmp := CompareSortFields(media, i, j, fields, alg)
		if reverse {
			return cmp > 0
		}
		return cmp < 0
	})
}

// weightedRerank applies MCDA-style weighted re-ranking based on field positions.
// Fields earlier in the list have higher weights (position-based weighting).
func weightedRerank(media []models.MediaWithDB, fields []SortField, reverse bool) {
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
		sort.SliceStable(values, func(i, j int) bool {
			if values[i].isNumeric {
				if field.Reverse {
					return values[i].value > values[j].value
				}
				return values[i].value < values[j].value
			}
			// Use natural comparison for strings
			if field.Reverse {
				return textutil.NaturalLess(values[j].strValue, values[i].strValue)
			}
			return textutil.NaturalLess(values[i].strValue, values[j].strValue)
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
		if reverse {
			return ranked[i].totalScore < ranked[j].totalScore
		}
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

	first := (*media)[0]
	words := extractSearchWordsFromFlagsOrMedia(flags, first)
	if len(words) == 0 {
		return nil
	}

	words = topWordsByLength(words, 50)
	queryStr := buildRelatedMediaQueryStr(words)

	ftsTable := detectFTSTable(ctx, sqlDB)
	if ftsTable == "" {
		return nil
	}

	relatedRows, err := queryRelatedMediaWithRank(ctx, sqlDB, RelatedSearchParams{
		FTSTable:    ftsTable,
		ExcludePath: first.Path,
		QueryStr:    queryStr,
		Limit:       20,
	})
	if err != nil {
		return err
	}

	appendRelatedMedia(media, relatedRows)
	return nil
}

func extractSearchWordsFromFlagsOrMedia(flags models.GlobalFlags, first models.MediaWithDB) []string {
	if len(flags.Search) > 0 {
		return extractWordsFromSearch(flags.Search)
	}
	return extractSearchWords(first)
}

func extractWordsFromSearch(searchTerms []string) []string {
	var words []string
	queryStr := strings.Join(searchTerms, " ")
	hybrid := ParseHybridSearchQuery(queryStr)

	words = append(words, hybrid.FTSTerms...)
	for _, phrase := range hybrid.Phrases {
		for w := range strings.FieldsSeq(phrase) {
			if len(w) > 2 {
				words = append(words, w)
			}
		}
	}
	return words
}

func topWordsByLength(words []string, maxCount int) []string {
	sort.Slice(words, func(i, j int) bool {
		return len(words[i]) > len(words[j])
	})
	if len(words) > maxCount {
		return words[:maxCount]
	}
	return words
}

func buildRelatedMediaQueryStr(words []string) string {
	hybrid := &HybridSearchQuery{FTSTerms: words}
	return hybrid.BuildFTSQuery("OR")
}

func appendRelatedMedia(media *[]models.MediaWithDB, relatedRows []relatedMediaRow) {
	seenPaths := make(map[string]bool, len(*media))
	for _, m := range *media {
		seenPaths[m.Path] = true
	}

	for _, row := range relatedRows {
		if seenPaths[row.Path] {
			continue
		}
		*media = append(*media, relatedRowToMedia(row))
		seenPaths[row.Path] = true
	}
}

func relatedRowToMedia(row relatedMediaRow) models.MediaWithDB {
	return models.MediaWithDB{
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
	}
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
		ORDER BY bm25(%s) ASC, m.path
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
			Media: m.ToModel(),
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
	case "path_tokenized":
		m.PathTokenized = sql.NullString{String: s, Valid: true}
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
	case "fasthash":
		m.Fasthash = sql.NullString{String: s, Valid: true}
	case "sha256":
		m.Sha256 = sql.NullString{String: s, Valid: true}
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
	case "video_count":
		m.VideoCount = sql.NullInt64{Int64: i, Valid: true}
	case "audio_count":
		m.AudioCount = sql.NullInt64{Int64: i, Valid: true}
	case "subtitle_count":
		m.SubtitleCount = sql.NullInt64{Int64: i, Valid: true}
	case "is_deduped":
		m.IsDeduped = sql.NullInt64{Int64: i, Valid: true}
	default:
		return false
	}
	return true
}

func mapFloat64Field(m *db.Media, col string, val any) bool {
	switch col {
	case "score":
		m.Score = sql.NullFloat64{Float64: utils.GetFloat64(val), Valid: true}
	case "fps":
		m.Fps = sql.NullFloat64{Float64: utils.GetFloat64(val), Valid: true}
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

// MediaQuery executes a query against multiple databases concurrently.
//
// The SQL fetch intentionally returns every row matching the SQL predicates;
// ordering and pagination are applied once, in memory, after grouping and
// sibling expansion. This keeps one-database and multi-database results in the
// same order and makes LIMIT/OFFSET select from the final ordering.
func (qe *QueryExecutor) MediaQuery(ctx context.Context, dbs []string) ([]models.MediaWithDB, error) {
	flags := qe.filterBuilder.Flags

	resolvedFlags, err := qe.ResolvePercentileFlags(ctx, dbs, flags)
	if err != nil {
		return nil, err
	}
	flags = resolvedFlags

	if flags.Query != "" {
		fb := NewFilterBuilder(flags)
		query, args := fb.BuildQuery(ctx, "*")
		allMedia, errs := qe.executeMultiDB(ctx, dbs, query, args)
		if len(errs) > 0 {
			return allMedia, errors.Join(errs...)
		}
		return allMedia, nil
	}

	fetchFlags := flags
	fetchFlags.All = true
	fetchFlags.Limit = 0
	fetchFlags.Offset = 0
	fetchFlags.SortBy = ""
	fetchFlags.Random = false
	fetchFlags.NatSort = false

	fb := NewFilterBuilder(fetchFlags)
	query, args := fb.BuildQuery(ctx, "*")

	allMedia, errs := qe.executeMultiDB(ctx, dbs, query, args)
	if len(errs) > 0 {
		return allMedia, errors.Join(errs...)
	}

	// Apply the in-memory predicates too (existence checks and the exact
	// include/exclude semantics) so the result set matches FilterMedia.
	allMedia = NewFilterBuilder(flags).FilterMedia(allMedia)

	if flags.FileCounts != "" {
		allMedia = qe.applyEpisodicFilters(allMedia, flags)
	}

	if flags.GroupByParent {
		allMedia = qe.GroupByParent(allMedia, flags)
		NewSortBuilder(flags).Sort(allMedia)
		return paginateMedia(allMedia, flags), nil
	}

	// Sort after fetching every matching row: random order and --re-rank also
	// see the complete, filtered set.
	NewSortBuilder(flags).Sort(allMedia)

	// Pagination selects the page from the final ordering. Sibling expansion
	// happens for that page, then only the expanded rows are re-ordered.
	page := paginateMedia(allMedia, flags)
	if flags.FetchSiblings != "" {
		siblings, err := qe.FetchSiblings(ctx, page, flags)
		if err != nil {
			if logging.Log != nil {
				logging.Log.Warn("Sibling fetch failed", "error", err)
			}
		} else {
			page = siblings
			NewSortBuilder(flags).Sort(page)
		}
	}

	return page, nil
}

func paginateMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	offset := max(flags.Offset, 0)
	if offset > len(media) {
		return []models.MediaWithDB{}
	}
	media = media[offset:]

	if flags.All || flags.Limit <= 0 {
		return media
	}
	if len(media) > flags.Limit {
		media = media[:flags.Limit]
	}
	return media
}

func (qe *QueryExecutor) applyEpisodicFilters(
	allMedia []models.MediaWithDB,
	flags models.GlobalFlags,
) []models.MediaWithDB {
	counts := make(map[string]int64)
	for _, m := range allMedia {
		counts[m.Parent()]++
	}

	r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
		return strconv.ParseInt(s, 10, 64)
	})
	if err != nil {
		return allMedia
	}

	filtered := make([]models.MediaWithDB, 0, len(allMedia))
	for _, m := range allMedia {
		if r.Matches(counts[m.Parent()]) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// needsPostProcessedCount reports whether the count must be derived from the
// same post-processed result set as the response.
func needsPostProcessedCount(flags models.GlobalFlags) bool {
	if flags.GroupByParent || flags.FetchSiblings != "" || flags.FileCounts != "" || flags.Exists {
		return true
	}
	if flags.Related > 0 || strings.Contains(flags.PlayInOrder, "_related_media") ||
		strings.Contains(flags.SortBy, "_related_media") {
		return true
	}
	return false
}

// MediaQueryCount executes a count query against multiple databases concurrently.
func (qe *QueryExecutor) MediaQueryCount(ctx context.Context, dbs []string) (int64, error) {
	flags := qe.filterBuilder.Flags

	if needsPostProcessedCount(flags) {
		tempFlags := flags
		tempFlags.All = true
		tempFlags.Limit = 0
		tempFlags.Offset = 0

		allMedia, err := NewQueryExecutor(tempFlags).MediaQuery(ctx, dbs)
		if err != nil {
			return 0, err
		}
		return int64(len(allMedia)), nil
	}

	resolvedFlags, err := qe.ResolvePercentileFlags(ctx, dbs, flags)
	if err != nil {
		return 0, err
	}

	query, args := NewFilterBuilder(resolvedFlags).BuildCount(ctx)

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
		RepresentativePath      string  `json:"representative_path"`
		RepresentativeMediaType *string `json:"media_type,omitempty"`
		DB                      string  `json:"db"`
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
				DB:                      m.DB,
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
			DB:            g.DB,
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
		if limit <= 0 {
			limit = 2000
		}
		return limit, true
	case "each":
		if limit <= 0 {
			limit = -1 // SQLite: no limit
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
		if textutil.IsDigit(flags.FetchSiblings) {
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
	flags models.GlobalFlags,
) ([]models.MediaWithDB, error) {
	query := "SELECT path, path_tokenized, title, duration, size, time_created, time_modified, time_deleted, time_first_played, time_last_played, play_count, playhead, media_type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs, video_count, audio_count, subtitle_count, album, artist, genre, categories, description, language, time_downloaded, score, fasthash, sha256, is_deduped FROM media WHERE "
	args := make([]any, 0, 2)
	if flags.OnlyDeleted {
		query += "COALESCE(time_deleted, 0) > 0"
	} else if flags.HideDeleted {
		query += "COALESCE(time_deleted, 0) = 0"
	} else {
		query += "1 = 1"
	}
	query += " AND path LIKE ? ORDER BY path LIMIT ?"
	pattern := dir + "%"
	args = append(args, pattern, limit)
	return QueryDatabase(ctx, dbPath, query, args)
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
	return qe.querySiblings(ctx, dbPath, dir, limit, flags)
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

	NewSortBuilder(flags).Sort(allSiblings)
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
	tempFlags.Offset = 0
	// Percentile sampling only needs the column values; ordering is irrelevant
	// and may reference columns that the caller's sort does not need.
	tempFlags.SortBy = ""
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
			if err := rows.Scan(&p); err != nil {
				return nil, err
			}
			gCounts[filepath.Dir(p)]++
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		for _, c := range gCounts {
			values = append(values, c)
		}
		return values, nil
	}

	for rows.Next() {
		var v sql.NullInt64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid {
			values = append(values, v.Int64)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (qe *QueryExecutor) getPercentileValues(
	ctx context.Context,
	dbs []string,
	flags models.GlobalFlags,
	field string,
) ([]int64, error) {
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
	var errs []error
	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			localValues, err := qe.queryPercentileValues(ctx, path, sqlQuery, args, field)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", path, err))
				return
			}
			values = append(values, localValues...)
		}(dbPath)
	}
	wg.Wait()
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return values, nil
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

type TimePercentileOptions struct {
	dbs           []string
	flags         *models.GlobalFlags
	field         string
	ranges        []string
	after, before *string
}

func (qe *QueryExecutor) resolveTimePercentiles(
	ctx context.Context,
	opts TimePercentileOptions,
) error {
	values, err := qe.getPercentileValues(ctx, opts.dbs, *opts.flags, opts.field)
	if err != nil {
		return err
	}
	mapping := utils.CalculatePercentiles(values)
	for _, r := range opts.ranges {
		if pmin, pmax, ok := utils.ParsePercentileRange(r); ok {
			*opts.after = strconv.FormatInt(mapping[int(pmin)], 10)
			*opts.before = strconv.FormatInt(mapping[int(pmax)], 10)
		}
	}
	return nil
}

func (qe *QueryExecutor) resolveSizePercentiles(ctx context.Context, dbs []string, flags *models.GlobalFlags) error {
	values, err := qe.getPercentileValues(ctx, dbs, *flags, "size")
	if err != nil {
		return err
	}
	mapping := utils.CalculatePercentiles(values)
	flags.Size = qe.applyPercentileToRange(flags.Size, mapping)
	return nil
}

func (qe *QueryExecutor) resolveDurationPercentiles(
	ctx context.Context,
	dbs []string,
	flags *models.GlobalFlags,
) error {
	values, err := qe.getPercentileValues(ctx, dbs, *flags, "duration")
	if err != nil {
		return err
	}
	mapping := utils.CalculatePercentiles(values)
	flags.Duration = qe.applyPercentileToRange(flags.Duration, mapping)
	return nil
}

func (qe *QueryExecutor) resolveEpisodePercentiles(ctx context.Context, dbs []string, flags *models.GlobalFlags) error {
	values, err := qe.getPercentileValues(ctx, dbs, *flags, "episodes")
	if err != nil {
		return err
	}
	mapping := utils.CalculatePercentiles(values)
	if pmin, pmax, ok := utils.ParsePercentileRange(flags.FileCounts); ok {
		minVal := mapping[int(pmin)]
		maxVal := mapping[int(pmax)]
		flags.FileCounts = fmt.Sprintf("+%d,-%d", minVal, maxVal)
	}
	return nil
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

func validatePercentileFlags(flags models.GlobalFlags) error {
	groups := [][]string{
		flags.Size, flags.Duration, flags.Modified, flags.Created, flags.Downloaded,
	}
	for _, ranges := range groups {
		for _, s := range ranges {
			if err := utils.ValidatePercentileRange(s); err != nil {
				return err
			}
		}
	}
	return utils.ValidatePercentileRange(flags.FileCounts)
}

func (qe *QueryExecutor) resolveAllPercentiles(ctx context.Context, dbs []string, flags *models.GlobalFlags) error {
	if err := qe.resolveSizePercentiles(ctx, dbs, flags); err != nil {
		return err
	}
	if err := qe.resolveDurationPercentiles(ctx, dbs, flags); err != nil {
		return err
	}

	if err := qe.resolveTimePercentiles(ctx, TimePercentileOptions{
		dbs: dbs, flags: flags, field: "time_modified",
		ranges: flags.Modified, after: &flags.ModifiedAfter, before: &flags.ModifiedBefore,
	}); err != nil {
		return err
	}
	if err := qe.resolveTimePercentiles(ctx, TimePercentileOptions{
		dbs: dbs, flags: flags, field: "time_created",
		ranges: flags.Created, after: &flags.CreatedAfter, before: &flags.CreatedBefore,
	}); err != nil {
		return err
	}
	if err := qe.resolveTimePercentiles(ctx, TimePercentileOptions{
		dbs: dbs, flags: flags, field: "time_downloaded",
		ranges: flags.Downloaded, after: &flags.DownloadedAfter, before: &flags.DownloadedBefore,
	}); err != nil {
		return err
	}

	return qe.resolveEpisodePercentiles(ctx, dbs, flags)
}

func (qe *QueryExecutor) ResolvePercentileFlags(
	ctx context.Context,
	dbs []string,
	flags models.GlobalFlags,
) (models.GlobalFlags, error) {
	if err := validatePercentileFlags(flags); err != nil {
		return flags, err
	}
	if !qe.hasAnyPercentileFlag(flags) {
		return flags, nil
	}

	return flags, qe.resolveAllPercentiles(ctx, dbs, &flags)
}
