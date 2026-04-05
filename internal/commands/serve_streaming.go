package commands

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

const HlsSegmentDuration = 10

func (c *ServeCmd) HandleTranscode(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	m models.Media,
	strategy utils.TranscodeStrategy,
) {
	w.Header().Set("Content-Type", strategy.TargetMime)
	// Note: We don't support HTTP Range requests for transcoded content.
	// Seeking is handled via the "start" query parameter instead.
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(path)))

	start := r.URL.Query().Get("start")

	// Add flags to help with piped streaming duration and timestamp issues
	var args []string

	if start != "" {
		args = append(args, "-ss", start)
	}

	args = append(args, "-fflags", "+genpts", "-i", path)

	// If we have duration in metadata, tell ffmpeg so it can write it to headers
	if m.Duration != nil && *m.Duration > 0 {
		args = append(args, "-t", strconv.FormatInt(*m.Duration, 10))
	}

	if strategy.VideoCopy {
		args = append(args, "-c:v", "copy")
	} else {
		if strategy.TargetMime == "video/mp4" {
			args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-crf", "28")
		} else {
			// WebM
			args = append(
				args,
				"-c:v",
				"libvpx-vp9",
				"-deadline",
				"realtime",
				"-cpu-used",
				"8",
				"-crf",
				"30",
				"-b:v",
				"0",
			)
		}
	}

	if strategy.AudioCopy {
		args = append(args, "-c:a", "copy")
	} else {
		if strategy.TargetMime == "video/mp4" {
			args = append(args, "-c:a", "aac", "-b:a", "128k", "-ac", "2")
		} else {
			// WebM supports Opus
			args = append(args, "-c:a", "libopus", "-b:a", "128k", "-ac", "2")
		}
	}

	args = append(args, "-avoid_negative_ts", "make_zero", "-map_metadata", "-1", "-sn")

	if strategy.TargetMime == "video/mp4" {
		// frag_keyframe+empty_moov+default_base_moof+global_sidx is the standard for fragmented streaming
		args = append(
			args,
			"-f",
			"mp4",
			"-movflags",
			"frag_keyframe+empty_moov+default_base_moof+global_sidx",
			"pipe:1",
		)
	} else {
		// Matroska with index space reserved and cluster limits can help browsers determine duration
		args = append(
			args,
			"-f",
			"matroska",
			"-live",
			"1",
			"-reserve_index_space",
			"1024k",
			"-cluster_size_limit",
			"2M",
			"-cluster_time_limit",
			"5100",
			"pipe:1",
		)
	}

	ffmpegArgs := append([]string{"-hide_banner", "-loglevel", "error"}, args...)
	models.Log.Info(
		"Streaming with transcode",
		"path",
		path,
		"strategy",
		strategy,
		"args",
		strings.Join(ffmpegArgs, " "),
	)

	cmd := exec.CommandContext(r.Context(), "ffmpeg", ffmpegArgs...)
	cmd.Stdout = w

	if err := cmd.Start(); err != nil {
		models.Log.Error("Failed to start ffmpeg", "path", path, "error", err)
		http.Error(w, fmt.Sprintf("Unplayable: ffmpeg failed to start: %v", err), http.StatusUnsupportedMediaType)
		return
	}

	if err := cmd.Wait(); err != nil {
		if r.Context().Err() == nil {
			models.Log.Error("ffmpeg failed", "path", path, "error", err)
			http.Error(w, fmt.Sprintf("Unplayable: ffmpeg failed: %v", err), http.StatusUnsupportedMediaType)
		} else {
			models.Log.Debug("ffmpeg finished (client disconnected)", "path", path)
		}
	}
}

