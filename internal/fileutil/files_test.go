package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
)

func TestSampleHashFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "hash-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := "hello world"
	if _, err2 := f.WriteString(content); err2 != nil {
		t.Fatal(err2)
	}
	f.Close()

	// want (path string, threads int, gap float64, chunkSize int64)
	hash, err := fileutil.SampleHashFile(f.Name(), 1, 0.1, 1024)
	if err != nil {
		t.Fatalf("fileutil.SampleHashFile failed: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestFullHashFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "hash-full-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := "hello world full"
	if _, err2 := f.WriteString(content); err2 != nil {
		t.Fatal(err2)
	}
	f.Close()

	hash, err := fileutil.FullHashFile(f.Name())
	if err != nil {
		t.Fatalf("fileutil.FullHashFile failed: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestCommonPath(t *testing.T) {
	paths := []string{
		"/home/user/vids/v1.mp4",
		"/home/user/vids/v2.mp4",
		"/home/user/music/a1.mp3",
	}
	expected := filepath.FromSlash("/home/user")
	got := fileutil.CommonPath(paths)
	if got != expected {
		t.Errorf("fileutil.CommonPath expected %q, got %q", expected, got)
	}
}

func TestCommonPathFull(t *testing.T) {
	paths := []string{
		"/home/user/vids/action_movie_part1.mp4",
		"/home/user/vids/action_movie_part2.mp4",
		"/home/user/vids/action_movie_part3.mp4",
	}
	expected := filepath.FromSlash("/home/user/vids")
	got := fileutil.CommonPathFull(paths)
	if got != expected {
		t.Errorf("fileutil.CommonPathFull expected %q, got %q", expected, got)
	}
}

func TestMoveFile(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "src.txt")
	dst := filepath.Join(tempDir, "dst.txt")
	content := "move test"
	os.WriteFile(src, []byte(content), 0o644)

	if err := fileutil.MoveFile(src, dst); err != nil {
		t.Fatalf("fileutil.MoveFile failed: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("expected source file to be removed")
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}
	if string(got) != content {
		t.Errorf("expected content %q, got %q", content, string(got))
	}
}
