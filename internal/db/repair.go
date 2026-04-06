//go:build !windows

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	// Register SQLite driver for database/sql
	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/shellquote"
)

func Repair(ctx context.Context, dbPath string) error {
	start := time.Now()

	mu := getLock(dbPath)
	mu.Lock()
	defer mu.Unlock()

	lockFile, err := acquireCrossProcessLock(dbPath)
	if err != nil {
		return err
	}
	defer func() {
		lockFile.Close()
		os.Remove(dbPath + ".repair.lock")
	}()

	waitDuration := time.Since(start)

	if IsHealthy(ctx, dbPath) {
		if waitDuration > 1*time.Millisecond {
			Log.Info("Database was repaired by another goroutine", "path", dbPath, "wait_time", waitDuration.String())
		}
		return nil
	}

	if _, err2 := exec.LookPath("sqlite3"); err2 != nil {
		return errors.New("sqlite3 command line tool is required for auto-repair")
	}

	backupDir, err := backupCorruptDB(dbPath)
	if err != nil {
		return err
	}

	corruptMain := backupDir + "/main.db"
	if success := attemptRecovery(ctx, corruptMain, dbPath); !success {
		os.RemoveAll(backupDir)
		return errors.New("all recovery attempts failed to produce a healthy database")
	}

	if err := polishRecoveredDB(ctx, dbPath); err != nil {
		Log.Error("Polish failed, but data may be recovered", "error", err)
	}

	if IsHealthy(ctx, dbPath) {
		Log.Info("Database repair and polish successful")
		os.RemoveAll(backupDir)
		return nil
	}
	return errors.New("all recovery attempts failed to produce a healthy database")
}

func acquireCrossProcessLock(dbPath string) (*os.File, error) {
	lockPath := dbPath + ".repair.lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	if err2 := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err2 != nil {
		return nil, fmt.Errorf("failed to acquire flock: %w", err2)
	}
	_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	return lockFile, nil
}

func backupCorruptDB(dbPath string) (string, error) {
	now := time.Now().Unix()
	backupDir := fmt.Sprintf("%s.corrupt.%d", dbPath, now)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	corruptMain := backupDir + "/main.db"
	if err := os.Rename(dbPath, corruptMain); err != nil {
		return "", fmt.Errorf("failed to move corrupted database: %w", err)
	}

	for _, suffix := range []string{"-wal", "-shm"} {
		sidecar := dbPath + suffix
		if _, err := os.Stat(sidecar); err == nil {
			if err3 := os.Rename(sidecar, corruptMain+suffix); err3 != nil {
				Log.Warn("Failed to rename sidecar file", "suffix", suffix, "error", err3)
			}
		}
	}
	return backupDir, nil
}

func attemptRecovery(ctx context.Context, corruptMain, dbPath string) bool {
	Log.Info("Attempting recovery...", "from", corruptMain, "to", dbPath)

	quotedCorrupt := shellquote.ShellQuote(corruptMain)
	quotedDB := shellquote.ShellQuote(dbPath)

	if tryDumpRecovery(ctx, quotedCorrupt, quotedDB) {
		return true
	}

	os.Remove(dbPath)
	return tryRecoverRecovery(ctx, quotedCorrupt, quotedDB)
}

func tryDumpRecovery(ctx context.Context, corrupt, target string) bool {
	Log.Info("Trying recovery via .dump...")
	cmdDump := exec.CommandContext(
		ctx,
		"bash",
		"-c",
		fmt.Sprintf("sqlite3 %s \".dump\" | sqlite3 %s", corrupt, target),
	)
	out, err := cmdDump.CombinedOutput()
	if err == nil {
		Log.Info("Initial recovery step successful via .dump")
		return true
	}
	Log.Warn(".dump failed, falling back to .recover", "error", err, "output", string(out))
	return false
}

func tryRecoverRecovery(ctx context.Context, corrupt, target string) bool {
	cmdRecover := exec.CommandContext(
		ctx,
		"bash",
		"-c",
		fmt.Sprintf("sqlite3 %s \".recover\" \".quit\" | sqlite3 %s", corrupt, target),
	)
	out, err := cmdRecover.CombinedOutput()
	if err == nil {
		Log.Info("Initial recovery step successful via .recover")
		return true
	}
	Log.Error("Recovery failed completely", "error", err, "output", string(out))
	return false
}

func polishRecoveredDB(ctx context.Context, dbPath string) error {
	db, err := Connect(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("failed to open recovered database: %w", err)
	}
	defer db.Close()

	Log.Info("Running final polish (REINDEX, FTS REBUILD, VACUUM)...")

	if _, err := db.ExecContext(ctx, "REINDEX;"); err != nil {
		Log.Warn("REINDEX failed", "error", err)
	}

	rebuildFTSTableIfExists(ctx, db, "media_fts")
	rebuildFTSTableIfExists(ctx, db, "captions_fts")

	if _, err := db.ExecContext(ctx, "VACUUM;"); err != nil {
		return fmt.Errorf("VACUUM failed: %w", err)
	}
	return nil
}

func rebuildFTSTableIfExists(ctx context.Context, db *sql.DB, tableName string) {
	var exists bool
	_ = db.QueryRowContext(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='%s')", tableName)).
		Scan(&exists)
	if exists {
		query := fmt.Sprintf("INSERT INTO %s(%s) VALUES('rebuild');", tableName, tableName)
		if _, err := db.ExecContext(ctx, query); err != nil {
			Log.Warn("%s rebuild failed", tableName, "error", err)
		}
	}
}
