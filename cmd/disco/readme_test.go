package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/db"
)

func newTestParser(t *testing.T) *kong.Kong {
	t.Helper()
	cli := &CLI{}
	parser, err := kong.New(cli, kong.Name("disco"))
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	return parser
}

func TestReadmeExamplesParse(t *testing.T) {
	dir := t.TempDir()
	videoDB := filepath.Join(dir, "my_videos.db")
	musicDB := filepath.Join(dir, "my_music.db")
	for _, path := range []string{videoDB, musicDB} {
		conn, err := sql.Open("sqlite3", path)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.InitDB(context.Background(), conn); err != nil {
			conn.Close()
			t.Fatalf("failed to initialize %s: %v", path, err)
		}
		conn.Close()
	}

	for _, examples := range commands.SubcommandExamples() {
		for _, example := range examples {
			example = strings.ReplaceAll(example, "my_videos.db", videoDB)
			example = strings.ReplaceAll(example, "my_music.db", musicDB)
			fields := strings.Fields(example)
			if len(fields) < 2 || fields[0] != "disco" {
				t.Fatalf("invalid example: %q", example)
			}

			parser := newTestParser(t)
			if _, err := parser.Parse(fields[1:]); err != nil {
				t.Errorf("example does not parse: %s: %v", example, err)
			}
		}
	}
}
