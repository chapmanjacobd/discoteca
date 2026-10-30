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
	return runPlayback(ctx, c, playbackOptions{
		subtitles:    true,
		fullscreen:   true,
		savePlayhead: true,
		channelGap:   true,
	})
}

func (c *WatchCmd) flagSets() playbackFlagSets {
	return playbackFlagSets{
		core:        c.CoreFlags,
		query:       c.QueryFlags,
		pathFilter:  c.PathFilterFlags,
		filter:      c.FilterFlags,
		mediaFilter: c.MediaFilterFlags,
		timeFilter:  c.TimeFilterFlags,
		deleted:     c.DeletedFlags,
		sort:        c.SortFlags,
		display:     c.DisplayFlags,
		fts:         c.FTSFlags,
		playback:    c.PlaybackFlags,
		mpvAction:   c.MpvActionFlags,
		postAction:  c.PostActionFlags,
	}
}

func (c *WatchCmd) databases() []string { return c.Databases }

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
	return runPlayback(ctx, c, playbackOptions{audioOnly: true})
}

func (c *ListenCmd) flagSets() playbackFlagSets {
	return playbackFlagSets{
		core:        c.CoreFlags,
		query:       c.QueryFlags,
		pathFilter:  c.PathFilterFlags,
		filter:      c.FilterFlags,
		mediaFilter: c.MediaFilterFlags,
		timeFilter:  c.TimeFilterFlags,
		deleted:     c.DeletedFlags,
		sort:        c.SortFlags,
		display:     c.DisplayFlags,
		fts:         c.FTSFlags,
		playback:    c.PlaybackFlags,
		mpvAction:   c.MpvActionFlags,
		postAction:  c.PostActionFlags,
	}
}

func (c *ListenCmd) databases() []string { return c.Databases }

// playbackFlagSets is the common flag set embedded by watch and listen. Keeping
// the conversion here prevents the two commands from drifting apart.
type playbackFlagSets struct {
	core        models.CoreFlags
	query       models.QueryFlags
	pathFilter  models.PathFilterFlags
	filter      models.FilterFlags
	mediaFilter models.MediaFilterFlags
	timeFilter  models.TimeFilterFlags
	deleted     models.DeletedFlags
	sort        models.SortFlags
	display     models.DisplayFlags
	fts         models.FTSFlags
	playback    models.PlaybackFlags
	mpvAction   models.MpvActionFlags
	postAction  models.PostActionFlags
}

func (s playbackFlagSets) globalFlags() models.GlobalFlags {
	flags := models.BuildQueryGlobalFlags(models.BuildQueryOptions{
		Core:        s.core,
		Query:       s.query,
		PathFilter:  s.pathFilter,
		Filter:      s.filter,
		MediaFilter: s.mediaFilter,
		TimeFilter:  s.timeFilter,
		Deleted:     s.deleted,
		Sort:        s.sort,
		Display:     s.display,
		FTS:         s.fts,
	})
	flags.PlaybackFlags = s.playback
	flags.MpvActionFlags = s.mpvAction
	flags.PostActionFlags = s.postAction
	return flags
}

type playbackCommand interface {
	flagSets() playbackFlagSets
	databases() []string
}

// playbackOptions records the intentional differences between watch and listen.
type playbackOptions struct {
	audioOnly    bool
	subtitles    bool
	fullscreen   bool
	savePlayhead bool
	channelGap   bool
}

func runPlayback(ctx context.Context, cmd playbackCommand, opts playbackOptions) error {
	flags := cmd.flagSets().globalFlags()
	media, err := queryPlaybackMedia(ctx, cmd.databases(), flags)
	if err != nil {
		return err
	}

	for i, m := range media {
		if !utils.FileExists(m.Path) {
			continue
		}

		stop, err := playMedia(ctx, flags, m, opts)
		if err != nil {
			models.Log.Error("Play media failed", "path", m.Path, "error", err)
		}
		if stop {
			return nil
		}

		if opts.channelGap && i < len(media)-1 && flags.InterdimensionalCable > 0 {
			fmt.Printf("\nChanging channel...\n")
		}
	}

	return nil
}

