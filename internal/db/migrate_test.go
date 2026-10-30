package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

type legacyFixture struct {
	path string
	db   *sql.DB
}

func newLegacyFixture(t *testing.T) *legacyFixture {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=30000")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return &legacyFixture{path: dbPath, db: sqlDB}
}

func (f *legacyFixture) exec(t *testing.T, statement string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(statement, args...); err != nil {
		t.Fatalf("exec failed: %v\n%s", err, statement)
	}
}

func (f *legacyFixture) queryInt(t *testing.T, query string, args ...any) int {
	t.Helper()
	var value int
	if err := f.db.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("query failed: %v\n%s", err, query)
	}
	return value
}

func populateLegacySchema(t *testing.T, f *legacyFixture) {
	t.Helper()

	f.exec(t, `
CREATE TABLE media (
    path TEXT PRIMARY KEY,
    title TEXT,
    duration INTEGER,
    size INTEGER,
    time_created INTEGER,
    time_modified INTEGER,
    time_deleted INTEGER DEFAULT 0,
    type TEXT,
    uploader TEXT,
    webpath TEXT,
    decade TEXT,
    mood TEXT,
    bpm INTEGER,
    key TEXT
)`)

	f.exec(t, `
CREATE TABLE history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_id INTEGER,
    time_played INTEGER,
    playhead INTEGER,
    done INTEGER
)`)

	f.exec(t, `
CREATE TABLE captions (
    media_id INTEGER,
    time REAL,
    text TEXT
)`)

	f.exec(t, `
CREATE TABLE playlists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE,
    title TEXT,
    extractor_key TEXT,
    extractor_config TEXT,
    time_deleted INTEGER DEFAULT 0
)`)

	f.exec(t, `
CREATE TABLE playlist_items (
    playlist_id INTEGER NOT NULL,
    media_path TEXT NOT NULL,
    track_number INTEGER,
    time_added INTEGER,
    PRIMARY KEY (playlist_id, media_path),
    FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
)`)

	// Non-contiguous rowids: the legacy history.media_id values refer to these
	// rowids, not to a sequential id column.
	f.exec(t, `INSERT INTO media (rowid, path, title, duration, type, uploader, webpath, decade, mood, bpm, key)
		VALUES (10, '/media/a.mkv', 'Alpha', 120, 'video', 'user', '/web/a', '1990', 'calm', 120, 'C')`)
	f.exec(t, `INSERT INTO media (rowid, path, title, duration, type, decade)
		VALUES (50, '/media/b.mp3', 'Beta', 30, 'audio', '2000')`)
	f.exec(t, `INSERT INTO media (rowid, path, title, duration, type)
		VALUES (100, '/media/c.jpg', 'Gamma', 0, 'image')`)

	f.exec(t, `INSERT INTO history (id, media_id, time_played, playhead, done) VALUES (1, 50, 1000, 15, 0)`)
	f.exec(t, `INSERT INTO history (id, media_id, time_played, playhead, done) VALUES (2, 999, 2000, 3, 0)`)

	f.exec(t, `INSERT INTO captions (media_id, time, text) VALUES (50, 1.5, 'hello world')`)
	f.exec(t, `INSERT INTO captions (media_id, time, text) VALUES (999, 2.5, 'orphan caption')`)

	f.exec(t, `INSERT INTO playlists (id, path, title) VALUES (1, '/playlists/one', 'One')`)
	f.exec(t, `INSERT INTO playlist_items (playlist_id, media_path, track_number, time_added)
		VALUES (1, '/media/a.mkv', 1, 1000)`)
	f.exec(t, `INSERT INTO playlist_items (playlist_id, media_path, track_number, time_added)
		VALUES (1, '/media/b.mp3', 2, 1000)`)
	f.exec(t, `INSERT INTO playlist_items (playlist_id, media_path, track_number, time_added)
		VALUES (1, '/media/missing.mkv', 3, 1000)`)
}

