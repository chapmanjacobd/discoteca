package aggregate_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/aggregate"
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

	tests := []struct {
		name     string
		other    models.MediaWithDB
		expected bool
	}{
		{"4% size diff", models.MediaWithDB{Media: models.Media{Size: &s96, Duration: &d5}}, true},
		{"10% size diff", models.MediaWithDB{Media: models.Media{Size: &s110, Duration: &d5}}, false},
		{"large duration diff", models.MediaWithDB{Media: models.Media{Size: &s100, Duration: &d10}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aggregate.IsSameGroup(flags, m0, tt.other); got != tt.expected {
				t.Errorf("IsSameGroup() = %v, want %v", got, tt.expected)
			}
		})
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

	tests := []struct {
		name     string
		other    models.FolderStats
		expected bool
	}{
		{"4% count diff", models.FolderStats{ExistsCount: 96}, true},
		{"10% count diff", models.FolderStats{ExistsCount: 110}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aggregate.IsSameFolderGroup(flags, f0, tt.other); got != tt.expected {
				t.Errorf("IsSameFolderGroup() = %v, want %v", got, tt.expected)
			}
		})
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

	got := aggregate.ClusterByNumbers(flags, media)
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

	got := aggregate.ClusterFoldersByNumbers(flags, folders)
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

	got := aggregate.ByFolder(media)
	if len(got) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(got))
	}
}

func TestSortFolders_Aggregate(t *testing.T) {
	tests := []struct {
		name     string
		sortBy   string
		reverse  bool
		expected string
	}{
		{"by path asc", "path", false, "a"},
		{"by path desc", "path", true, "b"},
		{"by count asc", "count", false, "a"},
		{"by count desc", "count", true, "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folders := []aggregate.FolderStats{
				{Path: "b", Count: 2},
				{Path: "a", Count: 1},
			}
			aggregate.SortFolders(folders, tt.sortBy, tt.reverse)
			if folders[0].Path != tt.expected {
				t.Errorf("SortFolders(%s, %v) = %s, want %s", tt.sortBy, tt.reverse, folders[0].Path, tt.expected)
			}
		})
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

	got := aggregate.FilterNearDuplicates(groups)
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

	got := aggregate.ClusterFoldersByName(
		models.GlobalFlags{SimilarityFlags: models.SimilarityFlags{Similar: false}},
		folders,
	)
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

	got := aggregate.ClusterPaths(models.GlobalFlags{SimilarityFlags: models.SimilarityFlags{Clusters: 2}}, lines)
	if len(got) < 1 {
		t.Error("ClusterPaths returned no groups")
	}
	// KMeans is non-deterministic but with 2 clusters it should produce some grouping
}
