-- SQLite schema for media library

CREATE TABLE IF NOT EXISTS playlists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE,
    title TEXT,
    extractor_key TEXT,
    extractor_config TEXT,
    time_deleted INTEGER DEFAULT 0
) STRICT;

CREATE TABLE IF NOT EXISTS playlist_items (
    playlist_id INTEGER NOT NULL,
    media_path TEXT NOT NULL,
    track_number INTEGER,
    time_added INTEGER DEFAULT (unixepoch()),
    PRIMARY KEY (playlist_id, media_path),
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
) STRICT;

CREATE TABLE IF NOT EXISTS media (
    path TEXT PRIMARY KEY,
    path_tokenized TEXT, -- Processed path for FTS (dots replaced by spaces etc)
    title TEXT,
    duration INTEGER,
    size INTEGER,
    time_created INTEGER,
    time_modified INTEGER,
    time_deleted INTEGER DEFAULT 0,
    time_first_played INTEGER DEFAULT 0,
    time_last_played INTEGER DEFAULT 0,
    play_count INTEGER DEFAULT 0,
    playhead INTEGER DEFAULT 0,

    -- Media type info
    type TEXT,
    width INTEGER,
    height INTEGER,
    fps REAL,

    -- Codec info
    video_codecs TEXT,
    audio_codecs TEXT,
    subtitle_codecs TEXT,
    video_count INTEGER DEFAULT 0,
    audio_count INTEGER DEFAULT 0,
    subtitle_count INTEGER DEFAULT 0,

    -- Extra metadata
    album TEXT,
    artist TEXT,
    genre TEXT,
    categories TEXT,
    description TEXT,
    language TEXT,

    -- Metadata
    time_downloaded INTEGER, -- Repurposed as Time First Scanned
    score REAL
) STRICT;

CREATE TABLE IF NOT EXISTS captions (
    media_path TEXT NOT NULL,
    time REAL,
    text TEXT,
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_captions_path ON captions(media_path);

-- FTS for captions
CREATE VIRTUAL TABLE IF NOT EXISTS captions_fts USING fts5(
    media_path UNINDEXED,
    text,
    content='captions',
    tokenize = 'trigram',
    detail = 'none'
);

-- Triggers for captions FTS
CREATE TRIGGER IF NOT EXISTS captions_ai AFTER INSERT ON captions BEGIN
    INSERT INTO captions_fts(rowid, media_path, text)
    VALUES (new.rowid, new.media_path, new.text);
END;

CREATE TRIGGER IF NOT EXISTS captions_ad AFTER DELETE ON captions BEGIN
    DELETE FROM captions_fts WHERE rowid = old.rowid;
END;

CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_path TEXT NOT NULL,
    time_played INTEGER DEFAULT (unixepoch()),
    playhead INTEGER,
    done INTEGER,
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_history_path ON history(media_path);
CREATE INDEX IF NOT EXISTS idx_history_time ON history(time_played);

CREATE TABLE IF NOT EXISTS custom_keywords (
    category TEXT NOT NULL,
    keyword TEXT NOT NULL,
    PRIMARY KEY (category, keyword)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_time_deleted ON media(time_deleted);
CREATE INDEX IF NOT EXISTS idx_time_last_played ON media(time_last_played);
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

-- Optional FTS table
CREATE VIRTUAL TABLE IF NOT EXISTS media_fts USING fts5(
    path,
    path_tokenized,
    title,
    description,
    content='media',
    content_rowid='rowid',
    tokenize = 'trigram',
    detail = 'none'
);

-- Trigger to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS media_ai AFTER INSERT ON media BEGIN
    INSERT INTO media_fts(rowid, path, path_tokenized, title, description)
    VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description);
END;

CREATE TRIGGER IF NOT EXISTS media_ad AFTER DELETE ON media BEGIN
    DELETE FROM media_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS media_au AFTER UPDATE ON media BEGIN
    INSERT INTO media_fts(media_fts, rowid, path, path_tokenized, title, description) VALUES('delete', old.rowid, old.path, old.path_tokenized, old.title, old.description);
    INSERT INTO media_fts(rowid, path, path_tokenized, title, description) VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description);
END;
