package utils

import (
	"fmt"
	"strings"
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

func FormatDurationShort(seconds int) string {
	if seconds == 0 {
		return ""
	}

	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}

	minutes := float64(seconds) / 60.0
	if minutes < 1.1 {
		return "1 minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%.1f minutes", minutes)
	}

	hours := minutes / 60.0
	if hours < 1.1 {
		return "1 hour"
	} else if hours < 24 {
		return fmt.Sprintf("%.1f hours", hours)
	}

	days := int(hours / 24)
	years := days / 365
	remainingDays := days % 365

	if years > 0 {
		if remainingDays > 0 {
			return fmt.Sprintf("%d years and %d days", years, remainingDays)
		}
		return fmt.Sprintf("%d years", years)
	}

	if days > 1 {
		return fmt.Sprintf("%d days", days)
	}
	return "1 day"
}

func FormatSize(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
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

func RelativeDatetime(timestamp int64) string {
	if timestamp == 0 {
		return "-"
	}
	t := time.Unix(timestamp, 0)
	now := time.Now()
	diff := now.Sub(t)

	// Past
	if diff > 0 {
		if diff.Hours() < 24 && t.Day() == now.Day() {
			return t.Format("today, 15:04")
		}
		if diff.Hours() < 48 && t.Day() == now.AddDate(0, 0, -1).Day() {
			return t.Format("yesterday, 15:04")
		}
		if diff.Hours() < 24*45 {
			days := int(diff.Hours() / 24)
			if days == 0 {
				days = 1
			}
			return fmt.Sprintf("%d days ago, %s", days, t.Format("15:04"))
		}
	} else {
		// Future
		absDiff := -diff
		if absDiff.Hours() < 24 && t.Day() == now.Day() {
			return t.Format("today, 15:04")
		}
		if absDiff.Hours() < 48 && t.Day() == now.AddDate(0, 0, 1).Day() {
			return t.Format("tomorrow, 15:04")
		}
		if absDiff.Hours() < 24*45 {
			days := int(absDiff.Hours() / 24)
			return fmt.Sprintf("in %d days, %s", days, t.Format("15:04"))
		}
	}

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

func FormatPlaybackDuration(duration, segmentStart, segmentEnd int64) string {
	if segmentStart > duration && segmentEnd == 0 {
		segmentEnd = segmentStart + duration
	}

	if segmentStart > segmentEnd && segmentEnd > 0 && segmentStart+segmentEnd > duration {
		segmentStart, segmentEnd = segmentEnd, segmentStart
	}

	if segmentStart != 0 || segmentEnd != 0 {
		var segmentDuration int64
		if segmentEnd > 0 {
			segmentDuration = segmentStart - segmentEnd
		} else {
			segmentDuration = duration - segmentStart
		}

		if segmentDuration < 0 {
			segmentDuration = -segmentDuration
		}

		startStr := strings.TrimSpace(SecondsToHHMMSS(segmentStart))
		endVal := segmentEnd
		if endVal == 0 {
			endVal = duration
		}
		endStr := strings.TrimSpace(SecondsToHHMMSS(endVal))
		durationStr := strings.TrimSpace(SecondsToHHMMSS(segmentDuration))

		return fmt.Sprintf("Duration: %s (%s to %s)", durationStr, startStr, endStr)
	}

	durationStr := strings.TrimSpace(SecondsToHHMMSS(duration))
	return fmt.Sprintf("Duration: %s", durationStr)
}
