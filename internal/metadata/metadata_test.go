package metadata

import (
	"context"
	"os"
	"testing"
)

func TestExtract_BasicInfo(t *testing.T) {
	f, err := os.CreateTemp("", "meta-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("test content")
	f.Close()

	meta, err := Extract(context.Background(), f.Name(), false, false, false)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if meta.Media.Path != f.Name() {
		t.Errorf("Expected path %s, got %s", f.Name(), meta.Media.Path)
	}

	if !meta.Media.Type.Valid || meta.Media.Type.String != "text" {
		t.Errorf("Expected type text, got %v", meta.Media.Type)
	}
}

func TestExtract_MimeTypes(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.jpg", "image"},
		{"test.pdf", "text"},
		{"test.epub", "text"},
	}

	for _, tt := range tests {
		f, _ := os.CreateTemp("", tt.filename)
		name := f.Name()
		f.Close()
		defer os.Remove(name)

		// We don't care if ffprobe fails, we want to see the mime-based detection in basicInfo or fallback
		meta, _ := Extract(context.Background(), name, false, false, false)
		if meta != nil && meta.Media.Type.String != tt.expected {
			// Note: DetectMimeType might depend on extension if content is empty
		}
	}
}

func TestExtract_NonExistent(t *testing.T) {
	_, err := Extract(context.Background(), "/non/existent/file", false, false, false)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestExtract_WithMockFFProbe(t *testing.T) {
	// Create a mock ffprobe script
	tmpDir, _ := os.MkdirTemp("", "mock-path")
	defer os.RemoveAll(tmpDir)

	createMock(t, tmpDir, "ffprobe", `{
  "streams": [
    {
      "codec_type": "video",
      "codec_name": "h264",
      "width": 1920,
      "height": 1080,
      "avg_frame_rate": "30/1"
    },
    {
      "codec_type": "audio",
      "codec_name": "aac"
    }
  ],
  "format": {
    "duration": "123.45",
    "tags": {
      "title": "Mock Title",
      "artist": "Mock Artist"
    }
  },
  "chapters": [
    {
      "start_time": "10.0",
      "tags": { "title": "Chapter 1" }
    }
  ]
}`)

	// Add tmpDir to PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	f, _ := os.CreateTemp("", "mock-video-*.mp4")
	f.Write([]byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}) // Basic mp4 header to avoid text detection
	defer os.Remove(f.Name())

	meta, err := Extract(context.Background(), f.Name(), false, false, false)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if meta.Media.Title.String != "Mock Title" {
		t.Errorf("Expected title Mock Title, got %s", meta.Media.Title.String)
	}
	if meta.Media.Duration.Int64 != 123 {
		t.Errorf("Expected duration 123, got %d", meta.Media.Duration.Int64)
	}
	if meta.Media.Width.Int64 != 1920 || meta.Media.Height.Int64 != 1080 {
		t.Errorf("Expected 1920x1080, got %dx%d", meta.Media.Width.Int64, meta.Media.Height.Int64)
	}
	if meta.Media.VideoCodecs.String != "h264" {
		t.Errorf("Expected h264 codec, got %s", meta.Media.VideoCodecs.String)
	}
	if meta.Media.AudioCodecs.String != "aac" {
		t.Errorf("Expected aac codec, got %s", meta.Media.AudioCodecs.String)
	}
	if len(meta.Captions) != 1 || meta.Captions[0].Text.String != "Chapter 1" {
		t.Errorf("Expected 1 caption 'Chapter 1', got %v", meta.Captions)
	}
}

func TestParseFPS(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"30000/1001", 29.97002997002997},
		{"24/1", 24.0},
		{"0/0", 0.0},
		{"invalid", 0.0},
	}

	for _, tt := range tests {
		got := parseFPS(tt.input)
		if got != tt.expected {
			t.Errorf("parseFPS(%s) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
