package commands

import (
	"context"
)

// BigDirsCmd is a thin alias for `print --big-dirs`.
type BigDirsCmd struct {
	PrintCmd `embed:""`
}

func (c *BigDirsCmd) Run(ctx context.Context) error {
	c.BigDirs = true
	return c.PrintCmd.Run(ctx)
}
