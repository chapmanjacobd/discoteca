package utils

import (
	"runtime/debug"
	"sync"
)

var (
	// Version is the git version that produced this binary.
	Version string

	// When is the datestamp that produced this binary.
	When string

	// BuildSettings contains additional build information.
	BuildSettings []debug.BuildSetting

	// Deps contains dependency information.
	Deps []*debug.Module

	versionInitOnce sync.Once
)

// InitVersionInfo initializes version information from build info.
// This should be called early in the application lifecycle.
func InitVersionInfo() {
	versionInitOnce.Do(func() {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}

		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" {
				Version = s.Value
			}
			if s.Key == "vcs.time" {
				When = s.Value
			}
			BuildSettings = append(BuildSettings, s)
		}

		Deps = bi.Deps
	})
}
