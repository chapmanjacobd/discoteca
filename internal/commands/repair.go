package commands

import (
	"context"

	database "github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type RepairCmd struct {
	models.CoreFlags `embed:""`

	Database string `help:"Database file to repair" required:"true" arg:"" type:"existingfile"`
}

func (c *RepairCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	return database.Repair(ctx, c.Database)
}
