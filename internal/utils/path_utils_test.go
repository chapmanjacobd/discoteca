package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input    string
		opts     CleanPathOptions
		expected string
	}{
		{"example.txt", CleanPathOptions{}, "example.txt"},
		{"/folder/file.txt", CleanPathOptions{}, "/folder/file.txt"},
		{"/ -folder- / -file-.txt", CleanPathOptions{}, "/folder/file.txt"},
		{"/MyFolder/File.txt", CleanPathOptions{LowercaseFolders: true}, "/myfolder/File.txt"},
		{"/my folder/File.txt", CleanPathOptions{CaseInsensitive: true}, "/My Folder/File.txt"},
		{"/my folder/file.txt", CleanPathOptions{DotSpace: true}, "/my.folder/file.txt"},
		{"3_seconds_ago.../Mike.webm", CleanPathOptions{}, "3_seconds_ago/Mike.webm"},
		{"test/''/t", CleanPathOptions{}, "test/_/t"},
	}

	for _, tt := range tests {
		got := CleanPath(tt.input, tt.opts)
		// Normalize both to forward slashes for comparison
		gotNorm := strings.ReplaceAll(got, "\\", "/")
		expectedNorm := strings.ReplaceAll(tt.expected, "\\", "/")
		if gotNorm != expectedNorm {
			t.Errorf("CleanPath(%q) = %q (normalized: %q), want %q (normalized: %q)", tt.input, got, gotNorm, tt.expected, expectedNorm)
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
		{filepath.FromSlash("/ao/bo/co/do/eo/fo/go/ho"), 13, filepath.FromSlash("/a/b/.../g/ho")},
		{filepath.FromSlash("/a/b/c"), 10, filepath.FromSlash("/a/b/c")},
	}

	for _, tt := range tests {
		got := TrimPathSegments(tt.path, tt.desiredLength)
		// Normalize both to forward slashes for comparison
		gotNorm := strings.ReplaceAll(got, "\\", "/")
		expectedNorm := strings.ReplaceAll(tt.expected, "\\", "/")
		if gotNorm != expectedNorm {
			t.Errorf("TrimPathSegments(%q, %d) = %q (normalized: %q), want %q (normalized: %q)", tt.path, tt.desiredLength, got, gotNorm, tt.expected, expectedNorm)
		}
	}
}

func TestRelativize(t *testing.T) {
	got := Relativize("/home/user/file")
	gotNorm := strings.ReplaceAll(got, "\\", "/")
	expectedNorm := "home/user/file"
	if gotNorm != expectedNorm {
		t.Errorf("Relativize(/home/user/file) = %q (normalized: %q), want %q", got, gotNorm, expectedNorm)
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
		got := SafeJoin(base, tt.userPath)
		if got != tt.expected {
			t.Errorf("SafeJoin(%q, %q) = %q, want %q", base, tt.userPath, got, tt.expected)
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
		gotParent, gotFilename := PathTupleFromURL(tt.url)
		if gotParent != tt.expectedParent || gotFilename != tt.expectedFilename {
			t.Errorf("PathTupleFromURL(%q) = (%q, %q), want (%q, %q)", tt.url, gotParent, gotFilename, tt.expectedParent, tt.expectedFilename)
		}
	}
}

func TestRandomString(t *testing.T) {
	s := RandomString(10)
	if len(s) != 10 {
		t.Errorf("RandomString(10) len = %d, want 10", len(s))
	}
}

func TestRandomFilename(t *testing.T) {
	input := "test.txt"
	got := RandomFilename(input)
	if filepath.Ext(got) != ".txt" {
		t.Errorf("RandomFilename extension mismatch: %s", got)
	}
}

func TestStripMountSyntax(t *testing.T) {
	if got := StripMountSyntax(filepath.FromSlash("/home/user")); got != filepath.FromSlash("home/user") {
		t.Errorf("StripMountSyntax failed: %s", got)
	}
}

func TestFolderFunctions(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "folder-test")
	defer os.RemoveAll(tmpDir)

	if !IsEmptyFolder(tmpDir) {
		t.Error("IsEmptyFolder should be true for empty dir")
	}

	f, _ := os.Create(filepath.Join(tmpDir, "file.txt"))
	f.WriteString("hello")
	f.Close()

	if IsEmptyFolder(tmpDir) {
		t.Error("IsEmptyFolder should be false for non-empty dir")
	}

	if got := FolderSize(tmpDir); got != 5 {
		t.Errorf("FolderSize = %d, want 5", got)
	}
}
