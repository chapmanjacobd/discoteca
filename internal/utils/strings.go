package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// CompareBlockStrings implements SQL-like % wildcard matching
func CompareBlockStrings(pattern, value string) bool {
	pattern = strings.ToLower(pattern)
	value = strings.ToLower(value)

	if !strings.Contains(pattern, "%") {
		return strings.HasPrefix(value, pattern)
	}

	parts := strings.Split(pattern, "%")
	if len(parts) == 2 && parts[0] == "" && parts[1] == "" {
		return true // just "%"
	}

	// Simple cases
	if strings.HasPrefix(pattern, "%") && !strings.HasSuffix(pattern, "%") && len(parts) == 2 {
		return strings.HasSuffix(value, parts[1])
	}
	if !strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") && len(parts) == 2 {
		return strings.HasPrefix(value, parts[0])
	}
	if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") && len(parts) == 3 {
		return strings.Contains(value, parts[1])
	}

	// Complex case: translate to regex
	var regexPattern strings.Builder
	regexPattern.WriteString("^")
	for i, part := range parts {
		regexPattern.WriteString(regexp.QuoteMeta(part))
		if i < len(parts)-1 {
			regexPattern.WriteString(".*")
		}
	}
	regexPattern.WriteString("$")
	matched, _ := regexp.MatchString(regexPattern.String(), value)
	return matched
}

func MatchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if CompareBlockStrings(pattern, path) {
			return true
		}
	}
	return false
}

var cleanStringReplacer = strings.NewReplacer(
	"\\", " ",
	"/", " ",
	"(", " ",
	"-.", ".",
	" - ", " ",
	"- ", " ",
	" -", " ",
	" _ ", "_",
	" _", "_",
	"_ ", "_",
)

func CleanString(s string) string {
	s = RemoveTextInsideBrackets(s)
	s = html.UnescapeString(s)

	// Drop unwanted characters efficiently
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\x7f', '&', '%', '*', '$', '#', '!', '?', '|', '^', '\'', '"', ')', ':', '>', '<':
			return -1
		}
		return r
	}, s)

	s = cleanStringReplacer.Replace(s)
	s = RemoveConsecutives(s, []string{"."})
	s = RemoveConsecutiveWhitespace(s)

	return s
}

func RemoveTextInsideBrackets(s string) string {
	var result strings.Builder
	depth := 0
	for _, r := range s {
		if r == '(' || r == '[' || r == '{' {
			depth++
		} else if r == ')' || r == ']' || r == '}' {
			if depth > 0 {
				depth--
			}
		} else if depth == 0 {
			result.WriteRune(r)
		}
	}
	return result.String()
}

var tokenRe = regexp.MustCompile(`[/\\.\[\]\-\+(){}_&]`)

func PathToSentence(path string) string {
	s := filepath.Base(path)
	s = tokenRe.ReplaceAllString(s, " ")
	return CleanString(s)
}

func PathToTokenized(path string) string {
	s := tokenRe.ReplaceAllString(path, " ")
	return CleanString(s)
}

func IsGenericTitle(title string) bool {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return true
	}
	if strings.HasPrefix(title, "chapter") || strings.HasPrefix(title, "scene") {
		if len(title) < 12 {
			return true
		}
	}
	if strings.Contains(title, "untitled chapter") {
		return true
	}
	return IsTimecodeLike(title) || IsDigit(title)
}

func IsDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func IsTimecodeLike(s string) bool {
	if s == "" {
		return false
	}
	re := regexp.MustCompile(`^[\d:;._,\- ]+$`)
	return re.MatchString(s)
}

func RemoveConsecutiveWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func RemoveConsecutive(s, char string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(char) + "+")
	return re.ReplaceAllString(s, char)
}

func RemoveConsecutives(s string, chars []string) string {
	for _, char := range chars {
		s = RemoveConsecutive(s, char)
	}
	return s
}

