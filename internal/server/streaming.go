package server

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
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

const HlsSegmentDuration = 10

func (c *Server) HandleTranscode(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	m models.Media,
	strategy media.TranscodeStrategy,
) {
	w.Header().Set("Content-Type", strategy.TargetMime)
	// Note: We don't support HTTP Range requests for transcoded content.
	// Seeking is handled via the "start" query parameter instead.
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(path)))

	args := c.buildTranscodeArgs(r, path, m, strategy)

	ffmpegArgs := append([]string{"-hide_banner", "-loglevel", "error"}, args...)
	logging.Log.Info(
		"Streaming with transcode",
		"path",
		path,
		"strategy",
		strategy,
		"args",
		strings.Join(ffmpegArgs, " "),
	)

	c.runTranscodeCommand(r.Context(), w, path, ffmpegArgs)
}

// buildTranscodeArgs assembles all ffmpeg arguments for transcoding
func (c *Server) buildTranscodeArgs(
	r *http.Request,
	path string,
	m models.Media,
	strategy media.TranscodeStrategy,
) []string {
	var args []string

	// Add seek/start time
	if start := r.URL.Query().Get("start"); start != "" {
		args = append(args, "-ss", start)
	}

	// Input file
	args = append(args, "-fflags", "+genpts", "-i", path)

	// Add duration from metadata if available
	if m.Duration != nil && *m.Duration > 0 {
		args = append(args, "-t", strconv.FormatInt(*m.Duration, 10))
	}

	// Video codec settings
	args = append(args, c.buildVideoCodecArgs(strategy)...)

	// Audio codec settings
	args = append(args, c.buildAudioCodecArgs(strategy)...)

	// Common output flags
	args = append(args, "-avoid_negative_ts", "make_zero", "-map_metadata", "-1", "-sn")

	// Format-specific output flags
	args = append(args, c.buildFormatArgs(strategy)...)

	return args
}

// buildVideoCodecArgs returns the video codec arguments based on the strategy
func (c *Server) buildVideoCodecArgs(strategy media.TranscodeStrategy) []string {
	if strategy.VideoCopy {
		return []string{"-c:v", "copy"}
	}

	if strategy.TargetMime == "video/mp4" {
		return []string{"-c:v", "libx264", "-preset", "ultrafast", "-crf", "28"}
	}

	// WebM
	return []string{
		"-c:v", "libvpx-vp9",
		"-deadline", "realtime",
		"-cpu-used", "8",
		"-crf", "30",
		"-b:v", "0",
	}
}

// buildAudioCodecArgs returns the audio codec arguments based on the strategy
func (c *Server) buildAudioCodecArgs(strategy media.TranscodeStrategy) []string {
	if strategy.AudioCopy {
		return []string{"-c:a", "copy"}
	}

	if strategy.TargetMime == "video/mp4" {
		return []string{"-c:a", "aac", "-b:a", "128k", "-ac", "2"}
	}

	// WebM supports Opus
	return []string{"-c:a", "libopus", "-b:a", "128k", "-ac", "2"}
}

// buildFormatArgs returns the format-specific output arguments
func (c *Server) buildFormatArgs(strategy media.TranscodeStrategy) []string {
	if strategy.TargetMime == "video/mp4" {
		// frag_keyframe+empty_moov+default_base_moof+global_sidx is the standard for fragmented streaming
		return []string{
			"-f", "mp4",
			"-movflags", "frag_keyframe+empty_moov+default_base_moof+global_sidx",
			"pipe:1",
		}
	}

	// Matroska with index space reserved and cluster limits can help browsers determine duration
	return []string{
		"-f", "matroska",
		"-live", "1",
		"-reserve_index_space", "1024k",
		"-cluster_size_limit", "2M",
		"-cluster_time_limit", "5100",
		"pipe:1",
	}
}

// runTranscodeCommand executes the ffmpeg command and handles errors
func (c *Server) runTranscodeCommand(ctx context.Context, w http.ResponseWriter, path string, ffmpegArgs []string) {
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	cmd.Stdout = w

	if err := cmd.Start(); err != nil {
		logging.Log.Error("Failed to start ffmpeg", "path", path, "error", err)
		http.Error(w, fmt.Sprintf("Unplayable: ffmpeg failed to start: %v", err), http.StatusUnsupportedMediaType)
		return
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == nil {
			logging.Log.Error("ffmpeg failed", "path", path, "error", err)
			http.Error(w, fmt.Sprintf("Unplayable: ffmpeg failed: %v", err), http.StatusUnsupportedMediaType)
		} else {
			logging.Log.Debug("ffmpeg finished (client disconnected)", "path", path)
		}
	}
}

