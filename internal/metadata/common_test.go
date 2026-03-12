package metadata

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func createMock(t *testing.T, tmpDir, name, content string) string {
	fullName := name
	if runtime.GOOS == "windows" {
		fullName += ".bat"
	}
	path := filepath.Join(tmpDir, fullName)

	actualContent := content
	if runtime.GOOS == "windows" {
		if name == "ffmpeg" {
			actualContent = `@echo off
for %%a in (%*) do (
    if "%%a"=="20.00" exit /b 1
)
exit /b 0`
		} else {
			// On Windows, escaping JSON for echo in a .bat is painful.
			escaped := strings.ReplaceAll(content, "\"", "^\"")
			actualContent = "@echo off\necho " + escaped
		}
	} else {
		if name == "ffmpeg" {
			actualContent = "#!/bin/sh\n" + content
		} else {
			actualContent = "#!/bin/sh\necho '" + strings.ReplaceAll(content, "'", "'\\''") + "'"
		}
	}

	if err := os.WriteFile(path, []byte(actualContent), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
