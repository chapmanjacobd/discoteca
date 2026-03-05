package query

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/utils"
)

// QueryBuilder constructs SQL queries from flags
type QueryBuilder struct {
	Flags models.GlobalFlags
}

func NewQueryBuilder(flags models.GlobalFlags) *QueryBuilder {
	return &QueryBuilder{Flags: flags}
}

func (qb *QueryBuilder) Build() (string, []any) {
	return qb.BuildSelect("*")
}

func (qb *QueryBuilder) BuildCount() (string, []any) {
	return qb.BuildSelect("COUNT(*)")
}

func (qb *QueryBuilder) BuildSelect(columns string) (string, []any) {
	// If raw query provided, use it
	if qb.Flags.Query != "" {
		if columns == "COUNT(*)" {
			return "SELECT COUNT(*) FROM (" + qb.Flags.Query + ")", nil
		}
		return qb.Flags.Query, nil
	}

	var whereClauses []string
	var args []any

	// Base table
	table := "media"
	if qb.Flags.FTS {
		table = qb.Flags.FTSTable
		if table == "" {
			table = "media_fts"
		}
	}

	// Deleted status
	if qb.Flags.OnlyDeleted {
		whereClauses = append(whereClauses, "COALESCE(time_deleted, 0) > 0")
	} else if qb.Flags.HideDeleted {
		whereClauses = append(whereClauses, "COALESCE(time_deleted, 0) = 0")
	}

	if qb.Flags.DeletedAfter != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.DeletedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_deleted >= ?")
			args = append(args, ts)
		}
	}
	if qb.Flags.DeletedBefore != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.DeletedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_deleted <= ?")
			args = append(args, ts)
		}
	}

	// Category filter
	if len(qb.Flags.Category) > 0 {
		var catClauses []string
		for _, cat := range qb.Flags.Category {
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

	// Genre filter
	if qb.Flags.Genre != "" {
		whereClauses = append(whereClauses, "genre = ?")
		args = append(args, qb.Flags.Genre)
	}

	// Search terms (FTS or LIKE)
	allInclude := append([]string{}, qb.Flags.Search...)
	allInclude = append(allInclude, qb.Flags.Include...)

	// Path contains filters
	pathContains := append([]string{}, qb.Flags.PathContains...)

	var filteredInclude []string
	for _, term := range allInclude {
		if strings.HasPrefix(term, "./") {
			pathContains = append(pathContains, term[1:]) // Strip . keep /
		} else if strings.HasPrefix(term, "/") {
			pathContains = append(pathContains, term)
		} else {
			filteredInclude = append(filteredInclude, term)
		}
	}
	allInclude = filteredInclude

	if len(allInclude) > 0 {
		joinOp := " AND "
		if qb.Flags.FlexibleSearch {
			joinOp = " OR "
		}

		if qb.Flags.FTS {
			// FTS match syntax
			var ftsTerms []string
			for _, term := range allInclude {
				if strings.Contains(term, ":") {
					parts := strings.SplitN(term, ":", 2)
					col, val := parts[0], parts[1]
					// Validate column name to prevent injection
					validCols := map[string]bool{"title": true, "path": true, "text": true}
					if validCols[strings.ToLower(col)] {
						ftsTerms = append(ftsTerms, fmt.Sprintf("%s:%s", col, utils.FtsQuote([]string{val})[0]))
						continue
					}
				}
				ftsTerms = append(ftsTerms, utils.FtsQuote([]string{term})[0])
			}
			searchTerm := strings.Join(ftsTerms, joinOp)
			whereClauses = append(whereClauses, fmt.Sprintf("%s MATCH ?", table))
			args = append(args, searchTerm)
		} else {
			// Regular LIKE search
			var searchParts []string
			for _, term := range allInclude {
				searchParts = append(searchParts, "(path LIKE ? OR title LIKE ?)")
				pattern := "%" + strings.ReplaceAll(term, " ", "%") + "%"
				args = append(args, pattern, pattern)
			}
			whereClauses = append(whereClauses, "("+strings.Join(searchParts, joinOp)+")")
		}
	}

	for _, exc := range qb.Flags.Exclude {
		whereClauses = append(whereClauses, "path NOT LIKE ? AND title NOT LIKE ?")
		pattern := "%" + exc + "%"
		args = append(args, pattern, pattern)
	}

	// Regex filter (requires regex extension or post-filter)
	if qb.Flags.Regex != "" {
		whereClauses = append(whereClauses, "path REGEXP ?")
		args = append(args, qb.Flags.Regex)
	}

	// Path contains filters
	for _, contain := range pathContains {
		whereClauses = append(whereClauses, "path LIKE ?")
		args = append(args, "%"+contain+"%")
	}

	// Exact path filters
	if len(qb.Flags.Paths) > 0 {
		var inPaths []string
		for _, p := range qb.Flags.Paths {
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

	// Size filters
	for _, s := range qb.Flags.Size {
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

	// Duration filters
	for _, s := range qb.Flags.Duration {
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

	// Time filters
	if qb.Flags.CreatedAfter != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.CreatedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_created >= ?")
			args = append(args, ts)
		}
	}
	if qb.Flags.CreatedBefore != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.CreatedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_created <= ?")
			args = append(args, ts)
		}
	}
	if qb.Flags.ModifiedAfter != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.ModifiedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_modified >= ?")
			args = append(args, ts)
		}
	}
	if qb.Flags.ModifiedBefore != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.ModifiedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_modified <= ?")
			args = append(args, ts)
		}
	}
	if qb.Flags.PlayedAfter != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.PlayedAfter); ts > 0 {
			whereClauses = append(whereClauses, "time_last_played >= ?")
			args = append(args, ts)
		}
	}
	if qb.Flags.PlayedBefore != "" {
		if ts := utils.ParseDateOrRelative(qb.Flags.PlayedBefore); ts > 0 {
			whereClauses = append(whereClauses, "time_last_played <= ?")
			args = append(args, ts)
		}
	}

	// Watched status
	if qb.Flags.Watched != nil {
		if *qb.Flags.Watched {
			whereClauses = append(whereClauses, "time_last_played > 0")
		} else {
			whereClauses = append(whereClauses, "COALESCE(time_last_played, 0) = 0")
		}
	}

	// Unfinished (has playhead but presumably not done)
	if qb.Flags.Unfinished || qb.Flags.InProgress {
		whereClauses = append(whereClauses, "COALESCE(playhead, 0) > 0")
	}

	if qb.Flags.Partial != "" {
		if strings.Contains(qb.Flags.Partial, "s") {
			whereClauses = append(whereClauses, "COALESCE(time_first_played, 0) = 0")
		} else {
			whereClauses = append(whereClauses, "time_first_played > 0")
		}
	}

	if qb.Flags.Completed {
		whereClauses = append(whereClauses, "COALESCE(play_count, 0) > 0")
	}

	if qb.Flags.WithCaptions {
		whereClauses = append(whereClauses, "path IN (SELECT DISTINCT media_path FROM captions)")
	}

	// Play count filters
	if qb.Flags.PlayCountMin > 0 {
		whereClauses = append(whereClauses, "play_count >= ?")
		args = append(args, qb.Flags.PlayCountMin)
	}
	if qb.Flags.PlayCountMax > 0 {
		whereClauses = append(whereClauses, "play_count <= ?")
		args = append(args, qb.Flags.PlayCountMax)
	}

	// Content type filters
	var typeClauses []string
	if qb.Flags.VideoOnly {
		typeClauses = append(typeClauses, "type = 'video'")
	}
	if qb.Flags.AudioOnly {
		typeClauses = append(typeClauses, "type = 'audio'", "type = 'audiobook'")
	}
	if qb.Flags.ImageOnly {
		typeClauses = append(typeClauses, "type = 'image'")
	}
	if qb.Flags.TextOnly {
		typeClauses = append(typeClauses, "type = 'text'")
	}
	if len(typeClauses) > 0 {
		whereClauses = append(whereClauses, "("+strings.Join(typeClauses, " OR ")+")")
	}

	if qb.Flags.Portrait {
		whereClauses = append(whereClauses, "width < height")
	}

	if qb.Flags.OnlineMediaOnly {
		whereClauses = append(whereClauses, "path LIKE 'http%'")
	}
	if qb.Flags.LocalMediaOnly {
		whereClauses = append(whereClauses, "path NOT LIKE 'http%'")
	}

	// Custom WHERE clauses
	whereClauses = append(whereClauses, qb.Flags.Where...)

	// Extension filters
	if len(qb.Flags.Ext) > 0 {
		var extClauses []string
		for _, ext := range qb.Flags.Ext {
			extClauses = append(extClauses, "path LIKE ?")
			args = append(args, "%"+ext)
		}
		whereClauses = append(whereClauses, "("+strings.Join(extClauses, " OR ")+")")
	}

	if qb.Flags.DurationFromSize != "" {
		if r, err := utils.ParseRange(qb.Flags.DurationFromSize, utils.HumanToBytes); err == nil {
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

	// Build query
	fromClause := "media"
	useFTSJoin := qb.Flags.FTS && len(allInclude) > 0

	if useFTSJoin {
		fromClause = fmt.Sprintf("media JOIN %s ON media.rowid = %s.rowid", table, table)
		if columns == "*" {
			columns = "media.*"
		}
	}
	query := fmt.Sprintf("SELECT %s FROM %s", columns, fromClause)

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if columns == "COUNT(*)" {
		return query, args
	}

	// Order by
	if !qb.Flags.Random && !qb.Flags.NatSort && qb.Flags.SortBy != "" {
		sortExpr := OverrideSort(qb.Flags.SortBy)
		order := "ASC"
		if qb.Flags.Reverse {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", sortExpr, order)
	} else if qb.Flags.Random {
		// Optimization for large databases: select rowids randomly first
		// Python: and m.rowid in (select rowid as id from media {where_not_deleted} order by random() limit {limit})
		if !qb.Flags.All && !qb.Flags.FTS && len(allInclude) == 0 && qb.Flags.Limit > 0 {
			whereNotDeleted := "WHERE COALESCE(time_deleted, 0) = 0"
			if qb.Flags.OnlyDeleted {
				whereNotDeleted = "WHERE COALESCE(time_deleted, 0) > 0"
			}
			// We use a larger pool for random selection then limit it in the outer query
			randomLimit := qb.Flags.Limit * 16

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
	if !qb.Flags.All && qb.Flags.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.Flags.Limit)
	}
	if qb.Flags.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.Flags.Offset)
	}

	return query, args
}

