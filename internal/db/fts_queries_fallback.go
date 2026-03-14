//go:build !fts5

package db

import (
	"context"
	"database/sql"
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

// SearchMediaFTS returns empty results when FTS is not enabled
// This allows bleve-only builds to fall back gracefully
func (q *Queries) SearchMediaFTS(ctx context.Context, arg SearchMediaFTSParams) ([]SearchMediaFTSResult, error) {
	// Return empty slice instead of error to allow graceful fallback
	return []SearchMediaFTSResult{}, nil
}

// RankSearchResults does nothing when FTS is not enabled
func RankSearchResults(results []SearchMediaFTSResult, query string) {
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

// SearchCaptions returns empty results when FTS is not enabled
// This allows bleve-only builds to fall back gracefully
func (q *Queries) SearchCaptions(ctx context.Context, arg SearchCaptionsParams) ([]SearchCaptionsRow, error) {
	// Return empty slice instead of error to allow graceful fallback
	return []SearchCaptionsRow{}, nil
}

// RankCaptionsResults does nothing when FTS is not enabled
func RankCaptionsResults(results []SearchCaptionsRow, query string) {
}
