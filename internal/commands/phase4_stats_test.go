package commands_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func setupStatsDB(t *testing.T) *testutils.TestFixture {
	t.Helper()
	fixture := testutils.Setup(t)
	dbConn := fixture.GetDB()
	if err := testutils.InitTestDB(t, dbConn); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	if _, err := dbConn.Exec(`
		INSERT INTO media (path, media_type, size, duration, time_deleted, time_last_played, time_created, time_modified)
		VALUES
			('/video1.mp4', 'video', 100, 60, 0, 1700000000, 1600000000, 1600000000),
			('/video2.mp4', 'video', 200, 120, 0, 0, 1600000000, 1600000000),
			('/audio1.mp3', 'audio', 300, 180, 0, 1700000000, 1600000000, 1600000000)
	`); err != nil {
		dbConn.Close()
		t.Fatal(err)
	}
	dbConn.Close()
	return fixture
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := fn()
	w.Close()
	os.Stdout = oldStdout

	data, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(data), runErr
}
func TestStatsCmd_FiltersGeneralStats(t *testing.T) {
	fixture := setupStatsDB(t)
	defer fixture.Cleanup()

	cmd := &commands.StatsCmd{
		Databases:        []string{fixture.DBPath},
		MediaFilterFlags: models.MediaFilterFlags{VideoOnly: true},
		JSON:             true,
	}
	out, err := captureStdout(t, func() error { return cmd.Run(context.Background()) })
	if err != nil {
		t.Fatalf("StatsCmd failed: %v", err)
	}

	var payload struct {
		Summary struct {
			TotalCount int64 `json:"TotalCount"`
		} `json:"summary"`
		Breakdown []struct {
			MediaType struct {
				String string `json:"String"`
			} `json:"MediaType"`
			Count int64 `json:"Count"`
		} `json:"breakdown"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if payload.Summary.TotalCount != 2 {
		t.Errorf("filtered total = %d, want 2", payload.Summary.TotalCount)
	}
	if len(payload.Breakdown) != 1 {
		t.Errorf("expected one media type in the filtered breakdown, got %d", len(payload.Breakdown))
	}
}

func TestStatsCmd_FacetRequiresFrequency(t *testing.T) {
	fixture := setupStatsDB(t)
	defer fixture.Cleanup()

	cmd := &commands.StatsCmd{
		Facet:     "watched",
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected an error when --facet is used without --frequency")
	}
}

func TestStatsCmd_RejectsInvalidFacet(t *testing.T) {
	fixture := setupStatsDB(t)
	defer fixture.Cleanup()

	cmd := &commands.StatsCmd{
		Facet:     "not-a-facet",
		Frequency: "daily",
		Databases: []string{fixture.DBPath},
	}
	if err := cmd.Run(context.Background()); err == nil {
		t.Fatal("expected an error for an invalid facet")
	}
}

func TestStatsCmd_EachFacetWithFrequency(t *testing.T) {
	fixture := setupStatsDB(t)
	defer fixture.Cleanup()

	for _, facet := range []string{"watched", "deleted", "created", "modified"} {
		t.Run(facet, func(t *testing.T) {
			cmd := &commands.StatsCmd{
				Facet:     facet,
				Frequency: "monthly",
				Databases: []string{fixture.DBPath},
				JSON:      true,
			}
			if _, err := captureStdout(t, func() error { return cmd.Run(context.Background()) }); err != nil {
				t.Fatalf("facet %s failed: %v", facet, err)
			}
		})
	}
}
