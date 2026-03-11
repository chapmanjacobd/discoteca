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

	hash1, err := SampleHashFile(f.Name(), 1, 0, 0)
	if err != nil {
		t.Fatalf("SampleHashFile failed: %v", err)
	}
	hash2, err := SampleHashFile(f.Name(), 1, 0, 0)
	if err != nil {
		t.Fatalf("SampleHashFile failed: %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("Hashes for same file differ: %s, %s", hash1, hash2)
	}

	if _, err := SampleHashFile("/non/existent", 1, 0, 0); err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFullHashFile(t *testing.T) {
	f, err := os.CreateTemp("", "full-hash-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := "hello world"
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	hash, err := FullHashFile(f.Name())
	if err != nil {
		t.Fatalf("FullHashFile failed: %v", err)
	}
	if hash == "" {
		t.Error("Empty hash")
	}
}

func TestSimulationFunctions(t *testing.T) {
	flags := models.GlobalFlags{
		CoreFlags: models.CoreFlags{Simulate: true},
	}
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
		t.Errorf("AltName should be different for existing file: %s", alt)
	}

	nonExistent := filepath.Join(os.TempDir(), "non-existent-file-xyz.txt")
	if got := AltName(nonExistent); got != nonExistent {
		t.Errorf("AltName mismatch for non-existent file: %s", got)
	}
}

func TestGetExternalSubtitles(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "subs-test")
	defer os.RemoveAll(tmpDir)

	video := filepath.Join(tmpDir, "movie.mp4")
	os.WriteFile(video, nil, 0o644)
	os.WriteFile(filepath.Join(tmpDir, "movie.srt"), nil, 0o644)
	os.WriteFile(filepath.Join(tmpDir, "movie.en.srt"), nil, 0o644)

	subs := GetExternalSubtitles(video)
	if len(subs) != 2 {
		t.Errorf("Expected 2 subtitles, got %d: %v", len(subs), subs)
	}
}

func TestGetExternalSubtitles_MorePatterns(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "subs-test-patterns")
	defer os.RemoveAll(tmpDir)

	video := filepath.Join(tmpDir, "movie.mp4")
	os.WriteFile(video, nil, 0o644)

	// Create various subtitle patterns
	os.WriteFile(filepath.Join(tmpDir, "movie.srt"), nil, 0o644)          // exact match
	os.WriteFile(filepath.Join(tmpDir, "movie.en.srt"), nil, 0o644)       // dot notation
	os.WriteFile(filepath.Join(tmpDir, "movie.eng.srt"), nil, 0o644)      // 3-letter code
	os.WriteFile(filepath.Join(tmpDir, "movie_EN.srt"), nil, 0o644)       // underscore uppercase
	os.WriteFile(filepath.Join(tmpDir, "movie_es.srt"), nil, 0o644)       // underscore Spanish
	os.WriteFile(filepath.Join(tmpDir, "movie - French.srt"), nil, 0o644) // dash notation
	os.WriteFile(filepath.Join(tmpDir, "movie.ass"), nil, 0o644)          // different format
	os.WriteFile(filepath.Join(tmpDir, "movie.ja.ass"), nil, 0o644)       // Japanese ASS

	subs := GetExternalSubtitles(video)

	// Should find all subtitle files
	expectedMin := 8
	if len(subs) < expectedMin {
		t.Errorf("Expected at least %d subtitles, got %d: %v", expectedMin, len(subs), subs)
	}

	// Check that specific patterns are found
	foundPatterns := make(map[string]bool)
	for _, sub := range subs {
		base := filepath.Base(sub)
		foundPatterns[base] = true
	}

	expectedFiles := []string{
		"movie.srt", "movie.en.srt", "movie.eng.srt",
		"movie_EN.srt", "movie_es.srt", "movie.ass", "movie.ja.ass",
	}

	for _, expected := range expectedFiles {
		if !foundPatterns[expected] {
			t.Errorf("Expected to find %s, but it was not in the results: %v", expected, subs)
		}
	}
}

