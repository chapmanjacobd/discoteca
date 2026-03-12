// Package pathutil provides cross-platform path handling utilities.
// It centralizes path operations to handle both Unix and Windows paths,
// including UNC paths, consistently throughout the application.
package pathutil

import (
	"path/filepath"
	"strings"
)

// Split splits a path into its components, handling both forward and back slashes.
// For Windows paths with drive letters (C:\path), the drive letter is preserved as the first component.
// Returns (parts, isAbs) where parts contains the path components.
func Split(path string) ([]string, bool) {
	if path == "" {
		return []string{}, false
	}

	// Check if path is absolute (Unix, Windows, or UNC)
	isAbs := IsAbs(path)

	// Handle Windows drive letter specially
	if len(path) >= 2 && path[1] == ':' {
		// Extract drive letter as first component
		drive := path[:2]
		rest := path[2:]
		// Split the rest
		restParts := strings.FieldsFunc(rest, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		return append([]string{drive}, restParts...), isAbs
	}

	// Split on both separators
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	})

	return parts, isAbs
}

// Join joins path components using the OS-specific separator.
// If addLeadingSep is true, adds a leading separator (for absolute paths).
// Special handling: if first part looks like a drive letter (C:), adds separator after it.
func Join(parts []string, addLeadingSep bool) string {
	if len(parts) == 0 {
		if addLeadingSep {
			return string(filepath.Separator)
		}
		return ""
	}

	result := filepath.Join(parts...)
	if addLeadingSep {
		// Check if first part is a Windows drive letter
		if len(parts[0]) == 2 && parts[0][1] == ':' {
			// Drive letter already has separator from Join if there are more parts
			if len(parts) == 1 {
				result += string(filepath.Separator)
			}
		} else {
			result = string(filepath.Separator) + result
		}
	}
	return result
}

// IsAbs reports whether a path is absolute.
// Handles Unix paths (/path), Windows paths (C:\path), and UNC paths (\\server\share).
func IsAbs(path string) bool {
	if path == "" {
		return false
	}

	// Unix absolute path
	if path[0] == '/' {
		return true
	}

	// Windows drive letter (C:\)
	if len(path) >= 2 && path[1] == ':' {
		return true
	}

	// UNC path (\\server\share)
	if len(path) >= 2 && path[0] == '\\' && path[1] == '\\' {
		return true
	}

	return false
}

// HasLeadingSep checks if a path starts with a separator (either / or \ or \\).
func HasLeadingSep(path string) bool {
	if path == "" {
		return false
	}
	if path[0] == '/' {
		return true
	}
	if len(path) >= 2 && path[0] == '\\' && path[1] == '\\' {
		return true
	}
	return false
}

// Depth returns the number of path components.
// For absolute paths, this counts only the actual components (not the leading separator).
func Depth(path string) int {
	parts, _ := Split(path)
	return len(parts)
}

// Parent returns the parent directory of a path.
// Returns empty string if path has no parent.
func Parent(path string) string {
	return filepath.Dir(path)
}

// EnsureTrailingSep adds a trailing separator if not present.
func EnsureTrailingSep(path string) string {
	if path == "" {
		return path
	}
	if !strings.HasSuffix(path, "/") && !strings.HasSuffix(path, "\\") {
		return path + string(filepath.Separator)
	}
	return path
}

// HasTrailingSep checks if path ends with a separator.
func HasTrailingSep(path string) bool {
	if path == "" {
		return false
	}
	return strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\")
}

// StripTrailingSep removes trailing separator from path.
func StripTrailingSep(path string) string {
	for len(path) > 0 && (path[len(path)-1] == '/' || path[len(path)-1] == '\\') {
		path = path[:len(path)-1]
	}
	return path
}

// Prefix returns the prefix of an absolute path.
// For Unix: "/"
// For Windows: "C:\"
// For UNC: "\\server\share"
// For relative paths: ""
func Prefix(path string) string {
	if path == "" {
		return ""
	}

	// Unix
	if path[0] == '/' {
		return string(filepath.Separator)
	}

	// Windows drive
	if len(path) >= 2 && path[1] == ':' {
		prefix := path[:2] + string(filepath.Separator)
		return prefix
	}

	// UNC
	if len(path) >= 2 && path[0] == '\\' && path[1] == '\\' {
		// Find server and share
		parts := strings.FieldsFunc(path, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		if len(parts) >= 2 {
			return "\\\\" + parts[0] + "\\" + parts[1]
		} else if len(parts) == 1 {
			return "\\\\" + parts[0]
		}
		return "\\\\"
	}

	return ""
}
