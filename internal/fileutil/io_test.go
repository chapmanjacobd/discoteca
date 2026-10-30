package fileutil_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/fileutil"
)

func TestFileExists(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "exists-test")
	defer os.Remove(f.Name())
	f.Close()

	if !fileutil.FileExists(f.Name()) {
		t.Errorf("fileutil.FileExists(%s) should be true", f.Name())
	}
	if fileutil.FileExists("/non/existent/path") {
		t.Error("fileutil.FileExists should be false for non-existent path")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	if !fileutil.DirExists(tmpDir) {
		t.Errorf("fileutil.DirExists(%s) should be true", tmpDir)
	}

	f, _ := os.CreateTemp(tmpDir, "file")
	defer os.Remove(f.Name())
	f.Close()

	if fileutil.DirExists(f.Name()) {
		t.Errorf("fileutil.DirExists(%s) should be false for file", f.Name())
	}
}

func TestGetDefaultBrowser(t *testing.T) {
	got := fileutil.GetDefaultBrowser()
	if got == "" {
		t.Error("fileutil.GetDefaultBrowser returned empty string")
	}
}

func TestIsSQLite(t *testing.T) {
	tmpDir := t.TempDir()

	dbPath := filepath.Join(tmpDir, "test.db")
	os.WriteFile(dbPath, []byte("SQLite format 3\x00extra data"), 0o644)

	if !fileutil.IsSQLite(dbPath) {
		t.Error("fileutil.IsSQLite should be true for valid header")
	}

	notDBPath := filepath.Join(tmpDir, "not.db")
	os.WriteFile(notDBPath, []byte("Not a sqlite file"), 0o644)
	if fileutil.IsSQLite(notDBPath) {
		t.Error("fileutil.IsSQLite should be false for invalid header")
	}

	if fileutil.IsSQLite("/non/existent") {
		t.Error("fileutil.IsSQLite should be false for non-existent file")
	}
}

func TestReadLines(t *testing.T) {
	input := `line1
  line2  

line3
`
	r := strings.NewReader(input)
	got := fileutil.ReadLines(r)
	expected := []string{"line1", "line2", "line3"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("fileutil.ReadLines failed: got %v, want %v", got, expected)
	}
}

func TestExpandStdin(t *testing.T) {
	origStdin := fileutil.Stdin
	defer func() { fileutil.Stdin = origStdin }()

	input := `line1
line2`
	fileutil.Stdin = strings.NewReader(input)
	got := fileutil.ExpandStdin([]string{"-", "direct"})
	expected := []string{"line1", "line2", "direct"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("fileutil.ExpandStdin failed: got %v, want %v", got, expected)
	}
}
