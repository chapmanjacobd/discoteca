package aggregate

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestIsSameGroup(t *testing.T) {
	s100 := int64(100)
	s96 := int64(96)
	s110 := int64(110)
	d5 := int64(5)
	d10 := int64(10)

	flags := models.GlobalFlags{
		SimilarityFlags: models.SimilarityFlags{
			FilterSizes:     true,
			FilterDurations: true,
			SizesDelta:      5.0,
			DurationsDelta:  5.0,
		},
	}

	m0 := models.MediaWithDB{Media: models.Media{Size: &s100, Duration: &d5}}

	if !IsSameGroup(flags, m0, models.MediaWithDB{Media: models.Media{Size: &s96, Duration: &d5}}) {
		t.Error("Expected same group for 4% size diff")
	}
	if IsSameGroup(flags, m0, models.MediaWithDB{Media: models.Media{Size: &s110, Duration: &d5}}) {
		t.Error("Expected different group for 10% size diff")
	}
	if IsSameGroup(flags, m0, models.MediaWithDB{Media: models.Media{Size: &s100, Duration: &d10}}) {
		t.Error("Expected different group for large duration diff")
	}
}

func TestIsSameFolderGroup(t *testing.T) {
	flags := models.GlobalFlags{
		SimilarityFlags: models.SimilarityFlags{
			FilterCounts: true,
			CountsDelta:  5.0,
		},
	}

	f0 := models.FolderStats{ExistsCount: 100}

	if !IsSameFolderGroup(flags, f0, models.FolderStats{ExistsCount: 96}) {
		t.Error("Expected same folder group for 4% count diff")
	}
	if IsSameFolderGroup(flags, f0, models.FolderStats{ExistsCount: 110}) {
		t.Error("Expected different folder group for 10% count diff")
	}
}

func TestClusterByNumbers(t *testing.T) {
	s100 := int64(100)
	s104 := int64(104)
	s108 := int64(108)
	s116 := int64(116)
	d100 := int64(100)
	d104 := int64(104)
	d108 := int64(108)

	flags := models.GlobalFlags{
		SimilarityFlags: models.SimilarityFlags{
			FilterSizes:     true,
			FilterDurations: true,
			SizesDelta:      5.0,
			DurationsDelta:  5.0,
			Similar:         true,
		},
	}

	media := []models.MediaWithDB{
		{Media: models.Media{Path: "a", Size: &s100, Duration: &d100}},
		{Media: models.Media{Path: "b", Size: &s100, Duration: &d104}},
		{Media: models.Media{Path: "c", Size: &s104, Duration: &d104}},
		{Media: models.Media{Path: "d", Size: &s104, Duration: &d108}},
		{Media: models.Media{Path: "e", Size: &s108, Duration: &d108}},
		{Media: models.Media{Path: "f", Size: &s116, Duration: &d108}},
	}

	got := ClusterByNumbers(flags, media)
	// Python test said: [0, 0, 0, 1, 1, 2]
	// group 0: a, b, c
	// group 1: d, e
	// group 2: f
	// Since Similar=true, single-item groups are filtered out if OnlyDuplicates=true or Similar=true (in my impl)
	// Wait, similar_files.py: groups = [d for d in groups if len(d["grouped_paths"]) > 1]

	if len(got) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(got))
	}
}

func TestClusterFoldersByNumbers(t *testing.T) {
	flags := models.GlobalFlags{
		SimilarityFlags: models.SimilarityFlags{
			FilterCounts: true,
			CountsDelta:  5.0,
			Similar:      true,
		},
	}

	folders := []models.FolderStats{
		{Path: "/dir1", ExistsCount: 100, Count: 1},
		{Path: "/dir2", ExistsCount: 96, Count: 1},
		{Path: "/dir3", ExistsCount: 110, Count: 1},
	}

	got := ClusterFoldersByNumbers(flags, folders)
	if len(got) != 1 {
		t.Errorf("Expected 1 group, got %d", len(got))
	}
	if got[0].Count != 2 {
		t.Errorf("Expected group to have 2 folders, got %d", got[0].Count)
	}
}

func TestByFolder(t *testing.T) {
	s100 := int64(100)
	media := []models.Media{
		{Path: "/dir1/file1.mp4", Size: &s100},
		{Path: "/dir1/file2.mp4", Size: &s100},
		{Path: "/dir2/file3.mp4", Size: &s100},
	}

	got := ByFolder(media)
	if len(got) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(got))
	}
}

func TestSortFolders_Aggregate(t *testing.T) {
	folders := []FolderStats{
		{Path: "b", Count: 2},
		{Path: "a", Count: 1},
	}

	SortFolders(folders, "path", false)
	if folders[0].Path != "a" {
		t.Errorf("SortFolders by path failed")
	}
}

func TestFilterNearDuplicates(t *testing.T) {
	groups := []models.FolderStats{
		{
			Path: "/common",
			Files: []models.MediaWithDB{
				{Media: models.Media{Path: "/common/movie_final.mp4"}},
				{Media: models.Media{Path: "/common/movie_final_v2.mp4"}},
				{Media: models.Media{Path: "/common/something_else.mp4"}},
			},
		},
	}

	got := FilterNearDuplicates(groups)
	// movie_final and movie_final_v2 should be grouped together, something_else separate
	if len(got) < 2 {
		t.Errorf("Expected group to be split, got %d", len(got))
	}
}

func TestClusterFoldersByName(t *testing.T) {
	folders := []models.FolderStats{
		{Path: "/path/to/movie_part1", Count: 1},
		{Path: "/path/to/movie_part2", Count: 1},
		{Path: "/path/to/completely_different_thing", Count: 1},
	}

	got := ClusterFoldersByName(models.GlobalFlags{SimilarityFlags: models.SimilarityFlags{Similar: false}}, folders)
	if len(got) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(got))
	}
}

func TestClusterPaths(t *testing.T) {
	lines := []string{
		"/path/to/movie_a_part1.mp4",
		"/path/to/movie_a_part2.mp4",
		"/path/to/movie_b_part1.mp4",
		"/path/to/movie_b_part2.mp4",
		"/other/completely/different/file.txt",
	}

	got := ClusterPaths(models.GlobalFlags{SimilarityFlags: models.SimilarityFlags{Clusters: 2}}, lines)
	if len(got) < 1 {
		t.Error("ClusterPaths returned no groups")
	}
	// KMeans is non-deterministic but with 2 clusters it should produce some grouping
}
