package utils

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CountWordsFast estimates word count by counting spaces.
// This is much faster than [strings.Fields] and sufficient for duration estimation.
func CountWordsFast(b []byte) int {
	return bytes.Count(b, []byte{' '}) + bytes.Count(b, []byte{'\n'}) + bytes.Count(b, []byte{'\t'}) + 1
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// QuickWordCount extracts text and counts words for duration estimation.
// Optimized for speed over accuracy - suitable for ingest-time processing.
func QuickWordCount(ctx context.Context, path string, size int64) (int, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var count int
	var err error

	switch ext {
	case ".txt", ".md", ".log", ".ini", ".conf", ".cfg", ".text":
		count, err = countWordsFromPlainText(path)
	case ".html", ".htm":
		count, err = countWordsFromHTML(path, size)
	case ".epub", ".mobi", ".azw3", ".docx", ".odt":
		count, err = countWordsFromEbooks(path)
	case ".pdf":
		count = countWordsFromPDF(ctx, path, size)
	default:
		count = countWordsFromDefault(path, size)
	}

	if err != nil {
		return 0, err
	}

	// For very short files, use size-based estimate if it's higher
	if count < 300 {
		estimated := EstimateWordCountFromSize(path, size)
		if estimated > count {
			return estimated, nil
		}
	}
	return count, nil
}

func countWordsFromPlainText(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	buf := make([]byte, 32*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			count += bytes.Count(buf[:n], []byte{' '}) +
				bytes.Count(buf[:n], []byte{'\n'}) +
				bytes.Count(buf[:n], []byte{'\t'})
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	return count + 1, nil // Add 1 for the last word
}

func countWordsFromHTML(path string, size int64) (int, error) {
	maxSize := int64(10 * 1024 * 1024)
	readSize := min(size, maxSize)

	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	content := make([]byte, readSize)
	_, err = io.ReadFull(f, content)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return 0, err
	}

	text := htmlTagRe.ReplaceAll(content, []byte{' '})
	return CountWordsFast(text), nil
}

func countWordsFromEbooks(path string) (int, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	wordCount := 0
	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if strings.Contains(name, "cover") || strings.Contains(name, "titlepage") ||
			strings.Contains(name, "metadata") || strings.Contains(name, "nav.") {

			continue
		}

		if strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".xhtml") ||
			strings.HasSuffix(name, ".htm") || strings.HasSuffix(name, ".xml") {

			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, err := readAllLimited(rc)
			rc.Close()
			if err != nil {
				continue
			}
			text := htmlTagRe.ReplaceAll(content, []byte{' '})
			wordCount += CountWordsFast(text)
		}
	}
	return wordCount, nil
}

func countWordsFromPDF(ctx context.Context, path string, size int64) int {
	cmd := exec.CommandContext(ctx, "pdftotext", "-raw", "-eol", "unix", path, "-")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return CountWordsFast(output)
	}
	return EstimateWordCountFromSize(path, size)
}

