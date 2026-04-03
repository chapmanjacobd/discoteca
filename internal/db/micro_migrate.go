package db

import (
	"database/sql"
	"fmt"
	"strings"
)

type ColumnDef struct {
	Table  string
	Column string
	Schema string
}

type IndexDef struct {
	Name string
	SQL  string
}

// EnsureColumns dynamically adds missing columns to a table
func EnsureColumns(db *sql.DB, cols []ColumnDef) error {
	for _, c := range cols {
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", c.Table))
		if err != nil {
			return fmt.Errorf("failed to check column %s in table %s: %w", c.Column, c.Table, err)
		}

		exists := false
		for rows.Next() {
			var cid int
			var name, dtype string
			var notnull, pk int
			var dfltValue any
			if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
				rows.Close()
				return err
			}
			if strings.EqualFold(name, c.Column) {
				exists = true
				break
			}
		}
		rows.Close()

		if !exists {
			if _, err := db.Exec(
				fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.Table, c.Column, c.Schema),
			); err != nil {
				return fmt.Errorf("failed to add column %s to table %s: %w", c.Column, c.Table, err)
			}
		}
	}
	return nil
}

// EnsureIndexes dynamically adds missing indexes
func EnsureIndexes(db *sql.DB, indexes []IndexDef) error {
	for _, idx := range indexes {
		if _, err := db.Exec(idx.SQL); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.Name, err)
		}
	}
	return nil
}
