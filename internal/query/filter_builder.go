package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// FilterBuilder constructs SQL queries and in-memory filters from flags
// This is the single source of truth for all filter logic
type FilterBuilder struct {
	flags models.GlobalFlags
}

// NewFilterBuilder creates a new FilterBuilder from global flags
func NewFilterBuilder(flags models.GlobalFlags) *FilterBuilder {
	return &FilterBuilder{flags: flags}
}

// BuildWhereClauses builds WHERE clauses and arguments for SQL queries
// Order matters for performance: selective indexed filters come before expensive substring searches
// Based on benchmark: indexed equality (~277μs) << LIKE prefix (~500μs) << LIKE substring (~1ms)
func (fb *FilterBuilder) BuildWhereClauses() ([]string, []any) {
	var whereClauses []string
	var args []any

	// === PHASE 1: Highly selective indexed equality filters (fastest ~277μs) ===

	// Deleted status (indexed column)
	if fb.flags.OnlyDeleted {
		whereClauses = append(whereClauses, "COALESCE(time_deleted, 0) > 0")
	} else if fb.flags.HideDeleted {
		whereClauses = append(whereClauses, "COALESCE(time_deleted, 0) = 0")
	}

	// Content type filters (indexed column - should come before expensive searches)
	var typeClauses []string
	if fb.flags.VideoOnly {
		typeClauses = append(typeClauses, "type = 'video'")
	}
	if fb.flags.AudioOnly {
		typeClauses = append(typeClauses, "type = 'audio'", "type = 'audiobook'")
	}
	if fb.flags.ImageOnly {
		typeClauses = append(typeClauses, "type = 'image'")
	}
	if fb.flags.TextOnly {
		typeClauses = append(typeClauses, "type = 'text'")
	}
	if len(typeClauses) > 0 {
		whereClauses = append(whereClauses, "("+strings.Join(typeClauses, " OR ")+")")
	}

	// Genre filter (equality match)
	if fb.flags.Genre != "" {
		whereClauses = append(whereClauses, "genre = ?")
		args = append(args, fb.flags.Genre)
	}

	// Language filter (equality match)
	if len(fb.flags.Language) > 0 {
		var langClauses []string
		for _, lang := range fb.flags.Language {
			langClauses = append(langClauses, "language = ?")
			args = append(args, lang)
		}
		if len(langClauses) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(langClauses, " OR ")+")")
		}
	}

	// Exact path filters (IN clause - very selective)
	if len(fb.flags.Paths) > 0 {
		var inPaths []string
		for _, p := range fb.flags.Paths {
			if strings.Contains(p, "%") {
				whereClauses = append(whereClauses, "path LIKE ?")
				args = append(args, p)
			} else {
				inPaths = append(inPaths, p)
			}
		}
		if len(inPaths) > 0 {
			placeholders := make([]string, len(inPaths))
			for i := range inPaths {
				placeholders[i] = "?"
				args = append(args, inPaths[i])
			}
			whereClauses = append(whereClauses, fmt.Sprintf("path IN (%s)", strings.Join(placeholders, ", ")))
		}
	}

	// Size filters (indexed column)
	for _, s := range fb.flags.Size {
		if r, err := utils.ParseRange(s, utils.HumanToBytes); err == nil {
			if r.Value != nil {
				whereClauses = append(whereClauses, "size = ?")
				args = append(args, *r.Value)
			}
			if r.Min != nil {
				whereClauses = append(whereClauses, "size >= ?")
				args = append(args, *r.Min)
			}
			if r.Max != nil {
				whereClauses = append(whereClauses, "size <= ?")
				args = append(args, *r.Max)
			}
		}
	}

	// Duration filters (indexed column)
	for _, s := range fb.flags.Duration {
		if r, err := utils.ParseRange(s, utils.HumanToSeconds); err == nil {
			if r.Value != nil {
				whereClauses = append(whereClauses, "duration = ?")
				args = append(args, *r.Value)
			}
			if r.Min != nil {
				whereClauses = append(whereClauses, "duration >= ?")
				args = append(args, *r.Min)
			}
			if r.Max != nil {
				whereClauses = append(whereClauses, "duration <= ?")
				args = append(args, *r.Max)
			}
		}
	}

	// Play count filters (indexed column)
	if fb.flags.PlayCountMin > 0 {
		whereClauses = append(whereClauses, "play_count >= ?")
		args = append(args, fb.flags.PlayCountMin)
	}
	if fb.flags.PlayCountMax > 0 {
		whereClauses = append(whereClauses, "play_count <= ?")
		args = append(args, fb.flags.PlayCountMax)
	}

	// === PHASE 2: Time-based filters (indexed columns) ===

	if fb.flags.DeletedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.DeletedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_deleted >= ?")
			args = append(args, ts)
		}
	}
	if fb.flags.DeletedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.DeletedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_deleted <= ?")
			args = append(args, ts)
		}
	}
	if fb.flags.CreatedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.CreatedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_created >= ?")
			args = append(args, ts)
		}
	}
	if fb.flags.CreatedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.CreatedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_created <= ?")
			args = append(args, ts)
		}
	}
	if fb.flags.ModifiedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.ModifiedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_modified >= ?")
			args = append(args, ts)
		}
	}
	if fb.flags.ModifiedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.ModifiedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_modified <= ?")
			args = append(args, ts)
		}
	}
	if fb.flags.PlayedAfter != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.PlayedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_last_played >= ?")
			args = append(args, ts)
		}
	}
	if fb.flags.PlayedBefore != "" {
		if ts := utils.ParseDateOrRelative(fb.flags.PlayedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_last_played <= ?")
			args = append(args, ts)
		}
	}

	// Watched/unwatched status (indexed via time_last_played)
	if fb.flags.Watched != nil {
		if *fb.flags.Watched {
			whereClauses = append(whereClauses, "time_last_played > 0")
		} else {
			whereClauses = append(whereClauses, "COALESCE(time_last_played, 0) = 0")
		}
	}

	// Playhead/playback status
	if fb.flags.Unfinished || fb.flags.InProgress {
		whereClauses = append(whereClauses, "COALESCE(playhead, 0) > 0")
	}
	if fb.flags.Partial != "" {
		if strings.Contains(fb.flags.Partial, "s") {
			whereClauses = append(whereClauses, "COALESCE(time_first_played, 0) = 0")
		} else {
			whereClauses = append(whereClauses, "time_first_played > 0")
		}
	}
	if fb.flags.Completed {
		whereClauses = append(whereClauses, "COALESCE(play_count, 0) > 0")
	}

	// === PHASE 3: LIKE prefix searches (can use index ~500μs) ===

	if fb.flags.OnlineMediaOnly {
		whereClauses = append(whereClauses, "path LIKE 'http%'")
	}
	if fb.flags.LocalMediaOnly {
		whereClauses = append(whereClauses, "path NOT LIKE 'http%'")
	}

	// Extension filters (EndsWith pattern - surprisingly fast ~660μs per benchmark)
	if len(fb.flags.Ext) > 0 {
		var extClauses []string
		for _, ext := range fb.flags.Ext {
			extClauses = append(extClauses, "path LIKE ?")
			args = append(args, "%"+ext)
		}
		whereClauses = append(whereClauses, "("+strings.Join(extClauses, " OR ")+")")
	}

	// === PHASE 4: Expensive substring searches (full table scan ~1ms+) ===

	// Category filter (LIKE with wildcards - expensive)
	if len(fb.flags.Category) > 0 {
		var catClauses []string
		for _, cat := range fb.flags.Category {
			if cat == "Uncategorized" {
				catClauses = append(catClauses, "(categories IS NULL OR categories = '')")
			} else {
				catClauses = append(catClauses, "categories LIKE '%' || ? || '%'")
				args = append(args, ";"+cat+";")
			}
		}
		if len(catClauses) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(catClauses, " OR ")+")")
		}
	}

	// Search terms (FTS or LIKE - most expensive operation)
	allInclude := append([]string{}, fb.flags.Search...)
	allInclude = append(allInclude, fb.flags.Include...)

	// Path contains filters
	pathContains := append([]string{}, fb.flags.PathContains...)

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
		if fb.flags.FlexibleSearch {
			joinOp = " OR "
		}

		// Determine search mode: --no-fts > --bleve > --fts > auto-detect
		useBleve := fb.flags.Bleve
		useFTS := fb.flags.FTS
		noFTS := fb.flags.NoFTS

		// Auto-detect if not explicitly set
		if !useBleve && !useFTS && !noFTS {
			mode := DetectSearchMode(nil) // Will check bleve global state
			useBleve = (mode == SearchModeBleve)
			useFTS = (mode == SearchModeFTS5)
		}

		if noFTS {
			// Force substring search
			useBleve = false
			useFTS = false
		}

		if useBleve {
			// Bleve search - handled separately, just mark for in-memory filtering
			// Bleve search is executed before SQL query to get matching IDs
		} else if useFTS {
			// Hybrid FTS + LIKE search for phrase support with detail=none
			// Combine all terms into a single query string for parsing
			queryStr := strings.Join(allInclude, " ")
			hybrid := utils.ParseHybridSearchQuery(queryStr)

			// FTS terms (works with detail=none)
			if hybrid.HasFTSTerms() {
				ftsQuery := hybrid.BuildFTSQuery(joinOp)
				if ftsQuery != "" {
					whereClauses = append(whereClauses, fmt.Sprintf("%s MATCH ?", fb.getFTSTable()))
					args = append(args, ftsQuery)
				}
			}

			// Phrase searches via LIKE (trigram-optimized)
			for _, phrase := range hybrid.Phrases {
				whereClauses = append(whereClauses, "(path LIKE ? OR title LIKE ? OR description LIKE ?)")
				pattern := "%" + phrase + "%"
				args = append(args, pattern, pattern, pattern)
			}
		} else {
			// Regular LIKE search
			var searchParts []string
			for _, term := range allInclude {
				searchParts = append(searchParts, "(path LIKE ? OR title LIKE ?)")
				pattern := term
				if !fb.flags.Exact {
					pattern = "%" + strings.ReplaceAll(term, " ", "%") + "%"
				}
				args = append(args, pattern, pattern)
			}
			whereClauses = append(whereClauses, "("+strings.Join(searchParts, joinOp)+")")
		}
	}

	// Exclude patterns (expensive NOT LIKE)
	for _, exc := range fb.flags.Exclude {
		whereClauses = append(whereClauses, "path NOT LIKE ? AND title NOT LIKE ?")
		pattern := "%" + exc + "%"
		args = append(args, pattern, pattern)
	}

	// Regex filter (requires regex extension or post-filter)
	if fb.flags.Regex != "" {
		whereClauses = append(whereClauses, "path REGEXP ?")
		args = append(args, fb.flags.Regex)
	}

	// Path contains filters (substring search - expensive)
	for _, contain := range pathContains {
		whereClauses = append(whereClauses, "path LIKE ?")
		args = append(args, "%"+contain+"%")
	}

	// === PHASE 5: Other filters ===

	if fb.flags.Portrait {
		whereClauses = append(whereClauses, "width < height")
	}

	if fb.flags.WithCaptions {
		whereClauses = append(whereClauses, "path IN (SELECT DISTINCT media_path FROM captions)")
	}

	// Custom WHERE clauses
	whereClauses = append(whereClauses, fb.flags.Where...)

	if fb.flags.DurationFromSize != "" {
		if r, err := utils.ParseRange(fb.flags.DurationFromSize, utils.HumanToBytes); err == nil {
			var subWhere []string
			var subArgs []any
			if r.Value != nil {
				subWhere = append(subWhere, "size = ?")
				subArgs = append(subArgs, *r.Value)
			}
			if r.Min != nil {
				subWhere = append(subWhere, "size >= ?")
				subArgs = append(subArgs, *r.Min)
			}
			if r.Max != nil {
				subWhere = append(subWhere, "size <= ?")
				subArgs = append(subArgs, *r.Max)
			}

			if len(subWhere) > 0 {
				whereClauses = append(whereClauses, fmt.Sprintf("size IS NOT NULL AND duration IN (SELECT DISTINCT duration FROM media WHERE %s)", strings.Join(subWhere, " AND ")))
				args = append(args, subArgs...)
			}
		}
	}

	return whereClauses, args
}

