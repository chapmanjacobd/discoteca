package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/metadata"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type MediaCheckCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.DeletedFlags     `embed:""`

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`

	ChunkSize         float64 `help:"Chunk size in seconds. If set, recommended to use >0.1 seconds"                                                                                     default:"0.5"`
	Gap               string  `help:"Width between chunks to skip. Values greater than 1 are treated as number of seconds"                                                               default:"5%"`
	DeleteCorrupt     string  `help:"Delete media that is more corrupt or equal to this threshold. Values greater than 1 are treated as number of seconds"`
	FullScanIfCorrupt string  `help:"Full scan as second pass if initial scan result more corruption or equal to this threshold. Values greater than 1 are treated as number of seconds"`
	FullScan          bool    `help:"Decode the full media file"`
	AudioScan         bool    `help:"Count errors in audio track only"`
}

func (c *MediaCheckCmd) Run(ctx context.Context) error {
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		DeletedFlags:     c.DeletedFlags,
	}

	gap, _ := utils.FloatFromPercent(c.Gap)
	deleteThreshold, _ := utils.FloatFromPercent(c.DeleteCorrupt)
	fullScanThreshold, _ := utils.FloatFromPercent(c.FullScanIfCorrupt)

	return RunQuery(ctx, c.Databases, flags, func(media []models.MediaWithDB) error {
		if len(media) == 0 {
			return errors.New("no media found")
		}

		for _, m := range media {
			c.checkSingleMedia(ctx, m, gap, deleteThreshold, fullScanThreshold, flags.Simulate)
		}
		return nil
	})
}

func (c *MediaCheckCmd) checkSingleMedia(
	ctx context.Context,
	m models.MediaWithDB,
	gap, deleteThreshold, fullScanThreshold float64,
	simulate bool,
) {
	var corruption float64
	duration := 0.0
	if m.Duration != nil {
		duration = float64(*m.Duration)
	}

	if c.FullScan {
		var err error
		corruption, err = metadata.DecodeFullScan(ctx, m.Path)
		if err != nil {
			models.Log.Error("Full scan failed", "path", m.Path, "error", err)
			corruption = 0.5
		}
	} else {
		if duration == 0 {
			corruption = 0.5
		} else {
			scans := utils.CalculateSegments(duration, c.ChunkSize, gap)
			corruption = metadata.DecodeQuickScan(ctx, m.Path, scans, c.ChunkSize)

			if fullScanThreshold > 0 && corruption >= fullScanThreshold {
				models.Log.Info(
					"Corruption threshold reached, performing full scan",
					"path",
					m.Path,
					"corruption",
					corruption,
				)
				var err error
				corruption, err = metadata.DecodeFullScan(ctx, m.Path)
				if err != nil {
					models.Log.Error("Full scan failed", "path", m.Path, "error", err)
				}
			}
		}
	}

	fmt.Printf("%.2f%%\t%s\n", corruption*100, m.Path)

	if deleteThreshold > 0 && corruption >= deleteThreshold {
		c.handleCorruptFile(ctx, m, corruption, simulate)
	}
}

func (c *MediaCheckCmd) handleCorruptFile(
	ctx context.Context,
	m models.MediaWithDB,
	corruption float64,
	simulate bool,
) {
	models.Log.Warn("Deleting corrupt file", "path", m.Path, "corruption", corruption)
	if simulate {
		return
	}

	if err := os.Remove(m.Path); err != nil {
		models.Log.Error("Failed to delete corrupt file", "path", m.Path, "error", err)
		return
	}

	// Mark as deleted in DB
	sqlDB, err := db.Connect(ctx, m.DB)
	if err != nil {
		return
	}
	defer sqlDB.Close()

	queries := db.New(sqlDB)
	if err := queries.MarkDeleted(ctx, db.MarkDeletedParams{
		Path:        m.Path,
		TimeDeleted: utils.ToNullInt64(time.Now().Unix()),
	}); err != nil {
		models.Log.Warn("Failed to mark deleted in DB", "path", m.Path, "error", err)
	}
}
