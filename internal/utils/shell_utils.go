package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/shellquote"
)

// Trash moves a file to the system trash if available, otherwise deletes it
func Trash(ctx context.Context, flags models.GlobalFlags, path string) error {
	if !FileExists(path) {
		return nil
	}

	if flags.Simulate {
		fmt.Fprintf(Stdout, "trash %s\n", shellquote.ShellQuote(path))
		return nil
	}

	trashCmd := "trash-put" // Prefer trash-put from trash-cli
	err := CmdDetach(ctx, trashCmd, path)
	if err != nil {
		// Try 'trash' if 'trash-put' fails
		err = CmdDetach(ctx, "trash", path)
	}

	if err != nil {
		models.Log.Debug("trash command failed, unlinking instead", "path", path, "error", err)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", path, err)
		}
	}
	return nil
}

// FlattenWrapperFolder removes a single subfolder if it's the only entry in outputDir

// ResolveAbsolutePath expands user home and returns absolute path if it exists
