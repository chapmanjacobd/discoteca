package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// Migrate runs schema migrations on an existing database
func Migrate(db *sql.DB) error {
	// 1. Column migrations (Add new ones first)
	if err := migrateColumns(db); err != nil {
		return err
	}

	// 2. Data consolidation and table cleanup
	if err := cleanupMediaTable(db); err != nil {
		return err
	}

	// 3. Table migrations (FTS etc)
	if err := migrateTables(db); err != nil {
		return err
	}

	// 4. Index migrations
	if err := migrateIndexes(db); err != nil {
		return err
	}

	return nil
}

// pathToFtsPath converts a file path to FTS-friendly format
func pathToFtsPath(path string) string {
	re := regexp.MustCompile(`[/\\.\[\]\-\+(){}_&]`)
	s := re.ReplaceAllString(path, " ")
	return cleanString(s)
}

// cleanString removes brackets, special chars, and normalizes whitespace
func cleanString(s string) string {
	s = removeTextInsideBrackets(s)
	s = strings.ReplaceAll(s, "\x7f", "")
	s = strings.ReplaceAll(s, "&", "")
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "#", "")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, "|", "")
	s = strings.ReplaceAll(s, "^", "")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, ")", "")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, ">", "")
	s = strings.ReplaceAll(s, "<", "")
	s = strings.ReplaceAll(s, "\\", " ")
	s = strings.ReplaceAll(s, "/", " ")

	s = removeConsecutives(s, []string{"."})
	s = strings.ReplaceAll(s, "(", " ")
	s = strings.ReplaceAll(s, "-.", ".")
	s = strings.ReplaceAll(s, " - ", " ")
	s = strings.ReplaceAll(s, "- ", " ")
	s = strings.ReplaceAll(s, " -", " ")
	s = strings.ReplaceAll(s, " _ ", "_")
	s = strings.ReplaceAll(s, " _", "_")
	s = strings.ReplaceAll(s, "_ ", "_")

	s = removeConsecutiveWhitespace(s)

	return s
}

func removeTextInsideBrackets(s string) string {
	var result strings.Builder
	depth := 0
	for _, r := range s {
		if r == '(' || r == '[' || r == '{' {
			depth++
		} else if r == ')' || r == ']' || r == '}' {
			if depth > 0 {
				depth--
			}
		} else if depth == 0 {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func removeConsecutiveWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func removeConsecutives(s string, chars []string) string {
	for _, char := range chars {
		re := regexp.MustCompile(regexp.QuoteMeta(char) + "+")
		s = re.ReplaceAllString(s, char)
	}
	return s
}

func migrateColumns(db *sql.DB) error {
	cols := []struct {
		table  string
		column string
		schema string
	}{
		{"playlists", "title", "TEXT"},
		{"playlist_items", "time_added", "INTEGER DEFAULT 0"},
		{"media", "fts_path", "TEXT"},
		{"media", "description", "TEXT"},
		{"media", "time_downloaded", "INTEGER"},
	}

	for _, c := range cols {
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", c.table))
		if err != nil {
			if strings.Contains(err.Error(), "no such table") {
				continue
			}
			return err
		}

		exists := false
		for rows.Next() {
			var cid int
			var name, dtype string
			var notnull, pk int
			var dfltValue any
			if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
				rows.Close()
				return err
			}
			if strings.EqualFold(name, c.column) {
				exists = true
				break
			}
		}
		rows.Close()

		if !exists {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.table, c.column, c.schema)); err != nil {
				if !strings.Contains(err.Error(), "no such table") {
					return fmt.Errorf("failed to add column %s to table %s: %w", c.column, c.table, err)
				}
			}

			if c.table == "media" && c.column == "fts_path" {
				// New column added, populate it for existing rows
				if err := populateFtsPath(db); err != nil {
					return fmt.Errorf("failed to populate fts_path: %w", err)
				}
			}
		}
	}
	return nil
}

