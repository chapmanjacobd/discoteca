package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/history"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type WatchCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.FTSFlags         `embed:""`
	models.PlaybackFlags    `embed:""`
	models.MpvActionFlags   `embed:""`
	models.PostActionFlags  `embed:""`

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *WatchCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.BuildQueryGlobalFlags(
		c.CoreFlags,
		c.QueryFlags,
		c.PathFilterFlags,
		c.FilterFlags,
		c.MediaFilterFlags,
		c.TimeFilterFlags,
		c.DeletedFlags,
		c.SortFlags,
		c.DisplayFlags,
		c.FTSFlags,
	)
	flags.PlaybackFlags = c.PlaybackFlags
	flags.MpvActionFlags = c.MpvActionFlags
	flags.PostActionFlags = c.PostActionFlags
	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)
	query.SortMedia(media, flags)
	if c.ReRank != "" {
		media = query.ReRankMedia(media, flags)
	}

	if len(media) == 0 {
		return errors.New("no media found")
	}

	for i, m := range media {
		if !utils.FileExists(m.Path) {
			continue
		}

		// Build player command
		player := c.OverridePlayer
		if player == "" {
			player = "mpv"
		}
		args := []string{player}

		if player == "mpv" {
			if c.Volume > 0 {
				args = append(args, fmt.Sprintf("--volume=%d", c.Volume))
			}
			if c.Fullscreen {
				args = append(args, "--fullscreen")
			}

			// Subtitle Mix logic
			useSubs := !c.NoSubtitles
			if useSubs && c.SubtitleMix > 0 {
				if utils.RandomFloat() < c.SubtitleMix {
					useSubs = false
				}
			}

			if !useSubs {
				args = append(args, "--no-sub")
				args = append(args, c.PlayerArgsNoSub...)
			} else {
				args = append(args, c.PlayerArgsSub...)
			}

			if c.Speed != 1.0 {
				args = append(args, fmt.Sprintf("--speed=%.2f", c.Speed))
			}

			// Start/End and Interdimensional Cable
			start := c.Start
			end := c.End
			if c.InterdimensionalCable > 0 {
				duration := 0
				if m.Duration != nil {
					duration = int(*m.Duration)
				}
				if duration > c.InterdimensionalCable {
					s := utils.RandomInt(0, duration-c.InterdimensionalCable)
					start = strconv.Itoa(s)
					end = strconv.Itoa(s + c.InterdimensionalCable)
				}
			}

			if start != "" {
				args = append(args, fmt.Sprintf("--start=%s", start))
			}
			if end != "" {
				args = append(args, fmt.Sprintf("--end=%s", end))
			}

			if c.SavePlayhead {
				args = append(args, "--save-position-on-quit")
			}
			if c.Mute {
				args = append(args, "--mute=yes")
			}
			if c.Loop {
				args = append(args, "--loop-file=inf")
			}

			ipcSocket := c.MpvSocket
			if ipcSocket == "" {
				ipcSocket = utils.GetMpvWatchSocket()
			}
			args = append(args, fmt.Sprintf("--input-ipc-server=%s", ipcSocket))
		}
		args = append(args, m.Path)

		if c.Cast {
			// CastPlay handles its own loop, but we want to handle one by one for Cable?
			// For now, let's just call it with the single item
			if err := CastPlay(ctx, flags, []models.MediaWithDB{m}, false); err != nil {
				models.Log.Error("Cast failed", "path", m.Path, "error", err)
			}
			continue
		}

		// Execute mpv
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		startTime := time.Now()
		err := cmd.Run()

		// Update history
		if c.TrackHistory {
			mediaDuration := 0
			if m.Duration != nil {
				mediaDuration = int(*m.Duration)
			}
			existingPlayhead := 0
			if m.Playhead != nil {
				existingPlayhead = int(*m.Playhead)
			}
			playhead := utils.GetPlayhead(flags, m.Path, startTime, existingPlayhead, mediaDuration)

			if err := history.UpdateHistorySimple(ctx, m.DB, []string{m.Path}, playhead, false); err != nil {
				models.Log.Error("Warning: failed to update history", "path", m.Path, "error", err)
			}
		}

		// Handle Exit Code Hooks
		exitCode := 0
		if err != nil {
			exitError := &exec.ExitError{}
			if errors.As(err, &exitError) {
				exitCode = exitError.ExitCode()
			} else {
				return err
			}
		}

		if exitCode == 4 {
			return nil
		}

		if err := RunExitCommand(flags, exitCode, m.Path); err != nil {
			models.Log.Error("Exit command failed", "code", exitCode, "error", err)
		}

		// Interactive decision
		if c.Interactive {
			if err := InteractiveDecision(ctx, flags, m); err != nil {
				if errors.Is(err, ErrUserQuit) {
					return nil
				}
				models.Log.Error("Interactive decision failed", "error", err)
			}
		}

		// Execute post action for this item
		if err := ExecutePostAction(ctx, flags, []models.MediaWithDB{m}); err != nil {
			models.Log.Error("Post action failed", "path", m.Path, "error", err)
		}

		if i < len(media)-1 && c.InterdimensionalCable > 0 {
			fmt.Printf("\nChanging channel...\n")
		}
	}

	return nil
}

type ListenCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.FTSFlags         `embed:""`
	models.PlaybackFlags    `embed:""`
	models.MpvActionFlags   `embed:""`
	models.PostActionFlags  `embed:""`

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *ListenCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.BuildQueryGlobalFlags(
		c.CoreFlags,
		c.QueryFlags,
		c.PathFilterFlags,
		c.FilterFlags,
		c.MediaFilterFlags,
		c.TimeFilterFlags,
		c.DeletedFlags,
		c.SortFlags,
		c.DisplayFlags,
		c.FTSFlags,
	)
	flags.PlaybackFlags = c.PlaybackFlags
	flags.MpvActionFlags = c.MpvActionFlags
	flags.PostActionFlags = c.PostActionFlags
	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)
	query.SortMedia(media, flags)
	if c.ReRank != "" {
		media = query.ReRankMedia(media, flags)
	}

	if len(media) == 0 {
		return errors.New("no media found")
	}

	for _, m := range media {
		if !utils.FileExists(m.Path) {
			continue
		}

		player := c.OverridePlayer
		if player == "" {
			player = "mpv"
		}
		args := []string{player}

		if player == "mpv" {
			args = append(args, "--video=no")
			if c.Volume > 0 {
				args = append(args, fmt.Sprintf("--volume=%d", c.Volume))
			}
			if c.Speed != 1.0 {
				args = append(args, fmt.Sprintf("--speed=%.2f", c.Speed))
			}
			if c.Mute {
				args = append(args, "--mute=yes")
			}
			if c.Loop {
				args = append(args, "--loop-file=inf")
			}

			// Interdimensional Cable for audio too? why not.
			start := c.Start
			end := c.End
			if c.InterdimensionalCable > 0 {
				duration := 0
				if m.Duration != nil {
					duration = int(*m.Duration)
				}
				if duration > c.InterdimensionalCable {
					s := utils.RandomInt(0, duration-c.InterdimensionalCable)
					start = strconv.Itoa(s)
					end = strconv.Itoa(s + c.InterdimensionalCable)
				}
			}
			if start != "" {
				args = append(args, fmt.Sprintf("--start=%s", start))
			}
			if end != "" {
				args = append(args, fmt.Sprintf("--end=%s", end))
			}

			ipcSocket := c.MpvSocket
			if ipcSocket == "" {
				ipcSocket = utils.GetMpvWatchSocket()
			}
			args = append(args, fmt.Sprintf("--input-ipc-server=%s", ipcSocket))
		}
		args = append(args, m.Path)

		if c.Cast {
			if err := CastPlay(ctx, flags, []models.MediaWithDB{m}, true); err != nil {
				models.Log.Error("Cast failed", "path", m.Path, "error", err)
			}
			continue
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		startTime := time.Now()
		err := cmd.Run()

		if c.TrackHistory {
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

		exitCode := 0
		if err != nil {
			exitError := &exec.ExitError{}
			if errors.As(err, &exitError) {
				exitCode = exitError.ExitCode()
			}
		}

		if exitCode == 4 {
			return nil
		}

		if err := RunExitCommand(flags, exitCode, m.Path); err != nil {
			models.Log.Error("Exit command failed", "code", exitCode, "error", err)
		}

		if c.Interactive {
			if err := InteractiveDecision(ctx, flags, m); err != nil {
				if errors.Is(err, ErrUserQuit) {
					return nil
				}
				models.Log.Error("Interactive decision failed", "error", err)
			}
		}

		if err := ExecutePostAction(ctx, flags, []models.MediaWithDB{m}); err != nil {
			models.Log.Error("Post action failed", "path", m.Path, "error", err)
		}
	}

	return nil
}
