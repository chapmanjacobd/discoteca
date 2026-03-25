package metadata

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
)

// ffprobeBackend implements ProbeBackend using the ffprobe command
type ffprobeBackend struct{}

func newFFProbeBackend() *ffprobeBackend {
	return &ffprobeBackend{}
}

func (b *ffprobeBackend) Probe(ctx context.Context, path string) (*ProbeData, error) {
	// First attempt with optimized settings
	cmd := b.createFFProbeCommand(ctx, path,
		"-analyze_duration", "100000", // 0.1s
		"-probesize", "500000", // 500KB
	)

	output, err := cmd.Output()
	if err != nil {
		// Fallback without optimizations for corrupted or unusual files
		cmdFallback := b.createFFProbeCommand(ctx, path)
		output, _ = cmdFallback.Output()
	}

	if len(output) == 0 {
		return &ProbeData{}, nil
	}

	var ffprobeOut FFProbeOutput
	if err := json.Unmarshal(output, &ffprobeOut); err != nil {
		return &ProbeData{}, nil
	}

	return b.convertFFProbeOutput(&ffprobeOut), nil
}

func (b *ffprobeBackend) createFFProbeCommand(ctx context.Context, path string, extraArgs ...string) *exec.Cmd {
	args := []string{
		"-v", "error",
		"-hide_banner",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		"-of", "json",
		"-rw_timeout", "100000000", // 1m40s - timeout for network/remote files
	}

	args = append(args, extraArgs...)
	args = append(args, path)

	return exec.CommandContext(ctx, "ffprobe", args...)
}

func (b *ffprobeBackend) convertFFProbeOutput(data *FFProbeOutput) *ProbeData {
	result := &ProbeData{
		Tags:     make(map[string]string),
		Streams:  make([]StreamInfo, 0, len(data.Streams)),
		Chapters: make([]ChapterInfo, 0, len(data.Chapters)),
	}

	// Format info
	if data.Format.FormatName != "" {
		result.FormatName = data.Format.FormatName
	}

	if d, err := strconv.ParseFloat(data.Format.Duration, 64); err == nil {
		result.Duration = d
	}

	if data.Format.Tags != nil {
		result.Tags = data.Format.Tags
	}

	// Streams
	for _, s := range data.Streams {
		// Skip attached pictures
		if s.Disposition["attached_pic"] == 1 || s.CodecName == "mjpeg" || s.CodecName == "png" {
			continue
		}

		result.Streams = append(result.Streams, StreamInfo{
			CodecType:    s.CodecType,
			CodecName:    s.CodecName,
			Profile:      s.Profile,
			PixFmt:       s.PixFmt,
			Width:        s.Width,
			Height:       s.Height,
			AvgFrameRate: s.AvgFrameRate,
			SampleRate:   s.SampleRate,
			Channels:     s.Channels,
			Duration:     s.Duration,
			Tags:         s.Tags,
			Disposition:  s.Disposition,
		})
	}

	// Chapters
	for _, ch := range data.Chapters {
		title := ch.Tags["title"]
		if title == "" {
			continue
		}
		startTime, _ := strconv.ParseFloat(ch.StartTime, 64)
		endTime, _ := strconv.ParseFloat(ch.EndTime, 64)

		result.Chapters = append(result.Chapters, ChapterInfo{
			ID:        ch.ID,
			StartTime: startTime,
			EndTime:   endTime,
			Title:     title,
		})
	}

	return result
}
