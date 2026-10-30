// Package formatutil formats sizes, durations, timestamps, and parsed dates.
package formatutil

import (
	"fmt"
	"time"
)

func FormatDuration(seconds int) string {
	if seconds == 0 {
		return "-"
	}
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if d > 0 {
		// Show days for durations > 24 hours
		return fmt.Sprintf("%dd %02d:%02d", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func FormatSize(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatTime(timestamp int64) string {
	if timestamp == 0 {
		return "-"
	}
	t := time.Unix(timestamp, 0)
	return t.Format("2006-01-02 15:04")
}

func SecondsToHHMMSS(seconds int64) string {
	neg := false
	if seconds < 0 {
		neg = true
		seconds = -seconds
	}

	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	prefix := ""
	if neg {
		prefix = "-"
	}

	if h > 0 {
		return fmt.Sprintf("%s%d:%02d:%02d", prefix, h, m, s)
	}
	return fmt.Sprintf("%s%d:%02d", prefix, m, s)
}
