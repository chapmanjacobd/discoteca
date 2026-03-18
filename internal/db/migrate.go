package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// Migrate runs schema migrations on an existing database
func Migrate(db *sql.DB) error {
	// 0. Check SQLite version for STRICT support (3.37.0+)
	var version string
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&version); err != nil {
		return err
	}
	hasStrict := isVersionGreaterOrEqual(version, "3.37.0")

	// 1. Column migrations (Add new ones first)
	if err := migrateColumns(db); err != nil {
		return err
	}

	// 2. Data consolidation and table cleanup (now includes STRICT)
	if err := cleanupMediaTable(db, hasStrict); err != nil {
		return err
	}

	// 3. Table migrations (FTS etc, and STRICT for other tables)
	if err := migrateTables(db, hasStrict); err != nil {
		return err
	}

	// 4. Index migrations
	if err := migrateIndexes(db); err != nil {
		return err
	}

	return nil
}

func isVersionGreaterOrEqual(v, target string) bool {
	var v1, v2, v3 int
	fmt.Sscanf(v, "%d.%d.%d", &v1, &v2, &v3)
	var t1, t2, t3 int
	fmt.Sscanf(target, "%d.%d.%d", &t1, &t2, &t3)

	if v1 != t1 {
		return v1 > t1
	}
	if v2 != t2 {
		return v2 > t2
	}
	return v3 >= t3
}

func isTableStrict(db *sql.DB, tableName string) (bool, error) {
	var isStrict bool
	// PRAGMA table_list is available since 3.37.0
	err := db.QueryRow(fmt.Sprintf("SELECT strict FROM pragma_table_list WHERE name='%s'", tableName)).Scan(&isStrict)
	if err != nil {
		// If table_list is not available or table not found, assume not strict
		return false, nil
	}
	return isStrict, nil
}

func migrateToStrict(db *sql.DB, tableName string, createSql string) error {
	strict, err := isTableStrict(db, tableName)
	if err != nil {
		return err
	}
	if strict {
		return nil
	}

	// Disable foreign key checks during migration to avoid constraint violations
	// when copying data that may reference deleted entries
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		// Re-enable foreign key checks after migration
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			// Log but don't fail on this error
			fmt.Printf("Warning: failed to re-enable foreign keys: %v\n", err)
		}
	}()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Rename old table
	oldTable := tableName + "_old_strict"
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tableName, oldTable)); err != nil {
		return fmt.Errorf("failed to rename %s: %w", tableName, err)
	}

	// Create new STRICT table
	if _, err := tx.Exec(createSql); err != nil {
		return fmt.Errorf("failed to create strict %s: %w", tableName, err)
	}

	// Copy data
	// Get columns from old table to ensure they match
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s)", oldTable))
	if err != nil {
		return err
	}
	var cols []string
	for rows.Next() {
		var name string
		var ignored any
		if err := rows.Scan(&ignored, &name, &ignored, &ignored, &ignored, &ignored); err != nil {
			rows.Close()
			return err
		}
		cols = append(cols, name)
	}
	rows.Close()

	colStr := strings.Join(cols, ", ")
	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s", tableName, colStr, colStr, oldTable)); err != nil {
		return fmt.Errorf("failed to copy data for %s: %w", tableName, err)
	}

	// Drop old table
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE %s", oldTable)); err != nil {
		return fmt.Errorf("failed to drop %s: %w", oldTable, err)
	}

	return tx.Commit()
}

// pathToTokenized converts a file path to FTS-friendly format
func pathToTokenized(path string) string {
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
		{"media", "path_tokenized", "TEXT"},
		{"media", "description", "TEXT"},
		{"media", "time_downloaded", "INTEGER"},
		{"media", "play_count", "INTEGER DEFAULT 0"},
		{"media", "time_first_played", "INTEGER DEFAULT 0"},
		{"media", "time_last_played", "INTEGER DEFAULT 0"},
		{"media", "video_codecs", "TEXT"},
		{"media", "audio_codecs", "TEXT"},
		{"media", "subtitle_codecs", "TEXT"},
		{"media", "video_count", "INTEGER DEFAULT 0"},
		{"media", "audio_count", "INTEGER DEFAULT 0"},
		{"media", "subtitle_count", "INTEGER DEFAULT 0"},
		{"media", "album", "TEXT"},
		{"media", "artist", "TEXT"},
		{"media", "genre", "TEXT"},
		{"media", "categories", "TEXT"},
		{"media", "language", "TEXT"},
		{"media", "score", "REAL"},
		{"media", "width", "INTEGER"},
		{"media", "height", "INTEGER"},
		{"media", "fps", "REAL"},
		{"media", "playhead", "INTEGER DEFAULT 0"},
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

			if c.table == "media" && c.column == "path_tokenized" {
				// New column added, populate it for existing rows
				if err := populatePathTokenized(db); err != nil {
					return fmt.Errorf("failed to populate path_tokenized: %w", err)
				}
			}
		}
	}
	return nil
}