func countWordsFromDefault(path string, size int64) int {
	content, err := os.ReadFile(path)
	if err == nil {
		return CountWordsFast(content)
	}
	return EstimateWordCountFromSize(path, size)
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

// readAllLimited reads from an [io.Reader] up to 10MB limit.
func readAllLimited(r io.Reader) ([]byte, error) {
	// Use a limited reader to cap memory usage
	lr := io.LimitReader(r, 10*1024*1024)
	return io.ReadAll(lr)
}

// readFileLimited reads from a file up to a certain size.
func readFileLimited(path string, readSize int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	content := make([]byte, readSize)
	n, err := io.ReadFull(f, content)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	return string(content[:n]), nil
}

// ExtractText extracts plain text from a given file path.
func ExtractText(ctx context.Context, path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	// Limit text reading to 10MB to avoid memory exhaustion
	const maxTextSize = int64(10 * 1024 * 1024)
	readSize := min(stat.Size(), maxTextSize)

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	// ===== Plain text formats - read directly =====
	case ".txt", ".md", ".markdown", ".rst", ".asciidoc", ".adoc", ".tex", ".latex",
		".log", ".ini", ".conf", ".cfg", ".env", ".properties",
		".csv", ".tsv", ".json", ".jsonl", ".jsonld", ".yaml", ".yml", ".toml":
		return readFileLimited(path, readSize)

	case ".xml", ".svg", ".xhtml", ".xsl", ".xsd", ".plist":
		content, err := readFileLimited(path, readSize)
		if err != nil {
			return "", err
		}
		if ext == ".svg" || ext == ".xhtml" {
			return stripHTMLTags(content), nil
		}
		return content, nil

	case ".srt", ".vtt", ".ass", ".ssa", ".sub":
		content, err := readFileLimited(path, readSize)
		if err != nil {
			return "", err
		}
		return stripSubtitleTimings(content), nil

	// ===== HTML - strip tags =====
	case ".html", ".htm", ".mhtml", ".mht":
		content, err := readFileLimited(path, readSize)
		if err != nil {
			return "", err
		}
		return stripHTMLTags(content), nil

	// ===== Man pages =====
	case ".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8", ".9", ".man":
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return stripManPageFormatting(string(content)), nil

	// ===== Torrent metadata =====
	case ".torrent":
		return extractTorrentMetadata(path)

	// ===== Source code formats =====
	case ".py", ".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs",
		".go", ".rs", ".c", ".cc", ".cpp", ".cxx", ".h", ".hpp", ".hxx",
		".java", ".kt", ".kts", ".scala", ".sc",
		".rb", ".php", ".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd",
		".sql", ".graphql", ".gql", ".swift", ".m", ".mm":
		return extractTextFromCode(path, readSize)

	// ===== Document formats =====
	case ".rtf", ".pdf", ".ps", ".eps", ".epub", ".epub3",
		".odt", ".ods", ".odp", ".odg", ".odf", ".odm", ".ott", ".ots", ".otp",
		".docx", ".docm", ".dotx", ".dotm", ".xlsx", ".xlsm", ".xltx", ".xltm", ".xlsb",
		".pptx", ".pptm", ".potx", ".potm", ".ppsx", ".ppsm",
		".pages", ".numbers", ".key", ".doc", ".xls", ".ppt", ".chm",
		".mobi", ".azw", ".azw3", ".fb2", ".djvu", ".cbz", ".cbr":
		return extractTextFromDocuments(ctx, path)

	// ===== Archive formats =====
	case ".zip", ".jar", ".apk", ".aar", ".ipa",
		".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz", ".tar.zst", ".tzst", ".zst", ".zstd",
		".7z", ".rar":
		return extractTextFromArchives(ctx, path)
	}

	// Fallback: try calibre for all remaining ebook formats
	return extractTextWithCalibre(ctx, path)
}

func extractTextFromDocuments(ctx context.Context, path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if text, ok := extractTextFromBasicDocs(ctx, path, ext); ok {
		return text, nil
	}
	if text, ok := extractTextFromEbooksAndODF(ctx, path, ext); ok {
		return text, nil
	}
	if text, ok := extractTextFromOfficeDocs(ctx, path, ext); ok {
		return text, nil
	}

	switch ext {
	case ".mobi", ".azw", ".azw3", ".fb2", ".djvu", ".cbz", ".cbr":
		return extractTextWithCalibre(ctx, path)
	}
	return "", errors.New("not a document format")
}

func extractTextFromBasicDocs(ctx context.Context, path, ext string) (string, bool) {
	switch ext {
	case ".rtf":
		if text, err := extractTextFromRTF(ctx, path); err == nil {
			return text, true
		}
		if content, err := os.ReadFile(path); err == nil {
			return string(content), true
		}
	case ".pdf":
		if text, err := extractTextFromPDF(ctx, path); err == nil {
			return text, true
		}
		if text, err := extractTextWithCalibre(ctx, path); err == nil {
			return text, true
		}
	case ".ps", ".eps":
		if text, err := extractTextFromPS(ctx, path); err == nil {
			return text, true
		}
		if text, err := extractTextWithCalibre(ctx, path); err == nil {
			return text, true
		}
	default:
		return "", false
	}
	return "", true
}

func extractTextFromEbooksAndODF(ctx context.Context, path, ext string) (string, bool) {
	switch ext {
	case ".epub", ".epub3":
		if text, err := extractTextFromEPUB(path); err == nil && text != "" {
			return text, true
		}
		if text, err := extractTextWithCalibre(ctx, path); err == nil {
			return text, true
		}
	case ".odt", ".ods", ".odp", ".odg", ".odf", ".odm", ".ott", ".ots", ".otp":
		if text, err := extractTextFromOpenDocument(path); err == nil && text != "" {
			return text, true
		}
		if text, err := extractTextWithCalibre(ctx, path); err == nil {
			return text, true
		}
	default:
		return "", false
	}
	return "", true
}

