package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

func setupDB(t *testing.T) (*sql.DB, *db.Queries) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	schema := db.GetSchemaTables() + "\n" + db.GetSchemaTriggers() + "\n" + db.GetSchemaFTS()

	var version string
	if err2 := sqlDB.QueryRow("SELECT sqlite_version()").Scan(&version); err2 != nil {
		t.Fatal(err2)
	}
	var v1, v2, v3 int
	fmt.Sscanf(version, "%d.%d.%d", &v1, &v2, &v3)
	hasStrict := v1 > 3 || (v1 == 3 && v2 >= 37)

	if !hasStrict {
		schema = strings.ReplaceAll(schema, "STRICT", "")
		if v1 < 3 || (v1 == 3 && v2 < 38) {
			schema = strings.ReplaceAll(schema, "unixepoch()", "strftime('%s', 'now')")
		}
	}

	var hasFTS5 bool
	err = sqlDB.QueryRow("SELECT 1 FROM pragma_compile_options WHERE compile_options = 'ENABLE_FTS5'").Scan(&hasFTS5)
	if err != nil {
		_, err = sqlDB.Exec("CREATE VIRTUAL TABLE fts_test USING fts5(t)")
		if err == nil {
			hasFTS5 = true
			sqlDB.Exec("DROP TABLE fts_test")
		}
	}

	if !hasFTS5 {
		var filteredSchema strings.Builder
		skipNextEnd := false
		for line := range strings.SplitSeq(schema, ";") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			upper := strings.ToUpper(trimmed)
			if strings.Contains(upper, "FTS5") || strings.Contains(upper, "_FTS") {
				if strings.Contains(upper, "BEGIN") && !strings.Contains(upper, "END") {
					skipNextEnd = true
				}
				continue
			}
			if skipNextEnd && upper == "END" {
				skipNextEnd = false
				continue
			}
			filteredSchema.WriteString(trimmed)
			filteredSchema.WriteString(";")
		}
		schema = filteredSchema.String()
	}

	if _, err := sqlDB.Exec(schema); err != nil {
		t.Fatalf("Failed to execute schema: %v", err)
	}

	return sqlDB, db.New(sqlDB)
}

type queriesTestEnv struct {
	sqlDB *sql.DB
	q     *db.Queries
}

func TestQueries(t *testing.T) {
	sqlDB, q := setupDB(t)
	defer sqlDB.Close()

	env := &queriesTestEnv{
		sqlDB: sqlDB,
		q:     q,
	}
	ctx := context.Background()

	t.Run("UpsertAndGet", func(t *testing.T) { env.testUpsertAndGet(ctx, t) })
	t.Run("CategoryStats", func(t *testing.T) { env.testCategoryStats(ctx, t) })
	t.Run("MediaFiltering", func(t *testing.T) { env.testMediaFiltering(ctx, t) })
	t.Run("HistoryAndStats", func(t *testing.T) { env.testHistoryAndStats(ctx, t) })
	t.Run("Playlists", func(t *testing.T) { env.testPlaylists(ctx, t) })
	t.Run("UpdateOperations", func(t *testing.T) { env.testUpdateOperations(ctx, t) })
	t.Run("FTSAndCaptions", func(t *testing.T) { env.testFTSAndCaptions(ctx, t) })
	t.Run("MiscQueries", func(t *testing.T) { env.testMiscQueries(ctx, t) })
	t.Run("WithTx", func(t *testing.T) { env.testWithTx(ctx, t) })
	t.Run("StrictEnforcement", func(t *testing.T) { env.testStrictEnforcement(ctx, t) })
}

