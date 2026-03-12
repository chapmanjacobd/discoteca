package commands

import (
	"testing"

	"github.com/alecthomas/kong"
)

func TestReadmeCmd_Run(t *testing.T) {
	t.Parallel()
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
