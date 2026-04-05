package testutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestSetupCleanup(t *testing.T) {
	var tempDir string
	{
		f := testutils.Setup(t)
		tempDir = f.TempDir

		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Errorf("TempDir %s does not exist after Setup", tempDir)
		}

		if f.DBPath != filepath.Join(tempDir, "test.db") {
			t.Errorf("Unexpected DBPath: %s", f.DBPath)
		}

		f.Cleanup()
	}

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("TempDir %s still exists after Cleanup", tempDir)
	}
}

func TestCreateDummyFile(t *testing.T) {
	f := testutils.Setup(t)
	defer f.Cleanup()

	path := f.CreateDummyFile("dummy.txt")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Dummy file %s does not exist", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "dummy data" {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestCreateFileTree(t *testing.T) {
	f := testutils.Setup(t)
	defer f.Cleanup()

	tree := map[string]any{
		"dir1": map[string]any{
			"file1.txt": "content1",
			"file2.txt": 123, // should result in "dummy"
		},
		"file3.txt": "content3",
	}

	f.CreateFileTree(tree)

	checks := []struct {
		path    string
		content string
	}{
		{filepath.Join(f.TempDir, "dir1", "file1.txt"), "content1"},
		{filepath.Join(f.TempDir, "dir1", "file2.txt"), "dummy"},
		{filepath.Join(f.TempDir, "file3.txt"), "content3"},
	}

	for _, check := range checks {
		content, err := os.ReadFile(check.path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", check.path, err)
			continue
		}
		if string(content) != check.content {
			t.Errorf("Unexpected content for %s: %s (expected %s)", check.path, string(content), check.content)
		}
	}
}

func TestGetDB(t *testing.T) {
	f := testutils.Setup(t)
	defer f.Cleanup()

	db := f.GetDB()
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}

	if _, err := os.Stat(f.DBPath); os.IsNotExist(err) {
		t.Errorf("DB file %s was not created", f.DBPath)
	}
}