// checkPathInDB verifies the path exists in any database and checks for embedded subtitles
func (c *Server) checkFuzzyPathMatch(
	ctx context.Context,
	queries *database.Queries,
	path string,
) (found, hasSubtitles bool) {
	// Check if any media in the database shares the same directory and base name
	dir := filepath.Dir(path)
	filename := filepath.Base(path)
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	if secondExt := filepath.Ext(base); secondExt != "" {
		base = strings.TrimSuffix(base, secondExt)
	}

	// Look up only the indexed path range that can contain a sibling with this
	// base name, instead of scanning the first 1000 rows of the database.
	lower := filepath.Join(dir, base)
	upper := lower + "\U0010FFFF"
	mediaInDir, _ := queries.GetMediaByPathRange(ctx, lower, upper, 1000)
	for _, m := range mediaInDir {
		if filepath.Dir(m.Path) == dir {
			mBase := strings.TrimSuffix(filepath.Base(m.Path), filepath.Ext(m.Path))
			if mBase == base {
				found = true
				if m.SubtitleCount.Valid && m.SubtitleCount.Int64 > 0 {
					hasSubtitles = true
				}
				break
			}
		}
	}
	return found, hasSubtitles
}

func (c *Server) checkPathInDB(ctx context.Context, path string) (found, hasSubtitles bool) {
	for _, dbPath := range c.Databases {
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			media, err := queries.GetMediaByPathExact(ctx, path)
			if err == nil {
				found = true
				if media.SubtitleCount.Valid && media.SubtitleCount.Int64 > 0 {
					hasSubtitles = true
				}
				return nil
			}

			found, hasSubtitles = c.checkFuzzyPathMatch(ctx, queries, path)
			return nil
		})
		if found {
			break
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logging.Log.Error("Database error in handleSubtitles", "db", dbPath, "error", err)
		}
	}
	return found, hasSubtitles
}

// resolveSidecarSubtitle finds an external subtitle file for a media container
func (c *Server) resolveSidecarSubtitle(
	path, requestedExt string,
	hasSubtitles bool,
) (subPath, subType string, err error) {
	sidecars := media.GetExternalSubtitles(path)
	if len(sidecars) == 0 {
		if !hasSubtitles {
			logging.Log.Debug("No subtitles found (DB check)", "path", path)
			return "", "", errors.New("no subtitles available")
		}
		return "", "", errors.New("no index specified and no sidecar found")
	}

	if requestedExt != "" {
		for _, sub := range sidecars {
			if strings.ToLower(filepath.Ext(sub)) == "."+requestedExt {
				logging.Log.Debug(
					"Found matching sidecar for media file",
					"media",
					path,
					"sidecar",
					sub,
				)
				return sub, strings.ToLower(filepath.Ext(sub)), nil
			}
		}
		// No matching extension found, use the first one
		logging.Log.Debug(
			"Requested extension not found, using first sidecar",
			"media",
			path,
			"sidecar",
			sidecars[0],
		)
		return sidecars[0], strings.ToLower(filepath.Ext(sidecars[0])), nil
	}

	logging.Log.Debug("Found sidecar for media file", "media", path, "sidecar", sidecars[0])
	return sidecars[0], strings.ToLower(filepath.Ext(sidecars[0])), nil
}

// validateVobSubFiles checks that both .idx and .sub files exist for VobSub subtitles
func (c *Server) validateVobSubFiles(path string) (string, error) {
	subPath := strings.TrimSuffix(path, ".idx") + ".sub"
	if !fileutil.FileExists(subPath) {
		logging.Log.Warn("VobSub conversion requested but .sub file is missing", "idx", path)
		return "", errors.New("corresponding .sub file not found")
	}
	return subPath, nil
}

// convertSubtitleToVTT converts a subtitle file to WebVTT format using ffmpeg
func (c *Server) convertSubtitleToVTT(ctx context.Context, path, streamIndex string) ([]byte, error) {
	var args []string
	isImageSub := func() bool {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".idx" || ext == ".sub" || ext == ".sup"
	}()

	if streamIndex != "" {
		args = append(args, "-i", path, "-map", "0:s:"+streamIndex, "-f", "webvtt", "pipe:1")
	} else {
		args = append(args, "-i", path, "-f", "webvtt", "pipe:1")
	}

	ffmpegArgs := append([]string{"-hide_banner", "-loglevel", "error"}, args...)
	logging.Log.Debug("subtitle ffmpeg command", "args", strings.Join(ffmpegArgs, " "))

	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == nil {
			msg := "Failed to convert subtitles"
			if isImageSub || streamIndex != "" {
				msg = "Failed to convert subtitles (image-based formats require OCR which is not yet supported for direct VTT streaming)"
			}
			logging.Log.Error(msg, "path", path, "error", err, "output", string(output))
		}
		return nil, err
	}
	return output, nil
}

