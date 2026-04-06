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

	Args                    []string `help:"Database file followed by paths to scan"                                  required:"true" name:"args" arg:""`
	Parallel                int      `help:"Number of parallel extractors (default: CPU count * 4)"                                                      short:"p"`
	ExtractText             bool     `help:"Extract full text from documents (PDF, EPUB, TXT, MD) for caption search"`
	OCR                     bool     `help:"Extract text from images using OCR (tesseract) for caption search"`
	OCREngine               string   `help:"OCR engine to use"                                                                                                     default:"tesseract" enum:"tesseract,paddle"`
	SpeechRecognition       bool     `help:"Extract speech-to-text from audio/video files for caption search"`
	SpeechRecognitionEngine string   `help:"Speech recognition engine to use"                                                                                      default:"vosk"      enum:"vosk,whisper"`

	ScanPaths []string `kong:"-"`
	Database  string   `kong:"-"`
}

type meta struct {
	size    int64
	mtime   int64
	deleted bool
}

type processState struct {
	completedJobs      atomic.Int64
	activeWorkers      atomic.Int32
	totalWorkerSamples int64
	workerSum          int64
	targetConcurrency  atomic.Int32
}

func (c *AddCmd) flushBatch(ctx context.Context, opts processMediaTypeOptions, batch []*metadata.MediaMetadata) error {
	if len(batch) == 0 {
		return nil
	}

	var mediaBatch []db.UpsertMediaParams
	var captionsBatch []db.InsertCaptionParams

	for _, res := range batch {
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

		tx, err := opts.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			lastErr = err
			continue
		}

		qtx := opts.queries.WithTx(tx)
		if upsertErr := qtx.BulkUpsertMedia(ctx, mediaBatch); upsertErr != nil {
			_ = tx.Rollback()
			lastErr = fmt.Errorf("bulk upsert media failed: %w", upsertErr)
			continue
		}
		if insertErr := qtx.BulkInsertCaptions(ctx, captionsBatch); insertErr != nil {
			_ = tx.Rollback()
			lastErr = fmt.Errorf("bulk insert captions failed: %w", insertErr)
			continue
		}

		if commitErr := tx.Commit(); commitErr != nil {
			lastErr = commitErr
			continue
		}

		return nil
	}

	return fmt.Errorf("commit failed after %d retries: %w", maxRetries, lastErr)
}

func (c *AddCmd) loadMetadataCache(ctx context.Context, queries *db.Queries, dbExists bool) (map[string]meta, error) {
	if !dbExists {
		return make(map[string]meta), nil
	}

	existingMedia, err := queries.GetAllMediaMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing metadata: %w", err)
	}
	metaCache := make(map[string]meta, len(existingMedia))
	for _, m := range existingMedia {
		metaCache[m.Path] = meta{
			size:    m.Size.Int64,
			mtime:   m.TimeModified.Int64,
			deleted: m.TimeDeleted.Int64 > 0,
		}
	}
	models.Log.Info("Loaded metadata cache from database", "count", len(metaCache))
	return metaCache, nil
}

func (c *AddCmd) printScanProgress(absRoot string, res fs.FindMediaResult) {
	if res.DirsCount%100 == 0 || res.FilesCount%100 == 0 || res.FilesCount == 1 {
		fmt.Printf(
			"\rScanning %s: %d files, %d folders found%s",
			absRoot,
			res.FilesCount,
			res.DirsCount,
			utils.ClearSeq,
		)
	}
}

func (c *AddCmd) matchesFilters(path string, stat os.FileInfo, flags models.GlobalFlags) bool {
	// Apply PathFilterFlags
	if !utils.FilterPath(path, flags.PathFilterFlags) {
		return false
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
			return false
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
			return false
		}
	}

	return true
}