func (e *queriesTestEnv) testUpsertAndGet(ctx context.Context, t *testing.T) {
	err := e.q.UpsertMedia(ctx, db.UpsertMediaParams{
		Path:  "test.mp4",
		Title: sql.NullString{String: "Test Title", Valid: true},
		Size:  sql.NullInt64{Int64: 1000, Valid: true},
	})
	if err != nil {
		t.Errorf("UpsertMedia failed: %v", err)
	}

	m, err := e.q.GetMediaByPathExact(ctx, "test.mp4")
	if err != nil {
		t.Errorf("GetMediaByPathExact failed: %v", err)
	}
	if m.Title.String != "Test Title" {
		t.Errorf("Expected Test Title, got %s", m.Title.String)
	}
}

func (e *queriesTestEnv) testCategoryStats(ctx context.Context, t *testing.T) {
	err := e.q.UpdateMediaCategories(ctx, db.UpdateMediaCategoriesParams{
		Path:       "test.mp4",
		Categories: sql.NullString{String: ";comedy;", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	stats, err := e.q.GetCategoryStats(ctx)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, s := range stats {
		if s.Category == "comedy" && s.Count == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Comedy category stat not found")
	}
}

func (e *queriesTestEnv) testMediaFiltering(ctx context.Context, t *testing.T) {
	e.q.UpsertMedia(ctx, db.UpsertMediaParams{
		Path:      "video.mp4",
		MediaType: sql.NullString{String: "video", Valid: true},
		Duration:  sql.NullInt64{Int64: 100, Valid: true},
		Size:      sql.NullInt64{Int64: 5000, Valid: true},
	})
	e.q.UpsertMedia(ctx, db.UpsertMediaParams{
		Path:      "audio.mp3",
		MediaType: sql.NullString{String: "audio", Valid: true},
		Duration:  sql.NullInt64{Int64: 200, Valid: true},
		Size:      sql.NullInt64{Int64: 2000, Valid: true},
	})

	res, _ := e.q.GetMediaByType(ctx, db.GetMediaByTypeParams{
		VideoOnly: true,
		AudioOnly: false,
		ImageOnly: false,
		Limit:     10,
	})
	if len(res) != 1 || res[0].Path != "video.mp4" {
		t.Errorf("GetMediaByType video failed, got %v", res)
	}

	res, _ = e.q.GetMediaBySize(ctx, db.GetMediaBySizeParams{
		MinSize: 3000,
		MaxSize: 6000,
		Limit:   10,
	})
	if len(res) != 1 || res[0].Path != "video.mp4" {
		t.Errorf("GetMediaBySize failed, got %v", res)
	}

	res, _ = e.q.GetMediaByDuration(ctx, db.GetMediaByDurationParams{
		MinDuration: 150,
		MaxDuration: 250,
		Limit:       10,
	})
	if len(res) != 1 || res[0].Path != "audio.mp3" {
		t.Errorf("GetMediaByDuration failed, got %v", res)
	}
}

func (e *queriesTestEnv) testHistoryAndStats(ctx context.Context, t *testing.T) {
	path := "history.mp4"
	e.q.UpsertMedia(ctx, db.UpsertMediaParams{
		Path:     path,
		Duration: sql.NullInt64{Int64: 1000, Valid: true},
	})

	e.q.UpdatePlayHistory(ctx, db.UpdatePlayHistoryParams{
		Path:            path,
		Playhead:        sql.NullInt64{Int64: 500, Valid: true},
		TimeLastPlayed:  sql.NullInt64{Int64: 12345678, Valid: true},
		TimeFirstPlayed: sql.NullInt64{Int64: 12345678, Valid: true},
	})

	e.q.InsertHistory(ctx, db.InsertHistoryParams{
		MediaPath: path,
		Playhead:  sql.NullInt64{Int64: 500, Valid: true},
	})

	count, _ := e.q.GetHistoryCount(ctx, path)
	if count != 1 {
		t.Errorf("Expected 1 history entry, got %d", count)
	}

	unfinished, _ := e.q.GetUnfinishedMedia(ctx, 10)
	if len(unfinished) != 1 || unfinished[0].Path != path {
		t.Errorf("Expected 1 unfinished media, got %v", unfinished)
	}

	stats, _ := e.q.GetStats(ctx)
	if stats.WatchedCount != 1 {
		t.Errorf("Expected 1 watched media in stats, got %d", stats.WatchedCount)
	}
}

func (e *queriesTestEnv) testPlaylists(ctx context.Context, t *testing.T) {
	id, err := e.q.InsertPlaylist(ctx, db.InsertPlaylistParams{
		Path:         sql.NullString{String: "http://example.com/playlist", Valid: true},
		ExtractorKey: sql.NullString{String: "youtube", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID for playlist")
	}

	playlists, _ := e.q.GetPlaylists(ctx)
	if len(playlists) != 1 || playlists[0].Path.String != "http://example.com/playlist" {
		t.Errorf("Expected 1 playlist, got %v", playlists)
	}
}

func (e *queriesTestEnv) testUpdateOperations(ctx context.Context, t *testing.T) {
	e.q.UpsertMedia(ctx, db.UpsertMediaParams{Path: "old.mp4"})
	e.q.UpdatePath(ctx, db.UpdatePathParams{NewPath: "new.mp4", OldPath: "old.mp4"})
	_, err := e.q.GetMediaByPathExact(ctx, "old.mp4")
	if err == nil {
		t.Error("old.mp4 should not exist")
	}
	_, err = e.q.GetMediaByPathExact(ctx, "new.mp4")
	if err != nil {
		t.Error("new.mp4 should exist")
	}

	e.q.MarkDeleted(ctx, db.MarkDeletedParams{Path: "new.mp4", TimeDeleted: sql.NullInt64{Int64: 1, Valid: true}})
	m, _ := e.q.GetMediaByPathExact(ctx, "new.mp4")
	if m.TimeDeleted.Int64 == 0 {
		t.Error("Expected time_deleted to be set")
	}
}

func (e *queriesTestEnv) testFTSAndCaptions(ctx context.Context, t *testing.T) {
	var exists int
	e.sqlDB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='media_fts'").Scan(&exists)
	if exists == 0 {
		t.Skip("FTS5 not available")
	}

	path := "fts_video.mp4"
	e.q.UpsertMedia(ctx, db.UpsertMediaParams{
		Path:  path,
		Title: sql.NullString{String: "Unique Title for FTS", Valid: true},
	})

	res, err := e.q.SearchMediaFTS(ctx, db.SearchMediaFTSParams{
		Query: "Uni",
		Limit: 10,
	})
	if err != nil {
		t.Errorf("SearchMediaFTS failed: %v", err)
	}
	if len(res) == 0 {
		t.Error("SearchMediaFTS returned no results")
	}
	db.RankSearchResults(res, "Unique")
	if res[0].Rank == 0 {
		t.Logf("Warning: Search rank is 0")
	} else {
		t.Logf("Search rank: %f", res[0].Rank)
	}

	err = e.q.InsertCaption(ctx, db.InsertCaptionParams{
		MediaPath: path,
		Time:      sql.NullFloat64{Float64: 10.5, Valid: true},
		Text:      sql.NullString{String: "Hello from captions", Valid: true},
	})
	if err != nil {
		t.Fatalf("InsertCaption failed: %v", err)
	}

	resCaptions, err := e.q.SearchCaptions(ctx, db.SearchCaptionsParams{
		Query:     "Hel",
		VideoOnly: false,
		AudioOnly: false,
		ImageOnly: false,
		TextOnly:  false,
		Limit:     10,
	})
	if err != nil {
		t.Errorf("SearchCaptions failed: %v", err)
	}
	if len(resCaptions) == 0 {
		t.Error("SearchCaptions returned no results")
	}
}

func (e *queriesTestEnv) testMiscQueries(ctx context.Context, t *testing.T) {
	e.q.UpsertMedia(ctx, db.UpsertMediaParams{
		Path:      "random.mp4",
		MediaType: sql.NullString{String: "video", Valid: true},
		Score:     sql.NullFloat64{Float64: 5.0, Valid: true},
	})

	random, _ := e.q.GetRandomMedia(ctx, 1)
	if len(random) == 0 {
		t.Error("GetRandomMedia failed")
	}

	ratings, _ := e.q.GetRatingStats(ctx)
	if len(ratings) == 0 {
		t.Error("GetRatingStats failed")
	}

	stats, _ := e.q.GetStatsByType(ctx)
	if len(stats) == 0 {
		t.Error("GetStatsByType failed")
	}

	meta, _ := e.q.GetAllMediaMetadata(ctx)
	if len(meta) == 0 {
		t.Error("GetAllMediaMetadata failed")
	}

	res, _ := e.q.GetMedia(ctx, 10)
	if len(res) == 0 {
		t.Error("GetMedia failed")
	}

	res, _ = e.q.GetMediaByPath(ctx, db.GetMediaByPathParams{PathPattern: "%random%", Limit: 10})
	if len(res) == 0 {
		t.Error("GetMediaByPath failed")
	}

	res, _ = e.q.GetMediaByPlayCount(ctx, db.GetMediaByPlayCountParams{MinPlayCount: 0, MaxPlayCount: 10, Limit: 10})
	if len(res) == 0 {
		t.Error("GetMediaByPlayCount failed")
	}

	res, _ = e.q.GetSiblingMedia(
		ctx,
		db.GetSiblingMediaParams{PathPattern: "%", PathExclude: "non-existent", Limit: 10},
	)
	if len(res) == 0 {
		t.Error("GetSiblingMedia failed")
	}

	res, _ = e.q.GetUnwatchedMedia(ctx, 10)
	if len(res) == 0 {
		t.Error("GetUnwatchedMedia failed")
	}

	e.q.UpdatePlayHistory(
		ctx,
		db.UpdatePlayHistoryParams{Path: "random.mp4", TimeLastPlayed: sql.NullInt64{Int64: 1, Valid: true}},
	)
	res, _ = e.q.GetWatchedMedia(ctx, 10)
	if len(res) == 0 {
		t.Error("GetWatchedMedia failed")
	}
}

func (e *queriesTestEnv) testWithTx(ctx context.Context, t *testing.T) {
	tx, _ := e.sqlDB.Begin()
	qtx := e.q.WithTx(tx)
	err := qtx.UpsertMedia(ctx, db.UpsertMediaParams{Path: "tx.mp4"})
	if err != nil {
		t.Errorf("WithTx failed: %v", err)
	}
	tx.Commit()

	_, err = e.q.GetMediaByPathExact(ctx, "tx.mp4")
	if err != nil {
		t.Error("tx.mp4 should exist after successful transaction")
	}
}

func (e *queriesTestEnv) testStrictEnforcement(ctx context.Context, t *testing.T) {
	var version string
	e.sqlDB.QueryRow("SELECT sqlite_version()").Scan(&version)
	var v1, v2, v3 int
	fmt.Sscanf(version, "%d.%d.%d", &v1, &v2, &v3)
	if v1 < 3 || (v1 == 3 && v2 < 37) {
		t.Skip("STRICT not supported")
	}

	_, err := e.sqlDB.Exec("INSERT INTO media (path, duration) VALUES ('strict-test.mp4', 'not-an-int')")
	if err == nil {
		t.Error("Expected error when inserting string into INTEGER column in STRICT table, but got none")
	} else {
		msg := err.Error()
		if strings.Contains(msg, "datatype mismatch") ||
			strings.Contains(msg, "cannot store TEXT value in INTEGER column") {

			t.Logf("Caught expected STRICT error: %v", msg)
		} else {
			t.Errorf("Expected a datatype mismatch error from the STRICT table, but got a different error: %v", err)
		}
	}
}
