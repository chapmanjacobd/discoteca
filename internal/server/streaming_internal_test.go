package server

import (
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/media"
)

func TestBuildFormatArgsForMP4Pipe(t *testing.T) {
	args := (&Server{}).buildFormatArgs(media.TranscodeStrategy{TargetMime: "video/mp4"})
	argString := strings.Join(args, " ")

	if strings.Contains(argString, "global_sidx") {
		t.Fatalf("MP4 pipe output must not use global_sidx: %v", args)
	}
	if !strings.Contains(argString, "frag_keyframe+empty_moov+default_base_moof") {
		t.Fatalf("MP4 pipe output lost fragmented streaming flags: %v", args)
	}
}