func (c *AddCmd) collectFilesToProbe(
	absRoot string,
	metaCache map[string]meta,
	flags models.GlobalFlags,
	filter map[string]bool,
) (toProbe []string, newFilesFound bool, totalFiles, totalDirs, skipped int, err error) {
	foundFiles := make(chan fs.FindMediaResult, 100)
	var walkErr error
	go func() {
		defer close(foundFiles)
		walkErr = fs.FindMediaChan(absRoot, filter, foundFiles)
	}()

	for res := range foundFiles {
		totalFiles = res.FilesCount
		totalDirs = res.DirsCount

		c.printScanProgress(absRoot, res)

		if !c.matchesFilters(res.Path, res.Info, flags) {
			continue
		}

		if existing, ok := metaCache[res.Path]; ok {
			// Record exists, check if it's still valid
			if !existing.deleted && existing.size == res.Info.Size() && existing.mtime == res.Info.ModTime().Unix() {
				skipped++
				continue
			}
			// File exists but changed - will be updated, not new
		} else {
			// File not in cache - it's new
			newFilesFound = true
		}
		toProbe = append(toProbe, res.Path)
	}

	return toProbe, newFilesFound, totalFiles, totalDirs, skipped, walkErr
}

type processMediaTypeOptions struct {
	mediaType           fileMediaType
	sqlDB               *sql.DB
	queries             *db.Queries
	flags               models.GlobalFlags
	totalProcessedSoFar int
}

func (c *AddCmd) monitorConcurrency(
	ctx context.Context,
	state *processState,
	startWorker func(),
	monitorDone chan struct{},
) {
	ticker := time.NewTicker(4500 * time.Millisecond)
	defer ticker.Stop()

	var lastCompleted int64
	var lastThroughput int64
	direction := int32(1)

	for {
		select {
		case <-ctx.Done():
			close(monitorDone)
			return
		case <-ticker.C:
			completed := state.completedJobs.Load()
			throughput := completed - lastCompleted
			lastCompleted = completed

			current := state.targetConcurrency.Load()

			if throughput < lastThroughput {
				direction = -direction // Reverse direction if throughput drops
			} else if throughput == lastThroughput && throughput > 0 {
				direction = 1 // Gently push up if stable
			}

			newTarget := min(
				// Step by 2
				max(
					current+(direction*2), 1), 300)

			state.targetConcurrency.Store(newTarget)

			active := state.activeWorkers.Load()
			for active < newTarget {
				startWorker()
				active++
			}
			// Track worker statistics
			atomic.AddInt64(&state.workerSum, int64(active))
			atomic.AddInt64(&state.totalWorkerSamples, 1)
			lastThroughput = throughput
		case <-monitorDone:
			return
		}
	}
}

func (c *AddCmd) reportProgress(opts processMediaTypeOptions, count int, startTime time.Time, state *processState) {
	if count%10 == 0 || count == len(opts.mediaType.files) {
		etaStr := ""
		if count > 2 {
			elapsed := time.Since(startTime)
			estimatedTotal := time.Duration(
				float64(elapsed) / float64(count) * float64(len(opts.mediaType.files)),
			)
			remaining := (estimatedTotal - elapsed).Round(time.Second)
			if remaining > 0 {
				etaStr = fmt.Sprintf(" ETA: %v", remaining)
			}
		}

		typeTotal := opts.totalProcessedSoFar + count
		if c.Verbose > 0 {
			workers := state.activeWorkers.Load()
			if workers == 0 && state.totalWorkerSamples > 0 {
				avgWorkers := float64(state.workerSum) / float64(state.totalWorkerSamples)
				fmt.Printf(
					"\r  %s: Processed %d/%d files (avg: %.1f workers)%s%s",
					opts.mediaType.name,
					typeTotal,
					len(opts.mediaType.files),
					avgWorkers,
					etaStr,
					utils.ClearSeq,
				)
			} else {
				fmt.Printf(
					"\r  %s: Processed %d/%d files (%d workers)%s%s",
					opts.mediaType.name,
					typeTotal,
					len(opts.mediaType.files),
					workers,
					etaStr,
					utils.ClearSeq,
				)
			}
		} else {
			fmt.Printf(
				"\r  %s: Processed %d/%d files%s%s",
				opts.mediaType.name,
				typeTotal,
				len(opts.mediaType.files),
				etaStr,
				utils.ClearSeq,
			)
		}
	}
}

