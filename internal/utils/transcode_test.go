package utils

import (
	"strings"
	"testing"

	"github.com/chapmanjacobd/discotheque/internal/models"
)

func TestGenerateHLSPlaylist(t *testing.T) {
	path := "/media/video.mp4"
	duration := 15.0
	segmentDuration := 6

	playlist := GenerateHLSPlaylist(path, duration, segmentDuration)

	if !strings.Contains(playlist, "#EXTM3U") {
		t.Error("Playlist missing #EXTM3U")
	}
	if !strings.Contains(playlist, "#EXT-X-TARGETDURATION:6") {
		t.Error("Playlist missing or incorrect TARGETDURATION")
	}

	// 15s duration / 6s segments = 3 segments (6, 6, 3)
	if !strings.Contains(playlist, "#EXTINF:6.000000,") {
		t.Error("Playlist missing first segment duration")
	}
	if !strings.Contains(playlist, "#EXTINF:3.000000,") {
		t.Error("Playlist missing last segment duration")
	}

	segmentCount := strings.Count(playlist, "/api/hls/segment")
	if segmentCount != 3 {
		t.Errorf("Expected 3 segments, got %d", segmentCount)
	}
}

func TestGetHLSSegmentArgs(t *testing.T) {
	path := "/media/video.mp4"
	startTime := 12.0
	segmentDuration := 6
	
	t.Run("VideoCopy", func(t *testing.T) {
		strategy := TranscodeStrategy{VideoCopy: true}
		args := GetHLSSegmentArgs(path, startTime, segmentDuration, strategy)
		
		argStr := strings.Join(args, " ")
		if !strings.Contains(argStr, "-c:v copy") {
			t.Errorf("Expected video copy, got %v", args)
		}
		if !strings.Contains(argStr, "-ss 12.000000") {
			t.Errorf("Expected start time 12.0, got %v", args)
		}
	})

	t.Run("VideoTranscode", func(t *testing.T) {
		strategy := TranscodeStrategy{VideoCopy: false}
		args := GetHLSSegmentArgs(path, startTime, segmentDuration, strategy)
		
		argStr := strings.Join(args, " ")
		if !strings.Contains(argStr, "-c:v libx264") {
			t.Errorf("Expected libx264, got %v", args)
		}
		if !strings.Contains(argStr, "scale=-2:720") {
			t.Errorf("Expected scaling, got %v", args)
		}
	})
}

func TestGetTranscodeStrategy(t *testing.T) {
	t.Run("MP4_H264_AAC", func(t *testing.T) {
		vCodec := "h264"
		aCodec := "aac"
		mime := "video/mp4"
		m := models.Media{
			Path:        "video.mp4",
			VideoCodecs: &vCodec,
			AudioCodecs: &aCodec,
			Type:        &mime,
		}
		strategy := GetTranscodeStrategy(m)
		if strategy.NeedsTranscode {
			t.Error("Expected no transcode for H264/AAC in MP4")
		}
	})

	t.Run("MKV_H265_AC3", func(t *testing.T) {
		vCodec := "hevc"
		aCodec := "ac3"
		mime := "video/x-matroska"
		m := models.Media{
			Path:        "video.mkv",
			VideoCodecs: &vCodec,
			AudioCodecs: &aCodec,
			Type:        &mime,
		}
		strategy := GetTranscodeStrategy(m)
		if !strategy.NeedsTranscode {
			t.Error("Expected transcode for HEVC/AC3")
		}
		if strategy.VideoCopy {
			t.Error("Expected video transcode")
		}
		if strategy.AudioCopy {
			t.Error("Expected audio transcode")
		}
	})
}
