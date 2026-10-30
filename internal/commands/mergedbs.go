package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type MergeDBsCmd struct {
	models.CoreFlags  `embed:""`
	models.MergeFlags `embed:""`

	TargetDB  string   `help:"Target SQLite database file"  required:"true" arg:""`
	SourceDBs []string `help:"Source SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *MergeDBsCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	if err := c.MergeFlags.AfterApply(); err != nil {
		return err
	}
	if len(c.WhereMedia) > 0 && c.Simulate {
		logging.Log.Info("Simulating merge with --where applied to the media table only")
	}

	var (
		targetConn *sql.DB
		err        error
	)
	if c.Simulate {
		logging.Log.Info("Simulating merge; target database will not be modified", "target", c.TargetDB)
	} else {
		targetConn, err = db.Connect(ctx, c.TargetDB)
		if err != nil {
			return fmt.Errorf("failed to connect to target DB %s: %w", c.TargetDB, err)
		}
		defer targetConn.Close()

		// Ensure target schema is initialized (if it's a new file)
		if err := db.InitDB(ctx, targetConn); err != nil {
			logging.Log.Warn(
				"Target DB initialization might have partially failed or it was already initialized",
				"error",
				err,
			)
		}
	}

	var failures []error
	for _, srcPath := range c.SourceDBs {
		logging.Log.Info("Merging database", "src", srcPath)
		if err := c.mergeDatabase(ctx, srcPath, targetConn); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", srcPath, err))
		}
	}

	return errors.Join(failures...)
}

func (c *MergeDBsCmd) mergeDatabase(ctx context.Context, srcPath string, targetConn *sql.DB) error {
	srcConn, err := db.Connect(ctx, srcPath)
	if err != nil {
		return fmt.Errorf("failed to connect to source DB %s: %w", srcPath, err)
	}
	defer srcConn.Close()

	tables, err := c.getTables(ctx, srcConn)
	if err != nil {
		return err
	}
	if err := c.validateWhereTables(tables); err != nil {
		return err
	}

	var failures []error
	for _, table := range tables {
		if !c.shouldProcessTable(table) {
			continue
		}
		logging.Log.Info("Merging table", "table", table)
		if err := c.mergeTable(ctx, srcConn, targetConn, table); err != nil {
			logging.Log.Error("Failed to merge table", "table", table, "error", err)
			failures = append(failures, fmt.Errorf("table %s: %w", table, err))
		}
	}

	return errors.Join(failures...)
}

func (c *MergeDBsCmd) validateWhereTables(tables []string) error {
	for _, table := range c.WhereTableNames() {
		if !slices.Contains(tables, table) {
			return fmt.Errorf("--where-table references %q, which is not a table in the source database", table)
		}
	}
	return nil
}

func (c *MergeDBsCmd) getTables(ctx context.Context, conn *sql.DB) ([]string, error) {
	rows, err := conn.QueryContext(
		ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '%_fts%'",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func (c *MergeDBsCmd) shouldProcessTable(table string) bool {
	if len(c.OnlyTables) > 0 {
		return slices.Contains(c.OnlyTables, table)
	}
	return true
}

func (c *MergeDBsCmd) mergeTable(ctx context.Context, srcConn, targetConn *sql.DB, table string) error {
	srcCols, err := c.getTableColumns(ctx, srcConn, table)
	if err != nil {
		return err
	}

	if c.Simulate {
		selectedCols := c.selectColumns(srcCols, nil)
		if len(selectedCols) == 0 {
			return fmt.Errorf("no columns selected for table %s", table)
		}
		return c.copyRows(ctx, copyRowsParams{
			srcConn:      srcConn,
			table:        table,
			selectedCols: selectedCols,
		})
	}

	targetCols, err := c.getTableColumns(ctx, targetConn, table)
	if err != nil {
		return err
	}
	if len(targetCols) == 0 {
		return fmt.Errorf("table %s does not exist in target", table)
	}

	selectedCols := c.selectColumns(srcCols, targetCols)
	if len(selectedCols) == 0 {
		return fmt.Errorf("no columns selected for table %s (source and target schemas do not overlap)", table)
	}

	insertQuery, err := c.buildInsertQuery(ctx, targetConn, table, selectedCols)
	if err != nil {
		return err
	}

	return c.copyRows(ctx, copyRowsParams{
		srcConn:      srcConn,
		targetConn:   targetConn,
		table:        table,
		selectedCols: selectedCols,
		insertQuery:  insertQuery,
	})
}

func (c *MergeDBsCmd) selectColumns(srcCols, targetCols []string) []string {
	selected := srcCols

	if c.OnlyTargetColumns && len(targetCols) > 0 {
		selected = c.filterToTarget(srcCols, targetCols)
	}

	if len(c.SkipColumns) > 0 {
		selected = c.filterSkipColumns(selected)
	}

	return selected
}

func (c *MergeDBsCmd) filterToTarget(srcCols, targetCols []string) []string {
	targetSet := make(map[string]bool)
	for _, col := range targetCols {
		targetSet[col] = true
	}

	var filtered []string
	for _, col := range srcCols {
		if targetSet[col] {
			filtered = append(filtered, col)
		}
	}
	return filtered
}

func (c *MergeDBsCmd) filterSkipColumns(cols []string) []string {
	skipSet := make(map[string]bool)
	for _, col := range c.SkipColumns {
		skipSet[col] = true
	}

	var filtered []string
	for _, col := range cols {
		if !skipSet[col] {
			filtered = append(filtered, col)
		}
	}
	return filtered
}

func (c *MergeDBsCmd) buildInsertQuery(
	ctx context.Context,
	targetConn *sql.DB,
	table string,
	selectedCols []string,
) (string, error) {
	pks := c.resolvePrimaryKeys(ctx, targetConn, table)
	baseQuery := c.buildBaseInsert(table, selectedCols)

	if !c.Upsert {
		return baseQuery, nil
	}

	if len(pks) == 0 {
		return "", fmt.Errorf(
			"--upsert requires a primary key for table %s; pass --primary-key or --business-key",
			table,
		)
	}
	upsertQuery := c.buildUpsertQuery(table, selectedCols, pks)
	if upsertQuery == "" {
		return "", fmt.Errorf(
			"--upsert cannot build a conflict clause for table %s: primary key columns %v are not all in the selected columns %v",
			table, pks, selectedCols,
		)
	}
	return upsertQuery, nil
}

func (c *MergeDBsCmd) resolvePrimaryKeys(ctx context.Context, targetConn *sql.DB, table string) []string {
	pks := c.PrimaryKeys
	if len(c.BusinessKeys) > 0 {
		pks = c.BusinessKeys
	}
	if c.Upsert && len(pks) == 0 && targetConn != nil {
		pks, _ = c.getPrimaryKeyColumns(ctx, targetConn, table)
	}
	return pks
}

func (c *MergeDBsCmd) buildBaseInsert(table string, selectedCols []string) string {
	placeholders := strings.Repeat("?, ", len(selectedCols)-1) + "?"

	if c.Ignore {
		return fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)",
			table, strings.Join(selectedCols, ", "), placeholders)
	}

	if !c.Upsert {
		return fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
			table, strings.Join(selectedCols, ", "), placeholders)
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(selectedCols, ", "), placeholders)
}

func (c *MergeDBsCmd) buildUpsertQuery(table string, selectedCols, pks []string) string {
	colSet := make(map[string]bool)
	for _, col := range selectedCols {
		colSet[col] = true
	}

	for _, pk := range pks {
		if !colSet[pk] {
			return ""
		}
	}

	updateParts := []string{}
	for _, col := range selectedCols {
		if !slices.Contains(pks, col) {
			updateParts = append(updateParts, fmt.Sprintf("%s=excluded.%s", col, col))
		}
	}

	if len(updateParts) == 0 {
		return ""
	}

	placeholders := strings.Repeat("?, ", len(selectedCols)-1) + "?"
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		table, strings.Join(selectedCols, ", "), placeholders,
		strings.Join(pks, ", "), strings.Join(updateParts, ", "))
}

type copyRowsParams struct {
	srcConn      *sql.DB
	targetConn   *sql.DB
	table        string
	selectedCols []string
	insertQuery  string
}

func (c *MergeDBsCmd) copyRows(ctx context.Context, p copyRowsParams) error {
	whereClause := c.WhereForTable(p.table, c.WhereMedia)
	if whereClause != "" {
		whereClause = " WHERE " + whereClause
	}
	selectQuery := fmt.Sprintf("SELECT %s FROM %s%s", strings.Join(p.selectedCols, ", "), p.table, whereClause)

	rows, err := p.srcConn.QueryContext(ctx, selectQuery)
	if err != nil {
		return fmt.Errorf("failed to select from source: %w", err)
	}
	defer rows.Close()

	if c.Simulate {
		count := 0
		for rows.Next() {
			count++
		}
		if err = rows.Err(); err != nil {
			return err
		}
		logging.Log.Info("Simulated merge", "table", p.table, "rows", count, "where", whereClause)
		return nil
	}

	tx, err := p.targetConn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, p.insertQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %w", err)
	}
	defer stmt.Close()

	count, err := c.executeInserts(ctx, rows, stmt, len(p.selectedCols))
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logging.Log.Info("Merged rows", "table", p.table, "count", count)
	return nil
}

func (c *MergeDBsCmd) executeInserts(ctx context.Context, rows *sql.Rows, stmt *sql.Stmt, colCount int) (int, error) {
	dest := make([]any, colCount)
	destPtrs := make([]any, len(dest))
	for i := range dest {
		destPtrs[i] = &dest[i]
	}

	count := 0
	for rows.Next() {
		if err := rows.Scan(destPtrs...); err != nil {
			return 0, err
		}
		if _, err := stmt.ExecContext(ctx, dest...); err != nil {
			return 0, fmt.Errorf("failed to exec insert: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func (c *MergeDBsCmd) getTableColumns(ctx context.Context, conn *sql.DB, table string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}

func (c *MergeDBsCmd) getPrimaryKeyColumns(ctx context.Context, conn *sql.DB, table string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pks []string
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		if pk > 0 {
			pks = append(pks, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pks, nil
}
