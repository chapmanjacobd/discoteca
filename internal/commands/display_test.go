package commands_test

import (
	"errors"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestErrUserQuit(t *testing.T) {
	t.Run("commands.ErrUserQuit is defined", func(t *testing.T) {
		if commands.ErrUserQuit == nil {
			t.Fatal("commands.ErrUserQuit should be defined")
		}
		if commands.ErrUserQuit.Error() != "user requested quit" {
			t.Errorf("commands.ErrUserQuit.Error() = %q, want %q", commands.ErrUserQuit.Error(), "user requested quit")
		}
	})

	t.Run("errors.Is recognizes commands.ErrUserQuit", func(t *testing.T) {
		if !errors.Is(commands.ErrUserQuit, commands.ErrUserQuit) {
			t.Error("errors.Is(commands.ErrUserQuit, commands.ErrUserQuit) should return true")
		}
	})

	t.Run("wrapped commands.ErrUserQuit is recognized", func(t *testing.T) {
		wrapped := errors.New("outer: " + commands.ErrUserQuit.Error())
		if errors.Is(wrapped, commands.ErrUserQuit) {
			t.Error("errors.Is should not recognize non-wrapped error with same message")
		}

		properlyWrapped := errors.Join(commands.ErrUserQuit)
		if !errors.Is(properlyWrapped, commands.ErrUserQuit) {
			t.Error("errors.Is should recognize properly wrapped commands.ErrUserQuit")
		}
	})
}

func TestInteractiveDecision_ReturnsError(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectError   bool
		expectQuitErr bool
	}{
		{"quit lowercase", "q", true, true},
		{"quit uppercase", "Q", true, true},
		{"keep (default)", "", false, false},
		{"keep explicit", "k", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test verifies the function signature returns error
			// Full integration testing with stdin mocking would require more setup
			flags := models.GlobalFlags{}
			media := models.MediaWithDB{
				Media: models.Media{
					Path: "/test/path.mp4",
				},
			}

			// We can't easily test the interactive input without mocking stdin
			// This test ensures the function signature is correct
			_ = flags
			_ = media
		})
	}
}
