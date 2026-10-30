package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/textutil"
)

func DecodeQuickScan(ctx context.Context, path string, scans []float64, scanDuration float64) (float64, error) {
	if len(scans) == 0 {
		return 0, nil
	}

	var wg sync.WaitGroup
	failChan := make(chan struct{}, len(scans))
	semaphore := make(chan struct{}, 4) // max 4 parallel ffmpeg instances
	var fatalMu sync.Mutex
	var fatalErr error

	setFatal := func(err error) {
		fatalMu.Lock()
		defer fatalMu.Unlock()
		if fatalErr == nil {
			fatalErr = err
		}
	}

	for _, scan := range scans {
		wg.Add(1)
		go func(s float64) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			args := []string{
				"-nostdin",
				"-hide_banner",
				"-nostats",
				"-xerror",
				"-v", "16",
				"-err_detect", "buffer+crccheck+explode",
				"-ss", fmt.Sprintf("%.2f", s),
				"-i", path,
				"-t", fmt.Sprintf("%.2f", scanDuration),
				"-f", "null",
				os.DevNull,
			}

			cmd := exec.CommandContext(ctx, "ffmpeg", args...)
			if err := cmd.Run(); err != nil {
				// A missing tool or a cancelled context is not measured
				// corruption; only actual decode failures count.
				if errors.Is(err, exec.ErrNotFound) || ctx.Err() != nil {
					setFatal(err)
					return
				}
				failChan <- struct{}{}
			}
		}(scan)
	}

	wg.Wait()
	close(failChan)

	fatalMu.Lock()
	err := fatalErr
	fatalMu.Unlock()
	if err != nil {
		return 0, fmt.Errorf("quick scan failed: %w", err)
	}

	failCount := 0
	for range failChan {
		failCount++
	}

	return float64(failCount) / float64(len(scans)), nil
}

type ffprobeCorruptionOutput struct {
	Streams []struct {
		RFrameRate    string `json:"r_frame_rate"`
		NbReadFrames  string `json:"nb_read_frames"`
		NbReadPackets string `json:"nb_read_packets"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func DecodeFullScan(ctx context.Context, path string) (float64, error) {
	cmd := media.FFProbeCountFrames(ctx, path)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var data ffprobeCorruptionOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return 0, err
	}

	if len(data.Streams) == 0 {
		return 0, errors.New("no video streams found")
	}

	stream := data.Streams[0]
	fpsParts := textutil.SplitAndTrim(stream.RFrameRate, "/")
	if len(fpsParts) != 2 {
		return 0, fmt.Errorf("invalid frame rate: %s", stream.RFrameRate)
	}

	num, _ := strconv.ParseFloat(fpsParts[0], 64)
	den, _ := strconv.ParseFloat(fpsParts[1], 64)
	if num == 0 || den == 0 {
		return 0, fmt.Errorf("invalid frame rate values: %s", stream.RFrameRate)
	}

	nbFrames, _ := strconv.ParseFloat(stream.NbReadFrames, 64)
	metadataDuration, _ := strconv.ParseFloat(data.Format.Duration, 64)

	if metadataDuration == 0 {
		return 0, errors.New("ffprobe reported zero duration")
	}

	actualDuration := nbFrames * den / num
	difference := math.Abs(actualDuration - metadataDuration)
	corruption := difference / metadataDuration

	return corruption, nil
}
