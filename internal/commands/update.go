package commands

import (
	"context"
	"fmt"

	"github.com/chapmanjacobd/discoteca/internal/selfupdate"
)

type UpdateCmd struct{}

func (c *UpdateCmd) Run(ctx context.Context) error {
	updated, err := selfupdate.MaybeUpdateE(ctx)
	if err != nil {
		return err
	}
	if updated {
		fmt.Println("Update installed; restart disco to use the new version")
	} else {
		fmt.Println("No update available for this platform")
	}
	return nil
}
