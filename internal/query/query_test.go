package query_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestNewFilterBuilder(t *testing.T) {
	flags := models.GlobalFlags{QueryFlags: models.QueryFlags{Query: "SELECT 1"}}
	fb := query.NewFilterBuilder(flags)
	if fb.Flags.Query != "SELECT 1" {
		t.Errorf("Expected query SELECT 1, got %s", fb.Flags.Query)
	}
}

func TestFilterBuilder_Build(t *testing.T) {
	tests := []struct {
		name     string
		flags    models.GlobalFlags
		expected string
	}{
		{
			"Raw Query",
			models.GlobalFlags{QueryFlags: models.QueryFlags{Query: "SELECT * FROM test"}},
			"SELECT * FROM test",
		},
		{
			"Default Query",
			models.GlobalFlags{
				SortFlags:    models.SortFlags{SortBy: "path"},
				QueryFlags:   models.QueryFlags{Limit: 100},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 ORDER BY path ASC LIMIT 100",
		},
		{
			"Search Query",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Search: []string{"term"}},
				SortFlags:    models.SortFlags{SortBy: "path"},
				QueryFlags:   models.QueryFlags{Limit: 100},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND ((path LIKE ? OR title LIKE ? OR path_tokenized LIKE ?)) ORDER BY path ASC LIMIT 100",
		},
		{
			"Video Only",
			models.GlobalFlags{
				MediaFilterFlags: models.MediaFilterFlags{VideoOnly: true},
				SortFlags:        models.SortFlags{SortBy: "path"},
				QueryFlags:       models.QueryFlags{Limit: 100},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND (media_type = 'video') ORDER BY path ASC LIMIT 100",
		},
		{
			"Reverse Sort",
			models.GlobalFlags{
				SortFlags:    models.SortFlags{SortBy: "path", Reverse: true},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 ORDER BY path DESC LIMIT 10",
		},
		{
			"Random Sort",
			models.GlobalFlags{
				SortFlags:    models.SortFlags{Random: true},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND rowid IN (SELECT rowid FROM media WHERE COALESCE(time_deleted, 0) = 0 ORDER BY RANDOM() LIMIT 160) ORDER BY RANDOM() LIMIT 10",
		},
		{
			"FTS Search (Improved Join)",
			models.GlobalFlags{
				FTSFlags:     models.FTSFlags{FTS: true, FTSTable: "media_fts"},
				FilterFlags:  models.FilterFlags{Search: []string{"term"}},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT media.* FROM media JOIN media_fts ON media.rowid = media_fts.rowid WHERE COALESCE(media.time_deleted, 0) = 0 AND media_fts MATCH ? LIMIT 10",
		},
		{
			"FTS Search (Column specific)",
			models.GlobalFlags{
				FTSFlags:     models.FTSFlags{FTS: true, FTSTable: "media_fts"},
				FilterFlags:  models.FilterFlags{Search: []string{"title:term"}},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT media.* FROM media JOIN media_fts ON media.rowid = media_fts.rowid WHERE COALESCE(media.time_deleted, 0) = 0 AND media_fts MATCH ? LIMIT 10",
		},
		{
			"Flexible Search",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Search: []string{"a", "b"}, FlexibleSearch: true},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND ((path LIKE ? OR title LIKE ? OR path_tokenized LIKE ?) OR (path LIKE ? OR title LIKE ? OR path_tokenized LIKE ?)) LIMIT 10",
		},
		{
			"Mixed FTS and other filters",
			models.GlobalFlags{
				FTSFlags:         models.FTSFlags{FTS: true},
				FilterFlags:      models.FilterFlags{Search: []string{"term"}, Size: []string{">100MB"}},
				MediaFilterFlags: models.MediaFilterFlags{VideoOnly: true},
				QueryFlags:       models.QueryFlags{Limit: 10},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT media.* FROM media JOIN media_fts ON media.rowid = media_fts.rowid WHERE COALESCE(media.time_deleted, 0) = 0 AND (media.media_type = 'video') AND media.size >= ? AND media_fts MATCH ? LIMIT 10",
		},
		{
			"Only Deleted",
			models.GlobalFlags{
				DeletedFlags: models.DeletedFlags{OnlyDeleted: true},
				QueryFlags:   models.QueryFlags{Limit: 10},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) > 0 LIMIT 10",
		},
		{
			"Multiple Categories",
			models.GlobalFlags{
				MediaFilterFlags: models.MediaFilterFlags{Category: []string{"comedy", "music"}},
				QueryFlags:       models.QueryFlags{Limit: 10},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND (categories LIKE '%' || ? || '%' OR categories LIKE '%' || ? || '%') LIMIT 10",
		},
		{
			"Portrait",
			models.GlobalFlags{
				MediaFilterFlags: models.MediaFilterFlags{Portrait: true},
				QueryFlags:       models.QueryFlags{Limit: 10},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND width < height LIMIT 10",
		},
		{
			"Online Only",
			models.GlobalFlags{
				MediaFilterFlags: models.MediaFilterFlags{OnlineMediaOnly: true},
				QueryFlags:       models.QueryFlags{Limit: 10},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND path LIKE 'http%' LIMIT 10",
		},
		{
			"Custom Where",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Where: []string{"play_count > 5"}},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND play_count > 5 LIMIT 10",
		},
		{
			"Partial Skip",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Partial: "s"},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND COALESCE(time_first_played, 0) = 0 LIMIT 10",
		},
		{
			"Duration filter",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Duration: []string{">1h"}},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND duration >= ? LIMIT 10",
		},
		{
			"Size filter",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Size: []string{"<100MB"}},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND size <= ? LIMIT 10",
		},
		{
			"Modified after",
			models.GlobalFlags{
				TimeFilterFlags: models.TimeFilterFlags{ModifiedAfter: "2024-01-01"},
				QueryFlags:      models.QueryFlags{Limit: 10},
				DeletedFlags:    models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND time_modified >= ? LIMIT 10",
		},
		{
			"Created before",
			models.GlobalFlags{
				TimeFilterFlags: models.TimeFilterFlags{CreatedBefore: "2024-01-01"},
				QueryFlags:      models.QueryFlags{Limit: 10},
				DeletedFlags:    models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND time_created <= ? LIMIT 10",
		},
		{
			"Audio Only",
			models.GlobalFlags{
				MediaFilterFlags: models.MediaFilterFlags{AudioOnly: true},
				QueryFlags:       models.QueryFlags{Limit: 10},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND (media_type = 'audio' OR media_type = 'audiobook') LIMIT 10",
		},
		{
			"Text Only",
			models.GlobalFlags{
				MediaFilterFlags: models.MediaFilterFlags{TextOnly: true},
				QueryFlags:       models.QueryFlags{Limit: 10},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND (media_type = 'text') LIMIT 10",
		},
		{
			"Image Only",
			models.GlobalFlags{
				MediaFilterFlags: models.MediaFilterFlags{ImageOnly: true},
				QueryFlags:       models.QueryFlags{Limit: 10},
				DeletedFlags:     models.DeletedFlags{HideDeleted: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND (media_type = 'image') LIMIT 10",
		},
		{
			"Path-like Search (absolute)",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Search: []string{filepath.FromSlash("/home/")}},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
				FTSFlags:     models.FTSFlags{NoFTS: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND path LIKE ? LIMIT 10",
		},
		{
			"Path-like Search (relative)",
			models.GlobalFlags{
				FilterFlags:  models.FilterFlags{Search: []string{filepath.FromSlash("./home/")}},
				QueryFlags:   models.QueryFlags{Limit: 10},
				DeletedFlags: models.DeletedFlags{HideDeleted: true},
				FTSFlags:     models.FTSFlags{NoFTS: true},
			},
			"SELECT * FROM media WHERE COALESCE(time_deleted, 0) = 0 AND path LIKE ? LIMIT 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := query.NewFilterBuilder(tt.flags)
			query, _ := fb.BuildQuery("*")
			if query != tt.expected {
				t.Errorf("Build() query = %q, want %q", query, tt.expected)
			}
		})
	}
}

func TestFilterMedia(t *testing.T) {
	var size100 int64 = 100
	var size200 int64 = 200
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "test1.mp4", Size: &size100}, DB: "db1"},
		{Media: models.Media{Path: "test2.mkv", Size: &size200}, DB: "db1"},
	}

	tests := []struct {
		name     string
		flags    models.GlobalFlags
		expected int
	}{
		{"No filters", models.GlobalFlags{}, 2},
		{
			"Include filter",
			models.GlobalFlags{PathFilterFlags: models.PathFilterFlags{Include: []string{"test1.mp4"}}},
			1,
		},
		{
			"Exclude filter",
			models.GlobalFlags{PathFilterFlags: models.PathFilterFlags{Exclude: []string{"test1.mp4"}}},
			1,
		},
		{"Size filter", models.GlobalFlags{FilterFlags: models.FilterFlags{Size: []string{">150B"}}}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := query.FilterMedia(media, tt.flags)
			if len(got) != tt.expected {
				t.Errorf("FilterMedia() len = %v, want %v", len(got), tt.expected)
			}
		})
	}
}

func TestSortMedia(t *testing.T) {
	var size100 int64 = 100
	var size200 int64 = 200
	var dur10 int64 = 10
	var dur20 int64 = 20
	pc5 := int64(5)
	pc10 := int64(10)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "b.mp4", Size: &size200, Duration: &dur10, PlayCount: &pc5}},
		{Media: models.Media{Path: "a.mp4", Size: &size100, Duration: &dur20, PlayCount: &pc10}},
	}

	query.SortMedia(media, models.GlobalFlags{
		SortFlags: models.SortFlags{SortBy: "path"},
	})
	if media[0].Path != "a.mp4" {
		t.Errorf("SortMedia by path failed, got %s", media[0].Path)
	}

	query.SortMedia(media, models.GlobalFlags{
		SortFlags: models.SortFlags{SortBy: "size"},
	})
	if *media[0].Size != 100 {
		t.Errorf("SortMedia by size failed, got %d", *media[0].Size)
	}

	query.SortMedia(media, models.GlobalFlags{
		SortFlags: models.SortFlags{SortBy: "duration"},
	})
	if *media[0].Duration != 10 {
		t.Errorf("SortMedia by duration failed, got %d", *media[0].Duration)
	}

	query.SortMedia(media, models.GlobalFlags{
		SortFlags: models.SortFlags{SortBy: "play_count"},
	})
	if *media[0].PlayCount != 5 {
		t.Errorf("SortMedia by play_count failed, got %d", *media[0].PlayCount)
	}

	query.SortMedia(media, models.GlobalFlags{
		SortFlags: models.SortFlags{SortBy: "path", Reverse: true},
	})
	if media[0].Path != "b.mp4" {
		t.Errorf("SortMedia by path reverse failed, got %s", media[0].Path)
	}

	// Test that SortBy is respected even when PlayInOrder is set to default
	media = []models.MediaWithDB{
		{Media: models.Media{Path: "a.mp4", Size: &size200}},
		{Media: models.Media{Path: "b.mp4", Size: &size100}},
	}
	query.SortMedia(media, models.GlobalFlags{
		SortFlags:     models.SortFlags{SortBy: "size"},
		PlaybackFlags: models.PlaybackFlags{PlayInOrder: "natural_ps"},
	})
	if *media[0].Size != 100 {
		t.Errorf("SortMedia by size with natural_ps failed, got %d", *media[0].Size)
	}
}

func TestSortMediaAdvanced(t *testing.T) {
	media := []models.MediaWithDB{
		{Media: models.Media{Path: filepath.FromSlash("dir2/file.mp4")}},
		{Media: models.Media{Path: filepath.FromSlash("dir1/file.mp4")}},
	}

	query.NewSortBuilder(models.GlobalFlags{}).SortAdvanced(media, "natural_parent")
	if !strings.Contains(media[0].Path, "dir1") {
		t.Errorf("SortAdvanced by natural_parent failed, got %s", media[0].Path)
	}
}

func TestSortFolders(t *testing.T) {
	folders := []models.FolderStats{
		{Path: "b", Count: 2},
		{Path: "a", Count: 1},
	}

	query.SortFolders(folders, "path", false)
	if folders[0].Path != "a" {
		t.Errorf("SortFolders by path failed, got %s", folders[0].Path)
	}

	query.SortFolders(folders, "count", true)
	if folders[0].Count != 2 {
		t.Errorf("SortFolders by count desc failed, got %d", folders[0].Count)
	}
}

func TestQueryDatabase(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "query-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := testutils.InitTestDBNoFTS(dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}

	insert := `INSERT INTO media (path, title, duration, size, media_type) VALUES (?, ?, ?, ?, ?)`
	dbConn.Exec(insert, filepath.FromSlash("/test/movie.mp4"), "Test Movie", 7200, 1000000, "video")
	dbConn.Close()

	ctx := context.Background()
	results, err := query.QueryDatabase(ctx, dbPath, "SELECT path, title, duration, size, media_type FROM media", nil)
	if err != nil {
		t.Fatalf("query.QueryDatabase failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Path != filepath.FromSlash("/test/movie.mp4") {
		t.Errorf("Expected path %s, got %s", filepath.FromSlash("/test/movie.mp4"), results[0].Path)
	}
}

func TestSummarizeMedia(t *testing.T) {
	var size100 int64 = 100
	var size200 int64 = 200
	var size300 int64 = 300
	var dur10 int64 = 10
	var dur20 int64 = 20
	var dur30 int64 = 30

	media := []models.MediaWithDB{
		{Media: models.Media{Size: &size100, Duration: &dur10}},
		{Media: models.Media{Size: &size200, Duration: &dur20}},
		{Media: models.Media{Size: &size300, Duration: &dur30}},
	}

	got := query.SummarizeMedia(media)
	if len(got) != 2 {
		t.Fatalf("SummarizeMedia() returned %d items, want 2", len(got))
	}

	total := got[0]
	if total.Label != "Total" || total.Count != 3 || total.TotalSize != 600 || total.TotalDuration != 60 {
		t.Errorf("Total stats incorrect: %+v", total)
	}

	median := got[1]
	if median.Label != "Median" || median.TotalSize != 200 || median.TotalDuration != 20 {
		t.Errorf("Median stats incorrect: %+v", median)
	}
}

func TestMediaQuery(t *testing.T) {
	f1, _ := os.CreateTemp(t.TempDir(), "query-test1-*.db")
	dbPath1 := f1.Name()
	f1.Close()
	defer os.Remove(dbPath1)

	f2, _ := os.CreateTemp(t.TempDir(), "query-test2-*.db")
	dbPath2 := f2.Name()
	f2.Close()
	defer os.Remove(dbPath2)

	schema := `
	CREATE TABLE media (path TEXT PRIMARY KEY, time_deleted INTEGER DEFAULT 0, size INTEGER, duration INTEGER);
	CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, media_path TEXT NOT NULL, time_played INTEGER, playhead INTEGER, done INTEGER);
	CREATE TABLE captions (media_path TEXT NOT NULL, time REAL, text TEXT);
	`
	for _, dbPath := range []string{dbPath1, dbPath2} {
		dbConn, _ := sql.Open("sqlite3", dbPath)
		dbConn.Exec(schema)
		dbConn.Exec("INSERT INTO media (path) VALUES (?)", dbPath)
		dbConn.Close()
	}

	ctx := context.Background()
	flags := models.GlobalFlags{
		QueryFlags: models.QueryFlags{Limit: 10},
		SortFlags:  models.SortFlags{SortBy: "path"},
	}
	results, err := query.MediaQuery(ctx, []string{dbPath1, dbPath2}, flags)
	if err != nil {
		t.Fatalf("query.MediaQuery failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}

func TestReRankMedia(t *testing.T) {
	size100 := int64(100)
	size200 := int64(200)
	dur10 := int64(10)
	dur20 := int64(20)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "a", Size: &size200, Duration: &dur10}},
		{Media: models.Media{Path: "b", Size: &size100, Duration: &dur20}},
	}

	flags := models.GlobalFlags{SortFlags: models.SortFlags{ReRank: "-size=1 duration=1"}}
	got := query.ReRankMedia(media, flags)
	if len(got) != 2 {
		t.Errorf("Expected 2 results, got %d", len(got))
	}
}

func TestSortHistory(t *testing.T) {
	t1 := int64(1000)
	t2 := int64(2000)
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "a", TimeLastPlayed: &t1}},
		{Media: models.Media{Path: "b", TimeLastPlayed: &t2}},
	}

	query.SortHistory(media, "p", false)
	if len(media) != 2 {
		t.Errorf("Expected 2 results, got %d", len(media))
	}
}

