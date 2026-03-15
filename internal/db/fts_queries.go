//go:build fts5

package db

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

// SearchMediaFTSParams are parameters for FTS media search
type SearchMediaFTSParams struct {
	Query string
	Limit int64
}

// SearchMediaFTSResult is a Media item with optional rank
type SearchMediaFTSResult struct {
	Media Media
	Rank  float64
}

// SearchMediaFTS searches media using FTS5 with trigram-compatible queries
func (q *Queries) SearchMediaFTS(ctx context.Context, arg SearchMediaFTSParams) ([]SearchMediaFTSResult, error) {
	// Use trigram-compatible query (3-char terms for detail=none)
	// No ORDER BY - ranking done in Go for better control
	ftsQuery := tokenizeForFTS5(arg.Query)
	if ftsQuery == "" {
		return nil, nil
	}

	query := `
SELECT m.* FROM media m, media_fts
WHERE m.rowid = media_fts.rowid
AND media_fts MATCH ?
AND m.time_deleted = 0
`
	rows, err := q.db.QueryContext(ctx, query, ftsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchMediaFTSResult
	for rows.Next() {
		var m Media
		err := rows.Scan(
			&m.Path,
			&m.PathTokenized,
			&m.Title,
			&m.Duration,
			&m.Size,
			&m.TimeCreated,
			&m.TimeModified,
			&m.TimeDeleted,
			&m.TimeFirstPlayed,
			&m.TimeLastPlayed,
			&m.PlayCount,
			&m.Playhead,
			&m.Type,
			&m.Width,
			&m.Height,
			&m.Fps,
			&m.VideoCodecs,
			&m.AudioCodecs,
			&m.SubtitleCodecs,
			&m.VideoCount,
			&m.AudioCount,
			&m.SubtitleCount,
			&m.Album,
			&m.Artist,
			&m.Genre,
			&m.Categories,
			&m.Description,
			&m.Language,
			&m.TimeDownloaded,
			&m.Score,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, SearchMediaFTSResult{Media: m, Rank: 0})
	}

	return results, rows.Err()
}

// RankSearchResults applies in-memory ranking to search results
// This provides better relevance scoring than FTS5 with trigram + detail=none
func RankSearchResults(results []SearchMediaFTSResult, query string) {
	if len(results) == 0 || query == "" {
		return
	}

	queryLower := strings.ToLower(query)

	for i := range results {
		score := 0.0

		title := strings.ToLower(results[i].Media.Title.String)
		desc := strings.ToLower(results[i].Media.Description.String)
		path := strings.ToLower(results[i].Media.Path)

		// Count query occurrences in each field
		titleCount := float64(strings.Count(title, queryLower))
		descCount := float64(strings.Count(desc, queryLower))
		pathCount := float64(strings.Count(path, queryLower))

		// Weighted scoring: title > path > description
		score += titleCount * 10.0
		score += pathCount * 5.0
		score += descCount * 1.0

		// Bonus for exact title match
		if strings.Contains(title, queryLower) && titleCount > 0 {
			score += 5.0
		}

		results[i].Rank = score
	}

	// Sort by rank descending
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Rank != results[j].Rank {
			return results[i].Rank > results[j].Rank
		}
		// Tiebreaker: alphabetical by path
		return results[i].Media.Path < results[j].Media.Path
	})
}

// SearchCaptionsParams are parameters for caption search
type SearchCaptionsParams struct {
	Query     string
	VideoOnly bool
	AudioOnly bool
	ImageOnly bool
	TextOnly  bool
	Limit     int64
}

// SearchCaptionsRow represents a row from caption search with optional rank
type SearchCaptionsRow struct {
	MediaPath string
	Time      sql.NullFloat64
	Text      sql.NullString
	Title     sql.NullString
	Type      sql.NullString
	Size      sql.NullInt64
	Duration  sql.NullInt64
	Rank      float64
}

