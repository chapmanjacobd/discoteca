package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func (c *Server) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	titles := make(map[string]bool)
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			pls, err := queries.GetPlaylists(ctx)
			if err != nil {
				return err
			}
			for _, p := range pls {
				if p.Title.Valid {
					titles[p.Title.String] = true
				}
			}
			return nil
		})
		if err != nil {
			logging.Log.Error("Failed to fetch playlists", "db", dbPath, "error", err)
		}
	}

	uniqueTitles := make(models.PlaylistResponse, 0, len(titles))
	for t := range titles {
		uniqueTitles = append(uniqueTitles, t)
	}
	sort.Strings(uniqueTitles)

	sendJSON(w, http.StatusOK, uniqueTitles)
}

func (c *Server) handlePostPlaylist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "Title required", http.StatusBadRequest)
		return
	}

	playlistPath := "custom:" + fileutil.RandomString(12)

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			if _, err = queries.InsertPlaylist(ctx, database.InsertPlaylistParams{
				Title: sql.NullString{String: req.Title, Valid: true},
				Path:  sql.NullString{String: playlistPath, Valid: true},
			}); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return 1, nil
		},
	)
	sendWriteResult(
		w,
		writeResult{status: http.StatusCreated, affected: affected, succeeded: succeeded, failed: failed, errs: errs},
	)
}

func (c *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Title required", http.StatusBadRequest)
		return
	}

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			pls, err := queries.GetPlaylists(ctx)
			if err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			var deleted int64
			for _, p := range pls {
				if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
					if err = queries.DeletePlaylist(ctx, database.DeletePlaylistParams{
						ID:          p.ID,
						TimeDeleted: sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
					}); err != nil {
						_ = tx.Rollback()
						return 0, err
					}
					deleted++
				}
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return deleted, nil
		},
	)
	sendWriteResult(
		w,
		writeResult{status: http.StatusOK, affected: affected, succeeded: succeeded, failed: failed, errs: errs},
	)
}

func (c *Server) HandlePlaylists(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		c.handleGetPlaylists(w, r)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		c.handlePostPlaylist(w, r)
		return
	}

	if r.Method == http.MethodDelete {
		c.handleDeletePlaylist(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (c *Server) findPlaylistID(ctx context.Context, queries *database.Queries, title string) (int64, error) {
	pls, err := queries.GetPlaylists(ctx)
	if err != nil {
		return -1, err
	}

	for _, p := range pls {
		if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
			return p.ID, nil
		}
	}
	return -1, nil
}

func (c *Server) fetchPlaylistMedia(
	ctx context.Context,
	queries *database.Queries,
	playlistID int64,
	dbPath string,
) ([]models.MediaWithDB, error) {
	items, err := queries.GetPlaylistItems(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	var allMedia []models.MediaWithDB
	for _, item := range items {
		m := database.Media{
			Path:            item.Path,
			PathTokenized:   item.PathTokenized,
			Title:           item.Title,
			Duration:        item.Duration,
			Size:            item.Size,
			TimeCreated:     item.TimeCreated,
			TimeModified:    item.TimeModified,
			TimeDeleted:     item.TimeDeleted,
			TimeFirstPlayed: item.TimeFirstPlayed,
			TimeLastPlayed:  item.TimeLastPlayed,
			PlayCount:       item.PlayCount,
			Playhead:        item.Playhead,
			MediaType:       item.MediaType,
			Width:           item.Width,
			Height:          item.Height,
			Fps:             item.Fps,
			VideoCodecs:     item.VideoCodecs,
			AudioCodecs:     item.AudioCodecs,
			SubtitleCodecs:  item.SubtitleCodecs,
			VideoCount:      item.VideoCount,
			AudioCount:      item.AudioCount,
			SubtitleCount:   item.SubtitleCount,
			Album:           item.Album,
			Artist:          item.Artist,
			Genre:           item.Genre,
			Categories:      item.Categories,
			Description:     item.Description,
			Language:        item.Language,
			TimeDownloaded:  item.TimeDownloaded,
			Score:           item.Score,
		}.ToModel()
		m.TrackNumber = models.NullInt64Ptr(item.TrackNumber)
		mw := models.MediaWithDB{
			Media: m,
			DB:    dbPath,
		}
		if c.hasFfmpeg {
			mw.Transcode = media.GetTranscodeStrategy(m).NeedsTranscode
		}
		allMedia = append(allMedia, mw)
	}
	return allMedia, nil
}

func (c *Server) handleGetPlaylistItems(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Title required", http.StatusBadRequest)
		return
	}

	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			playlistID, err := c.findPlaylistID(ctx, queries, title)
			if err != nil || playlistID == -1 {
				return err
			}

			items, err := c.fetchPlaylistMedia(ctx, queries, playlistID, dbPath)
			if err == nil {
				allMedia = append(allMedia, items...)
			}
			return err
		})
	}

	// Sort to match reordering logic: TrackNumber, then Path
	sort.Slice(allMedia, func(i, j int) bool {
		tnA := int64(0)
		if allMedia[i].Media.TrackNumber != nil {
			tnA = *allMedia[i].Media.TrackNumber
		}
		tnB := int64(0)
		if allMedia[j].Media.TrackNumber != nil {
			tnB = *allMedia[j].Media.TrackNumber
		}

		if tnA != tnB {
			return tnA < tnB
		}
		return allMedia[i].Media.Path < allMedia[j].Media.Path
	})

	sendJSON(w, http.StatusOK, allMedia)
}