func (c *Server) HandleSubtitles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	logging.Log.Debug("handleSubtitles request", "path", path, "index", r.URL.Query().Get("index"))

	// Verify path or siblings and check subtitle_count for optimization
	found, hasSubtitles := c.checkPathInDB(r.Context(), path)
	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if !fileutil.FileExists(path) {
		logging.Log.Warn("File not found on disk", "path", path)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	streamIndex := r.URL.Query().Get("index")
	requestedExt := r.URL.Query().Get("ext")

	// If it's a media container but no index is specified, try to find an external sidecar
	if streamIndex == "" && (ext == ".mkv" || ext == ".mp4" || ext == ".m4v" || ext == ".mov" || ext == ".webm") {
		resolvedPath, resolvedExt, err := c.resolveSidecarSubtitle(path, requestedExt, hasSubtitles)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		path = resolvedPath
		ext = resolvedExt
	}

	if ext == ".idx" {
		if _, err := c.validateVobSubFiles(path); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}

	if ext == ".vtt" {
		w.Header().Set("Content-Type", "text/vtt")
		http.ServeFile(w, r, path)
		return
	}

	output, err := c.convertSubtitleToVTT(r.Context(), path, streamIndex)
	if err != nil {
		http.Error(w, "Unplayable: subtitle conversion failed", http.StatusUnsupportedMediaType)
		return
	}

	w.Header().Set("Content-Type", "text/vtt")
	_, _ = w.Write(output)
}

// generatePDFThumbnail generates a thumbnail for PDF files using pdftoppm or fallback
func (c *Server) generatePDFThumbnail(ctx context.Context, path string) ([]byte, string, error) {
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
func (c *Server) extractEpubCover(path string) ([]byte, error) {
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
func (c *Server) generateEpubThumbnail(path string) ([]byte, string, error) {
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

// getMediaTypeFromDB looks up the media type for a given path across all databases
func (c *Server) getMediaTypeFromDB(ctx context.Context, path string) (found bool, mediaType string) {
	for _, dbPath := range c.Databases {
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
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
			logging.Log.Error("Database error in handleThumbnail", "db", dbPath, "error", err)
		}
	}
	return found, mediaType
}

// writeThumbnailResponse writes the thumbnail bytes with appropriate headers
func (c *Server) writeThumbnailResponse(w http.ResponseWriter, data []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	if !c.Dev {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}
	_, _ = w.Write(data)
}

// tryServeSmallImage serves the original file if it is small enough to be used directly as a thumbnail
func (c *Server) tryServeSmallImage(w http.ResponseWriter, path string) bool {
	if info, err := os.Stat(path); err == nil && info.Size() < 500*1024 {
		if data, err := os.ReadFile(path); err == nil {
			ext := strings.ToLower(filepath.Ext(path))
			contentType := fileutil.GetContentTypeFromExt(ext)
			c.writeThumbnailResponse(w, data, contentType)
			return true
		}
	}
	return false
}

// generateFallbackThumbnail creates a thumbnail for video or audio files using ffmpeg.
// For video, it seeks to 25s and falls back to the first frame for short clips.
// It retries at 85s if the frame is too dark. For audio, it tries to extract
// embedded album art.
func (c *Server) generateFallbackThumbnail(ctx context.Context, path, mediaType string) ([]byte, error) {
	if mediaType == "video" {
		// Seek into the video to avoid black intro frames. Short clips have no
		// frame at 25s, so fall back to the first frame instead of returning 404.
		thumb, err := c.generateVideoFrame(ctx, path, "25")
		if err != nil || len(thumb) == 0 {
			thumb, err = c.generateVideoFrame(ctx, path, "0")
		}

		// If the first usable frame is too dark, try seeking further.
		if err == nil && len(thumb) > 0 && media.IsImageTooDark(thumb, 0.05) {
			logging.Log.Debug("Thumbnail too dark, retrying further in the video", "path", path)
			if retryThumb, retryErr := c.generateVideoFrame(ctx, path, "85"); retryErr == nil && len(retryThumb) > 0 {
				thumb = retryThumb
			}
		}

		return thumb, err
	}

	if mediaType != "audio" {
		return nil, errors.New("unsupported media type for thumbnail")
	}

	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-i", path, "-an", "-vcodec", "copy",
		"-f", "image2", "pipe:1",
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	return cmd.Output()
}

// generateVideoFrame extracts one scaled frame from path at the given seek
// position. An empty result is treated as a failure by callers.
func (c *Server) generateVideoFrame(ctx context.Context, path, seek string) ([]byte, error) {
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-ss", seek, "-i", path,
		"-frames:v", "1", "-q:v", "4",
		"-vf", "scale=320:-1", "-f", "image2", "pipe:1",
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	return cmd.Output()
}

func (c *Server) HandleThumbnail(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Verify path exists in database to prevent arbitrary file access
	found, mediaType := c.getMediaTypeFromDB(r.Context(), path)
	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check cache (skip cache in dev mode)
	if !c.Dev {
		if val, ok := c.thumbnailCache.Load(path); ok {
			if data, ok := val.([]byte); ok {
				c.writeThumbnailResponse(w, data, "image/jpeg")
				return
			}
		}
	}

	// Handle image files
	if mediaType == "image" && c.tryServeSmallImage(w, path) {
		return
	}

	// Handle document types with smart thumbnails
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		thumb, contentType, err := c.generatePDFThumbnail(r.Context(), path)
		if err != nil {
			logging.Log.Warn("PDF thumbnail generation failed", "path", path, "error", err)
			http.NotFound(w, r)
			return
		}
		if len(thumb) == 0 {
			logging.Log.Debug("PDF has no thumbnail", "path", path)
			http.NotFound(w, r)
			return
		}
		c.writeThumbnailResponse(w, thumb, contentType)
		return

	case ".epub":
		thumb, contentType, err := c.generateEpubThumbnail(path)
		if err != nil {
			logging.Log.Warn("EPUB thumbnail generation failed", "path", path, "error", err)
			http.NotFound(w, r)
			return
		}
		if len(thumb) == 0 {
			logging.Log.Debug("EPUB has no thumbnail", "path", path)
			http.NotFound(w, r)
			return
		}
		c.writeThumbnailResponse(w, thumb, contentType)
		return

	case ".txt", ".md", ".markdown", ".rtf":
		http.NotFound(w, r)
		return
	}

	// Default: handle video/audio with ffmpeg
	thumb, err := c.generateFallbackThumbnail(r.Context(), path, mediaType)
	if err != nil {
		logging.Log.Debug("Thumbnail generation failed", "path", path, "error", err)
		http.NotFound(w, r)
		return
	}

	// Cache it (skip in dev mode)
	if !c.Dev {
		c.thumbnailCache.Store(path, thumb)
	}

	c.writeThumbnailResponse(w, thumb, "image/jpeg")
}

func (c *Server) HandleHLSPlaylist(w http.ResponseWriter, r *http.Request) {
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
				m = dbMedia.ToModel()
				found = true
			}
			return err
		})
		if found {
			break
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logging.Log.Error("Database error in handleHLSPlaylist", "db", dbPath, "error", err)
		}
	}

	if !found {
		http.Error(w, "Media not found in database", http.StatusForbidden)
		return
	}
	if m.Duration == nil {
		http.Error(w, "Media not found or no duration", http.StatusNotFound)
		return
	}

	duration := float64(*m.Duration)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")

	playlist := media.GenerateHLSPlaylist(path, duration, HlsSegmentDuration, BasePathFromContext(r.Context()))
	fmt.Fprint(w, playlist)
}

