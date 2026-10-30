package server

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func (c *Server) fetchAllMedia(ctx context.Context, purpose string) []models.MediaWithDB {
	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMedia(ctx, 1000000)
			if err != nil {
				return err
			}
			for _, m := range dbMedia {
				allMedia = append(allMedia, models.MediaWithDB{
					Media: m.ToModel(),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			logging.Log.Error("Failed to fetch media", "purpose", purpose, "error", err)
		}
	}
	return allMedia
}

func (c *Server) filterPlayable(allMedia []models.MediaWithDB, targetMediaType string) []models.MediaWithDB {
	var playable []models.MediaWithDB
	for _, m := range allMedia {
		if m.MediaType == nil {
			continue
		}

		if targetMediaType != "" {
			if strings.HasPrefix(*m.MediaType, targetMediaType) {
				playable = append(playable, m)
			}
		} else {
			// Default behavior: video or audio
			if strings.HasPrefix(*m.MediaType, "video") || strings.HasPrefix(*m.MediaType, "audio") ||
				*m.MediaType == "audiobook" {

				playable = append(playable, m)
			}
		}
	}
	return playable
}

func (c *Server) HandleRandomClip(w http.ResponseWriter, r *http.Request) {
	allMedia := c.fetchAllMedia(r.Context(), "random clip")

	if len(allMedia) == 0 {
		http.Error(w, "No media found", http.StatusNotFound)
		return
	}

	// Filter for video/audio only
	targetMediaType := r.URL.Query().Get("type")
	playable := c.filterPlayable(allMedia, targetMediaType)

	if len(playable) == 0 {
		http.Error(w, "No playable media found", http.StatusNotFound)
		return
	}

	item := playable[utils.RandomInt(0, len(playable)-1)]

	// Play full content (no duration clipping)
	start := 0
	end := 0
	if item.Duration != nil {
		end = int(*item.Duration)
	}

	// Support fields parameter to limit response size
	fieldsParam := r.URL.Query().Get("fields")

	type clipResponse struct {
		models.MediaWithDB

		Start int `json:"start"`
		End   int `json:"end"`
	}

	response := clipResponse{
		MediaWithDB: item,
		Start:       start,
		End:         end,
	}

	// If fields parameter is provided, only include specified fields
	if fieldsParam != "" {
		requestedFields := strings.Split(fieldsParam, ",")
		fieldSet := make(map[string]bool)
		for _, f := range requestedFields {
			fieldSet[strings.TrimSpace(f)] = true
		}

		// Clear all fields first
		cleared := models.MediaWithDB{
			DB: item.DB,
		}

		// Only include requested fields
		if fieldSet["path"] {
			cleared.Path = item.Path
		}
		if fieldSet["type"] {
			cleared.MediaType = item.MediaType
		}
		if fieldSet["duration"] {
			cleared.Duration = item.Duration
		}
		if fieldSet["start"] || fieldSet["end"] {
			// Always include start/end if either is requested since they're part of the response wrapper
			response.Start = start
			response.End = end
		}
		if fieldSet["db"] {
			cleared.DB = item.DB
		}

		response.MediaWithDB = cleared
	}

	sendJSON(w, http.StatusOK, response)
}
