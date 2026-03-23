package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

// mediaColumns is the standard column list for media queries
const mediaColumns = "path, path_tokenized, title, duration, size, time_created, time_modified, time_deleted, time_first_played, time_last_played, play_count, playhead, media_type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs, video_count, audio_count, subtitle_count, album, artist, genre, categories, description, language, time_downloaded, score"

// GetMedia retrieves all non-deleted media
func (q *Queries) GetMedia(ctx context.Context, limit int64) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 ORDER BY path LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetMediaByTypeParams are parameters for GetMediaByType
type GetMediaByTypeParams struct {
	VideoOnly bool
	AudioOnly bool
	ImageOnly bool
	Limit     int64
}

// GetMediaByType retrieves media filtered by type
func (q *Queries) GetMediaByType(ctx context.Context, arg GetMediaByTypeParams) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND ((? AND media_type = 'video') OR (? AND (media_type = 'audio' OR media_type = 'audiobook')) OR (? AND media_type = 'image')) ORDER BY path LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, arg.VideoOnly, arg.AudioOnly, arg.ImageOnly, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetMediaBySizeParams are parameters for GetMediaBySize
type GetMediaBySizeParams struct {
	MinSize int64
	MaxSize int64
	Limit   int64
}

// GetMediaBySize retrieves media filtered by size range
func (q *Queries) GetMediaBySize(ctx context.Context, arg GetMediaBySizeParams) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND size >= ? AND size <= ? ORDER BY size DESC LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, arg.MinSize, arg.MaxSize, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetMediaByDurationParams are parameters for GetMediaByDuration
type GetMediaByDurationParams struct {
	MinDuration int64
	MaxDuration int64
	Limit       int64
}

// GetMediaByDuration retrieves media filtered by duration range
func (q *Queries) GetMediaByDuration(ctx context.Context, arg GetMediaByDurationParams) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND duration >= ? AND duration <= ? ORDER BY duration DESC LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, arg.MinDuration, arg.MaxDuration, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetMediaByPathParams are parameters for GetMediaByPath
type GetMediaByPathParams struct {
	PathPattern string
	Limit       int64
}

// GetMediaByPath retrieves media matching a path pattern
func (q *Queries) GetMediaByPath(ctx context.Context, arg GetMediaByPathParams) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND path LIKE ? ORDER BY path LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, arg.PathPattern, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetMediaByPathPrefixParams are parameters for GetMediaByPathPrefix
type GetMediaByPathPrefixParams struct {
	PathPrefix string
	PathNot    string
	Limit      int64
}