func (c *AddCmd) startExtractionWorkers(
	ctx context.Context,
	opts processMediaTypeOptions,
	jobs <-chan string,
	results chan<- *metadata.MediaMetadata,
	state *processState,
	wg *sync.WaitGroup,
) {
	worker := func() {
		state.activeWorkers.Add(1)
		defer state.activeWorkers.Add(-1)
		for {
			if state.activeWorkers.Load() > state.targetConcurrency.Load() {
				return // Scale down
			}
			select {
			case <-ctx.Done():
				return
			case path, ok := <-jobs:
				if !ok {
					return
				}
				res, extErr := metadata.Extract(ctx, path, metadata.ExtractOptions{
					ScanSubtitles:     opts.flags.ScanSubtitles,
					ExtractText:       c.ExtractText,
					OCR:               c.OCR,
					OCREngine:         c.OCREngine,
					SpeechRecognition: c.SpeechRecognition,
					SpeechRecEngine:   c.SpeechRecognitionEngine,
					ProbeImages:       c.ProbeImages,
				})
				if extErr != nil {
					models.Log.Error("\n  Metadata extraction failed", "path", path, "error", extErr)
				} else if res != nil {
					results <- res
				}
				state.completedJobs.Add(1)
			}
		}
	}

	for range state.targetConcurrency.Load() {
		wg.Go(worker)
	}

	// For dynamic scaling
	monitorDone := make(chan struct{})
	startOne := func() { wg.Go(worker) }
	go c.monitorConcurrency(ctx, state, startOne, monitorDone)
	go func() {
		wg.Wait()
		close(monitorDone)
		close(results)
	}()
}

func (c *AddCmd) writeResultsToDB(
	ctx context.Context,
	opts processMediaTypeOptions,
	results <-chan *metadata.MediaMetadata,
	startTime time.Time,
	state *processState,
) {
	count := 0
	batchSize := 500
	var currentBatch []*metadata.MediaMetadata

	for res := range results {
		currentBatch = append(currentBatch, res)

		if len(currentBatch) >= batchSize {
			if err := c.flushBatch(ctx, opts, currentBatch); err != nil {
				models.Log.Error("\n  Failed to commit batch", "error", err)
			}
			for i := range currentBatch {
				currentBatch[i] = nil
			}
			currentBatch = currentBatch[:0]
		}

		count++
		c.reportProgress(opts, count, startTime, state)
	}

	// Final flush
	if len(currentBatch) > 0 {
		if err := c.flushBatch(ctx, opts, currentBatch); err != nil {
			models.Log.Error("  Failed to commit final batch", "error", err)
		}
	}
}

func (c *AddCmd) processMediaType(ctx context.Context, opts processMediaTypeOptions) int {
	if len(opts.mediaType.files) == 0 {
		return 0
	}

	models.Log.Debug("  Processing media type", "mediaType", opts.mediaType.name, "count", len(opts.mediaType.files))

	startTime := time.Now()

	// Parallel extraction
	jobs := make(chan string, len(opts.mediaType.files))
	for _, f := range opts.mediaType.files {
		jobs <- f
	}
	close(jobs)

	// Larger buffer to decouple extraction from DB writes
	results := make(chan *metadata.MediaMetadata, 2000)
	var wg sync.WaitGroup

	state := &processState{}
	// Reset parallelism to initial value for each media type
	targetConcurrency := int32(c.Parallel)
	if targetConcurrency <= 0 {
		targetConcurrency = int32(runtime.NumCPU() * 4)
	}
	state.targetConcurrency.Store(targetConcurrency)

	c.startExtractionWorkers(ctx, opts, jobs, results, state, &wg)

	// Database writes
	c.writeResultsToDB(ctx, opts, results, startTime, state)

	fmt.Println()

	return len(opts.mediaType.files)
}

type scanRootOptions struct {
	root              string
	sqlDB             *sql.DB
	queries           *db.Queries
	metaCache         map[string]meta
	existingPlaylists []db.Playlists
	flags             models.GlobalFlags
}

