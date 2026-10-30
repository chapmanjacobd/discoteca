package media

import (
	"context"
	"os/exec"
)

// FFProbe creates a new ffprobe command with common options for full metadata extraction.
// Additional options can be appended to the returned command's Args.
func FFProbe(ctx context.Context, path string, extraArgs ...string) *exec.Cmd {
	// Base arguments with common options
	args := make([]string, 0, 10+len(extraArgs)+1)
	args = append(args,
		"-v", "error",
		"-hide_banner",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		"-of", "json",
		"-rw_timeout", "100000000", // 1m40s - timeout for network/remote files
	)

	// Add extra arguments before the file path
	args = append(args, extraArgs...)

	// Add the file path at the end
	args = append(args, path)

	return exec.CommandContext(ctx, "ffprobe", args...)
}

// FFProbeCountFrames creates a new ffprobe command for frame counting
// operations which can take a long time. Uses a much higher rw_timeout.
func FFProbeCountFrames(ctx context.Context, path string, extraArgs ...string) *exec.Cmd {
	args := make([]string, 0, 12+len(extraArgs)+1)
	args = append(args,
		"-v", "error",
		"-hide_banner",
		"-show_entries", "stream=r_frame_rate,nb_read_frames,duration:format=duration",
		"-select_streams", "v",
		"-count_frames",
		"-of", "json",
		"-rw_timeout", "600000000", // 10m - higher timeout for count_frames operations
	)

	// Add extra arguments before the file path
	args = append(args, extraArgs...)

	// Add the file path at the end
	args = append(args, path)

	return exec.CommandContext(ctx, "ffprobe", args...)
}
