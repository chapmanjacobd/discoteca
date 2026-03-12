package utils

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CmdResult wraps the output of a command execution
type CmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Cmd runs a command and returns its output and exit code
func Cmd(name string, args ...string) (CmdResult, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := CmdResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
			if strings.TrimSpace(res.Stderr) != "" {
				return res, fmt.Errorf("command failed with exit code %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			return res, fmt.Errorf("command failed with exit code %d", res.ExitCode)
		}
		return res, err
	}

	return res, nil
}

// CmdInteractive runs a command connected to the current process's terminal
func CmdInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = Stdout
	cmd.Stderr = Stderr
	cmd.Stdin = Stdin

	return cmd.Run()
}

// CmdWithTimeout runs a command with a timeout
func CmdWithTimeout(ctx context.Context, name string, args ...string) (CmdResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := CmdResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
			if strings.TrimSpace(res.Stderr) != "" {
				return res, fmt.Errorf("command failed with exit code %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			return res, fmt.Errorf("command failed with exit code %d", res.ExitCode)
		}
		return res, err
	}

	return res, nil
}

// FzfSelect runs fzf with the given items and returns the selected items
func FzfSelect(items []string, multi bool) ([]string, error) {
	args := []string{"--bind", "ctrl-a:toggle-all"}
	if multi {
		args = append(args, "--multi")
	}

	cmd := exec.Command("fzf", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	go func() {
		defer stdin.Close()
		for i := len(items) - 1; i >= 0; i-- {
			fmt.Fprintln(stdin, items[i])
		}
	}()

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil, nil // No selection
		}
		return nil, err
	}

	selected := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(selected) == 1 && selected[0] == "" {
		return nil, nil
	}
	return selected, nil
}

func AdjustDuration(duration int, start *int, end *int) int {
	if start != nil && *start > 0 {
		duration -= *start
	}
	if end != nil && *end > 0 {
		duration = *end
		if start != nil && *start > 0 {
			duration -= *start
		}
	}
	if duration < 0 {
		return 0
	}
	return duration
}

func SizeTimeout(timeoutSize string, totalSize int64) bool {
	if timeoutSize == "" {
		return false
	}
	limit, err := HumanToBytes(timeoutSize)
	if err != nil {
		return false
	}
	return totalSize >= limit
}
