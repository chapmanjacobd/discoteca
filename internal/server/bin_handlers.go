package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

func (c *Server) HandleRaw(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	dbParam := r.URL.Query().Get("db")

	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	if c.isPathBlocklisted(path) {
		logging.Log.Warn("Access denied: path is blocklisted", "path", path)
		http.Error(w, "Access denied: sensitive path", http.StatusForbidden)
		return
	}

	// Validate database if provided
	dbs := c.Databases
	if dbParam != "" {
		var err error
		dbs, err = c.filterDatabases([]string{dbParam})
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
			return
		}
	}

	logging.Log.Debug("handleRaw request", "path", path)

	var m models.Media
	found := false

	localPath := path

	for _, dbPath := range dbs {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(ctx, path)
			if err == nil {
				m = dbMedia.ToModel()
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	if !found {
		logging.Log.Warn("Access denied: file not in database", "path", path)
		http.Error(w, "Media not found in database", http.StatusNotFound)
		return
	}

	isLocal := fileutil.FileExists(localPath)
	if !isLocal {
		logging.Log.Warn("File not found on disk", "path", path)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	strategy := media.GetTranscodeStrategy(m)
	logging.Log.Debug(
		"handleRaw strategy",
		"path",
		path,
		"needs_transcode",
		strategy.NeedsTranscode,
		"vcopy",
		strategy.VideoCopy,
		"acopy",
		strategy.AudioCopy,
	)

	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(localPath)))

	if strategy.NeedsTranscode {
		if c.hasFfmpeg {
			c.HandleTranscode(w, r, localPath, m, strategy)
			return
		} else {
			logging.Log.Error("ffmpeg not found in PATH, skipping transcoding", "path", path)
		}
	}

	logging.Log.Debug("Serving local file", "path", localPath)
	http.ServeFile(w, r, localPath)
}

func (c *Server) HandleTrash(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	if rejectInvalidFlags(w, flags) {
		return
	}
	flags.OnlyDeleted = true
	flags.HideDeleted = false
	if sortBy := r.URL.Query().Get("sort"); sortBy == "" || sortBy == "default" {
		flags.SortBy = "time_deleted"
		flags.Reverse = true
	}

	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}
	c.applySearchMode(r.Context(), &flags, dbs)

	totalFlags := flags
	totalFlags.All = true
	totalFlags.Limit = 0
	totalFlags.Offset = 0
	total, countErr := query.MediaQueryCount(r.Context(), dbs, totalFlags)
	if countErr != nil {
		logging.Log.Error("Trash count query failed", "error", countErr)
	}

	media, err := query.MediaQuery(r.Context(), dbs, flags)
	if err != nil {
		logging.Log.Error("Trash query failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if countErr != nil {
		total = int64(len(media))
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	sendJSON(w, http.StatusOK, media)
}

// EmptyBinResult describes what an /api/empty-bin request did. Counts refer to
// unique paths, not databases or rows.
type EmptyBinResult struct {
	Deleted int    `json:"deleted"`
	Failed  int    `json:"failed"`
	Skipped int    `json:"skipped"`
	Error   string `json:"error,omitempty"`
}

func (c *Server) pathIsTrashed(ctx context.Context, path string) bool {
	found := false
	for _, dbPath := range c.Databases {
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			var exists int
			err := sqlDB.QueryRowContext(
				ctx,
				"SELECT 1 FROM media WHERE path = ? AND COALESCE(time_deleted, 0) > 0 LIMIT 1",
				path,
			).Scan(&exists)
			if err == nil {
				found = true
				return nil
			}
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		})
		if err != nil {
			logging.Log.Error("Failed to validate trashed path", "db", dbPath, "path", path, "error", err)
		}
		if found {
			break
		}
	}
	return found
}

