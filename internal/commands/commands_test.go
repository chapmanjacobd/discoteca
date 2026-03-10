package commands

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discotheque/internal/db"
	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/testutils"
)

func TestAddCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")
	f2 := fixture.CreateDummyFile("audio1.mp3")

	cmd := &AddCmd{
		Args: []string{fixture.DBPath, f1, f2},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}

	if err := cmd.Run(nil); err != nil {
		t.Fatalf("AddCmd failed: %v", err)
	}

	// Verify items added
	dbConn := fixture.GetDB()
	defer dbConn.Close()

	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM media").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 items in database, got %d", count)
	}
}

func TestFilesInfoCmd_Run(t *testing.T) {
	f, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &FilesInfoCmd{
		DisplayFlags: models.DisplayFlags{JSON: true},
		Args:         []string{f.Name()},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("FilesInfoCmd failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "text") {
		t.Errorf("Expected output to contain text, got %s", output)
	}
}

func TestSearchDBCmd_Run(t *testing.T) {
	f, err := os.CreateTemp("", "sdb-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	defer os.Remove(dbPath)

	dbConn, _ := sql.Open("sqlite3", dbPath)
	dbConn.Exec("CREATE TABLE test (name TEXT, val TEXT)")
	dbConn.Exec("INSERT INTO test VALUES ('apple', 'fruit'), ('carrot', 'vegetable')")
	dbConn.Close()

	t.Run("FuzzyTableMatching", func(t *testing.T) {
		cmd := &SearchDBCmd{
			DisplayFlags: models.DisplayFlags{JSON: true},
			Database:     dbPath,
			Table:        "tes", // fuzzy match for 'test'
			Search:       []string{"apple"},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("SearchDBCmd failed: %v", err)
		}
	})

	t.Run("DeleteRows", func(t *testing.T) {
		cmd := &SearchDBCmd{
			PostActionFlags: models.PostActionFlags{
				DeleteRows: true,
			},
			Database: dbPath,
			Table:    "test",
			Search:   []string{"carrot"},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("SearchDBCmd failed: %v", err)
		}

		// Verify deletion
		dbConn, _ = sql.Open("sqlite3", dbPath)
		defer dbConn.Close()
		var count int
		dbConn.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 row left, got %d", count)
		}
	})

	t.Run("MarkDeletedRows", func(t *testing.T) {
		dbConn, _ := sql.Open("sqlite3", dbPath)
		dbConn.Exec("ALTER TABLE test ADD COLUMN time_deleted INTEGER")
		dbConn.Exec("INSERT INTO test (name, val) VALUES ('banana', 'fruit')")
		dbConn.Close()

		cmd := &SearchDBCmd{
			PostActionFlags: models.PostActionFlags{
				MarkDeleted: true,
			},
			Database: dbPath,
			Table:    "test",
			Search:   []string{"banana"},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("SearchDBCmd failed: %v", err)
		}

		dbConn, _ = sql.Open("sqlite3", dbPath)
		defer dbConn.Close()
		var timeDeleted sql.NullInt64
		dbConn.QueryRow("SELECT time_deleted FROM test WHERE name = 'banana'").Scan(&timeDeleted)
		if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
			t.Error("Expected row to be marked as deleted")
		}
	})
}