func RemovePrefixes(s string, prefixes []string) string {
	for {
		changed := false
		for _, prefix := range prefixes {
			if after, ok := strings.CutPrefix(s, prefix); ok {
				s = after
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return s
}

func RemoveSuffixes(s string, suffixes []string) string {
	for {
		changed := false
		for _, suffix := range suffixes {
			if before, ok := strings.CutSuffix(s, suffix); ok {
				s = before
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return s
}

func Shorten(text string, maxWidth int) string {
	if runewidth.StringWidth(text) <= maxWidth {
		return text
	}

	ellipsis := "…"
	ellipsisWidth := runewidth.StringWidth(ellipsis)

	if maxWidth <= ellipsisWidth {
		return ellipsis
	}

	available := maxWidth - ellipsisWidth
	var truncated strings.Builder
	currentWidth := 0

	g := uniseg.NewGraphemes(text)
	for g.Next() {
		chunk := g.Str()
		chunkWidth := runewidth.StringWidth(chunk)
		if currentWidth+chunkWidth > available {
			break
		}
		truncated.WriteString(chunk)
		currentWidth += chunkWidth
	}

	return RemoveSuffixes(truncated.String(), []string{" ", "-", "."}) + "…"
}

func ShortenMiddle(text string, maxWidth int) string {
	if runewidth.StringWidth(text) <= maxWidth {
		return text
	}

	ellipsis := "..."
	ellipsisWidth := runewidth.StringWidth(ellipsis)

	if maxWidth <= ellipsisWidth {
		return ellipsis
	}

	available := maxWidth - ellipsisWidth
	leftWidth := available/2 + (available % 2)
	rightWidth := available / 2

	g := uniseg.NewGraphemes(text)
	var chunks []string
	for g.Next() {
		chunks = append(chunks, g.Str())
	}

	var left strings.Builder
	currentWidth := 0
	for _, chunk := range chunks {
		w := runewidth.StringWidth(chunk)
		if currentWidth+w > leftWidth {
			break
		}
		left.WriteString(chunk)
		currentWidth += w
	}

	var rightProper strings.Builder
	rightStart := len(chunks)
	currentWidth = 0
	for i := len(chunks) - 1; i >= 0; i-- {
		w := runewidth.StringWidth(chunks[i])
		if currentWidth+w > rightWidth {
			break
		}
		currentWidth += w
		rightStart = i
	}
	for i := rightStart; i < len(chunks); i++ {
		rightProper.WriteString(chunks[i])
	}

	return left.String() + ellipsis + rightProper.String()
}

func NaturalLess(s1, s2 string) bool {
	n1, n2 := extractNumbers(s1), extractNumbers(s2)

	idx1, idx2 := 0, 0
	for idx1 < len(n1) && idx2 < len(n2) {
		if n1[idx1].isNum && n2[idx2].isNum {
			if n1[idx1].num != n2[idx2].num {
				return n1[idx1].num < n2[idx2].num
			}
		} else {
			if n1[idx1].str != n2[idx2].str {
				return n1[idx1].str < n2[idx2].str
			}
		}
		idx1++
		idx2++
	}

	return len(n1) < len(n2)
}

type chunk struct {
	str   string
	num   int
	isNum bool
}

func extractNumbers(s string) []chunk {
	re := regexp.MustCompile(`\d+|\D+`)
	matches := re.FindAllString(s, -1)

	var chunks []chunk
	for _, m := range matches {
		if num, err := strconv.Atoi(m); err == nil {
			chunks = append(chunks, chunk{num: num, isNum: true})
		} else {
			chunks = append(chunks, chunk{str: strings.ToLower(m), isNum: false})
		}
	}
	return chunks
}

func UnParagraph(s string) string {
	s = RemoveConsecutiveWhitespace(s)
	// Replace smart quotes
	s = regexp.MustCompile(`[“”‘’]`).ReplaceAllString(s, "'")
	s = regexp.MustCompile(`[‛‟„]`).ReplaceAllString(s, "\"")
	s = strings.ReplaceAll(s, "…", "...")
	return StripEnclosingQuotes(s)
}

func Title(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			r := []rune(w)
			r[0] = unicode.ToUpper(r[0])
			words[i] = string(r)
		}
	}
	return strings.Join(words, " ")
}

func StripEnclosingQuotes(s string) string {
	if len(s) < 2 {
		return s
	}

	quotes := []string{"\"", "'", "＇", "‛", "‟", "＂", "‚", "〞", "〝", "〟", "„", "⹂", "❟", "❜", "❛", "❝", "❞"}
	for _, q := range quotes {
		if strings.HasPrefix(s, q) && strings.HasSuffix(s, q) {
			s = strings.TrimPrefix(s, q)
			s = strings.TrimSuffix(s, q)
			return StripEnclosingQuotes(s)
		}
	}

	pairs := [][]string{
		{"‘", "’"}, {"“", "”"}, {"❮", "❯"}, {"‹", "›"}, {"«", "»"},
	}
	for _, p := range pairs {
		if strings.HasPrefix(s, p[0]) && strings.HasSuffix(s, p[1]) {
			s = strings.TrimPrefix(s, p[0])
			s = strings.TrimSuffix(s, p[1])
			return StripEnclosingQuotes(s)
		}
		if strings.HasPrefix(s, p[1]) && strings.HasSuffix(s, p[0]) {
			s = strings.TrimPrefix(s, p[1])
			s = strings.TrimSuffix(s, p[0])
			return StripEnclosingQuotes(s)
		}
	}

	return s
}

func SplitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func Combine(vals ...any) string {
	var clean []string
	seen := make(map[string]bool)

	add := func(s string) {
		sub := strings.FieldsFunc(s, func(r rune) bool {
			return r == ',' || r == ';'
		})
		for _, part := range sub {
			part = RemoveConsecutiveWhitespace(part)
			low := strings.ToLower(part)
			if low == "unknown" || low == "none" || low == "und" || part == "" {
				continue
			}
			if !seen[low] {
				seen[low] = true
				clean = append(clean, part)
			}
		}
	}

	for _, v := range vals {
		switch val := v.(type) {
		case string:
			add(val)
		case []string:
			for _, s := range val {
				add(s)
			}
		case fmt.Stringer:
			add(val.String())
		default:
			add(fmt.Sprintf("%v", val))
		}
	}

	if len(clean) == 0 {
		return ""
	}
	return strings.Join(clean, ";")
}

func FromTimestampSeconds(s string) float64 {
	parts := strings.Split(s, ":")
	var seconds float64
	multiplier := 1.0
	for i := len(parts) - 1; i >= 0; i-- {
		val, _ := strconv.ParseFloat(parts[i], 64)
		seconds += val * multiplier
		multiplier *= 60
	}
	return seconds
}

func PartialStartswith(s string, list []string) (string, error) {
	if s == "" {
		return "", errors.New("empty string")
	}
	for _, item := range list {
		if strings.HasPrefix(item, s) {
			return item, nil
		}
	}
	return "", errors.New("no match found")
}

func GlobMatchAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}

func GlobMatchAll(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, _ := filepath.Match(pattern, path)
		if !matched {
			return false
		}
	}
	return true
}

func DurationShort(seconds int) string {
	return FormatDurationShort(seconds)
}

func RemoveExcessiveLinebreaks(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	re := regexp.MustCompile(`\n\s*\n\s*\n+`)
	s = re.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func LastChars(s string) string {
	parts := strings.Split(s, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return s
}

func ExtractWords(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]`)
	s = re.ReplaceAllString(s, " ")
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	return words
}

func SafeJSONLoads(s string) any {
	if s == "" {
		return nil
	}
	var res any
	if err := json.Unmarshal([]byte(s), &res); err != nil {
		return nil
	}
	return res
}

func LoadString(s string) any {
	if s == "" {
		return nil
	}
	if val := SafeJSONLoads(s); val != nil {
		return val
	}
	// Try a very basic "literal_eval" for Python-style dicts if they use single quotes
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		jsonS := strings.ReplaceAll(s, "'", "\"")
		if val := SafeJSONLoads(jsonS); val != nil {
			return val
		}
	}
	return s
}

// FtsQuote quotes search terms for FTS5 unless they already contain FTS operators
func FtsQuote(query []string) []string {
	ftsOperators := []string{" NOT ", " AND ", " OR ", "*", ":", "NEAR("}
	res := make([]string, len(query))
	for i, s := range query {
		hasOperator := false
		for _, op := range ftsOperators {
			if strings.Contains(s, op) {
				hasOperator = true
				break
			}
		}
		if hasOperator {
			res[i] = s
		} else {
			res[i] = `"` + s + `"`
		}
	}
	return res
}

func EscapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
