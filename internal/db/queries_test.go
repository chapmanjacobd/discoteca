package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupDB(t *testing.T) (*sql.DB, *Queries) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	schema := GetSchema()

	// Check for STRICT support
	var version string
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&version); err != nil {
		t.Fatal(err)
	}
	var v1, v2, v3 int
	fmt.Sscanf(version, "%d.%d.%d", &v1, &v2, &v3)
	hasStrict := v1 > 3 || (v1 == 3 && v2 >= 37)

	if !hasStrict {
		schema = strings.ReplaceAll(schema, "STRICT", "")
		// Also replace unixepoch() with strftime('%s', 'now') if using very old SQLite
		if v1 < 3 || (v1 == 3 && v2 < 38) {
			schema = strings.ReplaceAll(schema, "unixepoch()", "strftime('%s', 'now')")
		}
	}

	// Simple FTS5 check
	var hasFTS5 bool
	err = db.QueryRow("SELECT 1 FROM pragma_compile_options WHERE compile_options = 'ENABLE_FTS5'").Scan(&hasFTS5)
	if err != nil {
		// maybe it's just not in the list, try creating a virtual table
		_, err = db.Exec("CREATE VIRTUAL TABLE fts_test USING fts5(t)")
		if err == nil {
			hasFTS5 = true
			db.Exec("DROP TABLE fts_test")
		}
	}

	if !hasFTS5 {
		// Filter out FTS5 specific commands if not available
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

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to execute schema: %v", err)
	}

	return db, New(db)
}

