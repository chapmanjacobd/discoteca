package commands

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type ShrinkCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.HashingFlags     `embed:""`

	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`

	Valid   bool `default:"true" help:"Attempt to process files with valid metadata"`
	Invalid bool `help:"Attempt to process files with invalid metadata"`

	MinSavingsVideo   string `default:"5%" help:"Minimum savings for video (percentage or bytes)"`
	MinSavingsAudio   string `default:"10%" help:"Minimum savings for audio (percentage or bytes)"`
	MinSavingsImage   string `default:"15%" help:"Minimum savings for images (percentage or bytes)"`
	SourceAudioBitrate  string `default:"256kbps" help:"Used to estimate duration when files are inside of archives or invalid"`
	SourceVideoBitrate  string `default:"1500kbps" help:"Used to estimate duration when files are inside of archives or invalid"`
	TargetAudioBitrate  string `default:"128kbps" help:"Target audio bitrate"`
	TargetVideoBitrate  string `default:"800kbps" help:"Target video bitrate"`
	TargetImageSize     string `default:"30KiB" help:"Target image size"`
	TranscodingVideoRate float64 `default:"1.8" help:"Ratio of duration eg. 4x realtime speed"`
	TranscodingAudioRate float64 `default:"150" help:"Ratio of duration eg. 100x realtime speed"`
	TranscodingImageTime float64 `default:"1.5" help:"Seconds to process an image"`

	MaxVideoHeight int `default:"960" help:"Maximum video height"`
	MaxVideoWidth  int `default:"1440" help:"Maximum video width"`
	MaxImageHeight int `default:"2400" help:"Maximum image height"`
	MaxImageWidth  int `default:"2400" help:"Maximum image width"`
	Preset         string `default:"7" help:"SVT-AV1 preset (0-13, lower is slower/better)"`
	CRF            string `default:"40" help:"CRF value for SVT-AV1 (0-63, lower is better)"`

	ContinueFrom   string `help:"Skip media until specific file path is seen"`
	Move           string `help:"Directory to move successful files"`
	MoveBroken     string `help:"Directory to move unsuccessful files"`
	DeleteUnplayable bool `help:"Delete unplayable files"`

	OnlyHash    bool `help:"Only calculate hashes, don't shrink"`
	OnlyDedupe  bool `help:"Only mark deduplicated files, don't shrink"`
	ForceRehash bool `help:"Force recalculation of hashes"`
	ForceReshrink bool `help:"Force reprocessing of already shrinked files"`
}

type ShrinkMedia struct {
	Path          string
	Size          int64
	Duration      float64
	VideoCount    int
	AudioCount    int
	VideoCodecs   string
	AudioCodecs   string
	Type          string
	Ext           string
	MediaType     string
	FutureSize    int64
	Savings       int64
	ProcessingTime int
	CompressedSize int64
	IsArchived    bool
	ArchivePath   string
	FastHash      string
	Sha256        string
}

func (c *ShrinkCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)

	// Parse size/duration strings
	minSavingsVideo := utils.ParsePercentOrBytes(c.MinSavingsVideo)
	minSavingsAudio := utils.ParsePercentOrBytes(c.MinSavingsAudio)
	minSavingsImage := utils.ParsePercentOrBytes(c.MinSavingsImage)
	sourceAudioBitrate := utils.ParseBitrate(c.SourceAudioBitrate)
	sourceVideoBitrate := utils.ParseBitrate(c.SourceVideoBitrate)
	targetAudioBitrate := utils.ParseBitrate(c.TargetAudioBitrate)
	targetVideoBitrate := utils.ParseBitrate(c.TargetVideoBitrate)
	targetImageSize := utils.ParseSize(c.TargetImageSize)

	// Check for required tools
	ffmpegInstalled := utils.CommandExists("ffmpeg")
	magickInstalled := utils.CommandExists("magick")
	if !ffmpegInstalled {
		slog.Warn("ffmpeg not installed. Video and Audio files will be skipped")
	}
	if !magickInstalled {
		slog.Warn("ImageMagick not installed. Image files will be skipped")
	}

	for _, dbPath := range c.Databases {
		if err := c.processDatabase(dbPath, ffmpegInstalled, magickInstalled,
			minSavingsVideo, minSavingsAudio, minSavingsImage,
			sourceAudioBitrate, sourceVideoBitrate,
			targetAudioBitrate, targetVideoBitrate, targetImageSize); err != nil {
			return err
		}
	}

	return nil
}

func (c *ShrinkCmd) processDatabase(dbPath string, ffmpegInstalled, magickInstalled bool,
	minSavingsVideo, minSavingsAudio, minSavingsImage float64,
	sourceAudioBitrate, sourceVideoBitrate, targetAudioBitrate, targetVideoBitrate, targetImageSize int64) error {

	sqlDB, _, err := db.ConnectWithInit(dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Build query to get media
	query := `
		SELECT path, size, duration, video_count, audio_count, 
		       video_codecs, audio_codecs, type, fasthash, sha256, is_shrinked
		FROM media
		WHERE COALESCE(time_deleted, 0) = 0
		  AND size > 0
	`
	
	if !c.ForceReshrink {
		query += " AND (COALESCE(is_shrinked, 0) = 0 OR is_shrinked IS NULL)"
	}

	rows, err := sqlDB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var media []ShrinkMedia
	for rows.Next() {
		var m ShrinkMedia
		var fastHash, sha256 sql.NullString
		var isShrinked sql.NullInt64
		err := rows.Scan(&m.Path, &m.Size, &m.Duration, &m.VideoCount, &m.AudioCount,
			&m.VideoCodecs, &m.AudioCodecs, &m.Type, &fastHash, &sha256, &isShrinked)
		if err != nil {
			slog.Error("Scan error", "error", err)
			continue
		}
		if fastHash.Valid {
			m.FastHash = fastHash.String
		}
		if sha256.Valid {
			m.Sha256 = sha256.String
		}
		m.Ext = strings.ToLower(filepath.Ext(m.Path))
		media = append(media, m)
	}

	if err = rows.Err(); err != nil {
		return err
	}

	slog.Info("Found media to process", "count", len(media))

	// Process media and calculate hashes
	var toShrink []ShrinkMedia
	var hashWg sync.WaitGroup
	hashSem := make(chan struct{}, c.HashThreads)

	for i := range media {
		m := &media[i]
		
		// Calculate hashes if needed
		if c.ForceRehash || m.FastHash == "" {
			hashWg.Add(1)
			go func(mediaItem *ShrinkMedia) {
				defer hashWg.Done()
				hashSem <- struct{}{}
				defer func() { <-hashSem }()

				// Calculate sample hash
				h, err := utils.SampleHashFile(mediaItem.Path, c.HashThreads, c.HashGap, c.HashChunkSize)
				if err == nil && h != "" {
					mediaItem.FastHash = h
					updateHash(sqlDB, mediaItem.Path, h, "", false)
				}

				// Calculate full hash
				h, err = utils.FullHashFile(mediaItem.Path)
				if err == nil && h != "" {
					mediaItem.Sha256 = h
					updateHash(sqlDB, mediaItem.Path, "", h, false)
				}
			}(m)
		}

		// Check if file should be shrinked
		if shouldShrink := c.checkShrink(m, ffmpegInstalled, magickInstalled,
			minSavingsVideo, minSavingsAudio, minSavingsImage,
			sourceAudioBitrate, sourceVideoBitrate,
			targetAudioBitrate, targetVideoBitrate, targetImageSize); shouldShrink {
			toShrink = append(toShrink, *m)
		}
	}

	hashWg.Wait()

	if c.OnlyHash {
		slog.Info("Hash calculation complete")
		return nil
	}

	if len(toShrink) == 0 {
		slog.Info("No files to shrink")
		return nil
	}

	// Sort by savings/processing_time ratio
	sort.Slice(toShrink, func(i, j int) bool {
		ratioI := float64(toShrink[i].Savings) / float64(max(toShrink[i].ProcessingTime, 1))
		ratioJ := float64(toShrink[j].Savings) / float64(max(toShrink[j].ProcessingTime, 1))
		return ratioI > ratioJ
	})

	// Print summary
	var totalCurrentSize, totalFutureSize, totalSavings int64
	for _, m := range toShrink {
		totalCurrentSize += m.Size
		totalFutureSize += m.FutureSize
		totalSavings += m.Savings
		slog.Info("To shrink", 
			"path", m.Path,
			"type", m.MediaType,
			"current", utils.FormatSize(m.Size),
			"future", utils.FormatSize(m.FutureSize),
			"savings", utils.FormatSize(m.Savings))
	}

	slog.Info("Summary",
		"current_size", utils.FormatSize(totalCurrentSize),
		"future_size", utils.FormatSize(totalFutureSize),
		"savings", utils.FormatSize(totalSavings),
		"count", len(toShrink))

	if c.Simulate {
		slog.Info("Simulation mode - no files will be processed")
		return nil
	}

	if !c.NoConfirm {
		fmt.Print("Proceed with shrinking? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			return nil
		}
	}

	// Process files
	var successCount, failCount int
	for _, m := range toShrink {
		slog.Info("Processing", "path", m.Path, "type", m.MediaType)
		
		var err error
		switch m.MediaType {
		case "Audio":
			err = c.shrinkAudio(m.Path, m.Duration, targetAudioBitrate)
		case "Video":
			err = c.shrinkVideo(m.Path, m.Duration, targetVideoBitrate)
		case "Image":
			err = c.shrinkImage(m.Path)
		}

		if err != nil {
			slog.Error("Failed to process", "path", m.Path, "error", err)
			failCount++
			if c.MoveBroken != "" {
				dest := filepath.Join(c.MoveBroken, filepath.Base(m.Path))
				os.Rename(m.Path, dest)
			}
		} else {
			successCount++
			updateShrinkStatus(sqlDB, m.Path, true)
			if c.Move != "" {
				dest := filepath.Join(c.Move, filepath.Base(m.Path))
				os.Rename(m.Path, dest)
			}
		}
	}

	slog.Info("Complete", "success", successCount, "failed", failCount)
	return nil
}

func (c *ShrinkCmd) checkShrink(m *ShrinkMedia, ffmpegInstalled, magickInstalled bool,
	minSavingsVideo, minSavingsAudio, minSavingsImage float64,
	sourceAudioBitrate, sourceVideoBitrate, targetAudioBitrate, targetVideoBitrate, targetImageSize int64) bool {

	if m.Size == 0 {
		return false
	}

	filetype := strings.ToLower(m.Type)
	ext := m.Ext

	// Audio files
	if (strings.HasPrefix(filetype, "audio/") || strings.Contains(filetype, " audio")) ||
		(utils.AudioExtensionMap[ext] && m.VideoCount == 0) {
		
		if !ffmpegInstalled {
			return false
		}

		// Check if already opus
		if strings.ToLower(m.AudioCodecs) == "opus" {
			slog.Debug("Already opus", "path", m.Path)
			return false
		}

		duration := m.Duration
		if duration <= 0 {
			duration = float64(m.Size) / float64(sourceAudioBitrate) * 8
		}

		futureSize := int64(duration * float64(targetAudioBitrate) / 8)
		shouldShrinkBuffer := int64(float64(futureSize) * minSavingsAudio)
		
		m.MediaType = "Audio"
		m.FutureSize = futureSize
		m.Savings = m.Size - futureSize
		m.ProcessingTime = int(math.Ceil(duration / c.TranscodingAudioRate))

		canShrink := m.Size > (futureSize + shouldShrinkBuffer)
		return canShrink
	}

	// Video files
	if (strings.HasPrefix(filetype, "video/") || strings.Contains(filetype, " video")) ||
		(utils.VideoExtensionMap[ext] && m.VideoCount >= 1) {
		
		if !ffmpegInstalled {
			return false
		}

		// Check if already AV1
		if strings.ToLower(m.VideoCodecs) == "av1" {
			slog.Debug("Already AV1", "path", m.Path)
			return false
		}

		duration := m.Duration
		if duration <= 0 {
			duration = float64(m.Size) / float64(sourceVideoBitrate) * 8
		}

		futureSize := int64(duration * float64(targetVideoBitrate) / 8)
		shouldShrinkBuffer := int64(float64(futureSize) * minSavingsVideo)
		
		m.MediaType = "Video"
		m.FutureSize = futureSize
		m.Savings = m.Size - futureSize
		m.ProcessingTime = int(math.Ceil(duration / c.TranscodingVideoRate))

		canShrink := m.Size > (futureSize + shouldShrinkBuffer)
		return canShrink
	}

	// Image files
	if (strings.HasPrefix(filetype, "image/") || strings.Contains(filetype, " image")) ||
		(utils.ImageExtensionMap[ext] && m.Duration == 0) {
		
		if !magickInstalled {
			return false
		}

		// Skip existing AVIF
		if ext == ".avif" {
			slog.Debug("Already AVIF", "path", m.Path)
			return false
		}

		futureSize := targetImageSize
		shouldShrinkBuffer := int64(float64(futureSize) * minSavingsImage)
		
		m.MediaType = "Image"
		m.FutureSize = futureSize
		m.Savings = m.Size - futureSize
		m.ProcessingTime = int(c.TranscodingImageTime)

		canShrink := m.Size > (futureSize + shouldShrinkBuffer)
		return canShrink
	}

	return false
}

func (c *ShrinkCmd) shrinkAudio(path string, duration float64, targetBitrate int64) error {
	outputPath := path + ".tmp.mka"
	args := []string{
		"-y",
		"-i", path,
		"-c:a", "libopus",
		"-b:a", fmt.Sprintf("%dk", targetBitrate/1000),
		"-ar", "48000",
		"-ac", "2",
		"-af", "loudnorm=i=-18:tp=-3:lra=17",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("FFmpeg error", "output", string(output))
		return err
	}

	// Replace original file
	os.Rename(path, path+".bak")
	os.Rename(outputPath, path)
	os.Remove(path + ".bak")

	return nil
}

func (c *ShrinkCmd) shrinkVideo(path string, duration float64, targetBitrate int64) error {
	outputPath := path + ".tmp.mkv"
	args := []string{
		"-y",
		"-i", path,
		"-c:v", "libsvtav1",
		"-preset", c.Preset,
		"-crf", c.CRF,
		"-pix_fmt", "yuv420p10le",
		"-svtav1-params", "tune=0:enable-overlays=1",
		"-vf", fmt.Sprintf("scale=%d:-2", c.MaxVideoWidth),
		"-c:a", "libopus",
		"-b:a", "128k",
		"-ar", "48000",
		"-ac", "2",
		"-af", "loudnorm=i=-18:tp=-3:lra=17",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("FFmpeg error", "output", string(output))
		return err
	}

	// Replace original file
	os.Rename(path, path+".bak")
	os.Rename(outputPath, path)
	os.Remove(path + ".bak")

	return nil
}

func (c *ShrinkCmd) shrinkImage(path string) error {
	outputPath := path + ".tmp.avif"
	args := []string{
		"convert", path,
		"-resize", fmt.Sprintf("%dx%d>", c.MaxImageWidth, c.MaxImageHeight),
		"-quality", "80",
		outputPath,
	}

	cmd := exec.Command("magick", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("ImageMagick error", "output", string(output))
		return err
	}

	// Replace original file
	os.Rename(path, path+".bak")
	os.Rename(outputPath, path)
	os.Remove(path + ".bak")

	return nil
}

func updateHash(db *sql.DB, path, fastHash, sha256 string, isDeduped bool) {
	query := "UPDATE media SET "
	updates := []string{}
	args := []interface{}{}
	
	if fastHash != "" {
		updates = append(updates, "fasthash = ?")
		args = append(args, fastHash)
	}
	if sha256 != "" {
		updates = append(updates, "sha256 = ?")
		args = append(args, sha256)
	}
	if isDeduped {
		updates = append(updates, "is_deduped = 1")
	}
	
	if len(updates) == 0 {
		return
	}
	
	query += strings.Join(updates, ", ") + " WHERE path = ?"
	args = append(args, path)
	
	_, err := db.Exec(query, args...)
	if err != nil {
		slog.Error("Failed to update hash", "path", path, "error", err)
	}
}

func updateShrinkStatus(db *sql.DB, path string, isShrinked bool) {
	_, err := db.Exec("UPDATE media SET is_shrinked = 1 WHERE path = ?", path)
	if err != nil {
		slog.Error("Failed to update shrink status", "path", path, "error", err)
	}
}
