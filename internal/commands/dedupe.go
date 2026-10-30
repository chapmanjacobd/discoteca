package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/shellquote"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type DedupeCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.DedupeFlags      `embed:""`
	models.HashingFlags     `embed:""`

	Trash bool `help:"Trash duplicates instead of deleting them" group:"PostAction"`

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

type DedupeDuplicate struct {
	KeepPath      string
	DuplicatePath string
	DuplicateSize int64
}

// dedupeFilter is the SQL predicate shared by every candidate query so filters
// are honoured while rows are selected, not after the work is done.
type dedupeFilter struct {
	where string
	args  []any
}

func buildDedupeFilter(ctx context.Context, flags models.GlobalFlags) dedupeFilter {
	clauses, args := query.NewFilterBuilder(flags).BuildWhereClauses(ctx)
	return dedupeFilter{where: strings.Join(clauses, " AND "), args: args}
}

func (f dedupeFilter) appendTo(where string, args []any) (string, []any) {
	// The filter predicate is always placed first so its arguments come first.
	combinedArgs := make([]any, 0, len(f.args)+len(args))
	combinedArgs = append(combinedArgs, f.args...)
	combinedArgs = append(combinedArgs, args...)

	if f.where == "" {
		return where, args
	}
	if where == "" {
		return f.where, combinedArgs
	}
	return f.where + " AND " + where, combinedArgs
}

func (c *DedupeCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)

	if err := c.runMigrations(ctx); err != nil {
		return err
	}

	// Dedupe only considers live rows unless the caller explicitly asks for a
	// different deleted-row policy.
	if !c.OnlyDeleted && !c.HideDeleted {
		c.HideDeleted = true
	}

	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		DedupeFlags:      c.DedupeFlags,
		HashingFlags:     c.HashingFlags,
	}

	filter := buildDedupeFilter(ctx, flags)
	duplicates, err := c.collectDuplicates(ctx, flags, filter)
	if err != nil {
		return err
	}

	finalCandidates := c.filterDuplicates(duplicates)
	if len(finalCandidates) == 0 {
		models.Log.Info("No duplicates found")
		return nil
	}

	c.printSummary(finalCandidates)

	if c.Simulate || c.DryRun {
		fmt.Println("\nDry run: no files will be removed and no database rows will be updated.")
		for _, d := range finalCandidates {
			fmt.Printf("[Dry-run] keep %s, remove duplicate %s\n", d.KeepPath, d.DuplicatePath)
		}
		return nil
	}

	if !c.NoConfirm {
		if !c.confirmDeletion() {
			return nil
		}
	}

	return c.processDuplicates(ctx, finalCandidates, flags)
}

func (c *DedupeCmd) runMigrations(ctx context.Context) error {
	for _, dbPath := range c.Databases {
		sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
		if err != nil {
			return err
		}
		// Micro-migration for dedupe
		err = db.EnsureColumns(ctx, sqlDB, []db.ColumnDef{
			{Table: "media", Column: "is_deduped", Schema: "INTEGER DEFAULT 0"},
		})
		if err != nil {
			sqlDB.Close()
			return err
		}
		err = db.EnsureIndexes(ctx, sqlDB, []db.IndexDef{
			{
				Name: "idx_media_is_deduped",
				SQL:  "CREATE INDEX IF NOT EXISTS idx_media_is_deduped ON media(is_deduped) WHERE is_deduped = 1",
			},
			{
				Name: "idx_media_unprocessed",
				SQL:  "CREATE INDEX IF NOT EXISTS idx_media_unprocessed ON media(path) WHERE is_deduped = 0 OR is_deduped IS NULL",
			},
		})
		if err != nil {
			sqlDB.Close()
			return err
		}
		sqlDB.Close()
	}
	return nil
}

