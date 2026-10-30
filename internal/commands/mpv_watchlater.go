package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chapmanjacobd/discoteca/internal/config"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/history"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	mediautil "github.com/chapmanjacobd/discoteca/internal/media"
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

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *MpvWatchlaterCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
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
		watchLaterDir = config.GetMpvWatchLaterDir()
	}

	if !fileutil.DirExists(watchLaterDir) {
		return fmt.Errorf("mpv watch_later directory not found: %s", watchLaterDir)
	}

	// 1. Get all media from databases
	media, err := query.MediaQuery(ctx, c.Databases, flags)
	if err != nil {
		return err
	}

	// 2. Map MD5 hashes to media items
	md5Map := make(map[string]models.MediaWithDB)
	for _, m := range media {
		hash := mediautil.PathToMpvWatchLaterMD5(m.Path)
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
			val, err := mediautil.MpvWatchLaterValue(metadataPath, "start")
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

			if err := history.UpdateHistoryWithTime(
				ctx,
				m.DB,
				[]string{m.Path},
				history.HistoryEntry{
					Playhead:   playhead,
					TimePlayed: timePlayed,
					MarkDone:   false,
				},
			); err != nil {
				logging.Log.Error("Failed to import watchlater", "path", m.Path, "error", err)
			} else {
				count++
				logging.Log.Debug("Imported watchlater", "path", m.Path, "playhead", playhead)
			}
		}
	}

	fmt.Printf("Imported %d watch-later records\n", count)
	return nil
}
