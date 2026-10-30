package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/categorize"
	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/textutil"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func (c *Server) HandleCategorizeKeywords(w http.ResponseWriter, r *http.Request) {
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

func (c *Server) HandleCategorizeDeleteCategory(w http.ResponseWriter, r *http.Request) {
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

func (c *Server) HandleCategorizeKeyword(w http.ResponseWriter, r *http.Request) {
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

func (c *Server) fetchExistingKeywords(ctx context.Context) map[string]bool {
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

func (c *Server) isMediaMatched(m models.MediaWithDB, compiled map[string][]*regexp.Regexp) bool {
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

func (c *Server) extractMediaWords(m models.MediaWithDB, fullPath bool) []string {
	var sentence string
	if fullPath {
		sentence = textutil.PathToTokenized(m.Path)
	} else {
		sentence = textutil.PathToSentence(m.Path)
	}
	words := textutil.ExtractWords(sentence)
	if m.Title != nil {
		words = append(words, textutil.ExtractWords(*m.Title)...)
	}
	return words
}

func (c *Server) calculateWordFrequencies(
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

func (c *Server) HandleCategorizeSuggest(w http.ResponseWriter, r *http.Request) {
	fullPath := r.URL.Query().Get("full_path") == "true"

	allMedia := c.fetchAllMedia(r.Context(), "categorize suggest")

	// Fetch existing keywords to filter them out
	existingKeywords := c.fetchExistingKeywords(r.Context())

	compiled := categorize.CompileRegexes(r.Context(), c.Databases)

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

func (c *Server) findMatchedCategories(m models.MediaWithDB, compiled map[string][]*regexp.Regexp) []string {
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

func (c *Server) mergeAndFormatCategories(existing *string, found []string) string {
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

func (c *Server) HandleCategorizeApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c.ReadOnly {
		http.Error(w, "Read-only mode", http.StatusForbidden)
		return
	}

	allMedia := c.fetchAllMedia(r.Context(), "categorize apply")

	compiled := categorize.CompileRegexes(r.Context(), c.Databases)

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
				logging.Log.Error("Failed to apply category", "db", m.DB, "path", m.Path, "error", err)
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
