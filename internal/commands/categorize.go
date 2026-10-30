package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/categorize"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/textutil"
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
	logging.SetupLogging(c.Verbose)
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

	compiled := categorize.CompileRegexes(ctx, c.Databases)

	if c.Other {
		return c.mineCategories(media, compiled)
	}

	return c.applyCategories(ctx, media, compiled)
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
			logging.Log.Error("Failed to categorize media", "path", m.Path, "error", err)
			continue
		}
		if newCategories != "" {
			categorizedCount++
		}
	}

	fmt.Printf("Processed %d files, categorized %d\n", len(media), categorizedCount)
	return nil
}

func matchedCategories(
	m models.MediaWithDB,
	compiled map[string][]*regexp.Regexp,
	fullPath bool,
) []string {
	pathAndTitle := m.Path
	if !fullPath {
		pathAndTitle = filepath.Base(m.Path)
	}
	if m.Title != nil {
		pathAndTitle += " " + *m.Title
	}

	var found []string
	for cat, res := range compiled {
		for _, re := range res {
			if re.MatchString(pathAndTitle) {
				found = append(found, cat)
				break
			}
		}
	}
	return found
}

func (c *CategorizeCmd) categorizeSingleMedia(
	ctx context.Context,
	m models.MediaWithDB,
	compiled map[string][]*regexp.Regexp,
) (string, error) {
	foundCategories := matchedCategories(m, compiled, c.FullPath)
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
		if len(matchedCategories(m, compiled, c.FullPath)) > 0 {
			continue
		}

		unmatchedCount++
		var sentence string
		if c.FullPath {
			sentence = textutil.PathToTokenized(m.Path)
		} else {
			sentence = textutil.PathToSentence(m.Path)
		}
		words := textutil.ExtractWords(sentence)
		if m.Title != nil {
			words = append(words, textutil.ExtractWords(*m.Title)...)
		}

		for _, word := range words {
			if len(word) >= 4 {
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
