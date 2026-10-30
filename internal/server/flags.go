package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

// GetGlobalFlags returns all embedded flag structs
func (c *Server) GetGlobalFlags() models.GlobalFlags {
	return models.GlobalFlags{
		CoreFlags:          c.CoreFlags,
		QueryFlags:         c.QueryFlags,
		PathFilterFlags:    c.PathFilterFlags,
		FilterFlags:        c.FilterFlags,
		MediaFilterFlags:   c.MediaFilterFlags,
		TimeFilterFlags:    c.TimeFilterFlags,
		DeletedFlags:       c.DeletedFlags,
		SortFlags:          c.SortFlags,
		DisplayFlags:       c.DisplayFlags,
		AggregateFlags:     c.AggregateFlags,
		PlaybackFlags:      c.PlaybackFlags,
		PostActionFlags:    c.PostActionFlags,
		FTSFlags:           c.FTSFlags,
		SearchModeExplicit: c.FTS,
	}
}

// applySearchMode chooses FTS automatically when the request did not specify
// a search mode and every selected database has an FTS index. Mixed or
// non-FTS databases fall back to substring search.
func (c *Server) applySearchMode(ctx context.Context, flags *models.GlobalFlags, dbs []string) {
	if flags.SearchModeExplicit || len(flags.Search) == 0 {
		return
	}
	if query.DatabasesHaveFTS(ctx, dbs) {
		flags.FTS = true
		flags.NoFTS = false
	}
}

// parseSearchFlags extracts search-related flags from query parameters
func (c *Server) parseSearchFlags(flags *models.GlobalFlags, q url.Values) {
	if search := q.Get("search"); search != "" {
		flags.Search = splitSearchTerms(search)
	}
	if searchType := q.Get("search_type"); searchType != "" {
		flags.SearchModeExplicit = true
		switch searchType {
		case "substring":
			flags.FTS = false
		case "fts":
			flags.FTS = true
		}
	}
}

// splitSearchTerms behaves like [strings.Fields] but keeps quoted phrases
// together so the FTS parser can recognize them.
func splitSearchTerms(search string) []string {
	var terms []string
	var current strings.Builder
	var quote rune
	inQuote := false

	flush := func() {
		if current.Len() > 0 {
			terms = append(terms, current.String())
			current.Reset()
		}
	}

	for _, r := range search {
		if inQuote {
			current.WriteRune(r)
			if r == quote {
				inQuote = false
				flush()
			}
			continue
		}
		if r == '"' || r == '\'' {
			flush()
			quote = r
			inQuote = true
			current.WriteRune(r)
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' {
			flush()
			continue
		}
		current.WriteRune(r)
	}
	flush()
	return terms
}

// parseCategoryFlags extracts category and genre flags
func (c *Server) parseCategoryFlags(flags *models.GlobalFlags, q url.Values) {
	if categories := q["category"]; len(categories) > 0 {
		flags.Category = categories
	}
	if genre := q.Get("genre"); genre != "" {
		flags.Genre = genre
	}
}

// parseLanguageFlags extracts language filter flags
func (c *Server) parseLanguageFlags(flags *models.GlobalFlags, q url.Values) {
	if languages := q["language"]; len(languages) > 0 {
		flags.Language = languages
	}
}

// parsePathFlags extracts path-related flags
func (c *Server) parsePathFlags(flags *models.GlobalFlags, q url.Values) {
	if paths := q.Get("paths"); paths != "" {
		flags.Paths = strings.Split(paths, ",")
	}
	flags.Include = append(flags.Include, q["include"]...)
	flags.Exclude = append(flags.Exclude, q["exclude"]...)
	flags.PathContains = append(flags.PathContains, q["path_contains"]...)
	if regex := q.Get("regex"); regex != "" {
		if _, err := regexp.Compile(regex); err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid regex: %w", err))
		} else {
			flags.Regex = regex
		}
	}
}

