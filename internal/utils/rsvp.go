package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// CountWordsFast estimates word count by counting spaces.
// This is much faster than strings.Fields() and sufficient for duration estimation.
func CountWordsFast(b []byte) int {
	return bytes.Count(b, []byte{' '}) + bytes.Count(b, []byte{'\n'}) + bytes.Count(b, []byte{'\t'}) + 1
}

// QuickWordCount extracts text and counts words for duration estimation.
// Optimized for speed over accuracy - suitable for ingest-time processing.
// Returns word count and error.
// For files with very low word counts (<300), falls back to size-based estimation
// to avoid false positives from sparse or image-heavy files.
func QuickWordCount(path string, size int64) (int, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt", ".md", ".log", ".ini", ".conf", ".cfg", ".text":
		// Plain text: read and count spaces
		content, err := os.ReadFile(path)
		if err != nil {
			return 0, err
		}
		count := CountWordsFast(content)
		// For very short files, use size-based estimate if it's higher
		if count < 300 {
			estimated := EstimateWordCountFromSize(path, size)
			if estimated > count {
				return estimated, nil
			}
		}
		return count, nil

	case ".html", ".htm":
		// HTML: strip tags and count
		content, err := os.ReadFile(path)
		if err != nil {
			return 0, err
		}
		// Quick HTML tag removal
		text := regexp.MustCompile(`<[^>]*>`).ReplaceAll(content, []byte{' '})
		count := CountWordsFast(text)
		// For short HTML files, use size-based estimate
		if count < 300 {
			estimated := EstimateWordCountFromSize(path, size)
			if estimated > count {
				return estimated, nil
			}
		}
		return count, nil

	case ".epub", ".mobi", ".azw3", ".docx", ".odt":
		// ZIP-based formats: extract HTML content without full conversion
		r, err := zip.OpenReader(path)
		if err != nil {
			return 0, err
		}
		defer r.Close()

		wordCount := 0
		for _, f := range r.File {
			name := strings.ToLower(f.Name)
			// Skip metadata, covers, and non-content files
			if strings.Contains(name, "cover") ||
				strings.Contains(name, "titlepage") ||
				strings.Contains(name, "metadata") ||
				strings.Contains(name, "nav.") {
				continue
			}

			if strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".xhtml") ||
				strings.HasSuffix(name, ".htm") || strings.HasSuffix(name, ".xml") {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				content, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}
				// Strip HTML tags
				text := regexp.MustCompile(`<[^>]*>`).ReplaceAll(content, []byte{' '})
				wordCount += CountWordsFast(text)
			}
		}

		// For ebooks with low extracted word count, use size-based estimate
		// This handles image-heavy ebooks or those with DRM/complex formatting
		if wordCount < 300 {
			estimated := EstimateWordCountFromSize(path, size)
			if estimated > wordCount {
				return estimated, nil
			}
		}
		return wordCount, nil

	case ".pdf":
		// Use pdftotext if available (much faster than calibre)
		cmd := exec.Command("pdftotext", "-raw", "-eol", "unix", path, "-")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			count := CountWordsFast(output)
			// For PDFs with low text extraction, use size-based estimate
			// This handles image-heavy or scanned PDFs
			if count < 300 {
				estimated := EstimateWordCountFromSize(path, size)
				if estimated > count {
					return estimated, nil
				}
			}
			return count, nil
		}
		// Fallback: use size-based estimation for PDFs
		return EstimateWordCountFromSize(path, size), nil

	default:
		// Try reading as plain text
		content, err := os.ReadFile(path)
		if err == nil {
			count := CountWordsFast(content)
			if count < 300 {
				estimated := EstimateWordCountFromSize(path, size)
				if estimated > count {
					return estimated, nil
				}
			}
			return count, nil
		}
		// Final fallback: pure size-based estimation
		return EstimateWordCountFromSize(path, size), nil
	}
}

