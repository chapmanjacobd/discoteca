package utils

import (
	"io"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestMain(m *testing.M) {
	// Silence Stdout and Stderr during tests
	origStdout := Stdout
	origStderr := Stderr
	Stdout = io.Discard
	Stderr = io.Discard
	defer func() {
		Stdout = origStdout
		Stderr = origStderr
	}()

	// Silence slog during tests by setting Log to discard
	models.Log = &discardLogger{}

	os.Exit(m.Run())
}

type discardLogger struct{}

func (d *discardLogger) Info(msg string, args ...any)  {}
func (d *discardLogger) Debug(msg string, args ...any) {}
func (d *discardLogger) Warn(msg string, args ...any)  {}
func (d *discardLogger) Error(msg string, args ...any) {}
func (d *discardLogger) With(args ...any) models.Logger {
	return d
}
