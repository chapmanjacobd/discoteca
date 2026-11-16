package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/calibre"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/textutil"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type CalibreImportCmd struct {
	models.CoreFlags `embed:""`

	Database string `help:"SQLite database file to import into"                        required:"true" arg:""`
	Library  string `help:"Calibre library directory or metadata.db"                   type:"path"`
	WebDir   string `help:"Calibre-Web directory containing app.db (reads config_calibre_dir)" type:"path" name:"calibre-web-dir"`

	FormatPriority []string `help:"Preferred formats in order, comma-separated; used by --one-format (default: epub,azw3,mobi,azw,pdf,...)" name:"format-priority"`
	OneFormat      bool     `help:"Import only the highest-priority format for each book"`
	IncludeMissing bool     `help:"Import formats recorded in Calibre even when the file is missing on disk"`
	Prune          bool     `help:"Mark media under the library as deleted when no longer in the Calibre catalog"`
}

// defaultCalibreFormatPriority is the order used when --one-format is set and
// the caller does not provide --format-priority.
var defaultCalibreFormatPriority = []string{
	"epub", "azw3", "mobi", "azw", "pdf", "fb2", "djvu", "djv",
	"txt", "rtf", "html", "htm", "docx", "odt", "lit", "prc", "cbz", "cbr",
}

func (c *CalibreImportCmd) AfterApply() error {
	if err := c.CoreFlags.AfterApply(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Database) == "" {
		return errors.New("a database file is required")
	}
	if strings.TrimSpace(c.Library) == "" && strings.TrimSpace(c.WebDir) == "" {
		return errors.New("either --library or --calibre-web-dir is required")
	}
	if strings.TrimSpace(c.Library) != "" && strings.TrimSpace(c.WebDir) != "" {
		return errors.New("use either --library or --calibre-web-dir, not both")
	}
	if len(c.FormatPriority) == 0 {
		c.FormatPriority = slices.Clone(defaultCalibreFormatPriority)
	}
	for i, format := range c.FormatPriority {
		c.FormatPriority[i] = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(format), "."))
	}
	return nil
}

func (c *CalibreImportCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	db.InitFtsConfig()

	libraryPath := c.Library
	if strings.TrimSpace(libraryPath) == "" {
		var err error
		libraryPath, err = calibre.LibraryFromCalibreWebDir(ctx, c.WebDir)
		if err != nil {
			return err
		}
		logging.Log.Info("Using Calibre-Web library", "path", libraryPath)
	}

	library, err := calibre.Open(ctx, libraryPath)
	if err != nil {
		return err
	}
	defer library.Close()

	books, err := library.Books(ctx)
	if err != nil {
		return err
	}
	logging.Log.Info("Loaded Calibre catalog", "books", len(books), "library", library.Dir)

	items, importedPaths, skipped := buildCalibreMedia(books, c.FormatPriority, c.OneFormat, c.IncludeMissing)

	if c.Simulate {
		fmt.Printf("Would import %d files from %d books into %s (skipped %d missing files)\n",
			len(items), len(books), c.Database, skipped)
		return nil
	}

	sqlDB, queries, err := db.ConnectWithInit(ctx, c.Database)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if _, err := queries.InsertPlaylist(ctx, db.InsertPlaylistParams{
		Path:         sql.NullString{String: library.Dir, Valid: true},
		ExtractorKey: sql.NullString{String: "Calibre", Valid: true},
	}); err != nil {
		logging.Log.Warn("Failed to record Calibre library as a playlist root", "path", library.Dir, "error", err)
	}

	const batchSize = 500
	for start := 0; start < len(items); start += batchSize {
		end := min(start+batchSize, len(items))
		if err := queries.BulkUpsertMedia(ctx, items[start:end]); err != nil {
			return fmt.Errorf("failed to import Calibre media: %w", err)
		}
	}

	pruned := 0
	if c.Prune {
		pruned, err = pruneCalibreMedia(ctx, queries, library.Dir, importedPaths)
		if err != nil {
			return err
		}
	}

	if err := db.RebuildFTS(ctx, sqlDB, c.Database); err != nil {
		logging.Log.Error("Failed to rebuild FTS", "error", err)
	}
	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		logging.Log.Error("Failed to refresh folder_stats", "error", err)
	}

	logging.Log.Info("Calibre import complete",
		"imported", len(items),
		"skipped_missing", skipped,
		"pruned", pruned,
		"database", c.Database,
	)
	return nil
}

