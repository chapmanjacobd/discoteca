package pathutil_test

import (
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils/pathutil"
)

func TestIsAbs(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Unix paths
		{"/home/user", true},
		{"/", true},
		{"/var/log", true},

		// Windows paths
		{"C:\\Users\\user", true},
		{"C:/Users/user", true},
		{"D:\\data", true},
		{"Z:\\", true},

		// UNC paths
		{"\\\\server\\share\\path", true},
		{"\\\\server\\share", true},
		{"//server/share/path", true},

		// Relative paths
		{"relative/path", false},
		{"./relative", false},
		{"../parent", false},
		{"dir\\subdir", false},

		// Edge cases
		{"", false},
		{".", false},
		{"..", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := pathutil.IsAbs(tt.path); got != tt.want {
				t.Errorf("IsAbs(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		path      string
		wantParts []string
		wantAbs   bool
	}{
		// Unix paths
		{"/home/user", []string{"home", "user"}, true},
		{"/", []string{}, true},
		{"/var/log/syslog", []string{"var", "log", "syslog"}, true},

		// Windows paths (drive letter preserved)
		{"C:\\Users\\user", []string{"C:", "Users", "user"}, true},
		{"C:/Users/user", []string{"C:", "Users", "user"}, true},
		{"D:\\data\\file.txt", []string{"D:", "data", "file.txt"}, true},

		// UNC paths
		{"\\\\server\\share\\file", []string{"server", "share", "file"}, true},
		{"//server/share/file", []string{"server", "share", "file"}, true},

		// Relative paths
		{"relative/path", []string{"relative", "path"}, false},
		{"dir\\subdir\\file", []string{"dir", "subdir", "file"}, false},
		{"file.txt", []string{"file.txt"}, false},

		// Edge cases
		{"", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			parts, isAbs := pathutil.Split(tt.path)
			if len(parts) != len(tt.wantParts) {
				t.Errorf(
					"Split(%q) parts = %v (len=%d), want %v (len=%d)",
					tt.path,
					parts,
					len(parts),
					tt.wantParts,
					len(tt.wantParts),
				)
			}
			for i := range parts {
				if i < len(tt.wantParts) && parts[i] != tt.wantParts[i] {
					t.Errorf("Split(%q) part[%d] = %q, want %q", tt.path, i, parts[i], tt.wantParts[i])
				}
			}
			if isAbs != tt.wantAbs {
				t.Errorf("Split(%q) isAbs = %v, want %v", tt.path, isAbs, tt.wantAbs)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		parts         []string
		addLeadingSep bool
		want          string
	}{
		// With leading separator
		{[]string{"home", "user"}, true, sep + "home" + sep + "user"},
		{[]string{"home"}, true, sep + "home"},
		{[]string{}, true, sep},

		// Without leading separator
		{[]string{"home", "user"}, false, "home" + sep + "user"},
		{[]string{"single"}, false, "single"},
		{[]string{}, false, ""},

		// Windows drive letter
		{[]string{"C:", "Users"}, true, "C:" + sep + "Users"},
		{[]string{"C:"}, true, "C:" + sep},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := pathutil.Join(tt.parts, tt.addLeadingSep)
			if got != tt.want {
				t.Errorf("Join(%v, %v) = %q, want %q", tt.parts, tt.addLeadingSep, got, tt.want)
			}
		})
	}
}

func TestPrefix(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		path string
		want string
	}{
		// Unix paths
		{"/home/user", sep},
		{"/", sep},

		// Windows paths
		{"C:\\Users", "C:" + sep},
		{"C:/Users", "C:" + sep},
		{"D:\\", "D:" + sep},

		// UNC paths (backslash only - forward slash UNC is not standard)
		{"\\\\server\\share\\file", "\\\\server\\share"},
		{"\\\\server\\share", "\\\\server\\share"},

		// Relative paths
		{"relative", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := pathutil.Prefix(tt.path)
			if got != tt.want {
				t.Errorf("Prefix(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestCrossPlatformConsistency verifies that path functions work correctly
// for both Unix and Windows paths regardless of the OS they're running on.
func TestCrossPlatformConsistency(t *testing.T) {
	t.Run("Unix paths on any OS", func(t *testing.T) {
		paths := []string{
			"/home/user/file.txt",
			"/var/log",
			"/",
		}
		for _, p := range paths {
			_, isAbs := pathutil.Split(p)
			if !isAbs {
				t.Errorf("Unix path %q should be absolute", p)
			}
		}
	})

	t.Run("Windows paths on any OS", func(t *testing.T) {
		paths := []string{
			"C:\\Users\\file.txt",
			"D:\\data\\folder",
		}
		for _, p := range paths {
			parts, isAbs := pathutil.Split(p)
			if !isAbs {
				t.Errorf("Windows path %q should be absolute", p)
			}
			if len(parts) == 0 || parts[0][len(parts[0])-1] != ':' {
				t.Errorf("Windows path %q should have drive letter as first component", p)
			}
		}
	})

	t.Run("UNC paths on any OS", func(t *testing.T) {
		paths := []string{
			"\\\\server\\share\\file.txt",
			"\\\\nas\\media\\movies",
		}
		for _, p := range paths {
			_, isAbs := pathutil.Split(p)
			if !isAbs {
				t.Errorf("UNC path %q should be absolute", p)
			}
		}
	})

	t.Run("Mixed separators", func(t *testing.T) {
		// Paths with mixed separators should still work
		paths := []string{
			"C:/Users\\file.txt",
			"folder\\subfolder/file.txt",
		}
		for _, p := range paths {
			parts, _ := pathutil.Split(p)
			if len(parts) == 0 {
				t.Errorf("Mixed separator path %q should have parts", p)
			}
		}
	})
}

// TestToURL tests converting filesystem paths to URL-safe paths.

// TestFromURL tests converting URL paths to filesystem paths.
func TestFromURL(t *testing.T) {
	isWindows := filepath.Separator == '\\'

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"empty", "", ""},
		{"unix simple", "/home/user", func() string {
			if isWindows {
				return "\\home\\user"
			}
			return "/home/user"
		}()},
		{"unix nested", "/var/log/syslog", func() string {
			if isWindows {
				return "\\var\\log\\syslog"
			}
			return "/var/log/syslog"
		}()},
		{"windows style", "C:/Users/file", func() string {
			if isWindows {
				return "C:\\Users\\file"
			}
			return "C:/Users/file"
		}()},
		{"relative unix", "home/user", func() string {
			if isWindows {
				return "home\\user"
			}
			return "home/user"
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.FromURL(tt.url)
			if got != tt.want {
				t.Errorf("FromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

// TestToURLFromURLRoundTrip verifies that ToURL(FromURL(x)) and FromURL(ToURL(x)) are identity operations.
// Note: Round-trip only works correctly on the native OS. On Linux, Windows paths won't round-trip
// because FromURL doesn't convert forward slashes to backslashes (that's a Windows-only operation).