// BuildQuery constructs a complete SQL query with the given columns
func (fb *FilterBuilder) BuildQuery(columns string) (string, []any) {
	// If raw query provided, use it
	if fb.flags.Query != "" {
		if columns == "COUNT(*)" {
			return "SELECT COUNT(*) FROM (" + fb.flags.Query + ")", nil
		}
		return fb.flags.Query, nil
	}

	whereClauses, args := fb.BuildWhereClauses()

	// Base table
	table := "media"
	useFTSJoin := fb.flags.FTS && fb.hasSearchTerms()

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
	if !fb.flags.Random && !fb.flags.NatSort && fb.flags.SortBy != "" {
		sortExpr := fb.OverrideSort(fb.flags.SortBy)
		order := "ASC"
		if fb.flags.Reverse {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", sortExpr, order)
	} else if fb.flags.Random {
		// Optimization for large databases: select rowids randomly first
		if !fb.flags.All && !fb.flags.FTS && !fb.hasSearchTerms() && fb.flags.Limit > 0 {
			whereNotDeleted := "WHERE COALESCE(time_deleted, 0) = 0"
			if fb.flags.OnlyDeleted {
				whereNotDeleted = "WHERE COALESCE(time_deleted, 0) > 0"
			}
			// We use a larger pool for random selection then limit it in the outer query
			randomLimit := fb.flags.Limit * 16

			randomSubquery := fmt.Sprintf("rowid IN (SELECT rowid FROM media %s ORDER BY RANDOM() LIMIT %d)", whereNotDeleted, randomLimit)
			if strings.Contains(query, " WHERE ") {
				query += " AND " + randomSubquery
			} else {
				query += " WHERE " + randomSubquery
			}
		}
		query += " ORDER BY RANDOM()"
	}

	// Limit and offset
	if !fb.flags.All && fb.flags.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", fb.flags.Limit)
	}
	if fb.flags.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", fb.flags.Offset)
	}

	return query, args
}

