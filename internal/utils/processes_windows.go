//go:build windows

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
		// CREATE_NEW_PROCESS_GROUP = 0x00000200
		CreationFlags: 0x00000200,
	}

	return cmd.Start()
}
