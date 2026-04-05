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

func TestTubeDate(t *testing.T) {
	v := map[string]any{
		"upload_date": "20240520",
		"title":       "test",
	}
	got := utils.TubeDate(v)
	if got == nil {
		t.Fatal("utils.TubeDate returned nil")
	}

	if _, ok := v["upload_date"]; ok {
		t.Error("utils.TubeDate should have removed upload_date from map")
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

func TestIsTZAware(t *testing.T) {
	utc := time.Now().UTC()
	if utils.IsTZAware(utc) {
		t.Error("UTC should not be TZAware")
	}

	loc := time.FixedZone("TEST", 3600)
	testTime := time.Date(2020, 1, 1, 0, 0, 0, 0, loc)
	if !utils.IsTZAware(testTime) {
		t.Error("FixedZone should be TZAware")
	}
}

func TestTubeDateExtra(t *testing.T) {
	v1 := map[string]any{"timestamp": int64(40000000)}
	got1 := utils.TubeDate(v1)
	if got1 == nil || *got1 != 40000000 {
		t.Errorf("utils.TubeDate int64 failed: %v", got1)
	}

	v2 := map[string]any{"timestamp": int(40000000)}
	got2 := utils.TubeDate(v2)
	if got2 == nil || *got2 != 40000000 {
		t.Errorf("utils.TubeDate int failed: %v", got2)
	}

	now := time.Now()
	v3 := map[string]any{"timestamp": now}
	got3 := utils.TubeDate(v3)
	if got3 == nil || *got3 != now.Unix() {
		t.Errorf("utils.TubeDate time.Time failed: %v", got3)
	}

	v4 := map[string]any{"timestamp": nil}
	if got4 := utils.TubeDate(v4); got4 != nil {
		t.Errorf("utils.TubeDate nil failed: %v", got4)
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
