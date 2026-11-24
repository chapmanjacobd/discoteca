-- FTS for captions
CREATE VIRTUAL TABLE IF NOT EXISTS captions_fts USING fts5(
    media_id UNINDEXED,
    text,
    content='captions',
    tokenize = 'trigram',
    detail = 'full'
);

-- Triggers for captions FTS
CREATE TRIGGER IF NOT EXISTS captions_ai AFTER INSERT ON captions BEGIN
    INSERT INTO captions_fts(rowid, media_id, text)
    VALUES (new.rowid, new.media_id, new.text);
END;

CREATE TRIGGER IF NOT EXISTS captions_ad AFTER DELETE ON captions BEGIN
    DELETE FROM captions_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS captions_au AFTER UPDATE ON captions BEGIN
    INSERT INTO captions_fts(captions_fts, rowid, media_id, text)
    VALUES('delete', old.rowid, old.media_id, old.text);
    INSERT INTO captions_fts(rowid, media_id, text)
    VALUES (new.rowid, new.media_id, new.text);
END;
