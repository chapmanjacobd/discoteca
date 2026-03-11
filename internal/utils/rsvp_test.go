package utils

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGenerateRSVPAss(t *testing.T) {
	text := "Hello world this is RSVP"
	wpm := 60 // 1 word per second
	ass, duration := GenerateRSVPAss(text, wpm)

	if duration != 5.0 {
		t.Errorf("expected duration 5.0, got %f", duration)
	}

	if !strings.Contains(ass, "Dialogue: 0,0:00:00.00,0:00:01.00,Default,,0,0,0,,Hello") {
		t.Errorf("ASS content missing first word or timing incorrect")
	}
	if !strings.Contains(ass, "Dialogue: 0,0:00:04.00,0:00:05.00,Default,,0,0,0,,RSVP") {
		t.Errorf("ASS content missing last word or timing incorrect")
	}
}

func TestExtractText(t *testing.T) {
	// Test plain text
	tmpFile, _ := os.CreateTemp("", "test*.txt")
	defer os.Remove(tmpFile.Name())
	content := "Test content"
	os.WriteFile(tmpFile.Name(), []byte(content), 0o644)

	text, err := ExtractText(tmpFile.Name())
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}
	if strings.TrimSpace(text) != content {
		t.Errorf("expected %q, got %q", content, text)
	}

	// Test empty file
	emptyFile, _ := os.CreateTemp("", "empty*.txt")
	defer os.Remove(emptyFile.Name())
	text, err = ExtractText(emptyFile.Name())
	if err != nil {
		t.Fatalf("ExtractText failed on empty file: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}

	// Test non-existent file
	_, err = ExtractText("/non/existent/path.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	// Test malformed PDF/EPUB (if ebook-convert is available)
	ebookConvertPath, _ := exec.LookPath("ebook-convert")
	if ebookConvertPath != "" {
		badPdf, _ := os.CreateTemp("", "bad*.pdf")
		defer os.Remove(badPdf.Name())
		os.WriteFile(badPdf.Name(), []byte("not a pdf"), 0o644)
		_, err = ExtractText(badPdf.Name())
		if err == nil {
			t.Error("expected error for malformed PDF, got nil")
		}

		badEpub, _ := os.CreateTemp("", "bad*.epub")
		defer os.Remove(badEpub.Name())
		os.WriteFile(badEpub.Name(), []byte("not a zip"), 0o644)
		_, err = ExtractText(badEpub.Name())
		if err == nil {
			t.Error("expected error for malformed EPUB, got nil")
		}
	}
}

func TestGenerateRSVPAss_Empty(t *testing.T) {
	ass, duration := GenerateRSVPAss("", 60)
	if ass != "" || duration != 0 {
		t.Errorf("expected empty string and 0 duration for empty input, got %q and %f", ass, duration)
	}

	ass, duration = GenerateRSVPAss("   ", 60)
	if ass != "" || duration != 0 {
		t.Errorf("expected empty string and 0 duration for whitespace input, got %q and %f", ass, duration)
	}
}
