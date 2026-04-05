package utils

import (
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

type TranscodeStrategy struct {
	NeedsTranscode bool
	VideoCopy      bool
	AudioCopy      bool
	TargetMime     string
}

func GenerateHLSPlaylist(path string, duration float64, segmentDuration int) string {
	segments := int(math.Ceil(duration / float64(segmentDuration)))

	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n")
	fmt.Fprintf(&sb, "#EXT-X-TARGETDURATION:%d\n", segmentDuration)
	sb.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	sb.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")

	for i := range segments {
		segDuration := float64(segmentDuration)
		if i == segments-1 {
			rem := math.Mod(duration, float64(segmentDuration))
			if rem > 0 {
				segDuration = rem
			}
		}
		fmt.Fprintf(&sb, "#EXTINF:%f,\n", segDuration)
		fmt.Fprintf(&sb, "/api/hls/segment?path=%s&index=%d\n", url.QueryEscape(path), i)
	}

	sb.WriteString("#EXT-X-ENDLIST\n")
	return sb.String()
}

func GetHLSSegmentArgs(path string, startTime float64, segmentDuration int, strategy TranscodeStrategy) []string {
	args := []string{
		"-ss", fmt.Sprintf("%f", startTime),
		"-i", path,
		"-t", strconv.Itoa(segmentDuration),
	}

	if strategy.VideoCopy {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args,
			"-vf", "scale=-2:720", // Downscale to 720p for performance/bandwidth
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-pix_fmt", "yuv420p",
		)
	}

	// For HLS (MPEG-TS), AAC is the safest and most compatible choice.
	args = append(args,
		"-c:a", "aac",
		"-b:a", "128k",
		"-ac", "2",
		"-f", "mpegts",
		"-output_ts_offset", fmt.Sprintf("%f", startTime), // Align timestamps
		"pipe:1",
	)
	return args
}

func isSupportedVideoCodec(codec string) bool {
	codec = strings.ToLower(codec)
	return strings.Contains(codec, "h264") || strings.Contains(codec, "avc1") || strings.Contains(codec, "vp8") ||
		strings.Contains(codec, "vp9") ||
		strings.Contains(codec, "av1")
}

func isSupportedAudioCodec(codec string) bool {
	if codec == "" {
		return false
	}
	codec = strings.ToLower(codec)
	// If it contains any incompatible codec, return false
	incompatible := []string{"eac3", "ac3", "dts", "truehd", "mlp"}
	for _, inc := range incompatible {
		if strings.Contains(codec, inc) {
			return false
		}
	}

	// It must contain at least one supported codec
	supported := []string{"aac", "mp3", "opus", "vorbis", "flac", "pcm", "wav"}
	for _, sup := range supported {
		if strings.Contains(codec, sup) {
			return true
		}
	}
	return false
}

func getVideoStrategy(vCodecs, aCodecs, container string) TranscodeStrategy {
	vNeeds := !isSupportedVideoCodec(vCodecs)
	aNeeds := !isSupportedAudioCodec(aCodecs)

	// Prefer WebM for VP9/VP8/AV1/Opus/Vorbis
	preferWebm := strings.Contains(strings.ToLower(vCodecs), "vp9") ||
		strings.Contains(strings.ToLower(vCodecs), "vp8") ||
		strings.Contains(strings.ToLower(vCodecs), "av1") ||
		strings.Contains(strings.ToLower(aCodecs), "opus") ||
		strings.Contains(strings.ToLower(aCodecs), "vorbis")

	targetMime := "video/mp4"
	if preferWebm {
		targetMime = "video/webm"
	}

	// Check if container already matches the target mime type using ffprobe format_name
	containerMatches := false
	switch targetMime {
	case "video/mp4":
		// Most browsers support H264/AAC in MKV, MOV, MP4, M4V
		if container == "mp4" || container == "m4v" || container == "mov" || container == "mkv" ||
			container == "matroska" {

			containerMatches = true
		}
	case "video/webm":
		if container == "webm" || container == "mkv" || container == "matroska" {
			containerMatches = true
		}
	}

	if vNeeds || aNeeds || !containerMatches {
		return TranscodeStrategy{
			NeedsTranscode: true,
			VideoCopy:      !vNeeds,
			AudioCopy:      !aNeeds,
			TargetMime:     targetMime,
		}
	}
	return TranscodeStrategy{NeedsTranscode: false}
}

func getAudioStrategy(aCodecs, container string) TranscodeStrategy {
	// Audio container validation using ffprobe format_name
	if !isSupportedAudioCodec(aCodecs) ||
		(container != "mp3" && container != "mp4" && container != "m4a" && container != "ogg" && container != "flac" && container != "wav" && container != "opus" && container != "webm") {

		return TranscodeStrategy{
			NeedsTranscode: true,
			AudioCopy:      isSupportedAudioCodec(aCodecs),
			TargetMime:     "audio/webm",
		}
	}
	return TranscodeStrategy{NeedsTranscode: false}
}

func GetTranscodeStrategy(m models.Media) TranscodeStrategy {
	ext := strings.ToLower(filepath.Ext(m.Path))

	// If it's a known non-media format, don't even try
	if ext == ".sqlite" || ext == ".db" || ext == ".txt" {
		return TranscodeStrategy{NeedsTranscode: false}
	}

	vCodecs := ""
	if m.VideoCodecs != nil {
		vCodecs = *m.VideoCodecs
	}
	aCodecs := ""
	if m.AudioCodecs != nil {
		aCodecs = *m.AudioCodecs
	}

	mediaType := ""
	if m.MediaType != nil && *m.MediaType != "" {
		mediaType = *m.MediaType
	}

	// Use container format from ffprobe if available, otherwise fall back to extension
	var container string
	if m.ContainerFormat != nil && *m.ContainerFormat != "" {
		container = *m.ContainerFormat
	} else {
		container = strings.TrimPrefix(strings.ToLower(filepath.Ext(m.Path)), ".")
	}

	if mediaType == "image" {
		return TranscodeStrategy{NeedsTranscode: false}
	}

	switch mediaType {
	case "video":
		return getVideoStrategy(vCodecs, aCodecs, container)
	case "audio":
		return getAudioStrategy(aCodecs, container)
	default:
		return TranscodeStrategy{NeedsTranscode: false}
	}
}
