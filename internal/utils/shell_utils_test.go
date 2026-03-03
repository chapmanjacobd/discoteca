package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "''"},
		{"safe", "safe"},
		{"/path/to/file", "/path/to/file"},
		{"file with spaces", "'file with spaces'"},
		{"it's a file", "'it'\\''s a file'"},
		{"$", "'$'"},
	}

	for _, tt := range tests {
		got := ShellQuote(tt.input)
		if got != tt.expected {
			t.Errorf("ShellQuote(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResolveAbsolutePath(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "shell-test-*")
	defer os.RemoveAll(tmpDir)

	f := filepath.Join(tmpDir, "testfile")
	os.WriteFile(f, []byte("test"), 0o644)

	abs, _ := filepath.Abs(f)
	if got := ResolveAbsolutePath(f); got != abs {
		t.Errorf("ResolveAbsolutePath(%q) = %q, want %q", f, got, abs)
	}

	if got := ResolveAbsolutePath("nonexistent"); got != "nonexistent" {
		t.Errorf("ResolveAbsolutePath(nonexistent) = %q, want nonexistent", got)
	}
}

func TestFlattenWrapperFolder(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "flatten-test-*")
	defer os.RemoveAll(tmpDir)

	// struct: tmpDir/wrapper/file.txt
	wrapper := filepath.Join(tmpDir, "wrapper")
	os.Mkdir(wrapper, 0o755)
	file := filepath.Join(wrapper, "file.txt")
	os.WriteFile(file, []byte("data"), 0o644)

	if err := FlattenWrapperFolder(tmpDir); err != nil {
		t.Fatalf("FlattenWrapperFolder failed: %v", err)
	}

	if !FileExists(filepath.Join(tmpDir, "file.txt")) {
		t.Error("file.txt should be in the root folder")
	}
	if FileExists(wrapper) {
		t.Error("wrapper folder should be deleted")
	}
}

func TestFlattenWrapperFolderConflict(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "flatten-conflict-test-*")
	defer os.RemoveAll(tmpDir)

	// struct: tmpDir/wrapper/wrapper (file)
	wrapper := filepath.Join(tmpDir, "wrapper")
	os.Mkdir(wrapper, 0o755)
	file := filepath.Join(wrapper, "wrapper")
	os.WriteFile(file, []byte("conflict data"), 0o644)

	if err := FlattenWrapperFolder(tmpDir); err != nil {
		t.Fatalf("FlattenWrapperFolder failed: %v", err)
	}

	dstFile := filepath.Join(tmpDir, "wrapper")
	if !FileExists(dstFile) {
		t.Error("conflict file should be in the root folder")
	}

	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Error("Expected file, got directory")
	}
}
