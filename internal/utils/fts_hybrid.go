package utils

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
	terms := strings.Fields(remaining)
	for _, term := range terms {
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
func (h *HybridSearchQuery) BuildFTSQuery(joinOp string) string {
	if len(h.FTSTerms) == 0 {
		return ""
	}

	var parts []string
	for _, term := range h.FTSTerms {
		// Pass through boolean operators
		if term == "OR" || term == "AND" || term == "NOT" {
			parts = append(parts, term)
			continue
		}
		// Quote terms for exact token matching
		quoted := FtsQuote([]string{term})[0]
		parts = append(parts, quoted)
	}

	if len(parts) == 0 {
		return ""
	}

	// Join with OR by default for broader matching, unless explicit operators present
	hasOperators := false
	for _, term := range parts {
		if term == "OR" || term == "AND" || term == "NOT" {
			hasOperators = true
			break
		}
	}

	if hasOperators {
		return strings.Join(parts, " ")
	}
	return strings.Join(parts, joinOp)
}

// HasPhrases returns true if the query contains phrase searches
func (h *HybridSearchQuery) HasPhrases() bool {
	return len(h.Phrases) > 0
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
