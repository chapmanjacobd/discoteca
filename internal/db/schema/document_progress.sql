CREATE TABLE IF NOT EXISTS document_progress (
    media_id INTEGER PRIMARY KEY,
    scroll_percent REAL NOT NULL DEFAULT 0,
    chapter TEXT,
    time_updated INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE
) STRICT;
