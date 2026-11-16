// Package calibre reads Calibre library metadata without modifying the library.
//
// A Calibre library is a directory containing metadata.db and one folder per
// book. This package exposes the catalog fields that are useful for importing
// books into a discoteca database; it never writes to the Calibre database.
package calibre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	// Register the SQLite driver used to open Calibre's metadata.db.
	_ "github.com/mattn/go-sqlite3"
)

// Library is an open Calibre library. Close it when finished.
type Library struct {
	// Dir is the absolute library root that contains MetadataPath.
	Dir string
	// MetadataPath is the absolute path to metadata.db.
	MetadataPath string

	conn *sql.DB
}

// Book is one catalog entry. Files contains one entry per format recorded in
// the Calibre data table.
type Book struct {
	ID             int64
	Title          string
	Path           string // relative path inside the library, as stored by Calibre
	Authors        []string
	Series         string
	SeriesIndex    float64
	HasSeriesIndex bool
	Tags           []string
	Description    string
	Language       string
	Publisher      string
	// Rating is Calibre's 0-10 rating. Zero means unrated.
	Rating float64
	Files  []File
}

// File is one book format on disk.
type File struct {
	BookID int64
	Format string // uppercase, for example EPUB
	Name   string // filename without the extension, as stored by Calibre
	Size   int64
	Path   string // absolute path on disk
}

// MetadataFilename is the catalog file stored in every Calibre library.
const MetadataFilename = "metadata.db"

var (
	htmlBreakRE = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlBlockRE = regexp.MustCompile(`(?i)</(p|div|li|h[1-6]|tr|blockquote)\s*>`)
	htmlTagRE   = regexp.MustCompile(`(?s)<[^>]*>`)
	spaceRE     = regexp.MustCompile(`[ \t\f\v]+`)
)

// Open opens a Calibre library. path may be the library directory or the
// metadata.db file itself.
func Open(ctx context.Context, path string) (*Library, error) {
	dir, metadataPath, err := resolveLibraryPath(path)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite3", readOnlyDSN(metadataPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open Calibre metadata %s: %w", metadataPath, err)
	}
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read Calibre metadata %s: %w", metadataPath, err)
	}

	return &Library{Dir: dir, MetadataPath: metadataPath, conn: conn}, nil
}

// Close releases the underlying metadata database.
func (l *Library) Close() error {
	if l == nil || l.conn == nil {
		return nil
	}
	return l.conn.Close()
}

// LibraryFromCalibreWebDir returns the Calibre library configured in a
// Calibre-Web installation directory containing app.db.
func LibraryFromCalibreWebDir(ctx context.Context, dir string) (string, error) {
	appDB := filepath.Join(dir, "app.db")
	conn, err := sql.Open("sqlite3", readOnlyDSN(appDB))
	if err != nil {
		return "", fmt.Errorf("failed to open Calibre-Web settings %s: %w", appDB, err)
	}
	defer conn.Close()

	var libraryPath string
	err = conn.QueryRowContext(ctx, "SELECT config_calibre_dir FROM settings LIMIT 1").Scan(&libraryPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config_calibre_dir from %s: %w", appDB, err)
	}
	if strings.TrimSpace(libraryPath) == "" {
		return "", fmt.Errorf("calibre-web settings in %s have no config_calibre_dir", appDB)
	}
	return libraryPath, nil
}

// Books loads the whole catalog. The returned files are ordered by book and
// format so repeated imports produce the same rows.
func (l *Library) Books(ctx context.Context) ([]Book, error) {
	books, byID, err := l.loadBooks(ctx)
	if err != nil {
		return nil, err
	}
	if err := l.loadAuthors(ctx, books, byID); err != nil {
		return nil, err
	}
	if err := l.loadTags(ctx, books, byID); err != nil {
		return nil, err
	}
	if err := l.loadFiles(ctx, books, byID); err != nil {
		return nil, err
	}

	for i := range books {
		slices.Sort(books[i].Tags)
	}
	return books, nil
}

func (l *Library) loadBooks(ctx context.Context) ([]Book, map[int64]int, error) {
	const query = `
		SELECT
			b.id,
			b.title,
			b.path,
			b.series_index,
			COALESCE((SELECT text FROM comments WHERE book = b.id LIMIT 1), ''),
			COALESCE((SELECT l.lang_code FROM books_languages_link bl
				JOIN languages l ON l.id = bl.lang_code
				WHERE bl.book = b.id ORDER BY bl.item_order, bl.id LIMIT 1), ''),
			COALESCE((SELECT p.name FROM books_publishers_link bp
				JOIN publishers p ON p.id = bp.publisher
				WHERE bp.book = b.id LIMIT 1), ''),
			COALESCE((SELECT r.rating FROM books_ratings_link br
				JOIN ratings r ON r.id = br.rating
				WHERE br.book = b.id LIMIT 1), 0),
			COALESCE((SELECT s.name FROM books_series_link bs
				JOIN series s ON s.id = bs.series
				WHERE bs.book = b.id LIMIT 1), '')
		FROM books b
		ORDER BY b.id`

	rows, err := l.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query Calibre books: %w", err)
	}
	defer rows.Close()

	var books []Book
	byID := make(map[int64]int)
	for rows.Next() {
		var book Book
		var seriesIndex sql.NullFloat64
		var rating int64
		if err := rows.Scan(
			&book.ID,
			&book.Title,
			&book.Path,
			&seriesIndex,
			&book.Description,
			&book.Language,
			&book.Publisher,
			&rating,
			&book.Series,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan Calibre book: %w", err)
		}
		if seriesIndex.Valid {
			book.SeriesIndex = seriesIndex.Float64
			book.HasSeriesIndex = true
		}
		book.Rating = float64(rating)
		book.Description = plainText(book.Description)
		byID[book.ID] = len(books)
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read Calibre books: %w", err)
	}
	return books, byID, nil
}

