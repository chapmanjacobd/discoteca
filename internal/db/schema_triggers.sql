-- SQLite schema for media library - Core Triggers and Indexes

CREATE INDEX IF NOT EXISTS idx_captions_path ON captions(media_path);

CREATE INDEX IF NOT EXISTS idx_history_path ON history(media_path);
CREATE INDEX IF NOT EXISTS idx_history_time ON history(time_played);

-- Core indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_time_deleted ON media(time_deleted);
CREATE INDEX IF NOT EXISTS idx_time_last_played ON media(time_last_played);
CREATE INDEX IF NOT EXISTS idx_path ON media(path);

-- Composite indexes for common filtered queries (time_deleted is frequently used)
CREATE INDEX IF NOT EXISTS idx_media_deleted_type ON media(time_deleted, type);
CREATE INDEX IF NOT EXISTS idx_media_deleted_size ON media(time_deleted, size);
CREATE INDEX IF NOT EXISTS idx_media_deleted_duration ON media(time_deleted, duration);
CREATE INDEX IF NOT EXISTS idx_media_deleted_path ON media(time_deleted, path);

-- Partial index for active media (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_media_active ON media(path, type) WHERE time_deleted = 0;

-- Individual column indexes for non-composite queries
CREATE INDEX IF NOT EXISTS idx_duration ON media(duration);
CREATE INDEX IF NOT EXISTS idx_size ON media(size);
CREATE INDEX IF NOT EXISTS idx_type ON media(type);
CREATE INDEX IF NOT EXISTS idx_genre ON media(genre);
CREATE INDEX IF NOT EXISTS idx_artist ON media(artist);
CREATE INDEX IF NOT EXISTS idx_album ON media(album);
CREATE INDEX IF NOT EXISTS idx_categories ON media(categories);
CREATE INDEX IF NOT EXISTS idx_score ON media(score);
CREATE INDEX IF NOT EXISTS idx_time_created ON media(time_created);
CREATE INDEX IF NOT EXISTS idx_time_modified ON media(time_modified);
CREATE INDEX IF NOT EXISTS idx_time_downloaded ON media(time_downloaded);

-- Indexes for filter bins calculation (optimize include_counts)
CREATE INDEX IF NOT EXISTS idx_media_active_size ON media(size) WHERE time_deleted = 0 AND size > 0;
CREATE INDEX IF NOT EXISTS idx_media_active_duration ON media(duration) WHERE time_deleted = 0 AND duration > 0;
CREATE INDEX IF NOT EXISTS idx_media_active_time_modified ON media(time_modified) WHERE time_deleted = 0 AND time_modified > 0;
CREATE INDEX IF NOT EXISTS idx_media_active_time_created ON media(time_created) WHERE time_deleted = 0 AND time_created > 0;
CREATE INDEX IF NOT EXISTS idx_media_active_time_downloaded ON media(time_downloaded) WHERE time_deleted = 0 AND time_downloaded > 0;

-- Indexes for hash and processing status filtering
CREATE INDEX IF NOT EXISTS idx_media_fasthash ON media(fasthash) WHERE fasthash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_media_sha256 ON media(sha256) WHERE sha256 IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_media_is_deduped ON media(is_deduped) WHERE is_deduped = 1;
CREATE INDEX IF NOT EXISTS idx_media_is_shrinked ON media(is_shrinked) WHERE is_shrinked = 1;
CREATE INDEX IF NOT EXISTS idx_media_unprocessed ON media(path) WHERE is_deduped = 0 OR is_deduped IS NULL;
CREATE INDEX IF NOT EXISTS idx_media_unshrinked ON media(path) WHERE is_shrinked = 0 OR is_shrinked IS NULL;

-- Index for fast folder_stats queries
CREATE INDEX IF NOT EXISTS idx_folder_stats_depth ON folder_stats(depth);

-- Initialize maintenance tracking keys
INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('folder_stats_last_refresh', '0', 0);
INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('fts_last_rebuild', '0', 0);
