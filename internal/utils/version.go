package utils

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
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
)

func init() {
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
}

func RenderVersion(w io.Writer) {
	fmt.Fprintf(w, "discoteca built from %s on %s with %s\n",
		Version, When, runtime.Version())

	fmt.Fprintln(w, "Build Settings:")
	for _, s := range BuildSettings {
		fmt.Fprintf(w, "\t%s=%s\n", s.Key, s.Value)
	}

	fmt.Fprintln(w, "\nDeps:")
	for _, dep := range Deps {
		fmt.Fprintf(w, "\t%s@%s (%s)\n", dep.Path, dep.Version, dep.Sum)
		if dep.Replace != nil {
			r := dep.Replace
			fmt.Fprintf(w, "   replaced by %s@%s (%s)\n", r.Path, r.Version, r.Sum)
		}
	}
}
