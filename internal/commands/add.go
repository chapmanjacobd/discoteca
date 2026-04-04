package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

	Args                    []string `help:"Database file followed by paths to scan"                                  required:"" name:"args" arg:""`
	Parallel                int      `help:"Number of parallel extractors (default: CPU count * 4)"                                                  short:"p"`
	ExtractText             bool     `help:"Extract full text from documents (PDF, EPUB, TXT, MD) for caption search"`
	OCR                     bool     `help:"Extract text from images using OCR (tesseract) for caption search"`
	OCREngine               string   `help:"OCR engine to use"                                                                                                 default:"tesseract" enum:"tesseract,paddle"`
	SpeechRecognition       bool     `help:"Extract speech-to-text from audio/video files for caption search"`
	SpeechRecognitionEngine string   `help:"Speech recognition engine to use"                                                                                  default:"vosk"      enum:"vosk,whisper"`

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
		return errors.New("at least one database file and one path to scan are required")
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

func (c *AddCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	db.SetFtsEnabled(true)

	dbPath := c.Database
	c.ScanPaths = utils.ExpandStdin(c.ScanPaths)

	dbExists := utils.FileExists(dbPath)
	sqlDB, queries, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Create a context that can be cancelled for all operations
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
	}

	// Step 0: Load existing playlists (roots) to avoid redundant scans
	existingPlaylists, _ := queries.GetPlaylists(runCtx)

	// Step 1: Load existing metadata for O(1) cache checks
	// For large libraries, we load all metadata once at startup
	type meta struct {
		size    int64
		mtime   int64
		deleted bool
	}
	var metaCache map[string]meta
	if dbExists {
		existingMedia, err := queries.GetAllMediaMetadata(runCtx)
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
		models.Log.Info("Loaded metadata cache from database", "count", len(metaCache))
	} else {
		metaCache = make(map[string]meta)
	}

	// Track if we add new files across all scan paths (for folder_stats refresh)
	var newFilesAdded bool

	for _, root := range c.ScanPaths {
		fmt.Printf("\n%s\n", strings.Repeat("#", 60))

		absRoot, err := filepath.Abs(root)
		if err != nil {
			models.Log.Error("Failed to get absolute path", "path", root, "error", err)
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
					models.Log.Info(
						"Path is child of existing scan root, skipping",
						"path",
						absRoot,
						"root",
						pl.Path.String,
					)
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
		queries.InsertPlaylist(runCtx, db.InsertPlaylistParams{
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
				fmt.Printf(
					"\rScanning %s: %d files, %d folders found%s",
					absRoot,
					res.FilesCount,
					res.DirsCount,
					utils.ClearSeq,
				)
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

		// Print scanning summary
		fmt.Printf("\rScan of %s found %d files in %d folders%s\n", absRoot, totalFiles, totalDirs, utils.ClearSeq)
		if skipped > 0 {
			models.Log.Info("  Skipped unchanged files", "count", skipped)
		}

		if len(toProbe) == 0 {
			continue
		}

		// Group files by media type for separate processing with accurate ETA per media type
		mediaTypes := groupFilesByMediaType(toProbe)

		if c.Simulate {
			fmt.Printf("  (Simulated) would process %d new files\n", len(toProbe))
			continue
		}

		models.Log.Info("  Extracting metadata", "count", len(toProbe), "initial_parallelism", c.Parallel)

		// Process each media type separately for more accurate ETA
		totalProcessed := 0
		for _, mediaType := range mediaTypes {
			if len(mediaType.files) == 0 {
				continue
			}

			models.Log.Debug("  Processing media type", "mediaType", mediaType.name, "count", len(mediaType.files))

			startTime := time.Now()

			// Parallel extraction
			jobs := make(chan string, len(mediaType.files))
			for _, f := range mediaType.files {
				jobs <- f
			}
			close(jobs)

			// Larger buffer to decouple extraction from DB writes
			results := make(chan *metadata.MediaMetadata, 2000)
			var wg sync.WaitGroup

			var completedJobs atomic.Int64
			var activeWorkers atomic.Int32
			var totalWorkerSamples int64
			var workerSum int64
			// Reset parallelism to initial value for each media type
			targetConcurrency := int32(c.Parallel)
			if targetConcurrency <= 0 {
				targetConcurrency = int32(runtime.NumCPU() * 4)
			}

			startWorker := func() {
				wg.Go(func() {
					activeWorkers.Add(1)
					defer activeWorkers.Add(-1)
					for {
						if activeWorkers.Load() > atomic.LoadInt32(&targetConcurrency) {
							return // Scale down
						}
						select {
						case <-runCtx.Done():
							return
						case path, ok := <-jobs:
							if !ok {
								return
							}
							res, err := metadata.Extract(runCtx, path, metadata.ExtractOptions{
								ScanSubtitles:     flags.ScanSubtitles,
								ExtractText:       c.ExtractText,
								OCR:               c.OCR,
								OCREngine:         c.OCREngine,
								SpeechRecognition: c.SpeechRecognition,
								SpeechRecEngine:   c.SpeechRecognitionEngine,
								ProbeImages:       c.ProbeImages,
							})
							if err != nil {
								models.Log.Error("\n  Metadata extraction failed", "path", path, "error", err)
							} else if res != nil {
								results <- res
							}
							completedJobs.Add(1)
						}
					}
				})
			}

			for range targetConcurrency {
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
					case <-runCtx.Done():
						close(monitorDone)
						return
					case <-ticker.C:
						completed := completedJobs.Load()
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
								current+(direction*2), 1), 300)

						atomic.StoreInt32(&targetConcurrency, newTarget)

						active := activeWorkers.Load()
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

					// Retry logic for "database is locked" errors
					const maxRetries = 10
					var lastErr error
					for attempt := range maxRetries {
						if attempt > 0 {
							// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1.6s, 3.2s, 6.4s, 12.8s, 25.6s
							backoff := min(time.Duration(100*(1<<attempt))*time.Millisecond, 30*time.Second)
							time.Sleep(backoff)
						}

						tx, err := sqlDB.BeginTx(runCtx, nil)
						if err != nil {
							lastErr = err
							continue
						}

						qtx := queries.WithTx(tx)
						if err := qtx.BulkUpsertMedia(runCtx, mediaBatch); err != nil {
							tx.Rollback()
							lastErr = fmt.Errorf("bulk upsert media failed: %w", err)
							continue
						}
						if err := qtx.BulkInsertCaptions(runCtx, captionsBatch); err != nil {
							tx.Rollback()
							lastErr = fmt.Errorf("bulk insert captions failed: %w", err)
							continue
						}

						if err := tx.Commit(); err != nil {
							lastErr = err
							continue
						}

						return nil
					}

					return fmt.Errorf("commit failed after %d retries: %w", maxRetries, lastErr)
				}

				for res := range results {
					currentBatch = append(currentBatch, res)

					if len(currentBatch) >= batchSize {
						if err := flush(); err != nil {
							models.Log.Error("\n  Failed to commit batch", "error", err)
						}
						for i := range currentBatch {
							currentBatch[i] = nil
						}
						currentBatch = currentBatch[:0]
					}

					count++
					if count%10 == 0 || count == len(mediaType.files) {
						etaStr := ""
						if count > 2 {
							elapsed := time.Since(startTime)
							estimatedTotal := time.Duration(
								float64(elapsed) / float64(count) * float64(len(mediaType.files)),
							)
							remaining := (estimatedTotal - elapsed).Round(time.Second)
							if remaining > 0 {
								etaStr = fmt.Sprintf(" ETA: %v", remaining)
							}
						}

						typeTotal := totalProcessed + count
						if c.Verbose > 0 {
							workers := activeWorkers.Load()
							if workers == 0 && totalWorkerSamples > 0 {
								avgWorkers := float64(workerSum) / float64(totalWorkerSamples)
								fmt.Printf(
									"\r  %s: Processed %d/%d files (avg: %.1f workers)%s%s",
									mediaType.name,
									typeTotal,
									len(mediaType.files),
									avgWorkers,
									etaStr,
									utils.ClearSeq,
								)
							} else {
								fmt.Printf(
									"\r  %s: Processed %d/%d files (%d workers)%s%s",
									mediaType.name,
									typeTotal,
									len(mediaType.files),
									workers,
									etaStr,
									utils.ClearSeq,
								)
							}
						} else {
							fmt.Printf(
								"\r  %s: Processed %d/%d files%s%s",
								mediaType.name,
								typeTotal,
								len(mediaType.files),
								etaStr,
								utils.ClearSeq,
							)
						}
					}
				}
				// Final flush
				if err := flush(); err != nil {
					models.Log.Error("  Failed to commit final batch", "error", err)
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

			totalProcessed += len(mediaType.files)
		}
	}

	fmt.Println()
	// Refresh FTS after adding new media (always needed for search)
	if err := db.RebuildFTS(sqlDB, dbPath); err != nil {
		models.Log.Error("Failed to rebuild FTS", "error", err)
	}

	// Only refresh folder_stats if new files were added
	if newFilesAdded {
		models.Log.Info("Refreshing folder_stats after adding new files...")
		if err := db.RefreshFolderStats(sqlDB); err != nil {
			models.Log.Error("Failed to refresh folder_stats", "error", err)
		}
	} else {
		models.Log.Debug("No new files added, skipping folder_stats refresh")
	}

	return nil
}

// fileMediaType represents a media type for processing
type fileMediaType struct {
	name  string
	files []string
}

// groupFilesByMediaType groups files by their media type for separate processing with accurate ETA
func groupFilesByMediaType(paths []string) []fileMediaType {
	mediaTypes := []fileMediaType{
		{name: "non-media", files: make([]string, 0)},
		{name: "text", files: make([]string, 0)},
		{name: "images", files: make([]string, 0)},
		{name: "video", files: make([]string, 0)},
		{name: "audio", files: make([]string, 0)},
	}

	for _, path := range paths {
		ext := strings.ToLower(filepath.Ext(path))
		switch {
		case utils.TextExtensionMap[ext] || utils.ComicExtensionMap[ext]:
			mediaTypes[1].files = append(mediaTypes[1].files, path)
		case utils.ImageExtensionMap[ext]:
			mediaTypes[2].files = append(mediaTypes[2].files, path)
		case utils.VideoExtensionMap[ext]:
			mediaTypes[3].files = append(mediaTypes[3].files, path)
		case utils.AudioExtensionMap[ext]:
			mediaTypes[4].files = append(mediaTypes[4].files, path)
		default:
			mediaTypes[0].files = append(mediaTypes[0].files, path)
		}
	}

	return mediaTypes
}