func OverrideSort(s string) string {
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

	return s
}

// MediaQuery executes a query against multiple databases concurrently
func MediaQuery(ctx context.Context, dbs []string, flags models.GlobalFlags) ([]models.MediaWithDB, error) {
	origLimit := flags.Limit
	origOffset := flags.Offset
	isEpisodic := flags.FileCounts != ""

	if isEpisodic {
		// Fetch everything matching other filters so we can count directories accurately
		flags.All = true
		flags.Limit = 0
		flags.Offset = 0
	}

	resolvedFlags, err := ResolvePercentileFlags(ctx, dbs, flags)
	if err == nil {
		flags = resolvedFlags
	}

	qb := NewQueryBuilder(flags)
	query, args := qb.Build()

	var wg sync.WaitGroup
	results := make(chan []models.MediaWithDB, len(dbs))
	errors := make(chan error, len(dbs))

	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			media, err := QueryDatabase(ctx, path, query, args)
			if err != nil {
				errors <- fmt.Errorf("%s: %w", path, err)
				return
			}
			results <- media
		}(dbPath)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	allMedia := []models.MediaWithDB{}
	for media := range results {
		allMedia = append(allMedia, media...)
	}

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return allMedia, fmt.Errorf("query errors: %v", errs)
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

		// Apply sorting again because merging results from different DBs might break global order
		SortMedia(allMedia, flags)

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

	if flags.FetchSiblings != "" {
		var err error
		allMedia, err = FetchSiblings(ctx, allMedia, flags)
		if err != nil {
			return allMedia, err
		}
	}

	return allMedia, nil
}

