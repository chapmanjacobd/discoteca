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

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func DecodeQuickScan(ctx context.Context, path string, scans []float64, scanDuration float64) float64 {
	if len(scans) == 0 {
		return 0
	}

	var wg sync.WaitGroup
	failChan := make(chan struct{}, len(scans))
	semaphore := make(chan struct{}, 4) // max 4 parallel ffmpeg instances

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
				failChan <- struct{}{}
			}
		}(scan)
	}

	wg.Wait()
	close(failChan)

	failCount := 0
	for range failChan {
		failCount++
	}

	return float64(failCount) / float64(len(scans))
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
	cmd := utils.FFProbeCountFrames(ctx, path)

	output, err := cmd.Output()
	if err != nil {
		return 0.5, err
	}

	var data ffprobeCorruptionOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return 0.5, err
	}

	if len(data.Streams) == 0 {
		return 0.5, errors.New("no video streams found")
	}

	stream := data.Streams[0]
	fpsParts := utils.SplitAndTrim(stream.RFrameRate, "/")
	if len(fpsParts) != 2 {
		return 0.5, fmt.Errorf("invalid frame rate: %s", stream.RFrameRate)
	}

	num, _ := strconv.ParseFloat(fpsParts[0], 64)
	den, _ := strconv.ParseFloat(fpsParts[1], 64)
	if num == 0 || den == 0 {
		return 0.5, fmt.Errorf("invalid frame rate values: %s", stream.RFrameRate)
	}

	nbFrames, _ := strconv.ParseFloat(stream.NbReadFrames, 64)
	metadataDuration, _ := strconv.ParseFloat(data.Format.Duration, 64)

	if metadataDuration == 0 {
		return 0.5, nil
	}

	actualDuration := nbFrames * den / num
	difference := math.Abs(actualDuration - metadataDuration)
	corruption := difference / metadataDuration

	return corruption, nil
}
