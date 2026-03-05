package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractText extracts plain text from a given file path.
// Supports .txt, .pdf, .epub.
func ExtractText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".log", ".ini", ".conf", ".cfg":
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case ".pdf":
		// pdftotext -layout input.pdf -
		cmd := exec.Command("pdftotext", "-layout", path, "-")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("pdftotext failed: %w", err)
		}
		return string(out), nil
	case ".epub":
		// Simple EPUB extraction: unzip html/xhtml files and strip tags
		// We use sh -c to leverage globbing inside zip which unzip supports via arguments
		cmd := exec.Command("bash", "-c", fmt.Sprintf("set -o pipefail; unzip -p %q '*.html' '*.xhtml' '*.htm' '*.xml' | sed -e 's/<[^>]*>/ /g'", path))
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("epub extraction failed: %w", err)
		}
		return string(out), nil
	default:
		// Fallback: try reading as text
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), nil
		}
		return "", fmt.Errorf("unsupported format: %s", ext)
	}
}

// GenerateRSVPAss generates an ASS subtitle file content for RSVP.
func GenerateRSVPAss(text string, wpm int) (string, float64) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return "", 0
	}

	durationPerWord := 60.0 / float64(wpm)
	totalDuration := float64(len(words)) * durationPerWord

	var sb strings.Builder
	sb.WriteString("[Script Info]\n")
	sb.WriteString("ScriptType: v4.00+\n")
	sb.WriteString("PlayResX: 1280\n")
	sb.WriteString("PlayResY: 720\n")
	sb.WriteString("\n")
	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	// Centered large text
	sb.WriteString("Style: Default,Arial,80,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2,0,5,10,10,10,1\n")
	sb.WriteString("\n")
	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	startTime := 0.0
	for _, word := range words {
		endTime := startTime + durationPerWord

		startStr := formatAssTime(startTime)
		endStr := formatAssTime(endTime)

		// Sanitize word for ASS
		word = strings.ReplaceAll(word, "{", "\\{")
		word = strings.ReplaceAll(word, "}", "\\}")

		// Highlight the middle character/part if possible (ORP - Optimal Recognition Point)
		// Simple implementation: just show the word
		sb.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startStr, endStr, word))
		startTime = endTime
	}

	return sb.String(), totalDuration
}

func formatAssTime(seconds float64) string {
	h := int(seconds / 3600)
	m := int((seconds - float64(h)*3600) / 60)
	s := seconds - float64(h)*3600 - float64(m)*60
	return fmt.Sprintf("%d:%02d:%05.2f", h, m, s)
}

// GenerateTTS generates a WAV file from text using espeak-ng.
func GenerateTTS(text string, outputPath string, wpm int) error {
	// Check for espeak-ng
	espeakBin := "espeak-ng"
	if _, err := exec.LookPath(espeakBin); err != nil {
		return fmt.Errorf("espeak-ng not found")
	}

	cmd := exec.Command(espeakBin, "-w", outputPath, "-s", fmt.Sprintf("%d", wpm))
	cmd.Stdin = strings.NewReader(text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("espeak-ng failed: %s: %s", err, string(output))
	}
	return nil
}
