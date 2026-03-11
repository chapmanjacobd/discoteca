package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type SearchDBCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.DisplayFlags     `embed:""`
	models.PostActionFlags  `embed:""`

	Database string   `arg:"" required:"" help:"SQLite database file" type:"existingfile"`
	Table    string   `arg:"" required:"" help:"Table name (fuzzy matching supported)"`
	Search   []string `arg:"" required:"" help:"Search terms"`
}

func (c *SearchDBCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)

	sqlDB, err := db.Connect(c.Database)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// 1. Resolve table name
	tableName, err := c.getTableName(sqlDB)
	if err != nil {
		return err
	}

	// 2. Get searchable columns
	columns, err := c.getSearchableColumns(sqlDB, tableName)
	if err != nil {
		return err
	}

	// 3. Build search filters
	whereClauses, args := c.buildSearchFilters(columns)

	// 4. Handle Actions (Delete/MarkDeleted) or Print
	if c.DeleteRows {
		return c.deleteRows(sqlDB, tableName, whereClauses, args)
	} else if c.MarkDeleted {
		return c.markDeletedRows(sqlDB, tableName, whereClauses, args)
	}

	return c.printRows(sqlDB, tableName, whereClauses, args)
}

func (c *SearchDBCmd) getTableName(sqlDB *sql.DB) (string, error) {
	rows, err := sqlDB.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var allTables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		if name == c.Table {
			return name, nil // Exact match
		}
		// Skip internal/meta tables
		if strings.Contains(name, "_fts") || strings.HasPrefix(name, "sqlite_") {
			continue
		}
		allTables = append(allTables, name)
	}

	var matches []string
	for _, t := range allTables {
		if strings.HasPrefix(t, c.Table) {
			matches = append(matches, t)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous table name %q: matches %v", c.Table, matches)
	}

	return "", fmt.Errorf("table %q not found in %s", c.Table, c.Database)
}

func (c *SearchDBCmd) getSearchableColumns(sqlDB *sql.DB, table string) ([]string, error) {
	rows, err := sqlDB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		dtype = strings.ToUpper(dtype)
		if strings.Contains(dtype, "TEXT") || strings.Contains(dtype, "CHAR") || dtype == "" {
			columns = append(columns, name)
		}
	}
	return columns, nil
}

func (c *SearchDBCmd) buildSearchFilters(columns []string) ([]string, []any) {
	var whereClauses []string
	var args []any

	// Support for Search terms from command line
	if len(c.Search) > 0 {
		for _, term := range c.Search {
			var groupClauses []string
			pattern := term
			if !c.Exact {
				pattern = "%" + term + "%"
			}

			for _, col := range columns {
				if c.Exact {
					groupClauses = append(groupClauses, fmt.Sprintf("%s = ?", col))
				} else {
					groupClauses = append(groupClauses, fmt.Sprintf("%s LIKE ?", col))
				}
				args = append(args, pattern)
			}
			whereClauses = append(whereClauses, "("+strings.Join(groupClauses, " OR ")+")")
		}
	}

	// Support for GlobalFlags.Include/Exclude
	if len(c.Include) > 0 {
		for _, inc := range c.Include {
			var groupClauses []string
			pattern := "%" + inc + "%"
			for _, col := range columns {
				groupClauses = append(groupClauses, fmt.Sprintf("%s LIKE ?", col))
				args = append(args, pattern)
			}
			whereClauses = append(whereClauses, "("+strings.Join(groupClauses, " OR ")+")")
		}
	}

	if len(c.Exclude) > 0 {
		for _, exc := range c.Exclude {
			var groupClauses []string
			pattern := "%" + exc + "%"
			for _, col := range columns {
				groupClauses = append(groupClauses, fmt.Sprintf("%s NOT LIKE ?", col))
				args = append(args, pattern)
			}
			whereClauses = append(whereClauses, "("+strings.Join(groupClauses, " AND ")+")")
		}
	}

	return whereClauses, args
}

func (c *SearchDBCmd) deleteRows(sqlDB *sql.DB, table string, where []string, args []any) error {
	query := fmt.Sprintf("DELETE FROM %s", table)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	res, err := sqlDB.Exec(query, args...)
	if err != nil {
		return err
	}
	count, _ := res.RowsAffected()
	fmt.Printf("Deleted %d rows\n", count)
	return nil
}

func (c *SearchDBCmd) markDeletedRows(sqlDB *sql.DB, table string, where []string, args []any) error {
	now := time.Now().Unix()
	query := fmt.Sprintf("UPDATE %s SET time_deleted = ?", table)
	actualArgs := append([]any{now}, args...)

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	res, err := sqlDB.Exec(query, actualArgs...)
	if err != nil {
		return err
	}
	count, _ := res.RowsAffected()
	fmt.Printf("Marked %d rows as deleted\n", count)
	return nil
}

func (c *SearchDBCmd) printRows(sqlDB *sql.DB, table string, where []string, args []any) error {
	query := fmt.Sprintf("SELECT * FROM %s", table)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	if !c.All && c.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", c.Limit)
	}

	results, err := sqlDB.QueryContext(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("search query failed: %w", err)
	}
	defer results.Close()

	cols, _ := results.Columns()
	var allResults []map[string]any

	for results.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err := results.Scan(valuePtrs...); err != nil {
			return err
		}

		entry := make(map[string]any)
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				entry[col] = string(b)
			} else {
				entry[col] = val
			}
		}
		allResults = append(allResults, entry)
	}

	if c.JSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(allResults)
	}

	if len(allResults) > 0 {
		// Basic table print
		for _, res := range allResults {
			for k, v := range res {
				fmt.Printf("%s: %v\t", k, v)
			}
			fmt.Println()
		}
	}

	return nil
}
