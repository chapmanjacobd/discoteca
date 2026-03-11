package utils

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

func TestTextProcessor(t *testing.T) {
	lines := []string{
		"cherry cherry cherry",
		"apple banana cherry",
		"banana apple apple",
	}

	t.Run("LineSortAlpha", func(t *testing.T) {
		flags := models.GlobalFlags{
			TextFlags: models.TextFlags{
				LineSorts: []string{"line"},
			},
		}
		got := TextProcessor(flags, lines)
		// apple banana cherry, banana apple apple, cherry cherry cherry
		if got[0] != "apple banana cherry" {
			t.Errorf("Expected apple... first, got %s", got[0])
		}
	})

	t.Run("WordSortAlpha", func(t *testing.T) {
		flags := models.GlobalFlags{
			TextFlags: models.TextFlags{
				WordSorts: []string{"alpha"},
				LineSorts: []string{"line"},
			},
		}
		// This should sort lines by original string, but word sorting happens internally.
		// Since we only get original lines back, it's hard to verify word sorting unless it affects line sorting.
		got := TextProcessor(flags, lines)
		if len(got) != 3 {
			t.Error("Lost lines")
		}
	})
}

func TestComparisonHelpers(t *testing.T) {
	if compareInt(1, 2) != -1 {
		t.Error("compareInt failed")
	}
	if compareFloat(1.5, 1.1) != 1 {
		t.Error("compareFloat failed")
	}
	if compareBool(true, false) != 1 {
		t.Error("compareBool failed")
	}
	if compareString("a", "b") != -1 {
		t.Error("compareString failed")
	}
}

func TestStatsHelpers(t *testing.T) {
	words := []string{"a", "b", "a"}
	stats := map[string]int{"a": 2, "b": 1}

	if got := sumDups(words, stats); got != 2 {
		t.Errorf("sumDups = %d, want 2", got)
	}
	if got := sumUnique(words, stats); got != 1 {
		t.Errorf("sumUnique = %d, want 1", got)
	}
	if got := sumCounts(words, stats); got != 5 {
		t.Errorf("sumCounts = %d, want 5", got)
	}
	if got := maxCount(words, stats); got != 2 {
		t.Errorf("maxCount = %d, want 2", got)
	}
	if got := minCount(words, stats); got != 1 {
		t.Errorf("minCount = %d, want 1", got)
	}
}

func TestFilterCorpus(t *testing.T) {
	corpusStats := map[string]int{
		"apple":  2,
		"banana": 1,
		"cherry": 3,
	}

	trueVal := true
	falseVal := false

	tests := []struct {
		words    []string
		unique   *bool
		dups     *bool
		expected bool
	}{
		{[]string{"banana"}, &trueVal, nil, true},
		{[]string{"apple"}, &trueVal, nil, false},
		{[]string{"apple"}, nil, &trueVal, true},
		{[]string{"banana"}, nil, &trueVal, false},
		{[]string{"apple", "banana"}, &trueVal, &falseVal, false}, // not all unique
		{[]string{"banana"}, &trueVal, &falseVal, true},           // all unique
	}

	for _, tt := range tests {
		if got := filterCorpus(corpusStats, tt.words, tt.unique, tt.dups); got != tt.expected {
			t.Errorf("filterCorpus(%v, %v, %v) = %v, want %v", tt.words, tt.unique, tt.dups, got, tt.expected)
		}
	}
}
