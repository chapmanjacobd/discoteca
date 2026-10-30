package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/textutil"
)

type RegexSortCmd struct {
	models.CoreFlags           `embed:""`
	models.TextProcessingFlags `embed:""`

	InputPath  string `help:"Input file path (default stdin)"   default:"-" arg:"" optional:""`
	OutputPath string `help:"Output file path (default stdout)"`

	// For testing
	Reader io.Reader `kong:"-"`
	Writer io.Writer `kong:"-"`
}

func (c *RegexSortCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags: c.CoreFlags,
		TextFlags: models.TextFlags{
			Regexs:     c.Regexs,
			WordSorts:  c.WordSorts,
			LineSorts:  c.LineSorts,
			Preprocess: c.Preprocess,
			StopWords:  c.StopWords,
			Duplicates: c.Duplicates,
			UniqueOnly: c.UniqueOnly,
		},
	}

	var lines []string
	var scanner *bufio.Scanner

	if c.Reader != nil {
		scanner = bufio.NewScanner(c.Reader)
	} else if c.InputPath == "-" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		file, err := os.Open(c.InputPath)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	var processedLines []string
	if c.Preprocess {
		sentenceStrings := make([]string, len(lines))
		mapping := make(map[string][]string, len(lines))
		for i, line := range lines {
			sentence := textutil.PathToSentence(line)
			sentenceStrings[i] = sentence
			mapping[sentence] = append(mapping[sentence], line)
		}

		sortedSentences := textutil.TextProcessor(flags, sentenceStrings)
		seen := make(map[string]int, len(mapping))
		for _, s := range sortedSentences {
			idx := seen[s]
			if idx < len(mapping[s]) {
				processedLines = append(processedLines, mapping[s][idx])
				seen[s]++
			}
		}
	} else {
		processedLines = textutil.TextProcessor(flags, lines)
	}

	var writer *bufio.Writer
	if c.Writer != nil {
		writer = bufio.NewWriter(c.Writer)
	} else if c.OutputPath != "" {
		file, err := os.Create(c.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		writer = bufio.NewWriter(file)
	} else {
		writer = bufio.NewWriter(os.Stdout)
	}

	for _, line := range processedLines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}
	}
	writer.Flush()

	return nil
}