func (c *Server) HandleHLSSegment(w http.ResponseWriter, r *http.Request) {
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

	// Only stream files that are listed in a configured database.
	if found, _ := c.getMediaTypeFromDB(r.Context(), path); !found {
		http.Error(w, "Media not found in database", http.StatusForbidden)
		return
	}

	if !fileutil.FileExists(path) {
		logging.Log.Warn("File not found on disk", "path", path)
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
				m = dbMedia.ToModel()
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	strategy := media.GetTranscodeStrategy(m)
	logging.Log.Debug("HLS Segment request", "index", index, "start", startTime, "strategy", strategy, "path", path)

	args := media.GetHLSSegmentArgs(path, startTime, HlsSegmentDuration, strategy)

	// Skip logging for segments to avoid spam
	// logging.Log.Debug("HLS Segment", "index", index, "start", startTime)

	cmd := exec.CommandContext(
		r.Context(),
		"ffmpeg",
		append([]string{"-hide_banner", "-loglevel", "error"}, args...)...,
	)
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		if r.Context().Err() != nil {
			logging.Log.Debug("Client disconnected during HLS transcoding", "path", path, "index", index)
		} else {
			logging.Log.Error("HLS transcoding failed", "path", path, "index", index, "error", err)
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

	w.Header().Set("X-Content-Type-Options", "nosniff")
	if ext == ".html" || ext == ".htm" || ext == ".xhtml" {
		setDocumentSecurityHeaders(w)
	}
	if mimeType, ok := mimeTypes[ext]; ok {
		w.Header().Set("Content-Type", mimeType)
	}

	// Do not use http.ServeFile here: it redirects URLs ending in
	// "/index.html" to the directory, which makes the EPUB/text TOC wrapper
	// request itself recursively. ServeContent serves the actual file.
	f, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
