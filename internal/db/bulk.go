package db

import (
	"context"
	"fmt"
	"strings"
)

const SqliteParamLimit = 32766

// BulkUpsertMedia performs a bulk upsert of media items
func (q *Queries) BulkUpsertMedia(ctx context.Context, items []UpsertMediaParams) error {
	if len(items) == 0 {
		return nil
	}

	const columnsCount = 28
	maxBatchSize := min(SqliteParamLimit/columnsCount,
		// Keep it reasonable for memory
		500)

	for i := 0; i < len(items); i += maxBatchSize {
		end := min(i+maxBatchSize, len(items))
		batch := items[i:end]

		if err := q.bulkUpsertMediaBatch(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func (q *Queries) bulkUpsertMediaBatch(ctx context.Context, batch []UpsertMediaParams) error {
	if len(batch) == 0 {
		return nil
	}

	columns := []string{
		"path", "path_tokenized", "title", "duration", "size", "time_created", "time_modified",
		"media_type", "width", "height", "fps", "video_codecs", "audio_codecs", "subtitle_codecs",
		"video_count", "audio_count", "subtitle_count", "album", "artist", "genre", "categories",
		"description", "language", "time_downloaded", "score", "fasthash", "sha256", "is_deduped",
	}

	placeholders := make([]string, len(batch))
	args := make([]any, 0, len(batch)*len(columns))

	for i, item := range batch {
		placeholders[i] = "(" + strings.Repeat("?, ", len(columns)-1) + "?)"
		args = append(
			args,
			item.Path,
			item.PathTokenized,
			item.Title,
			item.Duration,
			item.Size,
			item.TimeCreated,
			item.TimeModified,
			item.MediaType,
			item.Width,
			item.Height,
			item.Fps,
			item.VideoCodecs,
			item.AudioCodecs,
			item.SubtitleCodecs,
			item.VideoCount,
			item.AudioCount,
			item.SubtitleCount,
			item.Album,
			item.Artist,
			item.Genre,
			item.Categories,
			item.Description,
			item.Language,
			item.TimeDownloaded,
			item.Score,
			item.Fasthash,
			item.Sha256,
			item.IsDeduped,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO media (%s)
		VALUES %s
		ON CONFLICT(path) DO UPDATE SET
			path_tokenized = excluded.path_tokenized,
			title = excluded.title,
			duration = excluded.duration,
			size = excluded.size,
			time_modified = excluded.time_modified,
			media_type = excluded.media_type,
			width = excluded.width,
			height = excluded.height,
			fps = excluded.fps,
			video_codecs = excluded.video_codecs,
			audio_codecs = excluded.audio_codecs,
			subtitle_codecs = excluded.subtitle_codecs,
			video_count = excluded.video_count,
			audio_count = excluded.audio_count,
			subtitle_count = excluded.subtitle_count,
			album = excluded.album,
			artist = excluded.artist,
			genre = excluded.genre,
			categories = excluded.categories,
			description = excluded.description,
			language = excluded.language,
			time_downloaded = COALESCE(media.time_downloaded, excluded.time_downloaded),
			score = excluded.score,
			fasthash = excluded.fasthash,
			sha256 = excluded.sha256,
			is_deduped = excluded.is_deduped,
			time_deleted = 0
	`, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	_, err := q.db.ExecContext(ctx, query, args...)
	return err
}

// BulkInsertCaptions performs a bulk insert of captions
func (q *Queries) BulkInsertCaptions(ctx context.Context, items []InsertCaptionParams) error {
	if len(items) == 0 {
		return nil
	}

	const columnsCount = 3
	maxBatchSize := min(SqliteParamLimit/columnsCount,
		// Keep it reasonable
		5000)

	for i := 0; i < len(items); i += maxBatchSize {
		end := min(i+maxBatchSize, len(items))
		batch := items[i:end]

		if err := q.bulkInsertCaptionsBatch(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func (q *Queries) bulkInsertCaptionsBatch(ctx context.Context, batch []InsertCaptionParams) error {
	if len(batch) == 0 {
		return nil
	}

	placeholders := make([]string, len(batch))
	args := make([]any, 0, len(batch)*3)

	for i, item := range batch {
		placeholders[i] = "(?, ?, ?)"
		args = append(args, item.MediaPath, item.Time, item.Text)
	}

	query := fmt.Sprintf("INSERT INTO captions (media_path, time, text) VALUES %s", strings.Join(placeholders, ", "))

	_, err := q.db.ExecContext(ctx, query, args...)
	return err
}
