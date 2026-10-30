package utils

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
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
		// Treat shell-style wildcards like SQL LIKE wildcards so the in-memory
		// filter and the SQL predicate accept the same patterns.
		likePattern := strings.ReplaceAll(pattern, "*", "%")
		likePattern = strings.ReplaceAll(likePattern, "?", "_")
		if CompareBlockStrings(likePattern, path) {
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

func NaturalLess(s1, s2 string) bool {
	n1, n2 := ExtractNumbers(s1), ExtractNumbers(s2)

	idx1, idx2 := 0, 0
	for idx1 < len(n1) && idx2 < len(n2) {
		if n1[idx1].IsNum && n2[idx2].IsNum {
			if n1[idx1].Num != n2[idx2].Num {
				return n1[idx1].Num < n2[idx2].Num
			}
		} else {
			if n1[idx1].Str != n2[idx2].Str {
				return n1[idx1].Str < n2[idx2].Str
			}
		}
		idx1++
		idx2++
	}

	return len(n1) < len(n2)
}

type Chunk struct {
	Str   string
	Num   int
	IsNum bool
}

func ExtractNumbers(s string) []Chunk {
	re := regexp.MustCompile(`\d+|\D+`)
	matches := re.FindAllString(s, -1)

	var chunks []Chunk
	for _, m := range matches {
		if num, err := strconv.Atoi(m); err == nil {
			chunks = append(chunks, Chunk{Num: num, IsNum: true})
		} else {
			chunks = append(chunks, Chunk{Str: strings.ToLower(m), IsNum: false})
		}
	}
	return chunks
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
	for _, part := range slices.Backward(parts) {
		val, _ := strconv.ParseFloat(part, 64)
		seconds += val * multiplier
		multiplier *= 60
	}
	return seconds
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

func EscapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// PathPatternLike translates the path patterns accepted by MatchesAny into a
// SQL LIKE pattern. Shell-style '*' and '?' become LIKE wildcards, a pattern
// without wildcards is treated as a prefix, and existing '%' wildcards are
// preserved.
func PathPatternLike(pattern string) string {
	var b strings.Builder
	hasWildcard := false
	for _, r := range pattern {
		switch r {
		case '*', '%':
			b.WriteByte('%')
			hasWildcard = true
		case '?':
			b.WriteByte('_')
			hasWildcard = true
		case '\\':
			b.WriteString(`\\`)
		case '_':
			b.WriteString(`\_`)
		default:
			b.WriteRune(r)
		}
	}
	if !hasWildcard {
		b.WriteByte('%')
	}
	return b.String()
}
