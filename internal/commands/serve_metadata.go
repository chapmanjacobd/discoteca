package commands

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

// handleMetadata returns detailed metadata for a specific media file.
// GET /api/metadata?db=...&path=...
func (c *ServeCmd) handleMetadata(w http.ResponseWriter, r *http.Request) {
	dbPath := r.URL.Query().Get("db")
	path := r.URL.Query().Get("path")

	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	var metadata models.Media
	found := false

	// If dbPath is provided, only check that database
	dbs := c.Databases
	if dbPath != "" {
		dbs = []string{dbPath}
	}

	for _, dp := range dbs {
		_ = c.execDB(r.Context(), dp, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				metadata = models.FromDB(dbMedia)
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	if !found {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	sendJSON(w, http.StatusOK, metadata)
}

// handleDatabases returns server configuration.
// GET /api/databases
func (c *ServeCmd) handleDatabases(w http.ResponseWriter, r *http.Request) {
	resp := models.DatabaseInfo{
		Databases: c.Databases,
		ReadOnly:  c.ReadOnly,
		Dev:       c.Dev,
	}
	sendJSON(w, http.StatusOK, resp)
}

// handleCategories returns a list of categories and their media counts.
// GET /api/categories
func (c *ServeCmd) handleCategories(w http.ResponseWriter, r *http.Request) {
	counts := make(map[string]int64)
	isCustom := make(map[string]bool)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)

			// 1. Get categories already assigned to media
			rows, err := queries.GetUsedCategories(r.Context())
			if err != nil {
				return err
			}
			for _, row := range rows {
				if row.Categories.Valid {
					trimmed := strings.Trim(row.Categories.String, ";")
					if trimmed == "" {
						continue
					}
					cats := strings.SplitSeq(trimmed, ";")
					for cat := range cats {
						if cat != "" {
							counts[cat] += row.Count
						}
					}
				}
			}

			// 2. Get categories from custom keywords
			customCats, err := queries.GetCustomCategories(r.Context())
			if err == nil {
				for _, cat := range customCats {
					isCustom[cat] = true
					if _, ok := counts[cat]; !ok {
						counts[cat] = 0
					}
				}
			}

			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch categories", "db", dbPath, "error", err)
		}
	}

	// 3. Add Uncategorized count
	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			var count int64
			err := sqlDB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM media WHERE time_deleted = 0 AND (categories IS NULL OR categories = '')").
				Scan(&count)
			if err == nil {
				counts["Uncategorized"] += count
			}
			return nil
		})
	}

	var res []models.CatStat
	for k, v := range counts {
		res = append(res, models.CatStat{Category: k, Count: v})
	}

	sort.Slice(res, func(i, j int) bool {
		if res[i].Category == "Uncategorized" {
			return false
		}
		if res[j].Category == "Uncategorized" {
			return true
		}
		if res[i].Count != res[j].Count {
			return res[i].Count > res[j].Count
		}
		return res[i].Category < res[j].Category
	})

	sendJSON(w, http.StatusOK, res)
}

// handleGenres returns genre statistics.
// GET /api/genres
func (c *ServeCmd) handleGenres(w http.ResponseWriter, r *http.Request) {
	counts := make(map[string]int64)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			rows, err := queries.GetGenreStats(r.Context())
			if err != nil {
				return err
			}
			for _, row := range rows {
				if row.Genre.Valid {
					counts[row.Genre.String] = row.Count
				}
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch genres", "db", dbPath, "error", err)
			continue
		}
	}

	var res []models.CatStat
	for k, v := range counts {
		res = append(res, models.CatStat{Category: k, Count: v})
	}

	sort.Slice(res, func(i, j int) bool {
		if res[i].Count != res[j].Count {
			return res[i].Count > res[j].Count
		}
		return res[i].Category < res[j].Category
	})

	sendJSON(w, http.StatusOK, res)
}

// handleRatings returns rating statistics.
// GET /api/ratings
func (c *ServeCmd) handleRatings(w http.ResponseWriter, r *http.Request) {
	counts := make(map[int64]int64)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			stats, err := queries.GetRatingStats(r.Context())
			if err != nil {
				return err
			}
			for _, s := range stats {
				counts[s.Rating] = counts[s.Rating] + s.Count
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch ratings", "db", dbPath, "error", err)
			sendError(w, http.StatusInternalServerError, "Failed to fetch ratings")
			return
		}
	}

	var res []models.RatStat
	res = make([]models.RatStat, 0)
	for k, v := range counts {
		res = append(res, models.RatStat{Rating: k, Count: v})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Rating > res[j].Rating
	})

	sendJSON(w, http.StatusOK, res)
}

