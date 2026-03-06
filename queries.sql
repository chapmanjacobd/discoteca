-- name: GetMedia :many
SELECT * FROM media
WHERE time_deleted = 0
ORDER BY path
LIMIT ?;

-- name: GetMediaByType :many
SELECT * FROM media
WHERE time_deleted = 0
  AND (
    (? AND type = 'video')
    OR (? AND (type = 'audio' OR type = 'audiobook'))
    OR (? AND type = 'image')
  )
ORDER BY path
LIMIT ?;
-- name: GetMediaBySize :many
SELECT * FROM media
WHERE time_deleted = 0
  AND size >= ?
  AND size <= ?
ORDER BY size DESC
LIMIT ?;

-- name: GetMediaByDuration :many
SELECT * FROM media
WHERE time_deleted = 0
  AND duration >= ?
  AND duration <= ?
ORDER BY duration DESC
LIMIT ?;

-- name: GetMediaByPath :many
SELECT * FROM media
WHERE time_deleted = 0
  AND path LIKE ?
ORDER BY path
LIMIT ?;

-- name: GetMediaByPathPrefix :many
SELECT * FROM media
WHERE time_deleted = 0
  AND path LIKE ?
  AND path NOT LIKE ?
ORDER BY path
LIMIT ?;

-- name: GetMediaByPathExact :one
SELECT * FROM media
WHERE path = ?
LIMIT 1;

-- name: GetAllMediaMetadata :many
SELECT path, size, time_modified, time_deleted FROM media;

-- name: GetWatchedMedia :many
SELECT * FROM media
WHERE time_deleted = 0
  AND COALESCE(time_last_played, 0) > 0
ORDER BY time_last_played DESC
LIMIT ?;

-- name: GetUnwatchedMedia :many
SELECT * FROM media
WHERE time_deleted = 0
  AND COALESCE(time_last_played, 0) = 0
ORDER BY path
LIMIT ?;

-- name: GetUnfinishedMedia :many
SELECT * FROM media
WHERE time_deleted = 0
  AND playhead > 0
  AND playhead < duration * 0.95
ORDER BY time_last_played DESC
LIMIT ?;

-- name: GetMediaByPlayCount :many
SELECT * FROM media
WHERE time_deleted = 0
  AND play_count >= ?
  AND play_count <= ?
ORDER BY play_count DESC
LIMIT ?;

-- name: GetRandomMedia :many
SELECT * FROM media
WHERE time_deleted = 0
ORDER BY RANDOM()
LIMIT ?;

-- name: GetSiblingMedia :many
SELECT * FROM media
WHERE time_deleted = 0
  AND path LIKE ?
  AND path != ?
ORDER BY path
LIMIT ?;

-- name: SearchMediaFTS :many
SELECT * FROM media
WHERE rowid IN (
    SELECT rowid FROM media_fts f WHERE f.title MATCH sqlc.arg('query')
)
AND time_deleted = 0
LIMIT sqlc.arg('limit');

-- name: UpdatePlayHistory :exec
UPDATE media
SET time_last_played = ?,
    time_first_played = COALESCE(time_first_played, ?),
    play_count = COALESCE(play_count, 0) + 1,
    playhead = ?
WHERE path = ?;

-- name: MarkDeleted :exec
UPDATE media
SET time_deleted = ?
WHERE path = ?;

-- name: UpdatePath :exec
UPDATE media
SET path = ?
WHERE path = ?;

-- name: UpdateMediaCategories :exec
UPDATE media
SET categories = ?
WHERE path = ?;

-- name: GetCategoryStats :many
SELECT 'sports' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;sports;%'
UNION ALL
SELECT 'fitness' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;fitness;%'
UNION ALL
SELECT 'documentary' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;documentary;%'
UNION ALL
SELECT 'comedy' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;comedy;%'
UNION ALL
SELECT 'music' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;music;%'
UNION ALL
SELECT 'educational' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;educational;%'
UNION ALL
SELECT 'news' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;news;%'
UNION ALL
SELECT 'gaming' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;gaming;%'
UNION ALL
SELECT 'tech' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;tech;%'
UNION ALL
SELECT 'audiobook' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND categories LIKE '%;audiobook;%'
UNION ALL
SELECT 'Uncategorized' as category, COUNT(*) as count FROM media WHERE time_deleted = 0 AND (categories IS NULL OR categories = '')
ORDER BY count DESC;

-- name: GetUsedCategories :many
SELECT categories, COUNT(*) as count
FROM media
WHERE time_deleted = 0 AND categories IS NOT NULL AND categories != ''
GROUP BY categories;

-- name: GetCustomCategories :many
SELECT DISTINCT category FROM custom_keywords;

-- name: GetRatingStats :many
SELECT CAST(COALESCE(score, 0) AS INTEGER) as rating, COUNT(*) as count
FROM media
WHERE time_deleted = 0
GROUP BY rating
ORDER BY rating DESC;

