package commands_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestRegexSortCmd_Run(t *testing.T) {
	input := "red apple\nbroccoli\nyellow\ngreen\norange apple\nred apple\n"

	t.Run("DefaultSort", func(t *testing.T) {
		var out strings.Builder
		cmd := &commands.RegexSortCmd{
			Reader: strings.NewReader(input),
			Writer: &out,
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.RegexSortCmd failed: %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "broccoli") {
			t.Errorf("Output missing expected content: %s", output)
		}
	})

	t.Run("LineSortDup", func(t *testing.T) {
		var out strings.Builder
		cmd := &commands.RegexSortCmd{
			TextFlags: models.TextFlags{
				LineSorts: []string{"dup", "natural"},
			},
			Reader: strings.NewReader(input),
			Writer: &out,
		}
		if err := cmd.Run(context.Background()); err != nil {
			t.Fatalf("commands.RegexSortCmd failed: %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "red apple") {
			t.Errorf("Output missing expected content: %s", output)
		}
	})
}
