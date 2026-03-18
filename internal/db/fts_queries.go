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
	if arg.Query == "" {
		return nil, nil
	}

	query := `
SELECT 
    m.path, m.path_tokenized, m.title, m.duration, m.size, 
    m.time_created, m.time_modified, m.time_deleted, 
    m.time_first_played, m.time_last_played, m.play_count, m.playhead, 
    m.type, m.width, m.height, m.fps, 
    m.video_codecs, m.audio_codecs, m.subtitle_codecs, 
    m.video_count, m.audio_count, m.subtitle_count, 
    m.album, m.artist, m.genre, m.categories, 
    m.description, m.language, m.time_downloaded, m.score
FROM media m
JOIN (
    SELECT rowid, rank FROM media_fts
    WHERE media_fts MATCH ? AND media_fts.time_deleted = 0
    ORDER BY rank
    LIMIT ?
) fts ON m.rowid = fts.rowid
WHERE m.time_deleted = 0
ORDER BY fts.rank
`
	rows, err := q.db.QueryContext(ctx, query, arg.Query, arg.Limit)
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
	if arg.Query == "" {
		return nil, nil
	}

	query := `
SELECT c.media_path, c.time, c.text, m.title, m.type, m.size, m.duration
FROM captions c
JOIN (
    SELECT rowid, rank FROM captions_fts
    WHERE captions_fts MATCH ?
    ORDER BY rank
    LIMIT ?
) fts ON c.rowid = fts.rowid
JOIN media m ON c.media_path = m.path
WHERE m.time_deleted = 0
AND (? = 0 OR m.type = 'video')
AND (? = 0 OR m.type IN ('audio', 'audiobook'))
AND (? = 0 OR m.type = 'image')
AND (? = 0 OR m.type = 'text')
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
		err := rows.Scan(&r.MediaPath, &r.Time, &r.Text, &r.Title, &r.Type, &r.Size, &r.Duration)
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
