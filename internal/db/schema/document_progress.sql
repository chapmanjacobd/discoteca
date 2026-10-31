CREATE TABLE IF NOT EXISTS document_progress (
    media_path TEXT PRIMARY KEY,
    scroll_percent REAL NOT NULL DEFAULT 0,
    chapter TEXT,
    time_updated INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
) STRICT;
