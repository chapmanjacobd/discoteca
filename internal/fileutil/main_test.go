package fileutil_test

import (
	"io"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
)

func TestMain(m *testing.M) {
	// Silence Stdout and Stderr during tests
	origStdout := fileutil.Stdout
	origStderr := fileutil.Stderr
	fileutil.Stdout = io.Discard
	fileutil.Stderr = io.Discard

	// Set verbose logging (go test will only show output on test failure)
	logging.SetupLogging(2) // Debug level

	// os.Exit(m.Run()) will terminate the program, so defer won't run.
	// But it's fine for tests.
	code := m.Run()
	fileutil.Stdout = origStdout
	fileutil.Stderr = origStderr
	os.Exit(code)
}