func TestMigratePreservesChildRowsAndConvertsMediaID(t *testing.T) {
	f := newLegacyFixture(t)
	populateLegacySchema(t, f)

	ctx := context.Background()
	if err := db.Migrate(ctx, f.db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if got := f.queryInt(t, "SELECT count(*) FROM media"); got != 3 {
		t.Errorf("media rows = %d, want 3", got)
	}
	if got := f.queryInt(t, "SELECT count(*) FROM history"); got != 1 {
		t.Errorf("history rows = %d, want 1 (orphan quarantined)", got)
	}
	if got := f.queryInt(t, "SELECT count(*) FROM captions"); got != 1 {
		t.Errorf("captions rows = %d, want 1 (orphan quarantined)", got)
	}
	if got := f.queryInt(t, "SELECT count(*) FROM playlist_items"); got != 2 {
		t.Errorf("playlist_items rows = %d, want 2; media recreation must not cascade", got)
	}
	if got := f.queryInt(t,
		"SELECT count(*) FROM playlist_items_orphaned WHERE media_path = '/media/missing.mkv'",
	); got != 1 {
		t.Errorf("playlist_items_orphaned rows = %d, want 1", got)
	}

	var historyPath string
	if err := f.db.QueryRow("SELECT media_path FROM history WHERE id = 1").Scan(&historyPath); err != nil {
		t.Fatal(err)
	}
	if historyPath != "/media/b.mp3" {
		t.Errorf("history media_path = %q, want /media/b.mp3", historyPath)
	}

	var captionPath string
	if err := f.db.QueryRow(
		"SELECT media_path FROM captions WHERE text = 'hello world'",
	).Scan(&captionPath); err != nil {
		t.Fatal(err)
	}
	if captionPath != "/media/b.mp3" {
		t.Errorf("caption media_path = %q, want /media/b.mp3", captionPath)
	}

	if got := f.queryInt(t, "SELECT count(*) FROM history_orphaned WHERE media_id = 999"); got != 1 {
		t.Errorf("history_orphaned rows = %d, want 1", got)
	}
	if got := f.queryInt(t, "SELECT count(*) FROM captions_orphaned WHERE media_id = 999"); got != 1 {
		t.Errorf("captions_orphaned rows = %d, want 1", got)
	}

	assertNoForeignKeyViolations(t, f.db)

	// Migrate alone does not create FTS on a database that never had it.
	// InitDB is the production entry point, so run it and verify the rebuilt
	// indexes map to the original media rows.
	if err := db.InitDB(ctx, f.db); err != nil {
		t.Fatalf("InitDB after Migrate failed: %v", err)
	}

	assertFTSPaths(t, f.db)
	assertCanonicalFTSSchema(t, f.db)
}

func assertNoForeignKeyViolations(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	rows, err := sqlDB.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var table string
		var rowid int64
		var parent string
		var fkid int
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			t.Fatal(err)
		}
		t.Errorf("foreign key violation: %s rowid=%d -> %s (fk %d)", table, rowid, parent, fkid)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func assertFTSPaths(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	var mediaPath string
	err := sqlDB.QueryRow(`
		SELECT m.path
		FROM media m
		JOIN media_fts ON m.rowid = media_fts.rowid
		WHERE media_fts MATCH 'mkv'
	`).Scan(&mediaPath)
	if err != nil {
		t.Fatalf("media_fts does not map to media rows: %v", err)
	}
	if mediaPath != "/media/a.mkv" {
		t.Errorf("media_fts path = %q, want /media/a.mkv", mediaPath)
	}

	var captionPath string
	err = sqlDB.QueryRow(`
		SELECT c.media_path
		FROM captions c
		JOIN captions_fts ON c.rowid = captions_fts.rowid
		WHERE captions_fts MATCH 'hello'
	`).Scan(&captionPath)
	if err != nil {
		t.Fatalf("captions_fts does not map to caption rows: %v", err)
	}
	if captionPath != "/media/b.mp3" {
		t.Errorf("captions_fts path = %q, want /media/b.mp3", captionPath)
	}
}

func assertCanonicalFTSSchema(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	for table, expectedColumns := range map[string][]string{
		"media_fts":    {"path", "path_tokenized", "title", "description", "time_deleted"},
		"captions_fts": {"media_path", "text"},
	} {
		columns := tableColumnNames(t, sqlDB, table)
		if !slices.Equal(columns, expectedColumns) {
			t.Errorf("%s columns = %v, want %v", table, columns, expectedColumns)
		}

		var tableSQL string
		if err := sqlDB.QueryRow(
			"SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&tableSQL); err != nil {
			t.Fatalf("%s SQL missing: %v", table, err)
		}
		normalizedTableSQL := strings.ReplaceAll(tableSQL, " ", "")
		for _, part := range []string{"trigram", "detail='full'"} {
			if !strings.Contains(normalizedTableSQL, part) {
				t.Errorf("%s SQL missing %q: %s", table, part, tableSQL)
			}
		}
	}

	for _, trigger := range []string{"media_ai", "media_ad", "media_au", "captions_ai", "captions_ad", "captions_au"} {
		var count int
		if err := sqlDB.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type = 'trigger' AND name = ?",
			trigger,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("trigger %s missing", trigger)
		}
	}
}