func TestExtractSubtitleInfo(t *testing.T) {
	tests := []struct {
		filename     string
		wantDisplay  string
		wantLangCode string
		wantCodec    string
	}{
		{"movie.en.srt", "English (srt)", "en", "srt"},
		{"movie_eng.ass", "English (ssa)", "eng", "ssa"}, // ass displayed as ssa
		{"movie.srt", "(srt)", "", "srt"},
		{"movie.EN.srt", "English (srt)", "en", "srt"},
		{"movie.es.vtt", "Spanish (vtt)", "es", "vtt"},
		{"movie_fra.ass", "French (ssa)", "fra", "ssa"}, // ass displayed as ssa
		{"movie.ja.srt", "Japanese (srt)", "ja", "srt"},
		{"movie.zh.srt", "Chinese (srt)", "zh", "srt"},
		{"movie.kor.ass", "Korean (ssa)", "kor", "ssa"},
		{"movie_deu.srt", "German (srt)", "deu", "srt"},
		{"movie - English.srt", "English (srt)", "en", "srt"}, // full language name supported
		{"movie.rus.vtt", "Russian (vtt)", "rus", "vtt"},
		{"movie.ara.srt", "Arabic (srt)", "ara", "srt"},
		{"movie.hin.ass", "Hindi (ssa)", "hin", "ssa"},
		{"movie.unknown.srt", "(srt)", "", "srt"},               // "unknown" is not a language code
		{"movie.part1.srt", "(srt)", "", "srt"},                 // "part1" is not a language code
		{"movie.cd1.en.srt", "English (srt)", "en", "srt"},      // should pick up "en" not "cd1"
		{"movie_es.srt", "Spanish (srt)", "es", "srt"},          // underscore pattern
		{"movie_ENG.srt", "English (srt)", "eng", "srt"},        // uppercase 3-letter
		{"movie - French.srt", "French (srt)", "fr", "srt"},     // full French name
		{"movie - Japanese.ass", "Japanese (ssa)", "ja", "ssa"}, // full name + ass
		{"movie - Deutsch.srt", "German (srt)", "de", "srt"},    // German full name
	}

	for _, tt := range tests {
		display, langCode, codec := ExtractSubtitleInfo(tt.filename)
		if display != tt.wantDisplay {
			t.Errorf("ExtractSubtitleInfo(%q) display = %q, want %q", tt.filename, display, tt.wantDisplay)
		}
		if langCode != tt.wantLangCode {
			t.Errorf("ExtractSubtitleInfo(%q) langCode = %q, want %q", tt.filename, langCode, tt.wantLangCode)
		}
		if codec != tt.wantCodec {
			t.Errorf("ExtractSubtitleInfo(%q) codec = %q, want %q", tt.filename, codec, tt.wantCodec)
		}
	}
}

func TestIsLanguageCode(t *testing.T) {
	validCodes := []string{"en", "es", "fr", "de", "eng", "spa", "fra", "jpn", "kor", "zho"}
	for _, code := range validCodes {
		if !isLanguageCode(code) {
			t.Errorf("isLanguageCode(%q) should be true", code)
		}
	}

	invalidCodes := []string{"a", "toolong", "part1", "cd2", "extended", "x", "xyz123"}
	for _, code := range invalidCodes {
		if isLanguageCode(code) {
			t.Errorf("isLanguageCode(%q) should be false", code)
		}
	}
}

func TestGetLanguageName(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"en", "English"},
		{"eng", "English"},
		{"es", "Spanish"},
		{"spa", "Spanish"},
		{"fr", "French"},
		{"fra", "French"},
		{"de", "German"},
		{"deu", "German"},
		{"ja", "Japanese"},
		{"jpn", "Japanese"},
		{"ko", "Korean"},
		{"kor", "Korean"},
		{"zh", "Chinese"},
		{"zho", "Chinese"},
		{"ru", "Russian"},
		{"rus", "Russian"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		got := getLanguageName(tt.code)
		if got != tt.want {
			t.Errorf("getLanguageName(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestFilterDeleted(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filter-deleted-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	f1 := filepath.Join(tempDir, "f1.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	f2 := filepath.Join(tempDir, "f2.txt")
	// f2 does not exist

	paths := []string{f1, f2}
	filtered := FilterDeleted(paths)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 path, got %d", len(filtered))
	}
	if filtered[0] != f1 {
		t.Errorf("Expected path %s, got %s", f1, filtered[0])
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

func TestIsFileOpen(t *testing.T) {
	f, err := os.CreateTemp("", "is-open-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Test closed file
	f.Close()
	if IsFileOpen(f.Name()) {
		t.Errorf("Expected file %s to be closed", f.Name())
	}

	// Test open file
	f2, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	// Note: on Linux, IsFileOpen checks /proc, so it should find its own open FD
	if !IsFileOpen(f.Name()) {
		t.Errorf("Expected file %s to be open", f.Name())
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
	expected := "/home/user"
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
	got := CommonPathFull(paths)
	expected := "/home/user/vids"
	if got != expected {
		t.Errorf("CommonPathFull expected %q, got %q", expected, got)
	}
}
