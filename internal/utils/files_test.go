package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestSampleHashFile(t *testing.T) {
	f, err := os.CreateTemp("", "hash-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := "hello world"
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// want (path string, threads int, gap float64, chunkSize int64)
	hash, err := SampleHashFile(f.Name(), 1, 0.1, 1024)
	if err != nil {
		t.Fatalf("SampleHashFile failed: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestFullHashFile(t *testing.T) {
	f, err := os.CreateTemp("", "hash-full-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := "hello world full"
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	hash, err := FullHashFile(f.Name())
	if err != nil {
		t.Fatalf("FullHashFile failed: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestSimulationFunctions(t *testing.T) {
	flags := models.GlobalFlags{}
	flags.Simulate = true

	if err := Rename(flags, "src", "dst"); err != nil {
		t.Errorf("Rename failed in simulation: %v", err)
	}

	if err := Unlink(flags, "path"); err != nil {
		t.Errorf("Unlink failed in simulation: %v", err)
	}

	if err := Rmtree(flags, "path"); err != nil {
		t.Errorf("Rmtree failed in simulation: %v", err)
	}
}

func TestAltName(t *testing.T) {
	f, err := os.CreateTemp("", "alt-test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	alt := AltName(f.Name())
	if alt == f.Name() {
		t.Errorf("Expected different name, got %s", alt)
	}

	nonexistent := f.Name() + ".nonexistent"
	alt2 := AltName(nonexistent)
	if alt2 != nonexistent {
		t.Errorf("Expected %s, got %s", nonexistent, alt2)
	}
}

func TestGetExternalSubtitles(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "subs-test")
	defer os.RemoveAll(tmpDir)

	movie := filepath.Join(tmpDir, "movie.mp4")
	os.WriteFile(movie, []byte(""), 0o644)

	srt := filepath.Join(tmpDir, "movie.srt")
	os.WriteFile(srt, []byte(""), 0o644)

	vtt := filepath.Join(tmpDir, "movie.en.vtt")
	os.WriteFile(vtt, []byte(""), 0o644)

	got := GetExternalSubtitles(movie)
	if len(got) != 2 {
		t.Errorf("Expected 2 subtitles, got %d", len(got))
	}
}

func TestGetExternalSubtitles_MorePatterns(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "subs-test-2")
	defer os.RemoveAll(tmpDir)

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

	got := GetExternalSubtitles(movie)
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
		_, lang, _ := ExtractSubtitleInfo(tt.filename)
		if lang != tt.lang {
			t.Errorf("ExtractSubtitleInfo(%q) lang = %q, want %q", tt.filename, lang, tt.lang)
		}
	}
}

func TestIsLanguageCode(t *testing.T) {
	if !IsLanguageCode("en") {
		t.Error("Expected en to be language code")
	}
	if !IsLanguageCode("eng") {
		t.Error("Expected eng to be language code")
	}
	if IsLanguageCode("forced") {
		t.Error("Expected forced not to be language code")
	}
}

func TestGetLanguageName(t *testing.T) {
	if GetLanguageName("en") != "English" {
		t.Errorf("Expected English, got %s", GetLanguageName("en"))
	}
	if GetLanguageName("eng") != "English" {
		t.Errorf("Expected English, got %s", GetLanguageName("eng"))
	}
	if GetLanguageName("unknown") != "" {
		t.Errorf("Expected empty string, got %s", GetLanguageName("unknown"))
	}
}

func TestFilterDeleted(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "filter-deleted")
	defer os.RemoveAll(tmpDir)

	f1 := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(f1, []byte(""), 0o644)

	f2 := filepath.Join(tmpDir, "missing.txt")

	paths := []string{f1, f2}
	got := FilterDeleted(paths)

	if len(got) != 1 || got[0] != f1 {
		t.Errorf("Expected [%s], got %v", f1, got)
	}
}

func TestGetFileStats(t *testing.T) {
	f, err := os.CreateTemp("", "stats-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := "hello"
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	stats, err := GetFileStats(f.Name())
	if err != nil {
		t.Fatalf("GetFileStats failed: %v", err)
	}

	if stats.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), stats.Size)
	}
	if stats.TimeModified == 0 {
		t.Error("Expected non-zero TimeModified")
	}
}

func TestDetectMimeType(t *testing.T) {
	f, err := os.CreateTemp("", "mime-test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString("hello"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	mime := DetectMimeType(f.Name())
	if mime != "text/plain; charset=utf-8" {
		t.Errorf("Expected text/plain; charset=utf-8, got %s", mime)
	}
}

func TestCommonPath(t *testing.T) {
	paths := []string{
		"/home/user/vids/v1.mp4",
		"/home/user/vids/v2.mp4",
		"/home/user/music/a1.mp3",
	}
	expected := filepath.FromSlash("/home/user")
	got := CommonPath(paths)
	if got != expected {
		t.Errorf("CommonPath expected %q, got %q", expected, got)
	}
}

func TestCommonPathFull(t *testing.T) {
	paths := []string{
		"/home/user/vids/action_movie_part1.mp4",
		"/home/user/vids/action_movie_part2.mp4",
		"/home/user/vids/action_movie_part3.mp4",
	}
	expected := filepath.FromSlash("/home/user/vids")
	got := CommonPathFull(paths)
	if got != expected {
		t.Errorf("CommonPathFull expected %q, got %q", expected, got)
	}
}
