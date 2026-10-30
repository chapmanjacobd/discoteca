// Package categorize matches media paths against user-defined category
// keywords stored in each database.
package categorize

import (
	"context"
	"regexp"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
)

// CompileRegexes loads custom keywords from the given databases and compiles
// them into case-insensitive word-boundary matchers keyed by category.
func CompileRegexes(ctx context.Context, databases []string) map[string][]*regexp.Regexp {
	compiled := make(map[string][]*regexp.Regexp)

	for _, dbPath := range databases {
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
				// Use case-insensitive word boundary match for categorization.
				// This ensures "rock" matches "rock_concert" but not "brock".
				re, err := regexp.Compile(`(?i)(?:^|[^a-zA-Z0-9])` + regexp.QuoteMeta(kw) + `(?:$|[^a-zA-Z0-9])`)
				if err == nil {
					compiled[cat] = append(compiled[cat], re)
				}
			}
			return rows.Err()
		}()
		if err != nil {
			logging.Log.Error("failed processing keywords database", "db", dbPath, "error", err)
		}
	}

	return compiled
}
