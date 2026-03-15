package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type ClusterSortCmd struct {
	models.CoreFlags       `embed:""`
	models.SimilarityFlags `embed:""`
	models.TextFlags       `embed:""`

	InputPath  string `arg:"" optional:"" help:"Input file path (default stdin)" default:"-"`
	OutputPath string `help:"Output file path (default stdout)"`
}

func (c *ClusterSortCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
	flags := models.GlobalFlags{
		CoreFlags:       c.CoreFlags,
		SimilarityFlags: c.SimilarityFlags,
		TextFlags:       c.TextFlags,
	}

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

	groups := aggregate.ClusterPaths(flags, lines)

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

	if c.PrintGroups {
		return utils.PrintJSON(groups)
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