func cleanupMediaTable(db *sql.DB) error {
	// 1. Check if we need cleanup (do dead columns exist?)
	rows, err := db.Query("PRAGMA table_info(media)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasDeadColumns := false
	deadColumns := map[string]bool{
		"upvote_ratio": true, "num_comments": true, "favorite_count": true,
		"view_count": true, "time_uploaded": true, "uploader": true,
		"webpath": true, "city": true, "country": true,
		"latitude": true, "longitude": true, "decade": true,
		"mood": true, "bpm": true, "key": true, "extension": true,
	}

	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if deadColumns[strings.ToLower(name)] {
			hasDeadColumns = true
			break
		}
	}
	rows.Close()

	if !hasDeadColumns {
		return nil
	}

	// 2. Consolidate metadata into description before dropping columns
	// decade, mood, bpm, key
	_, _ = db.Exec(`UPDATE media SET description = 
        COALESCE(description, '') || 
        CASE WHEN decade IS NOT NULL AND decade != '' THEN '\nDate: ' || decade ELSE '' END ||
        CASE WHEN mood IS NOT NULL AND mood != '' THEN '\nMood: ' || mood ELSE '' END ||
        CASE WHEN bpm IS NOT NULL AND bpm != 0 THEN '\nBPM: ' || bpm ELSE '' END ||
        CASE WHEN "key" IS NOT NULL AND "key" != '' THEN '\nKey: ' || "key" ELSE '' END
        WHERE decade IS NOT NULL OR mood IS NOT NULL OR bpm IS NOT NULL OR "key" IS NOT NULL`)

	// 3. Recreate table (SQLite standard way to drop multiple columns)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	sqls := []string{
		`CREATE TABLE media_dg_tmp (
            path TEXT PRIMARY KEY,
            fts_path TEXT,
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
            type TEXT,
            width INTEGER,
            height INTEGER,
            fps REAL,
            video_codecs TEXT,
            audio_codecs TEXT,
            subtitle_codecs TEXT,
            video_count INTEGER DEFAULT 0,
            audio_count INTEGER DEFAULT 0,
            subtitle_count INTEGER DEFAULT 0,
            album TEXT,
            artist TEXT,
            genre TEXT,
            categories TEXT,
            description TEXT,
            language TEXT,
            time_downloaded INTEGER,
            score REAL
        )`,
		`INSERT INTO media_dg_tmp (
            path, fts_path, title, duration, size, time_created, time_modified,
            time_deleted, time_first_played, time_last_played, play_count, playhead,
            type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs,
            video_count, audio_count, subtitle_count, album, artist, genre,
            categories, description, language, time_downloaded, score
        ) SELECT 
            path, fts_path, title, duration, size, time_created, time_modified,
            time_deleted, time_first_played, time_last_played, play_count, playhead,
            type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs,
            video_count, audio_count, subtitle_count, album, artist, genre,
            categories, description, language, time_downloaded, score
        FROM media`,
		`DROP TABLE media`,
		`ALTER TABLE media_dg_tmp RENAME TO media`,
	}

	for _, sql := range sqls {
		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("failed cleanup step: %w", err)
		}
	}

	return tx.Commit()
}