func ResolvePercentileFlags(ctx context.Context, dbs []string, flags models.GlobalFlags) (models.GlobalFlags, error) {
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

	// Helper to get values for a field
	getValues := func(field string) ([]int64, error) {
		tempFlags := flags
		// Clear all percentile filters to avoid nested resolution or circular dependencies
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
		// We need to disable limits to get the full distribution
		tempFlags.All = true
		tempFlags.Limit = 0

		qb := NewQueryBuilder(tempFlags)
		var sqlQuery string
		var args []any
		if field == "episodes" {
			sqlQuery, args = qb.BuildSelect("path")
		} else {
			sqlQuery, args = qb.BuildSelect(field)
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
					// Filtered counts
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

// MediaQueryCount executes a count query against multiple databases concurrently
func MediaQueryCount(ctx context.Context, dbs []string, flags models.GlobalFlags) (int64, error) {
	if flags.FileCounts != "" {
		// We must fetch all results to count episodic matches across multiple DBs correctly
		tempFlags := flags
		tempFlags.All = true
		tempFlags.Limit = 0
		tempFlags.Offset = 0
		allMedia, err := MediaQuery(ctx, dbs, tempFlags)
		if err != nil {
			return 0, err
		}
		return int64(len(allMedia)), nil
	}

	qb := NewQueryBuilder(flags)
	query, args := qb.BuildCount()

	var wg sync.WaitGroup
	results := make(chan int64, len(dbs))
	errors := make(chan error, len(dbs))

	for _, dbPath := range dbs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sqlDB, err := db.Connect(path)
			if err != nil {
				errors <- err
				return
			}
			defer sqlDB.Close()

			var count int64
			err = sqlDB.QueryRowContext(ctx, query, args...).Scan(&count)
			if err != nil {
				errors <- err
				return
			}
			results <- count
		}(dbPath)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	var total int64
	for count := range results {
		total += count
	}

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return total, fmt.Errorf("count query errors: %v", errs)
	}

	return total, nil
}

func FetchSiblings(ctx context.Context, media []models.MediaWithDB, flags models.GlobalFlags) ([]models.MediaWithDB, error) {
	if len(media) == 0 {
		return media, nil
	}

	parentToFiles := make(map[string][]models.MediaWithDB)
	for _, m := range media {
		dir := m.Parent() + "/"
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
				// Keep original files and move to next dir
				for _, f := range filesInDir {
					if !seenPaths[f.Path] {
						allSiblings = append(allSiblings, f)
						seenPaths[f.Path] = true
					}
				}
				continue
			}
			if limit <= 0 {
				limit = 2000 // default for audiobook siblings if not specified
			}
		} else if utils.IsDigit(flags.FetchSiblings) {
			if l, err := strconv.Atoi(flags.FetchSiblings); err == nil {
				limit = l
			}
		} else {
			// fallback: if not specified or unknown, just keep original
			for _, f := range filesInDir {
				if !seenPaths[f.Path] {
					allSiblings = append(allSiblings, f)
					seenPaths[f.Path] = true
				}
			}
			continue
		}

		// Fetch from DB
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

			switch strings.ToLower(col) {
			case "path":
				m.Path = utils.GetString(values[i])
			case "title":
				m.Title = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "duration":
				m.Duration = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "size":
				m.Size = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "time_created":
				m.TimeCreated = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "time_modified":
				m.TimeModified = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "time_deleted":
				m.TimeDeleted = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "time_first_played":
				m.TimeFirstPlayed = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "time_last_played":
				m.TimeLastPlayed = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "play_count":
				m.PlayCount = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "playhead":
				m.Playhead = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "album":
				m.Album = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "artist":
				m.Artist = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "genre":
				m.Genre = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "mood":
				m.Mood = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "bpm":
				m.Bpm = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "key":
				m.Key = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "decade":
				m.Decade = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "categories":
				m.Categories = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "city":
				m.City = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "country":
				m.Country = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "description":
				m.Description = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "language":
				m.Language = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "video_codecs":
				m.VideoCodecs = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "audio_codecs":
				m.AudioCodecs = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "subtitle_codecs":
				m.SubtitleCodecs = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			case "width":
				m.Width = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "height":
				m.Height = sql.NullInt64{Int64: utils.GetInt64(values[i]), Valid: true}
			case "type":
				m.Type = sql.NullString{String: utils.GetString(values[i]), Valid: true}
			}
		}

		allMedia = append(allMedia, models.MediaWithDB{
			Media: models.FromDB(m),
			DB:    dbPath,
		})
	}

	return allMedia, rows.Err()
}

