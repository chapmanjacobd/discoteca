package utils_test

import (
	"strings"
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "-"},
		{45, "0:45"},
		{60, "1:00"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
	}

	for _, tt := range tests {
		result := utils.FormatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("utils.FormatDuration(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{45, "45 seconds"},
		{60, "1 minute"},
		{66, "1.1 minutes"},
		{3600, "1 hour"},
		{86400, "1 day"},
		{172800, "2 days"},
		{946684800, "30 years and 7 days"},
	}

	for _, tt := range tests {
		result := utils.FormatDurationShort(tt.input)
		if result != tt.expected {
			t.Errorf("utils.FormatDurationShort(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "-"},
		{500, "500 B"},
		{1000, "1.0 KB"},
		{1000 * 1000, "1.0 MB"},
		{1000 * 1000 * 1000, "1.0 GB"},
	}

	for _, tt := range tests {
		result := utils.FormatSize(tt.input)
		if result != tt.expected {
			t.Errorf("utils.FormatSize(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatPlaybackDuration(t *testing.T) {
	tests := []struct {
		duration     int64
		segmentStart int64
		segmentEnd   int64
		expected     string
	}{
		{0, 0, 0, "Duration: 0:00"},
		{360, 1000, 0, "Duration: 6:00 (16:40 to 22:40)"},
		{3600, 0, 0, "Duration: 1:00:00"},
		{3600, 0, 3000, "Duration: 50:00 (0:00 to 50:00)"},
		{3600, 1800, 0, "Duration: 30:00 (30:00 to 1:00:00)"},
		{3600, 1800, 3000, "Duration: 20:00 (30:00 to 50:00)"},
		{3600, 3000, 0, "Duration: 10:00 (50:00 to 1:00:00)"},
		{3600, 3000, 2000, "Duration: 16:40 (33:20 to 50:00)"}, // Swap: 3000, 2000 -> 2000, 3000.
	}

	for _, tt := range tests {
		result := utils.FormatPlaybackDuration(tt.duration, tt.segmentStart, tt.segmentEnd)
		if result != tt.expected {
			t.Errorf(
				"utils.FormatPlaybackDuration(%d, %d, %d) = %q, want %q",
				tt.duration,
				tt.segmentStart,
				tt.segmentEnd,
				result,
				tt.expected,
			)
		}
	}
}

func TestFormatTime(t *testing.T) {
	ts := time.Now().Unix()
	expected := time.Unix(ts, 0).Format("2006-01-02 15:04")
	got := utils.FormatTime(ts)
	if got != expected {
		t.Errorf("utils.FormatTime incorrect, got %s, want %s", got, expected)
	}
	if utils.FormatTime(0) != "-" {
		t.Error("utils.FormatTime(0) should be -")
	}
}

func TestRelativeDatetime(t *testing.T) {
	now := time.Now()
	if utils.RelativeDatetime(0) != "-" {
		t.Error("utils.RelativeDatetime(0) should be -")
	}

	// Test today
	got := utils.RelativeDatetime(now.Unix())
	if !strings.HasPrefix(got, "today") {
		t.Errorf("utils.RelativeDatetime today failed, got %s", got)
	}

	// Test yesterday
	yesterday := now.AddDate(0, 0, -1).Unix()
	got = utils.RelativeDatetime(yesterday)
	if !strings.HasPrefix(got, "yesterday") {
		t.Errorf("utils.RelativeDatetime yesterday failed, got %s", got)
	}

	// Test 5 days ago (use a slightly larger offset to ensure it doesn't round down to 4)
	fiveDaysAgo := now.Add(-5*24*time.Hour - 1*time.Minute).Unix()
	got = utils.RelativeDatetime(fiveDaysAgo)
	if !strings.Contains(got, "5 days ago") {
		t.Errorf("utils.RelativeDatetime 5 days ago failed, got %s", got)
	}

	// Test tomorrow
	tomorrow := now.AddDate(0, 0, 1).Unix()
	got = utils.RelativeDatetime(tomorrow)
	if !strings.HasPrefix(got, "tomorrow") {
		t.Errorf("utils.RelativeDatetime tomorrow failed, got %s", got)
	}

	// Test in 5 days
	inFiveDays := now.Add(5*24*time.Hour + 1*time.Minute).Unix()
	got = utils.RelativeDatetime(inFiveDays)
	if !strings.Contains(got, "in 5 days") {
		t.Errorf("utils.RelativeDatetime in 5 days failed, got %s", got)
	}
}

func TestSecondsToHHMMSS(t *testing.T) {
	if got := utils.SecondsToHHMMSS(3661); got != "1:01:01" {
		t.Errorf("utils.SecondsToHHMMSS(3661) = %q, want 1:01:01", got)
	}
	if got := utils.SecondsToHHMMSS(-3661); got != "-1:01:01" {
		t.Errorf("utils.SecondsToHHMMSS(-3661) = %q, want -1:01:01", got)
	}
	if got := utils.SecondsToHHMMSS(61); got != "1:01" {
		t.Errorf("utils.SecondsToHHMMSS(61) = %q, want 1:01", got)
	}
}
