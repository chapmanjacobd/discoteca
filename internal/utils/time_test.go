package utils_test

import (
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/utils"
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
		got := utils.ParseDate(tt.input)
		if tt.expected > 0 && got <= 0 {
			t.Errorf("utils.ParseDate(%q) expected positive timestamp, got %v", tt.input, got)
		}
		if tt.expected == 0 && got != 0 {
			t.Errorf("utils.ParseDate(%q) expected 0, got %v", tt.input, got)
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
		got := utils.SuperParser(tt)
		if got == nil {
			t.Errorf("utils.SuperParser(%q) expected valid time, got nil", tt)
		}
	}
}

func TestSpecificDate(t *testing.T) {
	// Should pick earliest most-specific past date
	d1 := "2020-01-01" // specific
	d2 := "2019-05-20" // earlier and specific
	d3 := "2025-01-01" // future (assuming now > 2025)

	got := utils.SpecificDate(d1, d2, d3)
	if got == nil {
		t.Fatal("utils.SpecificDate returned nil")
	}

	t2 := *utils.SuperParser(d2)
	if got.Unix() != t2.Unix() {
		t.Errorf("utils.SpecificDate expected %v, got %v", t2, *got)
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
		got := utils.ParseDateOrRelative(tt.input)
		if tt.margin == 0 {
			if got != tt.expected {
				t.Errorf("utils.ParseDateOrRelative(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		} else {
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.margin {
				t.Errorf("utils.ParseDateOrRelative(%q) = %v, want near %v (diff %v)", tt.input, got, tt.expected, diff)
			}
		}
	}
}

func TestSpecificDateExtra(t *testing.T) {
	// Pick most specific date
	d1 := "2020-01-01" // Jan 1st is less specific than d2
	d2 := "2020-05-20" // more specific (has month/day != 1)
	got := utils.SpecificDate(d1, d2)
	if got == nil || got.Unix() != utils.SuperParser(d2).Unix() {
		t.Errorf("utils.SpecificDate failed to pick more specific date: %v", got)
	}

	if got := utils.SpecificDate(""); got != nil {
		t.Errorf("utils.SpecificDate empty failed: %v", got)
	}
}
