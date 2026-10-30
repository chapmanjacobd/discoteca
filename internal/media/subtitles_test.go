package media_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/media"
)

func TestGetExternalSubtitles(t *testing.T) {
	tmpDir := t.TempDir()

	movie := filepath.Join(tmpDir, "movie.mp4")
	os.WriteFile(movie, []byte(""), 0o644)

	srt := filepath.Join(tmpDir, "movie.srt")
	os.WriteFile(srt, []byte(""), 0o644)

	vtt := filepath.Join(tmpDir, "movie.en.vtt")
	os.WriteFile(vtt, []byte(""), 0o644)

	got := media.GetExternalSubtitles(movie)
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

	got := media.GetExternalSubtitles(movie)
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
		_, lang, _ := media.ExtractSubtitleInfo(tt.filename)
		if lang != tt.lang {
			t.Errorf("media.ExtractSubtitleInfo(%q) lang = %q, want %q", tt.filename, lang, tt.lang)
		}
	}
}

func TestIsLanguageCode(t *testing.T) {
	if !media.IsLanguageCode("en") {
		t.Error("Expected en to be language code")
	}
	if !media.IsLanguageCode("eng") {
		t.Error("Expected eng to be language code")
	}
	if media.IsLanguageCode("forced") {
		t.Error("Expected forced not to be language code")
	}
}

func TestGetLanguageName(t *testing.T) {
	if media.GetLanguageName("en") != "English" {
		t.Errorf("Expected English, got %s", media.GetLanguageName("en"))
	}
	if media.GetLanguageName("eng") != "English" {
		t.Errorf("Expected English, got %s", media.GetLanguageName("eng"))
	}
	if media.GetLanguageName("unknown") != "" {
		t.Errorf("Expected empty string, got %s", media.GetLanguageName("unknown"))
	}
}
