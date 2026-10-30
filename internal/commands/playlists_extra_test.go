package commands_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/testutils"
)

func TestPlaylistsCmd_JSONIncludesEveryDatabase(t *testing.T) {
	first := testutils.Setup(t)
	defer first.Cleanup()
	second := testutils.Setup(t)
	defer second.Cleanup()

	for i, fixture := range []*testutils.TestFixture{first, second} {
		dbConn := fixture.GetDB()
		if err := testutils.InitTestDB(t, dbConn); err != nil {
			dbConn.Close()
			t.Fatal(err)
		}
		path := "/playlist-" + string(rune('a'+i))
		//nolint:gosec // test-only SQL literal
		if _, err := dbConn.Exec("INSERT INTO playlists (path) VALUES ('" + path + "')"); err != nil {
			dbConn.Close()
			t.Fatal(err)
		}
		dbConn.Close()
	}

	cmd := &commands.PlaylistsCmd{
		Databases: []string{first.DBPath, second.DBPath},
		JSON:      true,
	}
	out, err := captureStdout(t, func() error { return cmd.Run(context.Background()) })
	if err != nil {
		t.Fatalf("PlaylistsCmd failed: %v", err)
	}

	var entries []commands.PlaylistEntry
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(entries) != 2 {
		t.Fatalf("expected playlists from both databases, got %d", len(entries))
	}
	if entries[0].Database == entries[1].Database {
		t.Error("JSON output did not identify both databases")
	}
}