func TestMergeDBsCmd_Run(t *testing.T) {
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

func TestRegexSortCmd_Run(t *testing.T) {
	input := "red apple\nbroccoli\nyellow\ngreen\norange apple\nred apple\n"

	t.Run("DefaultSort", func(t *testing.T) {
		var out strings.Builder
		cmd := &RegexSortCmd{
			Reader: strings.NewReader(input),
			Writer: &out,
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("RegexSortCmd failed: %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "broccoli") {
			t.Errorf("Output missing expected content: %s", output)
		}
	})

	t.Run("LineSortDup", func(t *testing.T) {
		var out strings.Builder
		cmd := &RegexSortCmd{
			TextFlags: models.TextFlags{
				LineSorts: []string{"dup", "natural"},
			},
			Reader: strings.NewReader(input),
			Writer: &out,
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("RegexSortCmd failed: %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "red apple") {
			t.Errorf("Output missing expected content: %s", output)
		}
	})
}

func TestStatsCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	t.Run("DefaultStats", func(t *testing.T) {
		cmd := &StatsCmd{
			Facet:     "watched",
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("StatsCmd failed: %v", err)
		}
	})

	t.Run("JSONStats", func(t *testing.T) {
		cmd := &StatsCmd{
			DisplayFlags: models.DisplayFlags{JSON: true},
			Facet:        "watched",
			Databases:    []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("StatsCmd failed: %v", err)
		}
	})
}

func TestPrintCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	t.Run("PrintFromDB", func(t *testing.T) {
		cmd := &PrintCmd{
			Args: []string{fixture.DBPath},
		}
		cmd.AfterApply()
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}
	})

	t.Run("PrintFromFS", func(t *testing.T) {
		cmd := &PrintCmd{
			Args: []string{fixture.TempDir},
		}
		cmd.AfterApply()
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}
	})

	t.Run("PrintJSONAggregated", func(t *testing.T) {
		cmd := &PrintCmd{
			DisplayFlags:   models.DisplayFlags{JSON: true},
			AggregateFlags: models.AggregateFlags{BigDirs: true},
			Args:           []string{fixture.DBPath},
		}
		cmd.AfterApply()
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("PrintCmd failed: %v", err)
		}
	})
}

func TestHistoryCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	// Add history
	addHist := &HistoryAddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addHist.AfterApply()
	addHist.Run(nil)

	t.Run("DefaultHistory", func(t *testing.T) {
		cmd := &HistoryCmd{
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("HistoryCmd failed: %v", err)
		}
	})

	t.Run("DeleteHistory", func(t *testing.T) {
		cmd := &HistoryCmd{
			PostActionFlags: models.PostActionFlags{
				DeleteRows: true,
			},
			Databases: []string{fixture.DBPath},
		}
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("HistoryCmd failed: %v", err)
		}
	})
}

func TestOptimizeCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath},
	}
	addCmd.AfterApply() // Will fail if no paths, but we just want to init DB
	// Manually init DB
	dbConn := fixture.GetDB()
	db.InitDB(dbConn)
	dbConn.Close()

	cmd := &OptimizeCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("OptimizeCmd failed: %v", err)
	}
}

func TestCheckCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	// Delete file from FS
	os.Remove(f1)

	cmd := &CheckCmd{
		Args: []string{fixture.DBPath},
	}
	cmd.AfterApply()
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("CheckCmd failed: %v", err)
	}

	// Verify it was marked as deleted
	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	err := dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
		t.Errorf("Expected file to be marked as deleted")
	}
}

func TestBigDirsCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("dir1/media1.mp4")
	fixture.CreateDummyFile("dir2/media2.mp4")

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	cmd := &BigDirsCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("BigDirsCmd failed: %v", err)
	}
}

func TestDiskUsageCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	fixture.CreateDummyFile("dir1/media1.mp4")

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	t.Run("DefaultDU", func(t *testing.T) {
		cmd := &DiskUsageCmd{
			Args: []string{fixture.DBPath},
		}
		cmd.AfterApply()
		if err := cmd.Run(nil); err != nil {
			t.Fatalf("DiskUsageCmd failed: %v", err)
		}
	})
}

func TestPlaylistsCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, fixture.TempDir},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	cmd := &PlaylistsCmd{
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("PlaylistsCmd failed: %v", err)
	}
}

func TestMarkDeletedItem(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
		DB:    fixture.DBPath,
	}

	if err := MarkDeletedItem(m); err != nil {
		t.Fatalf("MarkDeletedItem failed: %v", err)
	}

	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted sql.NullInt64
	err := dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !timeDeleted.Valid || timeDeleted.Int64 == 0 {
		t.Errorf("Expected file to be marked as deleted")
	}
}