func (c *ServeCmd) HandleSubtitles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	models.Log.Debug("handleSubtitles request", "path", path, "index", r.URL.Query().Get("index"))

	// Verify path or siblings and check subtitle_count for optimization
	found := false
	hasSubtitles := false
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			media, err := queries.GetMediaByPathExact(ctx, path)
			if err == nil {
				found = true
				// Check if media has embedded subtitles
				if media.SubtitleCount.Valid && media.SubtitleCount.Int64 > 0 {
					hasSubtitles = true
				}
				return nil
			}

			// If path doesn't exist, it might be an external subtitle file next to a media file
			// We'll check if any media in the database shares the same directory and base name
			dir := filepath.Dir(path)
			filename := filepath.Base(path)
			base := strings.TrimSuffix(filename, filepath.Ext(filename))
			// Handle cases like movie.en.srt by stripping one more extension if it exists
			if secondExt := filepath.Ext(base); secondExt != "" {
				base = strings.TrimSuffix(base, secondExt)
			}

			// Simple check: does this directory contain ANY media we know with the same base name?
			mediaInDir, _ := queries.GetMedia(ctx, 1000)
			for _, m := range mediaInDir {
				if filepath.Dir(m.Path) == dir {
					mBase := strings.TrimSuffix(filepath.Base(m.Path), filepath.Ext(m.Path))
					if mBase == base {
						found = true
						// Check if media has embedded subtitles
						if m.SubtitleCount.Valid && m.SubtitleCount.Int64 > 0 {
							hasSubtitles = true
						}
						break
					}
				}
			}
			return nil
		})
		if found {
			break
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			models.Log.Error("Database error in handleSubtitles", "db", dbPath, "error", err)
		}
	}

	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if !utils.FileExists(path) {
		models.Log.Warn("File not found on disk, marking as deleted in databases", "path", path)
		c.markDeletedInAllDBs(r.Context(), path, true)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	streamIndex := r.URL.Query().Get("index")
	requestedExt := r.URL.Query().Get("ext")

	// If it's a media container but no index is specified, we should try to find an external sidecar
	if streamIndex == "" && (ext == ".mkv" || ext == ".mp4" || ext == ".m4v" || ext == ".mov" || ext == ".webm") {
		// Try to find a sibling subtitle file
		sidecars := utils.GetExternalSubtitles(path)
		if len(sidecars) > 0 {
			// If a specific extension was requested, try to find a matching one
			if requestedExt != "" {
				for _, sub := range sidecars {
					if strings.ToLower(filepath.Ext(sub)) == "."+requestedExt {
						path = sub
						ext = strings.ToLower(filepath.Ext(path))
						models.Log.Debug(
							"Found matching sidecar for media file",
							"media",
							r.URL.Query().Get("path"),
							"sidecar",
							path,
						)
						break
					}
				}
				// If no matching extension found, use the first one anyway
				if ext != "."+requestedExt && len(sidecars) > 0 {
					path = sidecars[0]
					ext = strings.ToLower(filepath.Ext(path))
					models.Log.Debug(
						"Requested extension not found, using first sidecar",
						"media",
						r.URL.Query().Get("path"),
						"sidecar",
						path,
					)
				}
			} else {
				// Serve the first found sidecar
				path = sidecars[0]
				ext = strings.ToLower(filepath.Ext(path))
				models.Log.Debug("Found sidecar for media file", "media", r.URL.Query().Get("path"), "sidecar", path)
			}
		} else {
			// Optimization: If no external sidecars found and DB says no embedded subtitles, return early
			if !hasSubtitles {
				models.Log.Debug("No subtitles found (DB check)", "path", path)
				http.Error(w, "No subtitles available", http.StatusNotFound)
				return
			}
			http.Error(w, "No index specified and no sidecar found", http.StatusNotFound)
			return
		}
	}

	if ext == ".idx" {
		subPath := strings.TrimSuffix(path, ".idx") + ".sub"
		if !utils.FileExists(subPath) {
			models.Log.Warn("VobSub conversion requested but .sub file is missing", "idx", path)
			http.Error(w, "Corresponding .sub file not found", http.StatusNotFound)
			return
		}
	}

	if ext == ".vtt" {
		w.Header().Set("Content-Type", "text/vtt")
		http.ServeFile(w, r, path)
		return
	}

	var args []string
	isImageSub := ext == ".idx" || ext == ".sub" || ext == ".sup"

	if streamIndex != "" {
		// Embedded tracks
		args = append(args, "-i", path, "-map", "0:s:"+streamIndex, "-f", "webvtt", "pipe:1")
	} else {
		// Standalone file (srt, lrc, ass, etc.)
		args = append(args, "-i", path, "-f", "webvtt", "pipe:1")
	}

	ffmpegArgs := append([]string{"-hide_banner", "-loglevel", "error"}, args...)
	models.Log.Debug("subtitle ffmpeg command", "args", strings.Join(ffmpegArgs, " "))

	cmd := exec.CommandContext(r.Context(), "ffmpeg", ffmpegArgs...)

	// We don't set Content-Type yet to allow http.Error if ffmpeg fails immediately
	output, err := cmd.CombinedOutput()
	if err != nil {
		if r.Context().Err() == nil {
			msg := "Failed to convert subtitles"
			if isImageSub || streamIndex != "" {
				msg = "Failed to convert subtitles (image-based formats require OCR which is not yet supported for direct VTT streaming)"
			}
			models.Log.Error(msg, "path", path, "error", err, "output", string(output))
			http.Error(w, "Unplayable: subtitle conversion failed", http.StatusUnsupportedMediaType)
		}
		return
	}

	w.Header().Set("Content-Type", "text/vtt")
	_, _ = w.Write(output)
}