// OverrideSort translates logical sort fields into SQL expressions
func (fb *FilterBuilder) OverrideSort(s string) string {
	yearMonthSQL := func(v string) string {
		return fmt.Sprintf("cast(strftime('%%Y%%m', datetime(%s, 'unixepoch')) as int)", v)
	}
	yearMonthDaySQL := func(v string) string {
		return fmt.Sprintf("cast(strftime('%%Y%%m%%d', datetime(%s, 'unixepoch')) as int)", v)
	}

	s = strings.ReplaceAll(s, "month_created", yearMonthSQL("time_created"))
	s = strings.ReplaceAll(s, "month_modified", yearMonthSQL("time_modified"))
	s = strings.ReplaceAll(s, "date_created", yearMonthDaySQL("time_created"))
	s = strings.ReplaceAll(s, "date_modified", yearMonthDaySQL("time_modified"))
	s = strings.ReplaceAll(s, "time_deleted", "COALESCE(time_deleted, 0)")

	progressExpr := "CAST(COALESCE(playhead, 0) AS FLOAT) / CAST(COALESCE(duration, 1) AS FLOAT)"
	s = strings.ReplaceAll(s, "progress", fmt.Sprintf("(%s = 0), %s", progressExpr, progressExpr))

	s = strings.ReplaceAll(s, "play_count", "(COALESCE(play_count, 0) = 0), play_count")
	s = strings.ReplaceAll(s, "time_last_played", "(COALESCE(time_last_played, 0) = 0), time_last_played")

	s = strings.ReplaceAll(s, "type", "LOWER(type)")
	s = strings.ReplaceAll(s, "random()", "RANDOM()")
	s = strings.ReplaceAll(s, "random", "RANDOM()")
	s = strings.ReplaceAll(s, "default", "play_count, playhead DESC, time_last_played, duration DESC, size DESC, title IS NOT NULL DESC, path")
	s = strings.ReplaceAll(s, "priorityfast", "ntile(1000) over (order by size) desc, duration")
	s = strings.ReplaceAll(s, "priority", "ntile(1000) over (order by size/duration) desc")
	s = strings.ReplaceAll(s, "bitrate", "size/duration")
	s = strings.ReplaceAll(s, "time_scanned", "COALESCE(time_downloaded, 0)")
	s = strings.ReplaceAll(s, "time_downloaded", "COALESCE(time_downloaded, 0)")

	return s
}

// BuildSelect is an alias for BuildQuery for backward compatibility
func (fb *FilterBuilder) BuildSelect(columns string) (string, []any) {
	return fb.BuildQuery(columns)
}

// BuildCount builds a count query
func (fb *FilterBuilder) BuildCount() (string, []any) {
	return fb.BuildQuery("COUNT(*)")
}

// hasSearchTerms checks if there are any search/include terms
func (fb *FilterBuilder) hasSearchTerms() bool {
	allInclude := append([]string{}, fb.flags.Search...)
	allInclude = append(allInclude, fb.flags.Include...)
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
	if fb.flags.FTSTable != "" {
		return fb.flags.FTSTable
	}
	return "media_fts"
}

