//go:build !windows

package utils

import (
	"os/exec"
	"syscall"
)

// CmdDetach runs a command in the background, detached from the current process
func CmdDetach(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd.Start()
}
