package utils

import (
	"fmt"
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
	tmpFile, _ := os.CreateTemp(t.TempDir(), "test*.txt")
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
	emptyFile, _ := os.CreateTemp(t.TempDir(), "empty*.txt")
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
		badPdf, _ := os.CreateTemp(t.TempDir(), "bad*.pdf")
		defer os.Remove(badPdf.Name())
		os.WriteFile(badPdf.Name(), []byte("not a pdf"), 0o644)
		_, err = ExtractText(badPdf.Name())
		if err == nil {
			t.Error("expected error for malformed PDF, got nil")
		}

		badEpub, _ := os.CreateTemp(t.TempDir(), "bad*.epub")
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

func TestCountWordsFast(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 1},
		{"single word", "hello", 1},
		{"two words", "hello world", 2},
		{"with newline", "hello\nworld", 2},
		{"with tab", "hello\tworld", 2},
		{"multiple spaces", "hello  world", 3},
		{"sentence", "The quick brown fox", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountWordsFast([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("CountWordsFast(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQuickWordCount(t *testing.T) {
	// Test plain text file with enough content to avoid size fallback
	tmpFile, _ := os.CreateTemp(t.TempDir(), "test*.txt")
	defer os.Remove(tmpFile.Name())
	// Create content with >300 words to avoid size-based fallback
	var content strings.Builder
	for range 35 {
		content.WriteString("The quick brown fox jumps over the lazy dog. ")
	}
	os.WriteFile(tmpFile.Name(), []byte(content.String()), 0o644)
	stat, _ := os.Stat(tmpFile.Name())

	count, err := QuickWordCount(tmpFile.Name(), stat.Size())
	if err != nil {
		t.Fatalf("QuickWordCount failed: %v", err)
	}
	// 35 * 9 = 315 words expected
	if count < 310 || count > 320 {
		t.Errorf("expected ~315 words, got %d", count)
	}

	// Test HTML file with sufficient content
	htmlFile, _ := os.CreateTemp(t.TempDir(), "test*.html")
	defer os.Remove(htmlFile.Name())
	var htmlContent strings.Builder
	htmlContent.WriteString("<html><body>")
	for i := range 35 {
		htmlContent.WriteString("<p>Hello world test paragraph number ")
		fmt.Fprintf(&htmlContent, "%d", i)
		htmlContent.WriteString("</p>")
	}
	htmlContent.WriteString("</body></html>")
	os.WriteFile(htmlFile.Name(), []byte(htmlContent.String()), 0o644)
	stat, _ = os.Stat(htmlFile.Name())

	count, err = QuickWordCount(htmlFile.Name(), stat.Size())
	if err != nil {
		t.Fatalf("QuickWordCount for HTML failed: %v", err)
	}
	// Should find approximately 35 * 6 = 210 words plus tag artifacts
	// CountWordsFast counts spaces from tag replacement too
	if count < 200 || count > 400 {
		t.Errorf("expected 200-400 words for HTML, got %d", count)
	}

	// Test non-existent file
	_, err = QuickWordCount("/non/existent/file.txt", 0)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestEstimateReadingDuration(t *testing.T) {
	tests := []struct {
		wordCount int
		expected  int64
		tolerance int64
	}{
		{0, 0, 0},
		{220, 60, 1},  // 220 wpm = 60 seconds
		{110, 30, 1},  // 110 words = 30 seconds
		{440, 120, 2}, // 440 words = 120 seconds
	}

	for _, tt := range tests {
		result := EstimateReadingDuration(tt.wordCount)
		diff := result - tt.expected
		if diff < 0 {
			diff = -diff
		}
		if diff > tt.tolerance {
			t.Errorf(
				"EstimateReadingDuration(%d) = %d, expected ~%d (±%d)",
				tt.wordCount,
				result,
				tt.expected,
				tt.tolerance,
			)
		}
	}
}

func TestEstimateWordCountFromSize(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		size     int64
		minWords int
		maxWords int
	}{
		{"plain text", ".txt", 4200, 900, 1100}, // ~1000 words
		{"markdown", ".md", 4200, 900, 1100},    // ~1000 words
		{"pdf", ".pdf", 7000, 800, 1200},        // ~1000 words with images
		{"epub", ".epub", 5500, 850, 1150},      // ~1000 words with markup
		{"html", ".html", 4500, 850, 1150},      // ~1000 words with tags
		{"docx", ".docx", 6500, 850, 1150},      // ~1000 words with XML
		{"comic", ".cbz", 50000, 800, 1200},     // mostly images
		{"djvu", ".djvu", 15000, 800, 1200},     // scanned document
		{"small file", ".txt", 42, 10, 20},      // minimum threshold
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateWordCountFromSize("test"+tt.ext, tt.size)
			if result < tt.minWords || result > tt.maxWords {
				t.Errorf("EstimateWordCountFromSize(%s, %d) = %d, expected %d-%d",
					tt.ext, tt.size, result, tt.minWords, tt.maxWords)
			}
		})
	}
}

func BenchmarkCountWordsFast(b *testing.B) {
	text := []byte("The quick brown fox jumps over the lazy dog. " +
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
		"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.")

	b.ResetTimer()
	for range b.N {
		CountWordsFast(text)
	}
}

func BenchmarkQuickWordCount_PlainText(b *testing.B) {
	// Create a temporary text file with ~1000 words
	tmpFile, _ := os.CreateTemp(b.TempDir(), "benchmark*.txt")
	defer os.Remove(tmpFile.Name())

	var content strings.Builder
	for range 100 {
		content.WriteString("The quick brown fox jumps over the lazy dog. ")
		content.WriteString("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ")
	}
	os.WriteFile(tmpFile.Name(), []byte(content.String()), 0o644)
	stat, _ := os.Stat(tmpFile.Name())

	b.ResetTimer()
	for range b.N {
		QuickWordCount(tmpFile.Name(), stat.Size())
	}
}

func BenchmarkQuickWordCount_HTML(b *testing.B) {
	// Create a temporary HTML file with ~1000 words
	tmpFile, _ := os.CreateTemp(b.TempDir(), "benchmark*.html")
	defer os.Remove(tmpFile.Name())

	var content strings.Builder
	content.WriteString("<html><body>")
	for range 100 {
		content.WriteString("<p>The quick brown fox jumps over the lazy dog. ")
		content.WriteString("Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>")
	}
	content.WriteString("</body></html>")
	os.WriteFile(tmpFile.Name(), []byte(content.String()), 0o644)
	stat, _ := os.Stat(tmpFile.Name())

	b.ResetTimer()
	for range b.N {
		QuickWordCount(tmpFile.Name(), stat.Size())
	}
}
