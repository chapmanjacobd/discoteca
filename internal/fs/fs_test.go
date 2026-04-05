package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/fs"
)

func TestFindMedia(t *testing.T) {
	tempDir := t.TempDir()

	// Create a dummy structure
	files := []string{
		"movie.mp4",
		"song.mp3",
		"readme.txt", // text files are now considered media
		"folder/clip.mkv",
		"folder/image.jpg",
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	found, err := fs.FindMedia(tempDir, nil)
	if err != nil {
		t.Fatalf("fs.FindMedia failed: %v", err)
	}

	expectedCount := 5 // mp4, mp3, txt, mkv, jpg
	if len(found) != expectedCount {
		t.Errorf("Expected %d media files, got %d: %v", expectedCount, len(found), found)
	}

	expectedFiles := []string{"movie.mp4", "song.mp3", "readme.txt", "clip.mkv", "image.jpg"}
	for _, ef := range expectedFiles {
		matched := false
		for path := range found {
			if filepath.ToSlash(filepath.Base(path)) == filepath.ToSlash(ef) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("Expected to find %s", ef)
		}
	}
}
