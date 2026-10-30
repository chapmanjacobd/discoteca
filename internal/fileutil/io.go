package fileutil

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strings"
)

var (
	Stdin  io.Reader = os.Stdin
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func GetDefaultBrowser() string {
	switch runtime.GOOS {
	case "linux":
		return "xdg-open"
	case "darwin":
		return "open"
	case "windows":
		// 'start' is a cmd.exe builtin, not an executable
		return "cmd"
	}
	return "xdg-open"
}

func IsSQLite(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 16)
	if _, err := f.Read(header); err != nil {
		return false
	}
	return string(header) == "SQLite format 3\x00"
}

func ReadLines(r io.Reader) []string {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func ExpandStdin(paths []string) []string {
	var out []string
	for _, p := range paths {
		if p == "-" {
			out = append(out, ReadLines(Stdin)...)
		} else {
			out = append(out, p)
		}
	}
	return out
}

// PrintJSON encodes data as indented JSON to stdout
func PrintJSON(data any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