// EstimateReadingDuration calculates reading duration in seconds from word count.
// Uses average reading speed of 220 words per minute.
func EstimateReadingDuration(wordCount int) int64 {
	if wordCount <= 0 {
		return 0
	}
	// 220 words per minute = 3.67 words per second
	// duration = wordCount / 3.67
	return int64(float64(wordCount) / 3.67)
}

// EstimateWordCountFromSize estimates word count from file size.
// Uses format-specific ratios to account for images, formatting, etc.
// Returns estimated word count.
func EstimateWordCountFromSize(path string, size int64) int {
	ext := strings.ToLower(filepath.Ext(path))

	// Bytes per word varies by format due to images, formatting, fonts, etc.
	// Lower ratio = more overhead per word (images, formatting)
	var bytesPerWord float64

	switch ext {
	case ".pdf":
		// PDFs often have images, fonts, complex formatting
		// Assume 6-8 bytes per word average
		bytesPerWord = 7.0
	case ".epub", ".mobi", ".azw3":
		// Ebooks have HTML markup, CSS, embedded fonts
		// Assume 5-6 bytes per word
		bytesPerWord = 5.5
	case ".docx", ".odt":
		// Office documents have XML overhead, styles, metadata
		// Assume 6-7 bytes per word
		bytesPerWord = 6.5
	case ".html", ".htm":
		// HTML has tags but usually less embedded content
		// Assume 4-5 bytes per word
		bytesPerWord = 4.5
	case ".cbz", ".cbr":
		// Comics are mostly images, text is minimal
		// Very high bytes per word
		bytesPerWord = 50.0
	case ".djvu":
		// DjVu is image-based, often scanned documents
		bytesPerWord = 15.0
	default:
		// Plain text: ~4.2 bytes per word (average English word + space)
		bytesPerWord = 4.2
	}

	// Calculate word count from size
	estimatedWords := max(
		// Sanity check: minimum 10 words for any file
		int(float64(size)/bytesPerWord), 10)

	return estimatedWords
}

// ExtractText extracts plain text from a given file path.
// Supports .txt, .pdf, .epub and other text formats.
// For ebook formats (PDF, EPUB, MOBI, etc.), it uses calibre's ebook-convert
// to convert to HTML and then extracts text from the HTML files.
func ExtractText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".log", ".ini", ".conf", ".cfg":
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case ".pdf", ".epub", ".mobi", ".azw", ".azw3", ".fb2", ".djvu", ".cbz", ".cbr", ".docx", ".odt", ".rtf", ".html", ".htm":
		// Use calibre conversion for all ebook formats
		htmlDir, err := ConvertEpubToOEB(path)
		if err != nil {
			return "", fmt.Errorf("conversion failed: %w", err)
		}

		// Get all HTML files in order
		htmlFiles := GetHTMLFiles(htmlDir)
		// GetHTMLFiles now returns sorted files

		var fullText strings.Builder
		for _, relPath := range htmlFiles {
			absPath := filepath.Join(htmlDir, relPath)
			text, err := extractTextFromHTMLFile(absPath)
			if err != nil {
				continue
			}
			fullText.WriteString(text)
			fullText.WriteString(" ")
		}
		return fullText.String(), nil
	default:
		// Fallback: try reading as text
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), nil
		}
		return "", fmt.Errorf("unsupported format: %s", ext)
	}
}

// extractTextFromHTMLFile reads an HTML file and returns plain text
func extractTextFromHTMLFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Simple regex-based HTML tag stripping
	// This is not perfect but sufficient for RSVP/TTS purposes on calibre output
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(string(content), " ")

	// Decode HTML entities (basic ones)
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")

	// Collapse whitespace
	text = strings.Join(strings.Fields(text), " ")

	return text, nil
}