// parseRatingFlags extracts rating filter flags
func (c *Server) parseRatingFlags(flags *models.GlobalFlags, q url.Values) {
	if ratings := q["rating"]; len(ratings) > 0 {
		var clauses []string
		for _, rating := range ratings {
			if r, err := strconv.Atoi(rating); err == nil {
				if r == 0 {
					clauses = append(clauses, "(score IS NULL OR score = 0)")
				} else {
					clauses = append(clauses, fmt.Sprintf("score = %d", r))
				}
			}
		}
		if len(clauses) > 0 {
			if len(clauses) == 1 {
				flags.Where = append(flags.Where, clauses[0])
			} else {
				flags.Where = append(flags.Where, "("+strings.Join(clauses, " OR ")+")")
			}
		}
	}
}

// rejectInvalidFlags writes a 400 when ParseFlags found an unsafe parameter.
func rejectInvalidFlags(w http.ResponseWriter, flags models.GlobalFlags) bool {
	if flags.ValidationError == "" {
		return false
	}
	sendError(w, http.StatusBadRequest, flags.ValidationError)
	return true
}

func setFlagValidationError(flags *models.GlobalFlags, err error) {
	if flags.ValidationError == "" {
		flags.ValidationError = err.Error()
	}
}

// parseScoreFlags extracts min/max score and unplayed flags
func (c *Server) parseScoreFlags(flags *models.GlobalFlags, q url.Values) {
	parseScore := func(name, operator string) {
		raw := q.Get(name)
		if raw == "" {
			return
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			setFlagValidationError(flags, fmt.Errorf("invalid %s value %q", name, raw))
			return
		}
		flags.Where = append(flags.Where, operator+" "+strconv.FormatFloat(value, 'g', -1, 64))
	}
	parseScore("min_score", "score >=")
	parseScore("max_score", "score <=")
	if unplayed := q.Get("unplayed"); unplayed == "true" {
		flags.Where = append(flags.Where, "COALESCE(play_count, 0) = 0 AND COALESCE(playhead, 0) = 0")
	}
}

// parseSortFlags extracts sorting-related flags
func (c *Server) parseSortFlags(flags *models.GlobalFlags, q url.Values) {
	if seed := strings.TrimSpace(q.Get("random_seed")); seed != "" {
		parsed, err := strconv.ParseUint(seed, 10, 64)
		if err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid random_seed: %w", err))
		} else {
			flags.RandomSeed = parsed
		}
	}

	if sortBy := strings.TrimSpace(q.Get("sort")); sortBy != "" {
		if err := query.ValidateSortConfig(sortBy); err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid sort: %w", err))
		} else {
			flags.SortBy = sortBy
			if sortBy == "random" || sortBy == "random()" ||
				sortBy == "RANDOM" || sortBy == "RANDOM()" {
				flags.Random = true
			}
		}
	}
	// Support complex sorting with sort_fields array (JSON) or comma-separated string
	if sortFields := q.Get("sort_fields"); sortFields != "" {
		config := sortFields
		if strings.HasPrefix(sortFields, "[") {
			var fieldList []string
			if err := json.Unmarshal([]byte(sortFields), &fieldList); err != nil {
				setFlagValidationError(flags, fmt.Errorf("invalid sort_fields: %w", err))
				return
			}
			config = strings.Join(fieldList, ",")
		}
		if err := query.ValidateSortConfig(config); err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid sort_fields: %w", err))
		} else {
			flags.PlayInOrder = config
		}
	}
	// Also support sort_order for explicit direction
	if sortDesc := q.Get("sort_desc"); sortDesc != "" {
		descFields := make(map[string]bool)
		for f := range strings.SplitSeq(sortDesc, ",") {
			f = strings.TrimSpace(f)
			if err := query.ValidateSortConfig(f); err != nil {
				setFlagValidationError(flags, fmt.Errorf("invalid sort_desc: %w", err))
				return
			}
			descFields[f] = true
		}
		if flags.PlayInOrder != "" {
			var newOrder []string
			for f := range strings.SplitSeq(flags.PlayInOrder, ",") {
				f = strings.TrimSpace(f)
				f = strings.TrimPrefix(f, "-")
				f = strings.TrimSuffix(f, " desc")
				f = strings.TrimSuffix(f, " asc")
				if descFields[f] {
					newOrder = append(newOrder, "-"+f)
				} else {
					newOrder = append(newOrder, f)
				}
			}
			flags.PlayInOrder = strings.Join(newOrder, ",")
		}
	}
	if reverse := q.Get("reverse"); reverse == "true" {
		flags.Reverse = true
	}
	if q.Get("random") == "true" {
		flags.Random = true
	}
	if q.Get("nat_sort") == "true" || q.Get("natsort") == "true" {
		flags.NatSort = true
	}
	if rerank := q.Get("rerank"); rerank != "" {
		flags.ReRank = rerank
	}
}

