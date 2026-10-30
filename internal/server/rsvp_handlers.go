package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/reading"
)

func (c *Server) HandleRSVP(w http.ResponseWriter, r *http.Request) {
	path, wpm, err := c.validateAndExtractRSVPParams(w, r)
	if err != nil {
		return
	}

	text, err := reading.ExtractText(r.Context(), path)
	if err != nil {
		logging.Log.Error("Text extraction failed", "path", path, "error", err)
		http.Error(w, fmt.Sprintf("Text extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	assContent, duration := reading.GenerateRSVPAss(text, wpm)
	if duration <= 0 {
		http.Error(w, "Empty text content", http.StatusBadRequest)
		return
	}

	tmpDir, wavPath, err := c.prepareRSVPFiles(r.Context(), text, assContent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	c.streamRSVPVideo(w, r, assContent, wavPath, duration)
}

func (c *Server) validateAndExtractRSVPParams(w http.ResponseWriter, r *http.Request) (string, int, error) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return "", 0, errors.New("path required")
	}

	// Verify path in database
	found := false
	for _, dbPath := range c.Databases {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			_, err := queries.GetMediaByPathExact(ctx, path)
			if err == nil {
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	if !found {
		http.Error(w, "Access denied", http.StatusForbidden)
		return "", 0, errors.New("access denied")
	}

	wpm := 250
	if wpmStr := r.URL.Query().Get("wpm"); wpmStr != "" {
		if val, err := strconv.Atoi(wpmStr); err == nil && val > 0 {
			wpm = val
		}
	}

	return path, wpm, nil
}

func (c *Server) prepareRSVPFiles(ctx context.Context, text, assContent string) (tmpDir, wavPath string, err error) {
	tmpDir, err = os.MkdirTemp("", "disco-rsvp-*")
	if err != nil {
		return "", "", errors.New("failed to create temp directory")
	}

	assPath := filepath.Join(tmpDir, "subtitles.ass")
	if writeErr := os.WriteFile(assPath, []byte(assContent), 0o644); writeErr != nil {
		return tmpDir, "", errors.New("failed to write subtitles")
	}

	wavPath = filepath.Join(tmpDir, "audio.wav")
	if ttsErr := reading.GenerateTTS(ctx, text, wavPath, 250); ttsErr != nil {
		logging.Log.Warn("TTS generation failed", "error", ttsErr)
		wavPath = ""
	}

	return tmpDir, wavPath, nil
}

func (c *Server) streamRSVPVideo(
	w http.ResponseWriter,
	r *http.Request,
	assContent, wavPath string,
	duration float64,
) {
	// Create temp directory for ASS
	tmpDir, err := os.MkdirTemp("", "disco-rsvp-ass-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	assPath := filepath.Join(tmpDir, "subtitles.ass")
	if err := os.WriteFile(assPath, []byte(assContent), 0o644); err != nil {
		http.Error(w, "Failed to write subtitles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "video/webm")
	w.Header().Set("Accept-Ranges", "bytes")

	// FFmpeg: black background + audio + RSVP subtitles
	var args []string
	args = append(args, "-hide_banner", "-loglevel", "error")
	args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("color=c=black:s=1280x720:d=%f", duration))

	if wavPath != "" {
		args = append(args, "-i", wavPath)
	}

	// Escape path for ffmpeg filter (simple Linux paths should be fine, but let's be safe)
	escapedAssPath := strings.ReplaceAll(assPath, "\\", "/")
	escapedAssPath = strings.ReplaceAll(escapedAssPath, ":", "\\:")

	args = append(args,
		"-vf", fmt.Sprintf("ass='%s'", escapedAssPath),
		"-c:v", "libvpx-vp9",
		"-deadline", "realtime",
		"-cpu-used", "8",
		"-crf", "30",
		"-b:v", "0",
	)

	if wavPath != "" {
		args = append(args, "-c:a", "libopus", "-b:a", "64k")
	}

	args = append(args, "-f", "webm", "pipe:1")

	logging.Log.Info("Starting RSVP stream", "wpm", 250, "duration", duration)

	cmd := exec.CommandContext(r.Context(), "ffmpeg", args...)
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		if r.Context().Err() == nil {
			logging.Log.Error("FFmpeg RSVP streaming failed", "error", err)
		}
	}
}
