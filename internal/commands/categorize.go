package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`

	Other    bool `help:"Analyze 'other' category to find potential new categories"`
	FullPath bool `help:"Use full path for categorization suggestions instead of just filename"`
}

func (c *CategorizeCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.BuildQueryGlobalFlags(models.BuildQueryOptions{
		Core:        c.CoreFlags,
		Query:       c.QueryFlags,
		PathFilter:  c.PathFilterFlags,
		Filter:      c.FilterFlags,
		MediaFilter: c.MediaFilterFlags,
		TimeFilter:  c.TimeFilterFlags,
		Deleted:     c.DeletedFlags,
		Sort:        models.SortFlags{},
		Display:     models.DisplayFlags{},
		FTS:         models.FTSFlags{},
	})
	flags.PostActionFlags = c.PostActionFlags

	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return err
	}
	media = query.FilterMedia(media, flags)

	if len(media) == 0 {
		return errors.New("no media found")
	}

	compiled := c.CompileRegexes(ctx)

	if c.Other {
		return c.mineCategories(media, compiled)
	}

	return c.applyCategories(ctx, media, compiled)
}

func (c *CategorizeCmd) CompileRegexes(ctx context.Context) map[string][]*regexp.Regexp {
	compiled := make(map[string][]*regexp.Regexp)

	// Load custom keywords from databases
	for _, dbPath := range c.Databases {
		err := func() error {
			sqlDB, err := db.Connect(ctx, dbPath)
			if err != nil {
				return err
			}
			defer sqlDB.Close()

			rows, err := sqlDB.QueryContext(ctx, "SELECT category, keyword FROM custom_keywords")
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var cat, kw string
				if err := rows.Scan(&cat, &kw); err != nil {
					continue
				}
				// Use case-insensitive word boundary match for categorization
				// This ensures "rock" matches "rock_concert" but not "brock"
				re, err := regexp.Compile(`(?i)(?:^|[^a-zA-Z0-9])` + regexp.QuoteMeta(kw) + `(?:$|[^a-zA-Z0-9])`)
				if err == nil {
					compiled[cat] = append(compiled[cat], re)
				}
			}
			return rows.Err()
		}()
		if err != nil {
			models.Log.Error("failed processing keywords database", "db", dbPath, "error", err)
		}
	}

	return compiled
}

func (c *CategorizeCmd) applyCategories(
	ctx context.Context,
	media []models.MediaWithDB,
	compiled map[string][]*regexp.Regexp,
) error {
	categorizedCount := 0
	for _, m := range media {
		newCategories, err := c.categorizeSingleMedia(ctx, m, compiled)
		if err != nil {
			models.Log.Error("Failed to categorize media", "path", m.Path, "error", err)
			continue
		}
		if newCategories != "" {
			categorizedCount++
		}
	}

	fmt.Printf("Processed %d files, categorized %d\n", len(media), categorizedCount)
	return nil
}

func (c *CategorizeCmd) categorizeSingleMedia(
	ctx context.Context,
	m models.MediaWithDB,
	compiled map[string][]*regexp.Regexp,
) (string, error) {
	foundCategories := []string{}
	var pathAndTitle string
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

	if len(foundCategories) == 0 {
		return "", nil
	}

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
	combined := make([]string, 0, len(merged))
	for k := range merged {
		combined = append(combined, k)
	}
	sort.Strings(combined)
	newCategories := ";" + strings.Join(combined, ";") + ";"

	if !c.Simulate {
		sqlDB, queries, err := db.ConnectWithInit(ctx, m.DB)
		if err != nil {
			return "", err
		}
		defer sqlDB.Close()

		if err := queries.UpdateMediaCategories(ctx, db.UpdateMediaCategoriesParams{
			Categories: utils.ToNullString(newCategories),
			Path:       m.Path,
		}); err != nil {
			return "", err
		}
	}

	if c.Verbose > 0 {
		fmt.Printf("Categorized: %s -> %s\n", m.Path, newCategories)
	}
	return newCategories, nil
}

func (c *CategorizeCmd) mineCategories(media []models.MediaWithDB, compiled map[string][]*regexp.Regexp) error {
	wordCounts, unmatchedCount := c.collectUnmatchedWords(media, compiled)
	c.printMinedKeywords(wordCounts, unmatchedCount)
	return nil
}

func (c *CategorizeCmd) collectUnmatchedWords(
	media []models.MediaWithDB,
	compiled map[string][]*regexp.Regexp,
) (map[string]int, int) {
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
			var sentence string
			if c.FullPath {
				sentence = utils.PathToTokenized(m.Path)
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
	return wordCounts, unmatchedCount
}

func (c *CategorizeCmd) printMinedKeywords(wordCounts map[string]int, unmatchedCount int) {
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
}
