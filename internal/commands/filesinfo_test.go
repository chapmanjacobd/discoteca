package commands

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestFilesInfoCmd_Run(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &FilesInfoCmd{
		DisplayFlags: models.DisplayFlags{JSON: true},
		Args:         []string{f.Name()},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("FilesInfoCmd failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "text") {
		t.Errorf("Expected output to contain text, got %s", output)
	}
}
