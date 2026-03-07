package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/gabriel-vasile/mimetype"
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
		// Linear interpolation for chunk size based on file size
		dataPoints := [][2]float64{
			{26214400, 262144},      // 25MB -> 256KB
			{52428800000, 10485760}, // 50GB -> 10MB
		}
		chunkSize = int64(LinearInterpolation(float64(size), dataPoints))
	}

	segments := CalculateSegmentsInt(size, chunkSize, gap)
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
				slog.Error("Read error during hashing", "path", path, "offset", offset, "error", err)
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

	return fmt.Sprintf("%x", finalHash.Sum(nil)), nil
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

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// FilterDeleted returns only the paths that currently exist on the filesystem
func FilterDeleted(paths []string) []string {
	var existing []string
	deletedDirs := make(map[string]bool)

	for _, p := range paths {
		dir := filepath.Dir(p)
		if deletedDirs[dir] {
			continue
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			deletedDirs[dir] = true
			continue
		}

		if _, err := os.Stat(p); err == nil {
			existing = append(existing, p)
		}
	}
	return existing
}

type FileStats struct {
	Size         int64
	TimeCreated  int64
	TimeModified int64
}

// GetFileStats returns size and timestamps for a file
func GetFileStats(path string) (FileStats, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return FileStats{}, err
	}

	return FileStats{
		Size:         stat.Size(),
		TimeCreated:  stat.ModTime().Unix(), // Go doesn't have a cross-platform way to get creation time easily
		TimeModified: stat.ModTime().Unix(),
	}, nil
}

// IsFileOpen checks if a file is currently open by any process
func IsFileOpen(path string) bool {
	if runtime.GOOS == "windows" {
		// On Windows, try to open the file with exclusive access
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			return true
		}
		f.Close()
		return false
	}

	if runtime.GOOS == "linux" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		files, err := os.ReadDir("/proc")
		if err != nil {
			return false
		}

		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			// Check if name is a number (PID)
			isPid := true
			for _, r := range f.Name() {
				if r < '0' || r > '9' {
					isPid = false
					break
				}
			}
			if !isPid {
				continue
			}

			fdDir := filepath.Join("/proc", f.Name(), "fd")
			fds, err := os.ReadDir(fdDir)
			if err != nil {
				continue
			}

			for _, fd := range fds {
				link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
				if err == nil && link == absPath {
					return true
				}
			}
		}
	}

	return false
}

// DetectMimeType returns the mimetype of a file
func DetectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".apk" {
		return "application/vnd.android.package-archive"
	}
	if ext == ".zim" {
		return "application/x-zim"
	}
	if ext == ".epub" {
		return "application/epub+zip"
	}
	if ext == ".pdf" {
		return "application/pdf"
	}
	if ext == ".mobi" {
		return "application/x-mobipocket-ebook"
	}

	mime, err := mimetype.DetectFile(path)
	if err != nil {
		return ""
	}
	return mime.String()
}

// Rename renames a file, respecting simulation mode
func Rename(flags models.GlobalFlags, src, dst string) error {
	if flags.Simulate {
		fmt.Printf("rename %s %s\n", src, dst)
		return nil
	}
	slog.Debug("rename", "src", src, "dst", dst)
	return os.Rename(src, dst)
}

// Unlink deletes a file, respecting simulation mode
func Unlink(flags models.GlobalFlags, path string) error {
	if flags.Simulate {
		fmt.Printf("unlink %s\n", path)
		return nil
	}
	slog.Debug("unlink", "path", path)
	return os.Remove(path)
}

// Rmtree deletes a directory tree, respecting simulation mode
func Rmtree(flags models.GlobalFlags, path string) error {
	if flags.Simulate {
		fmt.Printf("rmtree %s\n", path)
		return nil
	}
	slog.Debug("rmtree", "path", path)
	return os.RemoveAll(path)
}

// AltName returns an alternative filename if the given path already exists
func AltName(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	counter := 1
	for {
		newPath := fmt.Sprintf("%s_%d%s", base, counter, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

// CommonPath returns the longest common path prefix
func CommonPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return filepath.Dir(paths[0])
	}

	sep := string(filepath.Separator)
	parts := strings.Split(filepath.Clean(paths[0]), sep)

	for i := 1; i < len(paths); i++ {
		p := strings.Split(filepath.Clean(paths[i]), sep)
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

	if len(parts) == 0 {
		return sep
	}
	return strings.Join(parts, sep)
}

// CommonPathFull returns a common path prefix.
// Previously it included common words in the suffix, but this was confusing for UI.
func CommonPathFull(paths []string) string {
	return CommonPath(paths)
}

// GetExternalSubtitles finds external subtitle files associated with a media file
func GetExternalSubtitles(path string) []string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)

	var subs []string
	subExts := []string{".srt", ".vtt", ".ass", ".ssa", ".lrc", ".idx", ".sub"}

	for _, sExt := range subExts {
		subPath := base + sExt
		if FileExists(subPath) {
			subs = append(subs, subPath)
		}

		// Check for language-specific suffixes like .en.srt
		// This is a simplified version of Python's glob logic
		matches, _ := filepath.Glob(base + ".*" + sExt)
		for _, m := range matches {
			if !strings.EqualFold(m, subPath) { // Already added above
				subs = append(subs, m)
			}
		}
	}

	return Unique(subs)
}
