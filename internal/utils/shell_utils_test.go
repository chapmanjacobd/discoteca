package utils_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/shellquote"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "''"},
		{"safe", "safe"},
		{"/path/to/file", "/path/to/file"},
		{"file with spaces", "'file with spaces'"},
		{"it's a file", "'it'\\''s a file'"},
		{"$", "'$'"},
	}

	for _, tt := range tests {
		got := shellquote.ShellQuote(tt.input)
		if got != tt.expected {
			t.Errorf("ShellQuote(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
