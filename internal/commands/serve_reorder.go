package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type playlistItem struct {
	DBPath      string
	PlaylistID  int64
	MediaPath   string
	TrackNumber int64
	TimeAdded   int64
}

func (c *ServeCmd) HandlePlaylistReorder(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PlaylistTitle string `json:"playlist_title"`
		MediaPath     string `json:"media_path"`
		NewIndex      int    `json:"new_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var allItems []playlistItem

	// 1. Gather all items from all DBs
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			pls, err := queries.GetPlaylists(ctx)
			if err != nil {
				return err
			}

			var playlistID int64 = -1
			for _, p := range pls {
				if p.Title.Valid && strings.EqualFold(p.Title.String, req.PlaylistTitle) {
					playlistID = p.ID
					break
				}
			}

			if playlistID == -1 {
				return nil
			}

			items, err := queries.GetPlaylistItems(ctx, playlistID)
			if err != nil {
				return err
			}

			for _, item := range items {
				tn := int64(0)
				if item.TrackNumber.Valid {
					tn = item.TrackNumber.Int64
				}
				ta := int64(0)
				if item.TimeAdded.Valid {
					ta = item.TimeAdded.Int64
				}

				allItems = append(allItems, playlistItem{
					DBPath:      dbPath,
					PlaylistID:  playlistID,
					MediaPath:   item.Path,
					TrackNumber: tn,
					TimeAdded:   ta,
				})
			}
			return nil
		})
		if err != nil {
			models.Log.Error("Failed to gather playlist items", "db", dbPath, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if len(allItems) == 0 {
		http.Error(w, "Playlist not found or empty", http.StatusNotFound)
		return
	}

	// 2. Sort globally to establish current order matching frontend
	sort.Slice(allItems, func(i, j int) bool {
		if allItems[i].TrackNumber != allItems[j].TrackNumber {
			return allItems[i].TrackNumber < allItems[j].TrackNumber
		}
		// Match frontend sort: Path
		return allItems[i].MediaPath < allItems[j].MediaPath
	})

	// 3. Find item to move
	currentIndex := -1
	var itemToMove playlistItem

	for i, item := range allItems {
		if item.MediaPath == req.MediaPath {
			currentIndex = i
			itemToMove = item
			break
		}
	}

	if currentIndex == -1 {
		http.Error(w, "Item not found in playlist", http.StatusNotFound)
		return
	}

	// 4. Reorder list in memory
	// Remove
	newItems := make([]playlistItem, 0, len(allItems))
	newItems = append(newItems, allItems[:currentIndex]...)
	newItems = append(newItems, allItems[currentIndex+1:]...)

	// Clamp
	if req.NewIndex < 0 {
		req.NewIndex = 0
	}
	if req.NewIndex > len(newItems) {
		req.NewIndex = len(newItems)
	}

	// Insert
	finalItems := make([]playlistItem, 0, len(allItems))
	finalItems = append(finalItems, newItems[:req.NewIndex]...)
	finalItems = append(finalItems, itemToMove)
	finalItems = append(finalItems, newItems[req.NewIndex:]...)

	// 5. Update track numbers in DBs
	for i, item := range finalItems {
		newTrackNum := int64(i + 1)

		err := c.execDB(r.Context(), item.DBPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			return queries.AddPlaylistItem(ctx, database.AddPlaylistItemParams{
				PlaylistID:  item.PlaylistID,
				MediaPath:   item.MediaPath,
				TrackNumber: sql.NullInt64{Int64: newTrackNum, Valid: true},
			})
		})
		if err != nil {
			models.Log.Error("Failed to update track number", "db", item.DBPath, "path", item.MediaPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}
