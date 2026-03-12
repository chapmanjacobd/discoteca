package commands

import (
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestRegexSortCmd_Run(t *testing.T) {
	t.Parallel()
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