func tableColumnNames(t *testing.T, sqlDB *sql.DB, table string) []string {
	t.Helper()
	rows, err := sqlDB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func TestFailedMediaRecreationRollsBack(t *testing.T) {
	f := newLegacyFixture(t)
	f.exec(t, `CREATE TABLE media (
		path TEXT PRIMARY KEY,
		title TEXT,
		uploader TEXT
	)`)
	f.exec(t, `INSERT INTO media (path, title, uploader) VALUES ('/media/a.mkv', 'Alpha', 'user')`)
	// Force CREATE TABLE media_dg_tmp to fail inside recreateMediaTable.
	f.exec(t, `CREATE TABLE media_dg_tmp (blocker TEXT)`)

	if err := db.Migrate(context.Background(), f.db); err == nil {
		t.Fatal("Migrate succeeded despite the recreation blocker")
	}

	if got := f.queryInt(t, "SELECT count(*) FROM media WHERE path = '/media/a.mkv'"); got != 1 {
		t.Errorf("media row lost after failed migration: count = %d", got)
	}
	if !slices.Contains(tableColumnNames(t, f.db, "media"), "uploader") {
		t.Error("media table was replaced despite the failed recreation")
	}
}

func TestMigrateRebuildsExistingFTS(t *testing.T) {
	f := newLegacyFixture(t)
	populateLegacySchema(t, f)

	// Simulate a database that already has canonical FTS tables and no
	// triggers. Recreating media must leave both indexes mapping correctly.
	f.exec(t, `CREATE VIRTUAL TABLE media_fts USING fts5(
		path, path_tokenized, title, description, time_deleted UNINDEXED,
		content='media', content_rowid='rowid', tokenize='trigram', detail='full')`)
	f.exec(t, `CREATE VIRTUAL TABLE captions_fts USING fts5(
		media_path UNINDEXED, text, content='captions', tokenize='trigram', detail='full')`)

	if err := db.Migrate(context.Background(), f.db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	assertFTSPaths(t, f.db)
	assertCanonicalFTSSchema(t, f.db)
	assertNoForeignKeyViolations(t, f.db)
}

func TestMigrateMetadataConsolidation(t *testing.T) {
	tests := []struct {
		name        string
		mediaSQL    string
		insertSQL   string
		wantContain []string
	}{
		{
			name: "only uploader present",
			mediaSQL: `CREATE TABLE media (
				path TEXT PRIMARY KEY,
				title TEXT,
				uploader TEXT
			)`,
			insertSQL: `INSERT INTO media (path, title, uploader) VALUES ('/media/a.mkv', 'Alpha', 'user')`,
		},
		{
			name: "only webpath present",
			mediaSQL: `CREATE TABLE media (
				path TEXT PRIMARY KEY,
				title TEXT,
				webpath TEXT
			)`,
			insertSQL: `INSERT INTO media (path, title, webpath) VALUES ('/media/a.mkv', 'Alpha', '/web/a')`,
		},
		{
			name: "all metadata columns present",
			mediaSQL: `CREATE TABLE media (
				path TEXT PRIMARY KEY,
				title TEXT,
				description TEXT,
				decade TEXT,
				mood TEXT,
				bpm INTEGER,
				key TEXT
			)`,
			insertSQL: `INSERT INTO media (path, title, description, decade, mood, bpm, key)
				VALUES ('/media/a.mkv', 'Alpha', 'base', '1990', 'calm', 120, 'C')`,
			wantContain: []string{"Date: 1990", "Mood: calm", "BPM: 120", "Key: C"},
		},
		{
			name: "strict change with no dead columns",
			mediaSQL: `CREATE TABLE media (
				path TEXT PRIMARY KEY,
				title TEXT,
				description TEXT,
				duration INTEGER,
				size INTEGER
			)`,
			insertSQL: `INSERT INTO media (path, title, description) VALUES ('/media/a.mkv', 'Alpha', 'base')`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLegacyFixture(t)
			f.exec(t, tt.mediaSQL)
			f.exec(t, tt.insertSQL)

			if err := db.Migrate(context.Background(), f.db); err != nil {
				t.Fatalf("Migrate failed: %v", err)
			}

			var description sql.NullString
			if err := f.db.QueryRow(
				"SELECT description FROM media WHERE path = '/media/a.mkv'",
			).Scan(&description); err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.wantContain {
				if !description.Valid || !strings.Contains(description.String, want) {
					t.Errorf("description = %q, want it to contain %q", description.String, want)
				}
			}

			// Dead columns are gone even when there was no metadata to
			// consolidate.
			for _, deadColumn := range []string{"uploader", "webpath", "decade", "mood", "bpm", "key"} {
				if slices.Contains(tableColumnNames(t, f.db, "media"), deadColumn) {
					t.Errorf("dead column %s still present", deadColumn)
				}
			}
		})
	}
}

func TestFreshAndMigratedSchemasMatch(t *testing.T) {
	ctx := context.Background()

	freshPath := filepath.Join(t.TempDir(), "fresh.db")
	freshDB, err := sql.Open("sqlite3", freshPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = freshDB.Close() })
	if err := db.InitDB(ctx, freshDB); err != nil {
		t.Fatalf("InitDB fresh failed: %v", err)
	}

	f := newLegacyFixture(t)
	populateLegacySchema(t, f)
	if err := db.InitDB(ctx, f.db); err != nil {
		t.Fatalf("InitDB legacy failed: %v", err)
	}

	for _, table := range []string{
		"media", "playlists", "playlist_items", "captions", "history",
		"custom_keywords", "folder_stats", "folder_stats_meta", "_maintenance_meta",
	} {
		freshColumns := tableInfo(t, freshDB, table)
		migratedColumns := tableInfo(t, f.db, table)
		if !slices.Equal(freshColumns, migratedColumns) {
			t.Errorf("%s schema mismatch:\n fresh:    %v\n migrated: %v", table, freshColumns, migratedColumns)
		}

		freshFKs := foreignKeys(t, freshDB, table)
		migratedFKs := foreignKeys(t, f.db, table)
		if !slices.Equal(freshFKs, migratedFKs) {
			t.Errorf("%s foreign key mismatch:\n fresh:    %v\n migrated: %v", table, freshFKs, migratedFKs)
		}
	}

	freshIndexes := nonInternalIndexes(t, freshDB)
	migratedIndexes := nonInternalIndexes(t, f.db)
	if !slices.Equal(freshIndexes, migratedIndexes) {
		t.Errorf("index mismatch:\n fresh:    %v\n migrated: %v", freshIndexes, migratedIndexes)
	}
}