// dbHasTrashedPath reports whether an exact path is present in one database.
func (c *Server) dbHasTrashedPath(ctx context.Context, dbPath, path string) bool {
	found := false
	_ = c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
		var exists int
		err := sqlDB.QueryRowContext(
			ctx,
			"SELECT 1 FROM media WHERE path = ? AND COALESCE(time_deleted, 0) > 0 LIMIT 1",
			path,
		).Scan(&exists)
		if err == nil {
			found = true
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	return found
}

// emptyBinDelete removes files and rows for paths that have already been
// validated. It returns a per-path result summary.
func (c *Server) emptyBinDelete(ctx context.Context, paths []string) EmptyBinResult {
	result := EmptyBinResult{}
	if len(paths) == 0 {
		return result
	}

	rowDeleted := make(map[string]bool)
	fileRemoved := make(map[string]bool)
	failedPaths := make(map[string]bool)

	// Validate membership before mutating anything.
	valid := make([]string, 0, len(paths))
	for _, p := range paths {
		if c.pathIsTrashed(ctx, p) {
			valid = append(valid, p)
		} else {
			result.Skipped++
		}
	}
	if len(valid) == 0 {
		return result
	}

	// Remove files first. A file that cannot be removed keeps its trash row so
	// the user can retry instead of leaving an orphaned file behind.
	for _, p := range valid {
		if fileutil.FileExists(p) {
			if err := os.Remove(p); err != nil {
				logging.Log.Error("Failed to delete file", "path", p, "error", err)
				failedPaths[p] = true
			} else {
				fileRemoved[p] = true
			}
		}
	}

	for _, dbPath := range c.Databases {
		var present []string
		for _, p := range valid {
			if failedPaths[p] {
				continue
			}
			if c.dbHasTrashedPath(ctx, dbPath, p) {
				present = append(present, p)
			}
		}
		if len(present) == 0 {
			continue
		}

		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(present)), ",")
		args := make([]any, 0, len(present))
		for _, p := range present {
			args = append(args, p)
		}
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "DELETE FROM media WHERE path IN ("+placeholders+")", args...); err != nil {
				_ = tx.Rollback()
				return err
			}
			return tx.Commit()
		})
		if err != nil {
			logging.Log.Error("Failed to delete rows from DB", "db", dbPath, "error", err)
			for _, p := range present {
				failedPaths[p] = true
			}
			continue
		}
		for _, p := range present {
			rowDeleted[p] = true
		}
	}

	for _, p := range valid {
		if failedPaths[p] {
			result.Failed++
			continue
		}
		if fileRemoved[p] || rowDeleted[p] {
			result.Deleted++
		} else {
			result.Failed++
		}
	}
	return result
}

func (c *Server) HandleEmptyBin(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var paths []string
	if len(req.Paths) > 0 {
		seen := make(map[string]bool, len(req.Paths))
		invalid := 0
		for _, p := range req.Paths {
			if c.isPathBlocklisted(p) {
				http.Error(w, "Access denied: sensitive path", http.StatusForbidden)
				return
			}
			if seen[p] {
				continue
			}
			seen[p] = true
			if !c.pathIsTrashed(r.Context(), p) {
				invalid++
				continue
			}
			paths = append(paths, p)
		}
		if invalid > 0 {
			sendJSON(w, http.StatusBadRequest, EmptyBinResult{
				Skipped: invalid,
				Error:   "one or more paths are not in the trash of a configured database",
			})
			return
		}
	} else {
		// Fallback: delete everything in trash if no paths were provided.
		flags := c.GetGlobalFlags()
		flags.OnlyDeleted = true
		flags.HideDeleted = false
		flags.All = true

		media, err := query.MediaQuery(r.Context(), c.Databases, flags)
		if err != nil {
			logging.Log.Error("Trash query failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		seen := make(map[string]bool, len(media))
		for _, m := range media {
			if c.isPathBlocklisted(m.Path) {
				continue
			}
			if !seen[m.Path] {
				seen[m.Path] = true
				paths = append(paths, m.Path)
			}
		}
	}

	result := c.emptyBinDelete(r.Context(), paths)
	logging.Log.Info("Bin emptied", "files_removed", result.Deleted, "failed", result.Failed, "skipped", result.Skipped)

	status := http.StatusOK
	if result.Failed > 0 && result.Deleted == 0 {
		status = http.StatusInternalServerError
	} else if result.Failed > 0 {
		status = http.StatusMultiStatus
	}
	sendJSON(w, status, result)
}
