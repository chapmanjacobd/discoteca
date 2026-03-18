package commands

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
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
	models.PostActionFlags  `embed:""`
	models.HashingFlags     `embed:""`

	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

type DedupeDuplicate struct {
	KeepPath      string
	DuplicatePath string
	DuplicateSize int64
}

func (c *DedupeCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		DedupeFlags:      c.DedupeFlags,
		PostActionFlags:  c.PostActionFlags,
		HashingFlags:     c.HashingFlags,
	}

	var duplicates []DedupeDuplicate
	var err error

	for _, dbPath := range c.Databases {
		var dbDups []DedupeDuplicate
		if c.Audio {
			dbDups, err = c.getMusicDuplicates(dbPath)
		} else if c.ExtractorID {
			dbDups, err = c.getIDDuplicates(dbPath)
		} else if c.TitleOnly {
			dbDups, err = c.getTitleDuplicates(dbPath)
		} else if c.DurationOnly {
			dbDups, err = c.getDurationDuplicates(dbPath)
		} else if c.Filesystem {
			dbDups, err = c.getFSDuplicates(dbPath, flags)
		} else {
			return fmt.Errorf("profile not set. Use --audio, --id, --title, --duration, or --fs")
		}

		if err != nil {
			return err
		}
		duplicates = append(duplicates, dbDups...)
	}

	// Apply name similarity filters and deduplicate candidates
	metric := metrics.NewSorensenDice()
	var finalCandidates []DedupeDuplicate
	seenDuplicates := make(map[string]bool)

	for _, d := range duplicates {
		if seenDuplicates[d.DuplicatePath] || d.KeepPath == d.DuplicatePath {
			continue
		}

		if c.Dirname {
			if strutil.Similarity(filepath.Dir(d.KeepPath), filepath.Dir(d.DuplicatePath), metric) < c.MinSimilarityRatio {
				continue
			}
		}

		if c.Basename {
			if strutil.Similarity(filepath.Base(d.KeepPath), filepath.Base(d.DuplicatePath), metric) < c.MinSimilarityRatio {
				continue
			}
		}

		// Check if keep path still exists
		if !utils.FileExists(d.KeepPath) {
			continue
		}

		finalCandidates = append(finalCandidates, d)
		seenDuplicates[d.DuplicatePath] = true
	}

	if len(finalCandidates) == 0 {
		slog.Info("No duplicates found")
		return nil
	}

	// Print summary
	var totalSavings int64
	for _, d := range finalCandidates {
		totalSavings += d.DuplicateSize
		fmt.Printf("Keep: %s\n  Dup: %s (%s)\n", d.KeepPath, d.DuplicatePath, utils.FormatSize(d.DuplicateSize))
	}
	fmt.Printf("\nApprox. space savings: %s (%d files)\n", utils.FormatSize(totalSavings), len(finalCandidates))

	if !c.NoConfirm {
		fmt.Print("\nDelete duplicates? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			return nil
		}
	}

	slog.Info("Deleting duplicates...")
	for _, d := range finalCandidates {
		if c.DedupeCmd != "" {
			quotedDup := shellquote.ShellQuote(d.DuplicatePath)
			quotedKeep := shellquote.ShellQuote(d.KeepPath)
			cmdStr := strings.ReplaceAll(c.DedupeCmd, "{}", quotedDup)
			// rmlint style is cmd duplicate keep
			exec.Command("bash", "-c", cmdStr+" "+quotedDup+" "+quotedKeep).Run()
		} else if flags.Trash {
			utils.Trash(flags, d.DuplicatePath)
		} else {
			os.Remove(d.DuplicatePath)
		}

		// Mark as deleted in DB - try all provided DBs
		for _, dbPath := range c.Databases {
			sqlDB, _, err := db.ConnectWithInit(dbPath)
			if err == nil {
				// Mark duplicate as deleted
				_, _ = sqlDB.Exec("UPDATE media SET time_deleted = unixepoch() WHERE path = ?", d.DuplicatePath)
				
				// Mark keep file as deduped
				_, _ = sqlDB.Exec("UPDATE media SET is_deduped = 1 WHERE path = ?", d.KeepPath)
				
				// Update hash if not already set
				if d.DuplicateSize > 0 {
					h, err := utils.SampleHashFile(d.KeepPath, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
					if err == nil && h != "" {
						_, _ = sqlDB.Exec("UPDATE media SET fasthash = ? WHERE path = ?", h, d.KeepPath)
					}
					h, err = utils.FullHashFile(d.KeepPath)
					if err == nil && h != "" {
						_, _ = sqlDB.Exec("UPDATE media SET sha256 = ? WHERE path = ?", h, d.KeepPath)
					}
				}
				
				sqlDB.Close()
			}
		}
	}

	return nil
}