// parsePaginationFlags extracts limit and offset flags
func (c *Server) parsePaginationFlags(flags *models.GlobalFlags, q url.Values) {
	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l >= 0 {
			if c.MaxLimit > 0 && l > c.MaxLimit {
				l = c.MaxLimit
			}
			flags.Limit = l
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			flags.Offset = o
		}
	}
}

// parseSizeFlags extracts size filter flags
func (c *Server) parseSizeFlags(flags *models.GlobalFlags, q url.Values) {
	withDefaultUnit := func(raw string) string {
		raw = strings.TrimSpace(raw)
		if _, err := strconv.ParseFloat(raw, 64); err == nil {
			return raw + "MB"
		}
		return raw
	}
	if minSize := q.Get("min_size"); minSize != "" {
		flags.Size = append(flags.Size, ">"+withDefaultUnit(minSize))
	}
	if maxSize := q.Get("max_size"); maxSize != "" {
		flags.Size = append(flags.Size, "<"+withDefaultUnit(maxSize))
	}
	if sizes := q["size"]; len(sizes) > 0 {
		flags.Size = append(flags.Size, sizes...)
	}
}

// parseDurationFlags extracts duration filter flags
func (c *Server) parseDurationFlags(flags *models.GlobalFlags, q url.Values) {
	if minDuration := q.Get("min_duration"); minDuration != "" {
		flags.Duration = append(flags.Duration, ">"+minDuration+"min")
	}
	if maxDuration := q.Get("max_duration"); maxDuration != "" {
		flags.Duration = append(flags.Duration, "<"+maxDuration+"min")
	}
	if durations := q["duration"]; len(durations) > 0 {
		flags.Duration = append(flags.Duration, durations...)
	}
}

// parseTimeFlags extracts time-based filter flags
func (c *Server) parseTimeFlags(flags *models.GlobalFlags, q url.Values) {
	if minModified := q.Get("min_modified"); minModified != "" {
		flags.ModifiedAfter = minModified
	}
	if maxModified := q.Get("max_modified"); maxModified != "" {
		flags.ModifiedBefore = maxModified
	}
	if minCreated := q.Get("min_created"); minCreated != "" {
		flags.CreatedAfter = minCreated
	}
	if maxCreated := q.Get("max_created"); maxCreated != "" {
		flags.CreatedBefore = maxCreated
	}
	if minDownloaded := q.Get("min_downloaded"); minDownloaded != "" {
		flags.DownloadedAfter = minDownloaded
	}
	if maxDownloaded := q.Get("max_downloaded"); maxDownloaded != "" {
		flags.DownloadedBefore = maxDownloaded
	}
	if deletedAfter := q.Get("deleted_after"); deletedAfter != "" {
		flags.DeletedAfter = deletedAfter
	}
	if deletedBefore := q.Get("deleted_before"); deletedBefore != "" {
		flags.DeletedBefore = deletedBefore
	}
	if playedAfter := q.Get("played_after"); playedAfter != "" {
		flags.PlayedAfter = playedAfter
	}
	if playedBefore := q.Get("played_before"); playedBefore != "" {
		flags.PlayedBefore = playedBefore
	}

	if modified := q["modified"]; len(modified) > 0 {
		flags.Modified = append(flags.Modified, modified...)
	}
	if created := q["created"]; len(created) > 0 {
		flags.Created = append(flags.Created, created...)
	}
	if downloaded := q["downloaded"]; len(downloaded) > 0 {
		flags.Downloaded = append(flags.Downloaded, downloaded...)
	}

	for _, m := range q["modified"] {
		for part := range strings.SplitSeq(m, ",") {
			if strings.HasPrefix(part, "+") {
				flags.ModifiedAfter = part[1:]
			} else if strings.HasPrefix(part, "-") {
				flags.ModifiedBefore = part[1:]
			}
		}
	}
	for _, m := range q["created"] {
		for part := range strings.SplitSeq(m, ",") {
			if strings.HasPrefix(part, "+") {
				flags.CreatedAfter = part[1:]
			} else if strings.HasPrefix(part, "-") {
				flags.CreatedBefore = part[1:]
			}
		}
	}
	for _, m := range q["downloaded"] {
		for part := range strings.SplitSeq(m, ",") {
			if strings.HasPrefix(part, "+") {
				flags.DownloadedAfter = part[1:]
			} else if strings.HasPrefix(part, "-") {
				flags.DownloadedBefore = part[1:]
			}
		}
	}
}

