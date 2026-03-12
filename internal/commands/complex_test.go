package commands

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestComplexFilteringAndAggregation(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	// Create some files with different properties
	// We'll manually insert them into the DB to have controlled metadata
	dbPath := fixture.DBPath
	sqlDB, _ := sql.Open("sqlite3", dbPath)
	defer sqlDB.Close()
	db.InitDB(sqlDB)

	files := []struct {
		path     string
		size     int64
		duration int64
		ext      string
	}{
		{filepath.FromSlash("/root/dir1/video1.mp4"), 100 * 1024 * 1024, 600, ".mp4"},   // 100MB, 10min
		{filepath.FromSlash("/root/dir1/video2.mkv"), 500 * 1024 * 1024, 1200, ".mkv"},  // 500MB, 20min
		{filepath.FromSlash("/root/dir2/audio1.mp3"), 10 * 1024 * 1024, 300, ".mp3"},    // 10MB, 5min
		{filepath.FromSlash("/root/dir2/video3.mp4"), 1000 * 1024 * 1024, 3600, ".mp4"}, // 1GB, 60min
	}

	for _, f := range files {
		sqlDB.Exec("INSERT INTO media (path, size, duration, type) VALUES (?, ?, ?, ?)",
			f.path, f.size, f.duration, strings.TrimPrefix(f.ext, "."))
	}

	t.Run("CombineFilters", func(t *testing.T) {
		// Filter: Size > 50MB, Duration < 30min, Extension .mp4
		cmd := &PrintCmd{
			FilterFlags: models.FilterFlags{
				Size:     []string{">50MB"},
				Duration: []string{"<30min"},
			},
			MediaFilterFlags: models.MediaFilterFlags{
				Ext: []string{".mp4"},
			},
			DisplayFlags: models.DisplayFlags{
				JSON: true,
			},
			Args: []string{dbPath},
		}
		cmd.AfterApply()

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Run(nil)
		w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}

		var results []models.MediaWithDB
		json.NewDecoder(r).Decode(&results)

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		} else if filepath.ToSlash(results[0].Path) != "/root/dir1/video1.mp4" {
			t.Errorf("Expected video1.mp4, got %s", results[0].Path)
		}
	})

	t.Run("AggregationAndSorting", func(t *testing.T) {
		// Aggregate by directory (BigDirs), sort by size reverse
		cmd := &PrintCmd{
			AggregateFlags: models.AggregateFlags{
				BigDirs: true,
			},
			SortFlags: models.SortFlags{
				SortBy:  "size",
				Reverse: true,
			},
			DisplayFlags: models.DisplayFlags{
				JSON: true,
			},
			Args: []string{dbPath},
		}
		cmd.AfterApply()

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Run(nil)
		w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}

		var folders []models.FolderStats
		json.NewDecoder(r).Decode(&folders)

		if len(folders) != 2 {
			t.Errorf("Expected 2 folders, got %d", len(folders))
		}
		// /root/dir2 should be first (1GB + 10MB > 100MB + 500MB)
		if !strings.Contains(filepath.ToSlash(folders[0].Path), "/root/dir2") {
			t.Errorf("Expected /root/dir2 to be first, got %s", folders[0].Path)
		}
	})
}

func TestClusterSort(t *testing.T) {
	input := `/path/to/movie_part1.mp4
/path/to/movie_part2.mp4
/other/file.txt
`

	t.Run("BasicClustering", func(t *testing.T) {
		cmd := &ClusterSortCmd{
			SimilarityFlags: models.SimilarityFlags{
				PrintGroups: true,
			},
			InputPath: "-",
		}

		// Mock stdin
		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		w.WriteString(input)
		w.Close()

		// Capture stdout
		oldStdout := os.Stdout
		ro, wo, _ := os.Pipe()
		os.Stdout = wo

		err := cmd.Run(nil)
		wo.Close()
		os.Stdin = oldStdin
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("ClusterSortCmd failed: %v", err)
		}

		var groups []models.FolderStats
		json.NewDecoder(ro).Decode(&groups)

		if len(groups) < 1 {
			t.Errorf("Expected at least one group, got 0")
		}
	})
}

func TestStatsWithFrequency(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	dbPath := fixture.DBPath
	sqlDB, _ := sql.Open("sqlite3", dbPath)
	db.InitDB(sqlDB)

	now := 1708358400 // 2024-02-19
	sqlDB.Exec("INSERT INTO media (path, size, duration, time_last_played) VALUES (?, ?, ?, ?)",
		filepath.FromSlash("/path1"), 100, 60, now)
	sqlDB.Exec("INSERT INTO media (path, size, duration, time_last_played) VALUES (?, ?, ?, ?)",
		filepath.FromSlash("/path2"), 200, 120, now-86400) // yesterday
	sqlDB.Close()

	cmd := &StatsCmd{
		Facet:     "watched",
		Databases: []string{dbPath},
		DisplayFlags: models.DisplayFlags{
			Frequency: "daily",
			JSON:      true,
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Run(nil)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("StatsCmd failed: %v", err)
	}

	var stats []any
	json.NewDecoder(r).Decode(&stats)
	if len(stats) == 0 {
		t.Errorf("Expected stats, got none")
	}
}