func (l *Library) loadAuthors(ctx context.Context, books []Book, byID map[int64]int) error {
	const query = `
		SELECT ba.book, a.name
		FROM books_authors_link ba
		JOIN authors a ON a.id = ba.author
		ORDER BY ba.book, ba.id`
	if err := l.loadBookNames(ctx, query, books, byID, func(book *Book, name string) {
		book.Authors = append(book.Authors, name)
	}); err != nil {
		return fmt.Errorf("failed to load Calibre authors: %w", err)
	}
	return nil
}

func (l *Library) loadTags(ctx context.Context, books []Book, byID map[int64]int) error {
	const query = `
		SELECT bt.book, t.name
		FROM books_tags_link bt
		JOIN tags t ON t.id = bt.tag
		ORDER BY bt.book, t.name`
	if err := l.loadBookNames(ctx, query, books, byID, func(book *Book, name string) {
		book.Tags = append(book.Tags, name)
	}); err != nil {
		return fmt.Errorf("failed to load Calibre tags: %w", err)
	}
	return nil
}

// loadBookNames reads a book-to-name link table and adds each name to the
// matching book through appendName.
func (l *Library) loadBookNames(
	ctx context.Context,
	query string,
	books []Book,
	byID map[int64]int,
	appendName func(*Book, string),
) error {
	rows, err := l.conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query Calibre link table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bookID int64
		var name string
		if err := rows.Scan(&bookID, &name); err != nil {
			return fmt.Errorf("failed to scan Calibre link name: %w", err)
		}
		if idx, ok := byID[bookID]; ok {
			if name = strings.TrimSpace(name); name != "" {
				appendName(&books[idx], name)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read Calibre link names: %w", err)
	}
	return nil
}

func (l *Library) loadFiles(ctx context.Context, books []Book, byID map[int64]int) error {
	const query = `SELECT book, format, name, uncompressed_size FROM data ORDER BY book, format`

	rows, err := l.conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query Calibre formats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bookID int64
		var format, name string
		var size int64
		if err := rows.Scan(&bookID, &format, &name, &size); err != nil {
			return fmt.Errorf("failed to scan Calibre format: %w", err)
		}
		idx, ok := byID[bookID]
		if !ok {
			continue
		}
		fileName := strings.TrimSpace(name)
		format = strings.TrimSpace(format)
		if fileName == "" || format == "" {
			continue
		}
		books[idx].Files = append(books[idx].Files, File{
			BookID: bookID,
			Format: strings.ToUpper(format),
			Name:   fileName,
			Size:   size,
			Path:   l.bookFilePath(books[idx].Path, fileName, format),
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read Calibre formats: %w", err)
	}
	return nil
}

func (l *Library) bookFilePath(bookPath, name, format string) string {
	fileName := filepath.Base(name) + "." + strings.ToLower(format)
	return filepath.Join(l.Dir, filepath.FromSlash(bookPath), fileName)
}

func resolveLibraryPath(path string) (libraryDir, metadataPath string, err error) {
	if strings.TrimSpace(path) == "" {
		return "", "", errors.New("calibre library path is empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve Calibre library path %s: %w", path, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", "", fmt.Errorf("calibre library not found: %w", err)
	}

	if info.IsDir() {
		metadataPath = filepath.Join(absPath, MetadataFilename)
	} else if strings.EqualFold(filepath.Base(absPath), MetadataFilename) {
		metadataPath = absPath
	} else {
		return "", "", fmt.Errorf("expected a Calibre library directory or %s: %s", MetadataFilename, path)
	}

	metaInfo, err := os.Stat(metadataPath)
	if err != nil {
		return "", "", fmt.Errorf("calibre catalog not found: %w", err)
	}
	if metaInfo.IsDir() {
		return "", "", fmt.Errorf("calibre catalog is a directory: %s", metadataPath)
	}

	return filepath.Dir(metadataPath), metadataPath, nil
}

func readOnlyDSN(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}
	return u.String() + "?mode=ro"
}

// plainText converts Calibre's HTML comments into readable text.
func plainText(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	s = htmlBreakRE.ReplaceAllString(s, "\n")
	s = htmlBlockRE.ReplaceAllString(s, "\n")
	s = htmlTagRE.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = spaceRE.ReplaceAllString(s, " ")

	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