func (c *DedupeCmd) collectDuplicates(
	ctx context.Context,
	flags models.GlobalFlags,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	var duplicates []DedupeDuplicate
	for _, dbPath := range c.Databases {
		var dbDups []DedupeDuplicate
		var err error
		if c.Audio {
			dbDups, err = c.getMusicDuplicates(ctx, dbPath, filter)
		} else if c.ExtractorID {
			dbDups, err = c.getIDDuplicates(ctx, dbPath, filter)
		} else if c.TitleOnly {
			dbDups, err = c.getTitleDuplicates(ctx, dbPath, filter)
		} else if c.DurationOnly {
			dbDups, err = c.getDurationDuplicates(ctx, dbPath, filter)
		} else if c.Filesystem {
			dbDups, err = c.getFSDuplicates(ctx, dbPath, flags, filter)
		} else {
			return nil, errors.New("profile not set. Use --audio, --id, --title, --duration, or --fs")
		}

		if err != nil {
			return nil, err
		}
		duplicates = append(duplicates, dbDups...)
	}
	return duplicates, nil
}

func (c *DedupeCmd) filterDuplicates(duplicates []DedupeDuplicate) []DedupeDuplicate {
	metric := metrics.NewSorensenDice()
	var finalCandidates []DedupeDuplicate
	seenDuplicates := make(map[string]bool)

	for _, d := range duplicates {
		if seenDuplicates[d.DuplicatePath] || d.KeepPath == d.DuplicatePath {
			continue
		}

		if c.CompareDirs && filepath.Dir(d.KeepPath) != filepath.Dir(d.DuplicatePath) {
			continue
		}

		if c.Dirname {
			if strutil.Similarity(
				filepath.Dir(d.KeepPath),
				filepath.Dir(d.DuplicatePath),
				metric,
			) < c.MinSimilarityRatio {

				continue
			}
		}

		if c.Basename {
			if strutil.Similarity(
				filepath.Base(d.KeepPath),
				filepath.Base(d.DuplicatePath),
				metric,
			) < c.MinSimilarityRatio {

				continue
			}
		}

		// Check if keep path still exists
		if !utils.FileExists(d.KeepPath) {
			continue
		}
		if c.Exists && !utils.FileExists(d.DuplicatePath) {
			continue
		}

		finalCandidates = append(finalCandidates, d)
		seenDuplicates[d.DuplicatePath] = true
	}
	return finalCandidates
}

func (c *DedupeCmd) printSummary(finalCandidates []DedupeDuplicate) {
	var totalSavings int64
	for _, d := range finalCandidates {
		totalSavings += d.DuplicateSize
		fmt.Printf("Keep: %s\n  Dup: %s (%s)\n", d.KeepPath, d.DuplicatePath, utils.FormatSize(d.DuplicateSize))
	}
	fmt.Printf("\nApprox. space savings: %s (%d files)\n", utils.FormatSize(totalSavings), len(finalCandidates))
}

func (c *DedupeCmd) confirmDeletion() bool {
	fmt.Print("\nDelete duplicates? [y/N] ")
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}

func (c *DedupeCmd) processDuplicates(
	ctx context.Context,
	finalCandidates []DedupeDuplicate,
	flags models.GlobalFlags,
) error {
	models.Log.Info("Deleting duplicates...")
	removed := 0
	var failures []error
	for _, d := range finalCandidates {
		if err := c.removeDuplicate(ctx, d, flags); err != nil {
			failures = append(failures, fmt.Errorf("failed to remove %s: %w", d.DuplicatePath, err))
			models.Log.Error("Failed to remove duplicate", "path", d.DuplicatePath, "error", err)
			continue
		}

		if err := c.updateDatabaseAfterDedupe(ctx, d, flags); err != nil {
			failures = append(failures, fmt.Errorf("failed to update database for %s: %w", d.DuplicatePath, err))
			continue
		}
		removed++
	}

	fmt.Printf("\nRemoved %d duplicate(s)\n", removed)
	if len(failures) > 0 {
		fmt.Printf("Failed to remove %d duplicate(s)\n", len(failures))
		return fmt.Errorf("dedupe incomplete: %d of %d duplicates failed: %w",
			len(failures), len(finalCandidates), errors.Join(failures...))
	}
	return nil
}