// generatePDFThumbnail generates a thumbnail for PDF files using pdftoppm or fallback
func (c *ServeCmd) generatePDFThumbnail(ctx context.Context, path string) ([]byte, string, error) {
	// Try pdftoppm first (fastest, best quality)
	tmpFile, err := os.CreateTemp("", "disco-thumb-*")
	if err == nil {
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpPath + ".png")
		if err = exec.CommandContext(ctx, "pdftoppm", "-png", "-f", "1", "-singlefile", "-scale-to", "320", path, tmpPath).
			Run(); err == nil {
			if data, err2 := os.ReadFile(tmpPath + ".png"); err2 == nil {
				return data, "image/png", nil
			}
		}
	}

	// Fallback: read first page text and render as SVG
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	// Read first 1KB to find text
	buf := make([]byte, 1024)
	n, _ := f.Read(buf)
	text := string(buf[:n])

	// Try to extract readable text (skip PDF headers)
	lines := strings.Split(text, "\n")
	_ = lines // Available but not currently used
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 3 && !strings.HasPrefix(line, "%") && !strings.HasPrefix(line, "/") {
			_ = line // Found but not used
			break
		}
	}

	// SVG thumbnails disabled, returning placeholder
	return []byte{}, "image/svg+xml", nil
}

// extractEpubCover extracts the cover image from an EPUB file
func (c *ServeCmd) extractEpubCover(path string) ([]byte, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Look for cover image in common locations
	coverPatterns := []string{"cover.jpg", "cover.png", "Cover.jpg", "Cover.png", "cover.jpeg"}
	imageExts := []string{".jpg", ".jpeg", ".png"}

	var coverFile *zip.File
	var imageFile *zip.File

	for _, f := range r.File {
		name := f.Name
		base := filepath.Base(name)

		// Check for explicit cover files
		for _, pattern := range coverPatterns {
			if strings.HasSuffix(name, pattern) || strings.HasSuffix(base, pattern) {
				coverFile = f
				break
			}
		}

		// Also look for images in images/ or cover/ directories
		if coverFile == nil {
			dir := filepath.Dir(name)
			if strings.Contains(strings.ToLower(dir), "cover") || strings.Contains(strings.ToLower(dir), "image") {
				for _, ext := range imageExts {
					if strings.HasSuffix(strings.ToLower(base), ext) {
						imageFile = f
						break
					}
				}
			}
		}
	}

	// Prefer explicit cover, fallback to any image
	target := coverFile
	if target == nil {
		target = imageFile
	}

	if target != nil {
		rc, err := target.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}

	return nil, errors.New("no cover image found")
}

