package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/history"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/shellquote"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// ExecutePostAction executes actions after a command
func ExecutePostAction(flags models.GlobalFlags, media []models.MediaWithDB) error {
	action := flags.PostAction

	if flags.DeleteFiles {
		action = "delete"
	} else if flags.MarkDeleted {
		action = "mark-deleted"
	} else if flags.MoveTo != "" {
		action = "move"
	} else if flags.CopyTo != "" {
		action = "copy"
	} else if flags.Trash {
		action = "trash"
	}

	if action == "" || action == "none" {
		return nil
	}

	var sizeLimit int64 = 0
	if flags.ActionSize != "" {
		if sl, err := utils.HumanToBytes(flags.ActionSize); err == nil {
			sizeLimit = sl
		}
	}

	var totalSize int64 = 0
	count := 0

	for _, m := range media {
		if flags.ActionLimit > 0 && count >= flags.ActionLimit {
			models.Log.Info("Action limit reached", "limit", flags.ActionLimit)
			break
		}
		if sizeLimit > 0 && totalSize >= sizeLimit {
			models.Log.Info("Action size limit reached", "limit", flags.ActionSize)
			break
		}

		var err error
		var size int64 = 0
		if m.Size != nil {
			size = *m.Size
		}

		switch action {
		case "delete":
			err = DeleteMediaItem(m)
		case "mark-deleted":
			err = MarkDeletedItem(m)
		case "move":
			err = MoveMediaItem(flags.MoveTo, m)
		case "copy":
			err = CopyMediaItem(flags.CopyTo, m)
		case "trash":
			err = utils.Trash(flags, m.Path)
		}

		if err != nil {
			models.Log.Error("Post-action failed", "path", m.Path, "error", err)
		} else {
			count++
			totalSize += size
		}
	}

	if count > 0 {
		fmt.Printf("\n%s %d files (%s total)\n", action, count, utils.FormatSize(totalSize))
	}

	return nil
}

func RunExitCommand(flags models.GlobalFlags, exitCode int, path string) error {
	var cmdStr string
	switch exitCode {
	case 0:
		cmdStr = flags.Cmd0
	case 1:
		cmdStr = flags.Cmd1
	case 2:
		cmdStr = flags.Cmd2
	case 3:
		cmdStr = flags.Cmd3
	case 4:
		cmdStr = flags.Cmd4
	case 5:
		cmdStr = flags.Cmd5
	case 6:
		cmdStr = flags.Cmd6
	case 7:
		cmdStr = flags.Cmd7
	case 8:
		cmdStr = flags.Cmd8
	case 9:
		cmdStr = flags.Cmd9
	case 10:
		cmdStr = flags.Cmd10
	case 11:
		cmdStr = flags.Cmd11
	case 12:
		cmdStr = flags.Cmd12
	case 13:
		cmdStr = flags.Cmd13
	case 14:
		cmdStr = flags.Cmd14
	case 15:
		cmdStr = flags.Cmd15
	case 20:
		cmdStr = flags.Cmd20
	case 127:
		cmdStr = flags.Cmd127
	}

	if cmdStr == "" {
		return nil
	}

	// Replace {} with path
	cmdStr = strings.ReplaceAll(cmdStr, "{}", shellquote.ShellQuote(path))

	models.Log.Info("Running exit command", "code", exitCode, "command", cmdStr)
	cmd := exec.CommandContext(context.Background(), "bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeleteMediaItem(m models.MediaWithDB) error {
	if utils.FileExists(m.Path) {
		if err := os.Remove(m.Path); err != nil {
			return err
		}
		fmt.Printf("Deleted: %s\n", m.Path)
	}
	return nil
}

func MarkDeletedItem(m models.MediaWithDB) error {
	sqlDB, err := db.Connect(m.DB)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	now := time.Now().Unix()
	_, err = sqlDB.ExecContext(context.Background(), "UPDATE media SET time_deleted = ? WHERE path = ?", now, m.Path)
	if err == nil {
		fmt.Printf("Marked deleted: %s\n", m.Path)
	}
	return err
}

func MoveMediaItem(destDir string, m models.MediaWithDB) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	if !utils.FileExists(m.Path) {
		return errors.New("file not found")
	}

	dest := filepath.Join(destDir, filepath.Base(m.Path))
	if err := os.Rename(m.Path, dest); err != nil {
		return err
	}

	// Update database
	sqlDB, err := db.Connect(m.DB)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	_, err = sqlDB.ExecContext(context.Background(), "UPDATE media SET path = ? WHERE path = ?", dest, m.Path)
	if err == nil {
		fmt.Printf("Moved: %s -> %s\n", m.Path, dest)
	}
	return err
}

func CopyMediaItem(destDir string, m models.MediaWithDB) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	if !utils.FileExists(m.Path) {
		return errors.New("file not found")
	}

	dest := filepath.Join(destDir, filepath.Base(m.Path))
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return err
	}

	fmt.Printf("Copied: %s -> %s\n", m.Path, dest)
	return nil
}

func CastPlay(ctx context.Context, flags models.GlobalFlags, media []models.MediaWithDB, audioOnly bool) error {
	for _, m := range media {
		if !utils.FileExists(m.Path) {
			continue
		}

		models.Log.Info("Casting", "path", m.Path)
		os.WriteFile(utils.GetCattNowPlayingFile(), []byte(m.Path), 0o644)

		args := []string{"catt"}
		if flags.CastDevice != "" {
			args = append(args, "-d", flags.CastDevice)
		}
		args = append(args, "cast")
		if audioOnly || flags.NoSubtitles {
			args = append(args, "--no-subs")
		}
		if flags.Start != "" {
			// Convert start time to seconds if needed
			seconds := flags.Start
			if strings.Contains(flags.Start, ":") {
				seconds = strconv.FormatInt(int64(utils.FromTimestampSeconds(flags.Start)), 10)
			}
			args = append(args, "--seek-to", seconds)
		}
		args = append(args, m.Path)
		startTime := time.Now()

		if flags.CastWithLocal {
			// Start catt in background
			cattCmd := exec.CommandContext(ctx, args[0], args[1:]...)
			cattCmd.Start()

			// Wait a bit for sync (lazy sync as in Python version)
			time.Sleep(974 * time.Millisecond)

			// Start local mpv
			localArgs := []string{"mpv"}
			if audioOnly {
				localArgs = append(localArgs, "--video=no")
			}
			localArgs = append(localArgs, m.Path)
			localCmd := exec.CommandContext(ctx, localArgs[0], localArgs[1:]...)
			localCmd.Stdout = os.Stdout
			localCmd.Stderr = os.Stderr
			localCmd.Stdin = os.Stdin
			localCmd.Run()

			// Wait for catt to finish if it hasn't
			cattCmd.Wait()
		} else {
			cmd := exec.CommandContext(ctx, args[0], args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				models.Log.Error("catt failed", "error", err)
			}
		}

		if flags.TrackHistory {
			mediaDuration := 0
			if m.Duration != nil {
				mediaDuration = int(*m.Duration)
			}
			existingPlayhead := 0
			if m.Playhead != nil {
				existingPlayhead = int(*m.Playhead)
			}
			playhead := utils.GetPlayhead(flags, m.Path, startTime, existingPlayhead, mediaDuration)
			history.UpdateHistorySimple(ctx, m.DB, []string{m.Path}, playhead, false)
		}
	}
	os.Remove(utils.GetCattNowPlayingFile())
	return nil
}
