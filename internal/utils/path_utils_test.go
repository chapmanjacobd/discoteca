package utils_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input    string
		opts     utils.CleanPathOptions
		expected string
	}{
		{"example.txt", utils.CleanPathOptions{}, "example.txt"},
		{"/folder/file.txt", utils.CleanPathOptions{}, "/folder/file.txt"},
		{"/ -folder- / -file-.txt", utils.CleanPathOptions{}, "/folder/file.txt"},
		{"/MyFolder/File.txt", utils.CleanPathOptions{LowercaseFolders: true}, "/myfolder/File.txt"},
		{"/my folder/File.txt", utils.CleanPathOptions{CaseInsensitive: true}, "/My Folder/File.txt"},
		{"/my folder/file.txt", utils.CleanPathOptions{DotSpace: true}, "/my.folder/file.txt"},
		{"3_seconds_ago.../Mike.webm", utils.CleanPathOptions{}, "3_seconds_ago/Mike.webm"},
		{"test/''/t", utils.CleanPathOptions{}, "test/_/t"},
	}

	for _, tt := range tests {
		got := utils.CleanPath(tt.input, tt.opts)
		// Normalize both to forward slashes for comparison
		gotNorm := strings.ReplaceAll(got, "\\", "/")
		expectedNorm := strings.ReplaceAll(tt.expected, "\\", "/")
		if gotNorm != expectedNorm {
			t.Errorf(
				"utils.CleanPath(%q) = %q (normalized: %q), want %q (normalized: %q)",
				tt.input,
				got,
				gotNorm,
				tt.expected,
				expectedNorm,
			)
		}
	}
}

func TestTrimPathSegments(t *testing.T) {
	tests := []struct {
		path          string
		desiredLength int
		expected      string
	}{
		{filepath.FromSlash("/aaaaaaaaaa/fans/001.jpg"), 16, filepath.FromSlash("/a/fans/001.jpg")},
		{filepath.FromSlash("/ao/bo/co/do/eo/fo/go/ho"), 13, filepath.FromSlash("/a/.../g/ho")},
		{filepath.FromSlash("/a/b/c"), 10, filepath.FromSlash("/a/b/c")},
		// Explicit Windows drive test (using backslashes manually to avoid filepath.FromSlash converting back to / on Linux)
		{"C:\\Users\\Username\\Videos\\Movie.mp4", 20, "C:\\U\\...\\V\\Movie.mp4"},
		{"C:\\ao\\bo\\co\\do\\eo\\fo\\go", 15, "C:\\a\\...\\f\\go"},
	}

	for _, tt := range tests {
		got := utils.TrimPathSegments(tt.path, tt.desiredLength)
		// Normalize both to forward slashes for comparison
		gotNorm := strings.ReplaceAll(got, "\\", "/")
		expectedNorm := strings.ReplaceAll(tt.expected, "\\", "/")
		if gotNorm != expectedNorm {
			t.Errorf(
				"utils.TrimPathSegments(%q, %d) = %q (normalized: %q), want %q (normalized: %q)",
				tt.path,
				tt.desiredLength,
				got,
				gotNorm,
				tt.expected,
				expectedNorm,
			)
		}
	}
}

func TestRelativize(t *testing.T) {
	got := utils.Relativize("/home/user/file")
	gotNorm := strings.ReplaceAll(got, "\\", "/")
	expectedNorm := "home/user/file"
	if gotNorm != expectedNorm {
		t.Errorf("utils.Relativize(/home/user/file) = %q (normalized: %q), want %q", got, gotNorm, expectedNorm)
	}
}

func TestSafeJoin(t *testing.T) {
	base := "/path/to/fakeroot"
	tests := []struct {
		userPath string
		expected string
	}{
		{"etc/passwd", filepath.FromSlash("/path/to/fakeroot/etc/passwd")},
		{"../../etc/passwd", filepath.FromSlash("/path/to/fakeroot/etc/passwd")},
		{"/absolute/path", filepath.FromSlash("/path/to/fakeroot/absolute/path")},
	}

	for _, tt := range tests {
		got := utils.SafeJoin(base, tt.userPath)
		if got != tt.expected {
			t.Errorf("utils.SafeJoin(%q, %q) = %q, want %q", base, tt.userPath, got, tt.expected)
		}
	}
}

func TestPathTupleFromURL(t *testing.T) {
	tests := []struct {
		url              string
		expectedParent   string
		expectedFilename string
	}{
		{"http://example.com/path/to/file.txt", filepath.FromSlash("example.com/path/to"), "file.txt"},
		{"https://www.example.org/another/file.jpg", filepath.FromSlash("www.example.org/another"), "file.jpg"},
		{"http://example.com/", "example.com", ""},
		{"invalid url", "", "invalid url"},
	}

	for _, tt := range tests {
		gotParent, gotFilename := utils.PathTupleFromURL(tt.url)
		if gotParent != tt.expectedParent || gotFilename != tt.expectedFilename {
			t.Errorf(
				"utils.PathTupleFromURL(%q) = (%q, %q), want (%q, %q)",
				tt.url,
				gotParent,
				gotFilename,
				tt.expectedParent,
				tt.expectedFilename,
			)
		}
	}
}

func TestRandomString(t *testing.T) {
	s := utils.RandomString(10)
	if len(s) != 10 {
		t.Errorf("utils.RandomString(10) len = %d, want 10", len(s))
	}
}

func TestRandomFilename(t *testing.T) {
	input := "test.txt"
	got := utils.RandomFilename(input)
	if filepath.Ext(got) != ".txt" {
		t.Errorf("utils.RandomFilename extension mismatch: %s", got)
	}
}

func TestStripMountSyntax(t *testing.T) {
	if got := utils.StripMountSyntax(filepath.FromSlash("/home/user")); got != filepath.FromSlash("home/user") {
		t.Errorf("utils.StripMountSyntax failed: %s", got)
	}
}

func TestFolderFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	if !utils.IsEmptyFolder(tmpDir) {
		t.Error("utils.IsEmptyFolder should be true for empty dir")
	}

	f, _ := os.Create(filepath.Join(tmpDir, "file.txt"))
	f.WriteString("hello")
	f.Close()

	if utils.IsEmptyFolder(tmpDir) {
		t.Error("utils.IsEmptyFolder should be false for non-empty dir")
	}

	if got := utils.FolderSize(tmpDir); got != 5 {
		t.Errorf("utils.FolderSize = %d, want 5", got)
	}
}
