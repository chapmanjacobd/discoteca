package commands

import (
	"context"

	"github.com/chapmanjacobd/discoteca/internal/selfupdate"
)

type UpdateCmd struct{}

func (c *UpdateCmd) Run(ctx context.Context) error {
	selfupdate.MaybeUpdate(ctx)
	return nil
}
