CREATE TABLE IF NOT EXISTS custom_keywords (
    category TEXT NOT NULL,
    keyword TEXT NOT NULL,
    PRIMARY KEY (category, keyword)
) STRICT;

-- Materialized view for folder statistics (optimizes /api/du endpoint)
-- This pre-aggregates folder-level stats to avoid expensive GROUP BY queries
CREATE TABLE IF NOT EXISTS folder_stats (
    parent TEXT PRIMARY KEY,
    file_count INTEGER,
    total_size INTEGER,
    total_duration INTEGER
);

CREATE TABLE IF NOT EXISTS folder_stats_meta (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    dirty INTEGER NOT NULL DEFAULT 1
);

INSERT OR IGNORE INTO folder_stats_meta (id, dirty) VALUES (1, 1);

-- Triggers mark folder_stats dirty after media changes. The dirty bit is
-- read by NeedsRefresh and cleared by RefreshFolderStats.
CREATE TRIGGER IF NOT EXISTS folder_stats_media_insert
AFTER INSERT ON media
BEGIN
    UPDATE folder_stats_meta SET dirty = 1 WHERE id = 1;
END;

CREATE TRIGGER IF NOT EXISTS folder_stats_media_delete
AFTER DELETE ON media
BEGIN
    UPDATE folder_stats_meta SET dirty = 1 WHERE id = 1;
END;

CREATE TRIGGER IF NOT EXISTS folder_stats_media_update
AFTER UPDATE OF path, size, duration, time_deleted ON media
BEGIN
    UPDATE folder_stats_meta SET dirty = 1 WHERE id = 1;
END;

-- Metadata table for tracking maintenance tasks
CREATE TABLE IF NOT EXISTS _maintenance_meta (
    key TEXT PRIMARY KEY,
    value TEXT,
    last_updated INTEGER
);

-- Initialize maintenance tracking keys
INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('folder_stats_last_refresh', '0', 0);
INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('fts_last_rebuild', '0', 0);