// CreateInMemoryFilter creates a function that can filter media in memory
func (fb *FilterBuilder) CreateInMemoryFilter() func(models.MediaWithDB) bool {
	// Pre-compile regex if needed
	var regex *regexp.Regexp
	if fb.flags.Regex != "" {
		regex = regexp.MustCompile(fb.flags.Regex)
	}

	// Pre-parse size ranges
	var sizeRanges []utils.Range
	for _, s := range fb.flags.Size {
		if r, err := utils.ParseRange(s, utils.HumanToBytes); err == nil {
			sizeRanges = append(sizeRanges, r)
		}
	}

	// Pre-parse duration ranges
	var durationRanges []utils.Range
	for _, s := range fb.flags.Duration {
		if r, err := utils.ParseRange(s, utils.HumanToSeconds); err == nil {
			durationRanges = append(durationRanges, r)
		}
	}

	return func(m models.MediaWithDB) bool {
		// Check existence
		if fb.flags.Exists && !utils.FileExists(m.Path) {
			return false
		}

		// Include/exclude patterns
		if len(fb.flags.Include) > 0 && !utils.MatchesAny(m.Path, fb.flags.Include) {
			return false
		}
		if len(fb.flags.Exclude) > 0 && utils.MatchesAny(m.Path, fb.flags.Exclude) {
			return false
		}

		// Path contains
		for _, contain := range fb.flags.PathContains {
			if !strings.Contains(m.Path, contain) {
				return false
			}
		}

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

		// Extension filters
		if len(fb.flags.Ext) > 0 {
			matched := false
			fileExt := strings.ToLower(filepath.Ext(m.Path))
			for _, ext := range fb.flags.Ext {
				if fileExt == strings.ToLower(ext) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}

		// Regex filter
		if regex != nil && !regex.MatchString(m.Path) {
			return false
		}

		// Mimetype filters
		if len(fb.flags.MimeType) > 0 {
			match := false
			if m.Type != nil && utils.IsMimeMatch(fb.flags.MimeType, *m.Type) {
				match = true
			}
			if !match {
				return false
			}
		}
		if len(fb.flags.NoMimeType) > 0 {
			if m.Type != nil && utils.IsMimeMatch(fb.flags.NoMimeType, *m.Type) {
				return false
			}
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
	flags models.GlobalFlags
}

func NewSortBuilder(flags models.GlobalFlags) *SortBuilder {
	return &SortBuilder{flags: flags}
}

func (sb *SortBuilder) Sort(media []models.MediaWithDB) {
	if sb.flags.Random {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(media), func(i, j int) {
			media[i], media[j] = media[j], media[i]
		})
		return
	}

	if sb.flags.NoPlayInOrder {
		sb.SortBasic(media)
		return
	}

	// If the user explicitly requested a specific sort field (and it's not the default "path" or "default"),
	// respect it and use basic sorting - this takes precedence over PlayInOrder
	if sb.flags.SortBy != "" && sb.flags.SortBy != "path" && sb.flags.SortBy != "default" {
		sb.SortBasic(media)
		return
	}

	// If the user explicitly requested "default", use xklb sorting (with optional reverse)
	if sb.flags.SortBy == "default" {
		if sb.flags.Reverse {
			sb.SortAdvanced(media, "reverse_xklb")
		} else {
			sb.SortAdvanced(media, "xklb")
		}
		return
	}

	// If PlayInOrder is explicitly set (and SortBy is default), use it
	if sb.flags.PlayInOrder != "" {
		if sb.flags.Reverse {
			// Prepend "reverse_" to the PlayInOrder config
			sb.SortAdvanced(media, "reverse_"+sb.flags.PlayInOrder)
		} else {
			sb.SortAdvanced(media, sb.flags.PlayInOrder)
		}
		return
	}

	// If SortBy is "path" (the default) with Reverse or NatSort flags, use basic sorting
	if sb.flags.Reverse || sb.flags.NatSort {
		sb.SortBasic(media)
		return
	}

	// Fall back to xklb default sorting when SortBy is "path" (the default)
	// This provides xklb-style sorting as the default behavior
	if sb.flags.SortBy == "path" || sb.flags.SortBy == "" {
		sb.SortAdvanced(media, "xklb")
		return
	}

	sb.SortBasic(media)
}

func (sb *SortBuilder) SortBasic(media []models.MediaWithDB) {
	sortBy := sb.flags.SortBy
	reverse := sb.flags.Reverse
	natSort := sb.flags.NatSort

	// Special handling for sparse fields where we want 0/nulls at the bottom always
	if sortBy == "play_count" || sortBy == "time_last_played" || sortBy == "progress" {
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
		return
	}

	less := func(i, j int) bool {
		switch sortBy {
		case "path":
			if natSort {
				return utils.NaturalLess(media[i].Path, media[j].Path)
			}
			return media[i].Path < media[j].Path
		case "title":
			return utils.StringValue(media[i].Title) < utils.StringValue(media[j].Title)
		case "duration":
			iNil := media[i].Duration == nil
			jNil := media[j].Duration == nil
			if iNil && jNil {
				return false
			}
			if iNil {
				return !reverse
			}
			if jNil {
				return reverse
			}
			return utils.Int64Value(media[i].Duration) < utils.Int64Value(media[j].Duration)
		case "size":
			return utils.Int64Value(media[i].Size) < utils.Int64Value(media[j].Size)
		case "bitrate", "priority":
			d1 := utils.Int64Value(media[i].Duration)
			d2 := utils.Int64Value(media[j].Duration)
			if d1 == 0 || d2 == 0 {
				return false
			}
			return float64(utils.Int64Value(media[i].Size))/float64(d1) < float64(utils.Int64Value(media[j].Size))/float64(d2)
		case "priorityfast":
			if utils.Int64Value(media[i].Size) != utils.Int64Value(media[j].Size) {
				return utils.Int64Value(media[i].Size) > utils.Int64Value(media[j].Size)
			}
			return utils.Int64Value(media[i].Duration) < utils.Int64Value(media[j].Duration)
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
		case "type":
			iNil := media[i].Type == nil || *media[i].Type == ""
			jNil := media[j].Type == nil || *media[j].Type == ""
			if iNil && jNil {
				return false
			}
			if iNil {
				return !reverse
			}
			if jNil {
				return reverse
			}
			return utils.StringValue(media[i].Type) < utils.StringValue(media[j].Type)
		case "extension":
			return strings.ToLower(filepath.Ext(media[i].Path)) < strings.ToLower(filepath.Ext(media[j].Path))
		default:
			return utils.NaturalLess(media[i].Path, media[j].Path)
		}
	}

	if reverse {
		sort.SliceStable(media, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.SliceStable(media, less)
	}
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

// parseSortConfig parses a sort configuration string into SortField slices
// Supports:
//   - Simple: "path", "title"
//   - Prefixed: "natural_path", "python_title"
//   - Reversed: "-path", "reverse_path"
//   - Multi-field: "video_count desc,audio_count desc,path asc"
//   - Complex: "natural_path,title desc"
//   - Array notation: "field1,field2,field3" (comma-separated)
//   - Meta-field markers: "_weighted_rerank", "_natural_order" (apply to fields below)
func parseSortConfig(config string) ([]SortField, string) {
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

// parseSortConfigWithGroups parses a sort configuration string into SortGroup slices
// Meta-field markers (_weighted_rerank, _natural_order) create new groups that apply to fields below them
func parseSortConfigWithGroups(config string) []SortGroup {
	if config == "" {
		return []SortGroup{{Fields: []SortField{{Field: "ps", Reverse: false}}, Alg: "natural"}}
	}

	var groups []SortGroup
	var currentGroup SortGroup
	currentAlg := "natural"

	// Known algorithms for prefix detection
	knownAlgs := map[string]bool{
		"natural": true, "ignorecase": true,
		"lowercase": true, "human": true, "locale": true,
		"signed": true, "os": true, "python": true,
	}

	parts := strings.SplitSeq(config, ",")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for meta-field markers
		if part == "_weighted_rerank" {
			// Save current group if it has fields
			if len(currentGroup.Fields) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = SortGroup{}
			}
			// Start new weighted group
			currentGroup = SortGroup{Fields: []SortField{}, Alg: "weighted"}
			continue
		}

		if part == "_natural_order" {
			// Save current group if it has fields
			if len(currentGroup.Fields) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = SortGroup{}
			}
			// Start new natural order group
			currentGroup = SortGroup{Fields: []SortField{}, Alg: "natural"}
			continue
		}

		if part == "_related_media" {
			// Save current group if it has fields
			if len(currentGroup.Fields) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = SortGroup{}
			}
			// Start new related media group
			currentGroup = SortGroup{Fields: []SortField{}, Alg: "related"}
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
				currentAlg = potentialParts[0]
				field = potentialParts[1]
			}
		}

		// Add field to current group
		if currentGroup.Alg == "" {
			currentGroup.Alg = currentAlg
		}
		currentGroup.Fields = append(currentGroup.Fields, SortField{Field: field, Reverse: reverse})
	}

	// Add final group if it has fields
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

// getSortValueFloat64 returns a numeric sort value for a field
func getSortValueFloat64(m models.MediaWithDB, field string) float64 {
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

// getSortValueString returns a string sort value for a field
func getSortValueString(m models.MediaWithDB, field string) string {
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
	case "type":
		return utils.StringValue(m.Type)
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

// hasField checks if a field is numeric (float64) or string
func isNumericField(field string) bool {
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

// xklbDefaultSort returns the xklb-style default sort fields
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
func xklbDefaultSort() []SortField {
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

// duDefaultSort returns the default sort for DU mode
// lambda x: ((size/count), size, count, folders, reverse(path))
func duDefaultSort() []SortField {
	return []SortField{
		{Field: "size_per_count", Reverse: true}, // size/count desc
		{Field: "size", Reverse: true},
		{Field: "count", Reverse: true},
		{Field: "folders", Reverse: true},
		{Field: "path", Reverse: true}, // reverse path
	}
}

// compareSortFields compares two media items using multiple sort fields
func compareSortFields(media []models.MediaWithDB, i, j int, sortFields []SortField, alg string) int {
	for _, sf := range sortFields {
		var cmp int

		if isNumericField(sf.Field) {
			// Special handling for computed fields
			var valI, valJ float64

			if sf.Field == "path_is_remote" {
				// 1 if remote (http), 0 if local
				pathI := media[i].Path
				pathJ := media[j].Path
				valI = 0
				valJ = 0
				if strings.HasPrefix(pathI, "http") {
					valI = 1
				}
				if strings.HasPrefix(pathJ, "http") {
					valJ = 1
				}
			} else if sf.Field == "title_is_null" {
				// 1 if null/empty, 0 if has title
				titleI := media[i].Title
				titleJ := media[j].Title
				valI = 0
				valJ = 0
				if titleI == nil || *titleI == "" {
					valI = 1
				}
				if titleJ == nil || *titleJ == "" {
					valJ = 1
				}
			} else if sf.Field == "size_per_count" {
				// size / count (for folder stats)
				sizeI := float64(utils.Int64Value(media[i].Size))
				sizeJ := float64(utils.Int64Value(media[j].Size))
				countI := float64(utils.Int64Value(media[i].Size)) // fallback to size if count not available
				countJ := float64(utils.Int64Value(media[j].Size))
				// For media items, just use size
				valI = sizeI / utils.Max(1.0, countI)
				valJ = sizeJ / utils.Max(1.0, countJ)
			} else {
				valI = getSortValueFloat64(media[i], sf.Field)
				valJ = getSortValueFloat64(media[j], sf.Field)
			}

			if valI < valJ {
				cmp = -1
			} else if valI > valJ {
				cmp = 1
			} else {
				cmp = 0
			}
		} else {
			// String comparison
			valI := getSortValueString(media[i], sf.Field)
			valJ := getSortValueString(media[j], sf.Field)

			var res bool
			if alg == "python" {
				res = valI < valJ
			} else {
				res = utils.NaturalLess(valI, valJ)
			}

			if valI == valJ {
				cmp = 0
			} else if res {
				cmp = -1
			} else {
				cmp = 1
			}
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
	if config == "xklb" || config == "xklb_default" {
		config = "video_count desc,audio_count desc,path_is_remote asc,subtitle_count desc,play_count asc,playhead desc,time_last_played asc,title_is_null asc,path asc"
	} else if config == "reverse_xklb" || config == "reverse_xklb_default" {
		// Reverse of xklb: prioritize audio, remote URLs, no subtitles, most played, etc.
		config = "video_count asc,audio_count asc,path_is_remote desc,subtitle_count asc,play_count desc,playhead asc,time_last_played desc,title_is_null desc,path desc"
	} else if config == "du" || config == "du_default" {
		config = "size_per_count desc,size desc,count desc,folders desc,path desc"
	} else if config == "reverse_du" || config == "reverse_du_default" {
		// Reverse of du: smallest folders first, alphabetical path
		config = "size_per_count asc,size asc,count asc,folders asc,path asc"
	}

	// Check for meta-field markers and use grouped sorting
	groups := parseSortConfigWithGroups(config)

	if len(groups) > 1 {
		// Multiple groups - apply each group's sorting in sequence
		for _, group := range groups {
			if group.Alg == "weighted" {
				// Apply weighted re-ranking for this group
				applyWeightedRerank(media, group.Fields)
			} else if group.Alg == "natural" {
				// Apply natural sorting as tiebreaker for this group
				applyNaturalSort(media, group.Fields)
			} else if group.Alg == "related" {
				// Related media - apply standard natural sort
				// Note: Media expansion happens earlier in SortMediaWithExpansion
				applyStandardSort(media, group.Fields, "natural")
			} else {
				// Standard multi-field sorting
				applyStandardSort(media, group.Fields, group.Alg)
			}
		}
		return
	}

	// Single group - use original logic
	sortFields, alg := parseSortConfig(config)

	if len(sortFields) == 1 && sortFields[0].Field == "ps" {
		// Legacy single-field sorting
		reverse := sortFields[0].Reverse
		less := func(i, j int) bool {
			valI := getSortValueString(media[i], "ps")
			valJ := getSortValueString(media[j], "ps")

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
		cmp := compareSortFields(media, i, j, sortFields, alg)
		return cmp < 0
	})
}

// applyStandardSort applies standard multi-field sorting to media
func applyStandardSort(media []models.MediaWithDB, fields []SortField, alg string) {
	if len(fields) == 0 {
		return
	}
	sort.SliceStable(media, func(i, j int) bool {
		cmp := compareSortFields(media, i, j, fields, alg)
		return cmp < 0
	})
}

// applyNaturalSort applies natural sorting to media using the specified fields as tiebreakers
func applyNaturalSort(media []models.MediaWithDB, fields []SortField) {
	if len(fields) == 0 {
		return
	}
	// Use natural algorithm for string comparisons
	applyStandardSort(media, fields, "natural")
}

// applyWeightedRerank applies MCDA-style weighted re-ranking based on field positions
// Fields earlier in the list have higher weights (position-based weighting)
func applyWeightedRerank(media []models.MediaWithDB, fields []SortField) {
	if len(fields) == 0 || len(media) == 0 {
		return
	}

	// Calculate ranks for each field and combine with weights
	// Weight decreases with position: first field gets highest weight
	type rankedMedia struct {
		index      int
		totalScore float64
	}

	ranked := make([]rankedMedia, len(media))
	for i := range media {
		ranked[i] = rankedMedia{index: i, totalScore: 0}
	}

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
			if isNumericField(field.Field) {
				values[i].value = getSortValueFloat64(m, field.Field)
				values[i].isNumeric = true
			} else {
				values[i].strValue = getSortValueString(m, field.Field)
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
func ExpandRelatedMedia(ctx context.Context, sqlDB *sql.DB, media *[]models.MediaWithDB, flags models.GlobalFlags) error {
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
	relatedRows, err := queryRelatedMediaWithRank(ctx, sqlDB, ftsTable, first.Path, queryStr, 20)
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
					Type:           models.NullStringPtr(row.Type),
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
	Type           sql.NullString
	DB             string
	Rank           float64
}

// queryRelatedMediaWithRank queries for related media using FTS, ordered by bm25 rank
func queryRelatedMediaWithRank(ctx context.Context, sqlDB *sql.DB, ftsTable, excludePath, queryStr string, limit int64) ([]relatedMediaRow, error) {
	// FTS query with bm25() ranking - works with detail=none + trigram
	// Note: bm25() requires the table name, not an alias
	query := fmt.Sprintf(`
		SELECT
			m.path, m.title, m.size, m.duration,
			m.video_count, m.audio_count, m.subtitle_count,
			m.play_count, m.playhead,
			m.time_created, m.time_modified, m.time_downloaded, m.time_last_played,
			m.score, m.type,
			bm25(%s) as rank
		FROM %s, media m
		WHERE m.rowid = %s.rowid
		AND %s MATCH ?
			AND m.path != ?
		ORDER BY bm25(%s) DESC, m.path
		LIMIT ?
	`, ftsTable, ftsTable, ftsTable, ftsTable, ftsTable)

	rows, err := sqlDB.QueryContext(ctx, query, queryStr, excludePath, limit)
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
			&r.Score, &r.Type, &r.Rank,
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
			switch strings.ToLower(col) {
			case "path":
				m.Path = utils.GetString(val)
			case "title":
				m.Title = sql.NullString{String: utils.GetString(val), Valid: true}
			case "duration":
				m.Duration = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "size":
				m.Size = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "time_created":
				m.TimeCreated = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "time_modified":
				m.TimeModified = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "time_deleted":
				m.TimeDeleted = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "time_first_played":
				m.TimeFirstPlayed = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "time_last_played":
				m.TimeLastPlayed = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "play_count":
				m.PlayCount = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "playhead":
				m.Playhead = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "album":
				m.Album = sql.NullString{String: utils.GetString(val), Valid: true}
			case "artist":
				m.Artist = sql.NullString{String: utils.GetString(val), Valid: true}
			case "genre":
				m.Genre = sql.NullString{String: utils.GetString(val), Valid: true}
			case "categories":
				m.Categories = sql.NullString{String: utils.GetString(val), Valid: true}
			case "description":
				m.Description = sql.NullString{String: utils.GetString(val), Valid: true}
			case "language":
				m.Language = sql.NullString{String: utils.GetString(val), Valid: true}
			case "time_downloaded":
				m.TimeDownloaded = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "score":
				m.Score = sql.NullFloat64{Float64: utils.GetFloat64(val), Valid: true}
			case "video_codecs":
				m.VideoCodecs = sql.NullString{String: utils.GetString(val), Valid: true}
			case "audio_codecs":
				m.AudioCodecs = sql.NullString{String: utils.GetString(val), Valid: true}
			case "subtitle_codecs":
				m.SubtitleCodecs = sql.NullString{String: utils.GetString(val), Valid: true}
			case "width":
				m.Width = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "height":
				m.Height = sql.NullInt64{Int64: utils.GetInt64(val), Valid: true}
			case "type":
				m.Type = sql.NullString{String: utils.GetString(val), Valid: true}
			}
		}

		allMedia = append(allMedia, models.MediaWithDB{
			Media: models.FromDB(m),
			DB:    dbPath,
		})
	}

	return allMedia, rows.Err()
}

// QueryDatabase executes a query against a single database
func QueryDatabase(ctx context.Context, dbPath, query string, args []any) ([]models.MediaWithDB, error) {
	sqlDB, err := db.Connect(dbPath)
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
func (qe *QueryExecutor) executeMultiDB(ctx context.Context, dbs []string, query string, args []any) ([]models.MediaWithDB, []error) {
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

// MediaQuery executes a query against multiple databases concurrently
func (qe *QueryExecutor) MediaQuery(ctx context.Context, dbs []string) ([]models.MediaWithDB, error) {
	flags := qe.filterBuilder.flags
	origLimit := flags.Limit
	origOffset := flags.Offset
	isEpisodic := flags.FileCounts != ""
	isMultiDB := len(dbs) > 1

	if isEpisodic {
		// Fetch everything matching other filters so we can count directories accurately
		flags.All = true
		flags.Limit = 0
		flags.Offset = 0
	}

	// For multiple databases, we need to fetch more results from each DB
	tempFlags := flags
	if isMultiDB && !flags.All && flags.Limit > 0 {
		tempFlags.Limit = flags.Limit + flags.Offset
		tempFlags.Offset = 0
	}

	resolvedFlags, err := qe.ResolvePercentileFlags(ctx, dbs, tempFlags)
	if err == nil {
		flags = resolvedFlags
	} else {
		flags = tempFlags
	}

	// Rebuild filter builder with resolved flags
	fb := NewFilterBuilder(flags)
	query, args := fb.BuildQuery("*")

	allMedia, errs := qe.executeMultiDB(ctx, dbs, query, args)
	if len(errs) > 0 {
		return allMedia, errors.Join(errs...)
	}

	if isEpisodic {
		counts := make(map[string]int64)
		for _, m := range allMedia {
			counts[m.Parent()]++
		}

		r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
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
		NewSortBuilder(flags).Sort(allMedia)

		// Apply original limit/offset
		if origOffset > 0 {
			if origOffset >= len(allMedia) {
				return []models.MediaWithDB{}, nil
			}
			allMedia = allMedia[origOffset:]
		}
		if origLimit > 0 && len(allMedia) > origLimit {
			allMedia = allMedia[:origLimit]
		}
	}

	// Group by parent directory
	if flags.GroupByParent {
		allMedia = qe.GroupByParent(allMedia, flags)
	}

	// Fetch siblings
	if flags.FetchSiblings != "" {
		// This still relies on top-level FetchSiblings for now, but uses unified executor
		var err error
		allMedia, err = qe.FetchSiblings(ctx, allMedia, flags)
		if err != nil {
			return allMedia, err
		}
	}

	// For multiple databases, apply limit/offset after merging and sorting
	if isMultiDB && !isEpisodic && !flags.GroupByParent && !flags.All && origLimit > 0 {
		NewSortBuilder(flags).Sort(allMedia)

		if origOffset > 0 {
			if origOffset >= len(allMedia) {
				return []models.MediaWithDB{}, nil
			}
			allMedia = allMedia[origOffset:]
		}
		if len(allMedia) > origLimit {
			allMedia = allMedia[:origLimit]
		}
	}

	return allMedia, nil
}

// MediaQueryCount executes a count query against multiple databases concurrently
func (qe *QueryExecutor) MediaQueryCount(ctx context.Context, dbs []string) (int64, error) {
	flags := qe.filterBuilder.flags

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

	query, args := qe.filterBuilder.BuildCount()

	var wg sync.WaitGroup
	results := make(chan int64, len(dbs))
	errorsChan := make(chan error, len(dbs))

	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sqlDB, err := db.Connect(path)
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
		ParentPath         string  `json:"parent_path"`
		EpisodeCount       int64   `json:"episode_count"`
		TotalSize          int64   `json:"total_size"`
		TotalDuration      int64   `json:"total_duration"`
		LatestEpisodeTime  *string `json:"latest_episode_time,omitempty"`
		RepresentativePath string  `json:"representative_path"`
		RepresentativeType *string `json:"representative_type,omitempty"`
	}

	groups := make(map[string]*GroupedMedia)
	for _, m := range allMedia {
		parent := m.Parent()
		if _, ok := groups[parent]; !ok {
			groups[parent] = &GroupedMedia{
				ParentPath:         parent,
				EpisodeCount:       0,
				TotalSize:          0,
				TotalDuration:      0,
				RepresentativePath: m.Path,
				RepresentativeType: m.Type,
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
				Path:     g.RepresentativePath,
				Type:     g.RepresentativeType,
				Size:     &g.TotalSize,
				Duration: &g.TotalDuration,
				Title:    &g.ParentPath,
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

func (qe *QueryExecutor) FetchSiblings(ctx context.Context, media []models.MediaWithDB, flags models.GlobalFlags) ([]models.MediaWithDB, error) {
	if len(media) == 0 {
		return media, nil
	}

	parentToFiles := make(map[string][]models.MediaWithDB)
	for _, m := range media {
		dir := m.Parent()
		if !strings.HasSuffix(dir, "/") && !strings.HasSuffix(dir, "\\") {
			dir += string(filepath.Separator)
		}
		parentToFiles[dir] = append(parentToFiles[dir], m)
	}

	var allSiblings []models.MediaWithDB
	seenPaths := make(map[string]bool)

	for dir, filesInDir := range parentToFiles {
		dbPath := filesInDir[0].DB

		limit := flags.FetchSiblingsMax
		if flags.FetchSiblings == "all" || flags.FetchSiblings == "always" {
			limit = 2000
		} else if flags.FetchSiblings == "each" {
			if limit <= 0 {
				limit = len(filesInDir)
			}
		} else if flags.FetchSiblings == "if-audiobook" {
			isAudiobook := false
			for _, f := range filesInDir {
				if strings.Contains(strings.ToLower(f.Path), "audiobook") {
					isAudiobook = true
					break
				}
			}
			if !isAudiobook {
				for _, f := range filesInDir {
					if !seenPaths[f.Path] {
						allSiblings = append(allSiblings, f)
						seenPaths[f.Path] = true
					}
				}
				continue
			}
			if limit <= 0 {
				limit = 2000
			}
		} else if utils.IsDigit(flags.FetchSiblings) {
			if l, err := strconv.Atoi(flags.FetchSiblings); err == nil {
				limit = l
			}
		} else {
			for _, f := range filesInDir {
				if !seenPaths[f.Path] {
					allSiblings = append(allSiblings, f)
					seenPaths[f.Path] = true
				}
			}
			continue
		}

		query := "SELECT * FROM media WHERE time_deleted = 0 AND path LIKE ? ORDER BY path LIMIT ?"
		pattern := dir + "%"
		siblings, err := QueryDatabase(ctx, dbPath, query, []any{pattern, limit})
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

func (qe *QueryExecutor) ResolvePercentileFlags(ctx context.Context, dbs []string, flags models.GlobalFlags) (models.GlobalFlags, error) {
	hasPSize := false
	for _, s := range flags.Size {
		if _, _, ok := utils.ParsePercentileRange(s); ok {
			hasPSize = true
			break
		}
	}

	hasPDuration := false
	for _, d := range flags.Duration {
		if _, _, ok := utils.ParsePercentileRange(d); ok {
			hasPDuration = true
			break
		}
	}

	hasPEpisodes := false
	if _, _, ok := utils.ParsePercentileRange(flags.FileCounts); ok {
		hasPEpisodes = true
	}

	if !hasPSize && !hasPDuration && !hasPEpisodes {
		return flags, nil
	}

	getValues := func(field string) ([]int64, error) {
		tempFlags := flags
		var cleanSize []string
		for _, s := range flags.Size {
			if _, _, ok := utils.ParsePercentileRange(s); !ok {
				cleanSize = append(cleanSize, s)
			}
		}
		tempFlags.Size = cleanSize

		var cleanDuration []string
		for _, d := range flags.Duration {
			if _, _, ok := utils.ParsePercentileRange(d); !ok {
				cleanDuration = append(cleanDuration, d)
			}
		}
		tempFlags.Duration = cleanDuration

		if _, _, ok := utils.ParsePercentileRange(flags.FileCounts); ok {
			tempFlags.FileCounts = ""
		}
		tempFlags.All = true
		tempFlags.Limit = 0

		fb := NewFilterBuilder(tempFlags)
		var sqlQuery string
		var args []any
		if field == "episodes" {
			sqlQuery, args = fb.BuildSelect("path")
		} else {
			sqlQuery, args = fb.BuildSelect(field)
		}

		var values []int64
		var mu sync.Mutex
		var wg sync.WaitGroup
		for _, dbPath := range dbs {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				sqlDB, err := db.Connect(path)
				if err != nil {
					return
				}
				defer sqlDB.Close()

				rows, err := sqlDB.QueryContext(ctx, sqlQuery, args...)
				if err != nil {
					return
				}
				defer rows.Close()

				if field == "episodes" {
					gCounts := make(map[string]int64)
					for rows.Next() {
						var p string
						if err := rows.Scan(&p); err == nil {
							gCounts[filepath.Dir(p)]++
						}
					}

					mu.Lock()
					for _, c := range gCounts {
						values = append(values, c)
					}
					mu.Unlock()
				} else {
					var localValues []int64
					for rows.Next() {
						var v sql.NullInt64
						if err := rows.Scan(&v); err == nil && v.Valid {
							localValues = append(localValues, v.Int64)
						}
					}
					mu.Lock()
					values = append(values, localValues...)
					mu.Unlock()
				}
			}(dbPath)
		}
		wg.Wait()
		return values, nil
	}

	if hasPSize {
		values, err := getValues("size")
		if err == nil && len(values) > 0 {
			mapping := utils.CalculatePercentiles(values)
			var newSize []string
			for _, s := range flags.Size {
				if min, max, ok := utils.ParsePercentileRange(s); ok {
					minVal := mapping[int(min)]
					maxVal := mapping[int(max)]
					newSize = append(newSize, fmt.Sprintf("+%d", minVal))
					newSize = append(newSize, fmt.Sprintf("-%d", maxVal))
				} else {
					newSize = append(newSize, s)
				}
			}
			flags.Size = newSize
		}
	}

	if hasPDuration {
		values, err := getValues("duration")
		if err == nil && len(values) > 0 {
			mapping := utils.CalculatePercentiles(values)
			var newDuration []string
			for _, d := range flags.Duration {
				if min, max, ok := utils.ParsePercentileRange(d); ok {
					minVal := mapping[int(min)]
					maxVal := mapping[int(max)]
					newDuration = append(newDuration, fmt.Sprintf("+%d", minVal))
					newDuration = append(newDuration, fmt.Sprintf("-%d", maxVal))
				} else {
					newDuration = append(newDuration, d)
				}
			}
			flags.Duration = newDuration
		}
	}

	if hasPEpisodes {
		values, err := getValues("episodes")
		if err == nil && len(values) > 0 {
			mapping := utils.CalculatePercentiles(values)
			if min, max, ok := utils.ParsePercentileRange(flags.FileCounts); ok {
				minVal := mapping[int(min)]
				maxVal := mapping[int(max)]
				flags.FileCounts = fmt.Sprintf("+%d,-%d", minVal, maxVal)
			}
		}
	}

	return flags, nil
}
