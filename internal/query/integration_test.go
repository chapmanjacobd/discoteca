package query_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func newPhase3DB(t *testing.T, name string, withFTS bool) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), name+"-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if withFTS {
		err = testutils.InitTestDB(t, conn)
	} else {
		err = testutils.InitTestDBNoFTS(conn)
	}
	if err != nil {
		t.Fatal(err)
	}
	return dbPath
}

func insertPhase3Rows(t *testing.T, dbPath, stmt string, rows ...[]any) {
	t.Helper()
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, row := range rows {
		if _, err := conn.Exec(stmt, row...); err != nil {
			t.Fatalf("insert %v: %v", row, err)
		}
	}
}

func TestScanMediaPopulatesEveryColumn(t *testing.T) {
	dbPath := newPhase3DB(t, "scan-all", false)
	title := "Everything"
	row := []any{
		"/media/all.mkv", "media all mkv", title, 3600, 4096,
		100, 200, 0, 10, 20, 7, 120,
		"video", 1920, 1080, 23.976,
		"h264", "aac", "srt", 2, 1, 3,
		"Album", "Artist", "Genre", ";cat;", "Description", "eng", 300,
		8.5, "fast-hash", "sha-hash", 1,
	}
	insertPhase3Rows(t, dbPath,
		`INSERT INTO media (path, path_tokenized, title, duration, size, time_created,
		 time_modified, time_deleted, time_first_played, time_last_played, play_count, playhead,
		 media_type, width, height, fps, video_codecs, audio_codecs, subtitle_codecs,
		 video_count, audio_count, subtitle_count, album, artist, genre, categories,
		 description, language, time_downloaded, score, fasthash, sha256, is_deduped)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		row)

	// Exercise every column, including ones no production query selects by name.
	got, err := query.QueryDatabase(
		context.Background(), dbPath, "SELECT * FROM media", nil, //nolint:unqueryvet // exercise every column
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows", len(got))
	}
	m := got[0]
	checks := map[string]any{
		"Path":            m.Path,
		"PathTokenized":   ptrValue(m.PathTokenized),
		"Title":           ptrValue(m.Title),
		"Duration":        ptrValue(m.Duration),
		"Size":            ptrValue(m.Size),
		"TimeCreated":     ptrValue(m.TimeCreated),
		"TimeModified":    ptrValue(m.TimeModified),
		"TimeDeleted":     ptrValue(m.TimeDeleted),
		"TimeFirstPlayed": ptrValue(m.TimeFirstPlayed),
		"TimeLastPlayed":  ptrValue(m.TimeLastPlayed),
		"PlayCount":       ptrValue(m.PlayCount),
		"Playhead":        ptrValue(m.Playhead),
		"MediaType":       ptrValue(m.MediaType),
		"Width":           ptrValue(m.Width),
		"Height":          ptrValue(m.Height),
		"Fps":             ptrValue(m.Fps),
		"VideoCodecs":     ptrValue(m.VideoCodecs),
		"AudioCodecs":     ptrValue(m.AudioCodecs),
		"SubtitleCodecs":  ptrValue(m.SubtitleCodecs),
		"VideoCount":      ptrValue(m.VideoCount),
		"AudioCount":      ptrValue(m.AudioCount),
		"SubtitleCount":   ptrValue(m.SubtitleCount),
		"Album":           ptrValue(m.Album),
		"Artist":          ptrValue(m.Artist),
		"Genre":           ptrValue(m.Genre),
		"Categories":      ptrValue(m.Categories),
		"Description":     ptrValue(m.Description),
		"Language":        ptrValue(m.Language),
		"TimeDownloaded":  ptrValue(m.TimeDownloaded),
		"Score":           ptrValue(m.Score),
		"Fasthash":        ptrValue(m.Fasthash),
		"Sha256":          ptrValue(m.Sha256),
		"IsDeduped":       ptrValue(m.IsDeduped),
	}
	for name, value := range checks {
		if value == nil {
			t.Errorf("%s was not populated by the scanner", name)
		}
	}
}

func ptrValue[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestDefaultSortUsesRealCountsAndMatchesAcrossDBs(t *testing.T) {
	single := newPhase3DB(t, "sort-single", false)
	insertPhase3Rows(t, single,
		`INSERT INTO media (path, video_count, audio_count, subtitle_count) VALUES (?,?,?,?)`,
		[]any{"/x/a.mkv", 0, 3, 0},
		[]any{"/x/b.mkv", 2, 0, 0},
		[]any{"/x/c.mkv", 1, 5, 1},
	)

	ctx := context.Background()
	flags := models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true},
		SortFlags:  models.SortFlags{SortBy: "default"},
	}
	got, err := query.MediaQuery(ctx, []string{single}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/b.mkv", "/x/c.mkv", "/x/a.mkv"})

	flags.SortBy = "video_count desc,audio_count desc,subtitle_count desc"
	got, err = query.MediaQuery(ctx, []string{single}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/b.mkv", "/x/c.mkv", "/x/a.mkv"})
}

func TestOneDBAndTwoDBReturnSameOrder(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	first := filepath.Join(dir, "first.db")
	second := filepath.Join(dir, "second.db")
	combined := filepath.Join(dir, "combined.db")
	for _, p := range []string{first, second, combined} {
		conn, err := sql.Open("sqlite3", p)
		if err != nil {
			t.Fatal(err)
		}
		if err := testutils.InitTestDBNoFTS(conn); err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}
	insertPhase3Rows(t, first,
		`INSERT INTO media (path) VALUES (?)`,
		[]any{"/x/a"}, []any{"/x/c"}, []any{"/x/e"})
	insertPhase3Rows(t, second,
		`INSERT INTO media (path) VALUES (?)`,
		[]any{"/x/b"}, []any{"/x/d"}, []any{"/x/f"})
	for _, p := range []string{"/x/a", "/x/b", "/x/c", "/x/d", "/x/e", "/x/f"} {
		insertPhase3Rows(t, combined, `INSERT INTO media (path) VALUES (?)`, []any{p})
	}

	flags := models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true},
		SortFlags:  models.SortFlags{SortBy: "default"},
	}
	one, err := query.MediaQuery(ctx, []string{combined}, flags)
	if err != nil {
		t.Fatal(err)
	}
	two, err := query.MediaQuery(ctx, []string{first, second}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, one, []string{"/x/a", "/x/b", "/x/c", "/x/d", "/x/e", "/x/f"})
	assertPaths(t, two, []string{"/x/a", "/x/b", "/x/c", "/x/d", "/x/e", "/x/f"})
}

func TestPaginationAndReverseUseFinalOrder(t *testing.T) {
	dbPath := newPhase3DB(t, "page", false)
	for i, count := range []int{1, 4, 2, 5, 3} {
		path := "/x/" + string(rune('a'+i))
		insertPhase3Rows(t, dbPath,
			`INSERT INTO media (path, play_count) VALUES (?,?)`, []any{path, count})
	}
	ctx := context.Background()

	flags := models.GlobalFlags{
		QueryFlags: models.QueryFlags{Limit: 2},
		SortFlags:  models.SortFlags{SortBy: "play_count desc"},
	}
	got, err := query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/d", "/x/b"})

	flags = models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true, Offset: 3},
		SortFlags:  models.SortFlags{SortBy: "path"},
	}
	got, err = query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/d", "/x/e"})

	flags = models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true},
		SortFlags:  models.SortFlags{SortBy: "path", Reverse: true},
	}
	got, err = query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/e", "/x/d", "/x/c", "/x/b", "/x/a"})
}

func TestBuildQueryOffsetWithoutLimit(t *testing.T) {
	fb := query.NewFilterBuilder(models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true, Offset: 5},
		SortFlags:  models.SortFlags{SortBy: "path"},
	})
	got, _ := fb.BuildQuery(context.Background(), "*")
	if !strings.Contains(got, "LIMIT -1 OFFSET 5") {
		t.Errorf("expected LIMIT -1 OFFSET, got %q", got)
	}
}

func TestPercentileValidation(t *testing.T) {
	dbPath := newPhase3DB(t, "percentile", false)
	insertPhase3Rows(t, dbPath, `INSERT INTO media (path, size) VALUES (?,?)`, []any{"/x/a", 100})
	ctx := context.Background()
	dbs := []string{dbPath}

	for _, bad := range []string{"p200-300", "p100-200", "p50-10", "pfoo-bar"} {
		flags := models.GlobalFlags{FilterFlags: models.FilterFlags{Size: []string{bad}}}
		if _, err := query.ResolvePercentileFlags(ctx, dbs, flags); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}

	flags := models.GlobalFlags{FilterFlags: models.FilterFlags{Size: []string{"p0-100"}}}
	resolved, err := query.ResolvePercentileFlags(ctx, dbs, flags)
	if err != nil {
		t.Fatalf("p0-100 should be valid: %v", err)
	}
	if len(resolved.Size) == 0 || strings.HasPrefix(resolved.Size[0], "p") {
		t.Errorf("percentile was not resolved: %v", resolved.Size)
	}

	missing := filepath.Join(t.TempDir(), "does-not-exist.db")
	if _, err := query.ResolvePercentileFlags(ctx, []string{missing}, flags); err == nil {
		t.Error("expected an error when the percentile database cannot be queried")
	}
}

func TestPlayCountMaxAcceptsZeroAndUnfinishedMeansStartedNotFinished(t *testing.T) {
	dbPath := newPhase3DB(t, "status", false)
	insertPhase3Rows(t, dbPath,
		`INSERT INTO media (path, play_count, playhead, duration) VALUES (?,?,?,?)`,
		[]any{"/x/never", 0, 50, 100},
		[]any{"/x/done", 3, 100, 100},
		[]any{"/x/unplayed", 0, 0, 100},
	)
	ctx := context.Background()

	flags := models.GlobalFlags{
		QueryFlags:  models.QueryFlags{All: true},
		SortFlags:   models.SortFlags{SortBy: "path"},
		FilterFlags: models.FilterFlags{PlayCountMax: new(0)},
	}
	got, err := query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/never", "/x/unplayed"})

	flags.FilterFlags = models.FilterFlags{Unfinished: true}
	got, err = query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/never"})
}

func TestIncludeIsAPathPattern(t *testing.T) {
	dbPath := newPhase3DB(t, "include", false)
	insertPhase3Rows(t, dbPath,
		`INSERT INTO media (path) VALUES (?)`,
		[]any{"/media/movies/a.mp4"}, []any{"/media/tv/b.mp4"})
	ctx := context.Background()

	flags := models.GlobalFlags{
		QueryFlags:      models.QueryFlags{All: true},
		SortFlags:       models.SortFlags{SortBy: "path"},
		PathFilterFlags: models.PathFilterFlags{Include: []string{"*movies*"}},
	}
	// Ensure the include term is not treated as free-text search.
	got, err := query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/media/movies/a.mp4"})
}

func TestGroupByParentPreservesDatabase(t *testing.T) {
	ctx := context.Background()
	db1 := newPhase3DB(t, "group1", false)
	db2 := newPhase3DB(t, "group2", false)
	insertPhase3Rows(t, db1, `INSERT INTO media (path) VALUES (?)`, []any{"/one/show/ep1.mkv"})
	insertPhase3Rows(t, db2, `INSERT INTO media (path) VALUES (?)`, []any{"/two/show/ep1.mkv"})

	flags := models.GlobalFlags{
		QueryFlags:     models.QueryFlags{All: true},
		SortFlags:      models.SortFlags{SortBy: "path"},
		AggregateFlags: models.AggregateFlags{GroupByParent: true},
	}
	got, err := query.MediaQuery(ctx, []string{db1, db2}, flags)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d groups", len(got))
	}
	seen := map[string]bool{}
	for _, m := range got {
		seen[m.DB] = true
	}
	if !seen[db1] || !seen[db2] {
		t.Fatalf("grouped rows lost their source DB: %+v", got)
	}
}

func TestHistoricalUsageFacetsAndQuarters(t *testing.T) {
	dbPath := newPhase3DB(t, "history", false)
	// 2024-02-15 and 2024-04-15, both deleted. Quarterly must be Q1 and Q2.
	insertPhase3Rows(t, dbPath,
		`INSERT INTO media (path, time_deleted, size, duration) VALUES (?,?,?,?)`,
		[]any{"/x/q1", int64(1707955200), 10, 1},
		[]any{"/x/q2", int64(1713139200), 20, 2},
	)
	ctx := context.Background()

	stats, err := query.HistoricalUsageFiltered(ctx, dbPath, "quarterly", "time_deleted", models.GlobalFlags{})
	if err != nil {
		t.Fatal(err)
	}
	labels := make([]string, 0, len(stats))
	for _, s := range stats {
		labels = append(labels, s.Label)
	}
	if !reflect.DeepEqual(labels, []string{"2024-Q2", "2024-Q1"}) {
		t.Errorf("quarter labels = %v", labels)
	}

	if _, err := query.HistoricalUsageFiltered(
		ctx,
		dbPath,
		"quarterly",
		"not_a_time",
		models.GlobalFlags{},
	); err == nil {
		t.Error("expected unknown facet to be rejected")
	}
}

func TestDatabasesHaveFTSRequiresEveryDatabase(t *testing.T) {
	ctx := context.Background()
	withFTS := newPhase3DB(t, "fts-yes", true)
	withoutFTS := newPhase3DB(t, "fts-no", false)

	if !query.DatabasesHaveFTS(ctx, []string{withFTS}) {
		t.Error("expected FTS database to be detected")
	}
	if query.DatabasesHaveFTS(ctx, []string{withFTS, withoutFTS}) {
		t.Error("mixed database list must not enable FTS for all")
	}
}

func assertPaths(t *testing.T, media []models.MediaWithDB, want []string) {
	t.Helper()
	got := make([]string, len(media))
	for i, m := range media {
		got[i] = m.Path
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestHasAnyFilterCoversEveryPredicate(t *testing.T) {
	base := models.GlobalFlags{DeletedFlags: models.DeletedFlags{HideDeleted: false}}
	tests := []struct {
		name  string
		flags models.GlobalFlags
	}{
		{"only deleted", models.GlobalFlags{DeletedFlags: models.DeletedFlags{OnlyDeleted: true}}},
		{"hide deleted", models.GlobalFlags{DeletedFlags: models.DeletedFlags{HideDeleted: true}}},
		{"video", models.GlobalFlags{MediaFilterFlags: models.MediaFilterFlags{VideoOnly: true}}},
		{"genre", models.GlobalFlags{MediaFilterFlags: models.MediaFilterFlags{Genre: "g"}}},
		{"language", models.GlobalFlags{MediaFilterFlags: models.MediaFilterFlags{Language: []string{"en"}}}},
		{"paths", models.GlobalFlags{PathFilterFlags: models.PathFilterFlags{Paths: []string{"/a"}}}},
		{"play count", models.GlobalFlags{FilterFlags: models.FilterFlags{PlayCountMax: new(0)}}},
		{"size", models.GlobalFlags{FilterFlags: models.FilterFlags{Size: []string{">1MB"}}}},
		{"deleted after", models.GlobalFlags{TimeFilterFlags: models.TimeFilterFlags{DeletedAfter: "2024-01-01"}}},
		{"played before", models.GlobalFlags{TimeFilterFlags: models.TimeFilterFlags{PlayedBefore: "2024-01-01"}}},
		{"watched", models.GlobalFlags{FilterFlags: models.FilterFlags{Watched: new(true)}}},
		{"unfinished", models.GlobalFlags{FilterFlags: models.FilterFlags{Unfinished: true}}},
		{"partial", models.GlobalFlags{FilterFlags: models.FilterFlags{Partial: "p"}}},
		{"completed", models.GlobalFlags{FilterFlags: models.FilterFlags{Completed: true}}},
		{"online", models.GlobalFlags{MediaFilterFlags: models.MediaFilterFlags{OnlineMediaOnly: true}}},
		{"ext", models.GlobalFlags{MediaFilterFlags: models.MediaFilterFlags{Ext: []string{".mp4"}}}},
		{"category", models.GlobalFlags{MediaFilterFlags: models.MediaFilterFlags{Category: []string{"c"}}}},
		{"include", models.GlobalFlags{PathFilterFlags: models.PathFilterFlags{Include: []string{"%x%"}}}},
		{"exclude", models.GlobalFlags{PathFilterFlags: models.PathFilterFlags{Exclude: []string{"%x%"}}}},
		{"path contains", models.GlobalFlags{PathFilterFlags: models.PathFilterFlags{PathContains: []string{"x"}}}},
		{"regex", models.GlobalFlags{PathFilterFlags: models.PathFilterFlags{Regex: "x.*"}}},
		{"portrait", models.GlobalFlags{MediaFilterFlags: models.MediaFilterFlags{Portrait: true}}},
		{"captions", models.GlobalFlags{FilterFlags: models.FilterFlags{WithCaptions: true}}},
		{"where", models.GlobalFlags{FilterFlags: models.FilterFlags{Where: []string{"1=1"}}}},
		{"duration from size", models.GlobalFlags{FilterFlags: models.FilterFlags{DurationFromSize: ">1MB"}}},
		{"file counts", models.GlobalFlags{AggregateFlags: models.AggregateFlags{FileCounts: ">1"}}},
		{"folder sizes", models.GlobalFlags{AggregateFlags: models.AggregateFlags{FolderSizes: []string{">1MB"}}}},
	}
	if query.HasAnyFilter(base) {
		t.Fatal("empty flags must not report a filter")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !query.HasAnyFilter(tt.flags) {
				t.Errorf("HasAnyFilter(%s) = false", tt.name)
			}
		})
	}
}

func TestUnknownSortFallsBackToDefaultNotPath(t *testing.T) {
	dbPath := newPhase3DB(t, "unknown-sort", false)
	insertPhase3Rows(t, dbPath,
		`INSERT INTO media (path, video_count) VALUES (?,?)`,
		[]any{"/x/a.mkv", 0},
		[]any{"/x/b.mkv", 2},
	)
	ctx := context.Background()
	flags := models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true},
		SortFlags:  models.SortFlags{SortBy: "not_a_real_column"},
	}
	got, err := query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	// Default xklb puts the row with video_count first; a silent path sort
	// would put /x/a.mkv first.
	assertPaths(t, got, []string{"/x/b.mkv", "/x/a.mkv"})
}

func TestNatSortAndReverseTies(t *testing.T) {
	dbPath := newPhase3DB(t, "natsort", false)
	insertPhase3Rows(t, dbPath, `INSERT INTO media (path) VALUES (?)`,
		[]any{"/x/episode10.mkv"}, []any{"/x/episode2.mkv"})
	ctx := context.Background()

	flags := models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true},
		SortFlags:  models.SortFlags{SortBy: "path"},
	}
	got, err := query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/episode10.mkv", "/x/episode2.mkv"})

	flags.SortFlags.NatSort = true
	got, err = query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/episode2.mkv", "/x/episode10.mkv"})

	// Ties keep their stable order when reversed.
	flags = models.GlobalFlags{
		QueryFlags: models.QueryFlags{All: true},
		SortFlags:  models.SortFlags{SortBy: "size", Reverse: true},
	}
	insertPhase3Rows(t, dbPath, `INSERT INTO media (path, size) VALUES (?,?)`,
		[]any{"/x/tie1.mkv", 100}, []any{"/x/tie2.mkv", 100})
	got, err = query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, got, []string{"/x/tie1.mkv", "/x/tie2.mkv", "/x/episode10.mkv", "/x/episode2.mkv"})
}

func TestMediaQueryCountMatchesPostProcessing(t *testing.T) {
	dbPath := newPhase3DB(t, "counts", false)
	insertPhase3Rows(t, dbPath, `INSERT INTO media (path) VALUES (?)`,
		[]any{"/show/one/e1.mkv"}, []any{"/show/one/e2.mkv"},
		[]any{"/show/two/e1.mkv"}, []any{"/show/two/e2.mkv"}, []any{"/show/two/e3.mkv"})
	ctx := context.Background()

	groups := models.GlobalFlags{
		QueryFlags:     models.QueryFlags{All: true},
		SortFlags:      models.SortFlags{SortBy: "path"},
		AggregateFlags: models.AggregateFlags{GroupByParent: true},
	}
	media, err := query.MediaQuery(ctx, []string{dbPath}, groups)
	if err != nil {
		t.Fatal(err)
	}
	count, err := query.MediaQueryCount(ctx, []string{dbPath}, groups)
	if err != nil {
		t.Fatal(err)
	}
	if int(count) != len(media) {
		t.Errorf("grouped count = %d, response has %d", count, len(media))
	}

	siblings := models.GlobalFlags{
		QueryFlags:  models.QueryFlags{All: true},
		SortFlags:   models.SortFlags{SortBy: "path"},
		FilterFlags: models.FilterFlags{FetchSiblings: "each"},
	}
	media, err = query.MediaQuery(ctx, []string{dbPath}, siblings)
	if err != nil {
		t.Fatal(err)
	}
	count, err = query.MediaQueryCount(ctx, []string{dbPath}, siblings)
	if err != nil {
		t.Fatal(err)
	}
	if int(count) != len(media) {
		t.Errorf("sibling count = %d, response has %d", count, len(media))
	}

	// With a limit, expansion applies to the selected page.
	siblings.QueryFlags = models.QueryFlags{Limit: 1}
	media, err = query.MediaQuery(ctx, []string{dbPath}, siblings)
	if err != nil {
		t.Fatal(err)
	}
	if len(media) != 2 {
		t.Errorf("expected 2 siblings for the first page, got %d", len(media))
	}

	included := models.GlobalFlags{
		QueryFlags:      models.QueryFlags{All: true},
		SortFlags:       models.SortFlags{SortBy: "path"},
		PathFilterFlags: models.PathFilterFlags{Include: []string{"%one%"}},
	}
	media, err = query.MediaQuery(ctx, []string{dbPath}, included)
	if err != nil {
		t.Fatal(err)
	}
	count, err = query.MediaQueryCount(ctx, []string{dbPath}, included)
	if err != nil {
		t.Fatal(err)
	}
	if int(count) != len(media) || int(count) != 2 {
		t.Errorf("include count = %d, response has %d (want 2)", count, len(media))
	}
}

func TestFetchSiblingsHonorsMaxSortAndDeletedFilter(t *testing.T) {
	dbPath := newPhase3DB(t, "siblings", false)
	insertPhase3Rows(t, dbPath, `INSERT INTO media (path, time_deleted) VALUES (?,?)`,
		[]any{"/show/e1.mkv", 0}, []any{"/show/e2.mkv", 0},
		[]any{"/show/e3.mkv", 0}, []any{"/show/e4-deleted.mkv", 100})
	ctx := context.Background()

	flags := models.GlobalFlags{
		QueryFlags:   models.QueryFlags{Limit: 1},
		SortFlags:    models.SortFlags{SortBy: "path"},
		FilterFlags:  models.FilterFlags{FetchSiblings: "each", FetchSiblingsMax: 2},
		DeletedFlags: models.DeletedFlags{HideDeleted: true},
	}
	got, err := query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("FetchSiblingsMax=2 returned %d rows: %v", len(got), got)
	}
	for _, m := range got {
		if strings.Contains(m.Path, "deleted") {
			t.Error("deleted sibling leaked into results")
		}
	}

	flags.HideDeleted = false
	got, err = query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected max limit to still apply with HideDeleted=false, got %d", len(got))
	}
}