// ConvertEpubToOEB converts EPUB/text documents to HTML format using calibre's ebook-convert.
// The converted files are stored in ~/.cache/disco with automatic cleanup of files older than 3 days.
// Returns the path to the converted HTML directory.
func ConvertEpubToOEB(inputPath string) (string, error) {
	// Check for ebook-convert
	ebookConvertBin := "ebook-convert"
	if _, err := exec.LookPath(ebookConvertBin); err != nil {
		return "", fmt.Errorf("ebook-convert not found (install calibre): %w", err)
	}

	// Create cache directory
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "disco")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Clean up old files (older than 3 days)
	cleanupOldCacheFiles(cacheDir, 3*24*time.Hour)

	// Generate output path based on input file name
	// Output to a directory (no extension) - calibre creates OEB/HTML structure
	// Sanitize the base name to avoid calibre misinterpreting it as a format
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	// Replace spaces and special chars with underscores for calibre compatibility
	safeBaseName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, baseName)
	// Limit length to avoid filesystem issues
	if len(safeBaseName) > 100 {
		safeBaseName = safeBaseName[:100]
	}
	outputDir := filepath.Join(cacheDir, safeBaseName)

	// Check if conversion already exists and is recent (less than 1 day old)
	if info, err := os.Stat(outputDir); err == nil && info.ModTime().After(time.Now().Add(-24*time.Hour)) {
		return outputDir, nil
	}

	// Remove existing output if it exists
	if err := os.RemoveAll(outputDir); err != nil {
		return "", fmt.Errorf("failed to remove existing output: %w", err)
	}

	// Run ebook-convert with HTML output
	// Output to a directory (no extension) creates an exploded HTML directory
	cmd := exec.Command(
		ebookConvertBin,
		inputPath,
		outputDir,
		"--output-profile", "tablet",
		"--pretty-print",
		"--minimum-line-height=105",
		"--unsmarten-punctuation",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ebook-convert failed: %w\n%s", err, string(output))
	}

	// Verify output was created
	if _, err := os.Stat(outputDir); err != nil {
		return "", fmt.Errorf("output directory not created: %w", err)
	}

	return outputDir, nil
}

// SanitizeFilename replaces special characters with underscores for calibre compatibility
func SanitizeFilename(name string) string {
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	if len(result) > 100 {
		result = result[:100]
	}
	return result
}

// cleanupOldCacheFiles removes files and directories older than the specified duration
func cleanupOldCacheFiles(cacheDir string, maxAge time.Duration) {
	now := time.Now()
	cutoff := now.Add(-maxAge)

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			fullPath := filepath.Join(cacheDir, entry.Name())
			os.RemoveAll(fullPath)
		}
	}
}

// GenerateRSVPAss generates an ASS subtitle file content for RSVP.
func GenerateRSVPAss(text string, wpm int) (string, float64) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return "", 0
	}

	durationPerWord := 60.0 / float64(wpm)
	totalDuration := float64(len(words)) * durationPerWord

	var sb strings.Builder
	sb.WriteString("[Script Info]\n")
	sb.WriteString("ScriptType: v4.00+\n")
	sb.WriteString("PlayResX: 1280\n")
	sb.WriteString("PlayResY: 720\n")
	sb.WriteString("\n")
	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	// Centered large text
	sb.WriteString("Style: Default,Arial,80,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2,0,5,10,10,10,1\n")
	sb.WriteString("\n")
	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	startTime := 0.0
	for _, word := range words {
		endTime := startTime + durationPerWord

		startStr := formatAssTime(startTime)
		endStr := formatAssTime(endTime)

		// Sanitize word for ASS
		word = strings.ReplaceAll(word, "{", "\\{")
		word = strings.ReplaceAll(word, "}", "\\}")

		// Highlight the middle character/part if possible (ORP - Optimal Recognition Point)
		// Simple implementation: just show the word
		sb.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startStr, endStr, word))
		startTime = endTime
	}

	return sb.String(), totalDuration
}

func formatAssTime(seconds float64) string {
	h := int(seconds / 3600)
	m := int((seconds - float64(h)*3600) / 60)
	s := seconds - float64(h)*3600 - float64(m)*60
	return fmt.Sprintf("%d:%02d:%05.2f", h, m, s)
}

