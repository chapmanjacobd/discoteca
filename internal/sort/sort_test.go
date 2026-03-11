package sort

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestApply_BySize(t *testing.T) {
	media := []models.Media{
		{Path: "/c", Size: new(int64(3000))},
		{Path: "/a", Size: new(int64(1000))},
		{Path: "/b", Size: new(int64(2000))},
	}

	Apply(media, BySize, false, false)

	expected := []int64{1000, 2000, 3000}
	for i, m := range media {
		if *m.Size != expected[i] {
			t.Errorf("Index %d: expected size %d, got %d", i, expected[i], *m.Size)
		}
	}
}

func TestApply_BySizeReverse(t *testing.T) {
	media := []models.Media{
		{Path: "/c", Size: new(int64(3000))},
		{Path: "/a", Size: new(int64(1000))},
		{Path: "/b", Size: new(int64(2000))},
	}

	Apply(media, BySize, true, false)

	expected := []int64{3000, 2000, 1000}
	for i, m := range media {
		if *m.Size != expected[i] {
			t.Errorf("Index %d: expected size %d, got %d", i, expected[i], *m.Size)
		}
	}
}

func TestApply_NaturalSort(t *testing.T) {
	media := []models.Media{
		{Path: "/show/episode10.mp4"},
		{Path: "/show/episode2.mp4"},
		{Path: "/show/episode1.mp4"},
		{Path: "/show/episode20.mp4"},
	}

	Apply(media, ByPath, false, true)

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
	media := []models.Media{
		{Path: "2", Title: &titleB, Duration: &dur200, TimeCreated: &time200, TimeModified: &time200, TimeLastPlayed: &time200, PlayCount: &time200},
		{Path: "1", Title: &titleA, Duration: &dur100, TimeCreated: &time100, TimeModified: &time100, TimeLastPlayed: &time100, PlayCount: &time100},
	}

	Apply(media, ByTitle, false, false)
	if *media[0].Title != "A" {
		t.Errorf("Expected A first")
	}

	Apply(media, ByDuration, false, false)
	if *media[0].Duration != 100 {
		t.Errorf("Expected 100 first")
	}

	Apply(media, ByTimeCreated, false, false)
	if *media[0].TimeCreated != 100 {
		t.Errorf("Expected timeCreated 100 first")
	}

	Apply(media, ByTimeModified, false, false)
	if *media[0].TimeModified != 100 {
		t.Errorf("Expected timeModified 100 first")
	}

	Apply(media, ByTimePlayed, false, false)
	if *media[0].TimeLastPlayed != 100 {
		t.Errorf("Expected timePlayed 100 first")
	}

	Apply(media, ByPlayCount, false, false)
	if *media[0].PlayCount != 100 {
		t.Errorf("Expected playCount 100 first")
	}

	Apply(media, Method("invalid"), false, false)
	if media[0].Path != "1" {
		t.Errorf("Expected fallback to path")
	}
}
