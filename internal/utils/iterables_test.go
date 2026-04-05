package utils_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestFlatten(t *testing.T) {
	tests := []struct {
		input    any
		expected []any
	}{
		{[]any{[]any{[]any{[]any{1}}}}, []any{1}},
		{[]any{[]any{[]any{1}}, []any{2}}, []any{1, 2}},
		{[]any{[]any{[]any{"test"}}}, []any{"test"}},
		{"", nil},
		{[]any{""}, nil},
		{"abc", []any{"abc"}},
	}

	for _, tt := range tests {
		got := utils.Flatten(tt.input)
		if len(got) == 0 && len(tt.expected) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("utils.Flatten(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestConform(t *testing.T) {
	tests := []struct {
		input    any
		expected []int
	}{
		{[]int{1, 2, 3}, []int{1, 2, 3}},
		{1, []int{1}},
	}

	for _, tt := range tests {
		got := utils.Conform[int](tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("utils.Conform(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestSafeUnpack(t *testing.T) {
	if got := utils.SafeUnpack(1, 2, 3); got != 1 {
		t.Errorf("utils.SafeUnpack(1, 2, 3) = %v, want 1", got)
	}
	if got := utils.SafeUnpack(0, 0, 1); got != 1 {
		t.Errorf("utils.SafeUnpack(0, 0, 1) = %v, want 1", got)
	}
	if got := utils.SafeUnpack[int](); got != 0 {
		t.Errorf("utils.SafeUnpack() = %v, want 0", got)
	}
}

func TestSafeLen(t *testing.T) {
	if got := utils.SafeLen([]int{1, 2, 3}); got != 3 {
		t.Errorf("utils.SafeLen() = %d, want 3", got)
	}
	if got := utils.SafeLen[int](nil); got != 0 {
		t.Errorf("utils.SafeLen(nil) = %d, want 0", got)
	}
}

func TestSafeSum(t *testing.T) {
	if got := utils.SafeSum([]int{1, 2, 3}); got != 6 {
		t.Errorf("utils.SafeSum() = %d, want 6", got)
	}
}

func TestSafePop(t *testing.T) {
	slice := []int{1, 2, 3}
	val, newSlice := utils.SafePop(slice)
	if val != 3 || len(newSlice) != 2 {
		t.Errorf("utils.SafePop() = %v, %v, want 3, [1, 2]", val, newSlice)
	}
}

func TestSafeIndex(t *testing.T) {
	slice := []int{1, 2, 3}
	if got := utils.SafeIndex(slice, 2); got != 1 {
		t.Errorf("utils.SafeIndex(2) = %d, want 1", got)
	}
	if got := utils.SafeIndex(slice, 4); got != -1 {
		t.Errorf("utils.SafeIndex(4) = %d, want -1", got)
	}
}

func TestListDictFilterBool(t *testing.T) {
	data := []map[string]any{
		{"t": 1},
		{"t": nil},
		{},
	}
	got := utils.ListDictFilterBool(data)
	if len(got) != 1 {
		t.Errorf("utils.ListDictFilterBool() len = %d, want 1", len(got))
	}
}

func TestChunks(t *testing.T) {
	slice := []int{1, 2, 3}
	got := utils.Chunks(slice, 2)
	expected := [][]int{{1, 2}, {3}}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.Chunks() = %v, want %v", got, expected)
	}
}

func TestSimilarity(t *testing.T) {
	if got := utils.Similarity([]int{1, 2, 3}, []int{1, 2, 3}); got != 1.0 {
		t.Errorf("utils.Similarity same = %v, want 1.0", got)
	}
	if got := utils.Similarity([]int{1, 2, 3}, []int{4, 5, 6}); got != 0.0 {
		t.Errorf("utils.Similarity diff = %v, want 0.0", got)
	}
	if got := utils.Similarity([]int{1, 2}, []int{2, 3}); got != 1.0/3.0 {
		t.Errorf("utils.Similarity partial = %v, want 0.333...", got)
	}
}

func TestValueCounts(t *testing.T) {
	input := []int{1, 2, 2, 3, 1, 1}
	expected := []int{3, 2, 2, 1, 3, 3}
	got := utils.ValueCounts(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.ValueCounts() = %v, want %v", got, expected)
	}
}

func TestDivisors(t *testing.T) {
	tests := []struct {
		input    int
		expected []int
	}{
		{4, []int{2}},
		{6, []int{2, 3}},
		{8, []int{2, 4}},
		{9, []int{3}},
		{10, []int{2, 5}},
	}

	for _, tt := range tests {
		got := utils.Divisors(tt.input)
		// We need to sort to compare if we want exact equality, or just check elements
		// Original Python yields them in order but my Go version might not depending on how I implement it.
		// Actually my Go version appends i and then n/i, so for 6 it's [2, 3]. For 8 it's [2, 4].
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("utils.Divisors(%d) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestDivideSequence(t *testing.T) {
	if got := utils.DivideSequence([]int{100, 2, 5}); got != 10.0 {
		t.Errorf("utils.DivideSequence(100, 2, 5) = %v, want 10.0", got)
	}
	if got := utils.DivideSequence([]int{10, 0}); !math.IsInf(got, 1) {
		t.Errorf("utils.DivideSequence(10, 0) = %v, want +inf", got)
	}
	if got := utils.DivideSequence[int](nil); got != 0 {
		t.Errorf("utils.DivideSequence(nil) = %v, want 0", got)
	}
}

func TestUnique(t *testing.T) {
	input := []int{1, 2, 2, 3, 1}
	expected := []int{1, 2, 3}
	got := utils.Unique(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.Unique() = %v, want %v", got, expected)
	}
}

func TestConcat(t *testing.T) {
	s1 := []int{1, 2}
	s2 := []int{3, 4}
	expected := []int{1, 2, 3, 4}
	got := utils.Concat(s1, s2)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.Concat() = %v, want %v", got, expected)
	}
}

func TestOrderedSet(t *testing.T) {
	input := []int{1, 2, 2, 3, 1}
	expected := []int{1, 2, 3}
	got := utils.OrderedSet(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.OrderedSet() = %v, want %v", got, expected)
	}
}
