package utils_test

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestFileExists(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "exists-test")
	defer os.Remove(f.Name())
	f.Close()

	if !utils.FileExists(f.Name()) {
		t.Errorf("utils.FileExists(%s) should be true", f.Name())
	}
	if utils.FileExists("/non/existent/path") {
		t.Error("utils.FileExists should be false for non-existent path")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	if !utils.DirExists(tmpDir) {
		t.Errorf("utils.DirExists(%s) should be true", tmpDir)
	}

	f, _ := os.CreateTemp(tmpDir, "file")
	defer os.Remove(f.Name())
	f.Close()

	if utils.DirExists(f.Name()) {
		t.Errorf("utils.DirExists(%s) should be false for file", f.Name())
	}
}

func TestGetDefaultBrowser(t *testing.T) {
	got := utils.GetDefaultBrowser()
	if got == "" {
		t.Error("utils.GetDefaultBrowser returned empty string")
	}
}

func TestIsSQLite(t *testing.T) {
	tmpDir := t.TempDir()

	dbPath := filepath.Join(tmpDir, "test.db")
	os.WriteFile(dbPath, []byte("SQLite format 3\x00extra data"), 0o644)

	if !utils.IsSQLite(dbPath) {
		t.Error("utils.IsSQLite should be true for valid header")
	}

	notDBPath := filepath.Join(tmpDir, "not.db")
	os.WriteFile(notDBPath, []byte("Not a sqlite file"), 0o644)
	if utils.IsSQLite(notDBPath) {
		t.Error("utils.IsSQLite should be false for invalid header")
	}

	if utils.IsSQLite("/non/existent") {
		t.Error("utils.IsSQLite should be false for non-existent file")
	}
}

func TestReadLines(t *testing.T) {
	input := `line1
  line2  

line3
`
	r := strings.NewReader(input)
	got := utils.ReadLines(r)
	expected := []string{"line1", "line2", "line3"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.ReadLines failed: got %v, want %v", got, expected)
	}
}

func TestExpandStdin(t *testing.T) {
	origStdin := utils.Stdin
	defer func() { utils.Stdin = origStdin }()

	input := `line1
line2`
	utils.Stdin = strings.NewReader(input)
	got := utils.ExpandStdin([]string{"-", "direct"})
	expected := []string{"line1", "line2", "direct"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.ExpandStdin failed: got %v, want %v", got, expected)
	}
}

func TestConfirm(t *testing.T) {
	origStdin := utils.Stdin
	origStdout := utils.Stdout
	defer func() {
		utils.Stdin = origStdin
		utils.Stdout = origStdout
	}()
	utils.Stdout = io.Discard

	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"Y\n", true},
		{"n\n", false},
		{"no\n", false},
		{"maybe\n", false},
		{"\n", false},
	}

	for _, tt := range tests {
		utils.Stdin = strings.NewReader(tt.input)
		if got := utils.Confirm("Is this a test?"); got != tt.want {
			t.Errorf("utils.Confirm(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPrompt(t *testing.T) {
	origStdin := utils.Stdin
	origStdout := utils.Stdout
	defer func() {
		utils.Stdin = origStdin
		utils.Stdout = origStdout
	}()
	utils.Stdout = io.Discard

	input := "test response\n"
	utils.Stdin = strings.NewReader(input)
	if got := utils.Prompt("Enter something"); got != "test response" {
		t.Errorf("utils.Prompt() = %q, want %q", got, "test response")
	}
}
