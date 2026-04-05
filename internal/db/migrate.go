package db

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

func renameMediaTypeColumn(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(media)")
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil
		}
		return err
	}
	defer rows.Close()

	hasType := false
	hasMediaType := false
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if strings.EqualFold(name, "type") {
			hasType = true
		}
		if strings.EqualFold(name, "media_type") {
			hasMediaType = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if hasType && !hasMediaType {
		if _, err := db.ExecContext(
			ctx,
			"ALTER TABLE media RENAME COLUMN type TO media_type",
		); err != nil {
			return fmt.Errorf("failed to rename column type to media_type: %w", err)
		}
	}
	return nil
}

// Migrate runs schema migrations on an existing database
func Migrate(ctx context.Context, db *sql.DB) error {
	// 0. Check SQLite version for STRICT support (3.37.0+)
	var version string
	if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err != nil {
		return err
	}
	hasStrict := isVersionGreaterOrEqual(version, "3.37.0")

	// 0.1 Rename 'type' to 'media_type' if needed
	if err := renameMediaTypeColumn(ctx, db); err != nil {
		return err
	}

	// 1. Column migrations (Add new ones first)
	if err := migrateColumns(ctx, db); err != nil {
		return err
	}

	// 2. Data consolidation and table cleanup (now includes STRICT)
	if err := cleanupMediaTable(ctx, db, hasStrict); err != nil {
		return err
	}

	// 3. Table migrations (FTS etc, and STRICT for other tables)
	if err := migrateTables(ctx, db, hasStrict); err != nil {
		return err
	}

	// 4. Index migrations
	return migrateIndexes(ctx, db)
}

func isVersionGreaterOrEqual(v, target string) bool {
	var v1, v2, v3 int
	_, _ = fmt.Sscanf(v, "%d.%d.%d", &v1, &v2, &v3)
	var t1, t2, t3 int
	_, _ = fmt.Sscanf(target, "%d.%d.%d", &t1, &t2, &t3)

	if v1 != t1 {
		return v1 > t1
	}
	if v2 != t2 {
		return v2 > t2
	}
	return v3 >= t3
}

func isTableStrict(ctx context.Context, db *sql.DB, tableName string) bool {
	var isStrict bool
	// PRAGMA table_list is available since 3.37.0
	err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT strict FROM pragma_table_list WHERE name='%s'", tableName)).
		Scan(&isStrict)
	if err != nil {
		// If table_list is not available or table not found, assume not strict
		return false
	}
	return isStrict
}

func migrateToStrict(ctx context.Context, db *sql.DB, tableName, createSQL string) error {
	strict := isTableStrict(ctx, db, tableName)
	if strict {
		return nil
	}

	// Disable foreign key checks during migration to avoid constraint violations
	// when copying data that may reference deleted entries
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		// Re-enable foreign key checks after migration
		if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
			// Log but don't fail on this error
			fmt.Printf("Warning: failed to re-enable foreign keys: %v\n", err)
		}
	}()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Rename old table
	oldTable := tableName + "_old_strict"
	if _, err2 := tx.ExecContext(
		ctx,
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tableName, oldTable),
	); err2 != nil {
		return fmt.Errorf("failed to rename %s: %w", tableName, err2)
	}

	// Create new STRICT table
	if _, err2 := tx.ExecContext(ctx, createSQL); err2 != nil {
		return fmt.Errorf("failed to create strict %s: %w", tableName, err2)
	}

	// Copy data
	// Get columns from old table to ensure they match
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", oldTable))
	if err != nil {
		return err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var name string
		var ignored any
		if err := rows.Scan(&ignored, &name, &ignored, &ignored, &ignored, &ignored); err != nil {
			return err
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	colStr := strings.Join(cols, ", ")
	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s", tableName, colStr, colStr, oldTable),
	); err != nil {
		return fmt.Errorf("failed to copy data for %s: %w", tableName, err)
	}

	// Drop old table
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", oldTable)); err != nil {
		return fmt.Errorf("failed to drop %s: %w", oldTable, err)
	}

	return tx.Commit()
}

