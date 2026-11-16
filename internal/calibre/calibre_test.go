package calibre_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/calibre"
)

func createLibrary(t *testing.T) string {
	t.Helper()
	libraryDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(libraryDir, "Alice", "Test Book (1)"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"Test Book - Alice.epub",
		"Test Book - Alice.pdf",
		"cover.jpg",
	} {
		if err := os.WriteFile(
			filepath.Join(libraryDir, "Alice", "Test Book (1)", name),
			[]byte("dummy"),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}

	dbPath := filepath.Join(libraryDir, calibre.MetadataFilename)
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	schema := `
		CREATE TABLE books (
			id INTEGER PRIMARY KEY, title TEXT, path TEXT, series_index REAL,
			has_cover BOOL, last_modified TEXT
		);
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

	statements := []string{
		`INSERT INTO books (id, title, path, series_index) VALUES (1, 'Test Book', 'Alice/Test Book (1)', 2)`,
		`INSERT INTO authors (id, name) VALUES (1, 'Alice'), (2, 'Bob')`,
		`INSERT INTO books_authors_link (id, book, author) VALUES (1, 1, 1), (2, 1, 2)`,
		`INSERT INTO series (id, name) VALUES (1, 'The Series')`,
		`INSERT INTO books_series_link (id, book, series) VALUES (1, 1, 1)`,
		`INSERT INTO tags (id, name) VALUES (1, 'Fantasy'), (2, 'Adventure')`,
		`INSERT INTO books_tags_link (id, book, tag) VALUES (1, 1, 1), (2, 1, 2)`,
		`INSERT INTO comments (id, book, text) VALUES (1, 1, '<p>Hello <b>world</b> &amp; friends</p>')`,
		`INSERT INTO data (id, book, format, uncompressed_size, name) VALUES
			(1, 1, 'EPUB', 10, 'Test Book - Alice'),
			(2, 1, 'PDF', 20, 'Test Book - Alice'),
			(3, 1, 'MOBI', 30, 'Test Book - Alice')`,
		`INSERT INTO languages (id, lang_code) VALUES (1, 'eng')`,
		`INSERT INTO books_languages_link (id, book, lang_code, item_order) VALUES (1, 1, 1, 1)`,
		`INSERT INTO publishers (id, name) VALUES (1, 'Example Press')`,
		`INSERT INTO books_publishers_link (id, book, publisher) VALUES (1, 1, 1)`,
		`INSERT INTO ratings (id, rating) VALUES (1, 8)`,
		`INSERT INTO books_ratings_link (id, book, rating) VALUES (1, 1, 1)`,
	}
	for _, statement := range statements {
		if _, err := sqlDB.Exec(statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}

	return libraryDir
}

func TestOpenAndBooks(t *testing.T) {
	libraryDir := createLibrary(t)
	ctx := context.Background()

	for _, path := range []string{libraryDir, filepath.Join(libraryDir, calibre.MetadataFilename)} {
		library, err := calibre.Open(ctx, path)
		if err != nil {
			t.Fatalf("Open(%q): %v", path, err)
		}

		books, err := library.Books(ctx)
		if err != nil {
			t.Fatalf("Books(): %v", err)
		}
		if len(books) != 1 {
			t.Fatalf("expected 1 book, got %d", len(books))
		}

		book := books[0]
		if book.Title != "Test Book" {
			t.Errorf("title = %q", book.Title)
		}
		if len(book.Authors) != 2 || book.Authors[0] != "Alice" || book.Authors[1] != "Bob" {
			t.Errorf("authors = %#v", book.Authors)
		}
		if len(book.Tags) != 2 || book.Tags[0] != "Adventure" || book.Tags[1] != "Fantasy" {
			t.Errorf("tags = %#v", book.Tags)
		}
		if book.Series != "The Series" || !book.HasSeriesIndex || book.SeriesIndex != 2 {
			t.Errorf("series = %#v index=%v valid=%v", book.Series, book.SeriesIndex, book.HasSeriesIndex)
		}
		if book.Rating != 8 {
			t.Errorf("rating = %v", book.Rating)
		}
		if book.Language != "eng" || book.Publisher != "Example Press" {
			t.Errorf("language=%q publisher=%q", book.Language, book.Publisher)
		}
		if book.Description != "Hello world & friends" {
			t.Errorf("description = %q", book.Description)
		}
		if len(book.Files) != 3 {
			t.Fatalf("expected 3 formats, got %d", len(book.Files))
		}
		if book.Files[0].Format != "EPUB" ||
			book.Files[0].Path != filepath.Join(libraryDir, "Alice", "Test Book (1)", "Test Book - Alice.epub") {
			t.Errorf("file = %#v", book.Files[0])
		}

		if err := library.Close(); err != nil {
			t.Errorf("Close(): %v", err)
		}
	}
}

func TestLibraryFromCalibreWebDir(t *testing.T) {
	libraryDir := createLibrary(t)
	webDir := t.TempDir()

	sqlDB, err := sql.Open("sqlite3", filepath.Join(webDir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err = sqlDB.Exec(`CREATE TABLE settings (config_calibre_dir TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err = sqlDB.Exec(`INSERT INTO settings (config_calibre_dir) VALUES (?)`, libraryDir); err != nil {
		t.Fatal(err)
	}

	got, err := calibre.LibraryFromCalibreWebDir(context.Background(), webDir)
	if err != nil {
		t.Fatalf("LibraryFromCalibreWebDir(): %v", err)
	}
	if got != libraryDir {
		t.Errorf("library = %q, want %q", got, libraryDir)
	}
}

func TestResolveLibraryPathRejectsNonLibrary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-library")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := calibre.Open(context.Background(), path); err == nil {
		t.Fatal("expected an error for a non-library file")
	}
}
