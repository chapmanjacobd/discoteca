package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func (c *ServeCmd) HandleCategorizeKeywords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type catKeywords struct {
		Category string   `json:"category"`
		Keywords []string `json:"keywords"`
	}

	data := make(map[string]map[string]bool)

	for _, dbPath := range c.Databases {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			rows, err := sqlDB.QueryContext(ctx, "SELECT category, keyword FROM custom_keywords")
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
			return rows.Err()
		})
	}

	results := make([]catKeywords, 0, len(data))
	for cat, kwSet := range data {
		kws := make([]string, 0, len(kwSet))
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

	sendJSON(w, http.StatusOK, results)
}

func (c *ServeCmd) HandleCategorizeDeleteCategory(w http.ResponseWriter, r *http.Request) {
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

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			res, err := sqlDB.ExecContext(ctx, "DELETE FROM custom_keywords WHERE category = ?", category)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
		},
	)
	sendWriteResult(w, http.StatusOK, affected, succeeded, failed, errs, "")
}

func (c *ServeCmd) HandleCategorizeKeyword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
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

		affected, succeeded, failed, errs := c.execWriteAcrossDBs(
			r.Context(),
			func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
				res, err := sqlDB.ExecContext(
					ctx,
					"DELETE FROM custom_keywords WHERE category = ? AND keyword = ?",
					req.Category,
					req.Keyword,
				)
				if err != nil {
					return 0, err
				}
				return res.RowsAffected()
			},
		)
		sendWriteResult(w, http.StatusOK, affected, succeeded, failed, errs, "")
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

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			res, err := sqlDB.ExecContext(
				ctx,
				"INSERT OR IGNORE INTO custom_keywords (category, keyword) VALUES (?, ?)",
				req.Category,
				req.Keyword,
			)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
		},
	)
	sendWriteResult(w, http.StatusOK, affected, succeeded, failed, errs, "")
}

func (c *ServeCmd) fetchAllMedia(ctx context.Context, purpose string) []models.MediaWithDB {
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
					Media: models.FromDB(m),
					DB:    dbPath,
				})
			}
			return nil
		})
		if err != nil {
			models.Log.Error("Failed to fetch media", "purpose", purpose, "error", err)
		}
	}
	return allMedia
}

func (c *ServeCmd) filterPlayable(allMedia []models.MediaWithDB, targetMediaType string) []models.MediaWithDB {
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

func (c *ServeCmd) HandleRandomClip(w http.ResponseWriter, r *http.Request) {
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

func (c *ServeCmd) fetchExistingKeywords(ctx context.Context) map[string]bool {
	existingKeywords := make(map[string]bool)
	for _, dbPath := range c.Databases {
		_ = c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			rows, err := sqlDB.QueryContext(ctx, "SELECT DISTINCT keyword FROM custom_keywords")
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
			return rows.Err()
		})
	}
	return existingKeywords
}

func (c *ServeCmd) isMediaMatched(m models.MediaWithDB, compiled map[string][]*regexp.Regexp) bool {
	pathAndTitle := m.Path
	if m.Title != nil {
		pathAndTitle += " " + *m.Title
	}

	for _, res := range compiled {
		for _, re := range res {
			if re.MatchString(pathAndTitle) {
				return true
			}
		}
	}
	return false
}

func (c *ServeCmd) extractMediaWords(m models.MediaWithDB, fullPath bool) []string {
	var sentence string
	if fullPath {
		sentence = utils.PathToTokenized(m.Path)
	} else {
		sentence = utils.PathToSentence(m.Path)
	}
	words := utils.ExtractWords(sentence)
	if m.Title != nil {
		words = append(words, utils.ExtractWords(*m.Title)...)
	}
	return words
}

