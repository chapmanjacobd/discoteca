CREATE TABLE IF NOT EXISTS custom_keywords (
    category TEXT NOT NULL,
    keyword TEXT NOT NULL,
    PRIMARY KEY (category, keyword)
) STRICT;

-- Materialized view for folder statistics (optimizes /api/du endpoint)
-- This pre-aggregates folder-level stats to avoid expensive GROUP BY queries
CREATE TABLE IF NOT EXISTS folder_stats (
    parent TEXT PRIMARY KEY,
    depth INTEGER,
    file_count INTEGER,
    total_size INTEGER,
    total_duration INTEGER
);

-- Metadata table for tracking maintenance tasks
CREATE TABLE IF NOT EXISTS _maintenance_meta (
    key TEXT PRIMARY KEY,
    value TEXT,
    last_updated INTEGER
);

-- Initialize maintenance tracking keys
INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('folder_stats_last_refresh', '0', 0);
INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('fts_last_rebuild', '0', 0);