func populateFtsPath(db *sql.DB) error {
	rows, err := db.Query("SELECT path FROM media WHERE fts_path IS NULL")
	if err != nil {
		return err
	}
	defer rows.Close()

	var updates []struct {
		path    string
		ftsPath string
	}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}
		updates = append(updates, struct {
			path    string
			ftsPath string
		}{path, pathToFtsPath(path)})
	}
	rows.Close()

	if len(updates) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE media SET fts_path = ? WHERE path = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, u := range updates {
		if _, err := stmt.Exec(u.ftsPath, u.path); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func migrateTables(db *sql.DB) error {
	// Create custom_keywords table if it doesn't exist
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS custom_keywords (
		category TEXT NOT NULL,
		keyword TEXT NOT NULL,
		PRIMARY KEY (category, keyword)
	)`); err != nil {
		return fmt.Errorf("failed to create custom_keywords table: %w", err)
	}

	// Check if FTS tables need upgrade to trigram or new columns
	upgradeFTS := func(tableName string, expectedSqlPart string) error {
		var existingSql string
		err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&existingSql)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil // Table doesn't exist
			}
			return err
		}

		if !strings.Contains(existingSql, "trigram") || (expectedSqlPart != "" && !strings.Contains(existingSql, expectedSqlPart)) {
			// Needs upgrade - drop it
			if _, err := db.Exec(fmt.Sprintf("DROP TABLE %s", tableName)); err != nil {
				return fmt.Errorf("failed to drop %s for upgrade: %w", tableName, err)
			}

			// Recreate immediately
			var createSql string
			if tableName == "media_fts" {
				createSql = `CREATE VIRTUAL TABLE media_fts USING fts5(
					path,
					fts_path,
					title,
                    description,
					content='media',
					content_rowid='rowid',
					tokenize = 'trigram'
				);`
			} else if tableName == "captions_fts" {
				createSql = `CREATE VIRTUAL TABLE captions_fts USING fts5(
					media_path UNINDEXED,
					text,
					content='captions',
					tokenize = 'trigram'
				);`
			}

			if _, err := db.Exec(createSql); err != nil {
				return fmt.Errorf("failed to recreate %s: %w", tableName, err)
			}

			// Recreate triggers if it's media_fts
			if tableName == "media_fts" {
				triggerSqls := []string{
					`CREATE TRIGGER IF NOT EXISTS media_ai AFTER INSERT ON media BEGIN
						INSERT INTO media_fts(rowid, path, fts_path, title, description)
						VALUES (new.rowid, new.path, new.fts_path, new.title, new.description);
					END;`,
					`CREATE TRIGGER IF NOT EXISTS media_ad AFTER DELETE ON media BEGIN
						DELETE FROM media_fts WHERE rowid = old.rowid;
					END;`,
					`CREATE TRIGGER IF NOT EXISTS media_au AFTER UPDATE ON media BEGIN
						INSERT INTO media_fts(media_fts, rowid, path, fts_path, title, description) VALUES('delete', old.rowid, old.path, old.fts_path, old.title, old.description);
						INSERT INTO media_fts(rowid, path, fts_path, title, description) VALUES (new.rowid, new.path, new.fts_path, new.title, new.description);
					END;`,
				}
				for _, tsql := range triggerSqls {
					if _, err := db.Exec(tsql); err != nil {
						return fmt.Errorf("failed to recreate trigger: %w", err)
					}
				}
			}

			// Rebuild data
			if _, err := db.Exec(fmt.Sprintf("INSERT INTO %s(%s) VALUES('rebuild')", tableName, tableName)); err != nil {
				// Non-fatal, might be empty
				return nil
			}
		}
		return nil
	}

	if err := upgradeFTS("media_fts", "description"); err != nil {
		return err
	}
	if err := upgradeFTS("captions_fts", ""); err != nil {
		return err
	}

	return nil
}

func migrateIndexes(db *sql.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_type ON media(type)",
		"CREATE INDEX IF NOT EXISTS idx_genre ON media(genre)",
		"CREATE INDEX IF NOT EXISTS idx_artist ON media(artist)",
		"CREATE INDEX IF NOT EXISTS idx_album ON media(album)",
		"CREATE INDEX IF NOT EXISTS idx_categories ON media(categories)",
		"CREATE INDEX IF NOT EXISTS idx_score ON media(score)",
		"CREATE INDEX IF NOT EXISTS idx_time_created ON media(time_created)",
		"CREATE INDEX IF NOT EXISTS idx_time_modified ON media(time_modified)",
		"CREATE INDEX IF NOT EXISTS idx_time_downloaded ON media(time_downloaded)",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	return nil
}
