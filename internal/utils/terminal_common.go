package utils

import (
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/term"
)

// ClearSeq is the escape sequence to clear from cursor to end of line
const ClearSeq = "\033[K"

// commandExistsCache caches the result of command existence checks
var commandExistsCache sync.Map

// getExecutableDir returns the directory containing the current executable
var getExecutableDir = sync.OnceValue(func() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
})

// CommandExists checks if a command is available in PATH or common installation paths
// Results are cached to avoid repeated syscalls
func CommandExists(cmd string) bool {
	if cached, ok := commandExistsCache.Load(cmd); ok {
		return cached.(bool)
	}

	path := GetCommandPath(cmd)
	exists := path != ""
	commandExistsCache.Store(cmd, exists)
	return exists
}

// TerminalSize tracks the current terminal width and updates on resize
type TerminalSize struct {
	mu       sync.RWMutex
	width    int
	height   int
	initOnce sync.Once
}

var terminalSize TerminalSize

// GetTerminalWidth returns the current terminal width
func GetTerminalWidth() int {
	terminalSize.initOnce.Do(func() {
		terminalSize.updateSize()
		terminalSize.watchResize()
	})
	terminalSize.mu.RLock()
	defer terminalSize.mu.RUnlock()
	return terminalSize.width
}

// GetTerminalHeight returns the current terminal height
func GetTerminalHeight() int {
	terminalSize.initOnce.Do(func() {
		terminalSize.updateSize()
		terminalSize.watchResize()
	})
	terminalSize.mu.RLock()
	defer terminalSize.mu.RUnlock()
	return terminalSize.height
}

// updateSize gets the current terminal size
func (t *TerminalSize) updateSize() {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w = 80 // Fallback
		h = 24
	}
	t.mu.Lock()
	t.width = w
	t.height = h
	t.mu.Unlock()
}

// TruncateMiddle truncates a string in the middle with ellipsis
func TruncateMiddle(s string, maxWidth int) string {
	if s == "" {
		return ""
	}
	if len(s) <= maxWidth {
		return s
	}
	half := max(maxWidth/2, 2)
	return s[:half-1] + "…" + s[len(s)-half+1:]
}

// GetPathDisplayWidth returns the recommended width for displaying paths
// Takes into account terminal width and leaves room for other columns
func GetPathDisplayWidth() int {
	width := GetTerminalWidth()
	// Reserve space for table columns (about 70 chars for table)
	// Use remaining space for path, but cap at reasonable length
	pathWidth := min(max(width-70, 40), 80)
	return pathWidth
}