func TestMoveMediaItem(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	addCmd.AfterApply()
	addCmd.Run(nil)

	destDir := filepath.Join(fixture.TempDir, "moved")
	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
		DB:    fixture.DBPath,
	}

	if err := MoveMediaItem(destDir, m); err != nil {
		t.Fatalf("MoveMediaItem failed: %v", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(f1))
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("Expected file to exist at %s", destPath)
	}
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Errorf("Expected original file to be gone")
	}

	dbConn := fixture.GetDB()
	defer dbConn.Close()
	var count int
	dbConn.QueryRow("SELECT COUNT(*) FROM media WHERE path = ?", destPath).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 row with new path, got %d", count)
	}
}

func TestCopyMediaItem(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
	}

	destDir := filepath.Join(fixture.TempDir, "copied")
	if err := CopyMediaItem(destDir, m); err != nil {
		t.Fatalf("CopyMediaItem failed: %v", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(f1))
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("Expected file to exist at %s", destPath)
	}
	if _, err := os.Stat(f1); err != nil {
		t.Errorf("Expected original file to still exist")
	}
}

func TestSearchCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	addCmd := &AddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	// We need a title to search for
	addCmd.AfterApply()
	addCmd.Run(nil)

	// Manually set title so we can search it
	dbConn := fixture.GetDB()
	dbConn.Exec("UPDATE media SET title = 'Super Secret Movie' WHERE path = ?", f1)
	dbConn.Close()

	cmd := &SearchCmd{
		FilterFlags: models.FilterFlags{Search: []string{"Secret"}},
		Databases:   []string{fixture.DBPath},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("SearchCmd failed: %v", err)
	}
}

func TestDeleteMediaItem(t *testing.T) {
	f, err := os.CreateTemp("", "delete-test")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	m := models.MediaWithDB{
		Media: models.Media{Path: f.Name()},
	}

	if err := DeleteMediaItem(m); err != nil {
		t.Fatalf("DeleteMediaItem failed: %v", err)
	}

	if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
		t.Error("File still exists after DeleteMediaItem")
	}
}

func TestHistoryAddCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")

	cmd := &HistoryAddCmd{
		Args: []string{fixture.DBPath, f1},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}

	if err := cmd.Run(nil); err != nil {
		t.Fatalf("HistoryAddCmd failed: %v", err)
	}
}

func TestReadmeCmd_Run(t *testing.T) {
	parser, err := kong.New(&struct {
		Print PrintCmd `cmd:""`
	}{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := parser.Parse([]string{"print", "db.db"})
	if err != nil {
		t.Fatal(err)
	}

	cmd := &ReadmeCmd{}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("ReadmeCmd failed: %v", err)
	}
}

func TestSampleHashCmd_Run(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("video1.mp4")

	cmd := &SampleHashCmd{
		Paths: []string{f1},
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("SampleHashCmd failed: %v", err)
	}
}

func TestExecutePostAction(t *testing.T) {
	fixture := testutils.Setup(t)
	defer fixture.Cleanup()

	f1 := fixture.CreateDummyFile("media1.mp4")
	m := models.MediaWithDB{
		Media: models.Media{Path: f1},
		DB:    fixture.DBPath,
	}

	// Test mark-deleted
	flags := models.GlobalFlags{
		PostActionFlags: models.PostActionFlags{
			PostAction: "mark-deleted",
		},
	}
	// Manually init DB
	dbConn := fixture.GetDB()
	db.InitDB(dbConn)
	dbConn.Exec("INSERT INTO media (path) VALUES (?)", f1)
	dbConn.Close()

	if err := ExecutePostAction(flags, []models.MediaWithDB{m}); err != nil {
		t.Fatalf("ExecutePostAction mark-deleted failed: %v", err)
	}

	dbConn = fixture.GetDB()
	defer dbConn.Close()
	var timeDeleted int64
	dbConn.QueryRow("SELECT time_deleted FROM media WHERE path = ?", f1).Scan(&timeDeleted)
	if timeDeleted == 0 {
		t.Error("Expected item to be marked as deleted")
	}
}
