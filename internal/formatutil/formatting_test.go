package formatutil_test

import (
	"testing"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/formatutil"
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
		result := formatutil.FormatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("formatutil.FormatDuration(%d) = %q, want %q", tt.input, result, tt.expected)
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
		result := formatutil.FormatSize(tt.input)
		if result != tt.expected {
			t.Errorf("formatutil.FormatSize(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatTime(t *testing.T) {
	ts := time.Now().Unix()
	expected := time.Unix(ts, 0).Format("2006-01-02 15:04")
	got := formatutil.FormatTime(ts)
	if got != expected {
		t.Errorf("formatutil.FormatTime incorrect, got %s, want %s", got, expected)
	}
	if formatutil.FormatTime(0) != "-" {
		t.Error("formatutil.FormatTime(0) should be -")
	}
}

func TestSecondsToHHMMSS(t *testing.T) {
	if got := formatutil.SecondsToHHMMSS(3661); got != "1:01:01" {
		t.Errorf("formatutil.SecondsToHHMMSS(3661) = %q, want 1:01:01", got)
	}
	if got := formatutil.SecondsToHHMMSS(-3661); got != "-1:01:01" {
		t.Errorf("formatutil.SecondsToHHMMSS(-3661) = %q, want -1:01:01", got)
	}
	if got := formatutil.SecondsToHHMMSS(61); got != "1:01" {
		t.Errorf("formatutil.SecondsToHHMMSS(61) = %q, want 1:01", got)
	}
}
