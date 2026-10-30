package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/logging"
)

type ExplodeCmd struct {
	TargetDir string `help:"Directory to create symlinks in" default:"." arg:"" optional:""`
}

func (c *ExplodeCmd) Run(ctx *kong.Context) error {
	absTarget, err := filepath.Abs(c.TargetDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(absTarget, 0o755); err != nil {
		return err
	}

	discoPath, err2 := os.Executable()
	if err2 != nil {
		return err2
	}

	for _, cmd := range ctx.Model.Node.Children {
		if cmd.Name == "explode" || cmd.Hidden {
			continue
		}

		linkPath := filepath.Join(absTarget, cmd.Name)
		if _, err := os.Lstat(linkPath); err == nil {
			os.Remove(linkPath)
		}

		if err := os.Symlink(discoPath, linkPath); err != nil {
			logging.Log.Error("Failed to create symlink", "command", cmd.Name, "path", linkPath, "error", err)
		} else {
			fmt.Printf("Created: %s -> %s\n", linkPath, discoPath)
		}
	}

	return nil
}
