// Package fileutil provides filesystem, hashing, process, and terminal I/O
// helpers shared by the CLI and HTTP server.
package fileutil

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// SampleHashFile calculates a hash based on small file segments
func SampleHashFile(path string, threads int, gap float64, chunkSize int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	size := info.Size()
	if size == 0 {
		return "", nil
	}

	if chunkSize <= 0 {
		// Linear interpolation for Chunk size based on file size
		dataPoints := [][2]float64{
			{26214400, 262144},      // 25MB -> 256KB
			{52428800000, 10485760}, // 50GB -> 10MB
		}
		chunkSize = int64(utils.LinearInterpolation(float64(size), dataPoints))
	}

	segments := utils.CalculateSegmentsInt(size, chunkSize, gap)
	if len(segments) == 0 {
		return "", nil
	}

	hashes := make([][]byte, len(segments))
	var wg sync.WaitGroup

	if threads <= 0 {
		threads = 1
	}

	sem := make(chan struct{}, threads)

	for i, start := range segments {
		wg.Add(1)
		go func(idx int, offset int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			buf := make([]byte, chunkSize)
			n, err := file.ReadAt(buf, offset)
			if err != nil && err != io.EOF {
				logging.Log.Error("Read error during hashing", "path", path, "offset", offset, "error", err)
				return
			}
			data := buf[:n]
			h := sha256.New()
			h.Write(data)
			hashes[idx] = h.Sum(nil)
		}(i, start)
	}

	wg.Wait()

	// Final hash of all segment hashes
	finalHash := sha256.New()
	for _, h := range hashes {
		if h != nil {
			finalHash.Write(h)
		}
	}

	return hex.EncodeToString(finalHash.Sum(nil)), nil
}

// FullHashFile calculates a full sha256 hash of a file
func FullHashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// FilterDeleted returns only the paths that currently exist on the filesystem

type FileStats struct {
	Size         int64
	TimeCreated  int64
	TimeModified int64
}

// GetFileStats returns size and timestamps for a file

// DetectMimeType returns the mimetype of a file based on extension
// Used only for HTTP Content-Type headers in streaming

// GetContentTypeFromExt returns the content type based on file extension
func GetContentTypeFromExt(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".heic", ".heif":
		return "image/heif"
	case ".avif":
		return "image/avif"
	default:
		return "application/octet-stream"
	}
}

// GetMountPoint returns the mount point for a given path

// MoveFile moves a file from source to destination, handling cross-filesystem moves
func MoveFile(src, dst string) error {
	// Capture source timestamps before any operations
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	atime := GetAccessTime(info)
	mtime := info.ModTime()

	// Try Rename first (fast on same filesystem)
	err = os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Ensure destination directory exists and retry rename
	if err2 := os.MkdirAll(filepath.Dir(dst), 0o755); err2 != nil {
		return err2
	}
	err = os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename fails (e.g. cross-filesystem), try copying
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	// Sync to ensure data is written before deleting source
	if err := out.Sync(); err != nil {
		return err
	}

	// Close files before deleting source
	in.Close()
	out.Close()

	// Restore timestamps on destination
	_ = os.Chtimes(dst, atime, mtime)

	return os.Remove(src)
}

// Rename renames a file, respecting simulation mode

// Unlink deletes a file, respecting simulation mode

// Rmtree deletes a directory tree, respecting simulation mode

// AltName returns an alternative filename if the given path already exists

// CommonPath returns the longest common path prefix
// Always returns forward slashes for cross-platform consistency
func CommonPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return filepath.Dir(paths[0])
	}

	// Clean all paths and split into components
	// filepath.Clean normalizes separators, but we split on both for cross-platform consistency
	splitPath := func(p string) []string {
		p = filepath.Clean(p)
		// Handle both forward and back slashes
		parts := strings.FieldsFunc(p, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		// Preserve leading slash for absolute paths
		if len(p) > 0 && (p[0] == '/' || p[0] == '\\') {
			parts = append([]string{""}, parts...)
		}
		return parts
	}

	parts := splitPath(paths[0])

	for i := 1; i < len(paths); i++ {
		p := splitPath(paths[i])
		if len(p) < len(parts) {
			parts = parts[:len(p)]
		}
		for j := 0; j < len(parts); j++ {
			if parts[j] != p[j] {
				parts = parts[:j]
				break
			}
		}
	}

	// Use filepath.Join to reconstruct the path properly
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 && parts[0] == "" {
		return string(filepath.Separator)
	}
	result := filepath.Join(parts...)
	// Ensure leading slash for absolute paths
	if len(parts) > 0 && parts[0] == "" && !strings.HasPrefix(result, string(filepath.Separator)) {
		result = string(filepath.Separator) + result
	}
	return result
}

// CommonPathFull returns a common path prefix.
// Previously it included common words in the suffix, but this was confusing for UI.
func CommonPathFull(paths []string) string {
	return CommonPath(paths)
}
