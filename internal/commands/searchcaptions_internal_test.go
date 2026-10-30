package commands

import (
	"slices"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestSearchCaptionsBuildPlayerArgs(t *testing.T) {
	cmd := &SearchCaptionsCmd{
		PlaybackFlags: models.PlaybackFlags{
			OverridePlayer: "vlc",
		},
	}
	caption := MergedCaption{Path: "/movie.mp4", Time: 10, End: 20}

	args := cmd.buildPlayerArgs(caption)
	if args[0] != "vlc" {
		t.Errorf("--override-player ignored: %v", args)
	}
	if !slices.Contains(args, "--start=8.000000") {
		t.Errorf("caption start was not passed: %v", args)
	}

	cmd.PlaybackFlags = models.PlaybackFlags{
		Volume:       55,
		Mute:         true,
		Fullscreen:   true,
		NoSubtitles:  true,
		Start:        "5",
		End:          "9",
		Speed:        1.5,
		Loop:         true,
		SavePlayhead: true,
	}
	args = cmd.buildPlayerArgs(caption)
	for _, want := range []string{
		"--volume=55", "--mute=yes", "--fullscreen", "--no-sub", "--speed=1.50",
		"--loop-file=inf", "--start=5", "--end=9",
	} {
		if !slices.Contains(args, want) {
			t.Errorf("missing %s in %v", want, args)
		}
	}
}

func TestApplyOffset(t *testing.T) {
	captions := []MergedCaption{{Path: "a"}, {Path: "b"}, {Path: "c"}}
	if got := applyOffset(captions, 1); len(got) != 2 || got[0].Path != "b" {
		t.Errorf("offset 1 returned %v", got)
	}
	if got := applyOffset(captions, 10); got != nil {
		t.Errorf("offset past the end should return nil, got %v", got)
	}
	if got := applyOffset(captions, 0); len(got) != 3 {
		t.Errorf("offset 0 changed the slice: %v", got)
	}
}
