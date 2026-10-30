package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type ClusterSortCmd struct {
	models.CoreFlags            `embed:""`
	models.DuplicateFilterFlags `embed:""`

	InputPath  string `help:"Input file path (default stdin)"   default:"-" arg:"" optional:""`
	OutputPath string `help:"Output file path (default stdout)"`
	Clusters   int    `help:"Number of clusters (default: sqrt(number of lines))"`
	JSON       bool   `help:"Output results as JSON"                                 short:"j"`
}

func (c *ClusterSortCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)

	var lines []string
	var scanner *bufio.Scanner

	if c.InputPath == "-" {
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

	if len(lines) == 0 {
		return nil
	}

	groups := aggregate.ClusterPaths(lines, c.Clusters)

	if c.Duplicates != nil && *c.Duplicates {
		groups = aggregate.FilterNearDuplicates(groups)
	}

	if c.UniqueOnly != nil {
		var filtered []models.FolderStats
		for _, g := range groups {
			if *c.UniqueOnly && len(g.Files) == 1 {
				filtered = append(filtered, g)
			} else if !*c.UniqueOnly && len(g.Files) > 1 {
				filtered = append(filtered, g)
			}
		}
		groups = filtered
	}

	if c.JSON {
		return fileutil.PrintJSON(groups)
	}

	var writer *bufio.Writer
	if c.OutputPath != "" {
		file, err := os.Create(c.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		writer = bufio.NewWriter(file)
	} else {
		writer = bufio.NewWriter(os.Stdout)
	}

	for _, g := range groups {
		for _, m := range g.Files {
			if _, err := writer.WriteString(m.Path + "\n"); err != nil {
				return fmt.Errorf("error writing output: %w", err)
			}
		}
	}
	writer.Flush()

	return nil
}