// FilterMedia applies all filters to media list
func FilterMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	filtered := []models.MediaWithDB{}

	for _, m := range media {
		// Check existence
		if flags.Exists && !utils.FileExists(m.Path) {
			continue
		}

		// Include/exclude patterns
		if len(flags.Include) > 0 && !utils.MatchesAny(m.Path, flags.Include) {
			continue
		}
		if len(flags.Exclude) > 0 && utils.MatchesAny(m.Path, flags.Exclude) {
			continue
		}

		// Size filters
		matchedSize := true
		for _, s := range flags.Size {
			if r, err := utils.ParseRange(s, utils.HumanToBytes); err == nil {
				if m.Size == nil || !r.Matches(*m.Size) {
					matchedSize = false
					break
				}
			}
		}
		if !matchedSize {
			continue
		}

		// Duration filters
		matchedDuration := true
		for _, s := range flags.Duration {
			if r, err := utils.ParseRange(s, utils.HumanToSeconds); err == nil {
				if m.Duration == nil || !r.Matches(*m.Duration) {
					matchedDuration = false
					break
				}
			}
		}
		if !matchedDuration {
			continue
		}

		// Extension filters
		if len(flags.Ext) > 0 {
			matched := false
			fileExt := strings.ToLower(filepath.Ext(m.Path))
			for _, ext := range flags.Ext {
				if fileExt == strings.ToLower(ext) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Regex filter
		if flags.Regex != "" {
			if matched, _ := regexp.MatchString(flags.Regex, m.Path); !matched {
				continue
			}
		}

		// Mimetype filters
		if len(flags.MimeType) > 0 {
			match := false
			if m.Type != nil && utils.IsMimeMatch(flags.MimeType, *m.Type) {
				match = true
			}
			if !match {
				continue
			}
		}
		if len(flags.NoMimeType) > 0 {
			if m.Type != nil && utils.IsMimeMatch(flags.NoMimeType, *m.Type) {
				continue
			}
		}

		filtered = append(filtered, m)
	}

	return filtered
}

// SortMedia sorts media using various methods
func SortMedia(media []models.MediaWithDB, flags models.GlobalFlags) {
	if flags.Random {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(media), func(i, j int) {
			media[i], media[j] = media[j], media[i]
		})
		return
	}

	if flags.NoPlayInOrder {
		sortMediaBasic(media, flags.SortBy, flags.Reverse, flags.NatSort)
		return
	}

	// If the user explicitly requested a specific sort field other than "path",
	// we should respect it and skip the default play-in-order.
	if flags.SortBy != "" && flags.SortBy != "path" {
		sortMediaBasic(media, flags.SortBy, flags.Reverse, flags.NatSort)
		return
	}

	if flags.PlayInOrder != "" {
		SortMediaAdvanced(media, flags.PlayInOrder)
		return
	}

	sortMediaBasic(media, flags.SortBy, flags.Reverse, flags.NatSort)
}

