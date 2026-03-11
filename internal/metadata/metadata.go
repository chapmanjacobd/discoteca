package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
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
	Media    db.UpsertMediaParams
	Captions []db.InsertCaptionParams
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
	Filename string            `json:"filename"`
	Duration string            `json:"duration"`
	Size     string            `json:"size"`
	BitRate  string            `json:"bit_rate"`
	Tags     map[string]string `json:"tags"`
}

func Extract(ctx context.Context, path string, scanSubtitles bool) (*MediaMetadata, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Detect mimetype first
	mimeStr := utils.DetectMimeType(path)

	// Advanced Type Detection
	mediaType := ""
	if strings.HasPrefix(mimeStr, "image/") {
		mediaType = "image"
	} else if strings.HasPrefix(mimeStr, "text/") || mimeStr == "application/pdf" || mimeStr == "application/epub+zip" || mimeStr == "application/x-zim" {
		mediaType = "text"
	} else if mimeStr != "" {
		// Fallback to coarse mimetype category
		parts := strings.Split(mimeStr, "/")
		mediaType = parts[0]
	}

	params := db.UpsertMediaParams{
		Path:           path,
		FtsPath:        utils.ToNullString(utils.PathToSentenceFull(path)),
		Size:           utils.ToNullInt64(stat.Size()),
		TimeCreated:    utils.ToNullInt64(stat.ModTime().Unix()),
		TimeModified:   utils.ToNullInt64(stat.ModTime().Unix()),
		Type:           utils.ToNullString(mediaType),
		TimeDownloaded: utils.ToNullInt64(time.Now().Unix()),
	}

	// Fallback title to filename (without extension)
	params.Title = utils.ToNullString(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))

	result := &MediaMetadata{
		Media: params,
	}

	if mediaType == "text" && utils.TextExtensionMap[strings.ToLower(filepath.Ext(path))] {
		if params.Duration.Int64 == 0 {
			// Basic duration estimate for text
			d := int64(float64(stat.Size())/4.2/220*60) + 10
			params.Duration = utils.ToNullInt64(d)
		}
		result.Media = params
		return result, nil
	}

	var duration int64
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-hide_banner",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		"-of", "json",
		"-analyze_duration", "100000", // 0.1s
		"-probesize", "500000", // 500KB
		path,
	)

	var vCodecs, aCodecs, sCodecs []string
	var vCount, aCount, sCount int64

	output, err := cmd.Output()
	if err != nil {
		// Fallback without optimizations for corrupted or unusual files
		cmdFallback := exec.CommandContext(ctx, "ffprobe",
			"-v", "error",
			"-hide_banner",
			"-show_format",
			"-show_streams",
			"-show_chapters",
			"-of", "json",
			path,
		)
		output, _ = cmdFallback.Output()
	}

	if len(output) > 0 {
		var data FFProbeOutput
		if err := json.Unmarshal(output, &data); err == nil {
			// Format info
			if d, err := strconv.ParseFloat(data.Format.Duration, 64); err == nil {
				duration = int64(d)
				// Validate duration is reasonable (max 31 days for sanity)
				if duration > 0 && duration < 2678400 {
					params.Duration = utils.ToNullInt64(duration)
				}
			}

			if data.Format.Tags != nil {
				tags := data.Format.Tags
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
					tags["originalyear"],
					tags["TDOR"],
					tags["TORY"],
					tags["date"],
					tags["TDRC"],
					tags["TDRL"],
					tags["year"],
				)

				if bestDate != nil {
					extraInfo = append(extraInfo, utils.ToDecade(bestDate.Year()))
					if ts := bestDate.Unix(); ts < params.TimeCreated.Int64 {
						params.TimeCreated = utils.ToNullInt64(ts)
					}
				}

				if m := tags["mood"]; m != "" {
					extraInfo = append(extraInfo, "Mood: "+m)
				}
				if b := tags["bpm"]; b != "" {
					extraInfo = append(extraInfo, "BPM: "+b)
				}
				if k := tags["key"]; k != "" {
					extraInfo = append(extraInfo, "Key: "+k)
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

			// Streams info
			for _, s := range data.Streams {
				switch s.CodecType {
				case "video":
					if s.Disposition["attached_pic"] == 1 || s.CodecName == "mjpeg" || s.CodecName == "png" {
						continue
					}
					vCount++
					codecInfo := s.CodecName
					if s.Profile != "" && s.Profile != "unknown" {
						codecInfo += " (" + s.Profile + ")"
					}
					if s.PixFmt != "" {
						codecInfo += " [" + s.PixFmt + "]"
					}
					vCodecs = append(vCodecs, codecInfo)

					if params.Width.Int64 == 0 {
						params.Width = utils.ToNullInt64(int64(s.Width))
						params.Height = utils.ToNullInt64(int64(s.Height))
						params.Fps = utils.ToNullFloat64(parseFPS(s.AvgFrameRate))
					}
				case "audio":
					aCount++
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
					aCodecs = append(aCodecs, codecInfo)
				case "subtitle":
					sCount++
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
					sCodecs = append(sCodecs, label)
				}
			}

			// Chapters
			for _, ch := range data.Chapters {
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
		} else {
			slog.Debug("ffprobe returned invalid JSON", "path", path, "output", string(output))
		}
	} else {
		// If ffprobe fails, it might be a corrupted file or non-media file
		// We already have some basic info from os.Stat and mimetype
		slog.Debug("ffprobe failed to extract metadata (empty output)", "path", path)
	}

	params.VideoCodecs = utils.ToNullString(utils.Combine(vCodecs))
	params.AudioCodecs = utils.ToNullString(utils.Combine(aCodecs))

	// External Subtitles
	if scanSubtitles {
		externalSubs := utils.GetExternalSubtitles(path)
		for _, sub := range externalSubs {
			sCount++
			// Use ExtractSubtitleInfo to get a nice display name with language
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
	}

	params.SubtitleCodecs = utils.ToNullString(utils.Combine(sCodecs))
	params.VideoCount = utils.ToNullInt64(vCount)
	params.AudioCount = utils.ToNullInt64(aCount)
	params.SubtitleCount = utils.ToNullInt64(sCount)

	// Refine Type Detection
	if vCount > 0 && mediaType != "image" {
		mediaType = "video"
		if vCount == 1 && aCount == 0 && duration == 0 {
			mediaType = "image"
		}
	} else if aCount > 0 && mediaType != "image" {
		mediaType = "audio"
		lowerPath := strings.ToLower(path)
		if duration > 3600 || strings.Contains(lowerPath, "audiobook") {
			mediaType = "audiobook"
		}
	}
	params.Type = utils.ToNullString(mediaType)

	result.Media = params
	return result, nil
}

func parseFPS(s string) float64 {
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

func parseSubtitleFile(subPath, mediaPath string) ([]db.InsertCaptionParams, error) {
	data, err := os.ReadFile(subPath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	var captions []db.InsertCaptionParams
	timeRegex := regexp.MustCompile(`(\d{2}:)?\d{2}:\d{2}[.,]\d{3}`)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if timeRegex.MatchString(line) && strings.Contains(line, "-->") {
			matches := timeRegex.FindAllString(line, -1)
			if len(matches) > 0 {
				startTime := utils.FromTimestampSeconds(strings.ReplaceAll(matches[0], ",", "."))

				// Skip captions that start before 10 seconds (typically intro sequences or malformed early captions)
				if startTime < 10.0 {
					continue
				}

				// Text can span multiple lines until empty line
				var textLines []string
				for j := i + 1; j < len(lines); j++ {
					textLine := strings.TrimSpace(lines[j])
					if textLine == "" {
						i = j
						break
					}
					textLines = append(textLines, textLine)
					i = j
				}

				if len(textLines) > 0 {
					text := cleanCaptionText(strings.Join(textLines, " "))
					if text != "" {
						captions = append(captions, db.InsertCaptionParams{
							MediaPath: mediaPath,
							Time:      sql.NullFloat64{Float64: startTime, Valid: true},
							Text:      sql.NullString{String: text, Valid: true},
						})
					}
				}
			}
		}
	}

	return captions, nil
}

func cleanCaptionText(s string) string {
	// Strip HTML tags like <v ...> or <i>
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
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
