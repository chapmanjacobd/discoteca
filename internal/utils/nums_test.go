package utils_test

import (
	"errors"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestHumanToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"100", 100},
		{"1KB", 1000},
		{"1MB", 1000 * 1000},
		{"1GB", 1000 * 1000 * 1000},
		{"1.5MB", 1500000},
		{" 100 MB ", 100 * 1000 * 1000},
	}

	for _, tt := range tests {
		result, err := utils.HumanToBytes(tt.input)
		if err != nil {
			t.Errorf("utils.HumanToBytes(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("utils.HumanToBytes(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestHumanToSeconds(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1 hour", 3600},
		{"30 min", 1800},
		{"45s", 45},
		{"100", 100},
		{"1 day", 86400},
		{"1 week", 604800},
	}
	for _, tt := range tests {
		result, err := utils.HumanToSeconds(tt.input)
		if err != nil {
			t.Errorf("utils.HumanToSeconds(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("utils.HumanToSeconds(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseRange(t *testing.T) {
	mockHumanToX := func(s string) (int64, error) {
		if s == "100" {
			return 100, nil
		}
		return 0, errors.New("invalid")
	}

	tests := []struct {
		input string
		check func(utils.Range) bool
	}{
		{">100", func(r utils.Range) bool { return r.Min != nil && *r.Min == 101 }},
		{"+100", func(r utils.Range) bool { return r.Min != nil && *r.Min == 100 }},
		{"<100", func(r utils.Range) bool { return r.Max != nil && *r.Max == 99 }},
		{"-100", func(r utils.Range) bool { return r.Max != nil && *r.Max == 100 }},
		{"100%10", func(r utils.Range) bool { return r.Min != nil && *r.Min == 90 && r.Max != nil && *r.Max == 110 }},
	}

	for _, tt := range tests {
		r, err := utils.ParseRange(tt.input, mockHumanToX)
		if err != nil {
			t.Errorf("utils.ParseRange(%q) error: %v", tt.input, err)
			continue
		}
		if !tt.check(r) {
			t.Errorf("utils.ParseRange(%q) failed check: %+v", tt.input, r)
		}
	}
}

func TestCalculatePercentiles(t *testing.T) {
	values := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	res := utils.CalculatePercentiles(values)
	if len(res) != 101 {
		t.Errorf("utils.CalculatePercentiles len = %d, want 101", len(res))
	}
	if res[0] != 10 {
		t.Errorf("res[0] = %d, want 10", res[0])
	}
	if res[100] != 100 {
		t.Errorf("res[100] = %d, want 100", res[100])
	}

	// Test empty
	resEmpty := utils.CalculatePercentiles(nil)
	if len(resEmpty) != 101 {
		t.Errorf("utils.CalculatePercentiles empty len = %d, want 101", len(resEmpty))
	}
}

func TestPercent(t *testing.T) {
	if got := utils.Percent(50, 200); got != 25.0 {
		t.Errorf("utils.Percent(50, 200) = %v, want 25.0", got)
	}
}

func TestFloatFromPercent(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"50%", 0.5},
		{"0.5", 0.5},
	}
	for _, tt := range tests {
		got, _ := utils.FloatFromPercent(tt.input)
		if got != tt.expected {
			t.Errorf("utils.FloatFromPercent(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestRandomFloat(t *testing.T) {
	got := utils.RandomFloat()
	if got < 0 || got > 1 {
		t.Errorf("utils.RandomFloat() = %v, want in [0, 1)", got)
	}
}

func TestRandomInt(t *testing.T) {
	got := utils.RandomInt(5, 10)
	if got < 5 || got >= 10 {
		t.Errorf("utils.RandomInt(5, 10) = %v, want in [5, 10)", got)
	}
	if got2 := utils.RandomInt(10, 5); got2 != 10 {
		t.Errorf("utils.RandomInt(10, 5) = %v, want 10", got2)
	}
}

func TestLinearInterpolation(t *testing.T) {
	data := [][2]float64{{0, 0}, {10, 100}}
	tests := []struct {
		x    float64
		want float64
	}{
		{-1, 0},
		{0, 0},
		{5, 50},
		{10, 100},
		{11, 100},
	}
	for _, tt := range tests {
		if got := utils.LinearInterpolation(tt.x, data); got != tt.want {
			t.Errorf("utils.LinearInterpolation(%v) = %v, want %v", tt.x, got, tt.want)
		}
	}
	if got := utils.LinearInterpolation(5, nil); got != 0 {
		t.Errorf("utils.LinearInterpolation with nil data = %v, want 0", got)
	}
}

func TestSafeMean(t *testing.T) {
	if got := utils.SafeMean([]int{1, 2, 3}); got != 2.0 {
		t.Errorf("utils.SafeMean([1, 2, 3]) = %v, want 2.0", got)
	}
	if got := utils.SafeMean([]float64{}); got != 0 {
		t.Errorf("utils.SafeMean([]) = %v, want 0", got)
	}
}

func TestSafeMedian(t *testing.T) {
	if got := utils.SafeMedian([]int{1, 3, 2}); got != 2.0 {
		t.Errorf("utils.SafeMedian([1, 3, 2]) = %v, want 2.0", got)
	}
	if got := utils.SafeMedian([]int{1, 2, 3, 4}); got != 2.5 {
		t.Errorf("utils.SafeMedian([1, 2, 3, 4]) = %v, want 2.5", got)
	}
	if got := utils.SafeMedian([]float64{}); got != 0 {
		t.Errorf("utils.SafeMedian([]) = %v, want 0", got)
	}
}

func TestHumanToBits(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1000", 1000},
		{"1KBIT", 1000},
		{"1M", 1000 * 1000},
		{"1.5GBIT", 1500000000},
	}
	for _, tt := range tests {
		got, err := utils.HumanToBits(tt.input)
		if err != nil {
			t.Errorf("utils.HumanToBits(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("utils.HumanToBits(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestPercentageDifference(t *testing.T) {
	if got := utils.PercentageDifference(10, 10); got != 0 {
		t.Errorf("utils.PercentageDifference(10, 10) = %v, want 0", got)
	}
	if got := utils.PercentageDifference(0, 0); got != 100.0 {
		t.Errorf("utils.PercentageDifference(0, 0) = %v, want 100.0", got)
	}
}

func TestCalculateSegments(t *testing.T) {
	got := utils.CalculateSegments(100, 10, 0.1)
	if len(got) == 0 {
		t.Errorf("utils.CalculateSegments(100, 10, 0.1) returned nil")
	}
	if got2 := utils.CalculateSegments(20, 10, 0.1); len(got2) != 1 || got2[0] != 0 {
		t.Errorf("utils.CalculateSegments(20, 10, 0.1) = %v, want [0]", got2)
	}
	if got3 := utils.CalculateSegments(0, 10, 0.1); got3 != nil {
		t.Errorf("utils.CalculateSegments(0, 10, 0.1) = %v, want nil", got3)
	}
}

func TestCalculateSegmentsInt(t *testing.T) {
	got := utils.CalculateSegmentsInt(100, 10, 5)
	if len(got) == 0 {
		t.Errorf("utils.CalculateSegmentsInt(100, 10, 5) returned nil")
	}
	if got2 := utils.CalculateSegmentsInt(20, 10, 0.1); len(got2) != 1 || got2[0] != 0 {
		t.Errorf("utils.CalculateSegmentsInt(20, 10, 0.1) = %v, want [0]", got2)
	}
	if got3 := utils.CalculateSegmentsInt(0, 10, 0.1); got3 != nil {
		t.Errorf("utils.CalculateSegmentsInt(0, 10, 0.1) = %v, want nil", got3)
	}
}

func TestSafeIntFloat(t *testing.T) {
	if got := utils.SafeInt("123"); got == nil || *got != 123 {
		t.Errorf("utils.SafeInt(123) = %v, want 123", got)
	}
	if got := utils.SafeInt(""); got != nil {
		t.Errorf("utils.SafeInt('') = %v, want nil", got)
	}
	if got := utils.SafeInt("abc"); got != nil {
		t.Errorf("utils.SafeInt('abc') = %v, want nil", got)
	}

	if got := utils.SafeFloat("123.45"); got == nil || *got != 123.45 {
		t.Errorf("utils.SafeFloat(123.45) = %v, want 123.45", got)
	}
	if got := utils.SafeFloat(""); got != nil {
		t.Errorf("utils.SafeFloat('') = %v, want nil", got)
	}
}

func TestSQLHumanTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10", "10 minutes"},
		{"10min", "10 minutes"},
		{"10s", "10 seconds"},
		{"other", "other"},
	}
	for _, tt := range tests {
		if got := utils.SQLHumanTime(tt.input); got != tt.expected {
			t.Errorf("utils.SQLHumanTime(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMaxMin(t *testing.T) {
	if got := utils.Max(1, 2); got != 2 {
		t.Errorf("utils.Max(1, 2) = %v, want 2", got)
	}
	if got := utils.Max(2, 1); got != 2 {
		t.Errorf("utils.Max(2, 1) = %v, want 2", got)
	}
	if got := utils.Min(1, 2); got != 1 {
		t.Errorf("utils.Min(1, 2) = %v, want 1", got)
	}
	if got := utils.Min(2, 1); got != 1 {
		t.Errorf("utils.Min(2, 1) = %v, want 1", got)
	}
}
