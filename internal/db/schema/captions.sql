CREATE TABLE IF NOT EXISTS captions (
    media_id INTEGER NOT NULL,
    time REAL,
    text TEXT,
    FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE
) STRICT;
