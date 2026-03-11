package commands

import (
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type UpdateCmd struct{}

func (c *UpdateCmd) Run() error {
	utils.MaybeUpdate()
	return nil
}
