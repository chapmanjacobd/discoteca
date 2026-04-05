package utils_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
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
		got := utils.TextProcessor(flags, lines)
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
		got := utils.TextProcessor(flags, lines)
		if len(got) != 3 {
			t.Error("Lost lines")
		}
	})
}

func TestComparisonHelpers(t *testing.T) {
	if utils.CompareInt(1, 2) != -1 {
		t.Error("utils.CompareInt failed")
	}
	if utils.CompareFloat(1.5, 1.1) != 1 {
		t.Error("utils.CompareFloat failed")
	}
	if utils.CompareBool(true, false) != 1 {
		t.Error("utils.CompareBool failed")
	}
	if utils.CompareString("a", "b") != -1 {
		t.Error("utils.CompareString failed")
	}
}

func TestStatsHelpers(t *testing.T) {
	words := []string{"a", "b", "a"}
	stats := map[string]int{"a": 2, "b": 1}

	if got := utils.SumDups(words, stats); got != 2 {
		t.Errorf("utils.SumDups = %d, want 2", got)
	}
	if got := utils.SumUnique(words, stats); got != 1 {
		t.Errorf("utils.SumUnique = %d, want 1", got)
	}
	if got := utils.SumCounts(words, stats); got != 5 {
		t.Errorf("utils.SumCounts = %d, want 5", got)
	}
	if got := utils.MaxCount(words, stats); got != 2 {
		t.Errorf("utils.MaxCount = %d, want 2", got)
	}
	if got := utils.MinCount(words, stats); got != 1 {
		t.Errorf("utils.MinCount = %d, want 1", got)
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
		if got := utils.FilterCorpus(corpusStats, tt.words, tt.unique, tt.dups); got != tt.expected {
			t.Errorf("utils.FilterCorpus(%v, %v, %v) = %v, want %v", tt.words, tt.unique, tt.dups, got, tt.expected)
		}
	}
}
