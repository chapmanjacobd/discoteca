package main

import (
	"reflect"
	"testing"
)

func testFlatNames() map[string]bool {
	return map[string]bool{
		"add":         true,
		"files-info":  true,
		"fs":          true,
		"history":     true,
		"history-add": true,
		"pause":       true,
		"play":        true,
	}
}

func TestResolveNestedArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantArgs    []string
		wantGroup   string
		wantMatched bool
	}{
		{
			name:     "flat command unchanged",
			args:     []string{"add", "my.db"},
			wantArgs: []string{"add", "my.db"},
		},
		{
			name:        "canonical nested command",
			args:        []string{"database", "add", "my.db"},
			wantArgs:    []string{"add", "my.db"},
			wantGroup:   "database",
			wantMatched: true,
		},
		{
			name:        "nested import",
			args:        []string{"database", "import", "calibre", "my.db"},
			wantArgs:    []string{"calibre-import", "my.db"},
			wantGroup:   "database",
			wantMatched: true,
		},
		{
			name:        "longest path wins",
			args:        []string{"database", "history", "add", "my.db"},
			wantArgs:    []string{"history-add", "my.db"},
			wantGroup:   "database",
			wantMatched: true,
		},
		{
			name:        "shorter path still resolves",
			args:        []string{"database", "history", "my.db"},
			wantArgs:    []string{"history", "my.db"},
			wantGroup:   "database",
			wantMatched: true,
		},
		{
			name:        "group alias",
			args:        []string{"db", "add", "my.db"},
			wantArgs:    []string{"add", "my.db"},
			wantGroup:   "database",
			wantMatched: true,
		},
		{
			name:        "filesystem usage",
			args:        []string{"filesystem", "usage", "my.db"},
			wantArgs:    []string{"disk-usage", "my.db"},
			wantGroup:   "filesystem",
			wantMatched: true,
		},
		{
			name:        "nested part alias",
			args:        []string{"filesystem", "ls"},
			wantArgs:    []string{"files-info"},
			wantGroup:   "filesystem",
			wantMatched: true,
		},
		{
			name:        "flat alias group takes nested path",
			args:        []string{"fs", "usage", "my.db"},
			wantArgs:    []string{"disk-usage", "my.db"},
			wantGroup:   "filesystem",
			wantMatched: true,
		},
		{
			name:     "flat alias keeps working",
			args:     []string{"fs"},
			wantArgs: []string{"fs"},
		},
		{
			name:     "flat command wins over group name",
			args:     []string{"play"},
			wantArgs: []string{"play"},
		},
		{
			name:        "play group",
			args:        []string{"play", "video", "my.db"},
			wantArgs:    []string{"watch", "my.db"},
			wantGroup:   "play",
			wantMatched: true,
		},
		{
			name:        "single-level group command",
			args:        []string{"captions", "my.db", "word"},
			wantArgs:    []string{"search-captions", "my.db", "word"},
			wantGroup:   "captions",
			wantMatched: true,
		},
		{
			name:        "flags are preserved",
			args:        []string{"database", "add", "--video-only", "my.db", "/media"},
			wantArgs:    []string{"add", "--video-only", "my.db", "/media"},
			wantGroup:   "database",
			wantMatched: true,
		},
		{
			name:        "help flag is preserved",
			args:        []string{"database", "add", "--help"},
			wantArgs:    []string{"add", "--help"},
			wantGroup:   "database",
			wantMatched: true,
		},
		{
			name:      "bare group needs help",
			args:      []string{"database"},
			wantArgs:  []string{"database"},
			wantGroup: "database",
		},
		{
			name:      "unknown nested command",
			args:      []string{"database", "nope"},
			wantArgs:  []string{"database", "nope"},
			wantGroup: "database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveNestedArgs(tt.args, testFlatNames())
			if !reflect.DeepEqual(got.Args, tt.wantArgs) {
				t.Errorf("Args = %v, want %v", got.Args, tt.wantArgs)
			}
			if got.Group != tt.wantGroup {
				t.Errorf("Group = %q, want %q", got.Group, tt.wantGroup)
			}
			if got.Matched != tt.wantMatched {
				t.Errorf("Matched = %v, want %v", got.Matched, tt.wantMatched)
			}
		})
	}
}
