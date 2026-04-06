package commands

import (
	"context"
	"os"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// PlaybackCommandParams contains the command and arguments for mpv and chromecast playback
type PlaybackCommandParams struct {
	MpvCmd   string
	MpvArgs  []any
	CastCmd  string
	CastArgs []string
}

// DispatchPlaybackCommand handles common logic for sending commands to mpv or Chromecast
func DispatchPlaybackCommand(
	ctx context.Context,
	c models.ControlFlags,
	params PlaybackCommandParams,
) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		args := append([]string{params.CastCmd}, params.CastArgs...)
		if err := utils.CastCommand(ctx, c.CastDevice, args...); err != nil {
			models.Log.Warn("Cast command failed", "error", err)
		}
		if params.CastCmd == "stop" {
			os.Remove(cattFile)
		}
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	if utils.FileExists(socketPath) {
		args := append([]any{params.MpvCmd}, params.MpvArgs...)
		_, err := utils.MpvCall(ctx, socketPath, args...)
		return err
	}
	return nil
}
