package pathutil

import (
	"path/filepath"
	"testing"
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
			if got := IsAbs(tt.path); got != tt.want {
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
			parts, isAbs := Split(tt.path)
			if len(parts) != len(tt.wantParts) {
				t.Errorf("Split(%q) parts = %v (len=%d), want %v (len=%d)", tt.path, parts, len(parts), tt.wantParts, len(tt.wantParts))
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
		parts       []string
		addLeadingSep bool
		want        string
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
			got := Join(tt.parts, tt.addLeadingSep)
			if got != tt.want {
				t.Errorf("Join(%v, %v) = %q, want %q", tt.parts, tt.addLeadingSep, got, tt.want)
			}
		})
	}
}

func TestDepth(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		// Unix paths
		{"/home/user", 2},
		{"/", 0},
		{"/var/log/syslog", 3},
		
		// Windows paths (drive counts as component)
		{"C:\\Users\\user\\file.txt", 4},
		{"C:\\", 1},
		{"D:\\data", 2},
		
		// UNC paths
		{"\\\\server\\share\\file", 3},
		{"\\\\server\\share", 2},
		
		// Relative paths
		{"relative", 1},
		{"a/b/c", 3},
		
		// Edge cases
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := Depth(tt.path); got != tt.want {
				t.Errorf("Depth(%q) = %d, want %d", tt.path, got, tt.want)
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
			got := Prefix(tt.path)
			if got != tt.want {
				t.Errorf("Prefix(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestHasLeadingSep(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/home", true},
		{"/", true},
		{"\\\\server\\share", true},
		{"C:\\Users", false},
		{"relative", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := HasLeadingSep(tt.path); got != tt.want {
				t.Errorf("HasLeadingSep(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestHasTrailingSep(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/home/", true},
		{"/home", false},
		{"C:\\Users\\", true},
		{"C:\\Users", false},
		{"\\\\server\\share\\", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := HasTrailingSep(tt.path); got != tt.want {
				t.Errorf("HasTrailingSep(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestEnsureTrailingSep(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		path string
		want string
	}{
		{"/home", "/home" + sep},
		{"/home/", "/home/"},
		{"C:\\Users", "C:\\Users" + sep},
		{"C:\\Users\\", "C:\\Users\\"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := EnsureTrailingSep(tt.path)
			if got != tt.want {
				t.Errorf("EnsureTrailingSep(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestStripTrailingSep(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/", "/home"},
		{"/home//", "/home"},
		{"C:\\Users\\", "C:\\Users"},
		{"C:\\Users\\\\", "C:\\Users"},
		{"/home", "/home"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := StripTrailingSep(tt.path)
			if got != tt.want {
				t.Errorf("StripTrailingSep(%q) = %q, want %q", tt.path, got, tt.want)
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
		for _, path := range paths {
			_, isAbs := Split(path)
			if !isAbs {
				t.Errorf("Unix path %q should be absolute", path)
			}
		}
	})

	t.Run("Windows paths on any OS", func(t *testing.T) {
		paths := []string{
			"C:\\Users\\file.txt",
			"D:\\data\\folder",
		}
		for _, path := range paths {
			parts, isAbs := Split(path)
			if !isAbs {
				t.Errorf("Windows path %q should be absolute", path)
			}
			if len(parts) == 0 || parts[0][len(parts[0])-1] != ':' {
				t.Errorf("Windows path %q should have drive letter as first component", path)
			}
		}
	})

	t.Run("UNC paths on any OS", func(t *testing.T) {
		paths := []string{
			"\\\\server\\share\\file.txt",
			"\\\\nas\\media\\movies",
		}
		for _, path := range paths {
			_, isAbs := Split(path)
			if !isAbs {
				t.Errorf("UNC path %q should be absolute", path)
			}
		}
	})

	t.Run("Mixed separators", func(t *testing.T) {
		// Paths with mixed separators should still work
		paths := []string{
			"C:/Users\\file.txt",
			"folder\\subfolder/file.txt",
		}
		for _, path := range paths {
			parts, _ := Split(path)
			if len(parts) == 0 {
				t.Errorf("Mixed separator path %q should have parts", path)
			}
		}
	})
}
