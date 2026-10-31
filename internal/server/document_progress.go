package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

const maxDocumentChapterLength = 2048

// HandleDocumentProgress reads or writes the last-read position for a
// document. Reading progress is separate from playback progress because a
// scroll percentage must never be interpreted as media seconds.
//
// GET  /api/document-progress?path=...
// POST /api/document-progress {"path":"...","scroll_percent":0.42,"chapter":"..."}
func (c *Server) HandleDocumentProgress(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.handleGetDocumentProgress(w, r)
	case http.MethodPost:
		c.handlePostDocumentProgress(w, r)
	default:
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (c *Server) handleGetDocumentProgress(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendError(w, http.StatusBadRequest, "Path required")
		return
	}
	if found, _ := c.getMediaTypeFromDB(r.Context(), path); !found {
		sendError(w, http.StatusNotFound, "Media not found")
		return
	}

	best := models.DocumentProgress{Path: path, TimeUpdated: -1}
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			var scrollPercent float64
			var chapter sql.NullString
			var timeUpdated int64
			err := sqlDB.QueryRowContext(
				ctx,
				"SELECT scroll_percent, chapter, time_updated FROM document_progress WHERE media_path = ?",
				path,
			).Scan(&scrollPercent, &chapter, &timeUpdated)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			if timeUpdated > best.TimeUpdated {
				best.ScrollPercent = scrollPercent
				best.Chapter = chapter.String
				best.TimeUpdated = timeUpdated
			}
			return nil
		})
		if err != nil {
			logging.Log.Error("Failed to read document progress", "db", dbPath, "error", err)
		}
	}

	// A missing progress row is not an error; return the default position so
	// clients can treat every response as a progress record.
	if best.TimeUpdated < 0 {
		best.TimeUpdated = 0
	}
	sendJSON(w, http.StatusOK, best)
}

func (c *Server) handlePostDocumentProgress(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}

	var req models.DocumentProgress
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Path == "" {
		sendError(w, http.StatusBadRequest, "Path required")
		return
	}
	if found, _ := c.getMediaTypeFromDB(r.Context(), req.Path); !found {
		sendError(w, http.StatusNotFound, "Media not found")
		return
	}

	scrollPercent := req.ScrollPercent
	if math.IsNaN(scrollPercent) || scrollPercent < 0 {
		scrollPercent = 0
	}
	if scrollPercent > 1 {
		scrollPercent = 1
	}

	chapter := strings.TrimSpace(req.Chapter)
	if len(chapter) > maxDocumentChapterLength {
		chapter = chapter[:maxDocumentChapterLength]
	}
	chapterValue := sql.NullString{String: chapter, Valid: chapter != ""}

	now := time.Now().Unix()
	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			// The INSERT ... SELECT makes the foreign-key check depend on the
			// media row existing, so databases that do not contain this path
			// are left untouched instead of failing the whole request.
			res, err := sqlDB.ExecContext(ctx, `
			INSERT INTO document_progress (media_path, scroll_percent, chapter, time_updated)
			SELECT path, ?, ?, ? FROM media WHERE path = ?
			ON CONFLICT(media_path) DO UPDATE SET
				scroll_percent = excluded.scroll_percent,
				chapter = excluded.chapter,
				time_updated = excluded.time_updated`,
				scrollPercent, chapterValue, now, req.Path)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
		},
	)

	if affected == 0 && failed == 0 {
		sendError(w, http.StatusNotFound, "Media not found")
		return
	}
	if succeeded == 0 {
		sendJSON(w, http.StatusInternalServerError, map[string]any{
			"error":  "all database writes failed",
			"failed": failed,
			"errors": errs,
		})
		return
	}

	status := http.StatusOK
	if failed > 0 {
		status = http.StatusMultiStatus
	}
	sendJSON(w, status, models.DocumentProgress{
		Path:          req.Path,
		ScrollPercent: scrollPercent,
		Chapter:       chapter,
		TimeUpdated:   now,
	})
}