// GetMediaByPathPrefix retrieves media with path starting with prefix but not matching exclusion
func (q *Queries) GetMediaByPathPrefix(ctx context.Context, arg GetMediaByPathPrefixParams) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND path LIKE ? AND path NOT LIKE ? ORDER BY path LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, arg.PathPrefix, arg.PathNot, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetMediaByPathExact retrieves a single media item by exact path match
func (q *Queries) GetMediaByPathExact(ctx context.Context, path string) (Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE path = ? LIMIT 1`
	var i Media
	err := q.db.QueryRowContext(ctx, query, path).Scan(
		&i.Path,
		&i.PathTokenized,
		&i.Title,
		&i.Duration,
		&i.Size,
		&i.TimeCreated,
		&i.TimeModified,
		&i.TimeDeleted,
		&i.TimeFirstPlayed,
		&i.TimeLastPlayed,
		&i.PlayCount,
		&i.Playhead,
		&i.MediaType,
		&i.Width,
		&i.Height,
		&i.Fps,
		&i.VideoCodecs,
		&i.AudioCodecs,
		&i.SubtitleCodecs,
		&i.VideoCount,
		&i.AudioCount,
		&i.SubtitleCount,
		&i.Album,
		&i.Artist,
		&i.Genre,
		&i.Categories,
		&i.Description,
		&i.Language,
		&i.TimeDownloaded,
		&i.Score,
	)
	if err != nil {
		return i, err
	}
	return i, nil
}

// GetAllMediaMetadataRow is a row from GetAllMediaMetadata
type GetAllMediaMetadataRow struct {
	Path         string
	Size         sql.NullInt64
	TimeModified sql.NullInt64
	TimeDeleted  sql.NullInt64
}

// GetAllMediaMetadata retrieves basic metadata for all media
func (q *Queries) GetAllMediaMetadata(ctx context.Context) ([]GetAllMediaMetadataRow, error) {
	const query = `SELECT path, size, time_modified, time_deleted FROM media`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetAllMediaMetadataRow
	for rows.Next() {
		var i GetAllMediaMetadataRow
		if err := rows.Scan(&i.Path, &i.Size, &i.TimeModified, &i.TimeDeleted); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetWatchedMedia retrieves media that has been watched
func (q *Queries) GetWatchedMedia(ctx context.Context, limit int64) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND COALESCE(time_last_played, 0) > 0 ORDER BY time_last_played DESC LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetUnwatchedMedia retrieves media that has not been watched
func (q *Queries) GetUnwatchedMedia(ctx context.Context, limit int64) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND COALESCE(time_last_played, 0) = 0 ORDER BY path LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetUnfinishedMedia retrieves media that was started but not finished
func (q *Queries) GetUnfinishedMedia(ctx context.Context, limit int64) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND playhead > 0 AND playhead < duration * 0.95 ORDER BY time_last_played DESC LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetMediaByPlayCountParams are parameters for GetMediaByPlayCount
type GetMediaByPlayCountParams struct {
	MinPlayCount int64
	MaxPlayCount int64
	Limit        int64
}

// GetMediaByPlayCount retrieves media filtered by play count range
func (q *Queries) GetMediaByPlayCount(ctx context.Context, arg GetMediaByPlayCountParams) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND play_count >= ? AND play_count <= ? ORDER BY play_count DESC LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, arg.MinPlayCount, arg.MaxPlayCount, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetRandomMedia retrieves random media items
func (q *Queries) GetRandomMedia(ctx context.Context, limit int64) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 ORDER BY RANDOM() LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetSiblingMediaParams are parameters for GetSiblingMedia
type GetSiblingMediaParams struct {
	PathPattern string
	PathExclude string
	Limit       int64
}

// GetSiblingMedia retrieves media with similar paths (siblings)
func (q *Queries) GetSiblingMedia(ctx context.Context, arg GetSiblingMediaParams) ([]Media, error) {
	const query = `SELECT ` + mediaColumns + ` FROM media WHERE time_deleted = 0 AND path LIKE ? AND path != ? ORDER BY path LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, arg.PathPattern, arg.PathExclude, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var i Media
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// UpdatePlayHistoryParams are parameters for UpdatePlayHistory
type UpdatePlayHistoryParams struct {
	TimeLastPlayed  sql.NullInt64
	TimeFirstPlayed sql.NullInt64
	Playhead        sql.NullInt64
	Path            string
}

// UpdatePlayHistory updates play history for a media item
func (q *Queries) UpdatePlayHistory(ctx context.Context, arg UpdatePlayHistoryParams) error {
	const query = `UPDATE media SET time_last_played = ?, time_first_played = COALESCE(time_first_played, ?), play_count = COALESCE(play_count, 0) + 1, playhead = ? WHERE path = ?`
	_, err := q.db.ExecContext(ctx, query, arg.TimeLastPlayed, arg.TimeFirstPlayed, arg.Playhead, arg.Path)
	return err
}

// MarkDeletedParams are parameters for MarkDeleted
type MarkDeletedParams struct {
	TimeDeleted sql.NullInt64
	Path        string
}

// MarkDeleted marks a media item as deleted
func (q *Queries) MarkDeleted(ctx context.Context, arg MarkDeletedParams) error {
	const query = `UPDATE media SET time_deleted = ? WHERE path = ?`
	_, err := q.db.ExecContext(ctx, query, arg.TimeDeleted, arg.Path)
	return err
}

// UpdatePathParams are parameters for UpdatePath
type UpdatePathParams struct {
	NewPath string
	OldPath string
}

// UpdatePath updates the path of a media item
func (q *Queries) UpdatePath(ctx context.Context, arg UpdatePathParams) error {
	const query = `UPDATE media SET path = ? WHERE path = ?`
	_, err := q.db.ExecContext(ctx, query, arg.NewPath, arg.OldPath)
	return err
}

// UpdateMediaCategoriesParams are parameters for UpdateMediaCategories
type UpdateMediaCategoriesParams struct {
	Categories sql.NullString
	Path       string
}

// UpdateMediaCategories updates categories for a media item
func (q *Queries) UpdateMediaCategories(ctx context.Context, arg UpdateMediaCategoriesParams) error {
	const query = `UPDATE media SET categories = ? WHERE path = ?`
	_, err := q.db.ExecContext(ctx, query, arg.Categories, arg.Path)
	return err
}

// GetCategoryStatsRow is a row from GetCategoryStats
type GetCategoryStatsRow struct {
	Category string
	Count    int64
}

// GetCategoryStats retrieves stats for each category
func (q *Queries) GetCategoryStats(ctx context.Context) ([]GetCategoryStatsRow, error) {
	const query = `SELECT 'sports' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;sports;%' UNION ALL SELECT 'fitness' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;fitness;%' UNION ALL SELECT 'documentary' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;documentary;%' UNION ALL SELECT 'comedy' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;comedy;%' UNION ALL SELECT 'music' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;music;%' UNION ALL SELECT 'educational' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;educational;%' UNION ALL SELECT 'news' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;news;%' UNION ALL SELECT 'gaming' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;gaming;%' UNION ALL SELECT 'tech' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;tech;%' UNION ALL SELECT 'audiobook' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;audiobook;%' UNION ALL SELECT 'Uncategorized' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND (categories IS NULL OR categories = '') ORDER BY count DESC`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetCategoryStatsRow
	for rows.Next() {
		var i GetCategoryStatsRow
		if err := rows.Scan(&i.Category, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetUsedCategoriesRow is a row from GetUsedCategories
type GetUsedCategoriesRow struct {
	Categories sql.NullString
	Count      int64
}

// GetUsedCategories retrieves categories that are in use
func (q *Queries) GetUsedCategories(ctx context.Context) ([]GetUsedCategoriesRow, error) {
	const query = `SELECT categories, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories IS NOT NULL AND categories != '' GROUP BY categories`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetUsedCategoriesRow
	for rows.Next() {
		var i GetUsedCategoriesRow
		if err := rows.Scan(&i.Categories, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetCustomCategories retrieves custom keyword categories
func (q *Queries) GetCustomCategories(ctx context.Context) ([]string, error) {
	const query = `SELECT DISTINCT category FROM custom_keywords`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		items = append(items, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetRatingStatsRow is a row from GetRatingStats
type GetRatingStatsRow struct {
	Rating int64
	Count  int64
}

// GetRatingStats retrieves stats for each rating
func (q *Queries) GetRatingStats(ctx context.Context) ([]GetRatingStatsRow, error) {
	const query = `SELECT CAST(COALESCE(score, 0) AS INTEGER) as rating, COUNT(*) as count FROM media WHERE time_deleted = 0 GROUP BY rating ORDER BY rating DESC`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetRatingStatsRow
	for rows.Next() {
		var i GetRatingStatsRow
		if err := rows.Scan(&i.Rating, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetGenreStatsRow is a row from GetGenreStats
type GetGenreStatsRow struct {
	Genre sql.NullString
	Count int64
}

// GetGenreStats retrieves stats for each genre
func (q *Queries) GetGenreStats(ctx context.Context) ([]GetGenreStatsRow, error) {
	const query = `SELECT genre, COUNT(*) as count FROM media WHERE time_deleted = 0 AND genre IS NOT NULL AND genre != '' GROUP BY genre ORDER BY count DESC`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetGenreStatsRow
	for rows.Next() {
		var i GetGenreStatsRow
		if err := rows.Scan(&i.Genre, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetLanguageStatsRow is a row from GetLanguageStats
type GetLanguageStatsRow struct {
	Language sql.NullString
	Count    int64
}

// GetLanguageStats retrieves stats for each language
func (q *Queries) GetLanguageStats(ctx context.Context) ([]GetLanguageStatsRow, error) {
	const query = `SELECT language, COUNT(*) as count FROM media WHERE time_deleted = 0 AND language IS NOT NULL AND language != '' GROUP BY language ORDER BY count DESC`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetLanguageStatsRow
	for rows.Next() {
		var i GetLanguageStatsRow
		if err := rows.Scan(&i.Language, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// UpsertMediaParams are parameters for UpsertMedia
type UpsertMediaParams struct {
	Path           string
	PathTokenized  sql.NullString
	Title          sql.NullString
	Duration       sql.NullInt64
	Size           sql.NullInt64
	TimeCreated    sql.NullInt64
	TimeModified   sql.NullInt64
	MediaType      sql.NullString
	Width          sql.NullInt64
	Height         sql.NullInt64
	Fps            sql.NullFloat64
	VideoCodecs    sql.NullString
	AudioCodecs    sql.NullString
	SubtitleCodecs sql.NullString
	VideoCount     sql.NullInt64
	AudioCount     sql.NullInt64
	SubtitleCount  sql.NullInt64
	Album          sql.NullString
	Artist         sql.NullString
	Genre          sql.NullString
	Categories     sql.NullString
	Description    sql.NullString
	Language       sql.NullString
	TimeDownloaded sql.NullInt64
	Score          sql.NullFloat64
	Fasthash       sql.NullString
	Sha256         sql.NullString
	IsDeduped      sql.NullInt64
}

// UpsertMedia inserts or updates a media item
func (q *Queries) UpsertMedia(ctx context.Context, arg UpsertMediaParams) error {
	const query = `INSERT INTO media (path, path_tokenized, title, duration, size, time_created, time_modified, media_type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs, video_count, audio_count, subtitle_count, album, artist, genre, categories, description, language, time_downloaded, score, fasthash, sha256, is_deduped) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(path) DO UPDATE SET path_tokenized = excluded.path_tokenized, title = excluded.title, duration = excluded.duration, size = excluded.size, time_modified = excluded.time_modified, media_type = excluded.media_type, width = excluded.width, height = excluded.height, fps = excluded.fps, video_codecs = excluded.video_codecs, audio_codecs = excluded.audio_codecs, subtitle_codecs = excluded.subtitle_codecs, video_count = excluded.video_count, audio_count = excluded.audio_count, subtitle_count = excluded.subtitle_count, album = excluded.album, artist = excluded.artist, genre = excluded.genre, categories = excluded.categories, description = excluded.description, language = excluded.language, time_downloaded = COALESCE(media.time_downloaded, excluded.time_downloaded), score = excluded.score, fasthash = COALESCE(NULLIF(excluded.fasthash, ''), media.fasthash), sha256 = COALESCE(NULLIF(excluded.sha256, ''), media.sha256), is_deduped = COALESCE(NULLIF(excluded.is_deduped, 0), media.is_deduped)`
	_, err := q.db.ExecContext(ctx, query,
		arg.Path,
		arg.PathTokenized,
		arg.Title,
		arg.Duration,
		arg.Size,
		arg.TimeCreated,
		arg.TimeModified,
		arg.MediaType,
		arg.Width,
		arg.Height,
		arg.Fps,
		arg.VideoCodecs,
		arg.AudioCodecs,
		arg.SubtitleCodecs,
		arg.VideoCount,
		arg.AudioCount,
		arg.SubtitleCount,
		arg.Album,
		arg.Artist,
		arg.Genre,
		arg.Categories,
		arg.Description,
		arg.Language,
		arg.TimeDownloaded,
		arg.Score,
		arg.Fasthash,
		arg.Sha256,
		arg.IsDeduped,
	)
	return err
}

// InsertPlaylistParams are parameters for InsertPlaylist
type InsertPlaylistParams struct {
	Path            sql.NullString
	Title           sql.NullString
	ExtractorKey    sql.NullString
	ExtractorConfig sql.NullString
}

// InsertPlaylist inserts a new playlist or updates existing
func (q *Queries) InsertPlaylist(ctx context.Context, arg InsertPlaylistParams) (int64, error) {
	const query = `INSERT INTO playlists (path, title, extractor_key, extractor_config) VALUES (?, ?, ?, ?) ON CONFLICT(path) DO UPDATE SET title = COALESCE(excluded.title, playlists.title), extractor_key = excluded.extractor_key, extractor_config = excluded.extractor_config RETURNING id`
	var id int64
	err := q.db.QueryRowContext(ctx, query, arg.Path, arg.Title, arg.ExtractorKey, arg.ExtractorConfig).Scan(&id)
	return id, err
}

// DeletePlaylistParams are parameters for DeletePlaylist
type DeletePlaylistParams struct {
	TimeDeleted sql.NullInt64
	ID          int64
}

// DeletePlaylist marks a playlist as deleted
func (q *Queries) DeletePlaylist(ctx context.Context, arg DeletePlaylistParams) error {
	const query = `UPDATE playlists SET time_deleted = ? WHERE id = ?`
	_, err := q.db.ExecContext(ctx, query, arg.TimeDeleted, arg.ID)
	return err
}

// GetPlaylists retrieves all non-deleted playlists
func (q *Queries) GetPlaylists(ctx context.Context) ([]Playlists, error) {
	const query = `SELECT id, path, title, extractor_key, extractor_config, time_deleted FROM playlists WHERE time_deleted = 0 ORDER BY title, path`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Playlists
	for rows.Next() {
		var i Playlists
		if err := rows.Scan(
			&i.ID,
			&i.Path,
			&i.Title,
			&i.ExtractorKey,
			&i.ExtractorConfig,
			&i.TimeDeleted,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// AddPlaylistItemParams are parameters for AddPlaylistItem
type AddPlaylistItemParams struct {
	PlaylistID  int64
	MediaPath   string
	TrackNumber sql.NullInt64
}

// AddPlaylistItem adds a media item to a playlist
func (q *Queries) AddPlaylistItem(ctx context.Context, arg AddPlaylistItemParams) error {
	const query = `INSERT INTO playlist_items (playlist_id, media_path, track_number, time_added) VALUES (?, ?, ?, unixepoch()) ON CONFLICT(playlist_id, media_path) DO UPDATE SET track_number = excluded.track_number`
	_, err := q.db.ExecContext(ctx, query, arg.PlaylistID, arg.MediaPath, arg.TrackNumber)
	return err
}

// RemovePlaylistItemParams are parameters for RemovePlaylistItem
type RemovePlaylistItemParams struct {
	PlaylistID int64
	MediaPath  string
}

// RemovePlaylistItem removes a media item from a playlist
func (q *Queries) RemovePlaylistItem(ctx context.Context, arg RemovePlaylistItemParams) error {
	const query = `DELETE FROM playlist_items WHERE playlist_id = ? AND media_path = ?`
	_, err := q.db.ExecContext(ctx, query, arg.PlaylistID, arg.MediaPath)
	return err
}

// GetPlaylistItemsRow is a row from GetPlaylistItems
type GetPlaylistItemsRow struct {
	Media
	TrackNumber sql.NullInt64 `json:"track_number"`
	TimeAdded   sql.NullInt64 `json:"time_added"`
}

// GetPlaylistItems retrieves all items in a playlist
func (q *Queries) GetPlaylistItems(ctx context.Context, playlistID int64) ([]GetPlaylistItemsRow, error) {
	const query = `SELECT m.path, m.path_tokenized, m.title, m.duration, m.size, m.time_created, m.time_modified, m.time_deleted, m.time_first_played, m.time_last_played, m.play_count, m.playhead, m.media_type, m.width, m.height, m.fps, m.video_codecs, m.audio_codecs, m.subtitle_codecs, m.video_count, m.audio_count, m.subtitle_count, m.album, m.artist, m.genre, m.categories, m.description, m.language, m.time_downloaded, m.score, pi.track_number, pi.time_added FROM media m JOIN playlist_items pi ON m.path = pi.media_path WHERE pi.playlist_id = ? AND m.time_deleted = 0 ORDER BY pi.track_number, pi.time_added, m.path`
	rows, err := q.db.QueryContext(ctx, query, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetPlaylistItemsRow
	for rows.Next() {
		var i GetPlaylistItemsRow
		if err := rows.Scan(
			&i.Path,
			&i.PathTokenized,
			&i.Title,
			&i.Duration,
			&i.Size,
			&i.TimeCreated,
			&i.TimeModified,
			&i.TimeDeleted,
			&i.TimeFirstPlayed,
			&i.TimeLastPlayed,
			&i.PlayCount,
			&i.Playhead,
			&i.MediaType,
			&i.Width,
			&i.Height,
			&i.Fps,
			&i.VideoCodecs,
			&i.AudioCodecs,
			&i.SubtitleCodecs,
			&i.VideoCount,
			&i.AudioCount,
			&i.SubtitleCount,
			&i.Album,
			&i.Artist,
			&i.Genre,
			&i.Categories,
			&i.Description,
			&i.Language,
			&i.TimeDownloaded,
			&i.Score,
			&i.TrackNumber,
			&i.TimeAdded,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ClearPlaylist removes all items from a playlist
func (q *Queries) ClearPlaylist(ctx context.Context, playlistID int64) error {
	const query = `DELETE FROM playlist_items WHERE playlist_id = ?`
	_, err := q.db.ExecContext(ctx, query, playlistID)
	return err
}

// InsertCaptionParams are parameters for InsertCaption
type InsertCaptionParams struct {
	MediaPath string
	Time      sql.NullFloat64
	Text      sql.NullString
}

// InsertCaption inserts a caption for a media item
func (q *Queries) InsertCaption(ctx context.Context, arg InsertCaptionParams) error {
	const query = `INSERT INTO captions (media_path, time, text) VALUES (?, ?, ?)`
	_, err := q.db.ExecContext(ctx, query, arg.MediaPath, arg.Time, arg.Text)
	return err
}

// InsertHistoryParams are parameters for InsertHistory
type InsertHistoryParams struct {
	MediaPath  string
	TimePlayed sql.NullInt64
	Playhead   sql.NullInt64
	Done       sql.NullInt64
}

// InsertHistory inserts a history entry
func (q *Queries) InsertHistory(ctx context.Context, arg InsertHistoryParams) error {
	const query = `INSERT INTO history (media_path, time_played, playhead, done) VALUES (?, ?, ?, ?)`
	_, err := q.db.ExecContext(ctx, query, arg.MediaPath, arg.TimePlayed, arg.Playhead, arg.Done)
	return err
}

// GetHistoryCount retrieves the history count for a media item
func (q *Queries) GetHistoryCount(ctx context.Context, mediaPath string) (int64, error) {
	const query = `SELECT COUNT(*) FROM history WHERE media_path = ?`
	var count int64
	err := q.db.QueryRowContext(ctx, query, mediaPath).Scan(&count)
	return count, err
}

// GetCaptionsForMedia retrieves all captions for a media item
func (q *Queries) GetCaptionsForMedia(ctx context.Context, mediaPath string) ([]Captions, error) {
	const query = `SELECT media_path, time, text FROM captions WHERE media_path = ? ORDER BY time`
	rows, err := q.db.QueryContext(ctx, query, mediaPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Captions
	for rows.Next() {
		var i Captions
		if err := rows.Scan(&i.MediaPath, &i.Time, &i.Text); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetAllCaptions retrieves random captions
func (q *Queries) GetAllCaptions(ctx context.Context, limit int64) ([]GetAllCaptionsRow, error) {
	const query = `SELECT c.media_path, c.time, c.text, m.title, m.media_type, m.size, m.duration FROM captions c JOIN media m ON c.media_path = m.path WHERE m.time_deleted = 0 AND c.text IS NOT NULL AND c.text != '' ORDER BY RANDOM() LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetAllCaptionsRow
	for rows.Next() {
		var i GetAllCaptionsRow
		if err := rows.Scan(&i.MediaPath, &i.Time, &i.Text, &i.Title, &i.MediaType, &i.Size, &i.Duration); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetAllCaptionsOrderedParams are parameters for GetAllCaptionsOrdered
type GetAllCaptionsOrderedParams struct {
	VideoOnly int64
	AudioOnly int64
	ImageOnly int64
	TextOnly  int64
	Limit     int64
}

// GetAllCaptionsOrdered retrieves captions ordered by media path and time
func (q *Queries) GetAllCaptionsOrdered(ctx context.Context, arg GetAllCaptionsOrderedParams) ([]GetAllCaptionsOrderedRow, error) {
	const query = `SELECT c.media_path, c.time, c.text, m.title, m.media_type, m.size, m.duration FROM captions c JOIN media m ON c.media_path = m.path WHERE m.time_deleted = 0 AND c.text IS NOT NULL AND c.text != '' AND ((CAST(? AS INT) = 0 AND CAST(? AS INT) = 0 AND CAST(? AS INT) = 0 AND CAST(? AS INT) = 0) OR (CAST(? AS INT) = 1 AND m.media_type = 'video') OR (CAST(? AS INT) = 1 AND m.media_type IN ('audio', 'audiobook')) OR (CAST(? AS INT) = 1 AND m.media_type = 'image') OR (CAST(? AS INT) = 1 AND m.media_type = 'text')) ORDER BY c.media_path, c.time LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query,
		arg.VideoOnly,
		arg.AudioOnly,
		arg.ImageOnly,
		arg.TextOnly,
		arg.VideoOnly,
		arg.AudioOnly,
		arg.ImageOnly,
		arg.TextOnly,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetAllCaptionsOrderedRow
	for rows.Next() {
		var i GetAllCaptionsOrderedRow
		if err := rows.Scan(&i.MediaPath, &i.Time, &i.Text, &i.Title, &i.MediaType, &i.Size, &i.Duration); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// PopulateMediaType populates the media_type column for media items with NULL media_type
func (q *Queries) PopulateMediaType(ctx context.Context) error {
	db, ok := q.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("underlying DBTX is not a *sql.DB")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get all media with NULL media_type
	rows, err := tx.QueryContext(ctx, "SELECT path FROM media WHERE media_type IS NULL")
	if err != nil {
		return err
	}
	defer rows.Close()

	var updates []struct {
		path      string
		mediaType string
	}

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		mediaType := ""

		// Simplified type detection for migration
		// In a real scenario, this would use the same logic as metadata.Extract
		if isVideo(ext) {
			mediaType = "video"
		} else if isAudio(ext) {
			mediaType = "audio"
		} else if isImage(ext) {
			mediaType = "image"
		} else if isText(ext) {
			mediaType = "text"
		}

		if mediaType != "" {
			updates = append(updates, struct {
				path      string
				mediaType string
			}{path, mediaType})
		}
	}
	rows.Close()

	if len(updates) > 0 {
		stmt, err := tx.PrepareContext(ctx, "UPDATE media SET media_type = ? WHERE path = ?")
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, u := range updates {
			if _, err := stmt.ExecContext(ctx, u.mediaType, u.path); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// Helper functions for PopulateMediaType (simplified)
func isVideo(ext string) bool {
	return ext == ".mp4" || ext == ".mkv" || ext == ".avi" || ext == ".mov" || ext == ".webm" || ext == ".m4v"
}

func isAudio(ext string) bool {
	return ext == ".mp3" || ext == ".flac" || ext == ".wav" || ext == ".m4a" || ext == ".ogg" || ext == ".opus"
}

func isImage(ext string) bool {
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".bmp"
}

func isText(ext string) bool {
	return ext == ".pdf" || ext == ".epub" || ext == ".txt" || ext == ".md" || ext == ".cbz" || ext == ".cbr"
}

// GetStatsRow is a row from GetStats
type GetStatsRow struct {
	TotalCount           int64
	TotalSize            sql.NullInt64
	TotalDuration        sql.NullInt64
	WatchedCount         int64
	UnwatchedCount       int64
	TotalWatchedDuration sql.NullInt64
}

// GetStats retrieves overall stats
func (q *Queries) GetStats(ctx context.Context) (GetStatsRow, error) {
	const query = `SELECT COUNT(*) as total_count, SUM(size) as total_size, SUM(duration) as total_duration, COUNT(CASE WHEN COALESCE(time_last_played, 0) > 0 THEN 1 END) as watched_count, COUNT(CASE WHEN COALESCE(time_last_played, 0) = 0 THEN 1 END) as unwatched_count, SUM(COALESCE(play_count, 0) * COALESCE(duration, 0) + COALESCE(playhead, 0)) as total_watched_duration FROM media WHERE time_deleted = 0`
	var i GetStatsRow
	err := q.db.QueryRowContext(ctx, query).Scan(
		&i.TotalCount,
		&i.TotalSize,
		&i.TotalDuration,
		&i.WatchedCount,
		&i.UnwatchedCount,
		&i.TotalWatchedDuration,
	)
	return i, err
}

// GetStatsByTypeRow is a row from GetStatsByType
type GetStatsByTypeRow struct {
	MediaType     sql.NullString
	Count         int64
	TotalSize     sql.NullInt64
	TotalDuration sql.NullInt64
}

// GetStatsByType retrieves stats grouped by type
func (q *Queries) GetStatsByType(ctx context.Context) ([]GetStatsByTypeRow, error) {
	const query = `SELECT media_type, COUNT(*) as count, SUM(size) as total_size, SUM(duration) as total_duration FROM media WHERE time_deleted = 0 GROUP BY media_type`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetStatsByTypeRow
	for rows.Next() {
		var i GetStatsByTypeRow
		if err := rows.Scan(&i.MediaType, &i.Count, &i.TotalSize, &i.TotalDuration); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
