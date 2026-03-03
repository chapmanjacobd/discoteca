package utils

import (
	"fmt"
	"testing"
)

func TestHumanToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"100", 100},
		{"1KB", 1024},
		{"1MB", 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"1.5MB", 1572864},
		{" 100 MB ", 100 * 1024 * 1024},
	}

	for _, tt := range tests {
		result, err := HumanToBytes(tt.input)
		if err != nil {
			t.Errorf("HumanToBytes(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("HumanToBytes(%q) = %d, want %d", tt.input, result, tt.expected)
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
		result, err := HumanToSeconds(tt.input)
		if err != nil {
			t.Errorf("HumanToSeconds(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("HumanToSeconds(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseRange(t *testing.T) {
	mockHumanToX := func(s string) (int64, error) {
		if s == "100" {
			return 100, nil
		}
		return 0, fmt.Errorf("invalid")
	}

	tests := []struct {
		input string
		check func(Range) bool
	}{
		{">100", func(r Range) bool { return r.Min != nil && *r.Min == 101 }},
		{"+100", func(r Range) bool { return r.Min != nil && *r.Min == 100 }},
		{"<100", func(r Range) bool { return r.Max != nil && *r.Max == 99 }},
		{"-100", func(r Range) bool { return r.Max != nil && *r.Max == 100 }},
		{"100%10", func(r Range) bool { return r.Min != nil && *r.Min == 90 && r.Max != nil && *r.Max == 110 }},
	}

	for _, tt := range tests {
		r, err := ParseRange(tt.input, mockHumanToX)
		if err != nil {
			t.Errorf("ParseRange(%q) error: %v", tt.input, err)
			continue
		}
		if !tt.check(r) {
			t.Errorf("ParseRange(%q) failed check: %+v", tt.input, r)
		}
	}
}

func TestCalculatePercentiles(t *testing.T) {
	values := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	res := CalculatePercentiles(values)
	if len(res) != 101 {
		t.Errorf("CalculatePercentiles len = %d, want 101", len(res))
	}
	if res[0] != 10 {
		t.Errorf("res[0] = %d, want 10", res[0])
	}
	if res[100] != 100 {
		t.Errorf("res[100] = %d, want 100", res[100])
	}

	// Test empty
	resEmpty := CalculatePercentiles(nil)
	if len(resEmpty) != 101 {
		t.Errorf("CalculatePercentiles empty len = %d, want 101", len(resEmpty))
	}
}

func TestPercent(t *testing.T) {
	if got := Percent(50, 200); got != 25.0 {
		t.Errorf("Percent(50, 200) = %v, want 25.0", got)
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
		got, _ := FloatFromPercent(tt.input)
		if got != tt.expected {
			t.Errorf("FloatFromPercent(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestRandomFloat(t *testing.T) {
	got := RandomFloat()
	if got < 0 || got > 1 {
		t.Errorf("RandomFloat() = %v, want in [0, 1)", got)
	}
}

func TestRandomInt(t *testing.T) {
	got := RandomInt(5, 10)
	if got < 5 || got >= 10 {
		t.Errorf("RandomInt(5, 10) = %v, want in [5, 10)", got)
	}
	if got2 := RandomInt(10, 5); got2 != 10 {
		t.Errorf("RandomInt(10, 5) = %v, want 10", got2)
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
		if got := LinearInterpolation(tt.x, data); got != tt.want {
			t.Errorf("LinearInterpolation(%v) = %v, want %v", tt.x, got, tt.want)
		}
	}
	if got := LinearInterpolation(5, nil); got != 0 {
		t.Errorf("LinearInterpolation with nil data = %v, want 0", got)
	}
}

func TestSafeMean(t *testing.T) {
	if got := SafeMean([]int{1, 2, 3}); got != 2.0 {
		t.Errorf("SafeMean([1, 2, 3]) = %v, want 2.0", got)
	}
	if got := SafeMean([]float64{}); got != 0 {
		t.Errorf("SafeMean([]) = %v, want 0", got)
	}
}

func TestSafeMedian(t *testing.T) {
	if got := SafeMedian([]int{1, 3, 2}); got != 2.0 {
		t.Errorf("SafeMedian([1, 3, 2]) = %v, want 2.0", got)
	}
	if got := SafeMedian([]int{1, 2, 3, 4}); got != 2.5 {
		t.Errorf("SafeMedian([1, 2, 3, 4]) = %v, want 2.5", got)
	}
	if got := SafeMedian([]float64{}); got != 0 {
		t.Errorf("SafeMedian([]) = %v, want 0", got)
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
		got, err := HumanToBits(tt.input)
		if err != nil {
			t.Errorf("HumanToBits(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("HumanToBits(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestPercentageDifference(t *testing.T) {
	if got := PercentageDifference(10, 10); got != 0 {
		t.Errorf("PercentageDifference(10, 10) = %v, want 0", got)
	}
	if got := PercentageDifference(0, 0); got != 100.0 {
		t.Errorf("PercentageDifference(0, 0) = %v, want 100.0", got)
	}
}

func TestCalculateSegments(t *testing.T) {
	got := CalculateSegments(100, 10, 0.1)
	if len(got) == 0 {
		t.Errorf("CalculateSegments(100, 10, 0.1) returned nil")
	}
	if got2 := CalculateSegments(20, 10, 0.1); len(got2) != 1 || got2[0] != 0 {
		t.Errorf("CalculateSegments(20, 10, 0.1) = %v, want [0]", got2)
	}
	if got3 := CalculateSegments(0, 10, 0.1); got3 != nil {
		t.Errorf("CalculateSegments(0, 10, 0.1) = %v, want nil", got3)
	}
}

func TestCalculateSegmentsInt(t *testing.T) {
	got := CalculateSegmentsInt(100, 10, 5)
	if len(got) == 0 {
		t.Errorf("CalculateSegmentsInt(100, 10, 5) returned nil")
	}
	if got2 := CalculateSegmentsInt(20, 10, 0.1); len(got2) != 1 || got2[0] != 0 {
		t.Errorf("CalculateSegmentsInt(20, 10, 0.1) = %v, want [0]", got2)
	}
	if got3 := CalculateSegmentsInt(0, 10, 0.1); got3 != nil {
		t.Errorf("CalculateSegmentsInt(0, 10, 0.1) = %v, want nil", got3)
	}
}

func TestSafeIntFloat(t *testing.T) {
	if got := SafeInt("123"); got == nil || *got != 123 {
		t.Errorf("SafeInt(123) = %v, want 123", got)
	}
	if got := SafeInt(""); got != nil {
		t.Errorf("SafeInt('') = %v, want nil", got)
	}
	if got := SafeInt("abc"); got != nil {
		t.Errorf("SafeInt('abc') = %v, want nil", got)
	}

	if got := SafeFloat("123.45"); got == nil || *got != 123.45 {
		t.Errorf("SafeFloat(123.45) = %v, want 123.45", got)
	}
	if got := SafeFloat(""); got != nil {
		t.Errorf("SafeFloat('') = %v, want nil", got)
	}
}

func TestSqlHumanTime(t *testing.T) {
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
		if got := SqlHumanTime(tt.input); got != tt.expected {
			t.Errorf("SqlHumanTime(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMaxMin(t *testing.T) {
	if got := Max(1, 2); got != 2 {
		t.Errorf("Max(1, 2) = %v, want 2", got)
	}
	if got := Max(2, 1); got != 2 {
		t.Errorf("Max(2, 1) = %v, want 2", got)
	}
	if got := Min(1, 2); got != 1 {
		t.Errorf("Min(1, 2) = %v, want 1", got)
	}
	if got := Min(2, 1); got != 1 {
		t.Errorf("Min(2, 1) = %v, want 1", got)
	}
}
