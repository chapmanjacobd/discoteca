//go:build windows

package db

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/sys/windows"
)

func Repair(ctx context.Context, dbPath string) error {
	start := time.Now()

	// 1. Process-local lock
	mu := getLock(dbPath)
	mu.Lock()
	defer mu.Unlock()

	// 2. Cross-process lock
	lockFile, err := acquireCrossProcessLock(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = lockFile.Close() }()

	waitDuration := time.Since(start)

	// 3. Check if it's actually corrupt
	if IsHealthy(ctx, dbPath) {
		if waitDuration > 1*time.Millisecond {
			Log.Info("Database was repaired by another goroutine", "path", dbPath, "wait_time", waitDuration.String())
		}
		return nil
	}

	sqliteTool := "sqlite3"
	if _, err := exec.LookPath(sqliteTool); err != nil {
		if _, err := exec.LookPath("sqlite3.exe"); err == nil {
			sqliteTool = "sqlite3.exe"
		} else {
			return fmt.Errorf("sqlite3 command line tool is required for auto-repair. Please ensure it is in your PATH")
		}
	}
	Log.Debug("Using sqlite3 tool", "path", sqliteTool)

	// 4. Backup
	backupDir := backupDirName(dbPath)
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
			_ = os.Rename(sidecar, corruptMain+suffix)
		}
	}

	// 5. Recover. Invoke sqlite3 with arguments and pipe the dump through Go's
	// stdin instead of building a cmd /c shell pipeline.
	Log.Info("Attempting recovery...", "from", corruptMain, "to", dbPath)

	repairStepSuccess := false
	Log.Info("Trying recovery via .dump...")
	if dumpErr := runSQLiteDump(ctx, sqliteTool, corruptMain, ".dump", dbPath); dumpErr == nil {
		Log.Info("Initial recovery step successful via .dump")
		repairStepSuccess = true
	} else {
		Log.Warn(".dump failed, falling back to .recover", "error", dumpErr)
		_ = os.Remove(dbPath)

		if recoverErr := runSQLiteDump(ctx, sqliteTool, corruptMain, ".recover", dbPath); recoverErr == nil {
			Log.Info("Initial recovery step successful via .recover")
			repairStepSuccess = true
		} else {
			Log.Error("Recovery failed completely", "error", recoverErr)
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

			rebuildFTSTableIfExists(ctx, db, "media_fts")
			rebuildFTSTableIfExists(ctx, db, "captions_fts")

			if _, err := db.ExecContext(ctx, "VACUUM;"); err != nil {
				Log.Error("Final VACUUM failed", "error", err)
			}
			db.Close()
		}

		if IsHealthy(ctx, dbPath) {
			Log.Info("Database repair and polish successful")
			_ = os.RemoveAll(backupDir)
			return nil
		}
	}
	return fmt.Errorf("all recovery attempts failed to produce a healthy database")
}

func acquireCrossProcessLock(dbPath string) (*os.File, error) {
	lockPath := dbPath + ".repair.lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	overlapped := new(windows.Overlapped)
	if err := windows.LockFileEx(
		windows.Handle(lockFile.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		1,
		0,
		overlapped,
	); err != nil {
		_ = lockFile.Close()
		return nil, fmt.Errorf("another repair is active or the repair lock failed: %w", err)
	}
	return lockFile, nil
}

// runSQLiteDump runs `sqlite3 corrupt <dotCommand>` and feeds the resulting
// SQL to `sqlite3 target` through stdin. No shell quoting is involved.
func runSQLiteDump(ctx context.Context, sqliteTool, corruptPath, dotCommand, targetPath string) error {
	var stderr bytes.Buffer

	dumpCmd := exec.CommandContext(ctx, sqliteTool, corruptPath, dotCommand)
	dumpCmd.Stderr = &stderr
	dump, err := dumpCmd.Output()
	if err != nil {
		return fmt.Errorf("%s failed: %w: %s", dotCommand, err, stderr.String())
	}

	stderr.Reset()
	loadCmd := exec.CommandContext(ctx, sqliteTool, targetPath)
	loadCmd.Stdin = bytes.NewReader(dump)
	loadCmd.Stderr = &stderr
	if err := loadCmd.Run(); err != nil {
		return fmt.Errorf("loading recovered SQL failed: %w: %s", err, stderr.String())
	}
	return nil
}
