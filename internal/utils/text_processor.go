package utils

import (
	"regexp"
	"sort"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

// LineSplitter splits a line into words using multiple regex patterns
func LineSplitter(regexs []*regexp.Regexp, line string) []string {
	words := []string{line}
	for _, rgx := range regexs {
		var newWords []string
		for _, word := range words {
			matches := rgx.FindAllString(word, -1)
			if matches != nil {
				newWords = append(newWords, matches...)
			}
		}
		words = newWords
	}
	return words
}

// CorpusStats returns word counts across all lines
func CorpusStats(corpus [][]string) map[string]int {
	stats := make(map[string]int)
	for _, words := range corpus {
		for _, word := range words {
			stats[word]++
		}
	}
	return stats
}

// WordSorter sorts words within a line based on various criteria
func WordSorter(flags models.GlobalFlags, corpusStats map[string]int, lineWords []string) []string {
	if len(flags.WordSorts) == 0 {
		return lineWords
	}

	// Helper for lastindex
	reversed := make([]string, len(lineWords))
	for i, w := range lineWords {
		reversed[len(lineWords)-1-i] = w
	}

	sort.SliceStable(lineWords, func(i, j int) bool {
		w1, w2 := lineWords[i], lineWords[j]
		for _, s := range flags.WordSorts {
			reverse := false
			if after, ok := strings.CutPrefix(s, "-"); ok {
				s = after
				reverse = true
			}

			var cmp int // -1 if w1 < w2, 1 if w1 > w2, 0 if equal
			switch s {
			case "skip":
				continue
			case "len":
				cmp = CompareInt(len(w1), len(w2))
			case "count":
				cmp = CompareInt(corpusStats[w1], corpusStats[w2])
			case "dup":
				cmp = CompareBool(corpusStats[w1] > 1, corpusStats[w2] > 1)
			case "unique":
				cmp = CompareBool(corpusStats[w1] == 1, corpusStats[w2] == 1)
			case "index":
				cmp = CompareInt(indexOf(lineWords, w1), indexOf(lineWords, w2))
			case "lastindex":
				cmp = CompareInt(indexOf(reversed, w1), indexOf(reversed, w2))
			case "linecount":
				cmp = CompareInt(countOf(lineWords, w1), countOf(lineWords, w2))
			case "alpha", "python":
				cmp = CompareString(w1, w2)
			case "natural", "natsort":
				if NaturalLess(w1, w2) {
					cmp = -1
				} else if NaturalLess(w2, w1) {
					cmp = 1
				} else {
					cmp = 0
				}
			default:
				continue
			}

			if cmp == 0 {
				continue
			}
			if reverse {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})

	return lineWords
}

// LineSorter sorts original lines based on their processed words
func LineSorter(
	flags models.GlobalFlags,
	corpusStats map[string]int,
	originalLines []string,
	corpus [][]string,
) []string {
	if len(flags.LineSorts) == 0 {
		return originalLines
	}

	type lineInfo struct {
		original string
		words    []string
	}
	infos := make([]lineInfo, len(originalLines))
	for i := range originalLines {
		infos[i] = lineInfo{originalLines[i], corpus[i]}
	}

	sort.SliceStable(infos, func(i, j int) bool {
		l1, l2 := infos[i], infos[j]
		for _, s := range flags.LineSorts {
			reverse := false
			if after, ok := strings.CutPrefix(s, "-"); ok {
				s = after
				reverse = true
			}

			var cmp int
			switch s {
			case "skip":
				continue
			case "line":
				cmp = CompareString(l1.original, l2.original)
			case "count":
				cmp = CompareInt(len(l1.words), len(l2.words))
			case "len":
				cmp = CompareInt(len(strings.Join(l1.words, "")), len(strings.Join(l2.words, "")))
			case "dup":
				cmp = CompareInt(SumDups(l1.words, corpusStats), SumDups(l2.words, corpusStats))
			case "unique":
				cmp = CompareInt(SumUnique(l1.words, corpusStats), SumUnique(l2.words, corpusStats))
			case "sum":
				cmp = CompareInt(SumCounts(l1.words, corpusStats), SumCounts(l2.words, corpusStats))
			case "dupmax":
				cmp = CompareInt(MaxCount(l1.words, corpusStats), MaxCount(l2.words, corpusStats))
			case "dupmin":
				cmp = CompareInt(MinCount(l1.words, corpusStats), MinCount(l2.words, corpusStats))
			case "dupavg", "dupmean":
				cmp = CompareFloat(avgCount(l1.words, corpusStats), avgCount(l2.words, corpusStats))
			case "dupmedian":
				cmp = CompareFloat(medianCount(l1.words, corpusStats), medianCount(l2.words, corpusStats))
			case "alpha", "python":
				cmp = CompareString(strings.Join(l1.words, " "), strings.Join(l2.words, " "))
			case "natural", "natsort":
				if NaturalLess(strings.Join(l1.words, " "), strings.Join(l2.words, " ")) {
					cmp = -1
				} else if NaturalLess(strings.Join(l2.words, " "), strings.Join(l1.words, " ")) {
					cmp = 1
				} else {
					cmp = 0
				}
			default:
				continue
			}

			if cmp == 0 {
				continue
			}
			if reverse {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})

	sortedLines := make([]string, len(infos))
	for i := range infos {
		sortedLines[i] = infos[i].original
	}
	return sortedLines
}

// TextProcessor orchestrates the splitting and sorting of text lines
func TextProcessor(flags models.GlobalFlags, lines []string) []string {
	if len(lines) == 0 {
		return lines
	}

	wordSorts := flags.WordSorts
	if len(wordSorts) == 0 {
		wordSorts = []string{"-dup", "count", "-len", "-lastindex", "alpha"}
	}
	lineSorts := flags.LineSorts
	if len(lineSorts) == 0 {
		lineSorts = []string{"-allunique", "alpha", "alldup", "dupmode", "line"}
	}

	// Create a new flags object with defaults for sorting functions
	processorFlags := flags
	processorFlags.WordSorts = wordSorts
	processorFlags.LineSorts = lineSorts

	stopWords := make(map[string]bool)
	for _, w := range flags.StopWords {
		stopWords[strings.ToLower(w)] = true
	}

	// Prepare regexs
	var regexs []*regexp.Regexp
	if len(flags.Regexs) == 0 {
		regexs = append(regexs, regexp.MustCompile(`\b\w\w+\b`))
	} else {
		for _, r := range flags.Regexs {
			re, err := regexp.Compile(r)
			if err != nil {
				models.Log.Error("Invalid regex", "pattern", r, "error", err)
				continue
			}
			regexs = append(regexs, re)
		}
	}

	corpus := make([][]string, len(lines))
	for i, line := range lines {
		// Remove protocol for processing
		processedLine := strings.TrimPrefix(line, "http://")
		processedLine = strings.TrimPrefix(processedLine, "https://")

		words := LineSplitter(regexs, processedLine)
		var filteredWords []string
		for _, w := range words {
			low := strings.ToLower(w)
			if !stopWords[low] {
				filteredWords = append(filteredWords, low)
			}
		}
		corpus[i] = filteredWords
	}

	corpusStats := CorpusStats(corpus)

	// Corpus filtering (if --unique or --duplicates flags are used)
	if flags.UniqueOnly != nil || flags.Duplicates != nil {
		var filteredLines []string
		var filteredCorpus [][]string
		for i, words := range corpus {
			if FilterCorpus(corpusStats, words, flags.UniqueOnly, flags.Duplicates) {
				filteredLines = append(filteredLines, lines[i])
				filteredCorpus = append(filteredCorpus, words)
			}
		}
		lines = filteredLines
		corpus = filteredCorpus
		corpusStats = CorpusStats(corpus) // Recompute stats for filtered corpus
	}

	// Word sorting within lines
	for i := range corpus {
		corpus[i] = WordSorter(processorFlags, corpusStats, corpus[i])
	}

	// Line sorting
	return LineSorter(processorFlags, corpusStats, lines, corpus)
}

func FilterCorpus(corpusStats map[string]int, words []string, unique, dups *bool) bool {
	if len(words) == 0 {
		return false
	}

	hasUnique := false
	hasDups := false
	allUnique := true
	allDups := true

	for _, w := range words {
		count := corpusStats[w]
		if count == 1 {
			hasUnique = true
		} else {
			allUnique = false
		}
		if count > 1 {
			hasDups = true
		} else {
			allDups = false
		}
	}

	// This logic matches the Python filter_corpus implementation's intent
	if unique != nil && dups == nil {
		if *unique {
			return hasUnique
		}
		return !hasUnique
	}
	if unique != nil && dups != nil {
		if *unique && !*dups {
			return allUnique
		}
		if !*unique && *dups {
			return allDups
		}
		return true // Other combinations return True in Python or are not explicitly handled
	}
	if unique == nil && dups != nil {
		if *dups {
			return hasDups
		}
		return !hasDups
	}

	return true
}

// Comparison helpers

func CompareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func CompareFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func CompareBool(a, b bool) int {
	if !a && b {
		return -1
	}
	if a && !b {
		return 1
	}
	return 0
}

func CompareString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}

func countOf(slice []string, val string) int {
	count := 0
	for _, v := range slice {
		if v == val {
			count++
		}
	}
	return count
}

func SumDups(words []string, stats map[string]int) int {
	sum := 0
	for _, w := range words {
		if stats[w] > 1 {
			sum++
		}
	}
	return sum
}

func SumUnique(words []string, stats map[string]int) int {
	sum := 0
	for _, w := range words {
		if stats[w] == 1 {
			sum++
		}
	}
	return sum
}

func SumCounts(words []string, stats map[string]int) int {
	sum := 0
	for _, w := range words {
		sum += stats[w]
	}
	return sum
}

func MaxCount(words []string, stats map[string]int) int {
	if len(words) == 0 {
		return 0
	}
	maxVal := -1
	for _, w := range words {
		if stats[w] > maxVal {
			maxVal = stats[w]
		}
	}
	return maxVal
}

func MinCount(words []string, stats map[string]int) int {
	if len(words) == 0 {
		return 0
	}
	minVal := 1 << 31
	for _, w := range words {
		if stats[w] < minVal {
			minVal = stats[w]
		}
	}
	return minVal
}

func avgCount(words []string, stats map[string]int) float64 {
	if len(words) == 0 {
		return 0
	}
	var counts []int
	for _, w := range words {
		counts = append(counts, stats[w])
	}
	return SafeMean(counts)
}

func medianCount(words []string, stats map[string]int) float64 {
	if len(words) == 0 {
		return 0
	}
	var counts []int
	for _, w := range words {
		counts = append(counts, stats[w])
	}
	return SafeMedian(counts)
}
