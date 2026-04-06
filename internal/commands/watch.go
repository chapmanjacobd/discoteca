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

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *WatchCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	flags := c.buildFlags()
	media, err := c.queryMedia(ctx, flags)
	if err != nil {
		return err
	}

	for i, m := range media {
		if !utils.FileExists(m.Path) {
			continue
		}

		stop, err := c.playMedia(ctx, flags, m)
		if err != nil {
			models.Log.Error("Play media failed", "path", m.Path, "error", err)
		}
		if stop {
			return nil
		}

		if i < len(media)-1 && c.InterdimensionalCable > 0 {
			fmt.Printf("\nChanging channel...\n")
		}
	}

	return nil
}

func (c *WatchCmd) buildFlags() models.GlobalFlags {
	flags := models.BuildQueryGlobalFlags(models.BuildQueryOptions{
		Core:        c.CoreFlags,
		Query:       c.QueryFlags,
		PathFilter:  c.PathFilterFlags,
		Filter:      c.FilterFlags,
		MediaFilter: c.MediaFilterFlags,
		TimeFilter:  c.TimeFilterFlags,
		Deleted:     c.DeletedFlags,
		Sort:        c.SortFlags,
		Display:     c.DisplayFlags,
		FTS:         c.FTSFlags,
	})
	flags.PlaybackFlags = c.PlaybackFlags
	flags.MpvActionFlags = c.MpvActionFlags
	flags.PostActionFlags = c.PostActionFlags
	return flags
}

func (c *WatchCmd) queryMedia(ctx context.Context, flags models.GlobalFlags) ([]models.MediaWithDB, error) {
	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return nil, err
	}

	media = query.FilterMedia(media, flags)
	query.SortMedia(media, flags)
	if c.ReRank != "" {
		media = query.ReRankMedia(media, flags)
	}

	if len(media) == 0 {
		return nil, errors.New("no media found")
	}
	return media, nil
}

func (c *WatchCmd) playMedia(
	ctx context.Context,
	flags models.GlobalFlags,
	m models.MediaWithDB,
) (stop bool, playErr error) {
	args := c.buildMpvArgs(m)

	if c.Cast {
		if err := CastPlay(ctx, flags, []models.MediaWithDB{m}, false); err != nil {
			models.Log.Error("Cast failed", "path", m.Path, "error", err)
		}
		return false, nil
	}

	startTime := time.Now()
	exitCode, err := runPlayer(ctx, args, m.Path)

	if c.TrackHistory {
		if err2 := c.updateHistory(ctx, flags, m, startTime); err2 != nil {
			models.Log.Error("Warning: failed to update history", "path", m.Path, "error", err2)
		}
	}

	if stop := handlePlayerExit(ctx, flags, exitCode, err, m); stop {
		return true, err
	}

	if postErr := ExecutePostAction(ctx, flags, []models.MediaWithDB{m}); postErr != nil {
		models.Log.Error("Post action failed", "path", m.Path, "error", postErr)
	}

	return false, err
}

func (c *WatchCmd) buildMpvArgs(m models.MediaWithDB) []string {
	player := c.OverridePlayer
	if player == "" {
		player = "mpv"
	}
	args := []string{player}

	if player == "mpv" {
		args = c.appendWatchMpvFlags(args)
		args = c.appendSubtitleArgs(args)
		args = c.appendPlaybackArgs(args, mustInt(m.Duration))
	}

	args = append(args, m.Path)
	return args
}

func (c *WatchCmd) appendWatchMpvFlags(args []string) []string {
	if c.Volume > 0 {
		args = append(args, fmt.Sprintf("--volume=%d", c.Volume))
	}
	if c.Fullscreen {
		args = append(args, "--fullscreen")
	}
	if c.Mute {
		args = append(args, "--mute=yes")
	}
	if c.Loop {
		args = append(args, "--loop-file=inf")
	}
	if c.SavePlayhead {
		args = append(args, "--save-position-on-quit")
	}

	ipcSocket := c.MpvSocket
	if ipcSocket == "" {
		ipcSocket = utils.GetMpvWatchSocket()
	}
	args = append(args, fmt.Sprintf("--input-ipc-server=%s", ipcSocket))
	return args
}

func (c *WatchCmd) appendSubtitleArgs(args []string) []string {
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
	return args
}

func (c *WatchCmd) appendPlaybackArgs(args []string, duration int) []string {
	if c.Speed != 1.0 {
		args = append(args, fmt.Sprintf("--speed=%.2f", c.Speed))
	}

	start, end := c.getStartEnd(duration)
	if start != "" {
		args = append(args, fmt.Sprintf("--start=%s", start))
	}
	if end != "" {
		args = append(args, fmt.Sprintf("--end=%s", end))
	}
	return args
}

func (c *WatchCmd) getStartEnd(duration int) (start, end string) {
	start = c.Start
	end = c.End
	if c.InterdimensionalCable > 0 && duration > c.InterdimensionalCable {
		s := utils.RandomInt(0, duration-c.InterdimensionalCable)
		start = strconv.Itoa(s)
		end = strconv.Itoa(s + c.InterdimensionalCable)
	}
	return start, end
}

func (c *WatchCmd) updateHistory(
	ctx context.Context,
	flags models.GlobalFlags,
	m models.MediaWithDB,
	startTime time.Time,
) error {
	mediaDuration := mustInt(m.Duration)
	existingPlayhead := mustInt(m.Playhead)
	playhead := utils.GetPlayhead(flags, m.Path, startTime, existingPlayhead, mediaDuration)
	return history.UpdateHistorySimple(ctx, m.DB, []string{m.Path}, playhead, false)
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

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *ListenCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)
	flags := c.buildFlags()
	media, err := c.queryMedia(ctx, flags)
	if err != nil {
		return err
	}

	for _, m := range media {
		if !utils.FileExists(m.Path) {
			continue
		}

		stop, err := c.playMedia(ctx, flags, m)
		if err != nil {
			models.Log.Error("Play media failed", "path", m.Path, "error", err)
		}
		if stop {
			return nil
		}
	}

	return nil
}