// generateEpubThumbnail generates a thumbnail for EPUB files
func (c *ServeCmd) generateEpubThumbnail(path string) ([]byte, string, error) {
	// Try to extract cover image first
	if coverData, err := c.extractEpubCover(path); err == nil && coverData != nil {
		// Detect cover image type
		if bytes.HasPrefix(coverData, []byte{0xFF, 0xD8, 0xFF}) {
			return coverData, "image/jpeg", nil
		}
		if bytes.HasPrefix(coverData, []byte{0x89, 0x50, 0x4E, 0x47}) {
			return coverData, "image/png", nil
		}
		return coverData, "image/jpeg", nil
	}

	// Fallback: extract title/author from metadata and render as SVG
	r, err := zip.OpenReader(path)
	if err != nil {
		// SVG thumbnails disabled, returning placeholder
		return []byte{}, "image/svg+xml", nil
	}
	defer r.Close()

	// Look for content.opf or .opf files for metadata
	var opfFile *zip.File
	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
			opfFile = f
			break
		}
	}

	_ = opfFile // Available but not currently used
	if opfFile != nil {
		rc, err := opfFile.Open()
		if err == nil {
			content, _ := io.ReadAll(rc)
			rc.Close()
			// Simple XML parsing for dc:title
			contentStr := string(content)
			if idx := strings.Index(contentStr, "<dc:title"); idx != -1 {
				start := strings.Index(contentStr[idx:], ">")
				end := strings.Index(contentStr[idx:], "</dc:title>")
				if start != -1 && end != -1 && end > start {
					_ = strings.TrimSpace(contentStr[idx+start+1 : idx+end]) // Title extracted but not used
				}
			}
		}
	}

	// SVG thumbnails disabled, returning placeholder
	return []byte{}, "image/svg+xml", nil
}

func (c *ServeCmd) HandleThumbnail(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Verify path exists in database to prevent arbitrary file access
	found := false
	var mediaType string
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(ctx, path)
			if err == nil {
				found = true
				if dbMedia.MediaType.Valid {
					mediaType = dbMedia.MediaType.String
				}
			}
			return err
		})
		if found {
			break
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			models.Log.Error("Database error in handleThumbnail", "db", dbPath, "error", err)
		}
	}

	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check cache (skip cache in dev mode)
	if !c.Dev {
		if data, ok := c.thumbnailCache.Load(path); ok {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "public, max-age=31536000")
			if bytes, ok := data.([]byte); ok {
				_, _ = w.Write(bytes)
			}
			return
		}
	}

	// Generate thumbnail using media_type from database
	ext := strings.ToLower(filepath.Ext(path))

	// Handle image files
	if mediaType == "image" {
		if info, err := os.Stat(path); err == nil && info.Size() < 500*1024 {
			data, err := os.ReadFile(path)
			if err == nil {
				contentType := utils.GetContentTypeFromExt(ext)
				w.Header().Set("Content-Type", contentType)
				if !c.Dev {
					w.Header().Set("Cache-Control", "public, max-age=31536000")
				} else {
					w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				}
				_, _ = w.Write(data)
				return
			}
		}
	}

	// Handle text-based documents with smart thumbnails
	switch ext {
	case ".pdf":
		thumb, contentType, err := c.generatePDFThumbnail(r.Context(), path)
		if err != nil || len(thumb) == 0 {
			models.Log.Warn("PDF thumbnail generation failed", "path", path, "error", err)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		if !c.Dev {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		} else {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		_, _ = w.Write(thumb)
		return

	case ".epub":
		thumb, contentType, err := c.generateEpubThumbnail(path)
		if err != nil || len(thumb) == 0 {
			models.Log.Warn("EPUB thumbnail generation failed", "path", path, "error", err)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		if !c.Dev {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		} else {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		_, _ = w.Write(thumb)
		return

	case ".txt", ".md", ".markdown", ".rtf":
		// SVG thumbnails disabled for text files
		http.NotFound(w, r)
		return
	}

	// Default: handle video/audio with ffmpeg
	var args []string
	isVideo := mediaType == "video"
	if isVideo {
		args = []string{
			"-ss",
			"25",
			"-i",
			path,
			"-frames:v",
			"1",
			"-q:v",
			"4",
			"-vf",
			"scale=320:-1",
			"-f",
			"image2",
			"pipe:1",
		}
	} else if mediaType == "audio" {
		// For audio files, try to extract embedded album art first
		// If no album art exists, ffmpeg will fail, so we return a placeholder
		args = []string{"-i", path, "-an", "-vcodec", "copy", "-f", "image2", "pipe:1"}
	} else {
		// SVG thumbnails disabled for documents and unsupported types
		http.NotFound(w, r)
		return
	}

	cmd := exec.CommandContext(
		r.Context(),
		"ffmpeg",
		append([]string{"-hide_banner", "-loglevel", "error"}, args...)...)
	thumb, err := cmd.Output()

	// If video thumbnail is too dark, try seeking further (e.g. 60 seconds later)
	if err == nil && isVideo && utils.IsImageTooDark(thumb, 0.05) {
		models.Log.Debug("Thumbnail too dark, retrying further in the video", "path", path)
		retryArgs := []string{
			"-ss",
			"85",
			"-i",
			path,
			"-frames:v",
			"1",
			"-q:v",
			"4",
			"-vf",
			"scale=320:-1",
			"-f",
			"image2",
			"pipe:1",
		}
		cmdRetry := exec.CommandContext(
			r.Context(),
			"ffmpeg",
			append([]string{"-hide_banner", "-loglevel", "error"}, retryArgs...)...)
		if retryThumb, retryErr := cmdRetry.Output(); retryErr == nil {
			thumb = retryThumb
		}
	}

	if err != nil {
		// For audio files without embedded art, or video files that fail, return 404
		// Frontend will fall back to client-generated thumbnails
		models.Log.Debug("Thumbnail generation failed", "path", path, "error", err)
		http.NotFound(w, r)
		return
	}

	// Cache it (skip in dev mode)
	if !c.Dev {
		c.thumbnailCache.Store(path, thumb)
	}

	w.Header().Set("Content-Type", "image/jpeg")
	if !c.Dev {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}
	_, _ = w.Write(thumb)
}

func (c *ServeCmd) HandleHLSPlaylist(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Fetch media to get duration
	var m models.Media
	found := false
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(ctx, path)
			if err == nil {
				m = models.FromDB(dbMedia)
				found = true
			}
			return err
		})
		if found {
			break
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			models.Log.Error("Database error in handleHLSPlaylist", "db", dbPath, "error", err)
		}
	}

	if !found || m.Duration == nil {
		http.Error(w, "Media not found or no duration", http.StatusNotFound)
		return
	}

	duration := float64(*m.Duration)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")

	playlist := utils.GenerateHLSPlaylist(path, duration, HlsSegmentDuration)
	fmt.Fprint(w, playlist)
}

