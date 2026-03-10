//go:build !fts5

package commands

import (
	"database/sql"
	"strings"

	"github.com/chapmanjacobd/discotheque/internal/db"
)

func InitDB(sqlDB *sql.DB) error {
	schema := db.GetSchema()

	// Filter out FTS5 specific commands
	var filteredSchema strings.Builder
	lines := strings.SplitSeq(schema, ";")
	skipNextEnd := false
	for line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "FTS5") || strings.Contains(upper, "_FTS") {
			if strings.Contains(upper, "BEGIN") && !strings.Contains(upper, "END") {
				skipNextEnd = true
			}
			continue
		}
		if skipNextEnd && upper == "END" {
			skipNextEnd = false
			continue
		}
		filteredSchema.WriteString(trimmed)
		filteredSchema.WriteString(";")
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(filteredSchema.String()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return runMigrations(sqlDB)
}