type columnInfo struct {
	Name    string
	Type    string
	NotNull int
	Default string
	PK      int
}

func tableInfo(t *testing.T, sqlDB *sql.DB, table string) []columnInfo {
	t.Helper()
	rows, err := sqlDB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var columns []columnInfo
	for rows.Next() {
		var cid int
		var info columnInfo
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &info.Name, &info.Type, &info.NotNull, &defaultValue, &info.PK); err != nil {
			t.Fatal(err)
		}
		info.Default = defaultValue.String
		columns = append(columns, info)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func nonInternalIndexes(t *testing.T, sqlDB *sql.DB) []string {
	t.Helper()
	rows, err := sqlDB.Query(`
		SELECT name FROM sqlite_master
		WHERE type = 'index' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

func TestInitDBCreatesFTSSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fresh-fts.db")
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	for _, table := range []string{"captions", "media_fts", "captions_fts"} {
		var count int
		if err := sqlDB.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("table %s missing from fresh InitDB database", table)
		}
	}

	if !slices.Contains(tableColumnNames(t, sqlDB, "media"), "path_tokenized") {
		t.Error("media.path_tokenized missing from fresh InitDB database")
	}
}

func TestMigrateUpgradesOldFTSTables(t *testing.T) {
	variants := []struct {
		name     string
		tableSQL string
	}{
		{
			name: "no trigram",
			tableSQL: `CREATE VIRTUAL TABLE media_fts USING fts5(
				path, title, content='media', content_rowid='rowid')`,
		},
		{
			name: "trigram missing path_tokenized",
			tableSQL: `CREATE VIRTUAL TABLE media_fts USING fts5(
				path, title, description, time_deleted UNINDEXED,
				content='media', content_rowid='rowid', tokenize='trigram', detail='full')`,
		},
		{
			name: "detail none",
			tableSQL: `CREATE VIRTUAL TABLE media_fts USING fts5(
				path, path_tokenized, title, description, time_deleted UNINDEXED,
				content='media', content_rowid='rowid', tokenize='trigram', detail='none')`,
		},
	}

	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "fts-upgrade.db")
			sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()

			if err := db.InitDB(context.Background(), sqlDB); err != nil {
				t.Fatalf("InitDB failed: %v", err)
			}
			if _, err := sqlDB.Exec("DROP TRIGGER IF EXISTS media_ai"); err != nil {
				t.Fatal(err)
			}
			if _, err := sqlDB.Exec("DROP TRIGGER IF EXISTS media_ad"); err != nil {
				t.Fatal(err)
			}
			if _, err := sqlDB.Exec("DROP TRIGGER IF EXISTS media_au"); err != nil {
				t.Fatal(err)
			}
			if _, err := sqlDB.Exec("DROP TABLE media_fts"); err != nil {
				t.Fatal(err)
			}
			if _, err := sqlDB.Exec(variant.tableSQL); err != nil {
				t.Fatal(err)
			}

			if _, err := sqlDB.Exec(
				`INSERT INTO media (path, title, description) VALUES ('/media/upgrade.mkv', 'Upgrade', 'findme')`,
			); err != nil {
				t.Fatal(err)
			}

			if err := db.Migrate(context.Background(), sqlDB); err != nil {
				t.Fatalf("Migrate failed: %v", err)
			}

			assertCanonicalFTSSchema(t, sqlDB)
			assertFTSPathsContain(t, sqlDB, "upgrade")
		})
	}
}

func TestMigrateReplacesOutdatedFTSTriggers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fts-trigger.db")
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err := db.InitDB(context.Background(), sqlDB); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	for _, trigger := range []string{"media_ai", "media_ad", "media_au"} {
		if _, err := sqlDB.Exec("DROP TRIGGER IF EXISTS " + trigger); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sqlDB.Exec(`CREATE TRIGGER media_ai AFTER INSERT ON media BEGIN
		INSERT INTO media_fts(rowid, path, title) VALUES (new.rowid, new.path, new.title);
	END`); err != nil {
		t.Fatal(err)
	}

	if err := db.Migrate(context.Background(), sqlDB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	var triggerSQL string
	if err := sqlDB.QueryRow(
		"SELECT sql FROM sqlite_master WHERE type = 'trigger' AND name = 'media_ai'",
	).Scan(&triggerSQL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(triggerSQL, "path_tokenized") {
		t.Errorf("media_ai still has the old body: %s", triggerSQL)
	}
}

func assertFTSPathsContain(t *testing.T, sqlDB *sql.DB, needle string) {
	t.Helper()
	var count int
	if err := sqlDB.QueryRow(
		"SELECT count(*) FROM media_fts WHERE media_fts MATCH ?",
		needle,
	).Scan(&count); err != nil {
		t.Fatalf("FTS query failed: %v", err)
	}
	if count == 0 {
		t.Errorf("FTS table has no match for %q", needle)
	}
}

func TestConnectWithInitCreatesFTSSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "connect-init.db")
	sqlDB, queries, err := db.ConnectWithInit(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("ConnectWithInit failed: %v", err)
	}
	defer sqlDB.Close()
	if queries == nil {
		t.Fatal("ConnectWithInit returned nil queries")
	}

	for _, table := range []string{"captions", "media_fts", "captions_fts"} {
		var count int
		if err := sqlDB.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("table %s missing from ConnectWithInit database", table)
		}
	}
	if !slices.Contains(tableColumnNames(t, sqlDB, "media"), "path_tokenized") {
		t.Error("media.path_tokenized missing from ConnectWithInit database")
	}
}

func TestNoDuplicateMediaIndexes(t *testing.T) {
	sqlDB, _ := newInitializedDB(t)

	var indexNames []string
	rows, err := sqlDB.Query("PRAGMA index_list(media)")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var seq int
		var name string
		var unique, partial int
		var origin string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatal(err)
		}
		if name == "sqlite_autoindex_media_1" || partial == 1 {
			continue
		}
		indexNames = append(indexNames, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()

	seen := make(map[string]string)
	for _, indexName := range indexNames {
		columns := indexColumns(t, sqlDB, indexName)
		slices.Sort(columns)
		key := strings.Join(columns, ",")
		if previous, ok := seen[key]; ok {
			t.Errorf("duplicate media indexes on (%s): %s and %s", key, previous, indexName)
		}
		seen[key] = indexName
	}

	for _, removed := range []string{"idx_path", "idx_type"} {
		var count int
		if err := sqlDB.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = ?",
			removed,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("redundant index %s still exists", removed)
		}
	}

	var mediaTypeIndex int
	if err := sqlDB.QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_media_type'",
	).Scan(&mediaTypeIndex); err != nil {
		t.Fatal(err)
	}
	if mediaTypeIndex != 1 {
		t.Error("idx_media_type missing")
	}
}

func indexColumns(t *testing.T, sqlDB *sql.DB, indexName string) []string {
	t.Helper()
	rows, err := sqlDB.Query(fmt.Sprintf("PRAGMA index_info(%s)", indexName))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var seqno, cid int
		var name string
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func foreignKeys(t *testing.T, sqlDB *sql.DB, table string) []string {
	t.Helper()
	rows, err := sqlDB.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var id, seq int
		var parent, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &parent, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, fmt.Sprintf("%s:%s->%s:%s:%s:%s", parent, from, to, onUpdate, onDelete, match))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	slices.Sort(keys)
	return keys
}

func TestRepeatedInitDBIsSchemaIdempotent(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idempotent.db")
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err := db.InitDB(ctx, sqlDB); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	var before int
	if err := sqlDB.QueryRow("PRAGMA schema_version").Scan(&before); err != nil {
		t.Fatal(err)
	}

	if err := db.InitDB(ctx, sqlDB); err != nil {
		t.Fatalf("second InitDB failed: %v", err)
	}
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatalf("second Migrate failed: %v", err)
	}

	var after int
	if err := sqlDB.QueryRow("PRAGMA schema_version").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Errorf("repeated initialization changed the schema: version %d -> %d", before, after)
	}
}
