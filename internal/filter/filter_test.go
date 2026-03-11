package filter

import (
	"regexp"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestApply_SizeFilter(t *testing.T) {
	media := []models.Media{
		{Path: "/test/small.mp4", Size: new(int64(1000))},
		{Path: "/test/medium.mp4", Size: new(int64(5000))},
		{Path: "/test/large.mp4", Size: new(int64(10000))},
	}

	criteria := Criteria{
		MinSize: 2000,
		MaxSize: 8000,
	}

	result := Apply(media, criteria)

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if result[0].Path != "/test/medium.mp4" {
		t.Fatalf("Expected medium.mp4, got %s", result[0].Path)
	}
}

func TestApply_DurationFilter(t *testing.T) {
	media := []models.Media{
		{Path: "/test/short.mp4", Duration: new(int64(300))},
		{Path: "/test/medium.mp4", Duration: new(int64(1800))},
		{Path: "/test/long.mp4", Duration: new(int64(7200))},
		{Path: "/test/info.txt", Duration: nil},
	}

	criteria := Criteria{
		MinDuration: 600,
		MaxDuration: 3600,
	}

	result := Apply(media, criteria)

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if *result[0].Duration != 1800 {
		t.Fatalf("Expected duration 1800, got %d", *result[0].Duration)
	}
}

func TestApply_PathContains(t *testing.T) {
	media := []models.Media{
		{Path: "/movies/2024/1080p/movie.mp4"},
		{Path: "/movies/2023/720p/movie.mp4"},
		{Path: "/tv/2024/1080p/show.mp4"},
	}

	criteria := Criteria{
		PathContains: []string{"2024", "1080p"},
	}

	result := Apply(media, criteria)

	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result))
	}
}

func TestApply_Regex(t *testing.T) {
	media := []models.Media{
		{Path: "/tv/Show.S01E01.mp4"},
		{Path: "/tv/Show.S01E02.mp4"},
		{Path: "/tv/Show.S02E01.mp4"},
	}

	criteria := Criteria{
		Regex: regexp.MustCompile(`S01E\d+`),
	}

	result := Apply(media, criteria)

	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result))
	}
}

func TestApply_IncludeExclude(t *testing.T) {
	media := []models.Media{
		{Path: "/test/movie.mp4"},
		{Path: "/test/movie.sample.mp4"},
		{Path: "/test/show.mkv"},
	}

	criteria := Criteria{
		Include: []string{".mp4"},
		Exclude: []string{"sample"},
	}

	result := Apply(media, criteria)

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if result[0].Path != "/test/movie.mp4" {
		t.Fatalf("Expected movie.mp4, got %s", result[0].Path)
	}
}

func TestApply_EmptyCriteria(t *testing.T) {
	media := []models.Media{
		{Path: "/test/file1.mp4"},
		{Path: "/test/file2.mp4"},
	}

	criteria := Criteria{}
	result := Apply(media, criteria)

	if len(result) != len(media) {
		t.Fatalf("Expected %d results, got %d", len(media), len(result))
	}
}

func TestApply_Exists(t *testing.T) {
	media := []models.Media{
		{Path: "filter_test.go"}, // This file exists
		{Path: "/non/existent/file"},
	}

	criteria := Criteria{Exists: true}
	result := Apply(media, criteria)

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
}