func (c *DedupeCmd) getDuplicatesBy(dbPath string, groupByCols, selectCols, whereClause string) ([]DedupeDuplicate, error) {
	sqlDB, _, err := db.ConnectWithInit(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	if whereClause != "" {
		whereClause = " AND " + whereClause
	}

	queryStr := fmt.Sprintf(`
		SELECT %s, COUNT(*) as count
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0 %s
		GROUP BY %s
		HAVING count > 1
	`, groupByCols, whereClause, groupByCols)

	rows, err := sqlDB.Query(queryStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		// We need to scan the groupByCols. Since it's dynamic, we use a slice of interface{}
		cols := strings.Split(groupByCols, ",")
		values := make([]any, len(cols)+1)
		for i := range values {
			values[i] = new(any)
		}
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		// Build the group query
		whereParts := make([]string, len(cols))
		args := make([]any, len(cols))
		for i, col := range cols {
			whereParts[i] = strings.TrimSpace(col) + " = ?"
			args[i] = *(values[i].(*any))
		}

		groupQuery := fmt.Sprintf(`
			SELECT path, size, duration
			FROM media
			WHERE %s AND COALESCE(time_deleted, 0) = 0
			ORDER BY size DESC, time_modified DESC
		`, strings.Join(whereParts, " AND "))

		gRows, err := sqlDB.Query(groupQuery, args...)
		if err != nil {
			continue
		}

		type item struct {
			path     string
			size     int64
			duration float64
		}
		var items []item
		for gRows.Next() {
			var i item
			if err := gRows.Scan(&i.path, &i.size, &i.duration); err == nil {
				items = append(items, i)
			}
		}
		gRows.Close()

		if len(items) < 2 {
			continue
		}

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
	}
	return dups, nil
}

func (c *DedupeCmd) getMusicDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(dbPath, "title, artist, album", "path, size, duration", "title != '' AND artist != ''")
}

func (c *DedupeCmd) getIDDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(dbPath, "webpath", "path, size, duration", "webpath != ''")
}

func (c *DedupeCmd) getTitleDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(dbPath, "title", "path, size, duration", "title != ''")
}

func (c *DedupeCmd) getDurationDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	return c.getDuplicatesBy(dbPath, "duration", "path, size", "duration > 0")
}

func (c *DedupeCmd) getFSDuplicates(dbPath string, flags models.GlobalFlags) ([]DedupeDuplicate, error) {
	sqlDB, _, err := db.ConnectWithInit(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	// 1. Group by size in SQL
	query := `
		SELECT size, COUNT(*) as count
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0 AND size > 0
		GROUP BY size
		HAVING count > 1
	`
	rows, err := sqlDB.Query(query)
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

		gRows, err := sqlDB.Query("SELECT path, fasthash, sha256 FROM media WHERE size = ? AND COALESCE(time_deleted, 0) = 0", size)
		if err != nil {
			continue
		}
		type pathInfo struct {
			path     string
			fasthash string
			sha256   string
		}
		var paths []pathInfo
		for gRows.Next() {
			var p pathInfo
			if err := gRows.Scan(&p.path, &p.fasthash, &p.sha256); err == nil {
				paths = append(paths, p)
			}
		}
		gRows.Close()

		if len(paths) < 2 {
			continue
		}

		// 2. Group by fasthash (calculate if not exists)
		fastHashGroups := make(map[string][]pathInfo)
		for _, p := range paths {
			if p.fasthash == "" {
				// Calculate fasthash if not exists
				h, err := utils.SampleHashFile(p.path, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
				if err == nil && h != "" {
					p.fasthash = h
					// Save hash back to database
					_, _ = sqlDB.Exec("UPDATE media SET fasthash = ? WHERE path = ?", h, p.path)
				} else {
					continue // Skip files that can't be hashed
				}
			}
			fastHashGroups[p.fasthash] = append(fastHashGroups[p.fasthash], p)
		}

		for _, fhPaths := range fastHashGroups {
			if len(fhPaths) < 2 {
				continue
			}

			// 3. Group by sha256 (calculate only if fasthash matches)
			sha256Groups := make(map[string][]pathInfo)
			for _, p := range fhPaths {
				if p.sha256 == "" {
					// Calculate sha256 only if fasthash matched
					h, err := utils.FullHashFile(p.path)
					if err == nil && h != "" {
						p.sha256 = h
						// Save hash back to database
						_, _ = sqlDB.Exec("UPDATE media SET sha256 = ? WHERE path = ?", h, p.path)
					} else {
						continue
					}
				}
				sha256Groups[p.sha256] = append(sha256Groups[p.sha256], p)
			}

			for _, sPaths := range sha256Groups {
				if len(sPaths) < 2 {
					continue
				}
				pathStrings := make([]string, len(sPaths))
				for i, p := range sPaths {
					pathStrings[i] = p.path
				}
				sort.Strings(pathStrings)
				keep := pathStrings[0]
				for _, dup := range pathStrings[1:] {
					// Mark keep file as deduped
					_, _ = sqlDB.Exec("UPDATE media SET is_deduped = 1 WHERE path = ?", keep)
					dups = append(dups, DedupeDuplicate{
						KeepPath:      keep,
						DuplicatePath: dup,
						DuplicateSize: size,
					})
				}
			}
		}
	}

	return dups, nil
}