// removeDuplicate performs one duplicate-removal action and verifies that the
// duplicate no longer exists before the database is updated.
func (c *DedupeCmd) removeDuplicate(
	ctx context.Context,
	d DedupeDuplicate,
	flags models.GlobalFlags,
) error {
	verifyGone := true
	switch {
	case c.DedupeCmd != "":
		quotedDup := shellquote.ShellQuote(d.DuplicatePath)
		quotedKeep := shellquote.ShellQuote(d.KeepPath)
		cmdStr := strings.ReplaceAll(c.DedupeCmd, "{}", quotedDup)
		// rmlint style is cmd duplicate keep
		if err := exec.CommandContext(ctx, "bash", "-c", cmdStr+" "+quotedDup+" "+quotedKeep).Run(); err != nil {
			return err
		}
	case c.Trash:
		if err := utils.Trash(ctx, flags, d.DuplicatePath); err != nil {
			return err
		}
		// Trash is started asynchronously. A successful start means the file
		// was handed to the trash command, so do not require it to disappear
		// before updating the database.
		verifyGone = false
	default:
		if err := os.Remove(d.DuplicatePath); err != nil {
			return err
		}
	}

	if verifyGone && utils.FileExists(d.DuplicatePath) {
		return fmt.Errorf("removal reported success but %s still exists", d.DuplicatePath)
	}
	return nil
}

func (c *DedupeCmd) updateDatabaseAfterDedupe(
	ctx context.Context,
	d DedupeDuplicate,
	flags models.GlobalFlags,
) error {
	var dbFailures []error
	for _, dbPath := range c.Databases {
		if err := c.updateDatabase(ctx, dbPath, d, flags); err != nil {
			dbFailures = append(dbFailures, fmt.Errorf("%s: %w", dbPath, err))
		}
	}
	return errors.Join(dbFailures...)
}

