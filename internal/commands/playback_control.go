package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/utils"
)

type MpvControlBase struct {
	models.ControlFlags `embed:""`
}

type NowCmd struct {
	MpvControlBase
}

func (c *NowCmd) Run(ctx *kong.Context) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		data, err := os.ReadFile(cattFile)
		if err == nil {
			fmt.Printf("Now Playing (Chromecast): %s\n", string(data))
		}
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	pathResp, err := utils.MpvCall(socketPath, "get_property", "path")
	if err != nil {
		if !utils.FileExists(cattFile) {
			return fmt.Errorf("no playback detected (mpv or chromecast)")
		}
		return nil
	}

	path := utils.GetString(pathResp.Data)
	fmt.Printf("Now Playing: %s\n", path)

	posResp, err := utils.MpvCall(socketPath, "get_property", "time-pos")
	if err == nil && posResp.Data != nil {
		pos := 0.0
		switch v := posResp.Data.(type) {
		case float64:
			pos = v
		}
		fmt.Printf("    Playhead: %s\n", utils.SecondsToHHMMSS(int64(pos)))
	}

	durResp, err := utils.MpvCall(socketPath, "get_property", "duration")
	if err == nil && durResp.Data != nil {
		dur := 0.0
		switch v := durResp.Data.(type) {
		case float64:
			dur = v
		}
		fmt.Printf("    Duration: %s\n", utils.SecondsToHHMMSS(int64(dur)))
	}

	return nil
}

type StopCmd struct {
	MpvControlBase
}

func (c *StopCmd) Run(ctx *kong.Context) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		utils.CastCommand(c.CastDevice, "stop")
		os.Remove(cattFile)
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	if !utils.FileExists(socketPath) {
		return nil
	}

	// Making mpv exit with code 3 like Python version (loadfile /dev/null)
	utils.MpvCall(socketPath, "loadfile", "/dev/null")
	// Also delete the socket file as Python version does
	os.Remove(socketPath)
	return nil
}

type PauseCmd struct {
	MpvControlBase
}

func (c *PauseCmd) Run(ctx *kong.Context) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		utils.CastCommand(c.CastDevice, "play_toggle")
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	if utils.FileExists(socketPath) {
		utils.MpvCall(socketPath, "cycle", "pause")
	}
	return nil
}

type NextCmd struct {
	MpvControlBase
}

func (c *NextCmd) Run(ctx *kong.Context) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		utils.CastCommand(c.CastDevice, "stop")
		// We don't remove cattFile here because CastPlay loop will handle it or update it
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	if utils.FileExists(socketPath) {
		utils.MpvCall(socketPath, "playlist_next", "force")
	}
	return nil
}

type SeekCmd struct {
	MpvControlBase
	Time string `arg:"" help:"Time to seek to (e.g. 10, +10, -10, 00:01:30)"`
}

func (c *SeekCmd) Run(ctx *kong.Context) error {
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

	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		var cmd string
		if isRelative && isNegative {
			cmd = "rewind"
			seconds = -seconds
		} else if isRelative {
			cmd = "ffwd"
		} else {
			cmd = "seek"
		}
		utils.CastCommand(c.CastDevice, cmd, fmt.Sprintf("%d", int64(seconds)))
	}

	mode := "absolute"
	if isRelative {
		mode = "relative"
	} else if !strings.Contains(c.Time, ":") {
		// If it's just a number without +/- or :, Python logic implies it might be relative too
		mode = "relative"
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	if utils.FileExists(socketPath) {
		utils.MpvCall(socketPath, "seek", seconds, mode)
	}
	return nil
}