// SearchCaptions searches captions using FTS5 with trigram-compatible queries
func (q *Queries) SearchCaptions(ctx context.Context, arg SearchCaptionsParams) ([]SearchCaptionsRow, error) {
	// Tokenize query for trigram-compatible FTS5 search
	// Convert "hello world" to "hel OR wor" for detail=none trigram index
	ftsQuery := tokenizeForFTS5(arg.Query)
	if ftsQuery == "" {
		return nil, nil
	}

	query := `
SELECT c.media_path, c.time, c.text, m.title, m.type, m.size, m.duration
FROM captions c, captions_fts, media m
WHERE c.rowid = captions_fts.rowid
AND c.media_path = m.path
AND captions_fts MATCH ?
AND m.time_deleted = 0
AND c.text IS NOT NULL AND c.text != ''
AND (? = 0 OR m.type = 'video')
AND (? = 0 OR m.type IN ('audio', 'audiobook'))
AND (? = 0 OR m.type = 'image')
AND (? = 0 OR m.type = 'text')
`
	videoOnly := 0
	audioOnly := 0
	imageOnly := 0
	textOnly := 0
	if arg.VideoOnly {
		videoOnly = 1
	}
	if arg.AudioOnly {
		audioOnly = 1
	}
	if arg.ImageOnly {
		imageOnly = 1
	}
	if arg.TextOnly {
		textOnly = 1
	}

	rows, err := q.db.QueryContext(ctx, query, ftsQuery, videoOnly, audioOnly, imageOnly, textOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchCaptionsRow
	for rows.Next() {
		var r SearchCaptionsRow
		err := rows.Scan(&r.MediaPath, &r.Time, &r.Text, &r.Title, &r.Type, &r.Size, &r.Duration)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// tokenizeForFTS5 converts a search query to trigram-compatible format
// "hello world" -> "hel OR wor"
// This works with FTS5 trigram indexes using detail=none
func tokenizeForFTS5(query string) string {
	if query == "" {
		return ""
	}

	// Remove phrase quotes - we'll search for individual trigrams
	query = strings.ReplaceAll(query, `"`, " ")
	query = strings.ReplaceAll(query, `'`, " ")

	// Remove FTS5 operators that don't work with detail=none
	query = strings.ReplaceAll(query, "(", " ")
	query = strings.ReplaceAll(query, ")", " ")
	query = strings.ReplaceAll(query, ":", " ")
	query = strings.ReplaceAll(query, "NEAR", " ")

	// Split into terms
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return ""
	}

	// Convert each term to its first 3 chars (trigram)
	var trigrams []string
	for _, term := range terms {
		term = strings.TrimSpace(term)
		upper := strings.ToUpper(term)

		// Pass through boolean operators
		if upper == "OR" || upper == "AND" || upper == "NOT" {
			trigrams = append(trigrams, upper)
			continue
		}

		// Skip very short terms
		if len(term) < 3 {
			continue
		}

		trigrams = append(trigrams, term[:3])
	}

	if len(trigrams) == 0 {
		return ""
	}

	// Use OR between trigrams for broader matching
	return strings.Join(trigrams, " OR ")
}

// RankCaptionsResults applies in-memory ranking to caption search results
func RankCaptionsResults(results []SearchCaptionsRow, query string) {
	if len(results) == 0 || query == "" {
		return
	}

	queryLower := strings.ToLower(query)

	for i := range results {
		score := 0.0

		text := strings.ToLower(results[i].Text.String)
		title := strings.ToLower(results[i].Title.String)

		// Count query occurrences
		textCount := float64(strings.Count(text, queryLower))
		titleCount := float64(strings.Count(title, queryLower))

		// Weighted scoring: exact text match > title match
		score += textCount * 10.0
		score += titleCount * 5.0

		// Bonus for exact phrase match in text
		if strings.Contains(text, queryLower) {
			score += 5.0
		}

		results[i].Rank = score
	}

	// Sort by rank descending
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Rank != results[j].Rank {
			return results[i].Rank > results[j].Rank
		}
		// Tiebreaker: by media path and time
		if results[i].MediaPath != results[j].MediaPath {
			return results[i].MediaPath < results[j].MediaPath
		}
		return results[i].Time.Float64 < results[j].Time.Float64
	})
}
