package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/chapmanjacobd/discoteca/internal/db/schema"
)

// minimumSQLiteVersion is required for STRICT tables, which the canonical
// schema and all migrations use unconditionally.
const minimumSQLiteVersion = "3.37.0"

func requireStrictSupport(ctx context.Context, db dbExecer) error {
	var version string
	if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err != nil {
		return err
	}
	if !isVersionGreaterOrEqual(version, minimumSQLiteVersion) {
		return fmt.Errorf(
			"discoteca requires SQLite %s or newer (found %s); upgrade SQLite or the sqlite driver",
			minimumSQLiteVersion,
			version,
		)
	}
	return nil
}

func renameMediaTypeColumn(ctx context.Context, db dbExecer) error {
	hasType, err := columnExists(ctx, db, "media", "type")
	if err != nil {
		return err
	}
	hasMediaType, err := columnExists(ctx, db, "media", "media_type")
	if err != nil {
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

// Migrate runs schema migrations on an existing database.
//
// The whole migration runs on a single connection with foreign key enforcement
// disabled. That prevents DROP TABLE media from cascading into history,
// captions, and playlist_items while the table is recreated.
func Migrate(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := requireStrictSupport(ctx, conn); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys for migration: %w", err)
	}
	reEnableForeignKeys := func() error {
		if _, err := conn.ExecContext(context.WithoutCancel(ctx), "PRAGMA foreign_keys = ON"); err != nil {
			return fmt.Errorf("failed to re-enable foreign keys after migration: %w", err)
		}
		return nil
	}
	defer func() { _ = reEnableForeignKeys() }()

	if err := migrate(ctx, conn); err != nil {
		return err
	}

	if err := checkForeignKeys(ctx, conn); err != nil {
		return err
	}

	return reEnableForeignKeys()
}

func migrate(ctx context.Context, db dbExecer) error {
	// Rename before adding columns so both old and new names are handled.
	if err := renameMediaTypeColumn(ctx, db); err != nil {
		return err
	}

	// Add missing columns first so every later step can rely on the canonical
	// column set.
	if err := migrateColumns(ctx, db); err != nil {
		return err
	}

	// Convert legacy media_id references while rowids still point at the
	// original media rows. Recreating media can reassign rowids.
	if err := convertColumnsBeforeStrict(ctx, db); err != nil {
		return fmt.Errorf("failed to convert columns: %w", err)
	}

	// Consolidate metadata and recreate media (dropping dead columns and/or
	// applying STRICT). Foreign keys are disabled for the whole migration.
	if err := cleanupMediaTable(ctx, db); err != nil {
		return err
	}

	// Rows that point at missing parents cannot satisfy foreign_key_check.
	// Preserve them in a quarantine table instead of silently deleting them.
	if err := quarantineOrphanPlaylistItems(ctx, db); err != nil {
		return err
	}

	if err := migrateTables(ctx, db); err != nil {
		return err
	}

	return migrateIndexes(ctx, db)
}

func checkForeignKeys(ctx context.Context, db dbExecer) error {
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("failed to run foreign_key_check: %w", err)
	}
	defer rows.Close()

	var violations []string
	for rows.Next() {
		var table string
		var rowid int64
		var parent string
		var fkid int
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			return err
		}
		if len(violations) < 10 {
			violations = append(violations, fmt.Sprintf("%s rowid=%d -> %s (fk %d)", table, rowid, parent, fkid))
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(violations) > 0 {
		return fmt.Errorf("foreign key violations remain after migration: %s", strings.Join(violations, "; "))
	}
	return nil
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

func isTableStrict(ctx context.Context, db dbExecer, tableName string) bool {
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

func tableExists(ctx context.Context, db dbExecer, tableName string) (bool, error) {
	var count int
	err := db.QueryRowContext(
		ctx,
		"SELECT count(*) FROM sqlite_master WHERE type IN ('table', 'view') AND name = ?",
		tableName,
	).Scan(&count)
	return count > 0, err
}

func tableColumns(ctx context.Context, db dbQueryer, tableName string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

func migrateToStrict(ctx context.Context, db dbExecer, tableName, createSQL string) error {
	strict := isTableStrict(ctx, db, tableName)
	if strict {
		return nil
	}

	// Rebuild into a temporary table instead of renaming the old table first.
	// Renaming the old table makes SQLite rewrite foreign key references in
	// other tables to point at the temporary name.
	tempTable := tableName + "_new_strict"
	tempCreateSQL := strings.Replace(createSQL, "CREATE TABLE "+tableName, "CREATE TABLE "+tempTable, 1)
	if tempCreateSQL == createSQL {
		return fmt.Errorf("could not build temporary create statement for %s", tableName)
	}

	// Foreign keys are disabled for the whole migration on this connection.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, tempCreateSQL); err != nil {
		return fmt.Errorf("failed to create temporary strict %s: %w", tableName, err)
	}

	// Copy only columns that exist in both tables, plus rowid so FTS content
	// mappings remain stable.
	oldColumns, err := tableColumns(ctx, tx, tableName)
	if err != nil {
		return err
	}
	newColumns, err := tableColumns(ctx, tx, tempTable)
	if err != nil {
		return err
	}
	var columns []string
	for _, column := range oldColumns {
		if slices.Contains(newColumns, column) {
			columns = append(columns, column)
		}
	}
	if len(columns) == 0 {
		return fmt.Errorf("no shared columns while migrating %s to STRICT", tableName)
	}

	columnList := "rowid, " + strings.Join(columns, ", ")
	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s", tempTable, columnList, columnList, tableName),
	); err != nil {
		return fmt.Errorf("failed to copy data for %s: %w", tableName, err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", tableName)); err != nil {
		return fmt.Errorf("failed to drop old %s: %w", tableName, err)
	}
	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tempTable, tableName),
	); err != nil {
		return fmt.Errorf("failed to rename temporary %s: %w", tableName, err)
	}

	return tx.Commit()
}

// convertColumnsBeforeStrict handles column type conversions before STRICT
// migration, e.g. history.media_id (INTEGER) -> history.media_path (TEXT).
func convertColumnsBeforeStrict(ctx context.Context, db dbExecer) error {
	if err := convertHistoryMediaID(ctx, db); err != nil {
		return fmt.Errorf("failed to convert history.media_id: %w", err)
	}
	if err := convertCaptionsMediaID(ctx, db); err != nil {
		return fmt.Errorf("failed to convert captions.media_id: %w", err)
	}
	return nil
}

func mediaIDJoinKey(ctx context.Context, db dbExecer) string {
	columns, err := tableColumns(ctx, db, "media")
	if err == nil && slices.Contains(columns, "id") {
		return "m.id"
	}
	return "m.rowid"
}

func convertHistoryMediaID(ctx context.Context, db dbExecer) error {
	columns, err := tableColumns(ctx, db, "history")
	if err != nil {
		return err
	}
	if !slices.Contains(columns, "media_id") {
		return nil
	}

	joinKey := mediaIDJoinKey(ctx, db)

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS history_orphaned (
		id INTEGER,
		media_id INTEGER NOT NULL,
		time_played INTEGER,
		playhead INTEGER,
		done INTEGER,
		quarantined_at INTEGER DEFAULT (unixepoch())
	)`); err != nil {
		return fmt.Errorf("failed to create history_orphaned: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

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

	// Preserve rows that cannot be resolved to a media path in a quarantine
	// table instead of silently mapping them to an unrelated path.
	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO history_orphaned (id, media_id, time_played, playhead, done)
			SELECT h.id, h.media_id, h.time_played, h.playhead, h.done
			FROM history h
			LEFT JOIN media m ON h.media_id = %s
			WHERE m.path IS NULL`, joinKey),
	); err != nil {
		return fmt.Errorf("failed to quarantine orphaned history rows: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO history_tmp (id, media_path, time_played, playhead, done)
			SELECT h.id, m.path, h.time_played, h.playhead, h.done
			FROM history h
			JOIN media m ON h.media_id = %s`, joinKey),
	); err != nil {
		return fmt.Errorf("failed to convert history media_id to media_path: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DROP TABLE history"); err != nil {
		return fmt.Errorf("failed to drop old history: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE history_tmp RENAME TO history"); err != nil {
		return fmt.Errorf("failed to rename history_tmp: %w", err)
	}

	return tx.Commit()
}

func convertCaptionsMediaID(ctx context.Context, db dbExecer) error {
	columns, err := tableColumns(ctx, db, "captions")
	if err != nil {
		return err
	}
	if !slices.Contains(columns, "media_id") {
		return nil
	}

	joinKey := mediaIDJoinKey(ctx, db)

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS captions_orphaned (
		media_id INTEGER NOT NULL,
		time REAL,
		text TEXT,
		quarantined_at INTEGER DEFAULT (unixepoch())
	)`); err != nil {
		return fmt.Errorf("failed to create captions_orphaned: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `CREATE TABLE captions_tmp (
		media_path TEXT NOT NULL,
		time REAL,
		text TEXT,
		FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("failed to create captions_tmp: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO captions_orphaned (media_id, time, text)
			SELECT c.media_id, c.time, c.text
			FROM captions c
			LEFT JOIN media m ON c.media_id = %s
			WHERE m.path IS NULL`, joinKey),
	); err != nil {
		return fmt.Errorf("failed to quarantine orphaned caption rows: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO captions_tmp (media_path, time, text)
			SELECT m.path, c.time, c.text
			FROM captions c
			JOIN media m ON c.media_id = %s`, joinKey),
	); err != nil {
		return fmt.Errorf("failed to convert captions media_id to media_path: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DROP TABLE captions"); err != nil {
		return fmt.Errorf("failed to drop old captions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE captions_tmp RENAME TO captions"); err != nil {
		return fmt.Errorf("failed to rename captions_tmp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Dropping captions also drops its FTS triggers. Recreate and rebuild now
	// so captions_fts never points at stale rowids. An incompatible table is
	// left for upgradeFTSTable, which can recreate it safely.
	if IsFtsEnabled() {
		matches, err := ftsTableMatches(ctx, db, "captions_fts")
		if err != nil {
			return err
		}
		if matches {
			if err := ensureCaptionsFTSTriggers(ctx, db); err != nil {
				return err
			}
			if err := rebuildFTSData(ctx, db, "captions_fts"); err != nil {
				return err
			}
		}
	}
	return nil
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

func migrateColumns(ctx context.Context, db dbExecer) error {
	cols := []columnDef{
		{"playlists", "title", "TEXT"},
		{"playlists", "time_created", "INTEGER"},
		{"playlists", "time_modified", "INTEGER"},
		{"playlists", "hours_update_delay", "INTEGER"},
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
		if err := addColumnIfNeeded(ctx, db, c); err != nil {
			return err
		}
	}
	return nil
}

type columnDef struct {
	table  string
	column string
	schema string
}

func addColumnIfNeeded(ctx context.Context, db dbExecer, c columnDef) error {
	exists, err := columnExists(ctx, db, c.table, c.column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return addColumn(ctx, db, c)
}

func columnExists(ctx context.Context, db dbExecer, table, column string) (bool, error) {
	columns, err := tableColumns(ctx, db, table)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return true, nil // Treat missing table as column existing (it will be created later)
		}
		return false, err
	}
	if len(columns) == 0 {
		return true, nil // Missing table
	}
	return slices.Contains(columns, column), nil
}

func addColumn(ctx context.Context, db dbExecer, c columnDef) error {
	if _, err := db.ExecContext(
		ctx,
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.table, c.column, c.schema),
	); err != nil {
		if !strings.Contains(err.Error(), "no such table") {
			return fmt.Errorf("failed to add column %s to table %s: %w", c.column, c.table, err)
		}
		return nil
	}

	if c.table == "media" && c.column == "path_tokenized" {
		if err := populatePathTokenized(ctx, db); err != nil {
			return fmt.Errorf("failed to populate path_tokenized: %w", err)
		}
	}
	return nil
}

func cleanupMediaTable(ctx context.Context, db dbExecer) error {
	hasDead, err := needsCleanup(ctx, db)
	if err != nil || !hasDead {
		return err
	}

	if err := consolidateMetadata(ctx, db); err != nil {
		return err
	}

	return recreateMediaTable(ctx, db)
}

func needsCleanup(ctx context.Context, db dbExecer) (bool, error) {
	columns, err := tableColumns(ctx, db, "media")
	if err != nil {
		return false, err
	}

	deadColumns := map[string]bool{
		"upvote_ratio": true, "num_comments": true, "favorite_count": true,
		"view_count": true, "time_uploaded": true, "uploader": true,
		"webpath": true, "city": true, "country": true,
		"latitude": true, "longitude": true, "decade": true,
		"mood": true, "bpm": true, "key": true, "extension": true,
	}
	for _, column := range columns {
		if deadColumns[strings.ToLower(column)] {
			return true, nil
		}
	}

	return !isTableStrict(ctx, db, "media"), nil
}

func consolidateMetadata(ctx context.Context, db dbExecer) error {
	columns, err := tableColumns(ctx, db, "media")
	if err != nil {
		return err
	}

	var valueParts []string
	var whereParts []string
	if slices.Contains(columns, "decade") {
		valueParts = append(valueParts, `CASE WHEN decade IS NOT NULL AND decade != '' `+
			`THEN '\nDate: ' || decade ELSE '' END`)
		whereParts = append(whereParts, "decade IS NOT NULL")
	}
	if slices.Contains(columns, "mood") {
		valueParts = append(valueParts, `CASE WHEN mood IS NOT NULL AND mood != '' `+
			`THEN '\nMood: ' || mood ELSE '' END`)
		whereParts = append(whereParts, "mood IS NOT NULL")
	}
	if slices.Contains(columns, "bpm") {
		valueParts = append(valueParts, `CASE WHEN bpm IS NOT NULL AND bpm != 0 `+
			`THEN '\nBPM: ' || bpm ELSE '' END`)
		whereParts = append(whereParts, "bpm IS NOT NULL")
	}
	if slices.Contains(columns, "key") {
		valueParts = append(valueParts, `CASE WHEN "key" IS NOT NULL AND "key" != '' `+
			`THEN '\nKey: ' || "key" ELSE '' END`)
		whereParts = append(whereParts, `"key" IS NOT NULL`)
	}

	if len(valueParts) == 0 {
		return nil
	}

	_, err = db.ExecContext(ctx, fmt.Sprintf(
		`UPDATE media SET description = COALESCE(description, '') || %s WHERE %s`,
		strings.Join(valueParts, " || "),
		strings.Join(whereParts, " OR "),
	))
	return err
}

// canonicalMediaColumns is the production column order, excluding id.
func canonicalMediaColumns(ftsEnabled bool) []string {
	columns := []string{"path"}
	if ftsEnabled {
		columns = append(columns, "path_tokenized")
	}
	return append(columns,
		"title",
		"duration",
		"size",
		"time_created",
		"time_modified",
		"time_deleted",
		"time_first_played",
		"time_last_played",
		"play_count",
		"playhead",
		"media_type",
		"width",
		"height",
		"fps",
		"video_codecs",
		"audio_codecs",
		"subtitle_codecs",
		"video_count",
		"audio_count",
		"subtitle_count",
		"album",
		"artist",
		"genre",
		"categories",
		"description",
		"language",
		"time_downloaded",
		"score",
		"fasthash",
		"sha256",
		"is_deduped",
	)
}

func mediaColumnDefinitions(ftsEnabled bool) map[string]string {
	definitions := map[string]string{
		"path":              "TEXT NOT NULL UNIQUE",
		"path_tokenized":    "TEXT",
		"title":             "TEXT",
		"duration":          "INTEGER",
		"size":              "INTEGER",
		"time_created":      "INTEGER",
		"time_modified":     "INTEGER",
		"time_deleted":      "INTEGER DEFAULT 0",
		"time_first_played": "INTEGER DEFAULT 0",
		"time_last_played":  "INTEGER DEFAULT 0",
		"play_count":        "INTEGER DEFAULT 0",
		"playhead":          "INTEGER DEFAULT 0",
		"media_type":        "TEXT",
		"width":             "INTEGER",
		"height":            "INTEGER",
		"fps":               "REAL",
		"video_codecs":      "TEXT",
		"audio_codecs":      "TEXT",
		"subtitle_codecs":   "TEXT",
		"video_count":       "INTEGER DEFAULT 0",
		"audio_count":       "INTEGER DEFAULT 0",
		"subtitle_count":    "INTEGER DEFAULT 0",
		"album":             "TEXT",
		"artist":            "TEXT",
		"genre":             "TEXT",
		"categories":        "TEXT",
		"description":       "TEXT",
		"language":          "TEXT",
		"time_downloaded":   "INTEGER",
		"score":             "REAL",
		"fasthash":          "TEXT",
		"sha256":            "TEXT",
		"is_deduped":        "INTEGER DEFAULT 0",
	}
	if !ftsEnabled {
		delete(definitions, "path_tokenized")
	}
	return definitions
}

func recreateMediaTable(ctx context.Context, db dbExecer) error {
	oldColumns, err := tableColumns(ctx, db, "media")
	if err != nil {
		return err
	}

	ftsEnabled := IsFtsEnabled()
	columns := canonicalMediaColumns(ftsEnabled)
	definitions := mediaColumnDefinitions(ftsEnabled)

	var definitionsSQL strings.Builder
	definitionsSQL.WriteString("id INTEGER PRIMARY KEY AUTOINCREMENT")
	for _, column := range columns {
		definitionsSQL.WriteString(",\n")
		definitionsSQL.WriteString(column)
		definitionsSQL.WriteString(" ")
		definitionsSQL.WriteString(definitions[column])
	}

	// Preserve ids when the old table has them so media rowids and FTS
	// content_rowid mappings stay stable.
	copyColumns := make([]string, 0, len(columns)+1)
	if slices.Contains(oldColumns, "id") {
		copyColumns = append(copyColumns, "id")
	}
	for _, column := range columns {
		if slices.Contains(oldColumns, column) {
			copyColumns = append(copyColumns, column)
		}
	}
	if !slices.Contains(copyColumns, "path") {
		return errors.New("cannot recreate media: old table has no path column")
	}
	columnList := strings.Join(copyColumns, ", ")

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	statements := []string{
		fmt.Sprintf("CREATE TABLE media_dg_tmp (\n%s\n) STRICT", definitionsSQL.String()),
		fmt.Sprintf(
			"INSERT INTO media_dg_tmp (%s) SELECT %s FROM media",
			columnList,
			columnList,
		),
		"DROP TABLE media",
		"ALTER TABLE media_dg_tmp RENAME TO media",
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("failed to recreate media table: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if ftsEnabled {
		if err := refreshFTSAfterMediaRecreation(ctx, db); err != nil {
			return err
		}
	}
	return nil
}

func refreshFTSAfterMediaRecreation(ctx context.Context, db dbExecer) error {
	mediaFTSMatches, err := ftsTableMatches(ctx, db, "media_fts")
	if err != nil {
		return err
	}
	if mediaFTSMatches {
		if err := ensureMediaFTSTriggers(ctx, db); err != nil {
			return err
		}
		if err := rebuildFTSData(ctx, db, "media_fts"); err != nil {
			return err
		}
	}

	captionsFTSMatches, err := ftsTableMatches(ctx, db, "captions_fts")
	if err != nil {
		return err
	}
	if captionsFTSMatches {
		if err := ensureCaptionsFTSTriggers(ctx, db); err != nil {
			return err
		}
		if err := rebuildFTSData(ctx, db, "captions_fts"); err != nil {
			return err
		}
	}
	return nil
}

func populatePathTokenized(ctx context.Context, db dbExecer) error {
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

func migrateTables(ctx context.Context, db dbExecer) error {
	if err := migrateStrictTables(ctx, db); err != nil {
		return err
	}

	if err := createMaintenanceTables(ctx, db); err != nil {
		return err
	}

	if IsFtsEnabled() {
		for _, tableName := range []string{"media_fts", "captions_fts"} {
			if err := upgradeFTSTable(ctx, db, tableName); err != nil {
				return err
			}
		}

		if exists, err := tableExists(ctx, db, "media_fts"); err != nil {
			return err
		} else if exists {
			if err := ensureMediaFTSTriggers(ctx, db); err != nil {
				return err
			}
		}
		if exists, err := tableExists(ctx, db, "captions_fts"); err != nil {
			return err
		} else if exists {
			if err := ensureCaptionsFTSTriggers(ctx, db); err != nil {
				return err
			}
		}
	}

	return nil
}

type tableMigration struct {
	name string
	sql  string
}

func strictTableMigrations() []tableMigration {
	return []tableMigration{
		{"playlists", `CREATE TABLE playlists (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                path TEXT UNIQUE,
                title TEXT,
                extractor_key TEXT,
                extractor_config TEXT,
                time_created INTEGER,
                time_modified INTEGER,
                hours_update_delay INTEGER,
                time_deleted INTEGER DEFAULT 0
            ) STRICT`},
		{"playlist_items", `CREATE TABLE playlist_items (
                playlist_id INTEGER NOT NULL,
                media_path TEXT NOT NULL,
                track_number INTEGER,
                time_added INTEGER DEFAULT (unixepoch()),
                PRIMARY KEY (playlist_id, media_path),
                FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) STRICT`},
		{"captions", `CREATE TABLE captions (
                media_path TEXT NOT NULL,
                time REAL,
                text TEXT,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) STRICT`},
		{"history", `CREATE TABLE history (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                media_path TEXT NOT NULL,
                time_played INTEGER DEFAULT (unixepoch()),
                playhead INTEGER,
                done INTEGER,
                FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
            ) STRICT`},
		{"custom_keywords", `CREATE TABLE custom_keywords (
                category TEXT NOT NULL,
                keyword TEXT NOT NULL,
                PRIMARY KEY (category, keyword)
            ) STRICT`},
	}
}

func migrateStrictTables(ctx context.Context, db dbExecer) error {
	for _, m := range strictTableMigrations() {
		if !IsFtsEnabled() && m.name == "captions" {
			continue
		}
		if err := migrateOrCreateTable(ctx, db, m.name, m.sql); err != nil {
			return err
		}
	}
	return nil
}

func migrateOrCreateTable(ctx context.Context, db dbExecer, name, createSQL string) error {
	exists, err := tableExists(ctx, db, name)
	if err != nil {
		return err
	}
	if exists {
		if err := migrateToStrict(ctx, db, name, createSQL); err != nil {
			return fmt.Errorf("migrateToStrict failed for %s: %w", name, err)
		}
	} else {
		if _, err := db.ExecContext(ctx, createSQL); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
	}
	return nil
}

func dropIncompatibleFolderStatsTable(ctx context.Context, db dbExecer) error {
	columns, err := tableColumns(ctx, db, "folder_stats")
	if err != nil {
		return err
	}
	if len(columns) > 0 && len(columns) != 4 {
		if _, err := db.ExecContext(ctx, "DROP TABLE folder_stats"); err != nil {
			return fmt.Errorf("failed to drop incompatible folder_stats table: %w", err)
		}
	}
	return nil
}

func createMaintenanceTables(ctx context.Context, db dbExecer) error {
	if err := dropIncompatibleFolderStatsTable(ctx, db); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS folder_stats (
		parent TEXT PRIMARY KEY,
		file_count INTEGER,
		total_size INTEGER,
		total_duration INTEGER
	)`); err != nil {
		return fmt.Errorf("failed to create folder_stats table: %w", err)
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS folder_stats_meta (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		dirty INTEGER NOT NULL DEFAULT 1
	)`); err != nil {
		return fmt.Errorf("failed to create folder_stats_meta table: %w", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO folder_stats_meta (id, dirty) VALUES (1, 1)`); err != nil {
		return fmt.Errorf("failed to insert initial folder_stats_meta row: %w", err)
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _maintenance_meta (
		key TEXT PRIMARY KEY,
		value TEXT,
		last_updated INTEGER
	)`); err != nil {
		return fmt.Errorf("failed to create _maintenance_meta table: %w", err)
	}

	if err := initMaintenanceMeta(ctx, db, "folder_stats_last_refresh"); err != nil {
		return err
	}
	if err := initMaintenanceMeta(ctx, db, "fts_last_rebuild"); err != nil {
		return err
	}

	for _, triggerSQL := range []string{
		`CREATE TRIGGER IF NOT EXISTS folder_stats_media_insert
		AFTER INSERT ON media
		BEGIN
			UPDATE folder_stats_meta SET dirty = 1 WHERE id = 1;
		END;`,
		`CREATE TRIGGER IF NOT EXISTS folder_stats_media_delete
		AFTER DELETE ON media
		BEGIN
			UPDATE folder_stats_meta SET dirty = 1 WHERE id = 1;
		END;`,
		`CREATE TRIGGER IF NOT EXISTS folder_stats_media_update
		AFTER UPDATE OF path, size, duration, time_deleted ON media
		BEGIN
			UPDATE folder_stats_meta SET dirty = 1 WHERE id = 1;
		END;`,
	} {
		if _, err := db.ExecContext(ctx, triggerSQL); err != nil {
			return fmt.Errorf("failed to create folder_stats trigger: %w", err)
		}
	}

	return nil
}

func initMaintenanceMeta(ctx context.Context, db dbExecer, key string) error {
	if _, err := db.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO _maintenance_meta (key, value, last_updated) VALUES (?, '0', 0)`,
		key,
	); err != nil {
		return fmt.Errorf("failed to initialize maintenance metadata: %w", err)
	}
	return nil
}

func normalizeSQL(statement string) string {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == ';' {
			return -1
		}
		return unicode.ToLower(r)
	}, statement)
	return strings.ReplaceAll(normalized, "ifnotexists", "")
}

func ftsTableMatches(ctx context.Context, db dbExecer, tableName string) (bool, error) {
	exists, err := tableExists(ctx, db, tableName)
	if err != nil || !exists {
		return false, err
	}

	columns, err := tableColumns(ctx, db, tableName)
	if err != nil {
		return false, err
	}

	var expectedColumns, expectedSQLParts []string
	switch tableName {
	case "media_fts":
		expectedColumns = []string{"path", "path_tokenized", "title", "description", "time_deleted"}
		expectedSQLParts = []string{"content='media'", "content_rowid='rowid'"}
	case "captions_fts":
		expectedColumns = []string{"media_path", "text"}
		expectedSQLParts = []string{"content='captions'"}
	default:
		return false, fmt.Errorf("unknown FTS table %s", tableName)
	}

	if !slices.Equal(columns, expectedColumns) {
		return false, nil
	}

	existingSQL, err := getTableSQL(ctx, db, tableName)
	if err != nil {
		return false, err
	}
	normalized := normalizeSQL(existingSQL)
	if !strings.Contains(normalized, "usingfts5(") {
		return false, nil
	}
	expectedSQLParts = append(expectedSQLParts, "tokenize='trigram'", "detail='full'")
	for _, part := range expectedSQLParts {
		if !strings.Contains(normalized, normalizeSQL(part)) {
			return false, nil
		}
	}
	return true, nil
}

func upgradeFTSTable(ctx context.Context, db dbExecer, tableName string) error {
	matches, err := ftsTableMatches(ctx, db, tableName)
	if err != nil {
		return err
	}
	if matches {
		return nil
	}

	existingSQL, err := getTableSQL(ctx, db, tableName)
	if err != nil {
		return err
	}
	if existingSQL == "" {
		return nil
	}

	return rebuildFTSTable(ctx, db, tableName)
}

func getTableSQL(ctx context.Context, db dbExecer, tableName string) (string, error) {
	var existingSQL string
	err := db.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).
		Scan(&existingSQL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return existingSQL, nil
}

func getTriggerSQL(ctx context.Context, db dbExecer, triggerName string) (string, error) {
	var triggerSQL string
	err := db.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='trigger' AND name=?", triggerName).
		Scan(&triggerSQL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return triggerSQL, nil
}

func rebuildFTSTable(ctx context.Context, db dbExecer, tableName string) error {
	createSQL := ftsCreateSQL(tableName)
	if createSQL == "" {
		return fmt.Errorf("no FTS schema defined for %s", tableName)
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", tableName)); err != nil {
		return fmt.Errorf("failed to drop %s for upgrade: %w", tableName, err)
	}

	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("failed to recreate %s: %w", tableName, err)
	}

	switch tableName {
	case "media_fts":
		if err := ensureMediaFTSTriggers(ctx, db); err != nil {
			return err
		}
	case "captions_fts":
		if err := ensureCaptionsFTSTriggers(ctx, db); err != nil {
			return err
		}
	}

	return rebuildFTSData(ctx, db, tableName)
}

func ftsCreateSQL(tableName string) string {
	switch tableName {
	case "media_fts":
		return `CREATE VIRTUAL TABLE media_fts USING fts5(
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
		return `CREATE VIRTUAL TABLE captions_fts USING fts5(
					media_path UNINDEXED,
					text,
					content='captions',
					tokenize = 'trigram',
					detail = 'full'
				);`
	}
	return ""
}

func ensureTrigger(ctx context.Context, db dbExecer, name, createSQL string) error {
	existingSQL, err := getTriggerSQL(ctx, db, name)
	if err != nil {
		return err
	}
	if existingSQL != "" && normalizeSQL(existingSQL) == normalizeSQL(createSQL) {
		return nil
	}
	if existingSQL != "" {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP TRIGGER %s", name)); err != nil {
			return fmt.Errorf("failed to drop outdated trigger %s: %w", name, err)
		}
	}
	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("failed to create trigger %s: %w", name, err)
	}
	return nil
}

func ensureMediaFTSTriggers(ctx context.Context, db dbExecer) error {
	return ensureTriggerSet(ctx, db, mediaFTSTriggerSQLs())
}

func ensureCaptionsFTSTriggers(ctx context.Context, db dbExecer) error {
	return ensureTriggerSet(ctx, db, captionsFTSTriggerSQLs())
}

func ensureTriggerSet(ctx context.Context, db dbExecer, triggers []triggerDef) error {
	for _, trigger := range triggers {
		if err := ensureTrigger(ctx, db, trigger.name, trigger.sql); err != nil {
			return err
		}
	}
	return nil
}

type triggerDef struct {
	name string
	sql  string
}

func mediaFTSTriggerSQLs() []triggerDef {
	return []triggerDef{
		{"media_ai", `CREATE TRIGGER IF NOT EXISTS media_ai AFTER INSERT ON media BEGIN
			INSERT INTO media_fts(rowid, path, path_tokenized, title, description, time_deleted)
			VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description, new.time_deleted);
		END;`},
		{"media_ad", `CREATE TRIGGER IF NOT EXISTS media_ad AFTER DELETE ON media BEGIN
			DELETE FROM media_fts WHERE rowid = old.rowid;
		END;`},
		{"media_au", `CREATE TRIGGER IF NOT EXISTS media_au AFTER UPDATE ON media BEGIN
			INSERT INTO media_fts(media_fts, rowid, path, path_tokenized, title, description, time_deleted) VALUES('delete', old.rowid, old.path, old.path_tokenized, old.title, old.description, old.time_deleted);
			INSERT INTO media_fts(rowid, path, path_tokenized, title, description, time_deleted) VALUES (new.rowid, new.path, new.path_tokenized, new.title, new.description, new.time_deleted);
		END;`},
	}
}

func captionsFTSTriggerSQLs() []triggerDef {
	return []triggerDef{
		{"captions_ai", `CREATE TRIGGER IF NOT EXISTS captions_ai AFTER INSERT ON captions BEGIN
			INSERT INTO captions_fts(rowid, media_path, text)
			VALUES (new.rowid, new.media_path, new.text);
		END;`},
		{"captions_ad", `CREATE TRIGGER IF NOT EXISTS captions_ad AFTER DELETE ON captions BEGIN
			DELETE FROM captions_fts WHERE rowid = old.rowid;
		END;`},
		{"captions_au", `CREATE TRIGGER IF NOT EXISTS captions_au AFTER UPDATE ON captions BEGIN
			INSERT INTO captions_fts(captions_fts, rowid, media_path, text)
			VALUES('delete', old.rowid, old.media_path, old.text);
			INSERT INTO captions_fts(rowid, media_path, text)
			VALUES (new.rowid, new.media_path, new.text);
		END;`},
	}
}

func rebuildFTSData(ctx context.Context, db dbExecer, tableName string) error {
	exists, err := tableExists(ctx, db, tableName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := db.ExecContext(
		ctx,
		fmt.Sprintf("INSERT INTO %s(%s) VALUES('rebuild')", tableName, tableName),
	); err != nil {
		return fmt.Errorf("failed to rebuild %s: %w", tableName, err)
	}
	return nil
}

func migrateIndexes(ctx context.Context, db dbExecer) error {
	// Use the canonical schema so fresh and migrated databases get exactly the
	// same index set.
	if _, err := db.ExecContext(ctx, schema.GetCoreIndexes()); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Remove legacy redundant indexes. The UNIQUE constraint on media.path
	// already provides the idx_path lookup, and idx_type was renamed to
	// idx_media_type.
	for _, indexName := range []string{"idx_path", "idx_type"} {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)); err != nil {
			return fmt.Errorf("failed to drop redundant index %s: %w", indexName, err)
		}
	}

	// The maintenance scan is no longer part of every migration. Populate only
	// when the materialized view has never been filled.
	empty, err := folderStatsEmpty(ctx, db)
	if err != nil {
		return err
	}
	if empty {
		if err := populateFolderStats(ctx, db); err != nil {
			return fmt.Errorf("failed to populate folder_stats: %w", err)
		}
		if err := markFolderStatsRefreshed(ctx, db); err != nil {
			return err
		}
	}

	return nil
}

// quarantineOrphanPlaylistItems moves playlist_items rows whose media or
// playlist is missing into playlist_items_orphaned. This keeps
// PRAGMA foreign_key_check clean without silently discarding the rows.
func quarantineOrphanPlaylistItems(ctx context.Context, db dbExecer) error {
	exists, err := tableExists(ctx, db, "playlist_items")
	if err != nil || !exists {
		return err
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS playlist_items_orphaned (
		playlist_id INTEGER NOT NULL,
		media_path TEXT NOT NULL,
		track_number INTEGER,
		time_added INTEGER,
		reason TEXT NOT NULL,
		quarantined_at INTEGER DEFAULT (unixepoch())
	)`); err != nil {
		return fmt.Errorf("failed to create playlist_items_orphaned: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO playlist_items_orphaned (playlist_id, media_path, track_number, time_added, reason)
		SELECT pi.playlist_id, pi.media_path, pi.track_number, pi.time_added,
			CASE
				WHEN NOT EXISTS (SELECT 1 FROM media m WHERE m.path = pi.media_path) THEN 'missing media'
				ELSE 'missing playlist'
			END
		FROM playlist_items pi
		WHERE NOT EXISTS (SELECT 1 FROM media m WHERE m.path = pi.media_path)
		   OR NOT EXISTS (SELECT 1 FROM playlists p WHERE p.id = pi.playlist_id)
	`); err != nil {
		return fmt.Errorf("failed to quarantine orphaned playlist_items: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM playlist_items
		WHERE NOT EXISTS (SELECT 1 FROM media m WHERE m.path = playlist_items.media_path)
		   OR NOT EXISTS (SELECT 1 FROM playlists p WHERE p.id = playlist_items.playlist_id)
	`); err != nil {
		return fmt.Errorf("failed to remove orphaned playlist_items: %w", err)
	}

	return tx.Commit()
}
