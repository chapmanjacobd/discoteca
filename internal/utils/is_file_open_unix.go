//go:build !windows

package utils

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func isFileOpenDarwin(absPath string) bool {
	// On macOS, use lsof -t to check if any process has the file open
	cmd := exec.CommandContext(context.Background(), "lsof", "-t", absPath)
	return cmd.Run() == nil
}

func isFileOpenLinux(absPath string) bool {
	files, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		// Check if name is a number (PID)
		isPid := true
		for _, r := range f.Name() {
			if r < '0' || r > '9' {
				isPid = false
				break
			}
		}
		if !isPid {
			continue
		}

		fdDir := filepath.Join("/proc", f.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err == nil && link == absPath {
				return true
			}
		}
	}
	return false
}

// IsFileOpen checks if a file is currently open by any process
func IsFileOpen(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	if runtime.GOOS == "darwin" {
		return isFileOpenDarwin(absPath)
	}

	if runtime.GOOS == "linux" {
		return isFileOpenLinux(absPath)
	}

	return false
}
