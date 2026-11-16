package downloader

import "testing"

func TestNewMediaFilesSkipsSidecars(t *testing.T) {
	before := map[string]struct{}{}
	after := map[string]struct{}{
		"/dl/Fake/Channel/video.mp4":         {},
		"/dl/Fake/Channel/video.webp":        {},
		"/dl/Fake/Channel/video.en.vtt":      {},
		"/dl/Fake/Channel/cover.jpg":         {},
		"/dl/Fake/Channel/gallery.jpg":       {},
		"/dl/Fake/Channel/video.info.json":   {},
		"/dl/Fake/Channel/video.mp4.part":    {},
		"/dl/Fake/Channel/another-video.mkv": {},
	}
	got := newMediaFiles(before, after)
	if len(got) != 3 {
		t.Fatalf("newMediaFiles() = %v, want 3 media files", got)
	}
}
