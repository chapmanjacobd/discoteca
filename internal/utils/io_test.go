package utils

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFileExists(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "exists-test")
	defer os.Remove(f.Name())
	f.Close()

	if !FileExists(f.Name()) {
		t.Errorf("FileExists(%s) should be true", f.Name())
	}
	if FileExists("/non/existent/path") {
		t.Error("FileExists should be false for non-existent path")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "dir-exists-test")
	defer os.RemoveAll(tmpDir)

	if !DirExists(tmpDir) {
		t.Errorf("DirExists(%s) should be true", tmpDir)
	}

	f, _ := os.CreateTemp(tmpDir, "file")
	defer os.Remove(f.Name())
	f.Close()

	if DirExists(f.Name()) {
		t.Errorf("DirExists(%s) should be false for file", f.Name())
	}
}

func TestGetDefaultBrowser(t *testing.T) {
	got := GetDefaultBrowser()
	if got == "" {
		t.Error("GetDefaultBrowser returned empty string")
	}
}

func TestIsSQLite(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "sqlite-test")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	os.WriteFile(dbPath, []byte("SQLite format 3\x00extra data"), 0o644)

	if !IsSQLite(dbPath) {
		t.Error("IsSQLite should be true for valid header")
	}

	notDBPath := filepath.Join(tmpDir, "not.db")
	os.WriteFile(notDBPath, []byte("Not a sqlite file"), 0o644)
	if IsSQLite(notDBPath) {
		t.Error("IsSQLite should be false for invalid header")
	}

	if IsSQLite("/non/existent") {
		t.Error("IsSQLite should be false for non-existent file")
	}
}

func TestReadLines(t *testing.T) {
	input := `line1
  line2  

line3
`
	r := strings.NewReader(input)
	got := ReadLines(r)
	expected := []string{"line1", "line2", "line3"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ReadLines failed: got %v, want %v", got, expected)
	}
}

func TestExpandStdin(t *testing.T) {
	origStdin := Stdin
	defer func() { Stdin = origStdin }()

	input := `line1
line2`
	Stdin = strings.NewReader(input)
	got := ExpandStdin([]string{"-", "direct"})
	expected := []string{"line1", "line2", "direct"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ExpandStdin failed: got %v, want %v", got, expected)
	}
}

func TestConfirm(t *testing.T) {
	origStdin := Stdin
	origStdout := Stdout
	defer func() {
		Stdin = origStdin
		Stdout = origStdout
	}()
	Stdout = io.Discard

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
		Stdin = strings.NewReader(tt.input)
		if got := Confirm("Is this a test?"); got != tt.want {
			t.Errorf("Confirm(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPrompt(t *testing.T) {
	origStdin := Stdin
	origStdout := Stdout
	defer func() {
		Stdin = origStdin
		Stdout = origStdout
	}()
	Stdout = io.Discard

	input := "test response\n"
	Stdin = strings.NewReader(input)
	if got := Prompt("Enter something"); got != "test response" {
		t.Errorf("Prompt() = %q, want %q", got, "test response")
	}
}
