package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type RegexSortCmd struct {
	models.CoreFlags `embed:""`
	models.SortFlags `embed:""`
	models.TextFlags `embed:""`

	InputPath  string `help:"Input file path (default stdin)"   default:"-" arg:"" optional:""`
	OutputPath string `help:"Output file path (default stdout)"`

	// For testing
	Reader io.Reader `kong:"-"`
	Writer io.Writer `kong:"-"`
}

func (c *RegexSortCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags: c.CoreFlags,
		SortFlags: c.SortFlags,
		TextFlags: c.TextFlags,
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
		mapping := make(map[string]string)
		for i, line := range lines {
			sentence := utils.PathToSentence(line)
			sentenceStrings[i] = sentence
			mapping[sentence] = line
		}

		sortedSentences := utils.TextProcessor(flags, sentenceStrings)
		for _, s := range sortedSentences {
			processedLines = append(processedLines, mapping[s])
		}
	} else {
		processedLines = utils.TextProcessor(flags, lines)
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
