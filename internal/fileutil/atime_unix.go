//go:build !linux && !darwin && !windows

package fileutil

import (
	"os"
	"time"
)

// GetAccessTime returns the access time of a file
func GetAccessTime(info os.FileInfo) time.Time {
	return info.ModTime()
}
