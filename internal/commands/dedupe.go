package commands

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/utils"
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
			quotedDup := utils.ShellQuote(d.DuplicatePath)
			quotedKeep := utils.ShellQuote(d.KeepPath)
			cmdStr := strings.ReplaceAll(c.DedupeCmd, "{}", quotedDup)
			// rmlint style is cmd duplicate keep
			exec.Command("bash", "-c", cmdStr+" "+quotedDup+" "+quotedKeep).Run()
		} else if flags.Trash {
			utils.Trash(flags, d.DuplicatePath)
		} else {
			os.Remove(d.DuplicatePath)
		}

		// Mark as deleted in DB
		// We need to find which DB this file came from.
		// For simplicity, we can just try to mark it in all provided DBs or track it in DedupeDuplicate
	}

	return nil
}

func (c *DedupeCmd) getMusicDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	// Find candidates using GROUP BY to avoid N^2 self-join
	query := `
		SELECT title, artist, album, COUNT(*) as count
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0
		  AND title != '' AND artist != ''
		GROUP BY title, artist, album
		HAVING count > 1
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var title, artist, album string
		var count int
		if err := rows.Scan(&title, &artist, &album, &count); err != nil {
			return nil, err
		}

		// Fetch all files for this candidate group
		groupQuery := `
			SELECT path, size, duration
			FROM media
			WHERE title = ? AND artist = ? AND album = ?
			  AND COALESCE(time_deleted, 0) = 0
			ORDER BY size DESC, time_modified DESC
		`
		gRows, err := sqlDB.Query(groupQuery, title, artist, album)
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
			// Basic duration check (already within the group mostly, but let's be sure)
			if keep.duration > 0 && dup.duration > 0 && MathAbs(keep.duration-dup.duration) > 8 {
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

func MathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func (c *DedupeCmd) getIDDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	query := `
		SELECT webpath, COUNT(*) as count
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0 AND webpath != ''
		GROUP BY webpath
		HAVING count > 1
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var webpath string
		var count int
		if err := rows.Scan(&webpath, &count); err != nil {
			return nil, err
		}

		gRows, err := sqlDB.Query("SELECT path, size, duration FROM media WHERE webpath = ? AND COALESCE(time_deleted, 0) = 0 ORDER BY size DESC", webpath)
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
			if keep.duration > 0 && dup.duration > 0 && MathAbs(keep.duration-dup.duration) > 8 {
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

func (c *DedupeCmd) getTitleDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	query := `
		SELECT title, COUNT(*) as count
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0 AND title != ''
		GROUP BY title
		HAVING count > 1
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var title string
		var count int
		if err := rows.Scan(&title, &count); err != nil {
			return nil, err
		}

		gRows, err := sqlDB.Query("SELECT path, size, duration FROM media WHERE title = ? AND COALESCE(time_deleted, 0) = 0 ORDER BY size DESC", title)
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
			if keep.duration > 0 && dup.duration > 0 && MathAbs(keep.duration-dup.duration) > 8 {
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

func (c *DedupeCmd) getDurationDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	query := `
		SELECT duration, COUNT(*) as count
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0 AND duration > 0
		GROUP BY duration
		HAVING count > 1
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var duration float64
		var count int
		if err := rows.Scan(&duration, &count); err != nil {
			return nil, err
		}

		gRows, err := sqlDB.Query("SELECT path, size FROM media WHERE duration = ? AND COALESCE(time_deleted, 0) = 0 ORDER BY size DESC", duration)
		if err != nil {
			continue
		}

		var paths []string
		var sizes []int64
		for gRows.Next() {
			var p string
			var s int64
			if err := gRows.Scan(&p, &s); err == nil {
				paths = append(paths, p)
				sizes = append(sizes, s)
			}
		}
		gRows.Close()

		if len(paths) < 2 {
			continue
		}

		keep := paths[0]
		for i, dup := range paths[1:] {
			dups = append(dups, DedupeDuplicate{
				KeepPath:      keep,
				DuplicatePath: dup,
				DuplicateSize: sizes[i+1],
			})
		}
	}
	return dups, nil
}

func (c *DedupeCmd) getFSDuplicates(dbPath string, flags models.GlobalFlags) ([]DedupeDuplicate, error) {
	sqlDB, err := db.Connect(dbPath)
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

		gRows, err := sqlDB.Query("SELECT path FROM media WHERE size = ? AND COALESCE(time_deleted, 0) = 0", size)
		if err != nil {
			continue
		}
		var paths []string
		for gRows.Next() {
			var p string
			if err := gRows.Scan(&p); err == nil {
				paths = append(paths, p)
			}
		}
		gRows.Close()

		if len(paths) < 2 {
			continue
		}

		// 2. Sample Hash within size group
		sampleHashes := make(map[string][]string)
		for _, p := range paths {
			h, err := utils.SampleHashFile(p, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
			if err == nil && h != "" {
				sampleHashes[h] = append(sampleHashes[h], p)
			}
		}

		for _, sPaths := range sampleHashes {
			if len(sPaths) < 2 {
				continue
			}

			// 3. Full Hash within sample group
			fullHashes := make(map[string][]string)
			for _, p := range sPaths {
				h, err := utils.FullHashFile(p)
				if err == nil && h != "" {
					fullHashes[h] = append(fullHashes[h], p)
				}
			}

			for _, fPaths := range fullHashes {
				if len(fPaths) < 2 {
					continue
				}
				sort.Strings(fPaths)
				keep := fPaths[0]
				for _, dup := range fPaths[1:] {
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
