package commands

import (
	"context"
)

// DiskUsageCmd is print with directory aggregation enabled by default.
type DiskUsageCmd struct {
	PrintCmd `embed:""`
}

func (c *DiskUsageCmd) Run(ctx context.Context) error {
	// Disk usage is essentially Print with aggregation by default if no
	// explicit aggregation mode was requested.
	if !c.BigDirs && !c.GroupByExtensions && !c.GroupBySize && c.Depth == 0 && !c.Parents {
		c.BigDirs = true
	}
	return c.PrintCmd.Run(ctx)
}
