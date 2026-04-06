package metadata

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type FFProbeOutput struct {
	Streams  []Stream  `json:"streams"`
	Format   Format    `json:"format"`
	Chapters []Chapter `json:"chapters"`
}

type Chapter struct {
	ID        int               `json:"id"`
	StartTime string            `json:"start_time"`
	EndTime   string            `json:"end_time"`
	Tags      map[string]string `json:"tags"`
}

type MediaMetadata struct {
	Media           db.UpsertMediaParams
	Captions        []db.InsertCaptionParams
	ContainerFormat *string // From ffprobe format_name, used for transcoding decisions
}

type Stream struct {
	CodecType    string            `json:"codec_type"`
	CodecName    string            `json:"codec_name"`
	Profile      string            `json:"profile"`
	PixFmt       string            `json:"pix_fmt"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	AvgFrameRate string            `json:"avg_frame_rate"`
	RFrameRate   string            `json:"r_frame_rate"`
	SampleRate   string            `json:"sample_rate"`
	Channels     int               `json:"channels"`
	Duration     string            `json:"duration"`
	Tags         map[string]string `json:"tags"`
	Disposition  map[string]int    `json:"disposition"`
}

type Format struct {
	Filename   string            `json:"filename"`
	Duration   string            `json:"duration"`
	Size       string            `json:"size"`
	BitRate    string            `json:"bit_rate"`
	FormatName string            `json:"format_name"`
	Tags       map[string]string `json:"tags"`
}

// ExtractOptions contains options for metadata extraction
type ExtractOptions struct {
	ScanSubtitles     bool
	ExtractText       bool
	OCR               bool
	OCREngine         string
	SpeechRecognition bool
	SpeechRecEngine   string
	ProbeImages       bool
}

func Extract(ctx context.Context, path string, opts ExtractOptions) (*MediaMetadata, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	mediaType := detectMediaType(ext)

	params := buildBasicParams(path, stat, mediaType)
	result := &MediaMetadata{Media: params}

	if mediaType == "text" {
		handleTextFile(ctx, path, ext, opts, stat, &params, result)
		return result, nil
	}

	if mediaType == "image" && opts.OCR {
		extractImageOCRCaptions(ctx, path, opts.OCREngine, result)
	}

	if opts.SpeechRecognition && (mediaType == "audio" || mediaType == "video") {
		extractSpeechCaptions(ctx, path, opts.SpeechRecEngine, result)
	}

	if !shouldProbeMedia(mediaType, opts) {
		result.Media = params
		return result, nil
	}

	return extractMediaMetadata(ctx, path, ext, stat, params, result, opts)
}

func detectMediaType(ext string) string {
	if utils.TextExtensionMap[ext] || utils.ComicExtensionMap[ext] || utils.ArchiveExtensionMap[ext] {
		return "text"
	}
	if utils.ImageExtensionMap[ext] {
		return "image"
	}
	if utils.AudioExtensionMap[ext] {
		return "audio"
	}
	if utils.VideoExtensionMap[ext] {
		return "video"
	}
	return ""
}

func buildBasicParams(path string, stat os.FileInfo, mediaType string) db.UpsertMediaParams {
	return db.UpsertMediaParams{
		Path:           path,
		PathTokenized:  utils.ToNullString(utils.PathToTokenized(path)),
		Size:           utils.ToNullInt64(stat.Size()),
		TimeCreated:    utils.ToNullInt64(stat.ModTime().Unix()),
		TimeModified:   utils.ToNullInt64(stat.ModTime().Unix()),
		MediaType:      utils.ToNullString(mediaType),
		TimeDownloaded: utils.ToNullInt64(time.Now().Unix()),
	}
}

func handleTextFile(
	ctx context.Context,
	path, ext string,
	opts ExtractOptions,
	stat os.FileInfo,
	params *db.UpsertMediaParams,
	result *MediaMetadata,
) {
	if !opts.ScanSubtitles && !opts.ExtractText {
		return
	}

	if params.Duration.Int64 == 0 {
		setTextDuration(ctx, path, stat, params)
	}
	result.Media = *params

	if utils.ComicExtensionMap[ext] && opts.OCR {
		extractComicText(ctx, path, opts.OCREngine, result)
	} else if opts.ExtractText && !utils.ComicExtensionMap[ext] {
		extractDocumentTextToCaptions(ctx, path, result)
	}
}

func setTextDuration(ctx context.Context, path string, stat os.FileInfo, params *db.UpsertMediaParams) {
	wordCount, err := utils.QuickWordCount(ctx, path, stat.Size())
	if err != nil || wordCount <= 0 {
		d := int64(float64(stat.Size())/4.2/220*60) + 10
		params.Duration = utils.ToNullInt64(d)
	} else {
		params.Duration = utils.ToNullInt64(utils.EstimateReadingDuration(wordCount))
	}
}

func extractComicText(ctx context.Context, path, engine string, result *MediaMetadata) {
	captions, err := ExtractImageTextFromComicArchive(ctx, path, engine)
	if err != nil {
		models.Log.Warn("Comic archive OCR extraction failed", "path", path, "error", err)
	} else {
		result.Captions = captions
	}
}

func extractDocumentTextToCaptions(ctx context.Context, path string, result *MediaMetadata) {
	captions, err := extractDocumentText(ctx, path)
	if err != nil {
		models.Log.Warn("Document text extraction failed", "path", path, "error", err)
	} else {
		result.Captions = captions
	}
}

func extractImageOCRCaptions(ctx context.Context, path, engine string, result *MediaMetadata) {
	captions, err := extractImageText(ctx, path, engine)
	if err != nil {
		models.Log.Warn("Image OCR extraction failed", "path", path, "error", err)
	} else {
		result.Captions = captions
	}
}

func extractSpeechCaptions(ctx context.Context, path, engine string, result *MediaMetadata) {
	captions, err := ExtractSpeechToText(ctx, path, engine)
	if err != nil {
		models.Log.Warn("Speech recognition failed", "path", path, "error", err)
	} else {
		result.Captions = append(result.Captions, captions...)
	}
}

func shouldProbeMedia(mediaType string, opts ExtractOptions) bool {
	if mediaType != "video" && mediaType != "audio" && mediaType != "image" {
		return false
	}
	if mediaType == "image" && !opts.ProbeImages {
		return false
	}
	return true
}

type streamCounts struct {
	vCount, aCount, sCount    int64
	vCodecs, aCodecs, sCodecs []string
}

func extractMediaMetadata(
	ctx context.Context,
	path, ext string,
	stat os.FileInfo,
	params db.UpsertMediaParams,
	result *MediaMetadata,
	opts ExtractOptions,
) (*MediaMetadata, error) {
	duration := int64(0)
	cmd := utils.FFProbe(ctx, path, "-analyze_duration", "100000", "-probesize", "500000")
	output, err := cmd.Output()
	if err != nil {
		cmdFallback := utils.FFProbe(ctx, path)
		output, _ = cmdFallback.Output()
	}

	sc := streamCounts{}
	if len(output) > 0 {
		var data FFProbeOutput
		if err := json.Unmarshal(output, &data); err == nil {
			processFormatInfo(data, path, stat, ext, &params, result, &duration)
			sc = processStreams(data.Streams, path, &params, result)
			processChapters(data.Chapters, path, result)
		} else {
			models.Log.Debug("ffprobe returned invalid JSON", "path", path, "output", string(output))
		}
	} else {
		models.Log.Debug("ffprobe failed to extract metadata (empty output)", "path", path)
	}

	if opts.ScanSubtitles {
		sc.sCount, sc.sCodecs = processExternalSubtitles(ctx, path, opts.ScanSubtitles, sc.sCount, sc.sCodecs, result)
	}

	params.VideoCodecs = utils.ToNullString(utils.Combine(sc.vCodecs))
	params.AudioCodecs = utils.ToNullString(utils.Combine(sc.aCodecs))
	params.SubtitleCodecs = utils.ToNullString(utils.Combine(sc.sCodecs))
	params.VideoCount = utils.ToNullInt64(sc.vCount)
	params.AudioCount = utils.ToNullInt64(sc.aCount)
	params.SubtitleCount = utils.ToNullInt64(sc.sCount)

	originalMediaType := params.MediaType.String
	params.MediaType = utils.ToNullString(
		refineMediaTypeFromStreams(originalMediaType, sc.vCount, sc.aCount, duration, path),
	)
	result.Media = params
	return result, nil
}

func refineMediaTypeFromStreams(mediaType string, vCount, aCount, duration int64, path string) string {
	if vCount > 0 && mediaType != "image" {
		if vCount == 1 && aCount == 0 && duration == 0 {
			return "image"
		}
		return "video"
	}
	if aCount > 0 && mediaType != "image" {
		lowerPath := strings.ToLower(path)
		if duration > 3600 || strings.Contains(lowerPath, "audiobook") {
			return "audiobook"
		}
		return "audio"
	}
	return mediaType
}

func processFormatInfo(
	data FFProbeOutput,
	path string,
	stat os.FileInfo,
	ext string,
	params *db.UpsertMediaParams,
	result *MediaMetadata,
	duration *int64,
) {
	if data.Format.FormatName != "" {
		result.ContainerFormat = &data.Format.FormatName
	}

	if d, err := strconv.ParseFloat(data.Format.Duration, 64); err == nil {
		*duration = int64(d)
		if *duration > 0 && *duration < 2678400 {
			if estimated, shouldOverride := utils.ShouldOverrideDuration(
				float64(*duration),
				stat.Size(),
				ext,
			); shouldOverride {
				models.Log.Debug("Replacing suspiciously low duration with format-specific estimate",
					"path", path, "reported", *duration, "estimated", int64(estimated),
					"bitrate", utils.GetEstimatedBitrate(ext))
				*duration = int64(estimated)
			}
			params.Duration = utils.ToNullInt64(*duration)
		}
	}

	if data.Format.Tags != nil {
		extractFormatTags(data.Format.Tags, params)
	}
}

func extractFormatTags(tags map[string]string, params *db.UpsertMediaParams) {
	if t := tags["title"]; t != "" {
		params.Title = utils.ToNullString(t)
	}
	if a := tags["artist"]; a != "" {
		params.Artist = utils.ToNullString(a)
	}
	if al := tags["album"]; al != "" {
		params.Album = utils.ToNullString(al)
	}
	if g := tags["genre"]; g != "" {
		params.Genre = utils.ToNullString(g)
	}
	if l := tags["language"]; l != "" {
		params.Language = utils.ToNullString(l)
	}

	var extraInfo []string
	bestDate := utils.SpecificDate(
		tags["originalyear"], tags["TDOR"], tags["TORY"],
		tags["date"], tags["TDRC"], tags["TDRL"], tags["year"],
	)
	if bestDate != nil {
		extraInfo = append(extraInfo, utils.ToDecade(bestDate.Year()))
		if ts := bestDate.Unix(); ts < params.TimeCreated.Int64 {
			params.TimeCreated = utils.ToNullInt64(ts)
		}
	}

	for key, prefix := range map[string]string{"mood": "Mood: ", "bpm": "BPM: ", "key": "Key: "} {
		if v := tags[key]; v != "" {
			extraInfo = append(extraInfo, prefix+v)
		}
	}

	desc := tags["description"]
	if desc == "" {
		desc = tags["comment"]
	}
	if len(extraInfo) > 0 {
		if desc != "" {
			desc += "\n\n"
		}
		desc += strings.Join(extraInfo, " | ")
	}
	params.Description = utils.ToNullString(desc)
	params.Categories = utils.ToNullString(tags["categories"])
}

func processStreams(streams []Stream, path string, params *db.UpsertMediaParams, result *MediaMetadata) streamCounts {
	sc := streamCounts{}
	for _, s := range streams {
		switch s.CodecType {
		case "video":
			processVideoStream(s, &sc, params)
		case "audio":
			processAudioStream(s, &sc)
		case "subtitle":
			processSubtitleStream(s, &sc)
		}
	}
	return sc
}

func processVideoStream(s Stream, sc *streamCounts, params *db.UpsertMediaParams) {
	if s.Disposition["attached_pic"] == 1 || s.CodecName == "mjpeg" || s.CodecName == "png" {
		return
	}
	sc.vCount++
	codecInfo := s.CodecName
	if s.Profile != "" && s.Profile != "unknown" {
		codecInfo += " (" + s.Profile + ")"
	}
	if s.PixFmt != "" {
		codecInfo += " [" + s.PixFmt + "]"
	}
	sc.vCodecs = append(sc.vCodecs, codecInfo)

	if params.Width.Int64 == 0 {
		params.Width = utils.ToNullInt64(int64(s.Width))
		params.Height = utils.ToNullInt64(int64(s.Height))
		params.Fps = utils.ToNullFloat64(ParseFPS(s.AvgFrameRate))
	}
}

func processAudioStream(s Stream, sc *streamCounts) {
	sc.aCount++
	codecInfo := s.CodecName
	if s.Channels > 0 {
		codecInfo += " " + strconv.Itoa(s.Channels) + "ch"
	}
	if s.SampleRate != "" {
		codecInfo += " " + s.SampleRate + "Hz"
	}
	var details []string
	if lang := s.Tags["language"]; lang != "" {
		details = append(details, lang)
	}
	if title := s.Tags["title"]; title != "" {
		details = append(details, title)
	}
	if len(details) > 0 {
		codecInfo += " (" + strings.Join(details, ", ") + ")"
	}
	sc.aCodecs = append(sc.aCodecs, codecInfo)
}

func processSubtitleStream(s Stream, sc *streamCounts) {
	sc.sCount++
	var label string
	if lang := s.Tags["language"]; lang != "" {
		label = lang
	}
	if title := s.Tags["title"]; title != "" {
		if label != "" {
			label += " - " + title
		} else {
			label = title
		}
	}
	if label == "" {
		label = s.CodecName
	}
	sc.sCodecs = append(sc.sCodecs, label)
}

func processChapters(chapters []Chapter, path string, result *MediaMetadata) {
	for _, ch := range chapters {
		title := ch.Tags["title"]
		if title == "" {
			continue
		}
		startTime, _ := strconv.ParseFloat(ch.StartTime, 64)
		result.Captions = append(result.Captions, db.InsertCaptionParams{
			MediaPath: path,
			Time:      sql.NullFloat64{Float64: startTime, Valid: true},
			Text:      sql.NullString{String: title, Valid: true},
		})
	}
}

func processExternalSubtitles(
	ctx context.Context,
	path string,
	scanSubtitles bool,
	sCount int64,
	sCodecs []string,
	result *MediaMetadata,
) (int64, []string) {
	if !scanSubtitles {
		return sCount, sCodecs
	}
	externalSubs := utils.GetExternalSubtitles(path)
	for _, sub := range externalSubs {
		sCount++
		displayName, _, _ := utils.ExtractSubtitleInfo(sub)
		if displayName != "" {
			sCodecs = append(sCodecs, displayName)
		} else {
			ext := strings.ToLower(filepath.Ext(sub))
			sCodecs = append(sCodecs, strings.TrimPrefix(ext, "."))
		}

		ext := strings.ToLower(filepath.Ext(sub))
		if ext == ".vtt" || ext == ".srt" {
			caps, err := parseSubtitleFile(sub, path)
			if err == nil {
				result.Captions = append(result.Captions, caps...)
			}
		}
	}
	return sCount, sCodecs
}

func ParseFPS(s string) float64 {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0
	}
	num, _ := strconv.ParseFloat(parts[0], 64)
	den, _ := strconv.ParseFloat(parts[1], 64)
	if den == 0 {
		return 0
	}
	return num / den
}

var timeRegex = regexp.MustCompile(`(\d{2}:)?\d{2}:\d{2}[.,]\d{3}`)

func parseSubtitleFile(subPath, mediaPath string) ([]db.InsertCaptionParams, error) {
	f, err := os.Open(subPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var captions []db.InsertCaptionParams
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if timeRegex.MatchString(line) && strings.Contains(line, "-->") {
			captions = captionForLine(line, mediaPath, scanner, captions)
		}
	}

	return captions, scanner.Err()
}

func captionForLine(
	line, mediaPath string,
	scanner *bufio.Scanner,
	captions []db.InsertCaptionParams,
) []db.InsertCaptionParams {
	matches := timeRegex.FindAllString(line, -1)
	if len(matches) == 0 {
		return captions
	}

	startTime := utils.FromTimestampSeconds(strings.ReplaceAll(matches[0], ",", "."))
	if startTime < 10.0 {
		return captions
	}

	textLines := scanCaptionText(scanner)
	if len(textLines) == 0 {
		return captions
	}

	text := cleanCaptionText(strings.Join(textLines, " "))
	if text == "" {
		return captions
	}

	return append(captions, db.InsertCaptionParams{
		MediaPath: mediaPath,
		Time:      sql.NullFloat64{Float64: startTime, Valid: true},
		Text:      sql.NullString{String: text, Valid: true},
	})
}

func scanCaptionText(scanner *bufio.Scanner) []string {
	var textLines []string
	for scanner.Scan() {
		textLine := strings.TrimSpace(scanner.Text())
		if textLine == "" {
			break
		}
		textLines = append(textLines, textLine)
	}
	return textLines
}

// extractDocumentText extracts full text from a document and returns it as captions.
// Text is chunked into paragraphs/sections for better search relevance.
// Each chunk is stored as a caption with time=0 (documents don't have timestamps).
func extractDocumentText(ctx context.Context, path string) ([]db.InsertCaptionParams, error) {
	// Use the existing ExtractText utility
	fullText, err := utils.ExtractText(ctx, path)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(fullText) == "" {
		return nil, nil
	}

	// Split text into chunks (paragraphs or fixed-size chunks for very long paragraphs)
	chunks := chunkDocumentText(fullText)

	captions := make([]db.InsertCaptionParams, 0, len(chunks))
	for i, chunk := range chunks {
		text := cleanCaptionText(chunk)
		if text == "" {
			continue
		}
		captions = append(captions, db.InsertCaptionParams{
			MediaPath: path,
			Time:      sql.NullFloat64{Float64: 0, Valid: false}, // No timestamp for documents
			Text:      sql.NullString{String: text, Valid: true},
		})
		_ = i // suppress unused variable warning
	}

	return captions, nil
}

// chunkDocumentText splits document text into searchable chunks.
// It tries to split by paragraphs first, then by sentences for very long paragraphs.
func chunkDocumentText(text string) []string {
	const (
		maxChunkSize = 2000 // Maximum characters per chunk
		minChunkSize = 50   // Minimum characters to create a chunk
	)

	var chunks []string

	// Split by paragraphs (double newlines or single newlines)
	paragraphs := strings.Split(text, "\n\n")
	if len(paragraphs) == 1 {
		paragraphs = strings.Split(text, "\n")
	}

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) < minChunkSize {
			continue
		}

		// If paragraph is small enough, use as-is
		if len(para) <= maxChunkSize {
			chunks = append(chunks, para)
			continue
		}

		// Split long paragraphs by sentences
		sentences := strings.Split(para, ". ")
		currentChunk := ""
		for _, sent := range sentences {
			sent = strings.TrimSpace(sent)
			if sent == "" {
				continue
			}
			if !strings.HasSuffix(sent, ".") {
				sent += "."
			}

			if len(currentChunk)+len(sent) <= maxChunkSize {
				currentChunk += sent + " "
			} else {
				if len(currentChunk) >= minChunkSize {
					chunks = append(chunks, strings.TrimSpace(currentChunk))
				}
				currentChunk = sent + " "
			}
		}
		if len(currentChunk) >= minChunkSize {
			chunks = append(chunks, strings.TrimSpace(currentChunk))
		}
	}

	// Fallback: if no chunks created, use the whole text as one chunk
	if len(chunks) == 0 && len(strings.TrimSpace(text)) >= minChunkSize {
		chunks = append(chunks, strings.TrimSpace(text))
	}

	return chunks
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func cleanCaptionText(s string) string {
	// Strip HTML tags like <v ...> or <i>
	s = htmlTagRe.ReplaceAllString(s, "")
	// Strip SRT-style formatting if any
	s = strings.TrimSpace(s)

	// Filter out malformed text that looks like unclosed/empty HTML attributes
	// e.g., "untitled chapter 1" from malformed <untitled chapter="" 1="">
	// These typically contain = signs with empty quoted values
	if strings.Contains(s, "=") && strings.Contains(s, `""`) {
		return ""
	}

	// Check if the remaining text is just whitespace or common noise patterns
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Filter out text that's only special characters or very short noise
	if len(s) < 2 {
		return ""
	}

	return s
}

// extractImageText extracts text from images using OCR.
// Returns captions with detected text (time=0 for images).
func extractImageText(ctx context.Context, path, engine string) ([]db.InsertCaptionParams, error) {
	// Convert image to PNG if it's in a format that OCR engines might struggle with
	convertedPath, err := ConvertImageForOCR(ctx, path)
	if err != nil {
		models.Log.Warn("Image conversion failed, using original", "path", path, "error", err)
		convertedPath = path
	}
	if convertedPath != path {
		defer os.Remove(convertedPath) // Clean up temp file
	}

	switch engine {
	case "paddle":
		return extractImageTextPaddleOCR(ctx, convertedPath)
	case "tesseract", "":
		return extractImageTextTesseract(ctx, convertedPath)
	default:
		return extractImageTextTesseract(ctx, convertedPath)
	}
}

// ConvertImageForOCR converts images to PNG format for better OCR compatibility.
// Tesseract and PaddleOCR work best with PNG/TIFF. This function converts
// problematic formats (webp, bmp, gif, tiff) to PNG using ffmpeg or ImageMagick.
// Returns the path to the converted image, or the original path if no conversion needed.
func ConvertImageForOCR(ctx context.Context, path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	// Formats that OCR engines handle well natively
	goodFormats := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".tif":  true,
		".tiff": true,
	}

	if goodFormats[ext] {
		return path, nil
	}

	// Try ffmpeg first (faster, handles most formats)
	ffmpegBin := "ffmpeg"
	if _, err := exec.LookPath(ffmpegBin); err == nil {
		tmpFile, err := os.CreateTemp("", "ocr-convert-*.png")
		if err != nil {
			return "", err
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)
		defer tmpFile.Close()

		args := []string{
			"-hide_banner",
			"-loglevel", "error",
			"-i", path,
			"-c:v", "png",
			tmpPath,
		}

		cmd := exec.CommandContext(ctx, ffmpegBin, args...)
		if err := cmd.Run(); err == nil {
			return tmpPath, nil
		}
	}

	// Try ImageMagick (convert)
	convertBin := "convert"
	if _, err := exec.LookPath(convertBin); err == nil {
		tmpFile, err := os.CreateTemp("", "ocr-convert-*.png")
		if err != nil {
			return "", err
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)
		defer tmpFile.Close()

		args := []string{path, tmpPath}
		cmd := exec.CommandContext(ctx, convertBin, args...)
		if err := cmd.Run(); err == nil {
			return tmpPath, nil
		}
	}

	// No converter available, return original path
	// OCR engine may still be able to handle it
	return path, nil
}

// extractImageTextTesseract extracts text from images using tesseract OCR
func extractImageTextTesseract(ctx context.Context, path string) ([]db.InsertCaptionParams, error) {
	// Check for tesseract
	tesseractBin := "tesseract"
	if _, err := exec.LookPath(tesseractBin); err != nil {
		return nil, errors.New("tesseract not found")
	}

	// Run tesseract with stdout output
	// Using --psm 3 (fully automatic page segmentation) for general images
	cmd := exec.CommandContext(ctx, tesseractBin, path, "stdout", "--psm", "3")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	text := string(output)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	// Split into chunks for better search relevance
	chunks := chunkDocumentText(text)

	captions := make([]db.InsertCaptionParams, 0, len(chunks))
	for _, chunk := range chunks {
		cleaned := cleanCaptionText(chunk)
		if cleaned == "" {
			continue
		}
		captions = append(captions, db.InsertCaptionParams{
			MediaPath: path,
			Time:      sql.NullFloat64{Float64: 0, Valid: false}, // No timestamp for images
			Text:      sql.NullString{String: cleaned, Valid: true},
		})
	}

	return captions, nil
}

// extractImageTextPaddleOCR extracts text from images using PaddleOCR
func extractImageTextPaddleOCR(ctx context.Context, path string) ([]db.InsertCaptionParams, error) {
	// Check for python and paddleocr
	pythonBin := "python3"
	if _, err := exec.LookPath(pythonBin); err != nil {
		pythonBin = "python"
		if _, err := exec.LookPath(pythonBin); err != nil {
			return nil, errors.New("python not found")
		}
	}

	// Run paddleocr with image
	// --type ocr for OCR only, --lang for language (default en)
	cmd := exec.CommandContext(
		ctx,
		pythonBin,
		"-m",
		"paddleocr",
		"-i",
		path,
		"--type",
		"ocr",
		"--lang",
		"en",
		"--show_log",
		"false",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	text := string(output)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	// Split into chunks for better search relevance
	chunks := chunkDocumentText(text)

	captions := make([]db.InsertCaptionParams, 0, len(chunks))
	for _, chunk := range chunks {
		cleaned := cleanCaptionText(chunk)
		if cleaned == "" {
			continue
		}
		captions = append(captions, db.InsertCaptionParams{
			MediaPath: path,
			Time:      sql.NullFloat64{Float64: 0, Valid: false},
			Text:      sql.NullString{String: cleaned, Valid: true},
		})
	}

	return captions, nil
}

// ExtractSpeechToText extracts speech-to-text from audio/video files
func ExtractSpeechToText(ctx context.Context, path, engine string) ([]db.InsertCaptionParams, error) {
	switch engine {
	case "whisper":
		return extractSpeechToTextWhisper(ctx, path)
	case "vosk", "":
		return extractSpeechToTextVosk(ctx, path)
	default:
		return extractSpeechToTextVosk(ctx, path)
	}
}

// extractSpeechToTextVosk extracts speech-to-text using Vosk
func extractSpeechToTextVosk(ctx context.Context, path string) ([]db.InsertCaptionParams, error) {
	// Check for python and vosk
	pythonBin := "python3"
	if _, err := exec.LookPath(pythonBin); err != nil {
		pythonBin = "python"
		if _, err := exec.LookPath(pythonBin); err != nil {
			return nil, errors.New("python not found")
		}
	}

	// Create temp directory for model
	modelDir := os.Getenv("VOSK_MODEL_PATH")
	if modelDir == "" {
		modelDir = filepath.Join(os.Getenv("HOME"), ".cache", "vosk-model")
	}

	// Check if model exists
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"vosk model not found at %s (download from https://alphacephei.com/vosk/models)",
			modelDir,
		)
	}

	// Run vosk transcription
	// Using a simple Python script to transcribe
	voskScript := `
import sys
import json
from vosk import Model, KaldiRecognizer
import wave

model = Model(sys.argv[1])
wf = wave.open(sys.argv[2], "rb")
if wf.getnchannels() != 1 or wf.getsampwidth() != 2 or wf.getcomptype() != "NONE":
    print("Audio file must be mono 16-bit PCM", file=sys.stderr)
    sys.exit(1)

rec = KaldiRecognizer(model, wf.getframerate())
rec.SetWords(True)

captions = []
while True:
    data = wf.readframes(4000)
    if len(data) == 0:
        break
    if rec.AcceptWaveform(data):
        result = json.loads(rec.Result())
        if "text" in result and result["text"]:
            captions.append(result["text"])
    else:
        result = json.loads(rec.PartialResult())
        if "partial" in result and result["partial"]:
            pass  # Ignore partial results

final = json.loads(rec.FinalResult())
if "text" in final and final["text"]:
    captions.append(final["text"])

for caption in captions:
    print(caption)
`

	cmd := exec.CommandContext(ctx, pythonBin, "-c", voskScript, modelDir, path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	text := string(output)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	// Split into chunks (each line is a caption)
	lines := strings.Split(text, "\n")
	captions := make([]db.InsertCaptionParams, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		captions = append(captions, db.InsertCaptionParams{
			MediaPath: path,
			Time:      sql.NullFloat64{Float64: float64(i * 5), Valid: true}, // Approximate timestamp
			Text:      sql.NullString{String: cleanCaptionText(line), Valid: true},
		})
	}

	return captions, nil
}

// extractSpeechToTextWhisper extracts speech-to-text using OpenAI Whisper
func extractSpeechToTextWhisper(ctx context.Context, path string) ([]db.InsertCaptionParams, error) {
	// Check for whisper CLI (pip install openai-whisper)
	whisperBin := "whisper"
	if _, err := exec.LookPath(whisperBin); err != nil {
		return nil, errors.New("whisper not found (pip install openai-whisper)")
	}

	// Create temp directory for output
	tmpDir, err := os.MkdirTemp("", "whisper-output-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Run whisper with txt output
	// --model tiny for speed, use base/small/medium/large for better accuracy
	cmd := exec.CommandContext(
		ctx,
		whisperBin,
		path,
		"--model",
		"tiny",
		"--output_dir",
		tmpDir,
		"--output_format",
		"txt",
		"--language",
		"en",
	)
	if err2 := cmd.Run(); err2 != nil {
		return nil, err2
	}

	// Read output txt file
	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	txtPath := filepath.Join(tmpDir, baseName+".txt")
	text, err := os.ReadFile(txtPath)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(string(text)) == "" {
		return nil, nil
	}

	// Split into chunks for better search relevance
	chunks := chunkDocumentText(string(text))

	captions := make([]db.InsertCaptionParams, 0, len(chunks))
	for _, chunk := range chunks {
		cleaned := cleanCaptionText(chunk)
		if cleaned == "" {
			continue
		}
		captions = append(captions, db.InsertCaptionParams{
			MediaPath: path,
			Time:      sql.NullFloat64{Float64: 0, Valid: false},
			Text:      sql.NullString{String: cleaned, Valid: true},
		})
	}

	return captions, nil
}

// ExtractImageTextFromComicArchive extracts text from images in CBZ/CBR archives using OCR.
// Returns captions with page numbers as timestamps (page 1 = 0s, page 2 = 1s, etc.)
func ExtractImageTextFromComicArchive(ctx context.Context, path, ocrEngine string) ([]db.InsertCaptionParams, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if !utils.ComicExtensionMap[ext] {
		return nil, fmt.Errorf("unsupported archive format: %s", ext)
	}

	if ext == ".cbz" {
		return extractImageTextFromCBZ(ctx, path, ocrEngine)
	}
	if ext == ".cbr" {
		return extractImageTextFromCBR(ctx, path, ocrEngine)
	}

	return nil, fmt.Errorf("unsupported comic format: %s", ext)
}

// imageFile represents an image file within a comic archive
type imageFile struct {
	name string
	idx  int
}

// extractImageTextFromCBZ extracts text from images in CBZ (ZIP-based) archives
func extractImageTextFromCBZ(ctx context.Context, path, ocrEngine string) ([]db.InsertCaptionParams, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Collect image files and sort them for consistent page ordering
	var imageFiles []imageFile

	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".bmp": true, ".tiff": true, ".tif": true,
	}

	for i, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if imageExts[ext] {
			imageFiles = append(imageFiles, imageFile{name: f.Name, idx: i})
		}
	}

	// Sort by filename for consistent page ordering (01.jpg, 02.jpg, etc.)
	sort.Slice(imageFiles, func(i, j int) bool {
		return imageFiles[i].name < imageFiles[j].name
	})

	var allCaptions []db.InsertCaptionParams

	for pageNum, imgFile := range imageFiles {
		captions, err := processComicPage(ctx, ComicPageParams{
			Reader:      r,
			ImageFile:   imgFile,
			OCREngine:   ocrEngine,
			ArchivePath: path,
		})
		if err != nil {
			continue
		}
		for _, cap := range captions {
			cap.Time = sql.NullFloat64{Float64: float64(pageNum), Valid: true}
			allCaptions = append(allCaptions, cap)
		}
	}

	return allCaptions, nil
}

// ComicPageParams contains the context needed to process a single comic page
type ComicPageParams struct {
	Reader      *zip.ReadCloser
	ImageFile   imageFile
	OCREngine   string
	ArchivePath string
}

// processComicPage extracts and OCRs a single comic page image
func processComicPage(
	ctx context.Context,
	params ComicPageParams,
) ([]db.InsertCaptionParams, error) {
	rc, err := params.Reader.File[params.ImageFile.idx].Open()
	if err != nil {
		models.Log.Warn(
			"Failed to open image in archive",
			"archive",
			params.ArchivePath,
			"image",
			params.ImageFile.name,
			"error",
			err,
		)
		return nil, err
	}
	defer rc.Close()

	// Extract image to temp file for OCR processing
	tmpFile, err2 := os.CreateTemp("", "comic-ocr-*.img")
	if err2 != nil {
		models.Log.Warn("Failed to create temp file for OCR", "error", err2)
		return nil, err2
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	defer tmpFile.Close()

	if _, err2 := io.Copy(tmpFile, rc); err2 != nil {
		models.Log.Warn("Failed to extract image for OCR", "error", err2)
		return nil, err2
	}

	// Run OCR on the extracted image
	captions, err := extractImageText(ctx, tmpPath, params.OCREngine)
	if err != nil {
		models.Log.Warn(
			"OCR failed on comic page",
			"archive",
			params.ArchivePath,
			"page",
			params.ImageFile.name,
			"error",
			err,
		)
		return nil, err
	}

	return captions, nil
}

// extractImageTextFromCBR extracts text from images in CBR (RAR-based) archives
func extractImageTextFromCBR(ctx context.Context, path, ocrEngine string) ([]db.InsertCaptionParams, error) {
	// Try unrar first
	unrarBin := "unrar"
	unrarErr := false
	if _, err := exec.LookPath(unrarBin); err != nil {
		unrarErr = true
	}

	// Try unar (The Unarchiver) as alternative
	unarBin := "unar"
	unarErr := false
	if _, err := exec.LookPath(unarBin); err != nil {
		unarErr = true
	}

	// Both not found
	if unrarErr && unarErr {
		return nil, errors.New("no RAR extractor found (install unrar or unar)")
	}

	// Extract to temp directory
	tmpDir, err := os.MkdirTemp("", "cbr-extract-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Extract all files - prefer unrar, fallback to unar
	var cmd *exec.Cmd
	if !unrarErr {
		cmd = exec.CommandContext(ctx, unrarBin, "e", "-y", path, tmpDir)
	} else {
		cmd = exec.CommandContext(ctx, unarBin, "-o", tmpDir, path)
	}
	if err2 := cmd.Run(); err2 != nil {
		return nil, err2
	}

	// Find all image files
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".bmp": true, ".tiff": true, ".tif": true,
	}

	var imageFiles []string
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if imageExts[ext] {
			imageFiles = append(imageFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort for consistent page ordering
	sort.Strings(imageFiles)

	var allCaptions []db.InsertCaptionParams

	for pageNum, imgPath := range imageFiles {
		captions, err := extractImageText(ctx, imgPath, ocrEngine)
		if err != nil {
			models.Log.Warn("OCR failed on comic page", "archive", path, "page", imgPath, "error", err)
			continue
		}

		// Add page number as timestamp
		for _, cap := range captions {
			cap.Time = sql.NullFloat64{Float64: float64(pageNum), Valid: true}
			allCaptions = append(allCaptions, cap)
		}
	}

	return allCaptions, nil
}
