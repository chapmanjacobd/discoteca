package commands

import (
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type MergeDBsCmd struct {
	models.CoreFlags   `embed:""`
	models.FilterFlags `embed:""`
	models.MergeFlags  `embed:""`

	TargetDB  string   `help:"Target SQLite database file"  required:"" arg:""`
	SourceDBs []string `help:"Source SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *MergeDBsCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)

	targetConn, err := db.Connect(c.TargetDB)
	if err != nil {
		return fmt.Errorf("failed to connect to target DB %s: %w", c.TargetDB, err)
	}
	defer targetConn.Close()

	// Ensure target schema is initialized (if it's a new file)
	if err := db.InitDB(targetConn); err != nil {
		slog.Warn("Target DB initialization might have partially failed or it was already initialized", "error", err)
	}

	for _, srcPath := range c.SourceDBs {
		slog.Info("Merging database", "src", srcPath)
		if err := c.mergeDatabase(srcPath, targetConn); err != nil {
			return err
		}
	}

	return nil
}

func (c *MergeDBsCmd) mergeDatabase(srcPath string, targetConn *sql.DB) error {
	srcConn, err := db.Connect(srcPath)
	if err != nil {
		return fmt.Errorf("failed to connect to source DB %s: %w", srcPath, err)
	}
	defer srcConn.Close()

	tables, err := c.getTables(srcConn)
	if err != nil {
		return err
	}

	for _, table := range tables {
		if !c.shouldProcessTable(table) {
			continue
		}
		slog.Info("Merging table", "table", table)
		if err := c.mergeTable(srcConn, targetConn, table); err != nil {
			slog.Error("Failed to merge table", "table", table, "error", err)
			continue
		}
	}

	return nil
}

func (c *MergeDBsCmd) getTables(conn *sql.DB) ([]string, error) {
	rows, err := conn.Query(
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
	return tables, nil
}

func (c *MergeDBsCmd) shouldProcessTable(table string) bool {
	if len(c.OnlyTables) > 0 {
		found := slices.Contains(c.OnlyTables, table)
		if !found {
			return false
		}
	}
	return true
}

func (c *MergeDBsCmd) mergeTable(srcConn, targetConn *sql.DB, table string) error {
	srcCols, err := c.getTableColumns(srcConn, table)
	if err != nil {
		return err
	}

	targetCols, err := c.getTableColumns(targetConn, table)
	if err != nil {
		// Table might not exist in target. If not OnlyTargetColumns, we might want to create it?
		// Python sqlite-utils insert_all(alter=True) handles this.
		// For now, let's assume it should exist or we only care about existing target columns if requested.
		if c.OnlyTargetColumns {
			return fmt.Errorf("table %s does not exist in target", table)
		}
		// Basic "CREATE TABLE IF NOT EXISTS" logic is missing here if we wanted parity with alter=True
	}

	selectedCols := srcCols
	if c.OnlyTargetColumns && len(targetCols) > 0 {
		var filtered []string
		targetSet := make(map[string]bool)
		for _, col := range targetCols {
			targetSet[col] = true
		}
		for _, col := range srcCols {
			if targetSet[col] {
				filtered = append(filtered, col)
			}
		}
		selectedCols = filtered
	}

	// Filter SkipColumns
	if len(c.SkipColumns) > 0 {
		skipSet := make(map[string]bool)
		for _, col := range c.SkipColumns {
			skipSet[col] = true
		}
		var filtered []string
		for _, col := range selectedCols {
			if !skipSet[col] {
				filtered = append(filtered, col)
			}
		}
		selectedCols = filtered
	}

	if len(selectedCols) == 0 {
		slog.Warn("No columns selected for table", "table", table)
		return nil
	}

	// Determine PKs for UPSERT
	pks := c.PrimaryKeys
	if len(c.BusinessKeys) > 0 {
		pks = c.BusinessKeys
	}
	// If no PKs provided, try to find from schema if we are doing UPSERT
	if c.Upsert && len(pks) == 0 {
		pks, _ = c.getPrimaryKeyColumns(targetConn, table)
	}

	// Build SELECT query
	whereClause := ""
	if len(c.Where) > 0 {
		whereClause = " WHERE " + strings.Join(c.Where, " AND ")
	}
	selectQuery := fmt.Sprintf("SELECT %s FROM %s%s", strings.Join(selectedCols, ", "), table, whereClause)

	rows, err := srcConn.Query(selectQuery)
	if err != nil {
		return fmt.Errorf("failed to select from source: %w", err)
	}
	defer rows.Close()

	// Build INSERT query
	insertVerb := "INSERT"
	if c.Ignore {
		insertVerb = "INSERT OR IGNORE"
	} else if !c.Upsert {
		insertVerb = "INSERT OR REPLACE"
	}

	placeholders := make([]string, len(selectedCols))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	insertQuery := fmt.Sprintf("%s INTO %s (%s) VALUES (%s)",
		insertVerb, table, strings.Join(selectedCols, ", "), strings.Join(placeholders, ", "))

	if c.Upsert && len(pks) > 0 {
		// Verify all pks are in selectedCols
		allIn := true
		colSet := make(map[string]bool)
		for _, col := range selectedCols {
			colSet[col] = true
		}
		for _, pk := range pks {
			if !colSet[pk] {
				allIn = false
				break
			}
		}

		if allIn {
			updateParts := []string{}
			for _, col := range selectedCols {
				isPk := slices.Contains(pks, col)
				if !isPk {
					updateParts = append(updateParts, fmt.Sprintf("%s=excluded.%s", col, col))
				}
			}
			if len(updateParts) > 0 {
				insertQuery = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
					table, strings.Join(selectedCols, ", "), strings.Join(placeholders, ", "),
					strings.Join(pks, ", "), strings.Join(updateParts, ", "))
			}
		}
	}

	tx, err := targetConn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(insertQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %w", err)
	}
	defer stmt.Close()

	dest := make([]any, len(selectedCols))
	destPtrs := make([]any, len(selectedCols))
	for i := range dest {
		destPtrs[i] = &dest[i]
	}

	count := 0
	for rows.Next() {
		if err := rows.Scan(destPtrs...); err != nil {
			return err
		}
		if _, err := stmt.Exec(dest...); err != nil {
			return fmt.Errorf("failed to exec insert: %w", err)
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	slog.Info("Merged rows", "table", table, "count", count)
	return nil
}

func (c *MergeDBsCmd) getTableColumns(conn *sql.DB, table string) ([]string, error) {
	rows, err := conn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
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
	return cols, nil
}

func (c *MergeDBsCmd) getPrimaryKeyColumns(conn *sql.DB, table string) ([]string, error) {
	rows, err := conn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
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
	return pks, nil
}
