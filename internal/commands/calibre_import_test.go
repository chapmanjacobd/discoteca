package commands_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func createImportLibrary(t *testing.T, presentFormats, catalogFormats []string) string {
	t.Helper()
	libraryDir := t.TempDir()

	bookDir := filepath.Join(libraryDir, "Alice", "Test Book (1)")
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, format := range presentFormats {
		path := filepath.Join(bookDir, "Test Book - Alice."+format)
		if err := os.WriteFile(path, []byte("dummy "+format), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	sqlDB, err := sql.Open("sqlite3", filepath.Join(libraryDir, "metadata.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	schema := `
		CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT, path TEXT, series_index REAL);
		CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE books_authors_link (id INTEGER PRIMARY KEY, book INTEGER, author INTEGER);
		CREATE TABLE series (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE books_series_link (id INTEGER PRIMARY KEY, book INTEGER, series INTEGER);
		CREATE TABLE tags (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE books_tags_link (id INTEGER PRIMARY KEY, book INTEGER, tag INTEGER);
		CREATE TABLE comments (id INTEGER PRIMARY KEY, book INTEGER, text TEXT);
		CREATE TABLE data (id INTEGER PRIMARY KEY, book INTEGER, format TEXT, uncompressed_size INTEGER, name TEXT);
		CREATE TABLE languages (id INTEGER PRIMARY KEY, lang_code TEXT);
		CREATE TABLE books_languages_link (id INTEGER PRIMARY KEY, book INTEGER, lang_code INTEGER, item_order INTEGER);
		CREATE TABLE publishers (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE books_publishers_link (id INTEGER PRIMARY KEY, book INTEGER, publisher INTEGER);
		CREATE TABLE ratings (id INTEGER PRIMARY KEY, rating INTEGER);
		CREATE TABLE books_ratings_link (id INTEGER PRIMARY KEY, book INTEGER, rating INTEGER);
	`
	if _, err := sqlDB.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO books (id, title, path, series_index) VALUES (1, 'Test Book', 'Alice/Test Book (1)', 2);
		INSERT INTO authors (id, name) VALUES (1, 'Alice');
		INSERT INTO books_authors_link (id, book, author) VALUES (1, 1, 1);
		INSERT INTO series (id, name) VALUES (1, 'The Series');
		INSERT INTO books_series_link (id, book, series) VALUES (1, 1, 1);
		INSERT INTO tags (id, name) VALUES (1, 'Fantasy'), (2, 'Adventure');
		INSERT INTO books_tags_link (id, book, tag) VALUES (1, 1, 1), (2, 1, 2);
		INSERT INTO comments (id, book, text) VALUES (1, 1, '<p>Hello <b>world</b></p>');
		INSERT INTO languages (id, lang_code) VALUES (1, 'eng');
		INSERT INTO books_languages_link (id, book, lang_code, item_order) VALUES (1, 1, 1, 1);
		INSERT INTO publishers (id, name) VALUES (1, 'Example Press');
		INSERT INTO books_publishers_link (id, book, publisher) VALUES (1, 1, 1);
		INSERT INTO ratings (id, rating) VALUES (1, 8);
		INSERT INTO books_ratings_link (id, book, rating) VALUES (1, 1, 1);
	`); err != nil {
		t.Fatal(err)
	}

	for i, format := range catalogFormats {
		if _, err := sqlDB.Exec(
			`INSERT INTO data (id, book, format, uncompressed_size, name) VALUES (?, 1, ?, ?, 'Test Book - Alice')`,
			i+1, format, 100+i,
		); err != nil {
			t.Fatal(err)
		}
	}

	return libraryDir
}

func runImport(t *testing.T, cmd *commands.CalibreImportCmd) error {
	t.Helper()
	if err := cmd.AfterApply(); err != nil {
		return err
	}
	return cmd.Run(context.Background())
}

func openMediaDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return sqlDB
}

func TestCalibreImportCmd(t *testing.T) {
	libraryDir := createImportLibrary(t, []string{"epub"}, []string{"EPUB", "PDF"})
	dbPath := filepath.Join(t.TempDir(), "books.db")

	cmd := &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir}
	if err := runImport(t, cmd); err != nil {
		t.Fatalf("Run(): %v", err)
	}

	sqlDB := openMediaDB(t, dbPath)
	var (
		path, title, artist, mediaType, categories, description, language string
		score                                                             float64
		size                                                              int64
	)
	err := sqlDB.QueryRow(`
		SELECT path, title, artist, media_type, categories, description, language, score, size
		FROM media
	`).Scan(&path, &title, &artist, &mediaType, &categories, &description, &language, &score, &size)
	if err != nil {
		t.Fatal(err)
	}

	wantPath := filepath.Join(libraryDir, "Alice", "Test Book (1)", "Test Book - Alice.epub")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}
	if title != "Test Book" || artist != "Alice" || mediaType != "text" {
		t.Errorf("title=%q artist=%q media_type=%q", title, artist, mediaType)
	}
	if categories != ";Adventure;Fantasy;" {
		t.Errorf("categories = %q", categories)
	}
	if description != "Hello world\n\nSeries: The Series #2\n\nPublisher: Example Press" {
		t.Errorf("description = %q", description)
	}
	if language != "eng" || score != 4 || size != int64(len("dummy epub")) {
		t.Errorf("language=%q score=%v size=%d", language, score, size)
	}

	var playlistCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM playlists WHERE path = ? AND extractor_key = 'Calibre'`, libraryDir).Scan(&playlistCount); err != nil {
		t.Fatal(err)
	}
	if playlistCount != 1 {
		t.Errorf("expected the library to be recorded as a playlist root, got %d", playlistCount)
	}
}

func TestCalibreImportIncludeMissingAndOneFormat(t *testing.T) {
	libraryDir := createImportLibrary(t, []string{"epub"}, []string{"EPUB", "PDF"})

	t.Run("missing format skipped by default", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "books.db")
		cmd := &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir}
		if err := runImport(t, cmd); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := openMediaDB(t, dbPath).QueryRow(`SELECT COUNT(*) FROM media`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("expected 1 imported format, got %d", count)
		}
	})

	t.Run("include missing", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "books.db")
		cmd := &commands.CalibreImportCmd{
			Database:       dbPath,
			Library:        libraryDir,
			IncludeMissing: true,
		}
		if err := runImport(t, cmd); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := openMediaDB(t, dbPath).QueryRow(`SELECT COUNT(*) FROM media`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("expected 2 imported formats, got %d", count)
		}
	})

	t.Run("one format", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "books.db")
		cmd := &commands.CalibreImportCmd{
			Database:       dbPath,
			Library:        libraryDir,
			IncludeMissing: true,
			OneFormat:      true,
		}
		if err := runImport(t, cmd); err != nil {
			t.Fatal(err)
		}
		var format string
		if err := openMediaDB(t, dbPath).QueryRow(`SELECT path FROM media`).Scan(&format); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(format) != ".epub" {
			t.Errorf("expected EPUB to win the default priority, got %q", format)
		}
	})

	t.Run("one format falls back when preferred is missing", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "books.db")
		cmd := &commands.CalibreImportCmd{
			Database:       dbPath,
			Library:        libraryDir,
			OneFormat:      true,
			FormatPriority: []string{"pdf"},
		}
		if err := runImport(t, cmd); err != nil {
			t.Fatal(err)
		}
		var format string
		if err := openMediaDB(t, dbPath).QueryRow(`SELECT path FROM media`).Scan(&format); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(format) != ".epub" {
			t.Errorf("expected EPUB fallback, got %q", format)
		}
	})

	t.Run("custom priority", func(t *testing.T) {
		bothFormats := createImportLibrary(t, []string{"epub", "pdf"}, []string{"EPUB", "PDF"})
		dbPath := filepath.Join(t.TempDir(), "books.db")
		cmd := &commands.CalibreImportCmd{
			Database:       dbPath,
			Library:        bothFormats,
			OneFormat:      true,
			FormatPriority: []string{"pdf", "epub"},
		}
		if err := runImport(t, cmd); err != nil {
			t.Fatal(err)
		}
		var format string
		if err := openMediaDB(t, dbPath).QueryRow(`SELECT path FROM media`).Scan(&format); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(format) != ".pdf" {
			t.Errorf("expected PDF to win the custom priority, got %q", format)
		}
	})
}

func TestCalibreImportProbesVideoMetadata(t *testing.T) {
	installMockFFProbe(t, `{
  "streams": [
    {"codec_type": "video", "codec_name": "h264", "width": 1920, "height": 1080, "avg_frame_rate": "30/1"},
    {"codec_type": "audio", "codec_name": "aac"}
  ],
  "format": {"duration": "123.45", "tags": {"title": "yt-dlp embedded title"}}
}`)

	libraryDir := createImportLibrary(t, []string{"mp4"}, []string{"MP4"})
	dbPath := filepath.Join(t.TempDir(), "books.db")

	cmd := &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir, ProbeMedia: true}
	if err := runImport(t, cmd); err != nil {
		t.Fatal(err)
	}

	sqlDB := openMediaDB(t, dbPath)
	var (
		title, mediaType, videoCodecs, audioCodecs string
		duration, width, height                    int64
	)
	if err := sqlDB.QueryRow(`
		SELECT title, media_type, video_codecs, audio_codecs, duration, width, height
		FROM media
	`).Scan(&title, &mediaType, &videoCodecs, &audioCodecs, &duration, &width, &height); err != nil {
		t.Fatal(err)
	}

	if title != "Test Book" {
		t.Errorf("Calibre title should win over embedded tags, got %q", title)
	}
	if mediaType != "video" {
		t.Errorf("media_type = %q, want video", mediaType)
	}
	if videoCodecs != "h264" || audioCodecs != "aac" {
		t.Errorf("codecs = %q / %q, want h264 / aac", videoCodecs, audioCodecs)
	}
	if duration != 123 || width != 1920 || height != 1080 {
		t.Errorf("duration=%d width=%d height=%d, want 123/1920/1080", duration, width, height)
	}
}

func TestCalibreImportProbesAudiobookDuration(t *testing.T) {
	installMockFFProbe(t, `{
  "streams": [
    {"codec_type": "audio", "codec_name": "aac"}
  ],
  "format": {"duration": "7200"}
}`)

	libraryDir := createImportLibrary(t, []string{"m4b"}, []string{"M4B"})
	dbPath := filepath.Join(t.TempDir(), "books.db")

	cmd := &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir, ProbeMedia: true}
	if err := runImport(t, cmd); err != nil {
		t.Fatal(err)
	}

	sqlDB := openMediaDB(t, dbPath)
	var (
		mediaType string
		duration  int64
	)
	if err := sqlDB.QueryRow(`SELECT media_type, duration FROM media`).Scan(&mediaType, &duration); err != nil {
		t.Fatal(err)
	}
	if mediaType != "audiobook" {
		t.Errorf("media_type = %q, want audiobook", mediaType)
	}
	if duration != 7200 {
		t.Errorf("duration = %d, want 7200", duration)
	}
}

func TestCalibreImportProbeDisabled(t *testing.T) {
	installMockFFProbe(t, `{
  "streams": [
    {"codec_type": "video", "codec_name": "h264", "width": 1920, "height": 1080, "avg_frame_rate": "30/1"}
  ],
  "format": {"duration": "123.45"}
}`)

	libraryDir := createImportLibrary(t, []string{"mp4"}, []string{"MP4"})
	dbPath := filepath.Join(t.TempDir(), "books.db")

	cmd := &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir, ProbeMedia: false}
	if err := runImport(t, cmd); err != nil {
		t.Fatal(err)
	}

	sqlDB := openMediaDB(t, dbPath)
	var (
		mediaType   string
		duration    sql.NullInt64
		videoCodecs sql.NullString
	)
	if err := sqlDB.QueryRow(`SELECT media_type, duration, video_codecs FROM media`).
		Scan(&mediaType, &duration, &videoCodecs); err != nil {
		t.Fatal(err)
	}
	if mediaType != "video" {
		t.Errorf("media_type = %q, want video", mediaType)
	}
	if duration.Valid || videoCodecs.Valid {
		t.Errorf("duration/codecs should stay unset, got %v / %v", duration, videoCodecs)
	}
}

func installMockFFProbe(t *testing.T, output string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock ffprobe needs a POSIX shell")
	}

	dir := t.TempDir()
	script := "#!/bin/sh\ncat <<'EOF'\n" + output + "\nEOF\n"
	if err := os.WriteFile(filepath.Join(dir, "ffprobe"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestCalibreImportPreservesPlaybackState(t *testing.T) {
	libraryDir := createImportLibrary(t, []string{"epub"}, []string{"EPUB"})
	dbPath := filepath.Join(t.TempDir(), "books.db")

	if err := runImport(t, &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir}); err != nil {
		t.Fatal(err)
	}

	sqlDB := openMediaDB(t, dbPath)
	mediaPath := filepath.Join(libraryDir, "Alice", "Test Book (1)", "Test Book - Alice.epub")
	if _, err := sqlDB.Exec(`
		UPDATE media SET play_count = 3, playhead = 42, time_last_played = 123, time_deleted = 9
		WHERE path = ?`, mediaPath); err != nil {
		t.Fatal(err)
	}

	metadataDB, err := sql.Open("sqlite3", filepath.Join(libraryDir, "metadata.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := metadataDB.Exec(`UPDATE books SET title = 'Updated Title' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	metadataDB.Close()

	if err := runImport(t, &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir}); err != nil {
		t.Fatal(err)
	}

	var (
		title                                string
		playCount, playhead, lastPlayed, del int64
	)
	if err := sqlDB.QueryRow(`
		SELECT title, play_count, playhead, time_last_played, time_deleted
		FROM media WHERE path = ?`, mediaPath).Scan(&title, &playCount, &playhead, &lastPlayed, &del); err != nil {
		t.Fatal(err)
	}
	if title != "Updated Title" {
		t.Errorf("title = %q", title)
	}
	if playCount != 3 || playhead != 42 || lastPlayed != 123 {
		t.Errorf("playback state changed: play_count=%d playhead=%d last_played=%d", playCount, playhead, lastPlayed)
	}
	if del != 0 {
		t.Errorf("re-import should undelete the row, time_deleted = %d", del)
	}
}

func TestCalibreImportPrune(t *testing.T) {
	libraryDir := createImportLibrary(t, []string{"epub"}, []string{"EPUB"})
	dbPath := filepath.Join(t.TempDir(), "books.db")

	if err := runImport(t, &commands.CalibreImportCmd{Database: dbPath, Library: libraryDir}); err != nil {
		t.Fatal(err)
	}

	sqlDB := openMediaDB(t, dbPath)
	stalePath := filepath.Join(libraryDir, "Alice", "Test Book (1)", "cover.jpg")
	outsidePath := filepath.Join(t.TempDir(), "outside.epub")
	if _, err := sqlDB.Exec(`
		INSERT INTO media (path, media_type) VALUES (?, 'image'), (?, 'text')`,
		stalePath, outsidePath); err != nil {
		t.Fatal(err)
	}

	if err := runImport(t, &commands.CalibreImportCmd{
		Database: dbPath,
		Library:  libraryDir,
		Prune:    true,
	}); err != nil {
		t.Fatal(err)
	}

	deleted := func(path string) int64 {
		var value int64
		if err := sqlDB.QueryRow(`SELECT time_deleted FROM media WHERE path = ?`, path).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	if deleted(stalePath) == 0 {
		t.Error("stale file under the library should be marked deleted")
	}
	if deleted(outsidePath) != 0 {
		t.Error("files outside the library must not be pruned")
	}
	if deleted(filepath.Join(libraryDir, "Alice", "Test Book (1)", "Test Book - Alice.epub")) != 0 {
		t.Error("imported files must not be pruned")
	}
}

func TestCalibreImportSimulateDoesNotCreateDatabase(t *testing.T) {
	libraryDir := createImportLibrary(t, []string{"epub"}, []string{"EPUB"})
	dbPath := filepath.Join(t.TempDir(), "books.db")

	cmd := &commands.CalibreImportCmd{
		Database: dbPath,
		Library:  libraryDir,
		CoreFlags: models.CoreFlags{
			Simulate: true,
		},
	}
	if err := runImport(t, cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Errorf("simulate should not create %s", dbPath)
	}
}
