package utils_test

import (
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
