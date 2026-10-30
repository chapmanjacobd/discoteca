package commands

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
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

	Database string   `help:"SQLite database file"                  required:"true" arg:"" type:"existingfile"`
	Table    string   `help:"Table name (fuzzy matching supported)" required:"true" arg:""`
	Terms    []string `help:"Search terms"                          arg:"" name:"terms"`

	// AllRows explicitly opts a destructive action into affecting the whole
	// table when no predicate was supplied.
	AllRows bool `help:"Allow --delete-rows/--mark-deleted without a predicate (whole table)"`
}

func (c *SearchDBCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)

	sqlDB, _, err := db.ConnectWithInit(ctx, c.Database)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// 1. Resolve table name
	tableName, err := c.getTableName(ctx, sqlDB)
	if err != nil {
		return err
	}

	// 2. Get searchable columns
	columns, err := c.getSearchableColumns(ctx, sqlDB, tableName)
	if err != nil {
		return err
	}

	// 3. Build search filters
	whereClauses, args := c.buildSearchFilters(columns)

	// 4. Handle Actions (Delete/MarkDeleted) or Print
	if c.DeleteRows {
		return c.deleteRows(ctx, sqlDB, tableName, whereClauses, args)
	} else if c.MarkDeleted {
		return c.markDeletedRows(ctx, sqlDB, tableName, whereClauses, args)
	}

	return c.printRows(ctx, sqlDB, tableName, whereClauses, args)
}

func (c *SearchDBCmd) getTableName(ctx context.Context, sqlDB *sql.DB) (string, error) {
	rows, err := sqlDB.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table'")
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
	if err := rows.Err(); err != nil {
		return "", err
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

func (c *SearchDBCmd) getSearchableColumns(ctx context.Context, sqlDB *sql.DB, table string) ([]string, error) {
	rows, err := sqlDB.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}

func (c *SearchDBCmd) buildSearchFilters(columns []string) ([]string, []any) {
	var whereClauses []string
	var args []any

	// Positional terms plus --search (FilterFlags.Search) are both applied.
	terms := make([]string, 0, len(c.Terms)+len(c.Search))
	terms = append(terms, c.Terms...)
	terms = append(terms, c.Search...)

	// Support for Search terms from command line
	if len(terms) > 0 {
		for _, term := range terms {
			if len(columns) == 0 {
				continue
			}
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
	if len(c.Include) > 0 && len(columns) > 0 {
		for _, inc := range c.Include {
			groupClauses := make([]string, 0, len(columns))
			pattern := "%" + inc + "%"
			for _, col := range columns {
				groupClauses = append(groupClauses, fmt.Sprintf("%s LIKE ?", col))
				args = append(args, pattern)
			}
			whereClauses = append(whereClauses, "("+strings.Join(groupClauses, " OR ")+")")
		}
	}

	if len(c.Exclude) > 0 && len(columns) > 0 {
		for _, exc := range c.Exclude {
			groupClauses := make([]string, 0, len(columns))
			pattern := "%" + exc + "%"
			for _, col := range columns {
				groupClauses = append(groupClauses, fmt.Sprintf("%s NOT LIKE ?", col))
				args = append(args, pattern)
			}
			whereClauses = append(whereClauses, "("+strings.Join(groupClauses, " AND ")+")")
		}
	}

	// User-provided safety predicate from FilterFlags.Where.
	for _, clause := range c.Where {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		whereClauses = append(whereClauses, "("+clause+")")
	}

	return whereClauses, args
}

func buildPredicate(where []string) string {
	if len(where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(where, " AND ")
}

func (c *SearchDBCmd) requirePredicate(where []string, action string) error {
	if len(where) > 0 || c.AllRows {
		return nil
	}
	return fmt.Errorf(
		"refusing to %s every row in %q: supply --where/--search or pass --all-rows explicitly",
		action,
		c.Table,
	)
}

func (c *SearchDBCmd) countRows(
	ctx context.Context,
	sqlDB *sql.DB,
	table string,
	where []string,
	args []any,
) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", table, buildPredicate(where))
	var count int64
	if err := sqlDB.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (c *SearchDBCmd) deleteRows(ctx context.Context, sqlDB *sql.DB, table string, where []string, args []any) error {
	if err := c.requirePredicate(where, "delete"); err != nil {
		return err
	}

	predicate := buildPredicate(where)
	if c.Simulate {
		count, err := c.countRows(ctx, sqlDB, table, where, args)
		if err != nil {
			return err
		}
		fmt.Printf("[Dry-run] DELETE FROM %s%s would affect %d rows\n", table, predicate, count)
		return nil
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s%s", table, predicate), args...)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("Deleted %d rows\n", count)
	return nil
}

func (c *SearchDBCmd) markDeletedRows(
	ctx context.Context,
	sqlDB *sql.DB,
	table string,
	where []string,
	args []any,
) error {
	if err := c.requirePredicate(where, "mark deleted"); err != nil {
		return err
	}

	predicate := buildPredicate(where)
	if c.Simulate {
		count, err := c.countRows(ctx, sqlDB, table, where, args)
		if err != nil {
			return err
		}
		fmt.Printf("[Dry-run] UPDATE %s SET time_deleted = ...%s would affect %d rows\n", table, predicate, count)
		return nil
	}

	now := time.Now().Unix()
	query := fmt.Sprintf("UPDATE %s SET time_deleted = ?%s", table, predicate)
	actualArgs := append([]any{now}, args...)

	res, err := sqlDB.ExecContext(ctx, query, actualArgs...)
	if err != nil {
		return err
	}
	count, _ := res.RowsAffected()
	fmt.Printf("Marked %d rows as deleted\n", count)
	return nil
}

func (c *SearchDBCmd) printRows(ctx context.Context, sqlDB *sql.DB, table string, where []string, args []any) error {
	// Get column names explicitly
	colRows, err := sqlDB.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer colRows.Close()

	var columns []string
	for colRows.Next() {
		var cid int
		var name, colType string
		var notnull, pk int
		var dfltValue sql.NullString
		if scanErr := colRows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); scanErr != nil {
			return scanErr
		}
		columns = append(columns, name)
	}
	if err2 := colRows.Err(); err2 != nil {
		return err2
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), table)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	if !c.All && c.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", c.Limit)
	}

	results, err := sqlDB.QueryContext(ctx, query, args...)
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
		// ... (rest of loop)

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
	if err := results.Err(); err != nil {
		return err
	}

	if c.JSON {
		return fileutil.PrintJSON(allResults)
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
