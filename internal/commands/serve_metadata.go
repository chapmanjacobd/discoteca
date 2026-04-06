package commands

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

// HandleMetadata returns detailed metadata for a specific media file.
// GET /api/metadata?db=...&path=...
func (c *ServeCmd) HandleMetadata(w http.ResponseWriter, r *http.Request) {
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
		_ = c.execDB(r.Context(), dp, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(ctx, path)
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

// HandleDatabases returns server configuration.
// GET /api/databases
func (c *ServeCmd) HandleDatabases(w http.ResponseWriter, _ *http.Request) {
	resp := models.DatabaseInfo{
		Databases: c.Databases,
		ReadOnly:  c.ReadOnly,
		Dev:       c.Dev,
	}
	sendJSON(w, http.StatusOK, resp)
}

// HandleCategories returns a list of categories and their media counts.
// GET /api/categories
func (c *ServeCmd) HandleCategories(w http.ResponseWriter, r *http.Request) {
	counts := make(map[string]int64)
	isCustom := make(map[string]bool)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)

			// 1. Get categories already assigned to media
			rows, err := queries.GetUsedCategories(ctx)
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
			customCats, err := queries.GetCustomCategories(ctx)
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
			models.Log.Error("Failed to fetch categories", "db", dbPath, "error", err)
		}
	}

	// 3. Add Uncategorized count
	for _, dbPath := range c.Databases {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			var count int64
			err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM media WHERE time_deleted = 0 AND (categories IS NULL OR categories = '')").
				Scan(&count)
			if err == nil {
				counts["Uncategorized"] += count
			}
			return nil
		})
	}

	res := make([]models.CatStat, 0, len(counts))
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

func (c *ServeCmd) handleCommonStats(
	w http.ResponseWriter,
	r *http.Request,
	fetch func(context.Context, *database.Queries) ([]models.CatStat, error),
	errorMsg string,
) {
	counts := make(map[string]int64)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			stats, err := fetch(ctx, queries)
			if err != nil {
				return err
			}
			for _, s := range stats {
				counts[s.Category] += s.Count
			}
			return nil
		})
		if err != nil {
			models.Log.Error(errorMsg, "db", dbPath, "error", err)
			continue
		}
	}

	res := make([]models.CatStat, 0, len(counts))
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

// HandleGenres returns genre statistics.
// GET /api/genres
func (c *ServeCmd) HandleGenres(w http.ResponseWriter, r *http.Request) {
	c.handleCommonStats(w, r, func(ctx context.Context, q *database.Queries) ([]models.CatStat, error) {
		rows, err := q.GetGenreStats(ctx)
		if err != nil {
			return nil, err
		}
		res := make([]models.CatStat, len(rows))
		for i, row := range rows {
			cat := ""
			if row.Genre.Valid {
				cat = row.Genre.String
			}
			res[i] = models.CatStat{Category: cat, Count: row.Count}
		}
		return res, nil
	}, "Failed to fetch genres")
}

// HandleRatings returns rating statistics.
// GET /api/ratings
func (c *ServeCmd) HandleRatings(w http.ResponseWriter, r *http.Request) {
	counts := make(map[int64]int64)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			stats, err := queries.GetRatingStats(ctx)
			if err != nil {
				return err
			}
			for _, s := range stats {
				counts[s.Rating] += s.Count
			}
			return nil
		})
		if err != nil {
			models.Log.Error("Failed to fetch ratings", "db", dbPath, "error", err)
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

// HandleLanguages returns language statistics.
// GET /api/languages
func (c *ServeCmd) HandleLanguages(w http.ResponseWriter, r *http.Request) {
	c.handleCommonStats(w, r, func(ctx context.Context, q *database.Queries) ([]models.CatStat, error) {
		rows, err := q.GetLanguageStats(ctx)
		if err != nil {
			return nil, err
		}
		res := make([]models.CatStat, len(rows))
		for i, row := range rows {
			cat := ""
			if row.Language.Valid {
				cat = row.Language.String
			}
			res[i] = models.CatStat{Category: cat, Count: row.Count}
		}
		return res, nil
	}, "Failed to fetch languages")
}

// CaptionsQueryParams contains the search query and filter options for caption retrieval
type CaptionsQueryParams struct {
	QueryStr  string
	Limit     int64
	VideoOnly bool
	AudioOnly bool
	ImageOnly bool
	TextOnly  bool
}

// getCaptionsWithContext fetches captions matching a query along with 2 captions before and after each match
func (c *ServeCmd) getCaptionsWithContext(
	ctx context.Context,
	queries *database.Queries,
	params CaptionsQueryParams,
) ([]database.SearchCaptionsRow, error) {
	// First, get the matching captions with media type filters
	matches, err := queries.SearchCaptions(ctx, database.SearchCaptionsParams{
		Query:     params.QueryStr,
		VideoOnly: params.VideoOnly,
		AudioOnly: params.AudioOnly,
		ImageOnly: params.ImageOnly,
		TextOnly:  params.TextOnly,
		Limit:     params.Limit,
	})
	if err != nil {
		return nil, err
	}

	// Apply in-memory ranking for better relevance
	database.RankCaptionsResults(matches, params.QueryStr)

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
			models.Log.Warn("Failed to get captions for media", "path", path, "error", err)
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
			result = append(result, m)
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
