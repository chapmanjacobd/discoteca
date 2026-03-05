package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chapmanjacobd/discotheque/internal/shellquote"
	_ "github.com/mattn/go-sqlite3"
)

var repairLocks sync.Map

func getLock(path string) *sync.Mutex {
	v, _ := repairLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// IsCorruptionError checks if the error is a database corruption error
func IsCorruptionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "database disk image is malformed")
}

func Repair(dbPath string) error {
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
	if isHealthy(dbPath) {
		if waitDuration > 1*time.Millisecond {
			slog.Info("Database was repaired by another goroutine", "path", dbPath, "wait_time", waitDuration.String())
		}
		return nil
	}

	if _, err := exec.LookPath("sqlite3"); err != nil {
		return fmt.Errorf("sqlite3 command line tool is required for auto-repair")
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
	slog.Info("Attempting recovery...", "from", corruptMain, "to", dbPath)

	quotedCorrupt := shellquote.ShellQuote(corruptMain)
	quotedDB := shellquote.ShellQuote(dbPath)

	// Fallback to .dump first as it preserves schema better if it works
	repairStepSuccess := false
	slog.Info("Trying recovery via .dump...")
	cmdDump := exec.Command("bash", "-c", fmt.Sprintf("sqlite3 %s \".dump\" | sqlite3 %s", quotedCorrupt, quotedDB))
	out, err := cmdDump.CombinedOutput()
	if err == nil {
		slog.Info("Initial recovery step successful via .dump")
		repairStepSuccess = true
	} else {
		slog.Warn(".dump failed, falling back to .recover", "error", err, "output", string(out))
		os.Remove(dbPath)

		// Fallback to .recover
		// We use .quit to ensure it doesn't hang if it somehow enters interactive mode
		cmdRecover := exec.Command("bash", "-c", fmt.Sprintf("sqlite3 %s \".recover\" \".quit\" | sqlite3 %s", quotedCorrupt, quotedDB))
		out, err = cmdRecover.CombinedOutput()
		if err == nil {
			slog.Info("Initial recovery step successful via .recover")
			repairStepSuccess = true
		} else {
			slog.Error("Recovery failed completely", "error", err, "output", string(out))
		}
	}

	if repairStepSuccess {
		// 6. Polish and Verify
		db, err := Connect(dbPath)
		if err != nil {
			slog.Error("Failed to open recovered database for polish", "error", err)
		} else {
			slog.Info("Running final polish (REINDEX, FTS REBUILD, VACUUM)...")
			if _, err := db.Exec("REINDEX;"); err != nil {
				slog.Warn("REINDEX failed", "error", err)
			}

			// FTS rebuilding is critical as corruption often hides here
			var hasMediaFTS bool
			_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')").Scan(&hasMediaFTS)
			if hasMediaFTS {
				if _, err := db.Exec("INSERT INTO media_fts(media_fts) VALUES('rebuild');"); err != nil {
					slog.Warn("media_fts rebuild failed", "error", err)
				}
			}

			var hasCaptionsFTS bool
			_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='captions_fts')").Scan(&hasCaptionsFTS)
			if hasCaptionsFTS {
				if _, err := db.Exec("INSERT INTO captions_fts(captions_fts) VALUES('rebuild');"); err != nil {
					slog.Warn("captions_fts rebuild failed", "error", err)
				}
			}

			if _, err := db.Exec("VACUUM;"); err != nil {
				slog.Error("Final VACUUM failed", "error", err)
			}
			db.Close()
		}

		if isHealthy(dbPath) {
			slog.Info("Database repair and polish successful")
			os.RemoveAll(backupDir)
			return nil
		}
	}
	return fmt.Errorf("all recovery attempts failed to produce a healthy database")
}

func isHealthy(dbPath string) bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}

	// Use sql.Open directly instead of Connect to avoid connection pool deadlocks.
	// isHealthy needs to be able to open its own connection regardless of global pool limits.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		slog.Debug("Health check: failed to open connection", "path", dbPath, "error", err)
		return false
	}
	defer db.Close()

	// Enable WAL mode to match application behavior and detect WAL corruption
	_, _ = db.Exec("PRAGMA journal_mode=WAL")

	// 1. Thorough integrity check
	rows, err := db.Query("PRAGMA integrity_check")
	if err != nil {
		slog.Debug("Health check: PRAGMA integrity_check query failed", "error", err)
		return false
	}
	defer rows.Close()

	foundOk := false
	for rows.Next() {
		var res string
		if err := rows.Scan(&res); err != nil {
			slog.Debug("Health check: failed to scan integrity row", "error", err)
			return false
		}
		if res == "ok" {
			foundOk = true
		} else {
			slog.Warn("Health check: integrity error found", "msg", res)
			return false
		}
	}
	if !foundOk {
		slog.Debug("Health check: integrity_check returned no rows")
		return false
	}

	// 2. Schema check
	row := db.QueryRow("SELECT name FROM sqlite_master LIMIT 1")
	var name string
	if err := row.Scan(&name); err != nil && err != sql.ErrNoRows {
		slog.Debug("Health check: schema check failed", "error", err)
		return false
	}

	// 3. Write check
	// We attempt to perform a real write inside a transaction and roll it back.
	// This ensures that indices and FTS triggers are actually working.
	tx, err := db.Begin()
	if err != nil {
		slog.Debug("Health check: failed to begin transaction", "error", err)
		return false
	}
	defer tx.Rollback()

	// Check for media table and perform a REAL write if possible
	var hasMedia bool
	_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media')").Scan(&hasMedia)
	if hasMedia {
		var somePath string
		_ = db.QueryRow("SELECT path FROM media LIMIT 1").Scan(&somePath)

		// If the table is not empty, we MUST update a real row to trigger the FTS and index logic.
		// If we only update a non-existent row, the triggers will not fire and we won't detect FTS corruption.
		if somePath != "" {
			if _, err = tx.Exec("UPDATE media SET time_deleted = time_deleted WHERE path = ?", somePath); err != nil {
				slog.Warn("Health check: write consistency check (media triggers) failed", "path", somePath, "error", err)
				return false
			}
		} else {
			if _, err = tx.Exec("UPDATE media SET time_deleted = time_deleted WHERE rowid = -1"); err != nil {
				slog.Warn("Health check: write consistency check (media) failed", "error", err)
				return false
			}
		}
	} else {
		// Generic write check for non-media DBs (e.g. in tests)
		if _, err = tx.Exec("CREATE TEMP TABLE _health_check(id INT); DROP TABLE _health_check;"); err != nil {
			slog.Debug("Health check: generic write check failed", "error", err)
			return false
		}
	}

	// Specifically check FTS virtual table consistency
	var hasFTS bool
	_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_fts')").Scan(&hasFTS)
	if hasFTS {
		if _, err = tx.Exec("SELECT rowid FROM media_fts LIMIT 1"); err != nil && err != sql.ErrNoRows {
			slog.Warn("Health check: FTS check (media_fts) failed", "error", err)
			return false
		}
	}

	return true
}
