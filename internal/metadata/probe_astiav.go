//go:build astiav

package metadata

import (
	"context"
	"fmt"
	"strconv"

	"github.com/asticode/go-astiav"
)

// astiavBackend implements ProbeBackend using libavformat via CGO
type astiavBackend struct{}

func newAstiavBackend() (*astiavBackend, error) {
	return &astiavBackend{}, nil
}

func (b *astiavBackend) Probe(ctx context.Context, path string) (*ProbeData, error) {
	// Allocate format context
	formatCtx := astiav.AllocFormatContext()
	if formatCtx == nil {
		return nil, fmt.Errorf("failed to allocate format context")
	}
	defer formatCtx.Free()

	// Open input file
	if err := formatCtx.OpenInput(path, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to open input: %w", err)
	}
	defer formatCtx.CloseInput()

	// Read stream information
	if err := formatCtx.FindStreamInfo(nil); err != nil {
		return nil, fmt.Errorf("failed to find stream info: %w", err)
	}

	result := &ProbeData{
		Tags:     make(map[string]string),
		Streams:  make([]StreamInfo, 0),
		Chapters: make([]ChapterInfo, 0),
	}

	// Format name from input format
	if formatCtx.InputFormat() != nil {
		result.FormatName = formatCtx.InputFormat().Name()
	}

	// Duration in seconds (AV_TIME_BASE = 1000000)
	const AV_TIME_BASE = 1000000
	result.Duration = float64(formatCtx.Duration()) / float64(AV_TIME_BASE)

	// Format tags using Dictionary API
	if md := formatCtx.Metadata(); md != nil {
		var entry *astiav.DictionaryEntry
		for {
			entry = md.Get("", entry, astiav.DictionaryFlags(0))
			if entry == nil {
				break
			}
			result.Tags[entry.Key()] = entry.Value()
		}
	}

	// Streams
	for i := 0; i < formatCtx.NbStreams(); i++ {
		stream := formatCtx.Streams()[i]
		if stream == nil || stream.CodecParameters() == nil {
			continue
		}

		codec := stream.CodecParameters()
		streamInfo := StreamInfo{
			CodecType:   codec.MediaType().String(),
			CodecName:   codec.CodecID().String(),
			Profile:     strconv.Itoa(int(codec.Profile())),
			Disposition: make(map[string]int),
		}

		// Video-specific fields
		if codec.MediaType() == astiav.MediaTypeVideo {
			streamInfo.Width = codec.Width()
			streamInfo.Height = codec.Height()
			streamInfo.PixFmt = codec.PixelFormat().String()

			// Frame rate
			avgFps := stream.AvgFrameRate()
			if avgFps.Den() != 0 {
				fps := float64(avgFps.Num()) / float64(avgFps.Den())
				streamInfo.AvgFrameRate = strconv.FormatFloat(fps, 'f', 3, 64)
			}
		}

		// Audio-specific fields
		if codec.MediaType() == astiav.MediaTypeAudio {
			streamInfo.SampleRate = strconv.Itoa(codec.SampleRate())
			streamInfo.Channels = codec.ChannelLayout().Channels()
		}

		// Stream tags
		if md := stream.Metadata(); md != nil {
			streamInfo.Tags = make(map[string]string)
			var entry *astiav.DictionaryEntry
			for {
				entry = md.Get("", entry, astiav.DictionaryFlags(0))
				if entry == nil {
					break
				}
				streamInfo.Tags[entry.Key()] = entry.Value()
			}
		}

		// Disposition flags
		disposition := stream.DispositionFlags()
		if disposition.Has(astiav.DispositionFlagDefault) {
			streamInfo.Disposition["default"] = 1
		}
		if disposition.Has(astiav.DispositionFlagDub) {
			streamInfo.Disposition["dub"] = 1
		}
		if disposition.Has(astiav.DispositionFlagOriginal) {
			streamInfo.Disposition["original"] = 1
		}
		if disposition.Has(astiav.DispositionFlagComment) {
			streamInfo.Disposition["comment"] = 1
		}
		if disposition.Has(astiav.DispositionFlagLyrics) {
			streamInfo.Disposition["lyrics"] = 1
		}
		if disposition.Has(astiav.DispositionFlagKaraoke) {
			streamInfo.Disposition["karaoke"] = 1
		}
		if disposition.Has(astiav.DispositionFlagForced) {
			streamInfo.Disposition["forced"] = 1
		}
		if disposition.Has(astiav.DispositionFlagHearingImpaired) {
			streamInfo.Disposition["hearing_impaired"] = 1
		}
		if disposition.Has(astiav.DispositionFlagVisualImpaired) {
			streamInfo.Disposition["visual_impaired"] = 1
		}
		if disposition.Has(astiav.DispositionFlagCleanEffects) {
			streamInfo.Disposition["clean_effects"] = 1
		}
		if disposition.Has(astiav.DispositionFlagAttachedPic) {
			streamInfo.Disposition["attached_pic"] = 1
		}

		// Skip attached pictures
		if streamInfo.Disposition["attached_pic"] == 1 {
			continue
		}

		result.Streams = append(result.Streams, streamInfo)
	}

	// Note: Chapters API not directly exposed in astiav v0.40
	// Would need to access AVFormatContext.chapters directly via CGO

	return result, nil
}
