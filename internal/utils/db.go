package utils

import (
	"fmt"
	"strings"
)

// MostSimilarSchema finds the table in existingTables that has the most columns in common with keys
func MostSimilarSchema(keys []string, existingTables map[string]map[string]string) string {
	if len(keys) == 0 || len(existingTables) == 0 {
		return ""
	}

	bestMatch := ""
	maxScore := 0

	for tableName, columns := range existingTables {
		score := 0
		for _, key := range keys {
			if _, ok := columns[key]; ok {
				score++
			}
		}

		if score > maxScore {
			maxScore = score
			bestMatch = tableName
		}
	}

	if maxScore == 0 {
		return ""
	}

	return bestMatch
}

// BlockDictsLikeSQL filters data by removing items that match any pattern in the blocklist
func BlockDictsLikeSQL(data, blocklist []map[string]any) []map[string]any {
	var res []map[string]any
	for _, item := range data {
		blocked := false
		for _, blockItem := range blocklist {
			matchAll := true
			for k, pattern := range blockItem {
				val, ok := item[k]
				if !ok {
					matchAll = false
					break
				}
				if !CompareBlockStrings(fmt.Sprintf("%v", pattern), fmt.Sprintf("%v", val)) {
					matchAll = false
					break
				}
			}
			if matchAll && len(blockItem) > 0 {
				blocked = true
				break
			}
		}
		if !blocked {
			res = append(res, item)
		}
	}
	return res
}

// AllowDictsLikeSQL filters data by keeping only items that match at least one pattern in the allowlist
func AllowDictsLikeSQL(data, allowlist []map[string]any) []map[string]any {
	var res []map[string]any
	for _, item := range data {
		allowed := false
		for _, allowItem := range allowlist {
			matchAll := true
			for k, pattern := range allowItem {
				val, ok := item[k]
				if !ok {
					matchAll = false
					break
				}
				if !CompareBlockStrings(fmt.Sprintf("%v", pattern), fmt.Sprintf("%v", val)) {
					matchAll = false
					break
				}
			}
			if matchAll && len(allowItem) > 0 {
				allowed = true
				break
			}
		}
		if allowed {
			res = append(res, item)
		}
	}
	return res
}

// ConstructSearchBindings returns a SQL where clause and a map of named bindings for inclusion and exclusion
func ConstructSearchBindings(include, exclude, columns []string, exact bool) (string, map[string]any) {
	var clauses []string
	bindings := make(map[string]any)

	if len(include) > 0 {
		var includeClauses []string
		for i, term := range include {
			var columnClauses []string
			key := fmt.Sprintf("S_include%d", i)
			pattern := term
			if !exact {
				pattern = "%" + term + "%"
			}
			bindings[key] = pattern

			for _, col := range columns {
				columnClauses = append(columnClauses, fmt.Sprintf("%s LIKE :%s", col, key))
			}
			if len(columnClauses) > 0 {
				includeClauses = append(includeClauses, "("+strings.Join(columnClauses, " OR ")+")")
			}
		}
		if len(includeClauses) > 0 {
			clauses = append(clauses, "("+strings.Join(includeClauses, " AND ")+")")
		}
	}

	if len(exclude) > 0 {
		var excludeClauses []string
		for i, term := range exclude {
			var columnClauses []string
			key := fmt.Sprintf("S_exclude%d", i)
			pattern := term
			if !exact {
				pattern = "%" + term + "%"
			}
			bindings[key] = pattern

			for _, col := range columns {
				columnClauses = append(columnClauses, fmt.Sprintf("COALESCE(%s,'') NOT LIKE :%s", col, key))
			}
			if len(columnClauses) > 0 {
				excludeClauses = append(excludeClauses, "("+strings.Join(columnClauses, " AND ")+")")
			}
		}
		if len(excludeClauses) > 0 {
			clauses = append(clauses, "AND ("+strings.Join(excludeClauses, " AND ")+")")
		}
	}

	return strings.Join(clauses, " AND "), bindings
}
