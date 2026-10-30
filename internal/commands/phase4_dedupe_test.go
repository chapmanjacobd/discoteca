package commands_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

type dedupeRow struct {
	path        string
	title       string
	mediaType   string
	timeDeleted int64
}

func setupDedupeFilterDB(t *testing.T, rows []dedupeRow) *testutils.TestFixture {
	t.Helper()

	fixture := testutils.Setup(t)
	dbConn := fixture.GetDB()
	if err := testutils.InitTestDB(t, dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}

	for _, row := range rows {
		if err := os.MkdirAll(filepath.Dir(row.path), 0o755); err != nil {
			dbConn.Close()
			t.Fatal(err)
		}
		if err := os.WriteFile(row.path, []byte("data"), 0o644); err != nil {
			dbConn.Close()
			t.Fatal(err)
		}
		if _, err := dbConn.Exec(
			"INSERT INTO media (path, title, media_type, size, time_deleted) VALUES (?, ?, ?, 4, ?)",
			row.path,
			row.title,
			row.mediaType,
			row.timeDeleted,
		); err != nil {
			dbConn.Close()
			t.Fatal(err)
		}
	}
	dbConn.Close()
	return fixture
}

func countExistingFiles(t *testing.T, paths ...string) int {
	t.Helper()
	count := 0
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			count++
		}
	}
	return count
}

func TestDedupeCmd_MediaTypeFilter(t *testing.T) {
	dir := t.TempDir()
	v1 := filepath.Join(dir, "video1.mp4")
	v2 := filepath.Join(dir, "video2.mp4")
	a1 := filepath.Join(dir, "audio1.mp3")
	a2 := filepath.Join(dir, "audio2.mp3")

	fixture := setupDedupeFilterDB(t, []dedupeRow{
		{v1, "Video Title", "video", 0},
		{v2, "Video Title", "video", 0},
		{a1, "Audio Title", "audio", 0},
		{a2, "Audio Title", "audio", 0},
	})
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases:        []string{fixture.DBPath},
		CoreFlags:        models.CoreFlags{NoConfirm: true},
		MediaFilterFlags: models.MediaFilterFlags{VideoOnly: true},
		DedupeFlags:      models.DedupeFlags{TitleOnly: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("DedupeCmd failed: %v", err)
	}

	if got := countExistingFiles(t, v1, v2); got != 1 {
		t.Errorf("expected one video file to remain, got %d", got)
	}
	if got := countExistingFiles(t, a1, a2); got != 2 {
		t.Errorf("audio files outside the filter were changed; %d remain", got)
	}
}

func TestDedupeCmd_ExcludedPath(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep", "movie.mp4")
	excluded := filepath.Join(dir, "excluded", "movie.mp4")

	fixture := setupDedupeFilterDB(t, []dedupeRow{
		{keep, "Same", "video", 0},
		{excluded, "Same", "video", 0},
	})
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases: []string{fixture.DBPath},
		CoreFlags: models.CoreFlags{NoConfirm: true},
		PathFilterFlags: models.PathFilterFlags{
			Exclude: []string{filepath.Join(dir, "excluded")},
		},
		DedupeFlags: models.DedupeFlags{TitleOnly: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("DedupeCmd failed: %v", err)
	}

	if got := countExistingFiles(t, keep, excluded); got != 2 {
		t.Errorf("excluded row was deduped; %d of 2 files remain", got)
	}
}

func TestDedupeCmd_TimeFilter(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp4")
	b := filepath.Join(dir, "b.mp4")

	fixture := setupDedupeFilterDB(t, []dedupeRow{
		{a, "Same", "video", 0},
		{b, "Same", "video", 0},
	})
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases: []string{fixture.DBPath},
		CoreFlags: models.CoreFlags{NoConfirm: true},
		TimeFilterFlags: models.TimeFilterFlags{
			ModifiedAfter: "2999-01-01",
		},
		DedupeFlags: models.DedupeFlags{TitleOnly: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("DedupeCmd failed: %v", err)
	}

	if got := countExistingFiles(t, a, b); got != 2 {
		t.Errorf("time filter was ignored; %d of 2 files remain", got)
	}
}

func TestDedupeCmd_OnlyDeleted(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp4")
	b := filepath.Join(dir, "b.mp4")

	fixture := setupDedupeFilterDB(t, []dedupeRow{
		{a, "Same", "video", 1},
		{b, "Same", "video", 1},
	})
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases:    []string{fixture.DBPath},
		CoreFlags:    models.CoreFlags{NoConfirm: true},
		DeletedFlags: models.DeletedFlags{OnlyDeleted: true},
		DedupeFlags:  models.DedupeFlags{TitleOnly: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("DedupeCmd failed: %v", err)
	}

	if got := countExistingFiles(t, a, b); got != 1 {
		t.Errorf("--only-deleted dedupe did not process the deleted pair; %d files remain", got)
	}

	dbConn, err := sql.Open("sqlite3", fixture.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()
	var deduped int
	if err := dbConn.QueryRow("SELECT COUNT(*) FROM media WHERE is_deduped = 1").Scan(&deduped); err != nil {
		t.Fatal(err)
	}
	if deduped != 1 {
		t.Errorf("expected one keep candidate to be marked deduped, got %d", deduped)
	}
}

func TestDedupeCmd_FilterWithArgumentsStillFindsCandidates(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp4")
	b := filepath.Join(dir, "b.mp4")

	fixture := setupDedupeFilterDB(t, []dedupeRow{
		{a, "Same", "video", 0},
		{b, "Same", "video", 0},
	})
	defer fixture.Cleanup()

	cmd := &commands.DedupeCmd{
		Databases:   []string{fixture.DBPath},
		CoreFlags:   models.CoreFlags{NoConfirm: true},
		FilterFlags: models.FilterFlags{Size: []string{">1B"}},
		DedupeFlags: models.DedupeFlags{TitleOnly: true},
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("DedupeCmd failed: %v", err)
	}

	if got := countExistingFiles(t, a, b); got != 1 {
		t.Errorf("size filter with SQL arguments matched no candidates; %d files remain", got)
	}
}
