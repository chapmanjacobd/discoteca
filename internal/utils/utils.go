package utils

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

func GetString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func GetInt(v any) int {
	switch i := v.(type) {
	case int:
		return i
	case int32:
		return int(i)
	case int64:
		return int(i)
	}
	return 0
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

type Slice struct {
	Start *int
	Stop  *int
	Step  *int
}

func ParseSlice(s string) (Slice, error) {
	parts := strings.Split(s, ":")
	if len(parts) > 3 {
		return Slice{}, fmt.Errorf("invalid slice: %s", s)
	}

	var res Slice
	if len(parts) >= 1 && parts[0] != "" {
		if val, err := strconv.Atoi(parts[0]); err == nil {
			res.Start = &val
		}
	}
	if len(parts) >= 2 && parts[1] != "" {
		if val, err := strconv.Atoi(parts[1]); err == nil {
			res.Stop = &val
		}
	}
	if len(parts) == 3 && parts[2] != "" {
		if val, err := strconv.Atoi(parts[2]); err == nil {
			res.Step = &val
		}
	}

	return res, nil
}

func DictFilterBool(d map[string]any) map[string]any {
	if d == nil {
		return nil
	}
	res := make(map[string]any)
	for k, v := range d {
		if v != nil && v != "" && v != 0 && v != false {
			res[k] = v
		}
	}
	if len(res) == 0 {
		return nil
	}
	return res
}
