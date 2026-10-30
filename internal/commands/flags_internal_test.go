package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDestructiveFlagsAreRead guards against exposing a documented destructive
// flag that no command reads. It is intentionally simple: the field names in
// this list must appear in at least one non-test file in this package.
func TestDestructiveFlagsAreRead(t *testing.T) {
	destructiveFields := []string{
		"DeleteFiles", "DeleteRows", "MarkDeleted", "MoveTo", "CopyTo", "Trash", "PostAction",
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	var source strings.Builder
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		source.Write(data)
	}

	for _, field := range destructiveFields {
		if !strings.Contains(source.String(), "."+field) {
			t.Errorf("destructive flag %s is exposed but never read in internal/commands", field)
		}
	}
}
