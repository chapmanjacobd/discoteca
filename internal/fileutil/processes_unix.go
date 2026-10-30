//go:build !windows

package fileutil

import (
	"context"
	"os/exec"
	"syscall"
)

// CmdDetach runs a command in the background, detached from the current process
func CmdDetach(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd.Start()
}
