//go:build !windows

package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/shellquote"
)

func Repair(ctx context.Context, dbPath string) error {
	start := time.Now()

	// 1. Process-local lock
	mu := getLock(dbPath)
	mu.Lock()
	defer mu.Unlock()

	// 2. Cross-process lock
	lockPath := dbPath + ".repair.lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer func() {
		lockFile.Close()
		os.Remove(lockPath)
	}()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire flock: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	waitDuration := time.Since(start)

	// 3. Check if it's actually corrupt
	if IsHealthy(ctx, dbPath) {
		if waitDuration > 1*time.Millisecond {
			Log.Info("Database was repaired by another goroutine", "path", dbPath, "wait_time", waitDuration.String())
		}
		return nil
	}

	if _, err := exec.LookPath("sqlite3"); err != nil {
		return errors.New("sqlite3 command line tool is required for auto-repair")
	}

	// 4. Backup
	now := time.Now().Unix()
	backupDir := fmt.Sprintf("%s.corrupt.%d", dbPath, now)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	corruptMain := backupDir + "/main.db"
	if err := os.Rename(dbPath, corruptMain); err != nil {
		return fmt.Errorf("failed to move corrupted database: %w", err)
	}

	for _, suffix := range []string{"-wal", "-shm"} {
		sidecar := dbPath + suffix
		if _, err := os.Stat(sidecar); err == nil {
			os.Rename(sidecar, corruptMain+suffix)
		}
	}

	// 5. Recover
	Log.Info("Attempting recovery...", "from", corruptMain, "to", dbPath)

	quotedCorrupt := shellquote.ShellQuote(corruptMain)
	quotedDB := shellquote.ShellQuote(dbPath)

	// Fallback to .dump first as it preserves schema better if it works
	repairStepSuccess := false
	Log.Info("Trying recovery via .dump...")
	cmdDump := exec.CommandContext(
		ctx,
		"bash",
		"-c",
		fmt.Sprintf("sqlite3 %s \".dump\" | sqlite3 %s", quotedCorrupt, quotedDB),
	)
	out, err := cmdDump.CombinedOutput()
	if err == nil {
		Log.Info("Initial recovery step successful via .dump")
		repairStepSuccess = true
	} else {
		Log.Warn(".dump failed, falling back to .recover", "error", err, "output", string(out))
		os.Remove(dbPath)

		// Fallback to .recover
		// We use .quit to ensure it doesn't hang if it somehow enters interactive mode
		cmdRecover := exec.CommandContext(
			ctx,
			"bash",
			"-c",
			fmt.Sprintf("sqlite3 %s \".recover\" \".quit\" | sqlite3 %s", quotedCorrupt, quotedDB),
		)
		out, err = cmdRecover.CombinedOutput()
		if err == nil {
			Log.Info("Initial recovery step successful via .recover")
			repairStepSuccess = true
		} else {
			Log.Error("Recovery failed completely", "error", err, "output", string(out))
		}
	}

	if repairStepSuccess {
		// 6. Polish and Verify
		db, err := Connect(ctx, dbPath)
		if err != nil {
			Log.Error("Failed to open recovered database for polish", "error", err)
		} else {
			Log.Info("Running final polish (REINDEX, FTS REBUILD, VACUUM)...")
			if _, err := db.ExecContext(ctx, "REINDEX;"); err != nil {
				Log.Warn("REINDEX failed", "error", err)
			}

			// FTS rebuilding is critical as corruption often hides here
			var hasMediaFTS bool
			_ = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')").
				Scan(&hasMediaFTS)
			if hasMediaFTS {
				if _, err := db.ExecContext(
					ctx,
					"INSERT INTO media_fts(media_fts) VALUES('rebuild');",
				); err != nil {
					Log.Warn("media_fts rebuild failed", "error", err)
				}
			}

			var hasCaptionsFTS bool
			_ = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='captions_fts')").
				Scan(&hasCaptionsFTS)
			if hasCaptionsFTS {
				if _, err := db.ExecContext(
					ctx,
					"INSERT INTO captions_fts(captions_fts) VALUES('rebuild');",
				); err != nil {
					Log.Warn("captions_fts rebuild failed", "error", err)
				}
			}

			if _, err := db.ExecContext(ctx, "VACUUM;"); err != nil {
				Log.Error("Final VACUUM failed", "error", err)
			}
			db.Close()
		}

		if IsHealthy(ctx, dbPath) {
			Log.Info("Database repair and polish successful")
			os.RemoveAll(backupDir)
			return nil
		}
	}
	return errors.New("all recovery attempts failed to produce a healthy database")
}
