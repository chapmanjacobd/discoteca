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
			cmdStr := strings.ReplaceAll(c.DedupeCmd, "{}", fmt.Sprintf("'%s'", d.DuplicatePath))
			// rmlint style is cmd duplicate keep
			exec.Command("bash", "-c", cmdStr+" "+fmt.Sprintf("'%s'", d.DuplicatePath)+" "+fmt.Sprintf("'%s'", d.KeepPath)).Run()
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

	// Simplified join query for duplicates
	query := `
		SELECT m1.path as keep_path, m2.path as duplicate_path, m2.size as duplicate_size
		FROM media m1
		JOIN media m2 ON m1.title = m2.title
			AND m1.artist = m2.artist
			AND m1.album = m2.album
			AND ABS(m1.duration - m2.duration) <= 8
			AND m1.path != m2.path
		WHERE COALESCE(m1.time_deleted, 0) = 0 AND COALESCE(m2.time_deleted, 0) = 0
		AND m1.title != '' AND m1.artist != ''
		ORDER BY m1.size DESC, m1.time_modified DESC
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var d DedupeDuplicate
		if err := rows.Scan(&d.KeepPath, &d.DuplicatePath, &d.DuplicateSize); err != nil {
			return nil, err
		}
		dups = append(dups, d)
	}
	return dups, nil
}

func (c *DedupeCmd) getIDDuplicates(dbPath string) ([]DedupeDuplicate, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	query := `
		SELECT m1.path as keep_path, m2.path as duplicate_path, m2.size as duplicate_size
		FROM media m1
		JOIN media m2 ON m1.webpath = m2.webpath
			AND ABS(m1.duration - m2.duration) <= 8
			AND m1.path != m2.path
		WHERE COALESCE(m1.time_deleted, 0) = 0 AND COALESCE(m2.time_deleted, 0) = 0
		AND m1.webpath != ''
		ORDER BY m1.size DESC
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var d DedupeDuplicate
		if err := rows.Scan(&d.KeepPath, &d.DuplicatePath, &d.DuplicateSize); err != nil {
			return nil, err
		}
		dups = append(dups, d)
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
		SELECT m1.path as keep_path, m2.path as duplicate_path, m2.size as duplicate_size
		FROM media m1
		JOIN media m2 ON m1.title = m2.title
			AND ABS(m1.duration - m2.duration) <= 8
			AND m1.path != m2.path
		WHERE COALESCE(m1.time_deleted, 0) = 0 AND COALESCE(m2.time_deleted, 0) = 0
		AND m1.title != ''
		ORDER BY m1.size DESC
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var d DedupeDuplicate
		if err := rows.Scan(&d.KeepPath, &d.DuplicatePath, &d.DuplicateSize); err != nil {
			return nil, err
		}
		dups = append(dups, d)
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
		SELECT m1.path as keep_path, m2.path as duplicate_path, m2.size as duplicate_size
		FROM media m1
		JOIN media m2 ON m1.duration = m2.duration
			AND m1.path != m2.path
		WHERE COALESCE(m1.time_deleted, 0) = 0 AND COALESCE(m2.time_deleted, 0) = 0
		AND m1.duration > 0
		ORDER BY m1.size DESC
	`

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dups []DedupeDuplicate
	for rows.Next() {
		var d DedupeDuplicate
		if err := rows.Scan(&d.KeepPath, &d.DuplicatePath, &d.DuplicateSize); err != nil {
			return nil, err
		}
		dups = append(dups, d)
	}
	return dups, nil
}

func (c *DedupeCmd) getFSDuplicates(dbPath string, flags models.GlobalFlags) ([]DedupeDuplicate, error) {
	sqlDB, err := db.Connect(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	// 1. Group by size
	query := "SELECT path, size FROM media WHERE COALESCE(time_deleted, 0) = 0 AND size > 0"
	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sizeGroups := make(map[int64][]string)
	for rows.Next() {
		var path string
		var size int64
		if err := rows.Scan(&path, &size); err != nil {
			return nil, err
		}
		sizeGroups[size] = append(sizeGroups[size], path)
	}

	var candidates []string
	for _, paths := range sizeGroups {
		if len(paths) > 1 {
			candidates = append(candidates, paths...)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// 2. Sample Hash
	sampleHashes := make(map[string][]string)
	for _, p := range candidates {
		h, err := utils.SampleHashFile(p, flags.HashThreads, flags.HashGap, flags.HashChunkSize)
		if err == nil && h != "" {
			sampleHashes[h] = append(sampleHashes[h], p)
		}
	}

	var fullHashCandidates []string
	for _, paths := range sampleHashes {
		if len(paths) > 1 {
			fullHashCandidates = append(fullHashCandidates, paths...)
		}
	}

	// 3. Full Hash
	fullHashes := make(map[string][]string)
	for _, p := range fullHashCandidates {
		h, err := utils.FullHashFile(p)
		if err == nil && h != "" {
			fullHashes[h] = append(fullHashes[h], p)
		}
	}

	var dups []DedupeDuplicate
	for _, paths := range fullHashes {
		if len(paths) > 1 {
			sort.Strings(paths) // consistent keep path
			keep := paths[0]
			var size int64
			sqlDB.QueryRow("SELECT size FROM media WHERE path = ?", keep).Scan(&size)
			for _, dup := range paths[1:] {
				dups = append(dups, DedupeDuplicate{
					KeepPath:      keep,
					DuplicatePath: dup,
					DuplicateSize: size,
				})
			}
		}
	}

	return dups, nil
}