-- name: GetGenreStats :many
SELECT genre, COUNT(*) as count
FROM media
WHERE time_deleted = 0 AND genre IS NOT NULL AND genre != ''
GROUP BY genre
ORDER BY count DESC;

-- name: UpsertMedia :exec
INSERT INTO media (
    path, title, duration, size, time_created, time_modified,
    type, width, height, fps,
    video_codecs, audio_codecs, subtitle_codecs,
    video_count, audio_count, subtitle_count,
    album, artist, genre, 
    mood, bpm, key, decade, categories, city, country,
    description, language,
    webpath, uploader, time_uploaded, time_downloaded,
    view_count, num_comments, favorite_count, score, upvote_ratio,
    latitude, longitude
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?,
    ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?
)
ON CONFLICT(path) DO UPDATE SET
    title = excluded.title,
    duration = excluded.duration,
    size = excluded.size,
    time_modified = excluded.time_modified,
    type = excluded.type,
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
    mood = excluded.mood,
    bpm = excluded.bpm,
    key = excluded.key,
    decade = excluded.decade,
    categories = excluded.categories,
    city = excluded.city,
    country = excluded.country,
    description = excluded.description,
    language = excluded.language,
    webpath = excluded.webpath,
    uploader = excluded.uploader,
    time_uploaded = excluded.time_uploaded,
    time_downloaded = excluded.time_downloaded,
    view_count = excluded.view_count,
    num_comments = excluded.num_comments,
    favorite_count = excluded.favorite_count,
    score = excluded.score,
    upvote_ratio = excluded.upvote_ratio,
    latitude = excluded.latitude,
    longitude = excluded.longitude;

-- name: InsertPlaylist :one
INSERT INTO playlists (path, title, extractor_key, extractor_config)
VALUES (?, ?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
    title = COALESCE(excluded.title, playlists.title),
    extractor_key = excluded.extractor_key,
    extractor_config = excluded.extractor_config
RETURNING id;

-- name: DeletePlaylist :exec
UPDATE playlists SET time_deleted = ? WHERE id = ?;

-- name: GetPlaylists :many
SELECT * FROM playlists WHERE time_deleted = 0 ORDER BY title, path;

-- name: AddPlaylistItem :exec
INSERT INTO playlist_items (playlist_id, media_path, track_number, time_added)
VALUES (?, ?, ?, strftime('%s', 'now'))
ON CONFLICT(playlist_id, media_path) DO UPDATE SET
    track_number = excluded.track_number;

-- name: RemovePlaylistItem :exec
DELETE FROM playlist_items WHERE playlist_id = ? AND media_path = ?;

-- name: GetPlaylistItems :many
SELECT m.*, pi.track_number, pi.time_added FROM media m
JOIN playlist_items pi ON m.path = pi.media_path
WHERE pi.playlist_id = ? AND m.time_deleted = 0
ORDER BY pi.track_number, pi.time_added, m.path;

-- name: ClearPlaylist :exec
DELETE FROM playlist_items WHERE playlist_id = ?;

-- name: InsertCaption :exec
INSERT INTO captions (media_path, time, text)
VALUES (?, ?, ?);

-- name: InsertHistory :exec
INSERT INTO history (media_path, time_played, playhead, done)
VALUES (?, ?, ?, ?);

-- name: GetHistoryCount :one
SELECT COUNT(*) FROM history WHERE media_path = ?;

-- name: GetCaptionsForMedia :many
SELECT * FROM captions
WHERE media_path = ?
ORDER BY time;

-- name: GetAllCaptions :many
SELECT c.media_path, c.time, c.text, m.title, m.type
FROM captions c
JOIN media m ON c.media_path = m.path
WHERE m.time_deleted = 0
  AND c.text IS NOT NULL AND c.text != ''
ORDER BY c.media_path, c.time
LIMIT ?;

-- name: SearchCaptions :many
SELECT c.media_path, c.time, c.text, m.title, m.type
FROM captions c
JOIN captions_fts f ON c.rowid = f.rowid
JOIN media m ON c.media_path = m.path
WHERE f.text MATCH sqlc.arg('query')
  AND m.time_deleted = 0
  AND c.text IS NOT NULL AND c.text != ''
ORDER BY c.media_path, c.time
LIMIT sqlc.arg('limit');

-- name: GetStats :one
SELECT
    COUNT(*) as total_count,
    SUM(size) as total_size,
    SUM(duration) as total_duration,
    COUNT(CASE WHEN COALESCE(time_last_played, 0) > 0 THEN 1 END) as watched_count,
    COUNT(CASE WHEN COALESCE(time_last_played, 0) = 0 THEN 1 END) as unwatched_count,
    SUM(COALESCE(play_count, 0) * COALESCE(duration, 0) + COALESCE(playhead, 0)) as total_watched_duration
FROM media
WHERE time_deleted = 0;

-- name: GetStatsByType :many
SELECT
    type,
    COUNT(*) as count,
    SUM(size) as total_size,
    SUM(duration) as total_duration
FROM media
WHERE time_deleted = 0
GROUP BY type;
