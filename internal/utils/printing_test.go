package utils_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestPrintOverwrite(t *testing.T) {
	origStdout := utils.Stdout
	defer func() { utils.Stdout = origStdout }()

	var buf bytes.Buffer
	utils.Stdout = &buf

	utils.PrintOverwrite("test message")

	got := buf.String()
	if !strings.Contains(got, "test message") {
		t.Errorf("utils.PrintOverwrite failed, got: %q", got)
	}
}

func TestColNaturalDate(t *testing.T) {
	tests := []struct {
		name      string
		data      []map[string]any
		col       string
		checkFunc func([]map[string]any) bool
	}{
		{
			"valid timestamps",
			[]map[string]any{
				{"time": int64(1708682400)},
				{"time": "invalid"},
				{"other": 1},
			},
			"time",
			func(data []map[string]any) bool {
				return reflect.TypeOf(data[0]["time"]).Kind() == reflect.String &&
					data[1]["time"] == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ColNaturalDate(tt.data, tt.col)
			if !tt.checkFunc(got) {
				t.Errorf("utils.ColNaturalDate check failed")
			}
		})
	}
}

func TestColFilesize(t *testing.T) {
	tests := []struct {
		name     string
		data     []map[string]any
		col      string
		expected string
	}{
		{
			"valid sizes",
			[]map[string]any{
				{"size": int64(1024)},
				{"size": int64(0)},
			},
			"size",
			"1.0 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ColFilesize(tt.data, tt.col)
			if got[0]["size"] != tt.expected {
				t.Errorf("Expected %s, got %v", tt.expected, got[0]["size"])
			}
			if got[1]["size"] != nil {
				t.Errorf("Expected nil for 0 size, got %v", got[1]["size"])
			}
		})
	}
}

func TestColDuration(t *testing.T) {
	tests := []struct {
		name     string
		data     []map[string]any
		col      string
		expected string
	}{
		{
			"valid durations",
			[]map[string]any{
				{"dur": 61},
				{"dur": 0},
			},
			"dur",
			"1:01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ColDuration(tt.data, tt.col)
			if got[0]["dur"] != tt.expected {
				t.Errorf("Expected %s, got %v", tt.expected, got[0]["dur"])
			}
			if got[1]["dur"] != "" {
				t.Errorf("Expected empty string for 0 duration, got %v", got[1]["dur"])
			}
		})
	}
}
