//go:build !windows

package utils_test

import (
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestIsFileOpen(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "is-open-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Test closed file
	f.Close()
	if utils.IsFileOpen(f.Name()) {
		t.Errorf("Expected file %s to be closed", f.Name())
	}

	// Test open file
	f2, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	if !utils.IsFileOpen(f.Name()) {
		t.Errorf("Expected file %s to be open", f.Name())
	}
}
