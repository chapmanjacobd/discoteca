//go:build !astiav

package metadata

import (
	"context"
)

// ProbeData contains the result of probing a media file
type ProbeData struct {
	FormatName string            // Container format (e.g., "matroska", "mp4")
	Duration   float64           // Duration in seconds
	Size       int64             // File size in bytes
	Tags       map[string]string // Format-level tags
	Streams    []StreamInfo      // Stream information
	Chapters   []ChapterInfo     // Chapter information
}

// StreamInfo contains information about a media stream
type StreamInfo struct {
	CodecType    string            // "video", "audio", "subtitle"
	CodecName    string            // Codec name
	Profile      string            // Codec profile
	PixFmt       string            // Pixel format (video only)
	Width        int               // Width (video only)
	Height       int               // Height (video only)
	AvgFrameRate string            // Average frame rate (video only)
	SampleRate   string            // Sample rate (audio only)
	Channels     int               // Channels (audio only)
	Duration     string            // Stream duration
	Tags         map[string]string // Stream tags
	Disposition  map[string]int    // Stream disposition (attached_pic, etc.)
}

// ChapterInfo contains chapter information
type ChapterInfo struct {
	ID        int
	StartTime float64
	EndTime   float64
	Title     string
}

// ProbeBackend defines the interface for media probing backends
type ProbeBackend interface {
	// Probe extracts metadata from a media file
	Probe(ctx context.Context, path string) (*ProbeData, error)
}

// BackendType represents the type of probe backend
type BackendType string

const (
	// BackendFFProbe uses the ffprobe command-line tool (default, portable)
	BackendFFProbe BackendType = "ffprobe"
	// BackendAstiav uses libavformat via CGO (faster, requires FFmpeg dev libs)
	BackendAstiav BackendType = "astiav"
)

// NewProbeBackend creates a new probe backend of the specified type
func NewProbeBackend(backendType BackendType) (ProbeBackend, error) {
	switch backendType {
	case BackendFFProbe:
		return newFFProbeBackend(), nil
	case BackendAstiav:
		return newAstiavBackend()
	default:
		return newFFProbeBackend(), nil
	}
}
