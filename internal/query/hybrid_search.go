package query

import (
	"regexp"
	"strings"
)

// HybridSearchQuery holds the split query components for hybrid FTS+LIKE search
type HybridSearchQuery struct {
	// FTS terms: individual words searched via FTS5 (works with detail=none)
	FTSTerms []string
	// Phrases: exact phrases searched via LIKE (trigram-optimized)
	Phrases []string
	// Original query for reference
	Original string
}

// ParseHybridSearchQuery splits a search query into FTS terms and phrase patterns
// Phrases are quoted strings like "exact phrase"
// Terms are unquoted words searched via FTS
func ParseHybridSearchQuery(query string) *HybridSearchQuery {
	result := &HybridSearchQuery{
		Original: query,
	}

	// Match quoted phrases: "..." or '...'
	phraseRegex := regexp.MustCompile(`["']([^"']+)["']`)

	// Extract all phrases first
	matches := phraseRegex.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 {
			phrase := strings.TrimSpace(match[1])
			if len(phrase) >= 3 {
				// Trigram index requires at least 3 characters
				result.Phrases = append(result.Phrases, phrase)
			}
		}
	}

	// Remove phrases from query to get remaining terms
	remaining := phraseRegex.ReplaceAllString(query, " ")

	// Clean up FTS operators that won't work with detail=none
	// Keep: OR, AND, NOT (basic boolean)
	// Remove: NEAR, phrase quotes, column filters
	remaining = strings.ReplaceAll(remaining, "(", " ")
	remaining = strings.ReplaceAll(remaining, ")", " ")
	remaining = strings.ReplaceAll(remaining, ":", " ")

	// Split into individual terms
	terms := strings.FieldsSeq(remaining)
	for term := range terms {
		term = strings.TrimSpace(term)
		upper := strings.ToUpper(term)

		// Keep boolean operators regardless of length
		if upper == "OR" || upper == "AND" || upper == "NOT" {
			result.FTSTerms = append(result.FTSTerms, upper)
			continue
		}

		// Skip very short terms (trigram needs 3+ chars)
		if len(term) < 3 {
			continue
		}

		result.FTSTerms = append(result.FTSTerms, term)
	}

	return result
}

// BuildFTSQuery constructs the FTS MATCH query from terms
// For trigram + detail=none, we use first 3 chars of each word for filtering
// This is a loose filter - actual matching is done by LIKE for phrases
func (h *HybridSearchQuery) BuildFTSQuery(joinOp string) string {
	if len(h.FTSTerms) == 0 {
		return ""
	}

	join := strings.ToUpper(strings.TrimSpace(joinOp))
	if join != "OR" {
		join = "AND"
	}

	var parts []string
	hasExplicitOperator := false
	for _, term := range h.FTSTerms {
		upper := strings.ToUpper(term)
		if upper == "OR" || upper == "AND" || upper == "NOT" {
			hasExplicitOperator = true
			parts = append(parts, upper)
			continue
		}

		// Use the first 3 characters as a trigram prefilter. The SQL layer
		// still applies the normal field predicates, so this is intentionally
		// a loose candidate filter.
		if len(term) > 3 {
			term = term[:3]
		}
		if term != "" {
			parts = append(parts, term)
		}
	}

	parts = cleanFTSOperators(parts)
	if len(parts) == 0 {
		return ""
	}
	if hasExplicitOperator {
		// Explicit boolean operators already carry their own joining semantics.
		return strings.Join(parts, " ")
	}
	return strings.Join(parts, " "+join+" ")
}

func cleanFTSOperators(parts []string) []string {
	isOperator := func(s string) bool {
		return s == "OR" || s == "AND" || s == "NOT"
	}

	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if isOperator(part) {
			// Leading/trailing or repeated operators would make the MATCH
			// expression invalid. Drop them rather than failing the request.
			if len(cleaned) == 0 || isOperator(cleaned[len(cleaned)-1]) {
				continue
			}
		}
		cleaned = append(cleaned, part)
	}
	for len(cleaned) > 0 && isOperator(cleaned[len(cleaned)-1]) {
		cleaned = cleaned[:len(cleaned)-1]
	}
	return cleaned
}

// HasFTSTerms returns true if the query contains FTS term searches
func (h *HybridSearchQuery) HasFTSTerms() bool {
	for _, term := range h.FTSTerms {
		if term != "OR" && term != "AND" && term != "NOT" {
			return true
		}
	}
	return false
}