func extractTextFromOfficeDocs(ctx context.Context, path, ext string) (string, bool) {
	var err error
	var text string
	switch ext {
	case ".docx", ".docm", ".dotx", ".dotm":
		text, err = extractTextFromDOCX(path)
	case ".xlsx", ".xlsm", ".xltx", ".xltm", ".xlsb":
		text, err = extractTextFromXLSX(path)
	case ".pptx", ".pptm", ".potx", ".potm", ".ppsx", ".ppsm":
		text, err = extractTextFromPPTX(path)
	case ".pages", ".numbers", ".key":
		text, err = extractTextFromIWork(path)
	case ".doc":
		text, err = extractTextFromDOC(ctx, path)
	case ".xls":
		text, err = extractTextFromXLS(ctx, path)
	case ".ppt":
		text, err = extractTextFromPPT(ctx, path)
	case ".chm":
		text, err = extractCHMContents(ctx, path)
	default:
		return "", false
	}

	if err == nil && text != "" {
		return text, true
	}
	if text, err := extractTextWithCalibre(ctx, path); err == nil {
		return text, true
	}
	return "", true
}

func extractTextFromCode(path string, readSize int64) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".py", ".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs",
		".go", ".rs", ".c", ".cc", ".cpp", ".cxx", ".h", ".hpp", ".hxx",
		".java", ".kt", ".kts", ".scala", ".sc",
		".rb", ".php", ".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd",
		".sql", ".graphql", ".gql", ".swift", ".m", ".mm":
		return readFileLimited(path, readSize)
	}
	return "", errors.New("not a source code format")
}

func extractTextFromArchives(ctx context.Context, path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".zip", ".jar", ".apk", ".aar", ".ipa":
		return listArchiveContents(path)

	case ".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz", ".tar.zst", ".tzst", ".zst", ".zstd":
		return listTarContents(ctx, path)

	case ".7z":
		return list7zContents(ctx, path)

	case ".rar":
		return listRarContents(ctx, path)
	}
	return "", errors.New("not an archive format")
}

// extractTextFromPDF extracts text from PDF using pdftotext
func extractTextFromPDF(ctx context.Context, path string) (string, error) {
	// Check for pdftotext (poppler-utils)
	pdftotextBin := "pdftotext"
	if _, err := exec.LookPath(pdftotextBin); err != nil {
		return "", errors.New("pdftotext not found")
	}

	// Run pdftotext with layout preservation
	cmd := exec.CommandContext(ctx, pdftotextBin, "-layout", "-q", path, "-")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// extractTextFromPS extracts text from PostScript using ps2ascii
func extractTextFromPS(ctx context.Context, path string) (string, error) {
	// Try ps2ascii first (comes with ghostscript)
	ps2asciiBin := "ps2ascii"
	if _, err := exec.LookPath(ps2asciiBin); err == nil {
		cmd := exec.CommandContext(ctx, ps2asciiBin, path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	// Try pstotext as alternative
	pstotextBin := "pstotext"
	if _, err := exec.LookPath(pstotextBin); err == nil {
		cmd := exec.CommandContext(ctx, pstotextBin, path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	return "", errors.New("no PostScript text extractor found")
}

// extractTextFromRTF extracts text from RTF using unrtf
func extractTextFromRTF(ctx context.Context, path string) (string, error) {
	// Check for unrtf
	unrtfBin := "unrtf"
	if _, err := exec.LookPath(unrtfBin); err != nil {
		return "", errors.New("unrtf not found")
	}

	// Run unrtf with text output
	cmd := exec.CommandContext(ctx, unrtfBin, "--text", path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// extractTextFromEPUB extracts text from EPUB using native zip reading
func extractTextFromEPUB(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Find content files (XHTML/HTML)
	var contentFiles []string
	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if strings.HasSuffix(name, ".xhtml") || strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".htm") {
			// Skip nav, cover, and metadata files
			if !strings.Contains(name, "nav.") && !strings.Contains(name, "cover") && !strings.Contains(name, "toc.") {
				contentFiles = append(contentFiles, f.Name)
			}
		}
	}

	if len(contentFiles) == 0 {
		return "", errors.New("no content files found in EPUB")
	}

	var fullText strings.Builder
	for _, fname := range contentFiles {
		rc, err := r.File[findFileIndex(r.File, fname)].Open()
		if err != nil {
			continue
		}
		// Limit each file extraction to 10MB
		content, err := readAllLimited(rc)
		rc.Close()
		if err != nil {
			continue
		}
		text := stripHTMLTags(string(content))
		if text != "" {
			fullText.WriteString(text)
			fullText.WriteString(" ")
		}
	}

	return strings.TrimSpace(fullText.String()), nil
}

// extractTextFromOpenDocument extracts text from ODT/ODS/ODP files
func extractTextFromOpenDocument(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Find content.xml in the archive
	var contentFile *zip.File
	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), "content.xml") {
			contentFile = f
			break
		}
	}

	if contentFile == nil {
		return "", errors.New("content.xml not found in OpenDocument")
	}

	rc, err := contentFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := readAllLimited(rc)
	if err != nil {
		return "", err
	}

	// Extract text from XML (strip all tags)
	text := stripXMLTags(string(content))
	return strings.TrimSpace(text), nil
}

// extractTextFromDOCX extracts text from DOCX using native zip reading
func extractTextFromDOCX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Find document.xml
	var docFile *zip.File
	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), "document.xml") {
			docFile = f
			break
		}
	}

	if docFile == nil {
		return "", errors.New("document.xml not found in DOCX")
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := readAllLimited(rc)
	if err != nil {
		return "", err
	}

	// Extract text from XML (simple approach - strip all tags)
	text := stripXMLTags(string(content))
	return strings.TrimSpace(text), nil
}

