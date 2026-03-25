//go:build astiav

package metadata

import (
	"context"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/asticode/go-astiav"
)

// astiavBackend implements ProbeBackend using libavformat via CGO
type astiavBackend struct {
	formatCtx *astiav.FormatContext
}

func newAstiavBackend() (*astiavBackend, error) {
	b := &astiavBackend{}
	return b, nil
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

	// Duration in seconds
	result.Duration = float64(formatCtx.Duration()) / float64(astiav.TimeBaseQ.Denom)

	// Format tags
	if formatCtx.Metadata() != nil {
		entry := formatCtx.Metadata().Get("", "", astiav.MetadataMatchCaseInsensitive)
		for entry != nil {
			result.Tags[entry.Key()] = entry.Value()
			entry = entry.Next()
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
			CodecType:   codec.CodecType().String(),
			CodecName:   codec.CodecID().String(),
			Profile:     codec.Profile(),
			Disposition: make(map[string]int),
		}

		// Video-specific fields
		if codec.CodecType() == astiav.CodecTypeVideo {
			streamInfo.Width = int(codec.Width())
			streamInfo.Height = int(codec.Height())
			streamInfo.PixFmt = codec.Format().String()

			// Frame rate
			if stream.AvgFrameRate().Denom != 0 {
				fps := float64(stream.AvgFrameRate().Num()) / float64(stream.AvgFrameRate().Denom)
				streamInfo.AvgFrameRate = strconv.FormatFloat(fps, 'f', 3, 64)
			}
		}

		// Audio-specific fields
		if codec.CodecType() == astiav.CodecTypeAudio {
			streamInfo.SampleRate = strconv.Itoa(int(codec.SampleRate()))
			streamInfo.Channels = int(codec.Channels())
		}

		// Stream tags
		if stream.Metadata() != nil {
			streamInfo.Tags = make(map[string]string)
			entry := stream.Metadata().Get("", "", astiav.MetadataMatchCaseInsensitive)
			for entry != nil {
				streamInfo.Tags[entry.Key()] = entry.Value()
				entry = entry.Next()
			}
		}

		// Disposition
		disposition := codec.Disposition()
		if disposition&astiav.DispositionDefault != 0 {
			streamInfo.Disposition["default"] = 1
		}
		if disposition&astiav.DispositionDub != 0 {
			streamInfo.Disposition["dub"] = 1
		}
		if disposition&astiav.DispositionOriginal != 0 {
			streamInfo.Disposition["original"] = 1
		}
		if disposition&astiav.DispositionComment != 0 {
			streamInfo.Disposition["comment"] = 1
		}
		if disposition&astiav.DispositionLyrics != 0 {
			streamInfo.Disposition["lyrics"] = 1
		}
		if disposition&astiav.DispositionKaraoke != 0 {
			streamInfo.Disposition["karaoke"] = 1
		}
		if disposition&astiav.DispositionForced != 0 {
			streamInfo.Disposition["forced"] = 1
		}
		if disposition&astiav.DispositionHearingImpaired != 0 {
			streamInfo.Disposition["hearing_impaired"] = 1
		}
		if disposition&astiav.DispositionVisualImpaired != 0 {
			streamInfo.Disposition["visual_impaired"] = 1
		}
		if disposition&astiav.DispositionCleanEffects != 0 {
			streamInfo.Disposition["clean_effects"] = 1
		}
		if disposition&astiav.DispositionAttachedPic != 0 {
			streamInfo.Disposition["attached_pic"] = 1
		}

		// Skip attached pictures
		if streamInfo.Disposition["attached_pic"] == 1 {
			continue
		}

		result.Streams = append(result.Streams, streamInfo)
	}

	// Chapters
	for i := 0; i < formatCtx.NbChapters(); i++ {
		chapter := formatCtx.Chapters()[i]
		if chapter == nil || chapter.Metadata() == nil {
			continue
		}

		titleEntry := chapter.Metadata().Get("title", "", astiav.MetadataMatchCaseInsensitive)
		if titleEntry == nil {
			continue
		}

		startTime := float64(chapter.Start()) * float64(chapter.TimeBase().Num()) / float64(chapter.TimeBase().Denom())
		endTime := float64(chapter.End()) * float64(chapter.TimeBase().Num()) / float64(chapter.TimeBase().Denom())

		result.Chapters = append(result.Chapters, ChapterInfo{
			ID:        int(chapter.ID()),
			StartTime: startTime,
			EndTime:   endTime,
			Title:     titleEntry.Value(),
		})
	}

	return result, nil
}

// Helper to convert C string to Go string safely
func cStringToString(cstr *uint8) string {
	if cstr == nil {
		return ""
	}
	return string(unsafe.Slice(cstr, findNullTerminator(cstr)))
}

func findNullTerminator(s *uint8) int {
	p := unsafe.Pointer(s)
	for i := 0; ; i++ {
		if *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(i))) == 0 {
			return i
		}
	}
}