func (c *ServeCmd) calculateWordFrequencies(
	allMedia []models.MediaWithDB,
	compiled map[string][]*regexp.Regexp,
	existingKeywords map[string]bool,
	fullPath bool,
) map[string]int {
	wordCounts := make(map[string]int)
	for _, m := range allMedia {
		// Skip files that already have categories assigned
		if m.Categories != nil && *m.Categories != "" {
			continue
		}

		if c.isMediaMatched(m, compiled) {
			continue
		}

		// Use a map to count each word only once per file
		uniqueWords := make(map[string]bool)
		words := c.extractMediaWords(m, fullPath)

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
	return wordCounts
}

func (c *ServeCmd) HandleCategorizeSuggest(w http.ResponseWriter, r *http.Request) {
	fullPath := r.URL.Query().Get("full_path") == "true"

	allMedia := c.fetchAllMedia(r.Context(), "categorize suggest")

	// Fetch existing keywords to filter them out
	existingKeywords := c.fetchExistingKeywords(r.Context())

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

	// We need to compile regexes first
	compiled := cmd.CompileRegexes(r.Context())

	wordCounts := c.calculateWordFrequencies(allMedia, compiled, existingKeywords, fullPath)

	type wordFreq struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}
	var freqs []wordFreq
	for word, count := range wordCounts {
		if count > 1 {
			freqs = append(freqs, wordFreq{Word: word, Count: count})
		}
	}

	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].Count > freqs[j].Count
	})

	limit := min(len(freqs), 100)
	sendJSON(w, http.StatusOK, freqs[:limit])
}

func (c *ServeCmd) findMatchedCategories(m models.MediaWithDB, compiled map[string][]*regexp.Regexp) []string {
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
	return foundCategories
}

func (c *ServeCmd) mergeAndFormatCategories(existing *string, found []string) string {
	merged := make(map[string]bool)
	if existing != nil && *existing != "" {
		parts := strings.SplitSeq(strings.Trim(*existing, ";"), ";")
		for e := range parts {
			if e != "" {
				merged[strings.TrimSpace(e)] = true
			}
		}
	}
	for _, f := range found {
		merged[f] = true
	}
	combined := make([]string, 0, len(merged))
	for k := range merged {
		combined = append(combined, k)
	}
	sort.Strings(combined)
	return ";" + strings.Join(combined, ";") + ";"
}

