package commands_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

func regexSortOutput(t *testing.T, cmd *commands.RegexSortCmd) string {
	t.Helper()
	var out strings.Builder
	cmd.Writer = &out
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("RegexSortCmd failed: %v", err)
	}
	return out.String()
}

func TestRegexSortCmd_DuplicateLinesArePreserved(t *testing.T) {
	cmd := &commands.RegexSortCmd{
		TextProcessingFlags: models.TextProcessingFlags{Preprocess: true},
		Reader:              strings.NewReader("red apple\nred apple\n"),
	}
	output := regexSortOutput(t, cmd)
	if strings.Count(output, "red apple") != 2 {
		t.Errorf("duplicate normalized lines were dropped: %q", output)
	}
}

func TestRegexSortCmd_NoInput(t *testing.T) {
	cmd := &commands.RegexSortCmd{Reader: strings.NewReader("")}
	if output := regexSortOutput(t, cmd); output != "" {
		t.Errorf("expected no output, got %q", output)
	}
}

func TestRegexSortCmd_WithoutPreprocess(t *testing.T) {
	cmd := &commands.RegexSortCmd{Reader: strings.NewReader("zeta.mp4\nalpha.mp4\n")}
	output := regexSortOutput(t, cmd)
	if !strings.Contains(output, "alpha.mp4") || !strings.Contains(output, "zeta.mp4") {
		t.Errorf("missing lines: %q", output)
	}
}

func TestRegexSortCmd_SortStrategies(t *testing.T) {
	strategies := []string{
		"line", "count", "len", "dup", "unique", "sum", "dupmax", "dupmin",
		"dupavg", "dupmedian", "allunique", "alldup", "dupmode", "alpha", "natural",
	}
	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			cmd := &commands.RegexSortCmd{
				TextProcessingFlags: models.TextProcessingFlags{LineSorts: []string{strategy}},
				Reader:              strings.NewReader("red apple\nred apple\nbroccoli\nyellow fruit\n"),
			}
			output := regexSortOutput(t, cmd)
			if strings.Count(output, "\n") != 4 {
				t.Errorf("strategy %s dropped lines: %q", strategy, output)
			}
		})
	}
}

func TestRegexSortCmd_WordSortStrategiesKeepLines(t *testing.T) {
	for _, strategy := range []string{"index", "lastindex", "count", "dup", "unique", "linecount"} {
		t.Run(strategy, func(t *testing.T) {
			cmd := &commands.RegexSortCmd{
				TextProcessingFlags: models.TextProcessingFlags{WordSorts: []string{strategy}},
				Reader:              strings.NewReader("gamma beta gamma\nalpha beta\n"),
			}
			output := regexSortOutput(t, cmd)
			if strings.Count(output, "\n") != 2 {
				t.Errorf("word sort %s dropped lines: %q", strategy, output)
			}
		})
	}
}
