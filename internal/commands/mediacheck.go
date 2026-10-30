package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
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
	DeleteCorrupt     string  `help:"Delete media whose corruption ratio is >= this value (e.g. 0.5 or 50%)"`
	FullScanIfCorrupt string  `help:"Full scan as a second pass when the quick-scan corruption ratio is >= this value (e.g. 0.2 or 20%)"`
	FullScan          bool    `help:"Decode the full media file"`
}

func (c *MediaCheckCmd) Run(ctx context.Context) error {
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		DeletedFlags:     c.DeletedFlags,
	}

	gap := 0.05
	var err error
	if c.Gap != "" {
		gap, err = utils.FloatFromPercent(c.Gap)
		if err != nil {
			return fmt.Errorf("invalid --gap %q: %w", c.Gap, err)
		}
	}
	if c.ChunkSize <= 0 {
		c.ChunkSize = 0.5
	}
	deleteThreshold := 0.0
	if c.DeleteCorrupt != "" {
		deleteThreshold, err = utils.FloatFromPercent(c.DeleteCorrupt)
		if err != nil {
			return fmt.Errorf("invalid --delete-corrupt %q: %w", c.DeleteCorrupt, err)
		}
	}
	fullScanThreshold := 0.0
	if c.FullScanIfCorrupt != "" {
		fullScanThreshold, err = utils.FloatFromPercent(c.FullScanIfCorrupt)
		if err != nil {
			return fmt.Errorf("invalid --full-scan-if-corrupt %q: %w", c.FullScanIfCorrupt, err)
		}
	}

	checked := 0
	var skipped []string

	err = RunQuery(ctx, c.Databases, flags, func(media []models.MediaWithDB) error {
		if len(media) == 0 {
			return errors.New("no media found")
		}

		for _, m := range media {
			corruption, ok := c.checkSingleMedia(ctx, checkMediaOptions{
				m:                 m,
				gap:               gap,
				fullScanThreshold: fullScanThreshold,
			})
			if !ok {
				skipped = append(skipped, m.Path)
				continue
			}
			checked++

			if deleteThreshold > 0 && corruption >= deleteThreshold {
				c.handleCorruptFile(ctx, m, corruption, flags.Simulate)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("\nMediaCheck complete: checked %d file(s), skipped %d file(s)\n", checked, len(skipped))
	for _, path := range skipped {
		fmt.Printf("  skipped: %s\n", path)
	}
	return nil
}

type checkMediaOptions struct {
	m                 models.MediaWithDB
	gap               float64
	fullScanThreshold float64
}

// checkSingleMedia returns the measured corruption ratio and whether the scan
// actually produced a measurement. A scan failure is not corruption.
func (c *MediaCheckCmd) checkSingleMedia(ctx context.Context, opts checkMediaOptions) (float64, bool) {
	var corruption float64
	duration := 0.0
	m := opts.m
	if m.Duration != nil {
		duration = float64(*m.Duration)
	}

	if c.FullScan {
		var err error
		corruption, err = metadata.DecodeFullScan(ctx, m.Path)
		if err != nil {
			logging.Log.Error("Full scan failed; skipping file", "path", m.Path, "error", err)
			return 0, false
		}
	} else {
		if duration == 0 {
			logging.Log.Warn("Skipping media with unknown duration", "path", m.Path)
			return 0, false
		}
		scans := utils.CalculateSegments(duration, c.ChunkSize, opts.gap)
		var err error
		corruption, err = metadata.DecodeQuickScan(ctx, m.Path, scans, c.ChunkSize)
		if err != nil {
			logging.Log.Error("Quick scan failed; skipping file", "path", m.Path, "error", err)
			return 0, false
		}

		if opts.fullScanThreshold > 0 && corruption >= opts.fullScanThreshold {
			logging.Log.Info(
				"Corruption threshold reached, performing full scan",
				"path",
				m.Path,
				"corruption",
				corruption,
			)
			corruption, err = metadata.DecodeFullScan(ctx, m.Path)
			if err != nil {
				logging.Log.Error("Full scan failed; skipping file", "path", m.Path, "error", err)
				return 0, false
			}
		}
	}

	fmt.Printf("%.2f%%\t%s\n", corruption*100, m.Path)
	return corruption, true
}

func (c *MediaCheckCmd) handleCorruptFile(
	ctx context.Context,
	m models.MediaWithDB,
	corruption float64,
	simulate bool,
) {
	if simulate {
		fmt.Printf("[Dry-run] would delete corrupt file: %s (corruption %.2f%%)\n", m.Path, corruption*100)
		return
	}

	logging.Log.Warn("Deleting corrupt file", "path", m.Path, "corruption", corruption)

	if err := os.Remove(m.Path); err != nil {
		logging.Log.Error("Failed to delete corrupt file", "path", m.Path, "error", err)
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
		logging.Log.Warn("Failed to mark deleted in DB", "path", m.Path, "error", err)
	}
}
