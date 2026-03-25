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
	"sync/atomic"
	"time"

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
	fileInfo, err := os.Stat(c.Args[0])
	isEmpty := err == nil && fileInfo.Size() == 0
	isDB := strings.HasSuffix(c.Args[0], ".db") && (utils.IsSQLite(c.Args[0]) || os.IsNotExist(err) || isEmpty)
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
	db.SetFtsEnabled(true)

	dbPath := c.Database
	c.ScanPaths = utils.ExpandStdin(c.ScanPaths)

	dbExists := utils.FileExists(dbPath)
	sqlDB, queries, err := db.ConnectWithInit(dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
	}

	// Step 0: Load existing playlists (roots) to avoid redundant scans
	existingPlaylists, _ := queries.GetPlaylists(context.Background())

	// Step 1: Load existing metadata for O(1) cache checks
	// For large libraries, we load all metadata once at startup
	type meta struct {
		size    int64
		mtime   int64
		deleted bool
	}
	var metaCache map[string]meta
	if dbExists {
		existingMedia, err := queries.GetAllMediaMetadata(context.Background())
		if err != nil {
			return fmt.Errorf("failed to load existing metadata: %w", err)
		}
		metaCache = make(map[string]meta, len(existingMedia))
		for _, m := range existingMedia {
			metaCache[m.Path] = meta{
				size:    m.Size.Int64,
				mtime:   m.TimeModified.Int64,
				deleted: m.TimeDeleted.Int64 > 0,
			}
		}
		existingMedia = nil // Allow GC
		slog.Info("Loaded metadata cache from database", "count", len(metaCache))
	} else {
		metaCache = make(map[string]meta)
	}

	// Track if we add new files across all scan paths (for folder_stats refresh)
	var newFilesAdded bool

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
		if c.VideoOnly || c.AudioOnly || c.ImageOnly || c.TextOnly {
			filter = make(map[string]bool)
			if c.VideoOnly {
				maps.Copy(filter, utils.VideoExtensionMap)
			}
			if c.AudioOnly {
				maps.Copy(filter, utils.AudioExtensionMap)
			}
			if c.ImageOnly {
				maps.Copy(filter, utils.ImageExtensionMap)
			}
			if c.TextOnly {
				maps.Copy(filter, utils.TextExtensionMap)
				maps.Copy(filter, utils.ComicExtensionMap)
			}
		}

		foundFiles := make(chan fs.FindMediaResult, 100)
		var walkErr error
		var totalFiles, totalDirs int
		go func() {
			defer close(foundFiles)
			walkErr = fs.FindMediaChan(absRoot, filter, foundFiles)
		}()

		// Step 2: Identify which files actually need probing using the cache
		var toProbe []string
		skipped := 0
		for res := range foundFiles {
			path := res.Path
			stat := res.Info
			totalFiles = res.FilesCount
			totalDirs = res.DirsCount

			// Print progress counter during scanning
			if res.DirsCount%100 == 0 || res.FilesCount%100 == 0 || res.FilesCount == 1 {
				fmt.Printf("\rScanning %s: %d files, %d folders found%s", absRoot, res.FilesCount, res.DirsCount, utils.ClearSeq)
			}

			// Apply PathFilterFlags
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
				// File exists but changed - will be updated, not new
			} else {
				// File not in cache - it's new
				newFilesAdded = true
			}
			toProbe = append(toProbe, path)
		}
		if walkErr != nil {
			return walkErr
		}

		// Print final scanning summary
		fmt.Printf("\rScan of %s found %d files in %d folders%s\n", absRoot, totalFiles, totalDirs, utils.ClearSeq)

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

		slog.Info("Extracting metadata", "count", len(toProbe), "initial_parallelism", c.Parallel)

		startTime := time.Now()

		// Parallel extraction
		jobs := make(chan string, len(toProbe))
		for _, f := range toProbe {
			jobs <- f
		}
		close(jobs)

		// Larger buffer to decouple extraction from DB writes
		results := make(chan *metadata.MediaMetadata, 2000)
		var wg sync.WaitGroup

		var completedJobs int64
		var activeWorkers int32
		var totalWorkerSamples int64
		var workerSum int64
		targetConcurrency := int32(c.Parallel)
		if targetConcurrency <= 0 {
			targetConcurrency = int32(runtime.NumCPU() * 4)
		}

		startWorker := func() {
			wg.Go(func() {
				atomic.AddInt32(&activeWorkers, 1)
				defer atomic.AddInt32(&activeWorkers, -1)
				for {
					if atomic.LoadInt32(&activeWorkers) > atomic.LoadInt32(&targetConcurrency) {
						return // Scale down
					}
					path, ok := <-jobs
					if !ok {
						return
					}
					res, err := metadata.Extract(context.Background(), path, metadata.ExtractOptions{
						ScanSubtitles:     flags.ScanSubtitles,
						ExtractText:       c.ExtractText,
						OCR:               c.OCR,
						OCREngine:         c.OCREngine,
						SpeechRecognition: c.SpeechRecognition,
						SpeechRecEngine:   c.SpeechRecognitionEngine,
						ProbeImages:       c.ProbeImages,
					})
					if err != nil {
						slog.Error("Metadata extraction failed", "path", path, "error", err)
					} else if res != nil {
						results <- res
					}
					atomic.AddInt64(&completedJobs, 1)
				}
			})
		}

		for i := int32(0); i < targetConcurrency; i++ {
			startWorker()
		}

		monitorDone := make(chan struct{})
		go func() {
			ticker := time.NewTicker(4500 * time.Millisecond)
			defer ticker.Stop()

			var lastCompleted int64
			var lastThroughput int64
			direction := int32(1)

			for {
				select {
				case <-ticker.C:
					completed := atomic.LoadInt64(&completedJobs)
					throughput := completed - lastCompleted
					lastCompleted = completed

					current := atomic.LoadInt32(&targetConcurrency)

					if throughput < lastThroughput {
						direction = -direction // Reverse direction if throughput drops
					} else if throughput == lastThroughput && throughput > 0 {
						direction = 1 // Gently push up if stable
					}

					newTarget := min(
						// Step by 2
						max(

							current+(direction*2), 1), 1000)

					atomic.StoreInt32(&targetConcurrency, newTarget)

					active := atomic.LoadInt32(&activeWorkers)
					for active < newTarget {
						startWorker()
						active++
					}
					// Track worker statistics
					atomic.AddInt64(&workerSum, int64(active))
					atomic.AddInt64(&totalWorkerSamples, 1)
					lastThroughput = throughput
				case <-monitorDone:
					return
				}
			}
		}()

		// Separate goroutine for database writes to avoid blocking extraction workers
		dbWriteDone := make(chan struct{})
		go func() {
			defer close(dbWriteDone)
			count := 0
			batchSize := 500
			var currentBatch []*metadata.MediaMetadata

			flush := func() error {
				if len(currentBatch) == 0 {
					return nil
				}

				var mediaBatch []db.UpsertMediaParams
				var captionsBatch []db.InsertCaptionParams

				for _, res := range currentBatch {
					mediaBatch = append(mediaBatch, res.Media)
					captionsBatch = append(captionsBatch, res.Captions...)
				}

				tx, err := sqlDB.BeginTx(context.Background(), nil)
				if err != nil {
					return err
				}
				defer tx.Rollback()

				qtx := queries.WithTx(tx)
				if err := qtx.BulkUpsertMedia(context.Background(), mediaBatch); err != nil {
					return fmt.Errorf("bulk upsert media failed: %w", err)
				}
				if err := qtx.BulkInsertCaptions(context.Background(), captionsBatch); err != nil {
					return fmt.Errorf("bulk insert captions failed: %w", err)
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
					for i := range currentBatch {
						currentBatch[i] = nil
					}
					currentBatch = currentBatch[:0]
				}

				count++
				if count%10 == 0 || count == len(toProbe) {
					etaStr := ""
					if count > 2 {
						elapsed := time.Since(startTime)
						estimatedTotal := time.Duration(float64(elapsed) / float64(count) * float64(len(toProbe)))
						remaining := (estimatedTotal - elapsed).Round(time.Second)
						if remaining > 0 {
							etaStr = fmt.Sprintf(" ETA: %v", remaining)
						}
					}

					if c.Verbose > 0 {
						workers := atomic.LoadInt32(&activeWorkers)
						if workers == 0 && totalWorkerSamples > 0 {
							avgWorkers := float64(workerSum) / float64(totalWorkerSamples)
							fmt.Printf("\rProcessed %d/%d files (avg: %.1f workers)%s%s", count, len(toProbe), avgWorkers, etaStr, utils.ClearSeq)
						} else {
							fmt.Printf("\rProcessed %d/%d files (%d workers)%s%s", count, len(toProbe), workers, etaStr, utils.ClearSeq)
						}
					} else {
						fmt.Printf("\rProcessed %d/%d files%s%s", count, len(toProbe), etaStr, utils.ClearSeq)
					}
				}
			}
			// Final flush
			if err := flush(); err != nil {
				slog.Error("Failed to commit final batch", "error", err)
			}
			for i := range currentBatch {
				currentBatch[i] = nil
			}
			currentBatch = currentBatch[:0]
		}()

		// Wait for extraction to complete
		go func() {
			wg.Wait()
			close(monitorDone)
			close(results)
		}()

		// Wait for DB writes to complete
		<-dbWriteDone
		fmt.Println()
	}

	// Refresh FTS after adding new media (always needed for search)
	if err := db.RebuildFTS(sqlDB, dbPath); err != nil {
		slog.Error("Failed to rebuild FTS", "error", err)
	}

	// Only refresh folder_stats if new files were added
	if newFilesAdded {
		slog.Info("Refreshing folder_stats after adding new files...")
		if err := db.RefreshFolderStats(sqlDB); err != nil {
			slog.Error("Failed to refresh folder_stats", "error", err)
		}
	} else {
		slog.Debug("No new files added, skipping folder_stats refresh")
	}

	return nil
}
