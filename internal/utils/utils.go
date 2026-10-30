package utils

import (
	"database/sql"
)

func GetString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return ""
}

func GetInt64(v any) int64 {
	switch i := v.(type) {
	case int:
		return int64(i)
	case int32:
		return int64(i)
	case int64:
		return i
	}
	return 0
}

func GetFloat64(v any) float64 {
	switch f := v.(type) {
	case float32:
		return float64(f)
	case float64:
		return f
	case int64:
		return float64(f)
	case int:
		return float64(f)
	}
	return 0
}

func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func Int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func Float64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

type Range struct {
	Min   *int64
	Max   *int64
	Value *int64
}

func (r Range) Matches(val int64) bool {
	if r.Value != nil && val != *r.Value {
		return false
	}
	if r.Min != nil && val < *r.Min {
		return false
	}
	if r.Max != nil && val > *r.Max {
		return false
	}
	return true
}

func ToNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: i != 0}
}

func ToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func ToNullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: f, Valid: f != 0}
}
