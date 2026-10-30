package commands_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func setupHistoryDB(t *testing.T) *testutils.TestFixture {
	t.Helper()
	fixture := testutils.Setup(t)
	dbConn := fixture.GetDB()
	if err := testutils.InitTestDB(t, dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	if _, err := dbConn.Exec(`
		INSERT INTO media (path, time_last_played, time_first_played, duration, playhead)
		VALUES
			('/a.mp4', 1700000003, 1700000000, 100, 10),
			('/b.mp4', 1700000002, 1700000000, 100, 50),
			('/c.mp4', 1700000001, 1700000000, 100, 90)
	`); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	dbConn.Close()
	return fixture
}

func runHistoryJSON(t *testing.T, cmd *commands.HistoryCmd) []map[string]any {
	t.Helper()
	cmd.DisplayFlags.JSON = true
	out, err := captureStdout(t, func() error { return cmd.Run(context.Background()) })
	if err != nil {
		t.Fatalf("HistoryCmd failed: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	return rows
}

func TestHistoryCmd_LimitOffsetAll(t *testing.T) {
	fixture := setupHistoryDB(t)
	defer fixture.Cleanup()

	t.Run("limit", func(t *testing.T) {
		rows := runHistoryJSON(t, &commands.HistoryCmd{
			Databases:  []string{fixture.DBPath},
			QueryFlags: models.QueryFlags{Limit: 1},
		})
		if len(rows) != 1 {
			t.Errorf("--limit 1 returned %d rows", len(rows))
		}
	})

	t.Run("offset", func(t *testing.T) {
		all := runHistoryJSON(t, &commands.HistoryCmd{
			Databases:  []string{fixture.DBPath},
			QueryFlags: models.QueryFlags{All: true},
		})
		offset := runHistoryJSON(t, &commands.HistoryCmd{
			Databases:  []string{fixture.DBPath},
			QueryFlags: models.QueryFlags{Limit: 1, Offset: 1},
		})
		if len(offset) != 1 {
			t.Fatalf("--offset returned %d rows", len(offset))
		}
		if len(all) != 3 {
			t.Fatalf("--all returned %d rows", len(all))
		}
		if offset[0]["path"] == all[0]["path"] {
			t.Errorf("--offset did not skip the first row")
		}
	})
}

func TestHistoryCmd_ExplicitPathSortIsPreserved(t *testing.T) {
	fixture := setupHistoryDB(t)
	defer fixture.Cleanup()

	cmd := &commands.HistoryCmd{
		Databases: []string{fixture.DBPath},
		SortFlags: models.SortFlags{SortBy: "path"},
	}
	if err := cmd.AfterApply(); err != nil {
		t.Fatal(err)
	}
	rows := runHistoryJSON(t, cmd)
	if len(rows) != 3 || rows[0]["path"] != "/a.mp4" {
		t.Errorf("explicit --sort path was overwritten: %v", rows)
	}
}

func TestHistoryCmd_DefaultOrderIsMostRecent(t *testing.T) {
	fixture := setupHistoryDB(t)
	defer fixture.Cleanup()

	rows := runHistoryJSON(t, &commands.HistoryCmd{
		Databases: []string{fixture.DBPath},
	})
	if len(rows) != 3 || rows[0]["path"] != "/a.mp4" {
		t.Errorf("default history order is not most-recent-first: %v", rows)
	}
}
