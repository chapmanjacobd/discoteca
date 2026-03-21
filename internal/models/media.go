package models

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/utils/pathutil"
)

type Media struct {
	Path            string   `json:"path"`
	PathTokenized   *string  `json:"path_tokenized,omitempty"`
	Title           *string  `json:"title,omitempty"`
	Duration        *int64   `json:"duration,omitempty"`
	Size            *int64   `json:"size,omitempty"`
	TimeCreated     *int64   `json:"time_created,omitempty"`
	TimeModified    *int64   `json:"time_modified,omitempty"`
	TimeDeleted     *int64   `json:"time_deleted,omitempty"`
	TimeFirstPlayed *int64   `json:"time_first_played,omitempty"`
	TimeLastPlayed  *int64   `json:"time_last_played,omitempty"`
	PlayCount       *int64   `json:"play_count,omitempty"`
	Playhead        *int64   `json:"playhead,omitempty"`
	MediaType       *string  `json:"media_type,omitempty"`
	Width           *int64   `json:"width,omitempty"`
	Height          *int64   `json:"height,omitempty"`
	Fps             *float64 `json:"fps,omitempty"`
	VideoCodecs     *string  `json:"video_codecs,omitempty"`
	AudioCodecs     *string  `json:"audio_codecs,omitempty"`
	SubtitleCodecs  *string  `json:"subtitle_codecs,omitempty"`
	VideoCount      *int64   `json:"video_count,omitempty"`
	AudioCount      *int64   `json:"audio_count,omitempty"`
	SubtitleCount   *int64   `json:"subtitle_count,omitempty"`
	Album           *string  `json:"album,omitempty"`
	Artist          *string  `json:"artist,omitempty"`
	Genre           *string  `json:"genre,omitempty"`
	Categories      *string  `json:"categories,omitempty"`
	Description     *string  `json:"description,omitempty"`
	Language        *string  `json:"language,omitempty"`
	TimeDownloaded  *int64   `json:"time_downloaded,omitempty"`
	Score           *float64 `json:"score,omitempty"`

	Fasthash  *string `json:"fasthash,omitempty"`
	Sha256    *string `json:"sha256,omitempty"`
	IsDeduped *int64  `json:"is_deduped,omitempty"`

	TrackNumber *int64 `json:"track_number,omitempty"`
}

type Playlist struct {
	ID              int64   `json:"id"`
	Path            *string `json:"path,omitempty"`
	Title           *string `json:"title,omitempty"`
	ExtractorKey    *string `json:"extractor_key,omitempty"`
	ExtractorConfig *string `json:"extractor_config,omitempty"`
	TimeDeleted     *int64  `json:"time_deleted,omitempty"`
	DB              string  `json:"db,omitempty"`
}

func PlaylistFromDB(p db.Playlists, dbPath string) Playlist {
	return Playlist{
		ID:              p.ID,
		Path:            NullStringPtr(p.Path),
		Title:           NullStringPtr(p.Title),
		ExtractorKey:    NullStringPtr(p.ExtractorKey),
		ExtractorConfig: NullStringPtr(p.ExtractorConfig),
		TimeDeleted:     NullInt64Ptr(p.TimeDeleted),
		DB:              dbPath,
	}
}

func (m *Media) Parent() string {
	return filepath.Dir(m.Path)
}

func (m *Media) Stem() string {
	base := filepath.Base(m.Path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" {
		return base
	}
	return stem
}

func (m *Media) Extension() string {
	return strings.ToLower(filepath.Ext(m.Path))
}

func (m *Media) ParentAtDepth(depth int) string {
	parts, isAbs := pathutil.Split(m.Path)

	if len(parts) == 0 {
		return "."
	}

	// Check if first part is a drive letter
	hasDrive := len(parts) > 0 && len(parts[0]) == 2 && parts[0][1] == ':'

	if depth <= 0 {
		if isAbs {
			return pathutil.Prefix(m.Path)
		}
		return "."
	}

	// Calculate how many parts to include
	// For paths with drive: depth 1 = drive + 1 dir = 2 parts
	// For paths without drive: depth 1 = 1 part
	numParts := depth
	if hasDrive {
		numParts = depth + 1 // Include drive letter
	}

	// Cap at available parts (excluding file if depth doesn't reach it)
	// If numParts would include the file, cap at the parent
	maxParts := len(parts)
	if numParts >= maxParts {
		numParts = maxParts - 1 // Don't include the file itself
	}

	if numParts <= 0 {
		if isAbs {
			return pathutil.Prefix(m.Path)
		}
		return "."
	}

	// For paths with drive, use Join without leading separator (drive provides it)
	if hasDrive {
		result := pathutil.Join(parts[:numParts], false)
		return result
	}

	result := pathutil.Join(parts[:numParts], isAbs)
	return result
}

// MediaWithDB wraps Media with the database path it came from
type MediaWithDB struct {
	Media
	DB              string  `json:"db,omitempty"`
	Transcode       bool    `json:"transcode"`
	CaptionText     string  `json:"caption_text"`
	CaptionTime     float64 `json:"caption_time"`
	CaptionCount    int64   `json:"caption_count"`
	CaptionDuration int64   `json:"caption_duration"`
	EpisodeCount    int64   `json:"episode_count"`
	TotalSize       int64   `json:"total_size"`
	TotalDuration   int64   `json:"total_duration"`
}