func sortMediaBasic(media []models.MediaWithDB, sortBy string, reverse bool, natSort bool) {
	// Special handling for sparse fields where we want 0/nulls at the bottom always
	if sortBy == "play_count" || sortBy == "time_last_played" || sortBy == "progress" {
		sort.Slice(media, func(i, j int) bool {
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
			// In ascending sort (less(i,j)), if i should come after j, return false.
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
			return utils.Int64Value(media[i].Duration) < utils.Int64Value(media[j].Duration)
		case "size":
			return utils.Int64Value(media[i].Size) < utils.Int64Value(media[j].Size)
		case "bitrate":
			d1 := utils.Int64Value(media[i].Duration)
			d2 := utils.Int64Value(media[j].Duration)
			if d1 == 0 || d2 == 0 {
				return false
			}
			return float64(utils.Int64Value(media[i].Size))/float64(d1) < float64(utils.Int64Value(media[j].Size))/float64(d2)
		case "priority":
			d1 := utils.Int64Value(media[i].Duration)
			d2 := utils.Int64Value(media[j].Duration)
			if d1 == 0 || d2 == 0 {
				return false
			}
			return float64(utils.Int64Value(media[i].Size))/float64(d1) < float64(utils.Int64Value(media[j].Size))/float64(d2)
		case "priorityfast":
			// Simplified version of ntile(1000) over (order by size) desc, duration
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
		case "type":
			return utils.StringValue(media[i].Type) < utils.StringValue(media[j].Type)
		default:
			return media[i].Path < media[j].Path
		}
	}

	if reverse {
		sort.Slice(media, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.Slice(media, less)
	}
}

// SortMediaAdvanced implements the PlayInOrder logic from Python's natsort_media
func SortMediaAdvanced(media []models.MediaWithDB, config string) {
	reverse := false
	if after, ok := strings.CutPrefix(config, "reverse_"); ok {
		config = after
		reverse = true
	}

	// For now, we simplify the algorithms to natural/python and focus on the keys
	var alg, sortKey string
	if strings.Contains(config, "_") {
		parts := strings.SplitN(config, "_", 2)
		alg, sortKey = parts[0], parts[1]
	} else {
		// If config matches an algorithm name, use default key "ps"
		// Otherwise, use config as key and default algorithm "natural"
		knownAlgs := map[string]bool{"natural": true, "path": true, "ignorecase": true, "lowercase": true, "human": true, "locale": true, "signed": true, "os": true, "python": true}
		if knownAlgs[config] {
			alg = config
			sortKey = "ps"
		} else {
			alg = "natural"
			sortKey = config
		}
	}

	getSortValue := func(m models.MediaWithDB, key string) string {
		switch key {
		case "parent":
			return m.Parent()
		case "stem":
			return m.Stem()
		case "ps":
			return m.Parent() + " " + m.Stem()
		case "pts":
			return m.Parent() + " " + utils.StringValue(m.Title) + " " + m.Stem()
		case "path":
			return m.Path
		case "title":
			return utils.StringValue(m.Title)
		default:
			return m.Path // fallback
		}
	}

	less := func(i, j int) bool {
		valI := getSortValue(media[i], sortKey)
		valJ := getSortValue(media[j], sortKey)

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
}

// ReRankMedia implements MCDA-like re-ranking
func ReRankMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	if flags.ReRank == "" {
		return media
	}

	// Parse re-rank flags (e.g., "size=3 duration=1 -play_count=2")
	weights := make(map[string]float64)
	parts := strings.FieldsSeq(flags.ReRank)
	for p := range parts {
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

	// For each weight, calculate rank and add to score
	for col, weight := range weights {
		direction := 1.0
		cleanCol := col
		if strings.HasPrefix(col, "-") {
			direction = -1.0
			cleanCol = col[1:]
		}

		// Sort by this column to get ranks
		sort.SliceStable(items, func(i, j int) bool {
			valI := getMediaValueFloat(items[i].media, cleanCol)
			valJ := getMediaValueFloat(items[j].media, cleanCol)
			if direction > 0 {
				return valI < valJ
			}
			return valI > valJ
		})

		// Assign ranks (0 to n-1) and multiply by weight
		for i := range n {
			items[i].score += float64(i) * weight
		}
	}

	// Final sort by score
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

// SortHistory applies specialized sorting for playback history (from filter_engine.history_sort)
func SortHistory(media []models.MediaWithDB, partial string, reverse bool) {
	if strings.Contains(partial, "s") {
		// filter out seen items - should be done by builder but just in case
		var filtered []models.MediaWithDB
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
			// weighted remaining: (duration / playhead) * -(duration - playhead)
			return (float64(duration) / float64(playhead)) * -float64(duration-playhead)
		} else if strings.Contains(partial, "t") {
			// time remaining: -(duration - playhead)
			return -float64(duration - playhead)
		} else {
			// percent remaining: playhead / duration
			return float64(playhead) / float64(duration)
		}
	}

	less := func(i, j int) bool {
		var valI, valJ float64

		if strings.Contains(partial, "f") {
			// first-viewed
			valI = float64(utils.Int64Value(media[i].TimeFirstPlayed))
			valJ = float64(utils.Int64Value(media[j].TimeFirstPlayed))
		} else if strings.Contains(partial, "p") || strings.Contains(partial, "t") {
			// sort by remaining duration
			valI = mpvProgress(media[i])
			valJ = mpvProgress(media[j])
		} else {
			// default: last played
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
}

// RegexSortMedia sorts media using the text processor (regex splitting and word sorting)
func RegexSortMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.MediaWithDB {
	if len(media) == 0 {
		return media
	}

	sentenceStrings := make([]string, len(media))
	mapping := make(map[string][]models.MediaWithDB)

	for i, m := range media {
		// Build a searchable sentence from path and title
		parts := []string{m.Path}
		if m.Title != nil {
			parts = append(parts, *m.Title)
		}
		sentence := utils.PathToSentence(strings.Join(parts, " "))
		sentenceStrings[i] = sentence
		mapping[sentence] = append(mapping[sentence], m)
	}

	sortedSentences := utils.TextProcessor(flags, sentenceStrings)

	// Reconstruct media list in sorted order
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

// SortFolders sorts folder stats
func SortFolders(folders []models.FolderStats, sortBy string, reverse bool) {
	less := func(i, j int) bool {
		switch sortBy {
		case "count":
			return folders[i].Count < folders[j].Count
		case "size":
			return folders[i].TotalSize < folders[j].TotalSize
		case "duration":
			return folders[i].TotalDuration < folders[j].TotalDuration
		case "priority":
			p1 := float64(folders[i].TotalSize) / float64(utils.Max(1, folders[i].Count))
			p2 := float64(folders[j].TotalSize) / float64(utils.Max(1, folders[j].Count))
			if p1 != p2 {
				return p1 < p2
			}
			return folders[i].TotalSize < folders[j].TotalSize
		case "path":
			return folders[i].Path < folders[j].Path
		default:
			return folders[i].Path < folders[j].Path
		}
	}

	if reverse {
		sort.Slice(folders, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.Slice(folders, less)
	}
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

type FrequencyStats struct {
	Label         string `json:"label"`
	Count         int    `json:"count"`
	TotalSize     int64  `json:"total_size"`
	TotalDuration int64  `json:"total_duration"`
}

func HistoricalUsage(ctx context.Context, dbPath string, freq string, timeColumn string) ([]FrequencyStats, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	var freqSql string
	switch freq {
	case "daily":
		freqSql = fmt.Sprintf("strftime('%%Y-%%m-%%d', datetime(%s, 'unixepoch'))", timeColumn)
	case "weekly":
		freqSql = fmt.Sprintf("strftime('%%Y-%%W', datetime(%s, 'unixepoch'))", timeColumn)
	case "monthly":
		freqSql = fmt.Sprintf("strftime('%%Y-%%m', datetime(%s, 'unixepoch'))", timeColumn)
	case "quarterly":
		freqSql = fmt.Sprintf("strftime('%%Y', datetime(%s, 'unixepoch', '-3 months')) || '-Q' || ((strftime('%%m', datetime(%s, 'unixepoch', '-3 months')) - 1) / 3 + 1)", timeColumn, timeColumn)
	case "yearly":
		freqSql = fmt.Sprintf("strftime('%%Y', datetime(%s, 'unixepoch'))", timeColumn)
	case "decadally":
		freqSql = fmt.Sprintf("(CAST(strftime('%%Y', datetime(%s, 'unixepoch')) AS INTEGER) / 10) * 10", timeColumn)
	case "hourly":
		freqSql = fmt.Sprintf("strftime('%%Y-%%m-%%d %%Hh', datetime(%s, 'unixepoch'))", timeColumn)
	case "minutely":
		freqSql = fmt.Sprintf("strftime('%%Y-%%m-%%d %%H:%%M', datetime(%s, 'unixepoch'))", timeColumn)
	default:
		return nil, fmt.Errorf("invalid frequency: %s", freq)
	}

	query := fmt.Sprintf(`
		SELECT
			%s AS label,
			COUNT(*) AS count,
			SUM(size) AS total_size,
			SUM(duration) AS total_duration
		FROM media
		WHERE %s > 0 AND time_deleted = 0
		GROUP BY label
		ORDER BY label DESC
	`, freqSql, timeColumn)

	rows, err := sqlDB.QueryContext(ctx, query)
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
	return stats, nil
}
