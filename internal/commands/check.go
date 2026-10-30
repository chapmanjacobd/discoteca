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
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

type CheckCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.MediaFilterFlags `embed:""`

	Args   []string `help:"Database file followed by scan roots; path/media filters select which database rows may be marked deleted (roots are always walked fully)" required:"true" arg:""`
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
	// The local --dry-run alias shadows CoreFlags.DryRun, so fold --simulate
	// into it here rather than relying on the embedded field.
	c.DryRun = c.DryRun || c.Simulate
	if len(c.Args) < 1 {
		return errors.New("at least one database file is required")
	}

	// First argument is always treated as a database file
	c.Databases = []string{c.Args[0]}
	if len(c.Args) > 1 {
		c.CheckPaths = c.Args[1:]
	}
	return nil
}

func (c *CheckCmd) buildPresenceSet(allowEmptyRoots bool) (map[string]bool, []string, error) {
	if len(c.CheckPaths) == 0 {
		return nil, nil, nil
	}

	presenceSet := make(map[string]bool)
	var absCheckPaths []string
	for _, root := range c.CheckPaths {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, nil, err
		}

		info, err := os.Stat(absRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot check %q: %w", absRoot, err)
		}
		if !info.IsDir() {
			return nil, nil, fmt.Errorf("check path is not a directory: %s", absRoot)
		}

		absCheckPaths = append(absCheckPaths, absRoot)
		logging.Log.Info("Scanning filesystem for presence set", "path", absRoot)
		filesInRoot := 0
		err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() {
				presenceSet[path] = true
				filesInRoot++
			}
			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan %q: %w", absRoot, err)
		}
		if filesInRoot == 0 && !allowEmptyRoots {
			return nil, nil, fmt.Errorf(
				"refusing to mark media deleted: scan root %q contains no files (empty mount or wrong path?)",
				absRoot,
			)
		}
	}
	return presenceSet, absCheckPaths, nil
}

func pathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func (c *CheckCmd) checkMedia(m models.MediaWithDB, presenceSet map[string]bool, absCheckPaths []string) bool {
	if presenceSet != nil {
		// Only check files that are within the scanned roots
		inScannedRoot := false
		for _, root := range absCheckPaths {
			if pathWithinRoot(m.Path, root) {
				inScannedRoot = true
				break
			}
		}

		if inScannedRoot {
			return !presenceSet[m.Path]
		}
		// Outside scanned roots, don't consider it missing (we don't know)
		return false
	}

	// No presence set, fallback to individual Stats
	return !fileutil.FileExists(m.Path)
}

func (c *CheckCmd) checkDatabase(
	ctx context.Context,
	dbPath string,
	presenceSet map[string]bool,
	absCheckPaths []string,
) error {
	sqlDB, queries, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
	}
	flags.HideDeleted = true
	flags.All = true

	allMedia, err := query.MediaQuery(ctx, []string{dbPath}, flags)
	if err != nil {
		return err
	}

	logging.Log.Info("Checking files", "count", len(allMedia), "database", dbPath)

	missingCount := 0
	now := time.Now().Unix()

	for _, m := range allMedia {
		if c.checkMedia(m, presenceSet, absCheckPaths) {
			missingCount++
			if !c.DryRun {
				logging.Log.Debug("Marking missing file as deleted", "path", m.Path)
				if err := queries.MarkDeleted(ctx, db.MarkDeletedParams{
					TimeDeleted: sql.NullInt64{Int64: now, Valid: true},
					Path:        m.Path,
				}); err != nil {
					logging.Log.Error("Failed to mark file as deleted", "path", m.Path, "error", err)
				}
			} else {
				fmt.Printf("[Dry-run] Missing: %s\n", m.Path)
			}
		}
	}

	if c.DryRun {
		logging.Log.Info("Check complete (dry-run)", "missing", missingCount)
	} else {
		logging.Log.Info("Check complete", "marked_deleted", missingCount)
		if missingCount > 0 {
			logging.Log.Info("Refreshing folder_stats and FTS after marking files deleted...")
			_ = db.RefreshFolderStats(ctx, sqlDB)
			_ = db.RebuildFTS(ctx, sqlDB, dbPath)
		}
	}
	return nil
}

func (c *CheckCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	c.CheckPaths = fileutil.ExpandStdin(c.CheckPaths)

	presenceSet, absCheckPaths, err := c.buildPresenceSet(false)
	if err != nil {
		return err
	}

	for _, dbPath := range c.Databases {
		if err := c.checkDatabase(ctx, dbPath, presenceSet, absCheckPaths); err != nil {
			return err
		}
	}
	return nil
}