// GenerateTTS generates a WAV file from text using espeak-ng.
func GenerateTTS(text string, outputPath string, wpm int) error {
	// Check for espeak-ng
	espeakBin := "espeak-ng"
	if _, err := exec.LookPath(espeakBin); err != nil {
		return fmt.Errorf("espeak-ng not found")
	}

	// Boost espeak speed slightly as it tends to drift slower than the calculated word timing
	espeakWpm := int(float64(wpm) * 1.1)
	cmd := exec.Command(espeakBin, "-w", outputPath, "-s", fmt.Sprintf("%d", espeakWpm))
	cmd.Stdin = strings.NewReader(text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("espeak-ng failed: %s: %s", err, string(output))
	}
	return nil
}

// GetHTMLFiles returns a list of HTML files in the directory sorted by filename
func GetHTMLFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".html" || ext == ".xhtml" || ext == ".htm" {
				base := strings.ToLower(filepath.Base(path))
				// Skip cover, titlepage, nav, and metadata files
				if !strings.Contains(base, "cover") &&
					!strings.Contains(base, "titlepage") &&
					!strings.Contains(base, "title_page") &&
					!strings.Contains(base, "nav.xhtml") &&
					!strings.Contains(base, "content.opf") {
					relPath, _ := filepath.Rel(dir, path)
					files = append(files, relPath)
				}
			}
		}
		return nil
	})

	// Sort files for consistent ordering
	sort.Strings(files)

	return files
}

// FindMainContentFile finds the main HTML content file in a calibre output directory
// Skips cover/metadata pages and finds the actual book content
func FindMainContentFile(oebDir string) string {
	// First, try to parse content.opf to find the actual content files
	opfPath := filepath.Join(oebDir, "content.opf")
	if content, err := os.ReadFile(opfPath); err == nil {
		// Parse OPF to find content files (skip cover)
		contentStr := string(content)
		// Look for itemref elements that reference content files
		// Skip items with idref containing "cover" or "title"
		lines := strings.Split(contentStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			lowerLine := strings.ToLower(line)
			if strings.Contains(line, "<itemref") &&
				!strings.Contains(lowerLine, "cover") &&
				!strings.Contains(lowerLine, "title") &&
				!strings.Contains(lowerLine, "nav") {
				// Extract idref value
				idrefMatch := strings.Index(line, `idref="`)
				if idrefMatch >= 0 {
					idrefStart := idrefMatch + 7
					idrefEnd := strings.Index(line[idrefStart:], `"`)
					if idrefEnd > 0 {
						idref := line[idrefStart : idrefStart+idrefEnd]
						// Find corresponding item with this id
						for _, itemLine := range lines {
							if strings.Contains(itemLine, `id="`+idref+`"`) && strings.Contains(itemLine, `href="`) {
								hrefStart := strings.Index(itemLine, `href="`) + 6
								hrefEnd := strings.Index(itemLine[hrefStart:], `"`)
								if hrefEnd > 0 {
									href := itemLine[hrefStart : hrefStart+hrefEnd]
									contentFile := filepath.Join(oebDir, href)
									if _, err := os.Stat(contentFile); err == nil {
										return contentFile
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Fallback: Find HTML files, preferring those that aren't cover/metadata
	var firstContentHTML string
	filepath.Walk(oebDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".html" || ext == ".xhtml" || ext == ".htm" {
				base := strings.ToLower(filepath.Base(path))
				// Skip cover, titlepage, and metadata files
				if strings.Contains(base, "cover") ||
					strings.Contains(base, "titlepage") ||
					strings.Contains(base, "title_page") ||
					strings.Contains(base, "nav.xhtml") {
					return nil
				}
				if firstContentHTML == "" {
					firstContentHTML = path
				}
				// Prefer files with chapter/content in the name
				if strings.Contains(base, "chapter") || strings.Contains(base, "content") || strings.Contains(base, "ch0") || strings.Contains(base, "split_") {
					firstContentHTML = path
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	if firstContentHTML != "" {
		return firstContentHTML
	}

	// Last resort: return index.html
	indexHtml := filepath.Join(oebDir, "index.html")
	if _, err := os.Stat(indexHtml); err == nil {
		return indexHtml
	}

	return ""
}
