package testutils

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

type TestFixture struct {
	T       *testing.T
	DBPath  string
	TempDir string
}

func Setup(t *testing.T) *TestFixture {
	tempDir, err := os.MkdirTemp("", "disco-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Get absolute path for tempDir to avoid confusion
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(absTempDir, "test.db")

	return &TestFixture{
		T:       t,
		DBPath:  dbPath,
		TempDir: absTempDir,
	}
}

func (f *TestFixture) Cleanup() {
	os.RemoveAll(f.TempDir)
}

func (f *TestFixture) CreateDummyFile(name string) string {
	path := filepath.Join(f.TempDir, name)
	f.writeFile(path, []byte("dummy data"))
	return path
}

func (f *TestFixture) CreateFileTree(tree map[string]any) {
	f.createTree(f.TempDir, tree)
}

func (f *TestFixture) createTree(parent string, tree map[string]any) {
	for name, content := range tree {
		path := filepath.Join(parent, name)
		if subTree, ok := content.(map[string]any); ok {
			if err := os.MkdirAll(path, 0o755); err != nil {
				f.T.Fatal(err)
			}
			f.createTree(path, subTree)
		} else if strContent, ok := content.(string); ok {
			f.writeFile(path, []byte(strContent))
		} else {
			f.writeFile(path, []byte("dummy"))
		}
	}
}

func (f *TestFixture) writeFile(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.T.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		f.T.Fatal(err)
	}
}

func (f *TestFixture) GetDB() *sql.DB {
	db, err := sql.Open("sqlite3", f.DBPath)
	if err != nil {
		f.T.Fatal(err)
	}
	return db
}

// GetSchema returns the canonical database schema
func GetSchema() string {
	return db.GetSchema()
}

// InitTestDB initializes a test database with the canonical schema
func InitTestDB(_ testing.TB, sqlDB *sql.DB) error {
	schema := db.GetSchema()
	_, err := sqlDB.Exec(schema)
	return err
}

// StripFTSFromSchema returns the schema without FTS5-specific statements
func StripFTSFromSchema(schema string) string {
	var filtered strings.Builder
	lines := strings.Split(schema, ";")
	skipNextEnd := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "FTS5") || strings.Contains(upper, "_FTS") {
			if strings.Contains(upper, "BEGIN") && !strings.Contains(upper, "END") {
				skipNextEnd = true
			}
			continue
		}
		if skipNextEnd && upper == "END" {
			skipNextEnd = false
			continue
		}
		filtered.WriteString(trimmed)
		filtered.WriteString(";")
	}
	return filtered.String()
}

// InitTestDBNoFTS initializes a test database with the schema minus FTS5 features
func InitTestDBNoFTS(sqlDB *sql.DB) error {
	schema := StripFTSFromSchema(db.GetSchema())
	_, err := sqlDB.Exec(schema)
	return err
}

// InitTestDBWithDB initializes a test database with the canonical schema using provided DB connection
func InitTestDBWithDB(_ testing.TB, sqlDB *sql.DB) error {
	schema := db.GetSchema()
	_, err := sqlDB.Exec(schema)
	return err
}
