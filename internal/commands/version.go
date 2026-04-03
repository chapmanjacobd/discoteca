package commands

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type VersionCmd struct {
	Verbose int `help:"Show verbose output including dependencies" short:"v" env:"DISCO_VERBOSE" type:"counter"`
}

func (c *VersionCmd) Run() error {
	// Get build info from runtime/debug
	bi, ok := debug.ReadBuildInfo()

	var version, when string
	if ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				version = s.Value
			case "vcs.time":
				when = s.Value
			case "vcs.modified":
				// Ignore
			}
		}
	}

	// If no VCS info, use fallback
	if version == "" {
		version = utils.Version
	}
	if when == "" {
		when = utils.When
	}

	// Try to get git tag from build info settings
	gitTag := ""
	if ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.git.tag" || s.Key == "git.tag" {
				gitTag = s.Value
				break
			}
		}
	}

	// Format version string
	versionStr := version
	if gitTag != "" {
		versionStr = fmt.Sprintf("%s (tag: %s)", version, gitTag)
	}

	fmt.Printf("discoteca built from %s on %s with %s\n",
		versionStr, when, runtime.Version())

	fmt.Println("Build Settings:")
	if ok {
		for _, s := range bi.Settings {
			fmt.Printf("\t%s=%s\n", s.Key, s.Value)
		}
	}

	if c.Verbose > 0 && ok {
		fmt.Println("\nDeps:")
		for _, dep := range bi.Deps {
			fmt.Printf("\t%s@%s (%s)\n", dep.Path, dep.Version, dep.Sum)
			if dep.Replace != nil {
				r := dep.Replace
				fmt.Printf("   replaced by %s@%s (%s)\n", r.Path, r.Version, r.Sum)
			}
		}
	}
	return nil
}
