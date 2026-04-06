package commands

import (
	"context"
	"os"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// DispatchPlaybackCommand handles common logic for sending commands to mpv or Chromecast
func DispatchPlaybackCommand(
	ctx context.Context,
	c models.ControlFlags,
	mpvCmd string,
	mpvArgs []any,
	castCmd string,
	castArgs ...string,
) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		args := append([]string{castCmd}, castArgs...)
		if err := utils.CastCommand(ctx, c.CastDevice, args...); err != nil {
			models.Log.Warn("Cast command failed", "error", err)
		}
		if castCmd == "stop" {
			os.Remove(cattFile)
		}
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	if utils.FileExists(socketPath) {
		args := append([]any{mpvCmd}, mpvArgs...)
		_, err := utils.MpvCall(ctx, socketPath, args...)
		return err
	}
	return nil
}
