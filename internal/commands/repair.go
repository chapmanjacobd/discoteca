package commands

import (
	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type RepairCmd struct {
	models.CoreFlags `embed:""`

	Database string `help:"Database file to repair" required:"" arg:"" type:"existingfile"`
}

func (c *RepairCmd) Run() error {
	models.SetupLogging(c.Verbose)
	return database.Repair(c.Database)
}
