package commands

import (
	"context"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestSampleHashCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")

	cmd := &SampleHashCmd{
		Paths: []string{f1},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("SampleHashCmd failed: %v", err)
	}
}
