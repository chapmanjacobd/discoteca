//go:build windows

package utils_test

import "github.com/chapmanjacobd/discoteca/internal/utils"


import (
	"os"
	"syscall"
	"testing"
)

func TestIsFileOpen(t *testing.T) {
	f, err := os.CreateTemp("", "is-open-test")
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
	// On Windows, open with no sharing to ensure utils.IsFileOpen detects it
	ptr, _ := syscall.UTF16PtrFromString(f.Name())
	handle, err := syscall.CreateFile(
		ptr,
		syscall.GENERIC_READ,
		0, // No sharing
		nil,
		syscall.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("Failed to open file with exclusive access for test: %v", err)
	}
	defer syscall.CloseHandle(handle)

	if !utils.IsFileOpen(f.Name()) {
		t.Errorf("Expected file %s to be open", f.Name())
	}
}
