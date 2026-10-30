package commands

import (
	"context"
	"os"

	"github.com/chapmanjacobd/discoteca/internal/config"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/models"
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
	cattFile := config.GetCattNowPlayingFile()
	if fileutil.FileExists(cattFile) {
		args := append([]string{params.CastCmd}, params.CastArgs...)
		if err := media.CastCommand(ctx, c.CastDevice, args...); err != nil {
			logging.Log.Warn("Cast command failed", "error", err)
		}
		if params.CastCmd == "stop" {
			os.Remove(cattFile)
		}
	}

	socketPath := media.GetMpvSocketPath(c.MpvSocket)
	if fileutil.FileExists(socketPath) {
		args := append([]any{params.MpvCmd}, params.MpvArgs...)
		_, err := media.MpvCall(ctx, socketPath, args...)
		return err
	}
	return nil
}