// convertColumnsBeforeStrict handles column type conversions before STRICT migration
// e.g., history.media_id (INTEGER) -> history.media_path (TEXT)
func convertColumnsBeforeStrict(ctx context.Context, db *sql.DB) error {
	// Check if history table has media_id column that needs conversion
	if err := convertHistoryMediaID(ctx, db); err != nil {
		return fmt.Errorf("failed to convert history.media_id: %w", err)
	}

	// Check if captions table has media_id column that needs conversion
	if err := convertCaptionsMediaID(ctx, db); err != nil {
		return fmt.Errorf("failed to convert captions.media_id: %w", err)
	}

	return nil
}

func convertHistoryMediaID(ctx context.Context, db *sql.DB) error {
	var hasMediaID bool
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(history)")
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil // Table doesn't exist yet, nothing to convert
		}
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue any
		if scanErr := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); scanErr != nil {
			return scanErr
		}
		if strings.EqualFold(name, "media_id") {
			hasMediaID = true
			break
		}
	}
	if err2 := rows.Err(); err2 != nil {
		return err2
	}

	if !hasMediaID {
		return nil // No conversion needed
	}

	// Convert media_id to media_path by joining with media table
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Create temp history table with media_path
	if _, err := tx.ExecContext(ctx, `CREATE TABLE history_tmp (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_path TEXT NOT NULL,
		time_played INTEGER,
		playhead INTEGER,
		done INTEGER,
		FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("failed to create history_tmp: %w", err)
	}

	// Copy data, converting media_id to media_path via JOIN
	// LEFT JOIN to preserve history entries even if media was deleted
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO history_tmp (id, media_path, time_played, playhead, done)
		SELECT h.id, COALESCE(m.path, ''), h.time_played, h.playhead, h.done
		FROM history h
		LEFT JOIN media m ON h.media_id = m.rowid`,
	); err != nil {
		return fmt.Errorf("failed to convert history media_id to media_path: %w", err)
	}

	// Drop old table and rename new one
	if _, err := tx.ExecContext(ctx, "DROP TABLE history"); err != nil {
		return fmt.Errorf("failed to drop old history: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "ALTER TABLE history_tmp RENAME TO history"); err != nil {
		return fmt.Errorf("failed to rename history_tmp: %w", err)
	}

	return tx.Commit()
}

