package commands

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type CategorizeCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.PostActionFlags  `embed:""`

	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`

	Other    bool `help:"Analyze 'other' category to find potential new categories"`
	FullPath bool `help:"Use full path for categorization suggestions instead of just filename"`
}

func (c *CategorizeCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		QueryFlags:       c.QueryFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		PostActionFlags:  c.PostActionFlags,
	}

	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		return err
	}
	media = query.FilterMedia(media, flags)

	if len(media) == 0 {
		return fmt.Errorf("no media found")
	}

	compiled := c.CompileRegexes()

	if c.Other {
		return c.mineCategories(media, compiled)
	}

	return c.applyCategories(media, compiled)
}

func (c *CategorizeCmd) CompileRegexes() map[string][]*regexp.Regexp {
	compiled := make(map[string][]*regexp.Regexp)

	// Load custom keywords from databases
	for _, dbPath := range c.Databases {
		sqlDB, err := db.Connect(dbPath)
		if err != nil {
			continue
		}
		rows, err := sqlDB.Query("SELECT category, keyword FROM custom_keywords")
		if err == nil {
			for rows.Next() {
				var cat, kw string
				if err := rows.Scan(&cat, &kw); err == nil {
					// Use case-insensitive word boundary match for categorization
					// This ensures "rock" matches "rock_concert" but not "brock"
					re, err := regexp.Compile(`(?i)(?:^|[^a-zA-Z0-9])` + regexp.QuoteMeta(kw) + `(?:$|[^a-zA-Z0-9])`)
					if err == nil {
						compiled[cat] = append(compiled[cat], re)
					}
				}
			}
			rows.Close()
		}
		sqlDB.Close()
	}

	return compiled
}

func (c *CategorizeCmd) applyCategories(media []models.MediaWithDB, compiled map[string][]*regexp.Regexp) error {
	categorizedCount := 0
	for _, m := range media {
		foundCategories := []string{}
		pathAndTitle := ""
		if c.FullPath {
			pathAndTitle = m.Path
		} else {
			pathAndTitle = filepath.Base(m.Path)
		}

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

			if !c.Simulate {
				sqlDB, err := db.Connect(m.DB)
				if err != nil {
					slog.Error("Failed to connect to database", "db", m.DB, "error", err)
					continue
				}
				queries := db.New(sqlDB)
				err = queries.UpdateMediaCategories(context.Background(), db.UpdateMediaCategoriesParams{
					Categories: utils.ToNullString(newCategories),
					Path:       m.Path,
				})
				sqlDB.Close()
				if err != nil {
					slog.Error("Failed to update categories", "path", m.Path, "error", err)
					continue
				}
			}

			if c.Verbose {
				fmt.Printf("Categorized: %s -> %s\n", m.Path, newCategories)
			}
			categorizedCount++
		}
	}

	fmt.Printf("Processed %d files, categorized %d\n", len(media), categorizedCount)
	return nil
}

func (c *CategorizeCmd) mineCategories(media []models.MediaWithDB, compiled map[string][]*regexp.Regexp) error {
	wordCounts := make(map[string]int)
	unmatchedCount := 0

	for _, m := range media {
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
			unmatchedCount++
			sentence := ""
			if c.FullPath {
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
				wordCounts[word]++
			}
		}
	}

	type wordFreq struct {
		word  string
		count int
	}
	var freqs []wordFreq
	for w, c := range wordCounts {
		if c > 1 {
			freqs = append(freqs, wordFreq{w, c})
		}
	}

	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].count > freqs[j].count
	})

	fmt.Printf("Mined %d unmatched files. Top potential keywords:\n", unmatchedCount)
	limit := min(len(freqs), 50)
	for i := range limit {
		fmt.Printf("%s: %d\n", freqs[i].word, freqs[i].count)
	}

	return nil
}