func cleanupMediaTable(db *sql.DB, hasStrict bool) error {
	// 1. Check if we need cleanup (do dead columns exist?) OR if we need to migrate to STRICT
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

	strict, _ := isTableStrict(db, "media")
	needsStrictMigration := hasStrict && !strict

	if !hasDeadColumns && !needsStrictMigration {
		return nil
	}

	// 2. Consolidate metadata into description before dropping columns
	if hasDeadColumns {
		_, _ = db.Exec(`UPDATE media SET description =
            COALESCE(description, '') ||
            CASE WHEN decade IS NOT NULL AND decade != '' THEN '\nDate: ' || decade ELSE '' END ||
            CASE WHEN mood IS NOT NULL AND mood != '' THEN '\nMood: ' || mood ELSE '' END ||
            CASE WHEN bpm IS NOT NULL AND bpm != 0 THEN '\nBPM: ' || bpm ELSE '' END ||
            CASE WHEN "key" IS NOT NULL AND "key" != '' THEN '\nKey: ' || "key" ELSE '' END
            WHERE decade IS NOT NULL OR mood IS NOT NULL OR bpm IS NOT NULL OR "key" IS NOT NULL`)
	}

	// 3. Recreate table (SQLite standard way to drop multiple columns and/or add STRICT)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	strictSql := ""
	if hasStrict {
		strictSql = "STRICT"
	}

	colsDef := "path TEXT PRIMARY KEY, path_tokenized TEXT,"
	colsNames := "path, path_tokenized,"
	if !IsFtsEnabled() {
		colsDef = "path TEXT PRIMARY KEY,"
		colsNames = "path,"
	}

	sqls := []string{
		fmt.Sprintf(`CREATE TABLE media_dg_tmp (
            %s
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
        ) %s`, colsDef, strictSql),
		fmt.Sprintf(`INSERT INTO media_dg_tmp (
            %s title, duration, size, time_created, time_modified,
            time_deleted, time_first_played, time_last_played, play_count, playhead,
            type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs,
            video_count, audio_count, subtitle_count, album, artist, genre,
            categories, description, language, time_downloaded, score
        ) SELECT
            %s title, duration, size, time_created, time_modified,
            time_deleted, time_first_played, time_last_played, play_count, playhead,
            type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs,
            video_count, audio_count, subtitle_count, album, artist, genre,
            categories, description, language, time_downloaded, score
        FROM media`, colsNames, colsNames),
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

func populatePathTokenized(db *sql.DB) error {
	rows, err := db.Query("SELECT path FROM media WHERE path_tokenized IS NULL")
	if err != nil {
		return err
	}
	defer rows.Close()

	var updates []struct {
		path      string
		tokenized string
	}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}
		updates = append(updates, struct {
			path      string
			tokenized string
		}{path, pathToTokenized(path)})
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

	stmt, err := tx.Prepare("UPDATE media SET path_tokenized = ? WHERE path = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, u := range updates {
		if _, err := stmt.Exec(u.tokenized, u.path); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func migrateTables(db *sql.DB, hasStrict bool) error {
	strictSql := ""
	if hasStrict {
		strictSql = "STRICT"
	}

	// 1. Migrate small tables to STRICT
	if hasStrict {
		migrations := []struct {
			name string
			sql  string
		}{
			{"playlists", fmt.Sprintf(`CREATE TABLE playlists (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                path TEXT UNIQUE,
                title TEXT,
                extractor_key TEXT,
                extractor_config TEXT,
                time_deleted INTEGER DEFAULT 0
            ) %s`, strictSql)},
			{"playlist_items", fmt.Sprintf(`CREATE TABLE playlist_items (
                playlist_id INTEGER NOT NULL,
                media_path TEXT NOT NULL,
                track_number INTEGER,
                time_added INTEGER DEFAULT (unixepoch()),
                PRIMARY KEY (playlist_id, media_path),
                FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) %s`, strictSql)},
			{"captions", fmt.Sprintf(`CREATE TABLE captions (
                media_path TEXT NOT NULL,
                time REAL,
                text TEXT,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) %s`, strictSql)},
			{"history", fmt.Sprintf(`CREATE TABLE history (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                media_path TEXT NOT NULL,
                time_played INTEGER DEFAULT (unixepoch()),
                playhead INTEGER,
                done INTEGER,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) %s`, strictSql)},
			{"custom_keywords", fmt.Sprintf(`CREATE TABLE custom_keywords (
                category TEXT NOT NULL,
                keyword TEXT NOT NULL,
                PRIMARY KEY (category, keyword)
            ) %s`, strictSql)},
		}

		for _, m := range migrations {
			// Skip captions table migration if FTS is disabled
			if !IsFtsEnabled() && m.name == "captions" {
				continue
			}
			// Check if table exists before migrating
			var exists int
			err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", m.name).Scan(&exists)
			if err != nil {
				return err
			}
			if exists > 0 {
				if err := migrateToStrict(db, m.name, m.sql); err != nil {
					return fmt.Errorf("migrateToStrict failed for %s: %w", m.name, err)
				}
			} else {
				// Create it if it doesn't exist
				if _, err := db.Exec(m.sql); err != nil {
					return fmt.Errorf("failed to create %s: %w", m.name, err)
				}
			}
		}
	} else {
		// Just ensure custom_keywords exists if not using STRICT (older SQLite)
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS custom_keywords (
            category TEXT NOT NULL,
            keyword TEXT NOT NULL,
            PRIMARY KEY (category, keyword)
        )`); err != nil {
			return fmt.Errorf("failed to create custom_keywords table: %w", err)
		}
	}

	// Ensure folder_stats and _maintenance_meta tables exist
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS folder_stats (
		parent TEXT PRIMARY KEY,
		depth INTEGER,
		file_count INTEGER,
		total_size INTEGER,
		total_duration INTEGER
	)`); err != nil {
		return fmt.Errorf("failed to create folder_stats table: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _maintenance_meta (
		key TEXT PRIMARY KEY,
		value TEXT,
		last_updated INTEGER
	)`); err != nil {
		return fmt.Errorf("failed to create _maintenance_meta table: %w", err)
	}

	// Initialize maintenance tracking keys
	if _, err := db.Exec(`INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('folder_stats_last_refresh', '0', 0)`); err != nil {
		return fmt.Errorf("failed to initialize maintenance metadata: %w", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('fts_last_rebuild', '0', 0)`); err != nil {
		return fmt.Errorf("failed to initialize maintenance metadata: %w", err)
	}

	// Create index on folder_stats for faster depth-based queries
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_folder_stats_depth ON folder_stats(depth)`); err != nil {
		return fmt.Errorf("failed to create folder_stats index: %w", err)
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

		// Normalize whitespace for comparison
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
					path_tokenized,
					title,
                    description,
					time_deleted UNINDEXED,
					content='media',
					content_rowid='rowid',
					tokenize = 'trigram',
					detail = 'full'
				);`
			} else if tableName == "captions_fts" {
				createSql = `CREATE VIRTUAL TABLE captions_fts USING fts5(
					media_path UNINDEXED,
					text,
					content='captions',
					tokenize = 'trigram',
					detail = 'full'
				);`
			}

			if _, err := db.Exec(createSql); err != nil {
				return fmt.Errorf("failed to recreate %s: %w", tableName, err)
			}

			// Recreate triggers if it's media_fts
			if tableName == "media_fts" {
				triggerSqls := []string{
					`CREATE TRIGGER IF NOT EXISTS media_ai AFTER INSERT ON media BEGIN
						INSERT INTO media_fts(rowid, path, path_tokenized, title, description, time_deleted)
						VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description, new.time_deleted);
					END;`,
					`CREATE TRIGGER IF NOT EXISTS media_ad AFTER DELETE ON media BEGIN
						DELETE FROM media_fts WHERE rowid = old.rowid;
					END;`,
					`CREATE TRIGGER IF NOT EXISTS media_au AFTER UPDATE ON media BEGIN
						INSERT INTO media_fts(media_fts, rowid, path, path_tokenized, title, description, time_deleted) VALUES('delete', old.rowid, old.path, old.path_tokenized, old.title, old.description, old.time_deleted);
						INSERT INTO media_fts(rowid, path, path_tokenized, title, description, time_deleted) VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description, new.time_deleted);
					END;`,
				}
				for _, tsql := range triggerSqls {
					if _, err := db.Exec(tsql); err != nil {
						return fmt.Errorf("failed to recreate trigger: %w", err)
					}
				}
			}

			// Rebuild data
			if tableName == "media_fts" {
				if _, err := db.Exec("INSERT INTO media_fts(rowid, path, path_tokenized, title, description, time_deleted) SELECT rowid, path, path_tokenized, title, description, time_deleted FROM media"); err != nil {
					return nil
				}
			} else if tableName == "captions_fts" {
				if _, err := db.Exec("INSERT INTO captions_fts(rowid, media_path, text) SELECT rowid, media_path, text FROM captions"); err != nil {
					return nil
				}
			}
		}
		return nil
	}

	if IsFtsEnabled() {
		if err := upgradeFTS("media_fts", "description"); err != nil {
			return err
		}
		if err := upgradeFTS("captions_fts", ""); err != nil {
			return err
		}
	}

	return nil
}

