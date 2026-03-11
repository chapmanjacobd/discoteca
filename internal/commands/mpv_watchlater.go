package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/history"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type MpvWatchlaterCmd struct {
	models.CoreFlags        `embed:""`
	models.MediaFilterFlags `embed:""`
	models.PathFilterFlags  `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.PlaybackFlags    `embed:""`

	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

func (c *MpvWatchlaterCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		PathFilterFlags:  c.PathFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		PlaybackFlags:    c.PlaybackFlags,
	}

	watchLaterDir := c.WatchLaterDir
	if watchLaterDir == "" {
		watchLaterDir = utils.GetMpvWatchLaterDir()
	}

	if !utils.DirExists(watchLaterDir) {
		return fmt.Errorf("mpv watch_later directory not found: %s", watchLaterDir)
	}

	// 1. Get all media from databases
	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		return err
	}

	// 2. Map MD5 hashes to media items
	md5Map := make(map[string]models.MediaWithDB)
	for _, m := range media {
		hash := utils.PathToMpvWatchLaterMD5(m.Path)
		md5Map[hash] = m
	}

	// 3. Scan watch_later directory
	entries, err := os.ReadDir(watchLaterDir)
	if err != nil {
		return err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		hash := entry.Name()
		if m, ok := md5Map[hash]; ok {
			metadataPath := filepath.Join(watchLaterDir, hash)

			// Get playhead
			val, err := utils.MpvWatchLaterValue(metadataPath, "start")
			if err != nil || val == "" {
				continue
			}

			playhead := 0
			if f := utils.SafeFloat(val); f != nil {
				playhead = int(*f)
			}

			// Get file times
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// We use mtime as time_played
			timePlayed := info.ModTime().Unix()

			if err := history.UpdateHistoryWithTime(m.DB, []string{m.Path}, playhead, timePlayed, false); err != nil {
				slog.Error("Failed to import watchlater", "path", m.Path, "error", err)
			} else {
				count++
				slog.Debug("Imported watchlater", "path", m.Path, "playhead", playhead)
			}
		}
	}

	fmt.Printf("Imported %d watch-later records\n", count)
	return nil
}
