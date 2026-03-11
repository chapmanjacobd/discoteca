package query

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestAggregateExtensions(t *testing.T) {
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "test1.mp4"}},
		{Media: models.Media{Path: "test2.MP4"}},
		{Media: models.Media{Path: "test3.mkv"}},
		{Media: models.Media{Path: "test4"}},
	}

	got := AggregateExtensions(media)
	if len(got) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(got))
	}

	foundMp4 := false
	for _, g := range got {
		if g.Path == ".mp4" {
			foundMp4 = true
			if g.Count != 2 {
				t.Errorf("Expected 2 files for .mp4, got %d", g.Count)
			}
		}
	}
	if !foundMp4 {
		t.Error("Did not find .mp4 group")
	}
}

func TestAggregateMimeTypes(t *testing.T) {
	video := "video/mp4"
	audio := "audio/mpeg"
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "v1.mp4", Type: &video}},
		{Media: models.Media{Path: "v2.mp4", Type: &video}},
		{Media: models.Media{Path: "a1.mp3", Type: &audio}},
	}

	got := AggregateMimeTypes(media)
	if len(got) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(got))
	}
}

func TestAggregateSizeBuckets(t *testing.T) {
	size1KB := int64(1024)
	size2KB := int64(2048)
	size5KB := int64(5120)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "1kb", Size: &size1KB}},
		{Media: models.Media{Path: "2kb", Size: &size2KB}},
		{Media: models.Media{Path: "5kb", Size: &size5KB}},
	}

	got := AggregateSizeBuckets(media)
	if len(got) != 3 {
		t.Errorf("Expected 3 size buckets, got %d", len(got))
	}
}

func TestAggregateByDepth(t *testing.T) {
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "/home/user/vids/v1.mp4"}},
		{Media: models.Media{Path: "/home/user/vids/v2.mp4"}},
		{Media: models.Media{Path: "/home/user/music/a1.mp3"}},
	}

	// Depth 3: /home/user/vids and /home/user/music
	got := AggregateByDepth(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 3}})
	if len(got) != 2 {
		t.Errorf("Expected 2 groups at depth 3, got %d", len(got))
	}

	// Parents mode
	got = AggregateByDepth(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Parents: true, MinDepth: 1}})
	// Should have: /home, /home/user, /home/user/vids, /home/user/vids/v1.mp4, /home/user/vids/v2.mp4, /home/user/music, /home/user/music/a1.mp3
	if len(got) != 7 {
		t.Errorf("Expected 7 groups in parents mode, got %d", len(got))
	}
}

func TestAggregateByDepthExtended(t *testing.T) {
	size100 := int64(100)
	size200 := int64(200)
	size300 := int64(300)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "/dir1/f1.mp4", Size: &size100}},
		{Media: models.Media{Path: "/dir1/f2.mp4", Size: &size200}},
		{Media: models.Media{Path: "/dir2/f3.mp4", Size: &size300}},
	}

	got := AggregateByDepth(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 1}})
	if len(got) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(got))
	}

	for _, g := range got {
		if g.Path == "/dir1" {
			if g.ExistsCount != 2 {
				t.Errorf("Expected 2 files in /dir1, got %d", g.ExistsCount)
			}
			if g.TotalSize != 300 {
				t.Errorf("Expected 300 total size in /dir1, got %d", g.TotalSize)
			}
			if g.MedianSize != 150 {
				t.Errorf("Expected 150 median size in /dir1, got %d", g.MedianSize)
			}
		}
	}
}

func TestAggregateMediaAllModes(t *testing.T) {
	size100 := int64(100)
	video := "video/mp4"
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "test.mp4", Size: &size100, Type: &video}},
	}

	// Extensions
	got := AggregateMedia(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{GroupByExtensions: true}})
	if len(got) != 1 || got[0].Path != ".mp4" {
		t.Errorf("Extensions mode failed: %v", got)
	}

	// MimeTypes
	got = AggregateMedia(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{GroupByMimeTypes: true}})
	if len(got) != 1 || got[0].Path != video {
		t.Errorf("MimeTypes mode failed: %v", got)
	}

	// Size
	got = AggregateMedia(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{GroupBySize: true}})
	if len(got) != 1 {
		t.Errorf("Size mode failed: %v", got)
	}
}

func TestAggregatePostFilteringExtra(t *testing.T) {
	size100 := int64(100)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "/dir1/f1.mp4", Size: &size100}},
		{Media: models.Media{Path: "/dir1/f2.mp4", Size: &size100}},
		{Media: models.Media{Path: "/dir2/f1.mp4", Size: &size100}},
	}

	// Filter by FileCounts > 1
	got := AggregateMedia(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 1, FileCounts: ">1"}})
	if len(got) != 1 || got[0].Path != "/dir1" {
		t.Errorf("FileCounts filtering failed: %v", got)
	}

	// Filter by FoldersOnly
	got = AggregateMedia(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 1, FoldersOnly: true}})
	if len(got) != 2 {
		t.Errorf("FoldersOnly failed: %v", got)
	}
}