func buildCalibreMedia(
	books []calibre.Book,
	formatPriority []string,
	oneFormat bool,
	includeMissing bool,
) ([]db.UpsertMediaParams, map[string]bool, int) {
	priority := make(map[string]int, len(formatPriority))
	for i, format := range formatPriority {
		if _, exists := priority[format]; !exists {
			priority[format] = i
		}
	}

	now := time.Now().Unix()
	seen := make(map[string]bool)
	importedPaths := make(map[string]bool)
	totalFiles := 0
	for _, book := range books {
		totalFiles += len(book.Files)
	}
	items := make([]db.UpsertMediaParams, 0, totalFiles)
	skipped := 0

	type candidate struct {
		file     calibre.File
		size     int64
		modified int64
	}

	for _, book := range books {
		candidates := make([]candidate, 0, len(book.Files))
		for _, file := range book.Files {
			info, statErr := os.Stat(file.Path)
			if statErr != nil && !includeMissing {
				skipped++
				logging.Log.Debug("Skipping Calibre file missing on disk", "path", file.Path)
				continue
			}

			entry := candidate{file: file, size: file.Size, modified: now}
			if statErr == nil {
				entry.size = info.Size()
				entry.modified = info.ModTime().Unix()
			}
			candidates = append(candidates, entry)
		}

		slices.SortStableFunc(candidates, func(a, b candidate) int {
			return calibreFormatRank(a.file.Format, priority) - calibreFormatRank(b.file.Format, priority)
		})

		if oneFormat && len(candidates) > 1 {
			candidates = candidates[:1]
		}

		for _, entry := range candidates {
			if seen[entry.file.Path] {
				continue
			}

			seen[entry.file.Path] = true
			importedPaths[entry.file.Path] = true
			items = append(items, db.UpsertMediaParams{
				Path:           entry.file.Path,
				PathTokenized:  utils.ToNullString(textutil.PathToTokenized(entry.file.Path)),
				Title:          utils.ToNullString(book.Title),
				Size:           utils.ToNullInt64(entry.size),
				TimeCreated:    utils.ToNullInt64(entry.modified),
				TimeModified:   utils.ToNullInt64(entry.modified),
				MediaType:      utils.ToNullString(calibreMediaType(entry.file.Format)),
				Artist:         utils.ToNullString(calibreArtists(book.Authors)),
				Categories:     utils.ToNullString(calibreCategories(book.Tags)),
				Description:    utils.ToNullString(calibreDescription(book)),
				Language:       utils.ToNullString(book.Language),
				TimeDownloaded: utils.ToNullInt64(now),
				Score:          calibreScore(book.Rating),
			})
		}
	}

	return items, importedPaths, skipped
}

func calibreFormatRank(format string, priority map[string]int) int {
	if rank, ok := priority[strings.ToLower(format)]; ok {
		return rank
	}
	return len(priority)
}

func calibreMediaType(format string) string {
	ext := "." + strings.ToLower(format)
	switch {
	case media.TextExtensionMap[ext] || media.ComicExtensionMap[ext] || media.ArchiveExtensionMap[ext]:
		return "text"
	case media.ImageExtensionMap[ext]:
		return "image"
	case media.AudioExtensionMap[ext]:
		return "audio"
	case media.VideoExtensionMap[ext]:
		return "video"
	default:
		return "text"
	}
}

func calibreArtists(authors []string) string {
	if len(authors) == 0 {
		return "Unknown"
	}
	return strings.Join(authors, " & ")
}

func calibreCategories(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	cleaned := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ReplaceAll(tag, ";", ","))
		if tag != "" {
			cleaned[tag] = true
		}
	}
	if len(cleaned) == 0 {
		return ""
	}
	categories := make([]string, 0, len(cleaned))
	for category := range cleaned {
		categories = append(categories, category)
	}
	slices.Sort(categories)
	return ";" + strings.Join(categories, ";") + ";"
}

func calibreDescription(book calibre.Book) string {
	var parts []string
	if description := strings.TrimSpace(book.Description); description != "" {
		parts = append(parts, description)
	}
	if book.Series != "" {
		series := "Series: " + book.Series
		if book.HasSeriesIndex && book.SeriesIndex > 0 {
			series += " #" + strconv.FormatFloat(book.SeriesIndex, 'f', -1, 64)
		}
		parts = append(parts, series)
	}
	if book.Publisher != "" {
		parts = append(parts, "Publisher: "+book.Publisher)
	}
	return strings.Join(parts, "\n\n")
}

func calibreScore(rating float64) sql.NullFloat64 {
	if rating <= 0 {
		return sql.NullFloat64{}
	}
	// Calibre ratings are 0-10; discoteca stores 0-5 stars.
	return sql.NullFloat64{Float64: rating / 2, Valid: true}
}

func pruneCalibreMedia(
	ctx context.Context,
	queries *db.Queries,
	libraryDir string,
	importedPaths map[string]bool,
) (int, error) {
	allMedia, err := queries.GetAllMediaMetadata(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list media for pruning: %w", err)
	}

	prefix := filepath.Clean(libraryDir) + string(os.PathSeparator)
	now := time.Now().Unix()
	pruned := 0
	for _, row := range allMedia {
		if row.TimeDeleted.Int64 > 0 || importedPaths[row.Path] {
			continue
		}
		if !strings.HasPrefix(row.Path, prefix) {
			continue
		}
		logging.Log.Info("Marking media missing from the Calibre catalog as deleted", "path", row.Path)
		if err := queries.MarkDeleted(ctx, db.MarkDeletedParams{
			TimeDeleted: sql.NullInt64{Int64: now, Valid: true},
			Path:        row.Path,
		}); err != nil {
			return pruned, fmt.Errorf("failed to mark %s as deleted: %w", row.Path, err)
		}
		pruned++
	}
	return pruned, nil
}
