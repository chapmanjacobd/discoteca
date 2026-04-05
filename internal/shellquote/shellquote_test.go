package shellquote_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/shellquote"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{"", "''"},
		{"foo", "foo"},
		{"foo*", "'foo*'"},
		{"foo bar", "'foo bar'"},
		{"foo'bar", "'foo'\\''bar'"},
		{"'foo", "\\''foo'"},
		{"foo'foo", "'foo'\\''foo'"},
		{"\\", "'\\'"},
		{"'", "\\'"},
		{"\\'", "'\\'\\'"},
		{"a''b", "'a'\"''\"'b'"},
		{"azAZ09_!%+,-./:@^", "azAZ09_!%+,-./:@^"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := shellquote.ShellQuote(tt.in)
			if got != tt.expected {
				t.Errorf("ShellQuote(%q) = %q; want %q", tt.in, got, tt.expected)
			}
		})
	}
}
