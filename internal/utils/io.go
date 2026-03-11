package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

var (
	Stdin  io.Reader = os.Stdin
	Stdout io.Writer = os.Stdout
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

func Confirm(message string) bool {
	fmt.Fprintf(Stdout, "%s [y/N]: ", message)
	scanner := bufio.NewScanner(Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}
	return false
}

func Prompt(message string) string {
	fmt.Fprintf(Stdout, "%s: ", message)
	scanner := bufio.NewScanner(Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