func queryPlaybackMedia(ctx context.Context, databases []string, flags models.GlobalFlags) ([]models.MediaWithDB, error) {
	media, err := query.MediaQuery(ctx, databases, flags)
	if err != nil {
		return nil, err
	}

	media = query.FilterMedia(media, flags)
	query.SortMedia(media, flags)

	if len(media) == 0 {
		return nil, errors.New("no media found")
	}
	return media, nil
}

func playMedia(
	ctx context.Context,
	flags models.GlobalFlags,
	m models.MediaWithDB,
	opts playbackOptions,
) (stop bool, playErr error) {
	args := buildPlayerArgs(m, flags.PlaybackFlags, opts)

	if flags.Cast {
		if err := CastPlay(ctx, flags, []models.MediaWithDB{m}, opts.audioOnly); err != nil {
			models.Log.Error("Cast failed", "path", m.Path, "error", err)
		}
		return false, nil
	}

	startTime := time.Now()
	exitCode, err := runPlayer(ctx, args)

	if flags.TrackHistory {
		if historyErr := updatePlaybackHistory(ctx, flags, m, startTime); historyErr != nil {
			models.Log.Warn("Failed to update history", "path", m.Path, "error", historyErr)
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

func buildPlayerArgs(m models.MediaWithDB, playback models.PlaybackFlags, opts playbackOptions) []string {
	player := playback.OverridePlayer
	if player == "" {
		player = "mpv"
	}
	args := []string{player}

	if player == "mpv" {
		if opts.audioOnly {
			args = append(args, "--video=no")
		}
		if playback.Volume > 0 {
			args = append(args, fmt.Sprintf("--volume=%d", playback.Volume))
		}
		if opts.fullscreen && playback.Fullscreen {
			args = append(args, "--fullscreen")
		}
		if playback.Speed != 1.0 {
			args = append(args, fmt.Sprintf("--speed=%.2f", playback.Speed))
		}
		if playback.Mute {
			args = append(args, "--mute=yes")
		}
		if playback.Loop {
			args = append(args, "--loop-file=inf")
		}
		if opts.savePlayhead && playback.SavePlayhead {
			args = append(args, "--save-position-on-quit")
		}

		ipcSocket := playback.MpvSocket
		if ipcSocket == "" {
			ipcSocket = utils.GetMpvWatchSocket()
		}
		args = append(args, fmt.Sprintf("--input-ipc-server=%s", ipcSocket))

		if opts.subtitles {
			args = appendSubtitleArgs(args, playback)
		}

		start, end := startEnd(playback, mustInt(m.Duration))
		if start != "" {
			args = append(args, fmt.Sprintf("--start=%s", start))
		}
		if end != "" {
			args = append(args, fmt.Sprintf("--end=%s", end))
		}
	}

	args = append(args, m.Path)
	return args
}

func appendSubtitleArgs(args []string, playback models.PlaybackFlags) []string {
	useSubs := !playback.NoSubtitles
	if useSubs && playback.SubtitleMix > 0 {
		if utils.RandomFloat() < playback.SubtitleMix {
			useSubs = false
		}
	}

	if !useSubs {
		args = append(args, "--no-sub")
		args = append(args, playback.PlayerArgsNoSub...)
	} else {
		args = append(args, playback.PlayerArgsSub...)
	}
	return args
}

func startEnd(playback models.PlaybackFlags, duration int) (start, end string) {
	start = playback.Start
	end = playback.End
	if playback.InterdimensionalCable > 0 && duration > playback.InterdimensionalCable {
		s := utils.RandomInt(0, duration-playback.InterdimensionalCable)
		start = strconv.Itoa(s)
		end = strconv.Itoa(s + playback.InterdimensionalCable)
	}
	return start, end
}

func updatePlaybackHistory(
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

func runPlayer(ctx context.Context, args []string) (exitCode int, err error) {
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
