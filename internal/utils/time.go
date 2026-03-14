package utils

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

// ParseDate parses a date string into a Unix timestamp using a fixed set of layouts
func ParseDate(dateStr string) int64 {
	// Try parsing as unix timestamp first
	if ts, err := strconv.ParseInt(dateStr, 10, 64); err == nil && ts > 0 {
		return ts
	}

	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"01/02/2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// IsTZAware checks if a time is not in the Local or UTC location (as a proxy for "aware")
// In Go, time.Time is always aware of its Location.
func IsTZAware(t time.Time) bool {
	name, offset := t.Zone()
	return name != "UTC" && name != "Local" || offset != 0
}

// SuperParser uses dateparse to attempt to parse a date string with various strategies
func SuperParser(dateStr string) *time.Time {
	t, err := dateparse.ParseAny(dateStr)
	if err == nil {
		return &t
	}

	// Try with specific layouts if dateparse fails
	layouts := []string{
		"02/01/2006",
		"02/01/2006 15:04:05",
		"20060102",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, dateStr); err == nil {
			return &t
		}
	}

	return nil
}

type dateSortKey struct {
	hasMonth bool
	hasDay   bool
	negTS    int64
}

func getDateSortKey(t time.Time) dateSortKey {
	// dateparse doesn't easily tell us if month/day were in the original string
	// But we can check if they are non-zero/default if we had a way.
	// Since we don't, we'll assume if it's not Jan 1st, it has month/day.
	// This is a bit of a hack compared to Python's dateutil.
	return dateSortKey{
		hasMonth: t.Month() != time.January || t.Day() != 1,
		hasDay:   t.Day() != 1,
		negTS:    -t.Unix(),
	}
}

// SpecificDate finds the earliest most-specific past date from a list of strings
func SpecificDate(dates ...string) *time.Time {
	var pastDates []time.Time
	now := time.Now()

	for _, s := range dates {
		if s == "" {
			continue
		}
		t := SuperParser(s)
		if t != nil && t.Before(now) {
			pastDates = append(pastDates, *t)
		}
	}

	if len(pastDates) == 0 {
		return nil
	}

	// Sort by specificity (has month, then has day) then by earliest date
	sort.Slice(pastDates, func(i, j int) bool {
		ki := getDateSortKey(pastDates[i])
		kj := getDateSortKey(pastDates[j])

		if ki.hasMonth != kj.hasMonth {
			return ki.hasMonth
		}
		if ki.hasDay != kj.hasDay {
			return ki.hasDay
		}
		return ki.negTS > kj.negTS
	})

	best := pastDates[0]
	for i := 1; i < len(pastDates); i++ {
		ki := getDateSortKey(best)
		kj := getDateSortKey(pastDates[i])

		better := false
		if kj.hasMonth != ki.hasMonth {
			if kj.hasMonth {
				better = true
			}
		} else if kj.hasDay != ki.hasDay {
			if kj.hasDay {
				better = true
			}
		} else if kj.negTS > ki.negTS {
			better = true
		}

		if better {
			best = pastDates[i]
		}
	}

	return &best
}

// TubeDate extracts and parses dates from various common metadata keys
func TubeDate(v map[string]any) *int64 {
	keys := []string{"release_date", "timestamp", "upload_date", "date", "created_at", "published", "updated"}
	var uploadDate any

	for _, k := range keys {
		if val, ok := v[k]; ok && val != nil {
			uploadDate = val
			delete(v, k)
			break
		}
	}

	if uploadDate == nil {
		return nil
	}

	switch d := uploadDate.(type) {
	case int64:
		if d > 30000000 {
			return &d
		}
	case int:
		d64 := int64(d)
		if d64 > 30000000 {
			return &d64
		}
	case time.Time:
		ts := d.Unix()
		return &ts
	case string:
		t := SuperParser(d)
		if t != nil {
			ts := t.Unix()
			return &ts
		}
	}

	return nil
}

// UtcFromLocalTimestamp converts a local Unix timestamp to a UTC time.Time
// ParseDateOrRelative parses a date string into a Unix timestamp.
// It supports absolute dates (YYYY-MM-DD) and relative strings (e.g., "3 days").
func ParseDateOrRelative(dateStr string) int64 {
	if ts := ParseDate(dateStr); ts > 0 {
		return ts
	}

	// Try relative
	s := strings.TrimSpace(dateStr)
	isFuture := false
	if strings.HasPrefix(s, "+") {
		isFuture = true
		s = s[1:]
	} else if strings.HasPrefix(s, "-") {
		s = s[1:]
	}

	if seconds, err := HumanToSeconds(s); err == nil && seconds > 0 {
		now := time.Now().Unix()
		if isFuture {
			return now + seconds
		}
		return now - seconds
	}

	return 0
}

func UtcFromLocalTimestamp(n int64) time.Time {
	return time.Unix(n, 0).UTC()
}

func ToDecade(year int) string {
	if year < 1900 {
		return ""
	}
	decade := (year / 10) * 10
	return fmt.Sprintf("%02ds", decade%100)
}