// handleLanguages returns language statistics.
// GET /api/languages
func (c *ServeCmd) handleLanguages(w http.ResponseWriter, r *http.Request) {
	counts := make(map[string]int64)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			rows, err := queries.GetLanguageStats(r.Context())
			if err != nil {
				return err
			}
			for _, row := range rows {
				if row.Language.Valid {
					counts[row.Language.String] = row.Count
				}
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch languages", "db", dbPath, "error", err)
			continue
		}
	}

	var res []models.CatStat
	for k, v := range counts {
		res = append(res, models.CatStat{Category: k, Count: v})
	}

	sort.Slice(res, func(i, j int) bool {
		if res[i].Count != res[j].Count {
			return res[i].Count > res[j].Count
		}
		return res[i].Category < res[j].Category
	})

	sendJSON(w, http.StatusOK, res)
}

// getCaptionsWithContext fetches captions matching a query along with 2 captions before and after each match
func (c *ServeCmd) getCaptionsWithContext(
	ctx context.Context,
	queries *database.Queries,
	queryStr string,
	limit int64,
	videoOnly, audioOnly, imageOnly, textOnly bool,
) ([]database.SearchCaptionsRow, error) {
	// First, get the matching captions with media type filters
	matches, err := queries.SearchCaptions(ctx, database.SearchCaptionsParams{
		Query:     queryStr,
		VideoOnly: videoOnly,
		AudioOnly: audioOnly,
		ImageOnly: imageOnly,
		TextOnly:  textOnly,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	// Apply in-memory ranking for better relevance
	database.RankCaptionsResults(matches, queryStr)

	if len(matches) == 0 {
		return matches, nil
	}

	// Get unique media paths that have matches
	pathSet := make(map[string]bool)
	for _, m := range matches {
		pathSet[m.MediaPath] = true
	}
	var paths []string
	for path := range pathSet {
		paths = append(paths, path)
	}

	// Get all captions for those media paths
	var allCaptions []database.Captions
	for _, path := range paths {
		captions, err := queries.GetCaptionsForMedia(ctx, path)
		if err != nil {
			slog.Warn("Failed to get captions for media", "path", path, "error", err)
			continue
		}
		allCaptions = append(allCaptions, captions...)
	}

	// Create a set of match times for each path
	matchTimes := make(map[string]map[float64]bool)
	for _, m := range matches {
		if matchTimes[m.MediaPath] == nil {
			matchTimes[m.MediaPath] = make(map[float64]bool)
		}
		if m.Time.Valid {
			matchTimes[m.MediaPath][m.Time.Float64] = true
		}
	}

	// For each match, find 2 captions before and after
	var result []database.SearchCaptionsRow
	added := make(map[string]map[float64]bool)

	for _, m := range matches {
		if !m.Time.Valid {
			continue
		}
		matchTime := m.Time.Float64
		path := m.MediaPath

		// Add the match itself
		if added[path] == nil {
			added[path] = make(map[float64]bool)
		}
		if !added[path][matchTime] {
			result = append(result, database.SearchCaptionsRow(m))
			added[path][matchTime] = true
		}

		// Find 2 captions before
		beforeCount := 0
		for _, c := range allCaptions {
			if c.MediaPath != path || !c.Time.Valid {
				continue
			}
			captionTime := c.Time.Float64
			if captionTime < matchTime && !matchTimes[path][captionTime] {
				if beforeCount < 2 && !added[path][captionTime] {
					result = append(result, database.SearchCaptionsRow{
						MediaPath: c.MediaPath,
						Time:      c.Time,
						Text:      c.Text,
						Title:     sql.NullString{},
						MediaType: sql.NullString{},
						Size:      sql.NullInt64{},
						Duration:  sql.NullInt64{},
					})
					added[path][captionTime] = true
					beforeCount++
				}
			}
		}

		// Find 2 captions after
		afterCount := 0
		for _, c := range allCaptions {
			if c.MediaPath != path || !c.Time.Valid {
				continue
			}
			captionTime := c.Time.Float64
			if captionTime > matchTime && !matchTimes[path][captionTime] {
				if afterCount < 2 && !added[path][captionTime] {
					result = append(result, database.SearchCaptionsRow{
						MediaPath: c.MediaPath,
						Time:      c.Time,
						Text:      c.Text,
						Title:     sql.NullString{},
						MediaType: sql.NullString{},
						Size:      sql.NullInt64{},
						Duration:  sql.NullInt64{},
					})
					added[path][captionTime] = true
					afterCount++
				}
			}
		}
	}

	// Sort by media_path and time
	sort.Slice(result, func(i, j int) bool {
		if result[i].MediaPath != result[j].MediaPath {
			return result[i].MediaPath < result[j].MediaPath
		}
		return result[i].Time.Float64 < result[j].Time.Float64
	})

	return result, nil
}