func TestQueries(t *testing.T) {
	db, q := setupDB(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("UpsertAndGet", func(t *testing.T) {
		err := q.UpsertMedia(ctx, UpsertMediaParams{
			Path:  "test.mp4",
			Title: sql.NullString{String: "Test Title", Valid: true},
			Size:  sql.NullInt64{Int64: 1000, Valid: true},
		})
		if err != nil {
			t.Errorf("UpsertMedia failed: %v", err)
		}

		m, err := q.GetMediaByPathExact(ctx, "test.mp4")
		if err != nil {
			t.Errorf("GetMediaByPathExact failed: %v", err)
		}
		if m.Title.String != "Test Title" {
			t.Errorf("Expected Test Title, got %s", m.Title.String)
		}
	})

	t.Run("CategoryStats", func(t *testing.T) {
		err := q.UpdateMediaCategories(ctx, UpdateMediaCategoriesParams{
			Path:       "test.mp4",
			Categories: sql.NullString{String: ";comedy;", Valid: true},
		})
		if err != nil {
			t.Fatal(err)
		}

		stats, err := q.GetCategoryStats(ctx)
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
	})

	t.Run("MediaFiltering", func(t *testing.T) {
		q.UpsertMedia(ctx, UpsertMediaParams{
			Path:     "video.mp4",
			Type:     sql.NullString{String: "video", Valid: true},
			Duration: sql.NullInt64{Int64: 100, Valid: true},
			Size:     sql.NullInt64{Int64: 5000, Valid: true},
		})
		q.UpsertMedia(ctx, UpsertMediaParams{
			Path:     "audio.mp3",
			Type:     sql.NullString{String: "audio", Valid: true},
			Duration: sql.NullInt64{Int64: 200, Valid: true},
			Size:     sql.NullInt64{Int64: 2000, Valid: true},
		})

		// GetMediaByType
		res, _ := q.GetMediaByType(ctx, GetMediaByTypeParams{
			Column1: true, // video
			Column2: false,
			Column3: false,
			Limit:   10,
		})
		if len(res) != 1 || res[0].Path != "video.mp4" {
			t.Errorf("GetMediaByType video failed, got %v", res)
		}

		// GetMediaBySize
		res, _ = q.GetMediaBySize(ctx, GetMediaBySizeParams{
			Size:   sql.NullInt64{Int64: 3000, Valid: true},
			Size_2: sql.NullInt64{Int64: 6000, Valid: true},
			Limit:  10,
		})
		if len(res) != 1 || res[0].Path != "video.mp4" {
			t.Errorf("GetMediaBySize failed, got %v", res)
		}

		// GetMediaByDuration
		res, _ = q.GetMediaByDuration(ctx, GetMediaByDurationParams{
			Duration:   sql.NullInt64{Int64: 150, Valid: true},
			Duration_2: sql.NullInt64{Int64: 250, Valid: true},
			Limit:      10,
		})
		if len(res) != 1 || res[0].Path != "audio.mp3" {
			t.Errorf("GetMediaByDuration failed, got %v", res)
		}
	})

	t.Run("HistoryAndStats", func(t *testing.T) {
		path := "history.mp4"
		q.UpsertMedia(ctx, UpsertMediaParams{
			Path:     path,
			Duration: sql.NullInt64{Int64: 1000, Valid: true},
		})

		q.UpdatePlayHistory(ctx, UpdatePlayHistoryParams{
			Path:            path,
			Playhead:        sql.NullInt64{Int64: 500, Valid: true},
			TimeLastPlayed:  sql.NullInt64{Int64: 12345678, Valid: true},
			TimeFirstPlayed: sql.NullInt64{Int64: 12345678, Valid: true},
		})

		q.InsertHistory(ctx, InsertHistoryParams{
			MediaPath: path,
			Playhead:  sql.NullInt64{Int64: 500, Valid: true},
		})

		count, _ := q.GetHistoryCount(ctx, path)
		if count != 1 {
			t.Errorf("Expected 1 history entry, got %d", count)
		}

		unfinished, _ := q.GetUnfinishedMedia(ctx, 10)
		if len(unfinished) != 1 || unfinished[0].Path != path {
			t.Errorf("Expected 1 unfinished media, got %v", unfinished)
		}

		stats, _ := q.GetStats(ctx)
		if stats.WatchedCount != 1 {
			t.Errorf("Expected 1 watched media in stats, got %d", stats.WatchedCount)
		}
	})

	t.Run("Playlists", func(t *testing.T) {
		id, err := q.InsertPlaylist(ctx, InsertPlaylistParams{
			Path:         sql.NullString{String: "http://example.com/playlist", Valid: true},
			ExtractorKey: sql.NullString{String: "youtube", Valid: true},
		})
		if err != nil {
			t.Fatal(err)
		}
		if id == 0 {
			t.Error("Expected non-zero ID for playlist")
		}

		playlists, _ := q.GetPlaylists(ctx)
		if len(playlists) != 1 || playlists[0].Path.String != "http://example.com/playlist" {
			t.Errorf("Expected 1 playlist, got %v", playlists)
		}
	})

	t.Run("UpdateOperations", func(t *testing.T) {
		q.UpsertMedia(ctx, UpsertMediaParams{Path: "old.mp4"})
		q.UpdatePath(ctx, UpdatePathParams{Path: "new.mp4", Path_2: "old.mp4"})
		_, err := q.GetMediaByPathExact(ctx, "old.mp4")
		if err == nil {
			t.Error("old.mp4 should not exist")
		}
		_, err = q.GetMediaByPathExact(ctx, "new.mp4")
		if err != nil {
			t.Error("new.mp4 should exist")
		}

		q.MarkDeleted(ctx, MarkDeletedParams{Path: "new.mp4", TimeDeleted: sql.NullInt64{Int64: 1, Valid: true}})
		m, _ := q.GetMediaByPathExact(ctx, "new.mp4")
		if m.TimeDeleted.Int64 == 0 {
			t.Error("Expected time_deleted to be set")
		}
	})

	t.Run("FTSAndCaptions", func(t *testing.T) {
		// Check if FTS tables exist before running
		var exists int
		db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='media_fts'").Scan(&exists)
		if exists == 0 {
			t.Skip("FTS5 not available")
		}

		path := "fts_video.mp4"
		q.UpsertMedia(ctx, UpsertMediaParams{
			Path:  path,
			Title: sql.NullString{String: "Unique Title for FTS", Valid: true},
		})

		// SearchMediaFTS
		res, err := q.SearchMediaFTS(ctx, SearchMediaFTSParams{
			Query: "Unique",
			Limit: 10,
		})
		if err != nil {
			t.Errorf("SearchMediaFTS failed: %v", err)
		}
		if len(res) == 0 {
			t.Error("SearchMediaFTS returned no results")
		}

		// Captions
		err = q.InsertCaption(ctx, InsertCaptionParams{
			MediaPath: path,
			Time:      sql.NullFloat64{Float64: 10.5, Valid: true},
			Text:      sql.NullString{String: "Hello from captions", Valid: true},
		})
		if err != nil {
			t.Fatalf("InsertCaption failed: %v", err)
		}

		resCaptions, err := q.SearchCaptions(ctx, SearchCaptionsParams{
			Query:     "Hello",
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
	})

	t.Run("MiscQueries", func(t *testing.T) {
		q.UpsertMedia(ctx, UpsertMediaParams{
			Path:  "random.mp4",
			Type:  sql.NullString{String: "video", Valid: true},
			Score: sql.NullFloat64{Float64: 5.0, Valid: true},
		})

		// GetRandomMedia
		random, _ := q.GetRandomMedia(ctx, 1)
		if len(random) == 0 {
			t.Error("GetRandomMedia failed")
		}

		// GetRatingStats
		ratings, _ := q.GetRatingStats(ctx)
		if len(ratings) == 0 {
			t.Error("GetRatingStats failed")
		}

		// GetStatsByType
		stats, _ := q.GetStatsByType(ctx)
		if len(stats) == 0 {
			t.Error("GetStatsByType failed")
		}

		// GetAllMediaMetadata
		meta, _ := q.GetAllMediaMetadata(ctx)
		if len(meta) == 0 {
			t.Error("GetAllMediaMetadata failed")
		}

		// GetMedia
		res, _ := q.GetMedia(ctx, 10)
		if len(res) == 0 {
			t.Error("GetMedia failed")
		}

		// GetMediaByPath
		res, _ = q.GetMediaByPath(ctx, GetMediaByPathParams{Path: "%random%", Limit: 10})
		if len(res) == 0 {
			t.Error("GetMediaByPath failed")
		}

		// GetMediaByPlayCount
		res, _ = q.GetMediaByPlayCount(ctx, GetMediaByPlayCountParams{PlayCount: sql.NullInt64{Int64: 0, Valid: true}, PlayCount_2: sql.NullInt64{Int64: 10, Valid: true}, Limit: 10})
		if len(res) == 0 {
			t.Error("GetMediaByPlayCount failed")
		}

		// GetSiblingMedia
		res, _ = q.GetSiblingMedia(ctx, GetSiblingMediaParams{Path: "%", Path_2: "non-existent", Limit: 10})
		if len(res) == 0 {
			t.Error("GetSiblingMedia failed")
		}

		// GetUnwatchedMedia
		res, _ = q.GetUnwatchedMedia(ctx, 10)
		if len(res) == 0 {
			t.Error("GetUnwatchedMedia failed")
		}

		// GetWatchedMedia
		q.UpdatePlayHistory(ctx, UpdatePlayHistoryParams{Path: "random.mp4", TimeLastPlayed: sql.NullInt64{Int64: 1, Valid: true}})
		res, _ = q.GetWatchedMedia(ctx, 10)
		if len(res) == 0 {
			t.Error("GetWatchedMedia failed")
		}
	})

	t.Run("WithTx", func(t *testing.T) {
		tx, _ := db.Begin()
		qtx := q.WithTx(tx)
		err := qtx.UpsertMedia(ctx, UpsertMediaParams{Path: "tx.mp4"})
		if err != nil {
			t.Errorf("WithTx failed: %v", err)
		}
		tx.Commit()

		_, err = q.GetMediaByPathExact(ctx, "tx.mp4")
		if err != nil {
			t.Error("tx.mp4 should exist after successful transaction")
		}
	})

	t.Run("StrictEnforcement", func(t *testing.T) {
		var version string
		db.QueryRow("SELECT sqlite_version()").Scan(&version)
		var v1, v2, v3 int
		fmt.Sscanf(version, "%d.%d.%d", &v1, &v2, &v3)
		if v1 < 3 || (v1 == 3 && v2 < 37) {
			t.Skip("STRICT not supported")
		}

		// Try to insert a string into an INTEGER column (duration)
		_, err := db.Exec("INSERT INTO media (path, duration) VALUES ('strict-test.mp4', 'not-an-int')")
		if err == nil {
			t.Error("Expected error when inserting string into INTEGER column in STRICT table, but got none")
		} else if !strings.Contains(err.Error(), "datatype mismatch") && !strings.Contains(err.Error(), "cannot store TEXT value in INTEGER column") {
			t.Errorf("Expected datatype mismatch error, got: %v", err)
		}
	})
}
