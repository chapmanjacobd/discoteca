package commands

import (
	"database/sql"
	"os"
	"testing"
)

func TestMergeDBsCmd_Run(t *testing.T) {
	t.Parallel()
	// 1. Setup Source 1
	src1, _ := os.CreateTemp("", "merge-src1-*.db")
	src1Path := src1.Name()
	src1.Close()
	defer os.Remove(src1Path)
	db1, _ := sql.Open("sqlite3", src1Path)
	db1.Exec("CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT)")
	db1.Exec("INSERT INTO media VALUES ('/path1', 'Title 1')")
	db1.Close()

	// 2. Setup Source 2
	src2, _ := os.CreateTemp("", "merge-src2-*.db")
	src2Path := src2.Name()
	src2.Close()
	defer os.Remove(src2Path)
	db2, _ := sql.Open("sqlite3", src2Path)
	db2.Exec("CREATE TABLE media (path TEXT PRIMARY KEY, title TEXT)")
	db2.Exec("INSERT INTO media VALUES ('/path2', 'Title 2')")
	db2.Close()

	// 3. Setup Target
	target, _ := os.CreateTemp("", "merge-target-*.db")
	targetPath := target.Name()
	target.Close()
	defer os.Remove(targetPath)

	// 4. Run Merge
	cmd := &MergeDBsCmd{
		TargetDB:  targetPath,
		SourceDBs: []string{src1Path, src2Path},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("MergeDBsCmd failed: %v", err)
	}

	// 5. Verify
	targetDB, _ := sql.Open("sqlite3", targetPath)
	defer targetDB.Close()
	var count int
	targetDB.QueryRow("SELECT COUNT(*) FROM media").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 rows in merged database, got %d", count)
	}
}
