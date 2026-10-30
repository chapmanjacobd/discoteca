package db

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

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
	MediaType sql.NullString
	Size      sql.NullInt64
	Duration  sql.NullInt64
	Rank      float64
}

// SearchCaptions searches captions using FTS5 with trigram-compatible queries
func (q *Queries) SearchCaptions(ctx context.Context, arg SearchCaptionsParams) ([]SearchCaptionsRow, error) {
	if arg.Query == "" {
		return nil, nil
	}

	query := `
SELECT c.media_path, c.time, c.text, m.title, m.media_type, m.size, m.duration
FROM captions c
JOIN (
    SELECT rowid, rank FROM captions_fts
    WHERE captions_fts MATCH ?
    ORDER BY rank
    LIMIT ?
) fts ON c.rowid = fts.rowid
JOIN media m ON c.media_path = m.path
WHERE m.time_deleted = 0
AND (? = 0 OR m.media_type = 'video')
AND (? = 0 OR m.media_type IN ('audio', 'audiobook'))
AND (? = 0 OR m.media_type = 'image')
AND (? = 0 OR m.media_type = 'text')
ORDER BY fts.rank
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

	rows, err := q.db.QueryContext(ctx, query, arg.Query, arg.Limit, videoOnly, audioOnly, imageOnly, textOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchCaptionsRow
	for rows.Next() {
		var r SearchCaptionsRow
		err := rows.Scan(&r.MediaPath, &r.Time, &r.Text, &r.Title, &r.MediaType, &r.Size, &r.Duration)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
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
