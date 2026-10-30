package formatutil_test

import (
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/formatutil"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected int64 // Just checking if it returns > 0 for valid dates
	}{
		{"2024-01-01", 1704067200},
		{"01/01/2024", 1704067200},
		{"invalid", 0},
	}
	for _, tt := range tests {
		got := formatutil.ParseDate(tt.input)
		if tt.expected > 0 && got <= 0 {
			t.Errorf("formatutil.ParseDate(%q) expected positive timestamp, got %v", tt.input, got)
		}
		if tt.expected == 0 && got != 0 {
			t.Errorf("formatutil.ParseDate(%q) expected 0, got %v", tt.input, got)
		}
	}
}

func TestSuperParser(t *testing.T) {
	tests := []string{
		"2024-05-20",
		"May 20, 2024",
		"20/05/2024",
		"2024/05/20 15:04:05",
	}
	for _, tt := range tests {
		got := formatutil.SuperParser(tt)
		if got == nil {
			t.Errorf("formatutil.SuperParser(%q) expected valid time, got nil", tt)
		}
	}
}

func TestSpecificDate(t *testing.T) {
	// Should pick earliest most-specific past date
	d1 := "2020-01-01" // specific
	d2 := "2019-05-20" // earlier and specific
	d3 := "2025-01-01" // future (assuming now > 2025)

	got := formatutil.SpecificDate(d1, d2, d3)
	if got == nil {
		t.Fatal("formatutil.SpecificDate returned nil")
	}

	t2 := *formatutil.SuperParser(d2)
	if got.Unix() != t2.Unix() {
		t.Errorf("formatutil.SpecificDate expected %v, got %v", t2, *got)
	}
}

func TestParseDateOrRelative(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		input    string
		expected int64
		margin   int64
	}{
		{"2024-01-01", 1704067200, 0},
		{"3 days", now - 3*86400, 5},
		{"-1 week", now - 7*86400, 5},
		{"+1 hour", now + 3600, 5},
	}

	for _, tt := range tests {
		got := formatutil.ParseDateOrRelative(tt.input)
		if tt.margin == 0 {
			if got != tt.expected {
				t.Errorf("formatutil.ParseDateOrRelative(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		} else {
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.margin {
				t.Errorf(
					"formatutil.ParseDateOrRelative(%q) = %v, want near %v (diff %v)",
					tt.input,
					got,
					tt.expected,
					diff,
				)
			}
		}
	}
}

func TestSpecificDateExtra(t *testing.T) {
	// Pick most specific date
	d1 := "2020-01-01" // Jan 1st is less specific than d2
	d2 := "2020-05-20" // more specific (has month/day != 1)
	got := formatutil.SpecificDate(d1, d2)
	if got == nil || got.Unix() != formatutil.SuperParser(d2).Unix() {
		t.Errorf("formatutil.SpecificDate failed to pick more specific date: %v", got)
	}

	if got := formatutil.SpecificDate(""); got != nil {
		t.Errorf("formatutil.SpecificDate empty failed: %v", got)
	}
}
