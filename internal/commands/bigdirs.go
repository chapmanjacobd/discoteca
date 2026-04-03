package commands

import (
	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

type BigDirsCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.AggregateFlags   `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`

	Databases []string `help:"SQLite database files" required:"" arg:"" type:"existingfile"`
}

func (c *BigDirsCmd) Run(ctx *kong.Context) error {
	// Bigdirs is Essentially Print with BigDirs enabled by default
	c.BigDirs = true
	printCmd := PrintCmd{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		DeletedFlags:     c.DeletedFlags,
		AggregateFlags:   c.AggregateFlags,
		SortFlags:        c.SortFlags,
		DisplayFlags:     c.DisplayFlags,
		Databases:        c.Databases,
	}
	return printCmd.Run(ctx)
}
