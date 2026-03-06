package models

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discotheque/internal/db"
)

type Media struct {
	Path            string   `json:"path"`
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
	Mood            *string  `json:"mood,omitempty"`
	Bpm             *int64   `json:"bpm,omitempty"`
	Key             *string  `json:"key,omitempty"`
	Decade          *string  `json:"decade,omitempty"`
	Categories      *string  `json:"categories,omitempty"`
	City            *string  `json:"city,omitempty"`
	Country         *string  `json:"country,omitempty"`
	Description     *string  `json:"description,omitempty"`
	Language        *string  `json:"language,omitempty"`

	Webpath        *string  `json:"webpath,omitempty"`
	Uploader       *string  `json:"uploader,omitempty"`
	TimeUploaded   *int64   `json:"time_uploaded,omitempty"`
	TimeDownloaded *int64   `json:"time_downloaded,omitempty"`
	ViewCount      *int64   `json:"view_count,omitempty"`
	NumComments    *int64   `json:"num_comments,omitempty"`
	FavoriteCount  *int64   `json:"favorite_count,omitempty"`
	Score          *float64 `json:"score,omitempty"`
	UpvoteRatio    *float64 `json:"upvote_ratio,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`

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
	ext := filepath.Ext(m.Path)
	base := filepath.Base(m.Path)
	if base == ext {
		return base
	}
	return strings.TrimSuffix(base, ext)
}

func (m *Media) ParentAtDepth(depth int) string {
	parts := strings.Split(filepath.Clean(m.Path), string(filepath.Separator))
	if depth <= 0 {
		return "/"
	}
	if depth >= len(parts)-1 {
		return filepath.Dir(m.Path)
	}
	return strings.Join(parts[:depth+1], string(filepath.Separator))
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
}

// FolderStats aggregates media by folder
type FolderStats struct {
	Path           string        `json:"path"`
	Count          int           `json:"count"`
	TotalSize      int64         `json:"total_size"`
	TotalDuration  int64         `json:"total_duration"`
	AvgSize        int64         `json:"avg_size"`
	AvgDuration    int64         `json:"avg_duration"`
	MedianSize     int64         `json:"median_size"`
	MedianDuration int64         `json:"median_duration"`
	DeletedCount   int           `json:"deleted_count"`
	ExistsCount    int           `json:"exists_count"`
	PlayedCount    int           `json:"played_count"`
	FolderCount    int           `json:"folder_count"`
	Files          []MediaWithDB `json:"files,omitempty"`
}

// Helper functions for mapping from sql.Null types

func FromDB(m db.Media) Media {
	return Media{
		Path:            m.Path,
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
		Mood:            NullStringPtr(m.Mood),
		Bpm:             NullInt64Ptr(m.Bpm),
		Key:             NullStringPtr(m.Key),
		Decade:          NullStringPtr(m.Decade),
		Categories:      NullStringPtr(m.Categories),
		City:            NullStringPtr(m.City),
		Country:         NullStringPtr(m.Country),
		Description:     NullStringPtr(m.Description),
		Language:        NullStringPtr(m.Language),
		Webpath:         NullStringPtr(m.Webpath),
		Uploader:        NullStringPtr(m.Uploader),
		TimeUploaded:    NullInt64Ptr(m.TimeUploaded),
		TimeDownloaded:  NullInt64Ptr(m.TimeDownloaded),
		ViewCount:       NullInt64Ptr(m.ViewCount),
		NumComments:     NullInt64Ptr(m.NumComments),
		FavoriteCount:   NullInt64Ptr(m.FavoriteCount),
		Score:           NullFloat64Ptr(m.Score),
		UpvoteRatio:     NullFloat64Ptr(m.UpvoteRatio),
		Latitude:        NullFloat64Ptr(m.Latitude),
		Longitude:       NullFloat64Ptr(m.Longitude),
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
		Mood:           ToNullString(m.Mood),
		Bpm:            ToNullInt64(m.Bpm),
		Key:            ToNullString(m.Key),
		Decade:         ToNullString(m.Decade),
		Categories:     ToNullString(m.Categories),
		City:           ToNullString(m.City),
		Country:        ToNullString(m.Country),
		Description:    ToNullString(m.Description),
		Language:       ToNullString(m.Language),
		Webpath:        ToNullString(m.Webpath),
		Uploader:       ToNullString(m.Uploader),
		TimeUploaded:   ToNullInt64(m.TimeUploaded),
		TimeDownloaded: ToNullInt64(m.TimeDownloaded),
		ViewCount:      ToNullInt64(m.ViewCount),
		NumComments:    ToNullInt64(m.NumComments),
		FavoriteCount:  ToNullInt64(m.FavoriteCount),
		Score:          ToNullFloat64(m.Score),
		UpvoteRatio:    ToNullFloat64(m.UpvoteRatio),
		Latitude:       ToNullFloat64(m.Latitude),
		Longitude:      ToNullFloat64(m.Longitude),
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