// parseMediaFlags extracts media type and status filter flags.
func (c *Server) parseMediaFlags(flags *models.GlobalFlags, q url.Values) {
	c.parseMediaTypeFlags(flags, q)
	c.parseMediaStatusFlags(flags, q)
	c.parseMediaExtraFlags(flags, q)
}

func (c *Server) parseMediaTypeFlags(flags *models.GlobalFlags, q url.Values) {
	mediaTypes := q["media_type"]
	if len(mediaTypes) == 0 {
		mediaTypes = q["type"]
	}
	for _, t := range mediaTypes {
		switch t {
		case "video":
			flags.VideoOnly = true
		case "audio":
			flags.AudioOnly = true
		case "image":
			flags.ImageOnly = true
		case "text":
			flags.TextOnly = true
		}
	}

	if q.Get("video") == "true" {
		flags.VideoOnly = true
	}
	if q.Get("audio") == "true" {
		flags.AudioOnly = true
	}
	if q.Get("image") == "true" {
		flags.ImageOnly = true
	}
	if q.Get("text") == "true" {
		flags.TextOnly = true
	}
}

func (c *Server) parseMediaStatusFlags(flags *models.GlobalFlags, q url.Values) {
	if q.Get("unfinished") == "true" {
		flags.Unfinished = true
	}
	if q.Get("in_progress") == "true" || q.Get("in-progress") == "true" {
		flags.InProgress = true
	}
	if q.Get("completed") == "true" {
		flags.Completed = true
	}
	if partial := q.Get("partial"); partial != "" {
		flags.Partial = partial
	}
	if raw := q.Get("watched"); raw != "" {
		watched, err := strconv.ParseBool(raw)
		if err != nil {
			setFlagValidationError(flags, fmt.Errorf("invalid watched value %q", raw))
		} else {
			flags.Watched = &watched
		}
	}
	if raw := q.Get("play_count_min"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			flags.PlayCountMin = &value
		} else {
			setFlagValidationError(flags, fmt.Errorf("invalid play_count_min value %q", raw))
		}
	}
	if raw := q.Get("play_count_max"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			flags.PlayCountMax = &value
		} else {
			setFlagValidationError(flags, fmt.Errorf("invalid play_count_max value %q", raw))
		}
	}
}