func (c *AddCmd) processScanRoot(ctx context.Context, opts scanRootOptions) (bool, error) {
	fmt.Printf("\n%s\n", strings.Repeat("#", 60))

	absRoot, err := filepath.Abs(opts.root)
	if err != nil {
		models.Log.Error("Failed to get absolute path", "path", opts.root, "error", err)
		return false, nil
	}

	// Check if this path is a child of an existing playlist root
	absRootSlash := filepath.ToSlash(absRoot)
	for _, pl := range opts.existingPlaylists {
		if pl.Path.Valid {
			plPathSlash := filepath.ToSlash(pl.Path.String)
			if strings.HasPrefix(absRootSlash, plPathSlash+"/") {
				models.Log.Info(
					"Path is child of existing scan root, skipping",
					"path",
					absRoot,
					"root",
					pl.Path.String,
				)
				return false, nil
			}
		}
	}

	// Record or update this scan root
	if _, playlistErr := opts.queries.InsertPlaylist(ctx, db.InsertPlaylistParams{
		Path:         sql.NullString{String: absRoot, Valid: true},
		ExtractorKey: sql.NullString{String: "Local", Valid: true},
	}); playlistErr != nil {
		models.Log.Warn("Failed to insert playlist root", "path", absRoot, "error", playlistErr)
	}

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

	var toProbe []string
	var newFilesFound bool
	var totalFiles, totalDirs, skipped int
	toProbe, newFilesFound, totalFiles, totalDirs, skipped, err = c.collectFilesToProbe(
		absRoot,
		opts.metaCache,
		opts.flags,
		filter,
	)
	if err != nil {
		return false, err
	}

	// Print scanning summary
	fmt.Printf("\rScan of %s found %d files in %d folders%s\n", absRoot, totalFiles, totalDirs, utils.ClearSeq)
	if skipped > 0 {
		models.Log.Info("  Skipped unchanged files", "count", skipped)
	}

	if len(toProbe) == 0 {
		return newFilesFound, nil
	}

	if c.Simulate {
		fmt.Printf("  (Simulated) would process %d new files\n", len(toProbe))
		return newFilesFound, nil
	}

	models.Log.Info("  Extracting metadata", "count", len(toProbe), "initial_parallelism", c.Parallel)

	// Group files by media type for separate processing with accurate ETA per media type
	mediaTypes := groupFilesByMediaType(toProbe)

	totalProcessed := 0
	for _, mediaType := range mediaTypes {
		processed := c.processMediaType(ctx, processMediaTypeOptions{
			mediaType:           mediaType,
			sqlDB:               opts.sqlDB,
			queries:             opts.queries,
			flags:               opts.flags,
			totalProcessedSoFar: totalProcessed,
		})
		totalProcessed += processed
	}

	return newFilesFound, nil
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
	if !isDB {
		return fmt.Errorf("first argument must be a database file (e.g. .db): %s", c.Args[0])
	}
	c.Database = c.Args[0]
	c.ScanPaths = c.Args[1:]

	if c.Parallel <= 0 {
		c.Parallel = runtime.NumCPU() * 4
	}
	return nil
}

func (c *AddCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	db.InitFtsConfig()
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
	metaCache, err := c.loadMetadataCache(runCtx, queries, dbExists)
	if err != nil {
		return err
	}

	// Track if we add new files across all scan paths (for folder_stats refresh)
	var newFilesAdded bool
	var newFound bool

	for _, root := range c.ScanPaths {
		newFound, err = c.processScanRoot(runCtx, scanRootOptions{
			root:              root,
			sqlDB:             sqlDB,
			queries:           queries,
			metaCache:         metaCache,
			existingPlaylists: existingPlaylists,
			flags:             flags,
		})
		if err != nil {
			return err
		}
		if newFound {
			newFilesAdded = true
		}
	}

	fmt.Println()
	// Refresh FTS after adding new media (always needed for search)
	if err := db.RebuildFTS(ctx, sqlDB, dbPath); err != nil {
		models.Log.Error("Failed to rebuild FTS", "error", err)
	}

	// Only refresh folder_stats if new files were added
	if newFilesAdded {
		models.Log.Info("Refreshing folder_stats after adding new files...")
		if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
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