func TestRegexSortMedia(t *testing.T) {
	media := []models.MediaWithDB{
		{Media: models.Media{Path: "movie_part2.mp4"}},
		{Media: models.Media{Path: "movie_part1.mp4"}},
	}

	got := query.RegexSortMedia(media, models.GlobalFlags{TextFlags: models.TextFlags{RegexSort: true}})
	if len(got) != 2 {
		t.Errorf("Expected 2 results, got %d", len(got))
	}
	if !strings.Contains(got[0].Path, "part1") {
		t.Errorf("Expected part1 first, got %s", got[0].Path)
	}
}

func TestHistoricalUsage(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "hist-usage-test-*.db")
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	dbConn, _ := sql.Open("sqlite3", dbPath)
	dbConn.Exec(
		"CREATE TABLE media (path TEXT PRIMARY KEY, time_deleted INTEGER DEFAULT 0, size INTEGER, duration INTEGER, time_last_played INTEGER)",
	)
	dbConn.Exec(
		"INSERT INTO media (path, size, duration, time_last_played) VALUES ('a', 100, 10, 1704067200)",
	) // 2024-01-01
	dbConn.Close()

	stats, err := query.HistoricalUsage(context.Background(), dbPath, "monthly", "time_last_played")
	if err != nil {
		t.Fatalf("HistoricalUsage failed: %v", err)
	}
	if len(stats) == 0 {
		t.Error("Expected stats, got none")
	}
}

func TestOverrideSort(t *testing.T) {
	fb := query.NewFilterBuilder(models.GlobalFlags{})
	got := fb.OverrideSort("month_created")
	if !strings.Contains(got, "strftime") {
		t.Errorf("OverrideSort failed: %s", got)
	}
}
