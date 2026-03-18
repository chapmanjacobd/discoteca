package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/fs"
	"github.com/chapmanjacobd/discoteca/internal/metadata"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type AddCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`

	Args                    []string `arg:"" name:"args" required:"" help:"Database file followed by paths to scan"`
	Parallel                int      `short:"p" help:"Number of parallel extractors (default: CPU count * 4)"`
	ExtractText             bool     `help:"Extract full text from documents (PDF, EPUB, TXT, MD) for caption search"`
	OCR                     bool     `help:"Extract text from images using OCR (tesseract) for caption search"`
	OCREngine               string   `default:"tesseract" enum:"tesseract,paddle" help:"OCR engine to use"`
	SpeechRecognition       bool     `help:"Extract speech-to-text from audio/video files for caption search"`
	SpeechRecognitionEngine string   `default:"vosk" enum:"vosk,whisper" help:"Speech recognition engine to use"`

	ScanPaths []string `kong:"-"`
	Database  string   `kong:"-"`
}

func (c *AddCmd) AfterApply() error {
	if err := c.CoreFlags.AfterApply(); err != nil {
		return err
	}
	if err := c.MediaFilterFlags.AfterApply(); err != nil {
		return err
	}
	if len(c.Args) < 2 {
		return fmt.Errorf("at least one database file and one path to scan are required")
	}

	// Smart DB detection: first arg MUST be a database for 'add'
	isDB := strings.HasSuffix(c.Args[0], ".db") && (utils.IsSQLite(c.Args[0]) || !utils.FileExists(c.Args[0]))
	if isDB {
		c.Database = c.Args[0]
		c.ScanPaths = c.Args[1:]
	} else {
		return fmt.Errorf("first argument must be a database file (e.g. .db): %s", c.Args[0])
	}

	if c.Parallel <= 0 {
		c.Parallel = runtime.NumCPU() * 4
	}
	return nil
}

func (c *AddCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
	}
	dbPath := c.Database
	c.ScanPaths = utils.ExpandStdin(c.ScanPaths)

	dbExists := utils.FileExists(dbPath)
	sqlDB, queries, err := db.ConnectWithInit(dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Step 0: Load existing playlists (roots) to avoid redundant scans
	existingPlaylists, _ := queries.GetPlaylists(context.Background())

	// Step 1: Load all existing metadata into memory for O(1) checks
	existingMedia, err := queries.GetAllMediaMetadata(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load existing metadata: %w", err)
	}

	type meta struct {
		size    int64
		mtime   int64
		deleted bool
	}
	metaCache := make(map[string]meta, len(existingMedia))
	for _, m := range existingMedia {
		metaCache[m.Path] = meta{
			size:    m.Size.Int64,
			mtime:   m.TimeModified.Int64,
			deleted: m.TimeDeleted.Int64 > 0,
		}
	}
	if dbExists {
		slog.Info("Loaded metadata cache from database", "count", len(metaCache))
	}

	for _, root := range c.ScanPaths {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			slog.Error("Failed to get absolute path", "path", root, "error", err)
			continue
		}

		// Check if this path is a child of an existing playlist root
		// We allow re-scanning the same path, but block child directories
		isChildPath := false
		absRootSlash := filepath.ToSlash(absRoot)
		for _, pl := range existingPlaylists {
			if pl.Path.Valid {
				plPathSlash := filepath.ToSlash(pl.Path.String)
				// Check if absRoot is a strict subpath (child) of existing root
				// strings.HasPrefix with "/" suffix ensures we match directory boundaries
				// e.g., /home/xk/sync is NOT a child of /home/xk/sync/audio
				// but /home/xk/sync/audio IS a child of /home/xk/sync
				if strings.HasPrefix(absRootSlash, plPathSlash+"/") {
					slog.Info("Path is child of existing scan root, skipping", "path", absRoot, "root", pl.Path.String)
					isChildPath = true
					break
				}
			}
		}
		if isChildPath {
			continue
		}

		// Record or update this scan root
		// If path already exists as a playlist, this will be a no-op for the insert
		// The actual scan logic below will process the files
		queries.InsertPlaylist(context.Background(), db.InsertPlaylistParams{
			Path:         sql.NullString{String: absRoot, Valid: true},
			ExtractorKey: sql.NullString{String: "Local", Valid: true},
		})

		var filter map[string]bool
		if c.VideoOnly || c.AudioOnly {
			filter = make(map[string]bool)
			if c.VideoOnly {
				maps.Copy(filter, utils.VideoExtensionMap)
			}
			if c.AudioOnly {
				maps.Copy(filter, utils.AudioExtensionMap)
			}
		}

		slog.Info("Scanning", "path", absRoot)
		foundFiles, err := fs.FindMedia(absRoot, filter)
		if err != nil {
			return err
		}

		// Apply PathFilterFlags
		filteredFiles := make(map[string]os.FileInfo)
		for path, stat := range foundFiles {
			if !utils.FilterPath(path, flags.PathFilterFlags) {
				continue
			}

			// Apply Size filter
			if len(c.Size) > 0 {
				matched := false
				for _, s := range c.Size {
					if r, err := utils.ParseRange(s, utils.HumanToBytes); err == nil {
						if r.Matches(stat.Size()) {
							matched = true
							break
						}
					}
				}
				if !matched {
					continue
				}
			}

			// Apply MimeType filter
			if len(c.MimeType) > 0 || len(c.NoMimeType) > 0 {
				mime := utils.DetectMimeType(path)
				if len(c.MimeType) > 0 {
					matched := false
					for _, m := range c.MimeType {
						if strings.Contains(mime, m) {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
				}
				if len(c.NoMimeType) > 0 {
					excluded := false
					for _, m := range c.NoMimeType {
						if strings.Contains(mime, m) {
							excluded = true
							break
						}
					}
					if excluded {
						continue
					}
				}
			}

			filteredFiles[path] = stat
		}
		foundFiles = filteredFiles

		if dbExists {
			slog.Info("Checking for updates", "count", len(foundFiles))
		}

		// Step 2: Identify which files actually need probing using the cache
		var toProbe []string
		skipped := 0
		for path, stat := range foundFiles {
			if len(c.Ext) > 0 {
				matched := false
				for _, e := range c.Ext {
					if strings.EqualFold(filepath.Ext(path), e) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}

			if existing, ok := metaCache[path]; ok {
				// Record exists, check if it's still valid
				if !existing.deleted && existing.size == stat.Size() && existing.mtime == stat.ModTime().Unix() {
					skipped++
					continue
				}
			}
			toProbe = append(toProbe, path)
		}

		if skipped > 0 {
			slog.Info("Skipped unchanged files", "count", skipped)
		}

		if len(toProbe) == 0 {
			fmt.Printf("Processed %d/%d files\n", skipped, skipped)
			continue
		}

		if c.Simulate {
			fmt.Printf("Simulated: would process %d new files\n", len(toProbe))
			continue
		}

		slog.Info("Extracting metadata", "count", len(toProbe), "parallelism", c.Parallel)

		// Parallel extraction
		jobs := make(chan string, len(toProbe))
		results := make(chan *metadata.MediaMetadata, len(toProbe))
		var wg sync.WaitGroup

		for i := 0; i < c.Parallel; i++ {
			wg.Go(func() {
				for path := range jobs {
					res, err := metadata.Extract(context.Background(), path, flags.ScanSubtitles, c.ExtractText, c.OCR, c.OCREngine, c.SpeechRecognition, c.SpeechRecognitionEngine)
					if err != nil {
						slog.Error("Metadata extraction failed", "path", path, "error", err)
						continue
					}
					results <- res
				}
			})
		}

		go func() {
			for _, f := range toProbe {
				jobs <- f
			}
			close(jobs)
		}()

		go func() {
			wg.Wait()
			close(results)
		}()

		count := 0
		batchSize := 100
		var currentBatch []*metadata.MediaMetadata

		flush := func() error {
			if len(currentBatch) == 0 {
				return nil
			}
			tx, err := sqlDB.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()

			qtx := queries.WithTx(tx)
			for _, res := range currentBatch {
				if err := qtx.UpsertMedia(context.Background(), res.Media); err != nil {
					slog.Error("Database upsert failed", "path", res.Media.Path, "error", err)
				}
				for _, cap := range res.Captions {
					if err := qtx.InsertCaption(context.Background(), cap); err != nil {
						slog.Error("Caption insertion failed", "path", res.Media.Path, "error", err)
					}
				}
			}
			if err := tx.Commit(); err != nil {
				return err
			}

			return nil
		}

		for res := range results {
			currentBatch = append(currentBatch, res)

			if len(currentBatch) >= batchSize {
				if err := flush(); err != nil {
					slog.Error("Failed to commit batch", "error", err)
				}
				currentBatch = currentBatch[:0]
			}

			count++
			if count%10 == 0 || count == len(toProbe) {
				fmt.Printf("\rProcessed %d/%d files", count, len(toProbe))
			}
		}
		// Final flush
		if err := flush(); err != nil {
			slog.Error("Failed to commit final batch", "error", err)
		}
		fmt.Println()
	}

	return nil
}
