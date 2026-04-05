package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type MpvControlBase struct {
	models.ControlFlags `embed:""`
}

type NowCmd struct {
	MpvControlBase
}

func (c *NowCmd) Run(ctx context.Context) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		data, err := os.ReadFile(cattFile)
		if err == nil {
			fmt.Printf("Now Playing (Chromecast): %s\n", string(data))
		}
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	pathResp, err := utils.MpvCall(ctx, socketPath, "get_property", "path")
	if err != nil {
		if !utils.FileExists(cattFile) {
			return errors.New("no playback detected (mpv or chromecast)")
		}
		return nil
	}

	path := utils.GetString(pathResp.Data)
	fmt.Printf("Now Playing: %s\n", path)

	posResp, err := utils.MpvCall(ctx, socketPath, "get_property", "time-pos")
	if err == nil && posResp.Data != nil {
		pos := 0.0
		if v, ok := posResp.Data.(float64); ok {
			pos = v
		}
		fmt.Printf("    Playhead: %s\n", utils.SecondsToHHMMSS(int64(pos)))
	}

	durResp, err := utils.MpvCall(ctx, socketPath, "get_property", "duration")
	if err == nil && durResp.Data != nil {
		dur := 0.0
		if v, ok := durResp.Data.(float64); ok {
			dur = v
		}
		fmt.Printf("    Duration: %s\n", utils.SecondsToHHMMSS(int64(dur)))
	}

	return nil
}

type StopCmd struct {
	MpvControlBase
}

func (c *StopCmd) Run(ctx context.Context) error {
	return DispatchPlaybackCommand(ctx, c.ControlFlags, "loadfile", []any{"/dev/null"}, "stop")
}

type PauseCmd struct {
	MpvControlBase
}

func (c *PauseCmd) Run(ctx context.Context) error {
	return DispatchPlaybackCommand(ctx, c.ControlFlags, "cycle", []any{"pause"}, "play_toggle")
}

type NextCmd struct {
	MpvControlBase
}

func (c *NextCmd) Run(ctx context.Context) error {
	// Note: We don't remove cattFile for next because CastPlay loop handles it
	return DispatchPlaybackCommand(ctx, c.ControlFlags, "playlist_next", []any{"force"}, "stop")
}

type SeekCmd struct {
	MpvControlBase

	Time string `help:"Time to seek to (e.g. 10, +10, -10, 00:01:30)" arg:""`
}

func (c *SeekCmd) Run(ctx context.Context) error {
	s := c.Time
	isRelative := false
	isNegative := false

	if strings.HasPrefix(s, "-") {
		isNegative = true
		isRelative = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		isRelative = true
		s = s[1:]
	}

	var seconds float64
	if strings.Contains(s, ":") {
		seconds = utils.FromTimestampSeconds(s)
	} else {
		if f := utils.SafeFloat(s); f != nil {
			seconds = *f
		} else {
			return fmt.Errorf("invalid time format: %s", c.Time)
		}
	}

	if isNegative {
		seconds = -seconds
	}

	castCmd := "seek"
	if isRelative && isNegative {
		castCmd = "rewind"
		seconds = -seconds
	} else if isRelative {
		castCmd = "ffwd"
	}

	mode := "absolute"
	if isRelative {
		mode = "relative"
	} else if !strings.Contains(c.Time, ":") {
		// If it's just a number without +/- or :, Python logic implies it might be relative too
		mode = "relative"
	}

	return DispatchPlaybackCommand(
		ctx,
		c.ControlFlags,
		"seek",
		[]any{seconds, mode},
		castCmd,
		strconv.FormatInt(int64(seconds), 10),
	)
}