func extractTextFromZipXML(path, filePattern, noFilesErr string, extractFunc func(string) string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var targetFiles []string
	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if strings.Contains(name, filePattern) && strings.HasSuffix(name, ".xml") {
			targetFiles = append(targetFiles, f.Name)
		}
	}

	if len(targetFiles) == 0 {
		return "", errors.New(noFilesErr)
	}

	var fullText strings.Builder
	for _, fname := range targetFiles {
		idx := findFileIndex(r.File, fname)
		if idx < 0 {
			continue
		}
		rc, err := r.File[idx].Open()
		if err != nil {
			continue
		}
		content, err := readAllLimited(rc)
		rc.Close()
		if err != nil {
			continue
		}
		text := extractFunc(string(content))
		if text != "" {
			fullText.WriteString(text)
			fullText.WriteString(" ")
		}
	}

	return strings.TrimSpace(fullText.String()), nil
}

// extractTextFromXLSX extracts text from XLSX (cell values from all sheets)
func extractTextFromXLSX(path string) (string, error) {
	return extractTextFromZipXML(path, "worksheets/sheet", "no worksheets found in XLSX", extractXLSXCellValues)
}

// extractTextFromPPTX extracts text from PPTX (slide content)
func extractTextFromPPTX(path string) (string, error) {
	return extractTextFromZipXML(path, "slides/slide", "no slides found in PPTX", stripXMLTags)
}

// extractXLSXCellValues extracts cell values from XLSX worksheet XML
func extractXLSXCellValues(xml string) string {
	// Extract content from <v>...</v> tags (cell values)
	re := regexp.MustCompile(`<v[^>]*>([^<]*)</v>`)
	matches := re.FindAllStringSubmatch(xml, -1)

	var values []string
	for _, match := range matches {
		if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
			values = append(values, strings.TrimSpace(match[1]))
		}
	}

	return strings.Join(values, " ")
}

