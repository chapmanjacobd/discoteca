package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/chapmanjacobd/discoteca/internal/downloader"
)

// HandleDownload starts a download (POST) or returns the current job (GET).
// POST /api/download
// GET  /api/download
func (c *Server) HandleDownload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.handleDownloadStatus(w)
	case http.MethodPost:
		c.handleDownloadStart(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *Server) handleDownloadStatus(w http.ResponseWriter) {
	if c.downloadManager == nil {
		sendError(w, http.StatusServiceUnavailable, "Downloads are unavailable")
		return
	}
	status := c.downloadManager.Status()
	if status == nil {
		sendJSON(w, http.StatusOK, downloader.Status{
			State:    downloader.StateIdle,
			Progress: -1,
			Output:   []string{},
			Paths:    []string{},
		})
		return
	}
	sendJSON(w, http.StatusOK, status)
}

func (c *Server) handleDownloadStart(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if c.downloadManager == nil {
		sendError(w, http.StatusServiceUnavailable, "Downloads are unavailable; install lb or lb-wrapper")
		return
	}

	var req downloader.Options
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Database == "" {
		if len(c.Databases) == 0 {
			sendError(w, http.StatusBadRequest, "No target database is configured")
			return
		}
		req.Database = c.Databases[0]
	} else {
		dbs, err := c.filterDatabases([]string{req.Database})
		if err != nil {
			sendError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.Database = dbs[0]
	}

	status, err := c.downloadManager.Start(req)
	if err != nil {
		if errors.Is(err, downloader.ErrBusy) {
			current := c.downloadManager.Status()
			if current == nil {
				sendError(w, http.StatusConflict, err.Error())
				return
			}
			sendJSON(w, http.StatusConflict, current)
			return
		}
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusAccepted, status)
}

// HandleDownloadCancel stops the active download.
// POST /api/download/cancel
func (c *Server) HandleDownloadCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c.ReadOnly {
		sendError(w, http.StatusForbidden, "Read-only mode")
		return
	}
	if c.downloadManager == nil {
		sendError(w, http.StatusServiceUnavailable, "Downloads are unavailable")
		return
	}

	c.downloadManager.Cancel()
	status := c.downloadManager.Status()
	if status == nil {
		sendJSON(w, http.StatusOK, downloader.Status{
			State:    downloader.StateIdle,
			Progress: -1,
			Output:   []string{},
			Paths:    []string{},
		})
		return
	}
	sendJSON(w, http.StatusOK, status)
}
