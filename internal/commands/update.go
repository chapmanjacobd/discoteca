package commands

import (
	"context"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type UpdateCmd struct{}

func (c *UpdateCmd) Run(ctx context.Context) error {
	utils.MaybeUpdate(ctx)
	return nil
}
