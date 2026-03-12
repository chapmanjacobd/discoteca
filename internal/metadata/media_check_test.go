package metadata

import (
	"context"
	"os"
	"testing"
)

func TestDecodeQuickScan_MockFFmpeg(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "mock-ffmpeg-path")
	defer os.RemoveAll(tmpDir)

	createMock(t, tmpDir, "ffmpeg", `
for arg in "$@"; do
    if [ "$arg" = "20.00" ]; then
        exit 1
    fi
done
exit 0
`)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	scans := []float64{10.0, 20.0, 30.0, 40.0}
	corruption := DecodeQuickScan(context.Background(), "dummy.mp4", scans, 1.0)

	// One scan (20.0) should fail out of four
	if corruption != 0.25 {
		t.Errorf("Expected corruption 0.25, got %f", corruption)
	}
}

func TestDecodeFullScan_MockFFProbe(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "mock-ffprobe-path")
	defer os.RemoveAll(tmpDir)

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
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	// actualDuration = 3000 * 1 / 30 = 100.0
	// metadataDuration = 100.0
	// corruption = 0.0
	corruption, err := DecodeFullScan(context.Background(), "dummy.mp4")
	if err != nil {
		t.Fatalf("DecodeFullScan failed: %v", err)
	}
	if corruption != 0.0 {
		t.Errorf("Expected corruption 0.0, got %f", corruption)
	}
}