func (c *ServeCmd) HandleCategorizeApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	fullPath := r.URL.Query().Get("full_path") == "true"
	allMedia := c.fetchAllMedia(r.Context(), "categorize apply")

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
	compiled := cmd.CompileRegexes(r.Context())

	if len(compiled) == 0 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"count": 0}`)
		return
	}

	count := 0
	failed := 0
	for _, m := range allMedia {
		foundCategories := c.findMatchedCategories(m, compiled)

		if len(foundCategories) > 0 {
			newCategories := c.mergeAndFormatCategories(m.Categories, foundCategories)

			err := c.execDB(r.Context(), m.DB, func(ctx context.Context, sqlDB *sql.DB) error {
				queries := database.New(sqlDB)
				return queries.UpdateMediaCategories(ctx, database.UpdateMediaCategoriesParams{
					Categories: utils.ToNullString(newCategories),
					Path:       m.Path,
				})
			})
			if err == nil {
				count++
			} else {
				failed++
				models.Log.Error("Failed to apply category", "db", m.DB, "path", m.Path, "error", err)
			}
		}
	}

	status := http.StatusOK
	if failed > 0 {
		if count == 0 {
			status = http.StatusInternalServerError
		} else {
			status = http.StatusMultiStatus
		}
	}
	sendJSON(w, status, map[string]int{"count": count, "failed": failed})
}

func (c *ServeCmd) HandleRaw(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	dbParam := r.URL.Query().Get("db")

	if path == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	if c.isPathBlocklisted(path) {
		models.Log.Warn("Access denied: path is blocklisted", "path", path)
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

	models.Log.Debug("handleRaw request", "path", path)

	var m models.Media
	found := false

	localPath := path

	for _, dbPath := range dbs {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			dbMedia, err := queries.GetMediaByPathExact(ctx, path)
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
		models.Log.Warn("Access denied: file not in database", "path", path)
		http.Error(w, "Media not found in database", http.StatusNotFound)
		return
	}

	isLocal := utils.FileExists(localPath)
	if !isLocal {
		models.Log.Warn("File not found on disk", "path", path)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	strategy := utils.GetTranscodeStrategy(m)
	models.Log.Debug(
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
			models.Log.Error("ffmpeg not found in PATH, skipping transcoding", "path", path)
		}
	}

	models.Log.Debug("Serving local file", "path", localPath)
	http.ServeFile(w, r, localPath)
}

func (c *ServeCmd) HandleTrash(w http.ResponseWriter, r *http.Request) {
	flags := c.GetGlobalFlags()
	flags.OnlyDeleted = true
	flags.HideDeleted = false
	flags.All = true
	flags.SortBy = "time_deleted"
	flags.Reverse = true

	media, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		models.Log.Error("Trash query failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(len(media)))
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

func (c *ServeCmd) pathIsTrashed(ctx context.Context, path string) bool {
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
			models.Log.Error("Failed to validate trashed path", "db", dbPath, "path", path, "error", err)
		}
		if found {
			break
		}
	}
	return found
}

// dbHasTrashedPath reports whether an exact path is present in one database.
func (c *ServeCmd) dbHasTrashedPath(ctx context.Context, dbPath, path string) bool {
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
func (c *ServeCmd) emptyBinDelete(ctx context.Context, paths []string) EmptyBinResult {
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
		if utils.FileExists(p) {
			if err := os.Remove(p); err != nil {
				models.Log.Error("Failed to delete file", "path", p, "error", err)
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
			models.Log.Error("Failed to delete rows from DB", "db", dbPath, "error", err)
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

func (c *ServeCmd) HandleEmptyBin(w http.ResponseWriter, r *http.Request) {
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
			models.Log.Error("Trash query failed", "error", err)
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
	models.Log.Info("Bin emptied", "files_removed", result.Deleted, "failed", result.Failed, "skipped", result.Skipped)

	status := http.StatusOK
	if result.Failed > 0 && result.Deleted == 0 {
		status = http.StatusInternalServerError
	} else if result.Failed > 0 {
		status = http.StatusMultiStatus
	}
	sendJSON(w, status, result)
}

func (c *ServeCmd) HandleOPDS(w http.ResponseWriter, r *http.Request) {
	flags := c.GetGlobalFlags()
	flags.TextOnly = true
	flags.All = true

	media, err := query.MediaQuery(r.Context(), c.Databases, flags)
	if err != nil {
		models.Log.Error("OPDS query failed", "error", err)
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
	basePath := BasePathFromContext(r.Context())

	for _, m := range media {
		title := m.Stem()
		if m.Title != nil && *m.Title != "" {
			title = *m.Title
		}

		author := "Unknown"
		if m.Artist != nil && *m.Artist != "" {
			author = *m.Artist
		}

		mimeMediaType := "application/octet-stream"
		if m.MediaType != nil {
			mimeMediaType = *m.MediaType
		}

		fmt.Fprintf(w, `
  <entry>
    <title>%s</title>
    <id>%s</id>
    <updated>%s</updated>
    <author><name>%s</name></author>
    <content type="text">%s</content>
    <link rel="http://opds-spec.org/acquisition" href="%s://%s%s/api/raw?path=%s" type="%s"/>
  </entry>`,
			utils.EscapeXML(title),
			utils.EscapeXML(m.Path),
			time.Now().Format(time.RFC3339), // Ideally use modification time
			utils.EscapeXML(author),
			utils.EscapeXML(m.Path),
			scheme, host, basePath, strings.ReplaceAll(url.QueryEscape(m.Path), "+", "%20"),
			mimeMediaType,
		)
	}

	fmt.Fprint(w, "\n</feed>")
}

func (c *ServeCmd) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	titles := make(map[string]bool)
	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			pls, err := queries.GetPlaylists(ctx)
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
			models.Log.Error("Failed to fetch playlists", "db", dbPath, "error", err)
		}
	}

	uniqueTitles := make(models.PlaylistResponse, 0, len(titles))
	for t := range titles {
		uniqueTitles = append(uniqueTitles, t)
	}
	sort.Strings(uniqueTitles)

	sendJSON(w, http.StatusOK, uniqueTitles)
}

func (c *ServeCmd) handlePostPlaylist(w http.ResponseWriter, r *http.Request) {
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

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			if _, err = queries.InsertPlaylist(ctx, database.InsertPlaylistParams{
				Title: sql.NullString{String: req.Title, Valid: true},
				Path:  sql.NullString{String: playlistPath, Valid: true},
			}); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return 1, nil
		},
	)
	sendWriteResult(w, http.StatusCreated, affected, succeeded, failed, errs, "")
}

func (c *ServeCmd) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Title required", http.StatusBadRequest)
		return
	}

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			pls, err := queries.GetPlaylists(ctx)
			if err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			var deleted int64
			for _, p := range pls {
				if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
					if err = queries.DeletePlaylist(ctx, database.DeletePlaylistParams{
						ID:          p.ID,
						TimeDeleted: sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
					}); err != nil {
						_ = tx.Rollback()
						return 0, err
					}
					deleted++
				}
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return deleted, nil
		},
	)
	sendWriteResult(w, http.StatusOK, affected, succeeded, failed, errs, "")
}

func (c *ServeCmd) HandlePlaylists(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		c.handleGetPlaylists(w, r)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		c.handlePostPlaylist(w, r)
		return
	}

	if r.Method == http.MethodDelete {
		c.handleDeletePlaylist(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (c *ServeCmd) findPlaylistID(ctx context.Context, queries *database.Queries, title string) (int64, error) {
	pls, err := queries.GetPlaylists(ctx)
	if err != nil {
		return -1, err
	}

	for _, p := range pls {
		if p.Title.Valid && strings.EqualFold(p.Title.String, title) {
			return p.ID, nil
		}
	}
	return -1, nil
}

func (c *ServeCmd) fetchPlaylistMedia(
	ctx context.Context,
	queries *database.Queries,
	playlistID int64,
	dbPath string,
) ([]models.MediaWithDB, error) {
	items, err := queries.GetPlaylistItems(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	var allMedia []models.MediaWithDB
	for _, item := range items {
		m := models.FromDB(database.Media{
			Path:            item.Path,
			PathTokenized:   item.PathTokenized,
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
			MediaType:       item.MediaType,
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
	return allMedia, nil
}

func (c *ServeCmd) handleGetPlaylistItems(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Title required", http.StatusBadRequest)
		return
	}

	var allMedia []models.MediaWithDB
	for _, dbPath := range c.Databases {
		_ = c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			queries := database.New(sqlDB)
			playlistID, err := c.findPlaylistID(ctx, queries, title)
			if err != nil || playlistID == -1 {
				return err
			}

			items, err := c.fetchPlaylistMedia(ctx, queries, playlistID, dbPath)
			if err == nil {
				allMedia = append(allMedia, items...)
			}
			return err
		})
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

	sendJSON(w, http.StatusOK, allMedia)
}

func (c *ServeCmd) handlePostPlaylistItem(w http.ResponseWriter, r *http.Request) {
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

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			playlistID, err := c.findPlaylistID(ctx, queries, req.PlaylistTitle)
			if err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if playlistID == -1 {
				_ = tx.Rollback()
				return 0, nil
			}

			// Get max track number
			var maxTrack sql.NullInt64
			_ = tx.QueryRowContext(ctx, "SELECT MAX(track_number) FROM playlist_items WHERE playlist_id = ?", playlistID).
				Scan(&maxTrack)

			trackNum := req.TrackNumber
			if trackNum == 0 {
				if maxTrack.Valid {
					trackNum = maxTrack.Int64 + 1
				} else {
					trackNum = 1
				}
			}

			if err = queries.AddPlaylistItem(ctx, database.AddPlaylistItemParams{
				PlaylistID:  playlistID,
				MediaPath:   req.MediaPath,
				TrackNumber: sql.NullInt64{Int64: trackNum, Valid: true},
			}); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return 1, nil
		},
	)
	sendWriteResult(w, http.StatusOK, affected, succeeded, failed, errs, "")
}

func (c *ServeCmd) handleDeletePlaylistItem(w http.ResponseWriter, r *http.Request) {
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

	affected, succeeded, failed, errs := c.execWriteAcrossDBs(
		r.Context(),
		func(ctx context.Context, sqlDB *sql.DB) (int64, error) {
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			queries := database.New(tx)
			pls, err := queries.GetPlaylists(ctx)
			if err != nil {
				_ = tx.Rollback()
				return 0, err
			}

			var playlistID int64 = -1
			for _, p := range pls {
				if p.Title.Valid && strings.EqualFold(p.Title.String, req.PlaylistTitle) {
					playlistID = p.ID
					break
				}
			}

			if playlistID == -1 {
				_ = tx.Rollback()
				return 0, nil
			}

			if err = queries.RemovePlaylistItem(ctx, database.RemovePlaylistItemParams{
				PlaylistID: playlistID,
				MediaPath:  req.MediaPath,
			}); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			if err = tx.Commit(); err != nil {
				return 0, err
			}
			return 1, nil
		},
	)
	sendWriteResult(w, http.StatusOK, affected, succeeded, failed, errs, "")
}

func (c *ServeCmd) HandlePlaylistItems(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		c.handleGetPlaylistItems(w, r)
		return
	}

	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		c.handlePostPlaylistItem(w, r)
		return
	}

	if r.Method == http.MethodDelete {
		c.handleDeletePlaylistItem(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (c *ServeCmd) HandleRSVP(w http.ResponseWriter, r *http.Request) {
	path, wpm, err := c.validateAndExtractRSVPParams(w, r)
	if err != nil {
		return
	}

	text, err := utils.ExtractText(r.Context(), path)
	if err != nil {
		models.Log.Error("Text extraction failed", "path", path, "error", err)
		http.Error(w, fmt.Sprintf("Text extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	assContent, duration := utils.GenerateRSVPAss(text, wpm)
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

func (c *ServeCmd) validateAndExtractRSVPParams(w http.ResponseWriter, r *http.Request) (string, int, error) {
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

func (c *ServeCmd) prepareRSVPFiles(ctx context.Context, text, assContent string) (tmpDir, wavPath string, err error) {
	tmpDir, err = os.MkdirTemp("", "disco-rsvp-*")
	if err != nil {
		return "", "", errors.New("failed to create temp directory")
	}

	assPath := filepath.Join(tmpDir, "subtitles.ass")
	if writeErr := os.WriteFile(assPath, []byte(assContent), 0o644); writeErr != nil {
		return tmpDir, "", errors.New("failed to write subtitles")
	}

	wavPath = filepath.Join(tmpDir, "audio.wav")
	if ttsErr := utils.GenerateTTS(ctx, text, wavPath, 250); ttsErr != nil {
		models.Log.Warn("TTS generation failed", "error", ttsErr)
		wavPath = ""
	}

	return tmpDir, wavPath, nil
}

func (c *ServeCmd) streamRSVPVideo(
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

	models.Log.Info("Starting RSVP stream", "wpm", 250, "duration", duration)

	cmd := exec.CommandContext(r.Context(), "ffmpeg", args...)
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		if r.Context().Err() == nil {
			models.Log.Error("FFmpeg RSVP streaming failed", "error", err)
		}
	}
}