func (c *Server) parseMediaExtraFlags(flags *models.GlobalFlags, q url.Values) {
	if q.Get("portrait") == "true" {
		flags.Portrait = true
	}
	if q.Get("online_media_only") == "true" || q.Get("online-only") == "true" {
		flags.OnlineMediaOnly = true
	}
	if q.Get("local_media_only") == "true" || q.Get("local-only") == "true" {
		flags.LocalMediaOnly = true
	}
	if q.Get("captions") == "true" || q.Get("view") == "captions" || q.Get("with_captions") == "true" {
		flags.WithCaptions = true
	}
	if q.Get("exists") == "true" {
		flags.Exists = true
	}
	if q.Get("flexible_search") == "true" || q.Get("flexible-search") == "true" {
		flags.FlexibleSearch = true
	}
	if q.Get("exact") == "true" {
		flags.Exact = true
	}
	if siblings := q.Get("fetch_siblings"); siblings != "" {
		flags.FetchSiblings = siblings
	}
	if raw := q.Get("fetch_siblings_max"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			flags.FetchSiblingsMax = value
		}
	}
	if durationFromSize := q.Get("duration_from_size"); durationFromSize != "" {
		flags.DurationFromSize = durationFromSize
	}
	if ext := q["ext"]; len(ext) > 0 {
		flags.Ext = append(flags.Ext, ext...)
	}
}

// parseStatusFlags extracts status-related flags (trash, all, episodes, group_by)
func (c *Server) parseStatusFlags(flags *models.GlobalFlags, q url.Values) {
	if all := q.Get("all"); all == "true" {
		flags.All = true
	}
	if q.Get("trash") == "true" || q.Get("only_deleted") == "true" {
		flags.OnlyDeleted = true
		flags.HideDeleted = false
	}
	if q.Get("hide_deleted") == "false" {
		flags.HideDeleted = false
	}
	if episodes := q["episodes"]; len(episodes) > 0 {
		flags.FileCounts = strings.Join(episodes, ",")
	}
	if groupBy := q.Get("group_by"); groupBy == "parent" {
		flags.GroupByParent = true
	}
}

// parseDatabaseFlags extracts database filter flags
func (c *Server) parseDatabaseFlags(flags *models.GlobalFlags, q url.Values) {
	if dbs := q["db"]; len(dbs) > 0 {
		flags.Databases = dbs
	}
}

// ParseFlags extracts query parameters into GlobalFlags
func (c *Server) ParseFlags(r *http.Request) models.GlobalFlags {
	flags := c.GetGlobalFlags()
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		flags.ValidationError = "invalid query string: " + err.Error()
		return flags
	}

	c.parseSearchFlags(&flags, q)
	c.parseCategoryFlags(&flags, q)
	c.parseLanguageFlags(&flags, q)
	c.parsePathFlags(&flags, q)
	c.parseRatingFlags(&flags, q)
	c.parseScoreFlags(&flags, q)
	c.parseSortFlags(&flags, q)
	c.parsePaginationFlags(&flags, q)
	c.parseSizeFlags(&flags, q)
	c.parseDurationFlags(&flags, q)
	c.parseTimeFlags(&flags, q)
	c.parseMediaFlags(&flags, q)
	c.parseStatusFlags(&flags, q)
	c.parseDatabaseFlags(&flags, q)

	return flags
}

// filterDatabases validates and filters the requested databases against the server's allowed list
// Returns an error if any requested database is not in the allowed list
func (c *Server) filterDatabases(requested []string) ([]string, error) {
	// If no specific databases requested, use all configured databases.
	if len(requested) == 0 {
		return c.Databases, nil
	}
	// The web client uses this sentinel when the user excludes every database.
	if len(requested) == 1 && requested[0] == "__none__" {
		return []string{}, nil
	}

	// Build a set of allowed databases for quick lookup
	allowedSet := make(map[string]bool, len(c.Databases))
	for _, db := range c.Databases {
		allowedSet[db] = true
	}

	// Validate each requested database
	var filtered []string
	for _, db := range requested {
		if !allowedSet[db] {
			return nil, fmt.Errorf("database not in allowed list: %s", db)
		}
		filtered = append(filtered, db)
	}

	return filtered, nil
}

// getDBs returns the list of databases to query based on request flags
// It validates that all requested databases are in the server's allowed list
func (c *Server) getDBs(flags models.GlobalFlags) ([]string, error) {
	return c.filterDatabases(flags.Databases)
}
