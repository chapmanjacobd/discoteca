package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
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
	hash, err := utils.SampleHashFile(f.Name(), 1, 0.1, 1024)
	if err != nil {
		t.Fatalf("utils.SampleHashFile failed: %v", err)
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

	hash, err := utils.FullHashFile(f.Name())
	if err != nil {
		t.Fatalf("utils.FullHashFile failed: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestGetExternalSubtitles(t *testing.T) {
	tmpDir := t.TempDir()

	movie := filepath.Join(tmpDir, "movie.mp4")
	os.WriteFile(movie, []byte(""), 0o644)

	srt := filepath.Join(tmpDir, "movie.srt")
	os.WriteFile(srt, []byte(""), 0o644)

	vtt := filepath.Join(tmpDir, "movie.en.vtt")
	os.WriteFile(vtt, []byte(""), 0o644)

	got := utils.GetExternalSubtitles(movie)
	if len(got) != 2 {
		t.Errorf("Expected 2 subtitles, got %d", len(got))
	}
}

func TestGetExternalSubtitles_MorePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	movie := filepath.Join(tmpDir, "show S01E01.mkv")
	os.WriteFile(movie, []byte(""), 0o644)

	subs := []string{
		"show S01E01.en.srt",
		"show S01E01_eng.srt",
		"show S01E01.ES.srt",
	}

	for _, s := range subs {
		os.WriteFile(filepath.Join(tmpDir, s), []byte(""), 0o644)
	}

	got := utils.GetExternalSubtitles(movie)
	// We expect 3 matches
	if len(got) != 3 {
		t.Errorf("Expected 3 subtitles, got %d: %v", len(got), got)
	}
}

func TestExtractSubtitleInfo(t *testing.T) {
	tests := []struct {
		filename string
		lang     string
	}{
		{"movie.en.srt", "en"},
		{"movie_eng.srt", "eng"},
	}

	for _, tt := range tests {
		_, lang, _ := utils.ExtractSubtitleInfo(tt.filename)
		if lang != tt.lang {
			t.Errorf("utils.ExtractSubtitleInfo(%q) lang = %q, want %q", tt.filename, lang, tt.lang)
		}
	}
}

func TestIsLanguageCode(t *testing.T) {
	if !utils.IsLanguageCode("en") {
		t.Error("Expected en to be language code")
	}
	if !utils.IsLanguageCode("eng") {
		t.Error("Expected eng to be language code")
	}
	if utils.IsLanguageCode("forced") {
		t.Error("Expected forced not to be language code")
	}
}

func TestGetLanguageName(t *testing.T) {
	if utils.GetLanguageName("en") != "English" {
		t.Errorf("Expected English, got %s", utils.GetLanguageName("en"))
	}
	if utils.GetLanguageName("eng") != "English" {
		t.Errorf("Expected English, got %s", utils.GetLanguageName("eng"))
	}
	if utils.GetLanguageName("unknown") != "" {
		t.Errorf("Expected empty string, got %s", utils.GetLanguageName("unknown"))
	}
}

func TestCommonPath(t *testing.T) {
	paths := []string{
		"/home/user/vids/v1.mp4",
		"/home/user/vids/v2.mp4",
		"/home/user/music/a1.mp3",
	}
	expected := filepath.FromSlash("/home/user")
	got := utils.CommonPath(paths)
	if got != expected {
		t.Errorf("utils.CommonPath expected %q, got %q", expected, got)
	}
}

func TestCommonPathFull(t *testing.T) {
	paths := []string{
		"/home/user/vids/action_movie_part1.mp4",
		"/home/user/vids/action_movie_part2.mp4",
		"/home/user/vids/action_movie_part3.mp4",
	}
	expected := filepath.FromSlash("/home/user/vids")
	got := utils.CommonPathFull(paths)
	if got != expected {
		t.Errorf("utils.CommonPathFull expected %q, got %q", expected, got)
	}
}

func TestMoveFile(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "src.txt")
	dst := filepath.Join(tempDir, "dst.txt")
	content := "move test"
	os.WriteFile(src, []byte(content), 0o644)

	if err := utils.MoveFile(src, dst); err != nil {
		t.Fatalf("utils.MoveFile failed: %v", err)
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
