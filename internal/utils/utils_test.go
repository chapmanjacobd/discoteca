package utils_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"hello", "hello"},
		{123, ""},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := utils.GetString(tt.input); got != tt.expected {
			t.Errorf("GetString(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		input    any
		expected int64
	}{
		{int64(123), 123},
		{123, 123},
		{nil, 0},
	}
	for _, tt := range tests {
		if got := utils.GetInt64(tt.input); got != tt.expected {
			t.Errorf("GetInt64(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestStringValue(t *testing.T) {
	s := "hello"
	if got := utils.StringValue(&s); got != "hello" {
		t.Errorf("StringValue(&s) = %v, want hello", got)
	}
	if got := utils.StringValue(nil); got != "" {
		t.Errorf("StringValue(nil) = %v, want empty string", got)
	}
}

func TestInt64Value(t *testing.T) {
	var i int64 = 123
	if got := utils.Int64Value(&i); got != 123 {
		t.Errorf("Int64Value(&i) = %v, want 123", got)
	}
	if got := utils.Int64Value(nil); got != 0 {
		t.Errorf("Int64Value(nil) = %v, want 0", got)
	}
}

func TestRangeMatches(t *testing.T) {
	val10 := int64(10)
	val20 := int64(20)
	tests := []struct {
		r     utils.Range
		val   int64
		match bool
	}{
		{utils.Range{Value: &val10}, 10, true},
		{utils.Range{Value: &val10}, 11, false},
		{utils.Range{Min: &val10}, 10, true},
		{utils.Range{Min: &val10}, 9, false},
		{utils.Range{Max: &val20}, 20, true},
		{utils.Range{Max: &val20}, 21, false},
		{utils.Range{Min: &val10, Max: &val20}, 15, true},
	}
	for _, tt := range tests {
		if got := tt.r.Matches(tt.val); got != tt.match {
			t.Errorf("%+v.Matches(%d) = %v, want %v", tt.r, tt.val, got, tt.match)
		}
	}
}

func TestToNull(t *testing.T) {
	if got := utils.ToNullInt64(123); !got.Valid || got.Int64 != 123 {
		t.Errorf("ToNullInt64(123) failed: %v", got)
	}
	if got := utils.ToNullInt64(0); got.Valid {
		t.Error("ToNullInt64(0) should be invalid")
	}

	if got := utils.ToNullString("hello"); !got.Valid || got.String != "hello" {
		t.Errorf("ToNullString(hello) failed: %v", got)
	}
	if got := utils.ToNullString(""); got.Valid {
		t.Error("ToNullString('') should be invalid")
	}

	if got := utils.ToNullFloat64(1.23); !got.Valid || got.Float64 != 1.23 {
		t.Errorf("ToNullFloat64(1.23) failed: %v", got)
	}
	if got := utils.ToNullFloat64(0); got.Valid {
		t.Error("ToNullFloat64(0) should be invalid")
	}
}
