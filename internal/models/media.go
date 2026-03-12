package models

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

type Media struct {
	Path            string   `json:"path"`
	FtsPath         *string  `json:"fts_path,omitempty"`
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
	Type            *string  `json:"type,omitempty"`
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

	TrackNumber *int64 `json:"track_number,omitempty"`
}

var DefaultCategories = map[string][]string{
	"sports":      {"sports?", "football", "soccer", "basketball", "tennis", "olympics", "training"},
	"fitness":     {"workout", "fitness", "gym", "yoga", "pilates", "exercise", "bodybuilding", "cardio"},
	"documentary": {"documentaries", "documentary", "docu", "history", "biography", "nature", "science", "planet", "wildlife", "factual"},
	"comedy":      {"comedy", "comedies", "standup", "funny", "sitcom", "humor", "prank", "roast", "satire"},
	"music":       {"music", "concerts?", "performance", "live", "musical", "video clip", "remix(es)?", "feat", "official video", "soundtracks?"},
	"educational": {"educational", "tutorials?", "lessons?", "lectures?", "courses?", "learning", "how to", "explainers?", "masterclass(es)?"},
	"news":        {"news", "reports?", "politics", "interviews?", "journalists?", "coverage", "current affairs", "broadcasts?", "press release"},
	"gaming":      {"gaming", "gameplay", "walkthroughs?", "playthroughs?", "twitch", "nintendo", "playstation", "xbox", "steam", "speedruns?", "lets play"},
	"tech":        {"tech", "technology", "software", "hardware", "programming", "coding", "reviews?", "unboxings?", "gadgets?", "silicon"},
	"audiobook":   {"audiobooks?", "audio book", "narrated", "reading", "unabridged"},
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
	slashPath := filepath.ToSlash(m.Path)
	isAbs := strings.HasPrefix(slashPath, "/") || (len(slashPath) >= 2 && slashPath[1] == ':')

	dir := filepath.ToSlash(filepath.Dir(m.Path))
	if dir == "." || dir == "" {
		if depth <= 0 && isAbs {
			vol := filepath.VolumeName(m.Path)
			if vol != "" {
				return vol + string(filepath.Separator)
			}
			return string(filepath.Separator)
		}
		return "."
	}

	if depth <= 0 {
		if isAbs {
			vol := filepath.VolumeName(m.Path)
			if vol != "" {
				return vol + string(filepath.Separator)
			}
			return string(filepath.Separator)
		}
		return "."
	}

	vol := filepath.ToSlash(filepath.VolumeName(m.Path))
	relDir := strings.TrimPrefix(dir, vol)
	parts := strings.Split(strings.Trim(relDir, "/"), "/")

	if depth > len(parts) {
		depth = len(parts)
	}

	res := strings.Join(parts[:depth], "/")
	if vol != "" {
		return filepath.FromSlash(vol + "/" + res)
	}
	if isAbs {
		return filepath.FromSlash("/" + res)
	}
	return filepath.FromSlash(res)
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

// Helper functions for mapping from sql.Null types

func FromDB(m db.Media) Media {
	return Media{
		Path:            m.Path,
		FtsPath:         NullStringPtr(m.FtsPath),
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
		Type:            NullStringPtr(m.Type),
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
		FtsPath:        ToNullString(m.FtsPath),
		Title:          ToNullString(m.Title),
		Duration:       ToNullInt64(m.Duration),
		Size:           ToNullInt64(m.Size),
		TimeCreated:    ToNullInt64(m.TimeCreated),
		TimeModified:   ToNullInt64(m.TimeModified),
		Type:           ToNullString(m.Type),
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