// FolderStats aggregates media by folder
type FolderStats struct {
	Path           string        `json:"path"`
	Count          int           `json:"count"`
	ExistsCount    int           `json:"exists_count"`
	PlayedCount    int           `json:"played_count"`
	DeletedCount   int           `json:"deleted_count"`
	FolderCount    int           `json:"folder_count"`
	TotalSize      int64         `json:"total_size"`
	TotalDuration  int64         `json:"total_duration"`
	AvgSize        int64         `json:"avg_size"`
	AvgDuration    int64         `json:"avg_duration"`
	MedianSize     int64         `json:"median_size"`
	MedianDuration int64         `json:"median_duration"`
	TimeLastPlayed int64         `json:"time_last_played"`
	Files          []MediaWithDB `json:"files,omitempty"`
}

// DUResponse is the response for the /api/du endpoint
type DUResponse struct {
	Folders     []FolderStats       `json:"folders"`
	Files       []MediaWithDB       `json:"files"`
	TotalCount  int                 `json:"total_count"`      // Total items (folders + files)
	FolderCount int                 `json:"folder_count"`     // Number of folders
	FileCount   int                 `json:"file_count"`       // Number of direct files
	Counts      *FilterBinsResponse `json:"counts,omitempty"` // Filter bins data (when include_counts=true)
}

// Helper functions for mapping from sql.Null types

func FromDB(m db.Media) Media {
	return Media{
		Path:            m.Path,
		PathTokenized:   NullStringPtr(m.PathTokenized),
		Title:           NullStringPtr(m.Title),
		Duration:        NullInt64Ptr(m.Duration),
		Size:            NullInt64Ptr(m.Size),
		TimeCreated:     NullInt64Ptr(m.TimeCreated),
		TimeModified:    NullInt64Ptr(m.TimeModified),
		TimeDeleted:     NullInt64Ptr(m.TimeDeleted),
		TimeFirstPlayed: NullInt64Ptr(m.TimeFirstPlayed),
		TimeLastPlayed:  NullInt64Ptr(m.TimeLastPlayed),
		PlayCount:       NullInt64Ptr(m.PlayCount),
		Playhead:        NullInt64Ptr(m.Playhead),
		MediaType:       NullStringPtr(m.MediaType),
		Width:           NullInt64Ptr(m.Width),
		Height:          NullInt64Ptr(m.Height),
		Fps:             NullFloat64Ptr(m.Fps),
		VideoCodecs:     NullStringPtr(m.VideoCodecs),
		AudioCodecs:     NullStringPtr(m.AudioCodecs),
		SubtitleCodecs:  NullStringPtr(m.SubtitleCodecs),
		VideoCount:      NullInt64Ptr(m.VideoCount),
		AudioCount:      NullInt64Ptr(m.AudioCount),
		SubtitleCount:   NullInt64Ptr(m.SubtitleCount),
		Album:           NullStringPtr(m.Album),
		Artist:          NullStringPtr(m.Artist),
		Genre:           NullStringPtr(m.Genre),
		Categories:      NullStringPtr(m.Categories),
		Description:     NullStringPtr(m.Description),
		Language:        NullStringPtr(m.Language),
		TimeDownloaded:  NullInt64Ptr(m.TimeDownloaded),
		Score:           NullFloat64Ptr(m.Score),
		Fasthash:        NullStringPtr(m.Fasthash),
		Sha256:          NullStringPtr(m.Sha256),
		IsDeduped:       NullInt64Ptr(m.IsDeduped),
	}
}

func FromDBWithDB(m db.Media, dbPath string) MediaWithDB {
	return MediaWithDB{
		Media: FromDB(m),
		DB:    dbPath,
	}
}

func ToDBUpsert(m Media) db.UpsertMediaParams {
	return db.UpsertMediaParams{
		Path:           m.Path,
		PathTokenized:  ToNullString(m.PathTokenized),
		Title:          ToNullString(m.Title),
		Duration:       ToNullInt64(m.Duration),
		Size:           ToNullInt64(m.Size),
		TimeCreated:    ToNullInt64(m.TimeCreated),
		TimeModified:   ToNullInt64(m.TimeModified),
		MediaType:      ToNullString(m.MediaType),
		Width:          ToNullInt64(m.Width),
		Height:         ToNullInt64(m.Height),
		Fps:            ToNullFloat64(m.Fps),
		VideoCodecs:    ToNullString(m.VideoCodecs),
		AudioCodecs:    ToNullString(m.AudioCodecs),
		SubtitleCodecs: ToNullString(m.SubtitleCodecs),
		VideoCount:     ToNullInt64(m.VideoCount),
		AudioCount:     ToNullInt64(m.AudioCount),
		SubtitleCount:  ToNullInt64(m.SubtitleCount),
		Album:          ToNullString(m.Album),
		Artist:         ToNullString(m.Artist),
		Genre:          ToNullString(m.Genre),
		Categories:     ToNullString(m.Categories),
		Description:    ToNullString(m.Description),
		Language:       ToNullString(m.Language),
		TimeDownloaded: ToNullInt64(m.TimeDownloaded),
		Score:          ToNullFloat64(m.Score),
		Fasthash:       ToNullString(m.Fasthash),
		Sha256:         ToNullString(m.Sha256),
		IsDeduped:      ToNullInt64(m.IsDeduped),
	}
}

func ToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

func ToNullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

func NullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func NullInt64Ptr(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}

func NullFloat64Ptr(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	return &nf.Float64
}

// IndexStat represents database index statistics
type IndexStat struct {
	IndexName string `json:"index_name"`
	TableName string `json:"table_name"`
	SizeEst   int64  `json:"size_est,omitempty"`
	Used      bool   `json:"used,omitempty"`
}