func convertCaptionsMediaID(ctx context.Context, db *sql.DB) error {
	var hasMediaID bool
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(captions)")
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil // Table doesn't exist yet, nothing to convert
		}
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue any
		if scanErr := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); scanErr != nil {
			return scanErr
		}
		if strings.EqualFold(name, "media_id") {
			hasMediaID = true
			break
		}
	}
	if err2 := rows.Err(); err2 != nil {
		return err2
	}
	rows.Close()

	if !hasMediaID {
		return nil // No conversion needed
	}

	// Convert media_id to media_path by joining with media table
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Create temp captions table with media_path
	if _, err := tx.ExecContext(ctx, `CREATE TABLE captions_tmp (
		media_path TEXT NOT NULL,
		time REAL,
		text TEXT,
		FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("failed to create captions_tmp: %w", err)
	}

	// Copy data, converting media_id to media_path via JOIN
	if _, err := tx.ExecContext(ctx, `INSERT INTO captions_tmp (media_path, time, text)
		SELECT COALESCE(m.path, ''), c.time, c.text
		FROM captions c
		LEFT JOIN media m ON c.media_id = m.rowid`); err != nil {
		return fmt.Errorf("failed to convert captions media_id to media_path: %w", err)
	}

	// Drop old table and rename new one
	if _, err := tx.ExecContext(ctx, "DROP TABLE captions"); err != nil {
		return fmt.Errorf("failed to drop old captions: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "ALTER TABLE captions_tmp RENAME TO captions"); err != nil {
		return fmt.Errorf("failed to rename captions_tmp: %w", err)
	}

	return tx.Commit()
}

var (
	tokenRe             = regexp.MustCompile(`[/\\.\[\]\-\+(){}_&]`)
	cleanStringReplacer = strings.NewReplacer(
		"\\", " ",
		"/", " ",
		"(", " ",
		"-.", ".",
		" - ", " ",
		"- ", " ",
		" -", " ",
		" _ ", "_",
		" _", "_",
		"_ ", "_",
	)
)

// pathToTokenized converts a file path to FTS-friendly format
func pathToTokenized(path string) string {
	s := tokenRe.ReplaceAllString(path, " ")
	return cleanString(s)
}

// cleanString removes brackets, special chars, and normalizes whitespace
func cleanString(s string) string {
	s = removeTextInsideBrackets(s)

	// Drop unwanted characters efficiently
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\x7f', '&', '%', '*', '$', '#', '!', '?', '|', '^', '\'', '"', ')', ':', '>', '<':
			return -1
		}
		return r
	}, s)

	s = cleanStringReplacer.Replace(s)
	s = removeConsecutives(s, []string{"."})
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

func migrateColumns(ctx context.Context, db *sql.DB) error {
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
		{"media", "fasthash", "TEXT"},
		{"media", "sha256", "TEXT"},
		{"media", "is_deduped", "INTEGER DEFAULT 0"},
	}

	for _, c := range cols {
		err := func() error {
			rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", c.table))
			if err != nil {
				if strings.Contains(err.Error(), "no such table") {
					return nil
				}
				return err
			}
			defer rows.Close()

			exists := false
			for rows.Next() {
				var cid int
				var name, dtype string
				var notnull, pk int
				var dfltValue any
				if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
					return err
				}
				if strings.EqualFold(name, c.column) {
					exists = true
					break
				}
			}
			if err := rows.Err(); err != nil {
				return err
			}

			if !exists {
				if _, err := db.ExecContext(
					ctx,
					fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.table, c.column, c.schema),
				); err != nil {
					if !strings.Contains(err.Error(), "no such table") {
						return fmt.Errorf("failed to add column %s to table %s: %w", c.column, c.table, err)
					}
				}

				if c.table == "media" && c.column == "path_tokenized" {
					// New column added, populate it for existing rows
					if err := populatePathTokenized(ctx, db); err != nil {
						return fmt.Errorf("failed to populate path_tokenized: %w", err)
					}
				}
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func cleanupMediaTable(ctx context.Context, db *sql.DB, hasStrict bool) error {
	hasDead, err := needsCleanup(ctx, db, hasStrict)
	if err != nil || !hasDead {
		return err
	}

	// 2. Consolidate metadata into description before dropping columns
	if err := consolidateMetadata(ctx, db); err != nil {
		return err
	}

	// 3. Recreate table (SQLite standard way to drop multiple columns and/or add STRICT)
	return recreateMediaTable(ctx, db, hasStrict)
}

func needsCleanup(ctx context.Context, db *sql.DB, hasStrict bool) (bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(media)")
	if err != nil {
		return false, err
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
		if scanErr := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); scanErr != nil {
			return false, scanErr
		}
		if deadColumns[strings.ToLower(name)] {
			hasDeadColumns = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	strict := isTableStrict(ctx, db, "media")
	needsStrictMigration := hasStrict && !strict

	return hasDeadColumns || needsStrictMigration, nil
}

func consolidateMetadata(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `UPDATE media SET description =
		COALESCE(description, '') ||
		CASE WHEN decade IS NOT NULL AND decade != '' THEN '\nDate: ' || decade ELSE '' END ||
		CASE WHEN mood IS NOT NULL AND mood != '' THEN '\nMood: ' || mood ELSE '' END ||
		CASE WHEN bpm IS NOT NULL AND bpm != 0 THEN '\nBPM: ' || bpm ELSE '' END ||
		CASE WHEN "key" IS NOT NULL AND "key" != '' THEN '\nKey: ' || "key" ELSE '' END
		WHERE decade IS NOT NULL OR mood IS NOT NULL OR bpm IS NOT NULL OR "key" IS NOT NULL`)
	return err
}

func recreateMediaTable(ctx context.Context, db *sql.DB, hasStrict bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	strictSQL := ""
	if hasStrict {
		strictSQL = "STRICT"
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
            media_type TEXT,
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
            score REAL,
            fasthash TEXT,
            sha256 TEXT,
            is_deduped INTEGER DEFAULT 0
        ) %s`, colsDef, strictSQL),
		fmt.Sprintf(
			"INSERT INTO media_dg_tmp (%s title, duration, size, time_created, time_modified, time_deleted, time_first_played, time_last_played, play_count, playhead, media_type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs, video_count, audio_count, subtitle_count, album, artist, genre, categories, description, language, time_downloaded, score, fasthash, sha256, is_deduped) SELECT %s title, duration, size, time_created, time_modified, time_deleted, time_first_played, time_last_played, play_count, playhead, media_type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs, video_count, audio_count, subtitle_count, album, artist, genre, categories, description, language, time_downloaded, score, fasthash, sha256, is_deduped FROM media",
			colsNames,
			colsNames,
		),
		"DROP TABLE media",
		"ALTER TABLE media_dg_tmp RENAME TO media",
	}

	for _, s := range sqls {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func populatePathTokenized(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT path FROM media WHERE path_tokenized IS NULL")
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
		if scanErr := rows.Scan(&path); scanErr != nil {
			return scanErr
		}
		updates = append(updates, struct {
			path      string
			tokenized string
		}{path, pathToTokenized(path)})
	}
	if err2 := rows.Err(); err2 != nil {
		return err2
	}
	rows.Close()

	if len(updates) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "UPDATE media SET path_tokenized = ? WHERE path = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, u := range updates {
		if _, err := stmt.ExecContext(ctx, u.tokenized, u.path); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func migrateTables(ctx context.Context, db *sql.DB, hasStrict bool) error {
	strictSQL := ""
	if hasStrict {
		strictSQL = "STRICT"
	}

	// 0. Pre-migration: Handle column renames/conversions for tables with schema changes
	if err := convertColumnsBeforeStrict(ctx, db); err != nil {
		return fmt.Errorf("failed to convert columns: %w", err)
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
                time_created INTEGER,
                time_modified INTEGER,
                hours_update_delay INTEGER,
                time_deleted INTEGER DEFAULT 0
            ) %s`, strictSQL)},
			{"playlist_items", fmt.Sprintf(`CREATE TABLE playlist_items (
                playlist_id INTEGER NOT NULL,
                media_path TEXT NOT NULL,
                track_number INTEGER,
                time_added INTEGER DEFAULT (unixepoch()),
                PRIMARY KEY (playlist_id, media_path),
                FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) %s`, strictSQL)},
			{"captions", fmt.Sprintf(`CREATE TABLE captions (
                media_path TEXT NOT NULL,
                time REAL,
                text TEXT,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) %s`, strictSQL)},
			{"history", fmt.Sprintf(`CREATE TABLE history (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                media_path TEXT NOT NULL,
                time_played INTEGER DEFAULT (unixepoch()),
                playhead INTEGER,
                done INTEGER,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) %s`, strictSQL)},
			{"custom_keywords", fmt.Sprintf(`CREATE TABLE custom_keywords (
                category TEXT NOT NULL,
                keyword TEXT NOT NULL,
                PRIMARY KEY (category, keyword)
            ) %s`, strictSQL)},
		}

		for _, m := range migrations {
			// Skip captions table migration if FTS is disabled
			if !IsFtsEnabled() && m.name == "captions" {
				continue
			}
			// Check if table exists before migrating
			var exists int
			err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", m.name).
				Scan(&exists)
			if err != nil {
				return err
			}
			if exists > 0 {
				if err := migrateToStrict(ctx, db, m.name, m.sql); err != nil {
					return fmt.Errorf("migrateToStrict failed for %s: %w", m.name, err)
				}
			} else {
				// Create it if it doesn't exist
				if _, err := db.ExecContext(ctx, m.sql); err != nil {
					return fmt.Errorf("failed to create %s: %w", m.name, err)
				}
			}
		}
	} else {
		// Just ensure custom_keywords exists if not using STRICT (older SQLite)
		if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS custom_keywords (
            category TEXT NOT NULL,
            keyword TEXT NOT NULL,
            PRIMARY KEY (category, keyword)
        )`); err != nil {
			return fmt.Errorf("failed to create custom_keywords table: %w", err)
		}
	}

	// Ensure folder_stats and _maintenance_meta tables exist
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS folder_stats (
		parent TEXT PRIMARY KEY,
		depth INTEGER,
		file_count INTEGER,
		total_size INTEGER,
		total_duration INTEGER
	)`); err != nil {
		return fmt.Errorf("failed to create folder_stats table: %w", err)
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _maintenance_meta (
		key TEXT PRIMARY KEY,
		value TEXT,
		last_updated INTEGER
	)`); err != nil {
		return fmt.Errorf("failed to create _maintenance_meta table: %w", err)
	}

	// Initialize maintenance tracking keys
	if _, err := db.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('folder_stats_last_refresh', '0', 0)`,
	); err != nil {
		return fmt.Errorf("failed to initialize maintenance metadata: %w", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES ('fts_last_rebuild', '0', 0)`,
	); err != nil {
		return fmt.Errorf("failed to initialize maintenance metadata: %w", err)
	}

	// Create index on folder_stats for faster depth-based queries
	if _, err := db.ExecContext(
		ctx,
		`CREATE INDEX IF NOT EXISTS idx_folder_stats_depth ON folder_stats(depth)`,
	); err != nil {
		return fmt.Errorf("failed to create folder_stats index: %w", err)
	}

	// Check if FTS tables need upgrade to trigram or new columns
	upgradeFTS := func(tableName, expectedSqlPart string) error {
		var existingSQL string
		err := db.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).
			Scan(&existingSQL)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil // Table doesn't exist
			}
			return err
		}

		// Normalize whitespace for comparison
		if !strings.Contains(existingSQL, "trigram") ||
			(expectedSqlPart != "" && !strings.Contains(existingSQL, expectedSqlPart)) {
			// Needs upgrade - drop it
			if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", tableName)); err != nil {
				return fmt.Errorf("failed to drop %s for upgrade: %w", tableName, err)
			}

			// Recreate immediately
			var createSQL string
			switch tableName {
			case "media_fts":
				createSQL = `CREATE VIRTUAL TABLE media_fts USING fts5(
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
			case "captions_fts":
				createSQL = `CREATE VIRTUAL TABLE captions_fts USING fts5(
					media_path UNINDEXED,
					text,
					content='captions',
					tokenize = 'trigram',
					detail = 'full'
				);`
			}

			if _, err := db.ExecContext(ctx, createSQL); err != nil {
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
					if _, err := db.ExecContext(ctx, tsql); err != nil {
						return fmt.Errorf("failed to recreate trigger: %w", err)
					}
				}
			}

			// Rebuild data
			switch tableName {
			case "media_fts":
				if _, err := db.ExecContext(
					ctx,
					"INSERT INTO media_fts(rowid, path, path_tokenized, title, description, time_deleted) SELECT rowid, path, path_tokenized, title, description, time_deleted FROM media",
				); err != nil {
					return fmt.Errorf("failed to rebuild media_fts: %w", err)
				}
			case "captions_fts":
				if _, err := db.ExecContext(
					ctx,
					"INSERT INTO captions_fts(rowid, media_path, text) SELECT rowid, media_path, text FROM captions",
				); err != nil {
					return fmt.Errorf("failed to rebuild captions_fts: %w", err)
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

func migrateIndexes(ctx context.Context, db *sql.DB) error {
	indexes := []string{
		// Core indexes
		"CREATE INDEX IF NOT EXISTS idx_path ON media(path)",
		"CREATE INDEX IF NOT EXISTS idx_media_type ON media(media_type)",
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
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_type ON media(time_deleted, media_type)",
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_size ON media(time_deleted, size)",
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_duration ON media(time_deleted, duration)",
		"CREATE INDEX IF NOT EXISTS idx_media_deleted_path ON media(time_deleted, path)",
		// Partial index for active media (most common query pattern)
		"CREATE INDEX IF NOT EXISTS idx_media_active ON media(path, media_type) WHERE time_deleted = 0",
		// Indexes for filter bins calculation (optimize include_counts)
		"CREATE INDEX IF NOT EXISTS idx_media_active_size ON media(size) WHERE time_deleted = 0 AND size > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_duration ON media(duration) WHERE time_deleted = 0 AND duration > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_time_modified ON media(time_modified) WHERE time_deleted = 0 AND time_modified > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_time_created ON media(time_created) WHERE time_deleted = 0 AND time_created > 0",
		"CREATE INDEX IF NOT EXISTS idx_media_active_time_downloaded ON media(time_downloaded) WHERE time_deleted = 0 AND time_downloaded > 0",
	}

	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Remove unused function-based index (SQLite can't use it efficiently)
	if _, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_path_prefix"); err != nil {
		return fmt.Errorf("failed to drop idx_path_prefix: %w", err)
	}

	// Populate folder_stats materialized view
	if err := PopulateFolderStatsInGo(ctx, db); err != nil {
		return fmt.Errorf("failed to populate folder_stats: %w", err)
	}

	return nil
}
