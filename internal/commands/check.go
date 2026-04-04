package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type CheckCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.MediaFilterFlags `embed:""`

	Args   []string `help:"Database file followed by optional paths to check" required:"" arg:""`
	DryRun bool     `help:"Don't actually mark files as deleted"`

	CheckPaths []string `kong:"-"`
	Databases  []string `kong:"-"`
}

func (c *CheckCmd) AfterApply() error {
	if err := c.CoreFlags.AfterApply(); err != nil {
		return err
	}
	if err := c.MediaFilterFlags.AfterApply(); err != nil {
		return err
	}
	if len(c.Args) < 1 {
		return errors.New("at least one database file is required")
	}

	if utils.IsSQLite(c.Args[0]) || strings.HasSuffix(c.Args[0], ".db") {
		c.Databases = []string{c.Args[0]}
		if len(c.Args) > 1 {
			c.CheckPaths = c.Args[1:]
		}
	} else {
		// Fallback: first is DB
		c.Databases = []string{c.Args[0]}
		if len(c.Args) > 1 {
			c.CheckPaths = c.Args[1:]
		}
	}
	return nil
}

func (c *CheckCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	c.CheckPaths = utils.ExpandStdin(c.CheckPaths)

	// If paths provided, build a presence set
	var presenceSet map[string]bool
	var absCheckPaths []string
	if len(c.CheckPaths) > 0 {
		presenceSet = make(map[string]bool)
		for _, root := range c.CheckPaths {
			absRoot, err := filepath.Abs(root)
			if err != nil {
				return err
			}
			absCheckPaths = append(absCheckPaths, absRoot)
			models.Log.Info("Scanning filesystem for presence set", "path", absRoot)
			err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					absPath, _ := filepath.Abs(path)
					// Use path as-is
					presenceSet[absPath] = true
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	for _, dbPath := range c.Databases {
		sqlDB, queries, err := db.ConnectWithInit(ctx, dbPath)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		allMedia, err := queries.GetMedia(ctx, 1000000)
		if err != nil {
			return err
		}

		models.Log.Info("Checking files", "count", len(allMedia), "database", dbPath)

		missingCount := 0
		now := time.Now().Unix()

		for _, m := range allMedia {
			isMissing := false

			if presenceSet != nil {
				// Only check files that are within the scanned roots
				inScannedRoot := false
				for _, root := range absCheckPaths {
					if strings.HasPrefix(m.Path, root) {
						inScannedRoot = true
						break
					}
				}

				if inScannedRoot {
					if !presenceSet[m.Path] {
						isMissing = true
					}
				} else {
					// Outside scanned roots, skip or use Stat?
					// For safety, if user provided roots, we only check files in those roots.
					continue
				}
			} else {
				// No presence set, fallback to individual Stats
				if !utils.FileExists(m.Path) {
					isMissing = true
				}
			}

			if isMissing {
				missingCount++
				if !c.DryRun {
					models.Log.Debug("Marking missing file as deleted", "path", m.Path)
					if err := queries.MarkDeleted(ctx, db.MarkDeletedParams{
						TimeDeleted: sql.NullInt64{Int64: now, Valid: true},
						Path:        m.Path,
					}); err != nil {
						models.Log.Error("Failed to mark file as deleted", "path", m.Path, "error", err)
					}
				} else {
					fmt.Printf("[Dry-run] Missing: %s\n", m.Path)
				}
			}
		}

		if c.DryRun {
			models.Log.Info("Check complete (dry-run)", "missing", missingCount)
		} else {
			models.Log.Info("Check complete", "marked_deleted", missingCount)

			// Refresh folder_stats and FTS if files were marked as deleted
			if missingCount > 0 {
				models.Log.Info("Refreshing folder_stats and FTS after marking files deleted...")
				if err := db.RefreshFolderStats(sqlDB); err != nil {
					models.Log.Error("Failed to refresh folder_stats", "error", err)
				}
				if err := db.RebuildFTS(sqlDB, dbPath); err != nil {
					models.Log.Error("Failed to rebuild FTS", "error", err)
				}
			}
		}
	}
	return nil
}
