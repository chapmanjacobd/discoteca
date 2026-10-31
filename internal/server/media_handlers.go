package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

// HandlePlay triggers local playback of a media file via mpv.
// POST /api/play
// Body: {"path": "..."}
func (c *Server) HandlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.PlayResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !strings.HasPrefix(req.Path, "http") {
		found, _ := c.getMediaTypeFromDB(r.Context(), req.Path)
		if !found {
			sendError(w, http.StatusForbidden, "Media not found in database")
			return
		}
		if !fileutil.FileExists(req.Path) {
			// Playback is a probe, not a library action: do not mutate the DB.
			sendError(w, http.StatusNotFound, "File not found")
			return
		}
	}

	// Trigger local playback. The process must outlive this HTTP request:
	// using r.Context() here would kill mpv as soon as the handler returns.
	logging.Log.Info("Playing", "path", req.Path)
	cmd := exec.CommandContext(context.WithoutCancel(r.Context()), "mpv", req.Path)
	if err := cmd.Start(); err != nil {
		logging.Log.Error("Failed to start mpv", "error", err)
		sendError(w, http.StatusInternalServerError, "Failed to start playback: "+err.Error())
		return
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			logging.Log.Debug("mpv exited with error", "path", req.Path, "error", err)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// markDeletedInAllDBs marks or restores a path and reports per-database errors.
func (c *Server) markDeletedInAllDBs(ctx context.Context, path string, deleted bool) (
	affected int64, succeeded, failed int, errs []string,
) {
	if c.ReadOnly {
		return 0, 0, 0, nil
	}
	var deleteTime int64 = 0
	if deleted {
		deleteTime = time.Now().Unix()
	}

	return c.execWriteAcrossDBs(ctx, func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
		res, err := sqlDB.ExecContext(
			ctx,
			"UPDATE media SET time_deleted = ? WHERE path = ?",
			deleteTime,
			path,
		)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	})
}

// HandleDelete marks a file as deleted or restores it in all databases.
// POST /api/delete
// Body: {"path": "...", "restore": bool}
func (c *Server) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	affected, succeeded, failed, errs := c.markDeletedInAllDBs(r.Context(), req.Path, !req.Restore)
	sendWriteResult(
		w,
		writeResult{
			status:          http.StatusOK,
			affected:        affected,
			succeeded:       succeeded,
			failed:          failed,
			errs:            errs,
			notFoundMessage: "Media not found",
		},
	)
}

// HandleProgress updates the playback progress for a media file.
// POST /api/progress
// Body: {"path": "...", "playhead": int64, "completed": bool}
func (c *Server) HandleProgress(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.ProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	now := time.Now().Unix()
	increment := 0
	if req.Completed {
		increment = 1
	}

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			res, err := sqlDB.ExecContext(ctx, `
			UPDATE media
			SET time_last_played = ?,
			    time_first_played = CASE
			        WHEN COALESCE(time_first_played, 0) = 0 THEN ?
			        ELSE time_first_played
			    END,
			    playhead = ?,
			    play_count = COALESCE(play_count, 0) + ?
			WHERE path = ?`,
				now, now, req.Playhead, increment, req.Path)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
		},
	)
	sendWriteResult(
		w,
		writeResult{
			status:          http.StatusOK,
			affected:        affected,
			succeeded:       succeeded,
			failed:          failed,
			errs:            errs,
			notFoundMessage: "Media not found",
		},
	)
}

// HandleMarkUnplayed resets play count and progress for a media file.
// POST /api/mark-unplayed
// Body: {"path": "..."}
func (c *Server) HandleMarkUnplayed(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			res, err := sqlDB.ExecContext(ctx, `
			UPDATE media
			SET play_count = 0,
			    playhead = 0,
			    time_last_played = 0,
			    time_first_played = 0
			WHERE path = ?`,
				req.Path)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
		},
	)
	sendWriteResult(
		w,
		writeResult{
			status:          http.StatusOK,
			affected:        affected,
			succeeded:       succeeded,
			failed:          failed,
			errs:            errs,
			notFoundMessage: "Media not found",
		},
	)
}

// HandleMarkPlayed increments play count and resets progress for a media file.
// POST /api/mark-played
// Body: {"path": "..."}
func (c *Server) HandleMarkPlayed(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	now := time.Now().Unix()
	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			res, err := sqlDB.ExecContext(ctx, `
			UPDATE media
			SET time_last_played = ?,
			    time_first_played = CASE
			        WHEN COALESCE(time_first_played, 0) = 0 THEN ?
			        ELSE time_first_played
			    END,
			    play_count = COALESCE(play_count, 0) + 1,
			    playhead = 0
			WHERE path = ?`,
				now, now, req.Path)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
		},
	)
	sendWriteResult(
		w,
		writeResult{
			status:          http.StatusOK,
			affected:        affected,
			succeeded:       succeeded,
			failed:          failed,
			errs:            errs,
			notFoundMessage: "Media not found",
		},
	)
}

// HandleRate updates the rating for a media file.
// POST /api/rate
// Body: {"path": "...", "score": float64}
func (c *Server) HandleRate(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path  string  `json:"path"`
		Score float64 `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			res, err := sqlDB.ExecContext(
				ctx,
				"UPDATE media SET score = ? WHERE path = ?",
				req.Score,
				req.Path,
			)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
		},
	)
	sendWriteResult(
		w,
		writeResult{
			status:          http.StatusOK,
			affected:        affected,
			succeeded:       succeeded,
			failed:          failed,
			errs:            errs,
			notFoundMessage: "Media not found",
		},
	)
}