func (c *ServeCmd) HandleHLSSegment(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	indexStr := r.URL.Query().Get("index")
	if path == "" || indexStr == "" {
		http.Error(w, "Path and index required", http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "Invalid index", http.StatusBadRequest)
		return
	}

	if !utils.FileExists(path) {
		models.Log.Warn("File not found on disk, marking as deleted in databases", "path", path)
		c.markDeletedInAllDBs(r.Context(), path, true)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	startTime := float64(index * HlsSegmentDuration)

	// Check if we have ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		http.Error(w, "ffmpeg not found", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "video/MP2T")

	// Fetch media to get codec info
	var m models.Media
	found := false
	for _, dbPath := range c.Databases {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(ctx, path)
			if err == nil {
				m = models.FromDB(dbMedia)
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	strategy := utils.GetTranscodeStrategy(m)
	models.Log.Debug("HLS Segment request", "index", index, "start", startTime, "strategy", strategy, "path", path)

	args := utils.GetHLSSegmentArgs(path, startTime, HlsSegmentDuration, strategy)

	// Skip logging for segments to avoid spam
	// models.Log.Debug("HLS Segment", "index", index, "start", startTime)

	cmd := exec.CommandContext(
		r.Context(),
		"ffmpeg",
		append([]string{"-hide_banner", "-loglevel", "error"}, args...)...)
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		if r.Context().Err() != nil {
			models.Log.Debug("Client disconnected during HLS transcoding", "path", path, "index", index)
		} else {
			models.Log.Error("HLS transcoding failed", "path", path, "index", index, "error", err)
		}
	}
}

// serveFileWithMimeType serves a file with the correct MIME type based on extension
func serveFileWithMimeType(w http.ResponseWriter, r *http.Request, filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Set proper MIME types for common ebook formats
	mimeTypes := map[string]string{
		".css":   "text/css",
		".html":  "text/html",
		".htm":   "text/html",
		".xhtml": "application/xhtml+xml",
		".xml":   "application/xml",
		".ncx":   "application/xml",
		".opf":   "application/xml",
		".jpeg":  "image/jpeg",
		".jpg":   "image/jpeg",
		".png":   "image/png",
		".gif":   "image/gif",
		".svg":   "image/svg+xml",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		w.Header().Set("Content-Type", mimeType)
	}

	http.ServeFile(w, r, filePath)
}
