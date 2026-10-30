// Package config centralizes discoteca's platform-specific configuration paths.
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

var (
	IsWindows = runtime.GOOS == "windows"
	IsMac     = runtime.GOOS == "darwin"
)

func GetMpvWatchSocket() string {
	home, _ := os.UserHomeDir()
	if IsWindows {
		return filepath.Join(home, "AppData", "Roaming", "mpv", "socket")
	}
	if IsMac {
		return filepath.Join(home, "Library", "Application Support", "mpv", "socket")
	}
	return filepath.Join(home, ".config", "mpv", "socket")
}

func GetMpvWatchLaterDir() string {
	home, _ := os.UserHomeDir()
	if IsWindows {
		return filepath.Join(home, "AppData", "Roaming", "mpv", "watch_later")
	}
	if IsMac {
		return filepath.Join(home, "Library", "Application Support", "mpv", "watch_later")
	}
	return filepath.Join(home, ".config", "mpv", "watch_later")
}

func GetCattNowPlayingFile() string {
	return filepath.Join(os.TempDir(), "catt_playing")
}

func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	if IsWindows {
		return filepath.Join(home, "AppData", "Roaming", "disco")
	}
	return filepath.Join(home, ".config", "disco")
}
