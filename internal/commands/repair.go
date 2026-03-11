package commands

import (
	"github.com/alecthomas/kong"
	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type RepairCmd struct {
	models.CoreFlags `embed:""`
	Database         string `arg:"" required:"" help:"Database file to repair" type:"existingfile"`
}

func (c *RepairCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	return database.Repair(c.Database)
}
