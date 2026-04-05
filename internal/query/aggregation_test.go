package query_test

import (
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils/pathutil"
)

func TestAggregateExtensions(t *testing.T) {
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "test1.mp4"}},
		{Media: models.Media{Path: "test2.MP4"}},
		{Media: models.Media{Path: "test3.mkv"}},
		{Media: models.Media{Path: "test4"}},
	}

	got := query.AggregateExtensions(media)
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

func TestAggregateSizeBuckets(t *testing.T) {
	size1KB := int64(1024)
	size2KB := int64(2048)
	size5KB := int64(5120)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "1kb", Size: &size1KB}},
		{Media: models.Media{Path: "2kb", Size: &size2KB}},
		{Media: models.Media{Path: "5kb", Size: &size5KB}},
	}

	got := query.AggregateSizeBuckets(media)
	if len(got) != 3 {
		t.Errorf("Expected 3 size buckets, got %d", len(got))
	}
}

func TestAggregateByDepth(t *testing.T) {
	media := []models.MediaWithDB{
		{Media: models.Media{Path: filepath.FromSlash("/home/user/vids/v1.mp4")}},
		{Media: models.Media{Path: filepath.FromSlash("/home/user/vids/v2.mp4")}},
		{Media: models.Media{Path: filepath.FromSlash("/home/user/music/a1.mp3")}},
	}

	// Depth 3: /home/user/vids and /home/user/music
	got := query.AggregateByDepth(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 3}})
	if len(got) != 2 {
		t.Errorf("Expected 2 groups at depth 3, got %d", len(got))
	}

	// Parents mode
	got = query.AggregateByDepth(
		media,
		models.GlobalFlags{AggregateFlags: models.AggregateFlags{Parents: true, MinDepth: 1}},
	)
	// Should have: /home, /home/user, /home/user/vids, /home/user/vids/v1.mp4, /home/user/vids/v2.mp4, /home/user/music, /home/user/music/a1.mp3
	if len(got) != 7 {
		t.Errorf("Expected 7 groups in parents mode, got %d", len(got))
	}
}

// TestAggregateByDepth_WindowsPaths tests pathutil.Split with Windows-style paths
// This ensures Windows paths are parsed correctly regardless of the OS
func TestAggregateByDepth_WindowsPaths(t *testing.T) {
	// Test pathutil.Split directly with Windows paths (backslashes)
	tests := []struct {
		path      string
		wantParts []string
		wantAbs   bool
	}{
		// Windows paths with backslashes
		{"C:\\Users\\user\\vids\\v1.mp4", []string{"C:", "Users", "user", "vids", "v1.mp4"}, true},
		{"D:\\data\\file.txt", []string{"D:", "data", "file.txt"}, true},
		{"C:\\", []string{"C:"}, true},

		// Windows paths with forward slashes (also valid on Windows)
		{"C:/Users/user/vids/v2.mp4", []string{"C:", "Users", "user", "vids", "v2.mp4"}, true},

		// UNC paths
		{"\\\\server\\share\\file", []string{"server", "share", "file"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			parts, isAbs := pathutil.Split(tt.path)
			if len(parts) != len(tt.wantParts) {
				t.Errorf("Split(%q) returned %d parts, want %d: %v", tt.path, len(parts), len(tt.wantParts), parts)
			}
			if isAbs != tt.wantAbs {
				t.Errorf("Split(%q) isAbs=%v, want %v", tt.path, isAbs, tt.wantAbs)
			}
			for i, p := range parts {
				if i < len(tt.wantParts) && p != tt.wantParts[i] {
					t.Errorf("Split(%q) part[%d]=%q, want %q", tt.path, i, p, tt.wantParts[i])
				}
			}
		})
	}
}

func TestAggregateByDepthExtended(t *testing.T) {
	size100 := int64(100)
	size200 := int64(200)
	size300 := int64(300)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: filepath.FromSlash("/dir1/f1.mp4"), Size: &size100}},
		{Media: models.Media{Path: filepath.FromSlash("/dir1/f2.mp4"), Size: &size200}},
		{Media: models.Media{Path: filepath.FromSlash("/dir2/f3.mp4"), Size: &size300}},
	}

	got := query.AggregateByDepth(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 1}})
	if len(got) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(got))
	}

	for _, g := range got {
		if g.Path == filepath.FromSlash("/dir1") {
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
		{Media: models.Media{Path: "test.mp4", Size: &size100, MediaType: &video}},
	}

	// Extensions
	got := query.AggregateMedia(
		media,
		models.GlobalFlags{AggregateFlags: models.AggregateFlags{GroupByExtensions: true}},
	)
	if len(got) != 1 || got[0].Path != ".mp4" {
		t.Errorf("Extensions mode failed: %v", got)
	}

	// Size
	got = query.AggregateMedia(media, models.GlobalFlags{AggregateFlags: models.AggregateFlags{GroupBySize: true}})
	if len(got) != 1 {
		t.Errorf("Size mode failed: %v", got)
	}
}

func TestAggregatePostFilteringExtra(t *testing.T) {
	size100 := int64(100)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: filepath.FromSlash("/dir1/f1.mp4"), Size: &size100}},
		{Media: models.Media{Path: filepath.FromSlash("/dir1/f2.mp4"), Size: &size100}},
		{Media: models.Media{Path: filepath.FromSlash("/dir2/f1.mp4"), Size: &size100}},
	}

	// Filter by FileCounts > 1
	got := query.AggregateMedia(
		media,
		models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 1, FileCounts: ">1"}},
	)
	if len(got) != 1 || filepath.ToSlash(got[0].Path) != filepath.ToSlash(filepath.FromSlash("/dir1")) {
		t.Errorf("FileCounts filtering failed: %v", got)
	}

	// Filter by FoldersOnly
	got = query.AggregateMedia(
		media,
		models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 1, FoldersOnly: true}},
	)
	if len(got) != 2 {
		t.Errorf("FoldersOnly failed: %v", got)
	}

	t.Run("WindowsStyleBackslashes", func(t *testing.T) {
		winMedia := []models.MediaWithDB{
			{Media: models.Media{Path: "C:\\videos\\funny\\cat.mp4", Size: &size100}},
			{Media: models.Media{Path: "C:\\videos\\funny\\dog.mp4", Size: &size100}},
		}
		// Aggregate at depth 2 (C:\videos\funny)
		agg := query.AggregateMedia(winMedia, models.GlobalFlags{AggregateFlags: models.AggregateFlags{Depth: 2}})
		if len(agg) != 1 {
			t.Errorf("Expected 1 aggregated folder, got %d", len(agg))
		}
		expected := filepath.FromSlash("C:/videos")
		if agg[0].Path != expected {
			t.Errorf("Expected %s, got %s", expected, agg[0].Path)
		}
		if agg[0].Count != 2 {
			t.Errorf("Expected count 2, got %d", agg[0].Count)
		}
	})
}
