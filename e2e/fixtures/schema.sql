CREATE TABLE playlists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE,
    title TEXT,
    extractor_key TEXT,
    extractor_config TEXT,
    time_deleted INTEGER DEFAULT 0
) STRICT;
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE playlist_items (
    playlist_id INTEGER NOT NULL,
    media_path TEXT NOT NULL,
    track_number INTEGER,
    time_added INTEGER DEFAULT (unixepoch()),
    PRIMARY KEY (playlist_id, media_path),
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
) STRICT;
CREATE TABLE media (
    path TEXT PRIMARY KEY,
    fts_path TEXT, -- Processed path for FTS (dots replaced by spaces etc)
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
CREATE TABLE captions (
    media_path TEXT NOT NULL,
    time REAL,
    text TEXT,
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
) STRICT;
CREATE INDEX idx_captions_path ON captions(media_path);
CREATE VIRTUAL TABLE captions_fts USING fts5(
    media_path UNINDEXED,
    text,
    content='captions',
    tokenize = 'trigram'
)
/* captions_fts(media_path,text) */;
CREATE TABLE IF NOT EXISTS 'captions_fts_data'(id INTEGER PRIMARY KEY, block BLOB);
CREATE TABLE IF NOT EXISTS 'captions_fts_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS 'captions_fts_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
CREATE TABLE IF NOT EXISTS 'captions_fts_config'(k PRIMARY KEY, v) WITHOUT ROWID;
CREATE TRIGGER captions_ai AFTER INSERT ON captions BEGIN
    INSERT INTO captions_fts(rowid, media_path, text)
    VALUES (new.rowid, new.media_path, new.text);
END;
CREATE TRIGGER captions_ad AFTER DELETE ON captions BEGIN
    DELETE FROM captions_fts WHERE rowid = old.rowid;
END;
CREATE TABLE history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_path TEXT NOT NULL,
    time_played INTEGER DEFAULT (unixepoch()),
    playhead INTEGER,
    done INTEGER,
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
) STRICT;
CREATE INDEX idx_history_path ON history(media_path);
CREATE INDEX idx_history_time ON history(time_played);
CREATE TABLE custom_keywords (
    category TEXT NOT NULL,
    keyword TEXT NOT NULL,
    PRIMARY KEY (category, keyword)
) STRICT;
CREATE INDEX idx_time_deleted ON media(time_deleted);
CREATE INDEX idx_time_last_played ON media(time_last_played);
CREATE INDEX idx_duration ON media(duration);
CREATE INDEX idx_size ON media(size);
CREATE INDEX idx_type ON media(type);
CREATE INDEX idx_genre ON media(genre);
CREATE INDEX idx_artist ON media(artist);
CREATE INDEX idx_album ON media(album);
CREATE INDEX idx_categories ON media(categories);
CREATE INDEX idx_score ON media(score);
CREATE INDEX idx_time_created ON media(time_created);
CREATE INDEX idx_time_modified ON media(time_modified);
CREATE INDEX idx_time_downloaded ON media(time_downloaded);
CREATE VIRTUAL TABLE media_fts USING fts5(
    path,
    fts_path,
    title,
    description,
    content='media',
    content_rowid='rowid',
    tokenize = 'trigram'
)
/* media_fts(path,fts_path,title,description) */;
CREATE TABLE IF NOT EXISTS 'media_fts_data'(id INTEGER PRIMARY KEY, block BLOB);
CREATE TABLE IF NOT EXISTS 'media_fts_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS 'media_fts_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
CREATE TABLE IF NOT EXISTS 'media_fts_config'(k PRIMARY KEY, v) WITHOUT ROWID;
CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN
    INSERT INTO media_fts(rowid, path, fts_path, title, description)
    VALUES (new.rowid, new.path, new.fts_path, new.title, new.description);
END;
CREATE TRIGGER media_ad AFTER DELETE ON media BEGIN
    DELETE FROM media_fts WHERE rowid = old.rowid;
END;
CREATE TRIGGER media_au AFTER UPDATE ON media BEGIN
    INSERT INTO media_fts(media_fts, rowid, path, fts_path, title, description) VALUES('delete', old.rowid, old.path, old.fts_path, old.title, old.description);
    INSERT INTO media_fts(rowid, path, fts_path, title, description) VALUES (new.rowid, new.path, new.fts_path, new.title, new.description);
END;
