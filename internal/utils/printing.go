package utils

import (
	"fmt"
	"os"

	"github.com/mattn/go-runewidth"
)

// PrintOverwrite overwrites the current line in the terminal with the given text
func PrintOverwrite(text string) {
	// If not a terminal, just print normally
	file, ok := Stdout.(*os.File)
	if !ok {
		fmt.Fprintln(Stdout, text)
		return
	}
	fileInfo, _ := file.Stat()
	if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		fmt.Fprintln(Stdout, text)
		return
	}

	maxWidth := 80 // Default fallback
	if TERMINAL_SIZE.columns > 0 {
		maxWidth = TERMINAL_SIZE.columns - 1
	}

	if runewidth.StringWidth(text) > maxWidth {
		text = ShortenMiddle(text, maxWidth)
	}

	if IsLinux || IsMac {
		fmt.Fprintf(Stdout, "\r%s%s", text, ClearSeq)
	} else if IsWindows {
		fmt.Fprintf(Stdout, "\r%s", text)
	} else {
		fmt.Fprintln(Stdout, text)
	}
}

func ColNaturalDate(data []map[string]any, key string) []map[string]any {
	for _, d := range data {
		if v, ok := d[key]; ok {
			if ts := GetInt64(v); ts > 0 {
				d[key] = FormatTime(ts)
			} else {
				d[key] = nil
			}
		}
	}
	return data
}

func ColFilesize(data []map[string]any, key string) []map[string]any {
	for _, d := range data {
		if v, ok := d[key]; ok {
			if size := GetInt64(v); size > 0 {
				d[key] = FormatSize(size)
			} else {
				d[key] = nil
			}
		}
	}
	return data
}

func ColDuration(data []map[string]any, key string) []map[string]any {
	for _, d := range data {
		if v, ok := d[key]; ok {
			if dur := GetInt(v); dur > 0 {
				d[key] = FormatDuration(dur)
			} else {
				d[key] = ""
			}
		}
	}
	return data
}
