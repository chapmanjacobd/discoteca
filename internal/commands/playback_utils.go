package commands

import (
	"os"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// DispatchPlaybackCommand handles common logic for sending commands to mpv or Chromecast
func DispatchPlaybackCommand(
	c models.ControlFlags,
	mpvCmd string,
	mpvArgs []any,
	castCmd string,
	castArgs ...string,
) error {
	cattFile := utils.GetCattNowPlayingFile()
	if utils.FileExists(cattFile) {
		args := append([]string{castCmd}, castArgs...)
		utils.CastCommand(c.CastDevice, args...)
		if castCmd == "stop" {
			os.Remove(cattFile)
		}
	}

	socketPath := utils.GetMpvSocketPath(c.MpvSocket)
	if utils.FileExists(socketPath) {
		args := append([]any{mpvCmd}, mpvArgs...)
		_, err := utils.MpvCall(socketPath, args...)
		return err
	}
	return nil
}