func (c *DedupeCmd) updateDatabase(
	ctx context.Context,
	dbPath string,
	d DedupeDuplicate,
	flags models.GlobalFlags,
) error {
	sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var dbErrs []string

	// Mark duplicate as deleted only if it was not already marked, so a row is
	// transitioned to deleted exactly once.
	if _, err := tx.ExecContext(
		ctx,
		"UPDATE media SET time_deleted = unixepoch() WHERE path = ? AND COALESCE(time_deleted, 0) = 0",
		d.DuplicatePath,
	); err != nil {
		dbErrs = append(dbErrs, fmt.Sprintf("failed to mark duplicate as deleted: %v", err))
	}

	// is_deduped marks the keep candidate as processed by dedupe. It does not
	// mean the row itself was removed.
	if _, err := tx.ExecContext(
		ctx,
		"UPDATE media SET is_deduped = 1 WHERE path = ?",
		d.KeepPath,
	); err != nil {
		dbErrs = append(dbErrs, fmt.Sprintf("failed to mark keep file as deduped: %v", err))
	}

	// Update hash if not already set
	if d.DuplicateSize > 0 {
		h, err2 := utils.SampleHashFile(d.KeepPath, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
		if err2 == nil && h != "" {
			if _, err := tx.ExecContext(
				ctx,
				"UPDATE media SET fasthash = ? WHERE path = ?",
				h,
				d.KeepPath,
			); err != nil {
				dbErrs = append(dbErrs, fmt.Sprintf("failed to update fasthash: %v", err))
			}
		}
		h, err2 = utils.FullHashFile(d.KeepPath)
		if err2 == nil && h != "" {
			if _, err := tx.ExecContext(
				ctx,
				"UPDATE media SET sha256 = ? WHERE path = ?",
				h,
				d.KeepPath,
			); err != nil {
				dbErrs = append(dbErrs, fmt.Sprintf("failed to update sha256: %v", err))
			}
		}
	}

	if len(dbErrs) > 0 {
		return errors.New(strings.Join(dbErrs, "; "))
	}

	return tx.Commit()
}

func (c *DedupeCmd) getDuplicatesBy(
	ctx context.Context,
	dbPath, groupByCols, whereClause string,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	where, args := filter.appendTo(whereClause, nil)

	queryStr := fmt.Sprintf(`
		SELECT %s, COUNT(*) as count
		FROM media
		WHERE %s
		GROUP BY %s
		HAVING count > 1
	`, groupByCols, where, groupByCols)

	rows, err := sqlDB.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	cols := strings.Split(groupByCols, ",")
	for rows.Next() {
		values := make([]any, len(cols)+1)
		for i := range values {
			values[i] = new(any)
		}
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		groupDups, err := c.processDuplicateGroup(ctx, sqlDB, cols, values, filter)
		if err != nil {
			return nil, err
		}
		dups = append(dups, groupDups...)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dups, nil
}

type dedupeItem struct {
	path     string
	size     int64
	duration float64
}

func (c *DedupeCmd) processDuplicateGroup(
	ctx context.Context,
	sqlDB *sql.DB,
	cols []string,
	values []any,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	whereParts := make([]string, len(cols))
	args := make([]any, 0, len(cols)+len(filter.args))
	for i, col := range cols {
		whereParts[i] = strings.TrimSpace(col) + " = ?"
		if val, ok := values[i].(*any); ok {
			args = append(args, *val)
		}
	}
	where, args := filter.appendTo(strings.Join(whereParts, " AND "), args)

	groupQuery := fmt.Sprintf(`
		SELECT path, COALESCE(size, 0), COALESCE(duration, 0)
		FROM media
		WHERE %s
		ORDER BY COALESCE(is_deduped, 0) DESC, size DESC, time_modified DESC
	`, where)

	gRows, err := sqlDB.QueryContext(ctx, groupQuery, args...)
	if err != nil {
		return nil, err
	}
	defer gRows.Close()

	var items []dedupeItem
	for gRows.Next() {
		var i dedupeItem
		if err := gRows.Scan(&i.path, &i.size, &i.duration); err == nil {
			items = append(items, i)
		}
	}
	if err := gRows.Err(); err != nil {
		return nil, err
	}

	if len(items) < 2 {
		return nil, nil
	}

	var dups []DedupeDuplicate
	keep := items[0]
	for _, dup := range items[1:] {
		// Basic duration check
		if keep.duration > 0 && dup.duration > 0 && math.Abs(keep.duration-dup.duration) > 8 {
			continue
		}
		dups = append(dups, DedupeDuplicate{
			KeepPath:      keep.path,
			DuplicatePath: dup.path,
			DuplicateSize: dup.size,
		})
	}
	return dups, nil
}

func (c *DedupeCmd) getMusicDuplicates(
	ctx context.Context,
	dbPath string,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(ctx, dbPath, "title, artist, album", "title != '' AND artist != ''", filter)
}

func (c *DedupeCmd) getIDDuplicates(
	ctx context.Context,
	dbPath string,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(ctx, dbPath, "webpath", "webpath != ''", filter)
}

func (c *DedupeCmd) getTitleDuplicates(
	ctx context.Context,
	dbPath string,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(ctx, dbPath, "title", "title != ''", filter)
}

func (c *DedupeCmd) getDurationDuplicates(
	ctx context.Context,
	dbPath string,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(ctx, dbPath, "duration", "duration > 0", filter)
}

func (c *DedupeCmd) getFSDuplicates(
	ctx context.Context,
	dbPath string,
	flags models.GlobalFlags,
	filter dedupeFilter,
) ([]DedupeDuplicate, error) {
	sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	where, args := filter.appendTo("size > 0", nil)

	// 1. Group by size in SQL
	queryStr := fmt.Sprintf(`
		SELECT size, COUNT(*) as count
		FROM media
		WHERE %s
		GROUP BY size
		HAVING count > 1
	`, where)
	rows, err := sqlDB.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var size int64
		var count int
		if err := rows.Scan(&size, &count); err != nil {
			return nil, err
		}
		models.Log.Debug("Found potential duplicates by size", "size", size, "count", count)

		groupDups := c.processFSSizeGroup(ctx, sqlDB, size, flags, filter)
		dups = append(dups, groupDups...)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dups, nil
}

type pathInfo struct {
	path      string
	fasthash  string
	sha256    string
	isDeduped bool
}

func (c *DedupeCmd) processFSSizeGroup(
	ctx context.Context,
	sqlDB *sql.DB,
	size int64,
	flags models.GlobalFlags,
	filter dedupeFilter,
) []DedupeDuplicate {
	where, args := filter.appendTo("size = ?", []any{size})
	gRows, err := sqlDB.QueryContext(
		ctx,
		"SELECT path, COALESCE(fasthash, ''), COALESCE(sha256, ''), COALESCE(is_deduped, 0) "+
			"FROM media WHERE "+where,
		args...,
	)
	if err != nil {
		return nil // continue
	}
	defer gRows.Close()

	var paths []pathInfo
	for gRows.Next() {
		var p pathInfo
		var deduped int
		if err := gRows.Scan(&p.path, &p.fasthash, &p.sha256, &deduped); err == nil {
			p.isDeduped = deduped == 1
			paths = append(paths, p)
		}
	}
	if err := gRows.Err(); err != nil {
		return nil
	}

	if len(paths) < 2 {
		return nil
	}

	return c.groupByHashes(paths, size, flags)
}

func (c *DedupeCmd) groupByFastHash(
	paths []pathInfo,
	flags models.GlobalFlags,
) map[string][]pathInfo {
	fastHashGroups := make(map[string][]pathInfo)
	for _, p := range paths {
		if p.fasthash == "" {
			h, err := utils.SampleHashFile(p.path, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
			if err != nil || h == "" {
				continue
			}
			p.fasthash = h
		}
		fastHashGroups[p.fasthash] = append(fastHashGroups[p.fasthash], p)
	}
	return fastHashGroups
}

func (c *DedupeCmd) groupBySHA256(fhPaths []pathInfo) map[string][]pathInfo {
	sha256Groups := make(map[string][]pathInfo)
	for _, p := range fhPaths {
		if p.sha256 == "" {
			h, err := utils.FullHashFile(p.path)
			if err != nil || h == "" {
				continue
			}
			p.sha256 = h
		}
		sha256Groups[p.sha256] = append(sha256Groups[p.sha256], p)
	}
	return sha256Groups
}

func (c *DedupeCmd) collectDuplicatesFromGroup(sPaths []pathInfo, size int64) []DedupeDuplicate {
	dups := make([]DedupeDuplicate, 0, max(1, len(sPaths)-1))
	// Priority sorting for "keep" candidate
	sort.Slice(sPaths, func(i, j int) bool {
		if sPaths[i].isDeduped != sPaths[j].isDeduped {
			return sPaths[i].isDeduped
		}
		return sPaths[i].path < sPaths[j].path
	})

	keep := sPaths[0].path
	for _, dup := range sPaths[1:] {
		dups = append(dups, DedupeDuplicate{
			KeepPath:      keep,
			DuplicatePath: dup.path,
			DuplicateSize: size,
		})
	}
	return dups
}

func (c *DedupeCmd) groupByHashes(
	paths []pathInfo,
	size int64,
	flags models.GlobalFlags,
) []DedupeDuplicate {
	var dups []DedupeDuplicate

	// Group by fasthash (calculate if not exists). Candidate collection is
	// deliberately read-only; hashes are persisted after confirmation.
	fastHashGroups := c.groupByFastHash(paths, flags)

	for _, fhPaths := range fastHashGroups {
		if len(fhPaths) < 2 {
			continue
		}

		// Group by sha256 (calculate only if fasthash matches)
		sha256Groups := c.groupBySHA256(fhPaths)

		for _, sPaths := range sha256Groups {
			if len(sPaths) < 2 {
				continue
			}

			dups = append(dups, c.collectDuplicatesFromGroup(sPaths, size)...)
		}
	}
	return dups
}
