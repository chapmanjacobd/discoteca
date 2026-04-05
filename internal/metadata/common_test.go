package metadata_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func createMock(t *testing.T, tmpDir, name, content string) string {
	fullName := name
	if runtime.GOOS == "windows" {
		fullName += ".exe"
	}
	binPath := filepath.Join(tmpDir, fullName)

	// Create a small Go program that prints the content
	goFile := filepath.Join(tmpDir, name+".go")
	goSource := fmt.Sprintf(`package main
import "fmt"
import "os"
func main() {
	if "%s" == "ffmpeg" {
		for _, arg := range os.Args {
			if arg == "20.00" {
				os.Exit(1)
			}
		}
		os.Exit(0)
	}
	fmt.Print(`+"`"+`%s`+"`"+`)
}
`, name, content)

	if err := os.WriteFile(goFile, []byte(goSource), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "build", "-o", binPath, goFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build mock %s: %v\nOutput: %s", name, err, string(out))
	}

	return binPath
}