func (c *ListenCmd) buildFlags() models.GlobalFlags {
	flags := models.BuildQueryGlobalFlags(models.BuildQueryOptions{
		Core:        c.CoreFlags,
		Query:       c.QueryFlags,
		PathFilter:  c.PathFilterFlags,
		Filter:      c.FilterFlags,
		MediaFilter: c.MediaFilterFlags,
		TimeFilter:  c.TimeFilterFlags,
		Deleted:     c.DeletedFlags,
		Sort:        c.SortFlags,
		Display:     c.DisplayFlags,
		FTS:         c.FTSFlags,
	})
	flags.PlaybackFlags = c.PlaybackFlags
	flags.MpvActionFlags = c.MpvActionFlags
	flags.PostActionFlags = c.PostActionFlags
	return flags
}

func (c *ListenCmd) queryMedia(ctx context.Context, flags models.GlobalFlags) ([]models.MediaWithDB, error) {
	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return nil, err
	}

	media = query.FilterMedia(media, flags)
	query.SortMedia(media, flags)
	if c.ReRank != "" {
		media = query.ReRankMedia(media, flags)
	}

	if len(media) == 0 {
		return nil, errors.New("no media found")
	}
	return media, nil
}

func (c *ListenCmd) playMedia(
	ctx context.Context,
	flags models.GlobalFlags,
	m models.MediaWithDB,
) (stop bool, playErr error) {
	args := c.buildMpvArgs(m)

	if c.Cast {
		if err := CastPlay(ctx, flags, []models.MediaWithDB{m}, true); err != nil {
			models.Log.Error("Cast failed", "path", m.Path, "error", err)
		}
		return false, nil
	}

	startTime := time.Now()
	exitCode, err := runPlayer(ctx, args, m.Path)

	if c.TrackHistory {
		if err2 := c.updateHistory(ctx, flags, m, startTime); err2 != nil {
			models.Log.Warn("Failed to update history", "error", err2)
		}
	}

	if stop := handlePlayerExit(ctx, flags, exitCode, err, m); stop {
		return true, err
	}

	if postErr := ExecutePostAction(ctx, flags, []models.MediaWithDB{m}); postErr != nil {
		models.Log.Error("Post action failed", "path", m.Path, "error", postErr)
	}

	return false, err
}

func (c *ListenCmd) buildMpvArgs(m models.MediaWithDB) []string {
	player := c.OverridePlayer
	if player == "" {
		player = "mpv"
	}
	args := []string{player}

	if player == "mpv" {
		args = append(args, "--video=no")
		args = c.appendListenMpvFlags(args)
		args = c.appendPlaybackArgs(args, mustInt(m.Duration))
	}

	args = append(args, m.Path)
	return args
}

func (c *ListenCmd) appendListenMpvFlags(args []string) []string {
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

	ipcSocket := c.MpvSocket
	if ipcSocket == "" {
		ipcSocket = utils.GetMpvWatchSocket()
	}
	args = append(args, fmt.Sprintf("--input-ipc-server=%s", ipcSocket))
	return args
}

func (c *ListenCmd) appendPlaybackArgs(args []string, duration int) []string {
	start, end := c.getStartEnd(duration)
	if start != "" {
		args = append(args, fmt.Sprintf("--start=%s", start))
	}
	if end != "" {
		args = append(args, fmt.Sprintf("--end=%s", end))
	}
	return args
}

func (c *ListenCmd) getStartEnd(duration int) (start, end string) {
	start = c.Start
	end = c.End
	if c.InterdimensionalCable > 0 && duration > c.InterdimensionalCable {
		s := utils.RandomInt(0, duration-c.InterdimensionalCable)
		start = strconv.Itoa(s)
		end = strconv.Itoa(s + c.InterdimensionalCable)
	}
	return start, end
}

func (c *ListenCmd) updateHistory(
	ctx context.Context,
	flags models.GlobalFlags,
	m models.MediaWithDB,
	startTime time.Time,
) error {
	mediaDuration := mustInt(m.Duration)
	existingPlayhead := mustInt(m.Playhead)
	playhead := utils.GetPlayhead(flags, m.Path, startTime, existingPlayhead, mediaDuration)
	return history.UpdateHistorySimple(ctx, m.DB, []string{m.Path}, playhead, false)
}

// Shared helpers

func mustInt(p *int64) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func runPlayer(ctx context.Context, args []string, _ string) (exitCode int, err error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		exitError := &exec.ExitError{}
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		}
	}
	return exitCode, err
}

func handlePlayerExit(
	ctx context.Context,
	flags models.GlobalFlags,
	exitCode int,
	_ error,
	m models.MediaWithDB,
) bool {
	if exitCode == 4 {
		return true
	}

	if err := RunExitCommand(ctx, flags, exitCode, m.Path); err != nil {
		models.Log.Error("Exit command failed", "code", exitCode, "error", err)
	}

	if flags.MpvActionFlags.Interactive {
		if err := InteractiveDecision(ctx, flags, m); err != nil {
			if errors.Is(err, ErrUserQuit) {
				return true
			}
			models.Log.Error("Interactive decision failed", "error", err)
		}
	}

	return false
}