// stripSubtitleTimings removes timing markers from subtitle files
func stripSubtitleTimings(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip line numbers
		if _, err := strconv.Atoi(line); err == nil {
			continue
		}
		// Skip timing lines (contain --> or start with timestamp)
		if strings.Contains(line, "-->") || regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`).MatchString(line) {
			continue
		}
		// Skip SRT/ASS tags
		if strings.HasPrefix(line, "{\\") || strings.HasPrefix(line, "<") {
			continue
		}
		// Keep dialogue text
		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, " ")
}

// stripManPageFormatting removes troff/groff formatting from man pages
func stripManPageFormatting(content string) string {
	// Remove troff formatting commands (lines starting with .)
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(line, ".") {
			continue
		}
		// Remove inline formatting
		line = strings.ReplaceAll(line, "\\fB", "")
		line = strings.ReplaceAll(line, "\\fR", "")
		line = strings.ReplaceAll(line, "\\fI", "")
		line = strings.ReplaceAll(line, "\\&", "")
		result = append(result, line)
	}

	return strings.Join(result, " ")
}

// extractTextFromIWork extracts text from Apple iWork files (Pages, Numbers, Keynote)
func extractTextFromIWork(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Find index.xml or main.xml in the archive
	var indexFile *zip.File
	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if strings.HasSuffix(name, "index.xml") || strings.HasSuffix(name, "main.xml") {
			indexFile = f
			break
		}
	}

	if indexFile == nil {
		return "", errors.New("index.xml not found in iWork file")
	}

	rc, err := indexFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Extract text from XML (strip tags)
	text := stripXMLTags(string(content))
	return strings.TrimSpace(text), nil
}

// extractTextFromDOC extracts text from old .doc files using catdoc
func extractTextFromDOC(ctx context.Context, path string) (string, error) {
	// Try catdoc first
	catdocBin := "catdoc"
	if _, err := exec.LookPath(catdocBin); err == nil {
		cmd := exec.CommandContext(ctx, catdocBin, path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	// Try docx2txt as alternative (some versions support .doc)
	docx2txtBin := "docx2txt"
	if _, err := exec.LookPath(docx2txtBin); err != nil {
		cmd := exec.CommandContext(ctx, docx2txtBin, path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	return "", errors.New("no DOC extractor found (install catdoc)")
}

// extractTextFromXLS extracts text from old .xls files
func extractTextFromXLS(ctx context.Context, path string) (string, error) {
	// Try xls2csv first
	xls2csvBin := "xls2csv"
	if _, err := exec.LookPath(xls2csvBin); err != nil {
		cmd := exec.CommandContext(ctx, xls2csvBin, path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	// Try catdoc (also handles XLS)
	catdocBin := "catdoc"
	if _, err := exec.LookPath(catdocBin); err != nil {
		cmd := exec.CommandContext(ctx, catdocBin, path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	return "", errors.New("no XLS extractor found (install xls2csv or catdoc)")
}

// extractTextFromPPT extracts text from old .ppt files
func extractTextFromPPT(ctx context.Context, path string) (string, error) {
	// Try catdoc (supports PPT)
	catdocBin := "catdoc"
	if _, err := exec.LookPath(catdocBin); err != nil {
		cmd := exec.CommandContext(ctx, catdocBin, path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	return "", errors.New("no PPT extractor found (install catdoc)")
}

// listArchiveContents lists file names in a ZIP-based archive
func listArchiveContents(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var files []string
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			files = append(files, f.Name)
		}
	}

	return strings.Join(files, "\n"), nil
}

// listTarContents lists file names in a TAR archive
func listTarContents(ctx context.Context, path string) (string, error) {
	// Try tar command first
	tarBin := "tar"
	if _, err := exec.LookPath(tarBin); err != nil {
		cmd := exec.CommandContext(ctx, tarBin, "-tf", path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	// For zstd-compressed tar, try piping through zstd to tar
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".zst" || ext == ".tzst" || strings.HasSuffix(strings.ToLower(path), ".tar.zst") {
		zstdBin := "zstd"
		if _, err := exec.LookPath(zstdBin); err != nil {
			// zstd -d -c file | tar -tf -
			cmd := exec.CommandContext(
				ctx,
				"sh",
				"-c",
				"zstd -d -c "+strings.ReplaceAll(path, "'", "'\\''")+" | tar -tf -",
			)
			output, err := cmd.Output()
			if err == nil {
				return string(output), nil
			}
		}
	}

	// Fallback: try Go tar package
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Handle compression
	var reader io.Reader = f
	ext = strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".gz", ".tgz":
		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gzReader.Close()
		reader = gzReader
	case ".zst", ".tzst":
		// Zstd compression - use external zstd command
		zstdBin := "zstd"
		if _, err := exec.LookPath(zstdBin); err != nil {
			return "", errors.New("zstd not found")
		}
		cmd := exec.CommandContext(ctx, zstdBin, "-d", "-c", path)
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		reader = bytes.NewReader(output)
	}

	tr := tar.NewReader(reader)
	var files []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if header.Typeflag == tar.TypeReg {
			files = append(files, header.Name)
		}
	}

	return strings.Join(files, "\n"), nil
}

// list7zContents lists file names in a 7-Zip archive
func list7zContents(ctx context.Context, path string) (string, error) {
	// Try 7z first
	sevenzBin := "7z"
	if _, err := exec.LookPath(sevenzBin); err != nil {
		cmd := exec.CommandContext(ctx, sevenzBin, "l", "-ba", path)
		output, err := cmd.Output()
		if err == nil {
			return parse7zOutput(string(output)), nil
		}
	}

	// Try unar (The Unarchiver) as alternative
	unarBin := "unar"
	if _, err := exec.LookPath(unarBin); err != nil {
		cmd := exec.CommandContext(ctx, unarBin, "-t", path)
		output, err := cmd.Output()
		if err == nil {
			return parseUnarOutput(string(output)), nil
		}
	}

	return "", errors.New("7z extractor not found (install p7zip-full or unar)")
}

// parse7zOutput extracts file names from 7z list output
func parse7zOutput(output string) string {
	// Parse output to extract file names (lines 5+ contain file info)
	lines := strings.Split(output, "\n")
	var files []string
	for i, line := range lines {
		// Skip header and footer lines
		if i < 5 || strings.TrimSpace(line) == "" {
			continue
		}
		// Last line is usually summary
		if strings.HasPrefix(strings.TrimSpace(line), "-----") || strings.Contains(line, "files") {
			continue
		}
		// Extract filename (last column)
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			files = append(files, fields[len(fields)-1])
		}
	}

	return strings.Join(files, "\n")
}

// parseUnarOutput extracts file names from unar -t output
func parseUnarOutput(output string) string {
	lines := strings.Split(output, "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "Filename") {
			continue
		}
		files = append(files, line)
	}
	return strings.Join(files, "\n")
}

// listRarContents lists file names in a RAR archive
func listRarContents(ctx context.Context, path string) (string, error) {
	// Try unrar first
	unrarBin := "unrar"
	if _, err := exec.LookPath(unrarBin); err != nil {
		cmd := exec.CommandContext(ctx, unrarBin, "l", "-c-", path)
		output, err := cmd.Output()
		if err == nil {
			// Parse output to extract file names
			lines := strings.Split(string(output), "\n")
			var files []string
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 5 && fields[0] != "---------" {
					// Filename is typically the last field
					files = append(files, fields[len(fields)-1])
				}
			}
			return strings.Join(files, "\n"), nil
		}
	}

	// Try 7z as alternative
	sevenzBin := "7z"
	if _, err := exec.LookPath(sevenzBin); err != nil {
		cmd := exec.CommandContext(ctx, sevenzBin, "l", "-ba", path)
		output, err := cmd.Output()
		if err == nil {
			return parse7zOutput(string(output)), nil
		}
	}

	// Try unar (The Unarchiver) as alternative
	unarBin := "unar"
	if _, err := exec.LookPath(unarBin); err != nil {
		cmd := exec.CommandContext(ctx, unarBin, "-t", path)
		output, err := cmd.Output()
		if err == nil {
			return parseUnarOutput(string(output)), nil
		}
	}

	return "", errors.New("no RAR extractor found (install unrar, p7zip-full, or unar)")
}

// extractTorrentMetadata extracts metadata from a .torrent file
func extractTorrentMetadata(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Parse bencoded data (simple parser for common torrent fields)
	data := string(content)
	var metadata []string

	// Extract announce URL
	if idx := strings.Index(data, "8:announce"); idx != -1 {
		if endIdx := strings.Index(data[idx:], "e"); endIdx != -1 {
			url := data[idx+12 : idx+endIdx]
			metadata = append(metadata, "Announce: "+url)
		}
	}

	// Extract info.name (torrent name)
	if idx := strings.Index(data, "4:name"); idx != -1 {
		if endIdx := strings.Index(data[idx:], "e"); endIdx != -1 {
			name := data[idx+8 : idx+endIdx]
			metadata = append(metadata, "Name: "+name)
		}
	}

	// Extract creation date
	if idx := strings.Index(data, "13:creation date"); idx != -1 {
		if endIdx := strings.Index(data[idx:], "e"); endIdx != -1 {
			date := data[idx+17 : idx+endIdx]
			metadata = append(metadata, "Created: "+date)
		}
	}

	// Extract comment
	if idx := strings.Index(data, "7:comment"); idx != -1 {
		if endIdx := strings.Index(data[idx:], "e"); endIdx != -1 {
			comment := data[idx+10 : idx+endIdx]
			metadata = append(metadata, "Comment: "+comment)
		}
	}

	// Extract created by
	if idx := strings.Index(data, "10:created by"); idx != -1 {
		if endIdx := strings.Index(data[idx:], "e"); endIdx != -1 {
			creator := data[idx+14 : idx+endIdx]
			metadata = append(metadata, "By: "+creator)
		}
	}

	if len(metadata) == 0 {
		return data, nil
	}

	return strings.Join(metadata, "\n"), nil
}

// extractCHMContents extracts text from CHM (Microsoft Compiled HTML) files
func extractCHMContents(ctx context.Context, path string) (string, error) {
	// Try extract_chm if available
	extractChmBin := "extract_chm"
	if _, err := exec.LookPath(extractChmBin); err == nil {
		// Create temp directory for extraction
		tmpDir, err := os.MkdirTemp("", "chm-extract-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(tmpDir)

		cmd := exec.CommandContext(ctx, extractChmBin, "-l", tmpDir, path)
		if err := cmd.Run(); err != nil {
			return "", err
		}

		// Read extracted HTML files
		var content strings.Builder
		_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() &&
				(strings.HasSuffix(strings.ToLower(path), ".html") || strings.HasSuffix(strings.ToLower(path), ".htm")) {

				data, err := os.ReadFile(path)
				if err == nil {
					content.WriteString(stripHTMLTags(string(data)))
					content.WriteString(" ")
				}
			}
			return nil
		})

		return strings.TrimSpace(content.String()), nil
	}

	// Try chmcmd (from libmspack)
	chmcmdBin := "chmcmd"
	if _, err := exec.LookPath(chmcmdBin); err == nil {
		cmd := exec.CommandContext(ctx, chmcmdBin, "t", path)
		output, err := cmd.Output()
		if err == nil {
			return string(output), nil
		}
	}

	return "", errors.New("no CHM extractor found (install chmextractor or libmspack-tools)")
}

// extractTextWithCalibre uses calibre's ebook-convert as fallback
func extractTextWithCalibre(ctx context.Context, path string) (string, error) {
	htmlDir, err := ConvertEpubToOEB(ctx, path)
	if err != nil {
		return "", fmt.Errorf("calibre conversion failed: %w", err)
	}

	htmlFiles, err := GetHTMLFiles(htmlDir)
	if err != nil {
		return "", fmt.Errorf("failed to get HTML files: %w", err)
	}
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
}

// stripHTMLTags removes HTML tags and entities from text
func stripHTMLTags(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&ndash;", "-")
	text = strings.ReplaceAll(text, "&mdash;", "-")
	text = strings.ReplaceAll(text, "&lsquo;", "'")
	text = strings.ReplaceAll(text, "&rsquo;", "'")
	text = strings.ReplaceAll(text, "&ldquo;", "\"")
	text = strings.ReplaceAll(text, "&rdquo;", "\"")

	// Collapse whitespace
	return strings.Join(strings.Fields(text), " ")
}

// stripXMLTags removes XML tags from text (for DOCX)
func stripXMLTags(xml string) string {
	// Remove XML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(xml, " ")

	// Remove XML entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&apos;", "'")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	return strings.Join(strings.Fields(text), " ")
}

// findFileIndex finds the index of a file in a [zip.File] slice
func findFileIndex(files []*zip.File, name string) int {
	for i, f := range files {
		if f.Name == name {
			return i
		}
	}
	return -1
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
func ConvertEpubToOEB(ctx context.Context, inputPath string) (string, error) {
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
	cmd := exec.CommandContext(
		ctx,
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

	// Replace CSS with optimized version
	replaceCalibreCSS(outputDir)

	return outputDir, nil
}

// replaceCalibreCSS replaces the generated stylesheet with an optimized version
func replaceCalibreCSS(outputDir string) {
	cssPath := filepath.Join(outputDir, "stylesheet.css")
	// Optimized CSS for ebooks (matching Python implementation)
	css := `.calibre, body {
  font-family: Times New Roman,serif;
  display: block;
  font-size: 1em;
  padding-left: 0;
  padding-right: 0;
  margin: 0 5pt;
}
@media (min-width: 40em) {
  .calibre, body {
    width: 38em;
    margin: 0 auto;
  }
}
.calibre1 {
  font-size: 1.25em;
  border-bottom: 0;
  border-top: 0;
  display: block;
  padding-bottom: 0;
  padding-top: 0;
  margin: 0.5em 0;
}
.calibre2, img {
  max-height:100%;
  max-width:100%;
}
.calibre3 {
  font-weight: bold;
}
.calibre4 {
  font-style: italic;
}
p > .calibre3:not(:only-of-type) {
  font-size: 1.5em;
}
.calibre5 {
  display: block;
  font-size: 2em;
  font-weight: bold;
  line-height: 1.05;
  page-break-before: always;
  margin: 0.67em 0;
}
.calibre6 {
  display: block;
  list-style-type: disc;
  margin: 1em 0;
}
.calibre7 {
  display: list-item;
}
`
	_ = os.WriteFile(cssPath, []byte(css), 0o644)
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
	sb.WriteString(
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n",
	)
	// Centered large text
	sb.WriteString(
		"Style: Default,Arial,80,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2,0,5,10,10,10,1\n",
	)
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
		fmt.Fprintf(&sb, "Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startStr, endStr, word)
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
func GenerateTTS(ctx context.Context, text, outputPath string, wpm int) error {
	// Check for espeak-ng
	espeakBin := "espeak-ng"
	if _, err := exec.LookPath(espeakBin); err != nil {
		return errors.New("espeak-ng not found")
	}

	// Boost espeak speed slightly as it tends to drift slower than the calculated word timing
	espeakWpm := int(float64(wpm) * 1.1)
	cmd := exec.CommandContext(ctx, espeakBin, "-w", outputPath, "-s", strconv.Itoa(espeakWpm))
	cmd.Stdin = strings.NewReader(text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("espeak-ng failed: %w: %s", err, string(output))
	}
	return nil
}

// GetHTMLFiles returns a list of HTML files in the directory sorted by filename
func GetHTMLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

					relPath, err := filepath.Rel(dir, path)
					if err != nil {
						return err
					}
					files = append(files, relPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// FindMainContentFile finds the main HTML content file in a calibre output directory
// Skips cover/metadata pages and finds the actual book content
func FindMainContentFile(oebDir string) string {
	// First, try to parse content.opf to find the actual content files
	if contentFile := findMainContentFromOPF(oebDir); contentFile != "" {
		return contentFile
	}

	// Fallback: Find HTML files, preferring those that aren't cover/metadata
	if contentFile := findMainContentFromWalking(oebDir); contentFile != "" {
		return contentFile
	}

	// Last resort: return index.html
	indexHTML := filepath.Join(oebDir, "index.html")
	if _, err := os.Stat(indexHTML); err == nil {
		return indexHTML
	}

	return ""
}

func findMainContentFromOPF(oebDir string) string {
	opfPath := filepath.Join(oebDir, "content.opf")
	content, err := os.ReadFile(opfPath)
	if err != nil {
		return ""
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// 1. Find the first itemref that isn't a cover/title/nav
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lowerLine := strings.ToLower(line)
		if !strings.Contains(lowerLine, "<itemref") ||
			strings.Contains(lowerLine, "cover") ||
			strings.Contains(lowerLine, "title") ||
			strings.Contains(lowerLine, "nav") {

			continue
		}

		// Extract idref value
		idref := extractAttr(line, "idref")
		if idref == "" {
			continue
		}

		// 2. Find corresponding item with this id
		for _, itemLine := range lines {
			if strings.Contains(itemLine, `id="`+idref+`"`) {
				href := extractAttr(itemLine, "href")
				if href != "" {
					contentFile := filepath.Join(oebDir, href)
					if _, err := os.Stat(contentFile); err == nil {
						return contentFile
					}
				}
			}
		}
	}
	return ""
}

func extractAttr(line, attr string) string {
	search := attr + `="`
	match := strings.Index(line, search)
	if match < 0 {
		return ""
	}
	start := match + len(search)
	end := strings.Index(line[start:], `"`)
	if end < 0 {
		return ""
	}
	return line[start : start+end]
}

func findMainContentFromWalking(oebDir string) string {
	var firstContentHTML string
	_ = filepath.Walk(oebDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".xhtml" && ext != ".htm" {
			return nil
		}

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
		if strings.Contains(base, "chapter") || strings.Contains(base, "content") ||
			strings.Contains(base, "ch0") ||
			strings.Contains(base, "split_") {

			firstContentHTML = path
			return filepath.SkipAll
		}
		return nil
	})
	return firstContentHTML
}
