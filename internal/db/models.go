package db

import (
	"database/sql"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

type Captions struct {
	MediaPath string          `json:"media_path"`
	Time      sql.NullFloat64 `json:"time"`
	Text      sql.NullString  `json:"text"`
}

type Media struct {
	Path            string          `json:"path"`
	PathTokenized   sql.NullString  `json:"path_tokenized"`
	Title           sql.NullString  `json:"title"`
	Duration        sql.NullInt64   `json:"duration"`
	Size            sql.NullInt64   `json:"size"`
	TimeCreated     sql.NullInt64   `json:"time_created"`
	TimeModified    sql.NullInt64   `json:"time_modified"`
	TimeDeleted     sql.NullInt64   `json:"time_deleted"`
	TimeFirstPlayed sql.NullInt64   `json:"time_first_played"`
	TimeLastPlayed  sql.NullInt64   `json:"time_last_played"`
	PlayCount       sql.NullInt64   `json:"play_count"`
	Playhead        sql.NullInt64   `json:"playhead"`
	MediaType       sql.NullString  `json:"media_type"`
	Width           sql.NullInt64   `json:"width"`
	Height          sql.NullInt64   `json:"height"`
	Fps             sql.NullFloat64 `json:"fps"`
	VideoCodecs     sql.NullString  `json:"video_codecs"`
	AudioCodecs     sql.NullString  `json:"audio_codecs"`
	SubtitleCodecs  sql.NullString  `json:"subtitle_codecs"`
	VideoCount      sql.NullInt64   `json:"video_count"`
	AudioCount      sql.NullInt64   `json:"audio_count"`
	SubtitleCount   sql.NullInt64   `json:"subtitle_count"`
	Album           sql.NullString  `json:"album"`
	Artist          sql.NullString  `json:"artist"`
	Genre           sql.NullString  `json:"genre"`
	Categories      sql.NullString  `json:"categories"`
	Description     sql.NullString  `json:"description"`
	Language        sql.NullString  `json:"language"`
	TimeDownloaded  sql.NullInt64   `json:"time_downloaded"`
	Score           sql.NullFloat64 `json:"score"`

	// Hash and processing status
	Fasthash  sql.NullString `json:"fasthash"`
	Sha256    sql.NullString `json:"sha256"`
	IsDeduped sql.NullInt64  `json:"is_deduped"`
}

// ToModel converts the database row into the domain model used by callers.
func (m Media) ToModel() models.Media {
	return models.Media{
		Path:            m.Path,
		PathTokenized:   models.NullStringPtr(m.PathTokenized),
		Title:           models.NullStringPtr(m.Title),
		Duration:        models.NullInt64Ptr(m.Duration),
		Size:            models.NullInt64Ptr(m.Size),
		TimeCreated:     models.NullInt64Ptr(m.TimeCreated),
		TimeModified:    models.NullInt64Ptr(m.TimeModified),
		TimeDeleted:     models.NullInt64Ptr(m.TimeDeleted),
		TimeFirstPlayed: models.NullInt64Ptr(m.TimeFirstPlayed),
		TimeLastPlayed:  models.NullInt64Ptr(m.TimeLastPlayed),
		PlayCount:       models.NullInt64Ptr(m.PlayCount),
		Playhead:        models.NullInt64Ptr(m.Playhead),
		MediaType:       models.NullStringPtr(m.MediaType),
		Width:           models.NullInt64Ptr(m.Width),
		Height:          models.NullInt64Ptr(m.Height),
		Fps:             models.NullFloat64Ptr(m.Fps),
		VideoCodecs:     models.NullStringPtr(m.VideoCodecs),
		AudioCodecs:     models.NullStringPtr(m.AudioCodecs),
		SubtitleCodecs:  models.NullStringPtr(m.SubtitleCodecs),
		VideoCount:      models.NullInt64Ptr(m.VideoCount),
		AudioCount:      models.NullInt64Ptr(m.AudioCount),
		SubtitleCount:   models.NullInt64Ptr(m.SubtitleCount),
		Album:           models.NullStringPtr(m.Album),
		Artist:          models.NullStringPtr(m.Artist),
		Genre:           models.NullStringPtr(m.Genre),
		Categories:      models.NullStringPtr(m.Categories),
		Description:     models.NullStringPtr(m.Description),
		Language:        models.NullStringPtr(m.Language),
		TimeDownloaded:  models.NullInt64Ptr(m.TimeDownloaded),
		Score:           models.NullFloat64Ptr(m.Score),
		Fasthash:        models.NullStringPtr(m.Fasthash),
		Sha256:          models.NullStringPtr(m.Sha256),
		IsDeduped:       models.NullInt64Ptr(m.IsDeduped),
	}
}

type Playlists struct {
	ID              int64          `json:"id"`
	Path            sql.NullString `json:"path"`
	Title           sql.NullString `json:"title"`
	ExtractorKey    sql.NullString `json:"extractor_key"`
	ExtractorConfig sql.NullString `json:"extractor_config"`
	TimeDeleted     sql.NullInt64  `json:"time_deleted"`
}

// GetAllCaptionsRow represents a row from GetAllCaptions query.
type GetAllCaptionsRow struct {
	MediaPath string          `json:"media_path"`
	Time      sql.NullFloat64 `json:"time"`
	Text      sql.NullString  `json:"text"`
	Title     sql.NullString  `json:"title"`
	MediaType sql.NullString  `json:"media_type"`
	Size      sql.NullInt64   `json:"size"`
	Duration  sql.NullInt64   `json:"duration"`
}

type GetAllCaptionsOrderedRow struct {
	MediaPath string          `json:"media_path"`
	Time      sql.NullFloat64 `json:"time"`
	Text      sql.NullString  `json:"text"`
	Title     sql.NullString  `json:"title"`
	MediaType sql.NullString  `json:"media_type"`
	Size      sql.NullInt64   `json:"size"`
	Duration  sql.NullInt64   `json:"duration"`
}
