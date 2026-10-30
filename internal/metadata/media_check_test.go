package metadata_test

import (
	"context"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/metadata"
)

func TestDecodeQuickScan_MockFFmpeg(t *testing.T) {
	tmpDir := t.TempDir()

	createMock(t, tmpDir, "ffmpeg", `
for arg in "$@"; do
    if [ "$arg" = "20.00" ]; then
        exit 1
    fi
done
exit 0
`)

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)

	scans := []float64{10.0, 20.0, 30.0, 40.0}
	corruption, err := metadata.DecodeQuickScan(context.Background(), "dummy.mp4", scans, 1.0)
	if err != nil {
		t.Fatalf("metadata.DecodeQuickScan failed: %v", err)
	}

	// One scan (20.0) should fail out of four
	if corruption != 0.25 {
		t.Errorf("Expected corruption 0.25, got %f", corruption)
	}
}

func TestDecodeFullScan_MockFFProbe(t *testing.T) {
	tmpDir := t.TempDir()

	createMock(t, tmpDir, "ffprobe", `{
  "streams": [
    {
      "r_frame_rate": "30/1",
      "nb_read_frames": "3000"
    }
  ],
  "format": {
    "duration": "100.0"
  }
}`)

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)

	// actualDuration = 3000 * 1 / 30 = 100.0
	// metadataDuration = 100.0
	// corruption = 0.0
	corruption, err := metadata.DecodeFullScan(context.Background(), "dummy.mp4")
	if err != nil {
		t.Fatalf("metadata.DecodeFullScan failed: %v", err)
	}
	if corruption != 0.0 {
		t.Errorf("Expected corruption 0.0, got %f", corruption)
	}
}

func TestDecodeQuickScan_MissingTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	corruption, err := metadata.DecodeQuickScan(
		context.Background(),
		"dummy.mp4",
		[]float64{0, 10},
		1.0,
	)
	if err == nil {
		t.Fatal("expected error when ffmpeg is not available")
	}
	if corruption != 0 {
		t.Errorf("expected no corruption value on tool failure, got %f", corruption)
	}
}

func TestDecodeFullScan_Failures(t *testing.T) {
	tests := []struct {
		name    string
		tool    string
		noTool  bool
		content string
	}{
		{name: "not found", noTool: true},
		{name: "invalid json", tool: "ffprobe", content: `not json`},
		{name: "zero streams", tool: "ffprobe", content: `{"streams":[],"format":{"duration":"100.0"}}`},
		{
			name:    "zero duration",
			tool:    "ffprobe",
			content: `{"streams":[{"r_frame_rate":"30/1","nb_read_frames":"10"}],"format":{"duration":"0"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if tt.noTool {
				t.Setenv("PATH", tmpDir)
			} else {
				createMock(t, tmpDir, tt.tool, tt.content)
				t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			}

			corruption, err := metadata.DecodeFullScan(context.Background(), "dummy.mp4")
			if err == nil {
				t.Fatal("expected an error")
			}
			if corruption != 0 {
				t.Errorf("expected corruption 0 on failure, got %f", corruption)
			}
		})
	}
}

func TestDecodeFullScan_AboveThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	// 1500 frames at 30fps = 50s, but metadata says 100s => 0.5 corruption.
	createMock(t, tmpDir, "ffprobe", `{
  "streams": [
    {
      "r_frame_rate": "30/1",
      "nb_read_frames": "1500"
    }
  ],
  "format": {
    "duration": "100.0"
  }
}`)
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	corruption, err := metadata.DecodeFullScan(context.Background(), "dummy.mp4")
	if err != nil {
		t.Fatalf("metadata.DecodeFullScan failed: %v", err)
	}
	if corruption != 0.5 {
		t.Errorf("Expected corruption 0.5, got %f", corruption)
	}
}
