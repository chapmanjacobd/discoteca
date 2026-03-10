package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/chapmanjacobd/discotheque/internal/models"
)

// RandomString returns a random hexadecimal string of the given length
func RandomString(n int) string {
	b := make([]byte, n/2+1)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)[:n]
}

// RandomFilename appends a random string to the filename before the extension
func RandomFilename(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s.%s%s", base, RandomString(6), ext)
}

// TrimPathSegments reduces the length of path segments to fit within desiredLength.
// It uses a fish-shell style where parent/grandparent segments are reduced to their first letter.
func TrimPathSegments(path string, desiredLength int) string {
	if len(path) <= desiredLength {
		return path
	}

	ext := filepath.Ext(path)
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	if dir == "." || dir == "/" || dir == "" {
		if len(path) > desiredLength {
			return ShortenMiddle(path, desiredLength)
		}
		return path
	}

	sep := string(filepath.Separator)
	pre := ""
	if filepath.IsAbs(path) {
		pre = sep
		dir = strings.TrimPrefix(dir, sep)
	}

	segments := strings.Split(dir, sep)

	// Try shortening segments from left to right (grandparents first)
	for i := range segments {
		if len(pre+filepath.Join(append(segments, base)...)) <= desiredLength {
			break
		}
		if len(segments[i]) > 1 {
			segments[i] = string([]rune(segments[i])[0])
		}
	}

	res := pre + filepath.Join(append(segments, base)...)
	if len(res) > desiredLength {
		// If still too long, shorten the base name
		available := desiredLength - len(pre+filepath.Join(segments...)+sep)
		if available > 3 {
			stem := strings.TrimSuffix(base, ext)
			res = pre + filepath.Join(append(segments, ShortenMiddle(stem, available-len(ext))+ext)...)
		} else {
			res = ShortenMiddle(res, desiredLength)
		}
	}

	return res
}

// SafeJoin joins a base path with a user-provided path, preventing directory traversal
func SafeJoin(base string, userPath string) string {
	// Clean the user path to remove .. and other traversal elements
	userPath = filepath.Clean(userPath)

	// Split and filter out traversal elements just in case Clean didn't handle everything as expected for "safe" join
	parts := strings.Split(userPath, string(filepath.Separator))
	var safeParts []string
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			continue
		}
		safeParts = append(safeParts, p)
	}

	return filepath.Join(append([]string{base}, safeParts...)...)
}

// Relativize removes leading slashes and drive letters
func Relativize(path string) string {
	// Remove drive letter on Windows
	if len(path) >= 2 && path[1] == ':' {
		path = path[2:]
	}

	// Remove leading slashes
	path = strings.TrimLeft(path, `/\`)

	return path
}

// StripMountSyntax is a repeated relativize
func StripMountSyntax(path string) string {
	return Relativize(path)
}

// IsEmptyFolder checks if a folder contains no files (recursively)
func IsEmptyFolder(path string) bool {
	empty := true
	err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil && info.Size() > 0 {
				empty = false
				return filepath.SkipDir // Found a non-empty file, can stop
			}
		}
		return nil
	})
	if err != nil {
		return false
	}
	return empty
}

// FolderSize calculates the total size of all files in a folder (recursively)
func FolderSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
}

// PathTupleFromURL returns (parentDir, filename) from a URL
func PathTupleFromURL(rawURL string) (string, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", filepath.Base(rawURL)
	}

	path := u.Path
	host := strings.ReplaceAll(u.Host, ":", ".")

	if path == "" || path == "/" {
		return host, ""
	}

	filename := filepath.Base(path)
	parent := SafeJoin(host, filepath.Dir(path))

	return StripMountSyntax(parent), filename
}

type CleanPathOptions struct {
	MaxNameLen       int
	DotSpace         bool
	CaseInsensitive  bool
	LowercaseFolders bool
	DedupeParts      bool
}

func CleanPath(path string, opts CleanPathOptions) string {
	if opts.MaxNameLen == 0 {
		opts.MaxNameLen = 255
	}

	pre := ""
	sep := string(filepath.Separator)
	if strings.HasPrefix(path, sep) {
		pre = sep
		path = strings.TrimPrefix(path, sep)
	} else if len(path) >= 2 && path[1] == ':' {
		pre = path[:2]
		path = path[2:]
	}

	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(filepath.Base(path), ext)
	dir := filepath.Dir(path)

	// Split directory into parts
	var parts []string
	if dir != "." && dir != "" {
		parts = strings.Split(dir, sep)
	}

	var cleanParts []string
	for _, p := range parts {
		if p == "." || p == "" || p == sep {
			continue
		}
		cp := CleanString(p)
		cp = strings.TrimLeft(cp, " -")
		cp = strings.TrimRight(cp, " -_.")
		if cp == "" {
			cp = "_"
		}

		if opts.LowercaseFolders {
			cp = strings.ToLower(cp)
		} else if opts.CaseInsensitive {
			if strings.ContainsAny(cp, " _.") {
				cp = Title(cp)
			} else {
				cp = strings.ToLower(cp)
			}
		}
		cleanParts = append(cleanParts, cp)
	}

	if opts.DedupeParts {
		cleanParts = OrderedSet(cleanParts)
	}

	// Clean stem
	cleanStem := CleanString(stem)
	cleanStem = strings.TrimLeft(cleanStem, " -")
	cleanStem = strings.TrimRight(cleanStem, " -.")

	// Shorten stem if too long
	fsLimit := opts.MaxNameLen - len(ext) - 1
	if len(cleanStem) > fsLimit && fsLimit > 3 {
		cleanStem = ShortenMiddle(cleanStem, fsLimit)
	}

	res := filepath.Join(append(cleanParts, cleanStem)...) + ext
	if opts.DotSpace {
		res = strings.ReplaceAll(res, " ", ".")
	}

	return pre + res
}

// FilterPath checks if a path matches PathFilterFlags
func FilterPath(path string, flags models.PathFilterFlags) bool {
	if len(flags.Paths) > 0 {
		matched := slices.Contains(flags.Paths, path)
		if !matched {
			return false
		}
	}

	if len(flags.Include) > 0 {
		matched := false
		for _, pattern := range flags.Include {
			if strings.Contains(path, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(flags.Exclude) > 0 {
		for _, pattern := range flags.Exclude {
			if strings.Contains(path, pattern) {
				return false
			}
		}
	}

	if flags.Regex != "" {
		if matched, _ := regexp.MatchString(flags.Regex, path); !matched {
			return false
		}
	}

	if len(flags.PathContains) > 0 {
		for _, s := range flags.PathContains {
			if !strings.Contains(path, s) {
				return false
			}
		}
	}

	return true
}