func migrateIndexes(db *sql.DB) error {
	indexes := []string{
		// Core indexes
		"CREATE INDEX IF NOT EXISTS idx_path ON media(path)",
		"CREATE INDEX IF NOT EXISTS idx_type ON media(type)",
		"CREATE INDEX IF NOT EXISTS idx_genre ON media(genre)",
		"CREATE INDEX IF NOT EXISTS idx_artist ON media(artist)",
		"CREATE INDEX IF NOT EXISTS idx_album ON media(album)",
		"CREATE INDEX IF NOT EXISTS idx_categories ON media(categories)",
		"CREATE INDEX IF NOT EXISTS idx_score ON media(score)",
		"CREATE INDEX IF NOT EXISTS idx_time_created ON media(time_created)",
		"CREATE INDEX IF NOT EXISTS idx_time_modified ON media(time_modified)",
		"CREATE INDEX IF NOT EXISTS idx_time_downloaded ON media(time_downloaded)",
		"CREATE INDEX IF NOT EXISTS idx_size ON media(size)",
		"CREATE INDEX IF NOT EXISTS idx_duration ON media(duration)",
		// Composite indexes for common filtered queries
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_type ON media(time_deleted, type)",
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_size ON media(time_deleted, size)",
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_duration ON media(time_deleted, duration)",
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_path ON media(time_deleted, path)",
		// Partial index for active media (most common query pattern)
		"CREATE INDEX IF NOT EXISTS idx_media_active ON media(path, type) WHERE time_deleted = 0",
		// Indexes for filter bins calculation (optimize include_counts)
		"CREATE INDEX IF NOT EXISTS idx_media_active_size ON media(size) WHERE time_deleted = 0 AND size > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_duration ON media(duration) WHERE time_deleted = 0 AND duration > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_time_modified ON media(time_modified) WHERE time_deleted = 0 AND time_modified > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_time_created ON media(time_created) WHERE time_deleted = 0 AND time_created > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_time_downloaded ON media(time_downloaded) WHERE time_deleted = 0 AND time_downloaded > 0",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Remove unused function-based index (SQLite can't use it efficiently)
	if _, err := db.Exec("DROP INDEX IF EXISTS idx_path_prefix"); err != nil {
		return fmt.Errorf("failed to drop idx_path_prefix: %w", err)
	}

	// Populate folder_stats materialized view
	if err := PopulateFolderStatsInGo(db); err != nil {
		return fmt.Errorf("failed to populate folder_stats: %w", err)
	}

	return nil
}