func (c *Server) handlePostPlaylistItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlaylistTitle string `json:"playlist_title"`
		MediaPath     string `json:"media_path"`
		TrackNumber   int64  `json:"track_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.PlaylistTitle == "" || req.MediaPath == "" {
		http.Error(w, "Playlist title and media path required", http.StatusBadRequest)
		return
	}

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			playlistID, err := c.findPlaylistID(ctx, queries, req.PlaylistTitle)
			if err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if playlistID == -1 {
				_ = tx.Rollback()
				return 0, nil
			}

			// Get max track number
			var maxTrack sql.NullInt64
			_ = tx.QueryRowContext(ctx, "SELECT MAX(track_number) FROM playlist_items WHERE playlist_id = ?", playlistID).
				Scan(&maxTrack)

			trackNum := req.TrackNumber
			if trackNum == 0 {
				if maxTrack.Valid {
					trackNum = maxTrack.Int64 + 1
				} else {
					trackNum = 1
				}
			}

			if err = queries.AddPlaylistItem(ctx, database.AddPlaylistItemParams{
				PlaylistID:  playlistID,
				MediaPath:   req.MediaPath,
				TrackNumber: sql.NullInt64{Int64: trackNum, Valid: true},
			}); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return 1, nil
		},
	)
	sendWriteResult(
		w,
		writeResult{status: http.StatusOK, affected: affected, succeeded: succeeded, failed: failed, errs: errs},
	)
}

func (c *Server) handleDeletePlaylistItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlaylistTitle string `json:"playlist_title"`
		MediaPath     string `json:"media_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.PlaylistTitle == "" || req.MediaPath == "" {
		http.Error(w, "Playlist title and media path required", http.StatusBadRequest)
		return
	}

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			pls, err := queries.GetPlaylists(ctx)
			if err != nil {
				_ = tx.Rollback()
				return 0, err
			}

			var playlistID int64 = -1
			for _, p := range pls {
				if p.Title.Valid && strings.EqualFold(p.Title.String, req.PlaylistTitle) {
					playlistID = p.ID
					break
				}
			}

			if playlistID == -1 {
				_ = tx.Rollback()
				return 0, nil
			}

			if err = queries.RemovePlaylistItem(ctx, database.RemovePlaylistItemParams{
				PlaylistID: playlistID,
				MediaPath:  req.MediaPath,
			}); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return 1, nil
		},
	)
	sendWriteResult(
		w,
		writeResult{status: http.StatusOK, affected: affected, succeeded: succeeded, failed: failed, errs: errs},
	)
}

func (c *Server) HandlePlaylistItems(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		c.handleGetPlaylistItems(w, r)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		c.handlePostPlaylistItem(w, r)
		return
	}

	if r.Method == http.MethodDelete {
		c.handleDeletePlaylistItem(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
