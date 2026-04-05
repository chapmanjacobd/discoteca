package utils_test

import (
	"io"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestMain(m *testing.M) {
	// Silence Stdout and Stderr during tests
	origStdout := utils.Stdout
	origStderr := utils.Stderr
	utils.Stdout = io.Discard
	utils.Stderr = io.Discard

	// Set verbose logging (go test will only show output on test failure)
	models.SetupLogging(2) // Debug level

	// os.Exit(m.Run()) will terminate the program, so defer won't run.
	// But it's fine for tests.
	code := m.Run()
	utils.Stdout = origStdout
	utils.Stderr = origStderr
	os.Exit(code)
}
