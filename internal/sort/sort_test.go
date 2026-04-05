package sort_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/sort"
)

func TestApply_BySize(t *testing.T) {
	tests := []struct {
		name     string
		reverse  bool
		expected []int64
	}{
		{"ascending", false, []int64{1000, 2000, 3000}},
		{"descending", true, []int64{3000, 2000, 1000}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			media := []models.Media{
				{Path: "/c", Size: new(int64(3000))},
				{Path: "/a", Size: new(int64(1000))},
				{Path: "/b", Size: new(int64(2000))},
			}
			sort.Apply(media, sort.BySize, tt.reverse, false)

			for i, m := range media {
				if *m.Size != tt.expected[i] {
					t.Errorf("Index %d: expected size %d, got %d", i, tt.expected[i], *m.Size)
				}
			}
		})
	}
}

func TestApply_NaturalSort(t *testing.T) {
	media := []models.Media{
		{Path: "/show/episode10.mp4"},
		{Path: "/show/episode2.mp4"},
		{Path: "/show/episode1.mp4"},
		{Path: "/show/episode20.mp4"},
	}

	sort.Apply(media, sort.ByPath, false, true)

	expected := []string{
		"/show/episode1.mp4",
		"/show/episode2.mp4",
		"/show/episode10.mp4",
		"/show/episode20.mp4",
	}

	for i, m := range media {
		if m.Path != expected[i] {
			t.Errorf("Index %d: expected %s, got %s", i, expected[i], m.Path)
		}
	}
}

func TestApply_ByOtherFields(t *testing.T) {
	titleA := "A"
	titleB := "B"
	var dur100 int64 = 100
	var dur200 int64 = 200
	var time100 int64 = 100
	var time200 int64 = 200

	tests := []struct {
		name      string
		method    sort.Method
		reverse   bool
		natural   bool
		checkFunc func([]models.Media) bool
	}{
		{
			"sort.ByTitle",
			sort.ByTitle,
			false,
			false,
			func(m []models.Media) bool { return *m[0].Title == "A" },
		},
		{
			"sort.ByDuration",
			sort.ByDuration,
			false,
			false,
			func(m []models.Media) bool { return *m[0].Duration == 100 },
		},
		{
			"sort.ByTimeCreated",
			sort.ByTimeCreated,
			false,
			false,
			func(m []models.Media) bool { return *m[0].TimeCreated == 100 },
		},
		{
			"sort.ByTimeModified",
			sort.ByTimeModified,
			false,
			false,
			func(m []models.Media) bool { return *m[0].TimeModified == 100 },
		},
		{
			"sort.ByTimePlayed",
			sort.ByTimePlayed,
			false,
			false,
			func(m []models.Media) bool { return *m[0].TimeLastPlayed == 100 },
		},
		{
			"sort.ByPlayCount",
			sort.ByPlayCount,
			false,
			false,
			func(m []models.Media) bool { return *m[0].PlayCount == 100 },
		},
		{
			"sort.ByPath invalid fallback",
			sort.Method("invalid"),
			false,
			false,
			func(m []models.Media) bool { return m[0].Path == "1" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			media := []models.Media{
				{
					Path:           "2",
					Title:          &titleB,
					Duration:       &dur200,
					TimeCreated:    &time200,
					TimeModified:   &time200,
					TimeLastPlayed: &time200,
					PlayCount:      &time200,
				},
				{
					Path:           "1",
					Title:          &titleA,
					Duration:       &dur100,
					TimeCreated:    &time100,
					TimeModified:   &time100,
					TimeLastPlayed: &time100,
					PlayCount:      &time100,
				},
			}
			sort.Apply(media, tt.method, tt.reverse, tt.natural)
			if !tt.checkFunc(media) {
				t.Errorf("%s check failed", tt.name)
			}
		})
	}
}
