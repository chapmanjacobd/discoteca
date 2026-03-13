package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func (c *ServeCmd) handleCategorizeKeywords(w http.ResponseWriter, r *http.Request) {
	type catKeywords struct {
		Category string   `json:"category"`
		Keywords []string `json:"keywords"`
	}

	data := make(map[string]map[string]bool)

	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			rows, err := sqlDB.QueryContext(r.Context(), "SELECT category, keyword FROM custom_keywords")
			if err != nil {
				return nil
			}
			defer rows.Close()
			for rows.Next() {
				var cat, kw string
				if err := rows.Scan(&cat, &kw); err == nil {
					if _, ok := data[cat]; !ok {
						data[cat] = make(map[string]bool)
					}
					data[cat][kw] = true
				}
			}
			return nil
		})
	}

	var results []catKeywords
	for cat, kwSet := range data {
		var kws []string
		for kw := range kwSet {
			kws = append(kws, kw)
		}
		sort.Strings(kws)
		results = append(results, catKeywords{
			Category: cat,
			Keywords: kws,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Category < results[j].Category
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (c *ServeCmd) handleCategorizeDefaults(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count := 0
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			tx, err := sqlDB.Begin()
			if err != nil {
				return err
			}
			stmt, err := tx.PrepareContext(r.Context(), "INSERT OR IGNORE INTO custom_keywords (category, keyword) VALUES (?, ?)")
			if err != nil {
				tx.Rollback()
				return err
			}
			defer stmt.Close()

			for cat, keywords := range models.DefaultCategories {
				for _, kw := range keywords {
					_, err := stmt.ExecContext(r.Context(), cat, kw)
					if err == nil {
						count++
					}
				}
			}
			return tx.Commit()
		})
		if err != nil {
			slog.Error("Failed to insert default categories", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (c *ServeCmd) handleCategorizeDeleteCategory(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := r.URL.Query().Get("category")
	if category == "" {
		http.Error(w, "Category required", http.StatusBadRequest)
		return
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			_, err := sqlDB.ExecContext(r.Context(), "DELETE FROM custom_keywords WHERE category = ?", category)
			return err
		})
		if err != nil {
			slog.Error("Failed to delete category", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (c *ServeCmd) handleCategorizeKeyword(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodDelete {
		var req struct {
			Category string `json:"category"`
			Keyword  string `json:"keyword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				_, err := sqlDB.ExecContext(r.Context(), "DELETE FROM custom_keywords WHERE category = ? AND keyword = ?", req.Category, req.Keyword)
				return err
			})
			if err != nil {
				slog.Error("Failed to delete keyword", "db", dbPath, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	var req struct {
		Category string `json:"category"`
		Keyword  string `json:"keyword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Category == "" || req.Keyword == "" {
		http.Error(w, "Category and Keyword are required", http.StatusBadRequest)
		return
	}

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			_, err := sqlDB.ExecContext(r.Context(), "INSERT OR IGNORE INTO custom_keywords (category, keyword) VALUES (?, ?)", req.Category, req.Keyword)
			return err
		})
		if err != nil {
			slog.Error("Failed to save custom keyword", "db", dbPath, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (c *ServeCmd) handleRandomClip(w http.ResponseWriter, r *http.Request) {
	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMedia(r.Context(), 1000000)
			if err != nil {
				return err
			}
			for _, m := range dbMedia {
				allMedia = append(allMedia, models.MediaWithDB{
					Media: models.FromDB(m),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch media for random clip", "error", err)
		}
	}

	if len(allMedia) == 0 {
		http.Error(w, "No media found", http.StatusNotFound)
		return
	}

	// Filter for video/audio only
	var playable []models.MediaWithDB
	targetType := r.URL.Query().Get("type")

	for _, m := range allMedia {
		if m.Type == nil {
			continue
		}

		if targetType != "" {
			if strings.HasPrefix(*m.Type, targetType) {
				playable = append(playable, m)
			}
		} else {
			// Default behavior: video or audio
			if strings.HasPrefix(*m.Type, "video") || strings.HasPrefix(*m.Type, "audio") || *m.Type == "audiobook" {
				playable = append(playable, m)
			}
		}
	}

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
			cleared.Type = item.Type
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *ServeCmd) handleCategorizeSuggest(w http.ResponseWriter, r *http.Request) {
	fullPath := r.URL.Query().Get("full_path") == "true"

	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMedia(r.Context(), 1000000)
			if err != nil {
				return err
			}
			for _, m := range dbMedia {
				allMedia = append(allMedia, models.MediaWithDB{
					Media: models.FromDB(m),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch media for categorize suggest", "error", err)
		}
	}

	// Fetch existing keywords to filter them out
	existingKeywords := make(map[string]bool)
	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			rows, err := sqlDB.QueryContext(r.Context(), "SELECT DISTINCT keyword FROM custom_keywords")
			if err != nil {
				return nil
			}
			defer rows.Close()
			for rows.Next() {
				var kw string
				if err := rows.Scan(&kw); err == nil {
					existingKeywords[strings.ToLower(kw)] = true
				}
			}
			return nil
		})
	}

	cmd := CategorizeCmd{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		PostActionFlags:  c.PostActionFlags,
		Databases:        c.Databases,
		FullPath:         fullPath,
	}
	// Note: mineCategories and applyCategories need to be exported or called through a wrapper
	// Since I'm in the same package 'commands', I can call them directly.

	// We need to compile regexes first
	compiled := cmd.CompileRegexes()

	wordCounts := make(map[string]int)
	for _, m := range allMedia {
		// Skip files that already have categories assigned
		if m.Categories != nil && *m.Categories != "" {
			continue
		}

		matched := false
		pathAndTitle := m.Path
		if m.Title != nil {
			pathAndTitle += " " + *m.Title
		}

		for _, res := range compiled {
			for _, re := range res {
				if re.MatchString(pathAndTitle) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if !matched {
			// Use a map to count each word only once per file
			uniqueWords := make(map[string]bool)

			sentence := ""
			if fullPath {
				sentence = utils.PathToSentenceFull(m.Path)
			} else {
				sentence = utils.PathToSentence(m.Path)
			}
			words := utils.ExtractWords(sentence)
			if m.Title != nil {
				words = append(words, utils.ExtractWords(*m.Title)...)
			}

			for _, word := range words {
				if len(word) < 4 {
					continue
				}
				// Filter out already-assigned keywords
				if existingKeywords[strings.ToLower(word)] {
					continue
				}
				// Only count each word once per file
				if !uniqueWords[word] {
					uniqueWords[word] = true
					wordCounts[word]++
				}
			}
		}
	}

	type wordFreq struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}
	var freqs []wordFreq
	for w, c := range wordCounts {
		if c > 1 {
			freqs = append(freqs, wordFreq{Word: w, Count: c})
		}
	}

	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].Count > freqs[j].Count
	})

	limit := min(len(freqs), 100)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(freqs[:limit])
}

func (c *ServeCmd) handleCategorizeApply(w http.ResponseWriter, r *http.Request) {
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	fullPath := r.URL.Query().Get("full_path") == "true"

	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMedia(r.Context(), 1000000)
			if err != nil {
				return err
			}
			for _, m := range dbMedia {
				allMedia = append(allMedia, models.MediaWithDB{
					Media: models.FromDB(m),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			slog.Error("Failed to fetch media for categorize apply", "error", err)
		}
	}

	cmd := CategorizeCmd{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		PostActionFlags:  c.PostActionFlags,
		Databases:        c.Databases,
		FullPath:         fullPath,
	}
	compiled := cmd.CompileRegexes()

	if len(compiled) == 0 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"count": 0}`)
		return
	}

	count := 0
	for _, m := range allMedia {
		foundCategories := []string{}
		pathAndTitle := m.Path
		if m.Title != nil {
			pathAndTitle += " " + *m.Title
		}

		for cat, res := range compiled {
			for _, re := range res {
				if re.MatchString(pathAndTitle) {
					foundCategories = append(foundCategories, cat)
					break
				}
			}
		}

		if len(foundCategories) > 0 {
			merged := make(map[string]bool)
			if m.Categories != nil && *m.Categories != "" {
				existing := strings.SplitSeq(strings.Trim(*m.Categories, ";"), ";")
				for e := range existing {
					if e != "" {
						merged[strings.TrimSpace(e)] = true
					}
				}
			}
			for _, f := range foundCategories {
				merged[f] = true
			}
			combined := []string{}
			for k := range merged {
				combined = append(combined, k)
			}
			sort.Strings(combined)
			newCategories := ";" + strings.Join(combined, ";") + ";"

			err := c.execDB(r.Context(), m.DB, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				return queries.UpdateMediaCategories(r.Context(), database.UpdateMediaCategoriesParams{
					Categories: utils.ToNullString(newCategories),
					Path:       m.Path,
				})
			})
			if err == nil {
				count++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"count": %d}`, count)
}

func (c *ServeCmd) handleRaw(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	dbParam := r.URL.Query().Get("db")

	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	if c.isPathBlocklisted(path) {
		slog.Warn("Access denied: path is blocklisted", "path", path)
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

	slog.Debug("handleRaw request", "path", path)

	var m models.Media
	found := false

	localPath := path

	for _, dbPath := range dbs {
		_ = c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(r.Context(), path)
			if err == nil {
				m = models.FromDB(dbMedia)
				found = true
			}
			return err
		})
		if found {
			break
		}
	}

	if !found {
		slog.Warn("Access denied: file not in database", "path", path)
		http.Error(w, "Media not found in database", http.StatusNotFound)
		return
	}

	isLocal := utils.FileExists(localPath)
	if !isLocal {
		slog.Warn("File not found on disk, marking as deleted in databases", "path", path)
		c.markDeletedInAllDBs(r.Context(), path, true)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	strategy := utils.GetTranscodeStrategy(m)
	slog.Debug("handleRaw strategy", "path", path, "needs_transcode", strategy.NeedsTranscode, "vcopy", strategy.VideoCopy, "acopy", strategy.AudioCopy)

	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(localPath)))

	if strategy.NeedsTranscode {
		if c.hasFfmpeg {
			c.handleTranscode(w, r, localPath, m, strategy)
			return
		} else {
			slog.Error("ffmpeg not found in PATH, skipping transcoding", "path", path)
		}
	}

	slog.Debug("Serving local file", "path", localPath)
	http.ServeFile(w, r, localPath)
}

func (c *ServeCmd) handleTrash(w http.ResponseWriter, r *http.Request) {
	flags := c.GetGlobalFlags()
	flags.OnlyDeleted = true
	flags.HideDeleted = false
	flags.All = true
	flags.SortBy = "time_deleted"
	flags.Reverse = true

	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		slog.Error("Trash query failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(len(media)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

func (c *ServeCmd) handleEmptyBin(w http.ResponseWriter, r *http.Request) {
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

	var media []models.MediaWithDB
	if len(req.Paths) > 0 {
		// Only delete the requested paths
		for _, p := range req.Paths {
			media = append(media, models.MediaWithDB{Media: models.Media{Path: p}})
		}
	} else {
		// Fallback: Delete everything in trash if no paths provided
		flags := c.GetGlobalFlags()
		flags.OnlyDeleted = true
		flags.HideDeleted = false
		flags.All = true

		var err error
		media, err = query.MediaQuery(context.Background(), c.Databases, flags)
		if err != nil {
			slog.Error("Trash query failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	count := 0
	for _, m := range media {
		if utils.FileExists(m.Path) {
			if err := os.Remove(m.Path); err != nil {
				slog.Error("Failed to delete file", "path", m.Path, "error", err)
				continue
			}
		}

		// Remove from DB
		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				result, err := sqlDB.Exec("DELETE FROM media WHERE path = ?", m.Path)
				if err != nil {
					return err
				}
				rows, _ := result.RowsAffected()
				if rows > 0 {
					count++
				}
				return nil
			})
			if err != nil {
				slog.Error("Failed to delete from DB", "db", dbPath, "error", err)
			}
		}
	}

	slog.Info("Bin emptied", "files_removed", count)
	fmt.Fprintf(w, "Deleted %d files", count)
}

func (c *ServeCmd) handleOPDS(w http.ResponseWriter, r *http.Request) {
	flags := c.GetGlobalFlags()
	flags.TextOnly = true
	flags.All = true

	media, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		slog.Error("OPDS query failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml;charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opds="http://opds-spec.org/2010/catalog">
  <id>discoteca-text</id>
  <title>Discoteca Text</title>
  <updated>`+time.Now().Format(time.RFC3339)+`</updated>
  <author><name>Discoteca</name></author>
`)

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	for _, m := range media {
		title := m.Stem()
		if m.Title != nil && *m.Title != "" {
			title = *m.Title
		}

		author := "Unknown"
		if m.Artist != nil && *m.Artist != "" {
			author = *m.Artist
		}

		mimeType := "application/octet-stream"
		if m.Type != nil {
			mimeType = *m.Type
		}

		fmt.Fprintf(w, `
  <entry>
    <title>%s</title>
    <id>%s</id>
    <updated>%s</updated>
    <author><name>%s</name></author>
    <content type="text">%s</content>
    <link rel="http://opds-spec.org/acquisition" href="%s://%s/api/raw?path=%s" type="%s"/>
  </entry>`,
			utils.EscapeXML(title),
			utils.EscapeXML(m.Path),
			time.Now().Format(time.RFC3339), // Ideally use modification time
			utils.EscapeXML(author),
			utils.EscapeXML(m.Path),
			scheme, host, strings.ReplaceAll(url.QueryEscape(m.Path), "+", "%20"),
			mimeType,
		)
	}

	fmt.Fprint(w, "\n</feed>")
}

func (c *ServeCmd) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		titles := make(map[string]bool)
		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				pls, err := queries.GetPlaylists(r.Context())
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
				slog.Error("Failed to fetch playlists", "db", dbPath, "error", err)
			}
		}

		uniqueTitles := make(models.PlaylistResponse, 0, len(titles))
		for t := range titles {
			uniqueTitles = append(uniqueTitles, t)
		}
		sort.Strings(uniqueTitles)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uniqueTitles)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
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

		playlistPath := "custom:" + utils.RandomString(12)

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				_, err := queries.InsertPlaylist(r.Context(), database.InsertPlaylistParams{
					Title: sql.NullString{String: req.Title, Valid: true},
					Path:  sql.NullString{String: playlistPath, Valid: true},
				})
				return err
			})
			if err != nil {
				slog.Error("Failed to insert playlist", "db", dbPath, "title", req.Title, "error", err)
			}
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

	if r.Method == http.MethodDelete {
		title := r.URL.Query().Get("title")
		if title == "" {
			http.Error(w, "Title required", http.StatusBadRequest)
			return
		}

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				// We need to find the ID by title first because DeletePlaylist takes ID
				pls, err := queries.GetPlaylists(r.Context())
				if err != nil {
					return err
				}
				for _, p := range pls {
					if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
						err = queries.DeletePlaylist(r.Context(), database.DeletePlaylistParams{
							ID:          p.ID,
							TimeDeleted: sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
						})
						if err != nil {
							return err
						}
					}
				}
				return nil
			})
			if err != nil {
				slog.Error("Failed to delete playlist", "db", dbPath, "title", title, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (c *ServeCmd) handlePlaylistItems(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		title := r.URL.Query().Get("title")
		if title == "" {
			http.Error(w, "Title required", http.StatusBadRequest)
			return
		}

		allMedia := make([]models.MediaWithDB, 0)
		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				pls, err := queries.GetPlaylists(r.Context())
				if err != nil {
					return err
				}

				var playlistID int64 = -1
				for _, p := range pls {
					if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
						playlistID = p.ID
						break
					}
				}

				if playlistID == -1 {
					return nil
				}

				items, err := queries.GetPlaylistItems(r.Context(), playlistID)
				if err != nil {
					return err
				}

				for _, item := range items {
					m := models.FromDB(database.Media{
						Path:            item.Path,
						FtsPath:         item.FtsPath,
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
						Type:            item.Type,
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
					})
					m.TrackNumber = models.NullInt64Ptr(item.TrackNumber)
					mw := models.MediaWithDB{
						Media: m,
						DB:    dbPath,
					}
					if c.hasFfmpeg {
						mw.Transcode = utils.GetTranscodeStrategy(m).NeedsTranscode
					}
					allMedia = append(allMedia, mw)
				}
				return nil
			})
			if err != nil {
				slog.Error("Failed to fetch playlist items", "db", dbPath, "title", title, "error", err)
			}
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allMedia)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
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

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				pls, err := queries.GetPlaylists(r.Context())
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
					return fmt.Errorf("playlist not found: %s", req.PlaylistTitle)
				}

				// Get max track number
				var maxTrack sql.NullInt64
				err = sqlDB.QueryRowContext(r.Context(), "SELECT MAX(track_number) FROM playlist_items WHERE playlist_id = ?", playlistID).Scan(&maxTrack)
				if err != nil {
					return err
				}

				trackNum := req.TrackNumber
				if trackNum == 0 {
					if maxTrack.Valid {
						trackNum = maxTrack.Int64 + 1
					} else {
						trackNum = 1
					}
				}

				return queries.AddPlaylistItem(r.Context(), database.AddPlaylistItemParams{
					PlaylistID:  playlistID,
					MediaPath:   req.MediaPath,
					TrackNumber: sql.NullInt64{Int64: trackNum, Valid: true},
				})
			})
			if err != nil {
				slog.Error("Failed to insert playlist item", "db", dbPath, "title", req.PlaylistTitle, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodDelete {
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

		for _, dbPath := range c.Databases {
			err := c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				pls, err := queries.GetPlaylists(r.Context())
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

				return queries.RemovePlaylistItem(r.Context(), database.RemovePlaylistItemParams{
					PlaylistID: playlistID,
					MediaPath:  req.MediaPath,
				})
			})
			if err != nil {
				slog.Error("Failed to delete playlist item", "db", dbPath, "title", req.PlaylistTitle, "path", req.MediaPath, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (c *ServeCmd) handleRSVP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Verify path in database
	found := false
	for _, dbPath := range c.Databases {
		c.execDB(r.Context(), dbPath, func(sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			_, err := queries.GetMediaByPathExact(r.Context(), path)
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
		return
	}

	wpm := 250
	if wpmStr := r.URL.Query().Get("wpm"); wpmStr != "" {
		if val, err := strconv.Atoi(wpmStr); err == nil && val > 0 {
			wpm = val
		}
	}

	text, err := utils.ExtractText(path)
	if err != nil {
		slog.Error("Text extraction failed", "path", path, "error", err)
		http.Error(w, fmt.Sprintf("Text extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	assContent, duration := utils.GenerateRSVPAss(text, wpm)
	if duration <= 0 {
		http.Error(w, "Empty text content", http.StatusBadRequest)
		return
	}

	// Create temp directory for ASS and WAV
	tmpDir, err := os.MkdirTemp("", "disco-rsvp-*")
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

	wavPath := filepath.Join(tmpDir, "audio.wav")
	if err := utils.GenerateTTS(text, wavPath, wpm); err != nil {
		slog.Warn("TTS generation failed", "error", err)
		wavPath = ""
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

	slog.Info("Starting RSVP stream", "path", path, "wpm", wpm, "duration", duration)

	cmd := exec.CommandContext(r.Context(), "ffmpeg", args...)
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		if r.Context().Err() == nil {
			slog.Error("FFmpeg RSVP streaming failed", "error", err)
		}
	}
}
