package commands

import (
	"context"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type SimilarFilesCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.SimilarityFlags  `embed:""`

	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

type SimilarFoldersCmd struct {
	models.CoreFlags        `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.SimilarityFlags  `embed:""`

	Databases []string `arg:"" required:"" help:"SQLite database files" type:"existingfile"`
}

func (c *SimilarFilesCmd) Run(ctx *kong.Context) error {
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		SortFlags:        c.SortFlags,
		DisplayFlags:     c.DisplayFlags,
		SimilarityFlags:  c.SimilarityFlags,
	}
	return runSimilar(flags, c.Databases, false)
}

func (c *SimilarFoldersCmd) Run(ctx *kong.Context) error {
	flags := models.GlobalFlags{
		CoreFlags:        c.CoreFlags,
		PathFilterFlags:  c.PathFilterFlags,
		FilterFlags:      c.FilterFlags,
		MediaFilterFlags: c.MediaFilterFlags,
		TimeFilterFlags:  c.TimeFilterFlags,
		SortFlags:        c.SortFlags,
		DisplayFlags:     c.DisplayFlags,
		SimilarityFlags:  c.SimilarityFlags,
	}
	return runSimilar(flags, c.Databases, true)
}

func runSimilar(flags models.GlobalFlags, dbs []string, folderMode bool) error {
	models.SetupLogging(flags.Verbose)

	media, err := query.MediaQuery(context.Background(), dbs, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)

	if folderMode {
		// Defaults for similar folders
		if !flags.FilterSizes && !flags.FilterDurations && !flags.FilterNames && !flags.FilterCounts {
			flags.FilterCounts = true
			flags.FilterSizes = true
		}

		folders := query.AggregateMedia(media, flags)

		var groups []models.FolderStats
		if flags.FilterNames {
			// First pass: group by name
			groups = aggregate.ClusterFoldersByName(flags, folders)

			if flags.FilterSizes || flags.FilterCounts || flags.FilterDurations {
				// Second pass: filter each group by numerical similarity
				var refinedGroups []models.FolderStats
				for _, group := range groups {
					if len(group.Files) < 2 {
						continue
					}
					// Break this merged group back into individual folders
					subFolders := query.AggregateMedia(group.Files, flags)
					// Apply numerical clustering within this group
					subGroups := aggregate.ClusterFoldersByNumbers(flags, subFolders)
					refinedGroups = append(refinedGroups, subGroups...)
				}
				groups = refinedGroups
			}
		} else {
			groups = aggregate.ClusterFoldersByNumbers(flags, folders)
		}

		return PrintFolders(flags.DisplayFlags, flags.Columns, groups)
	}

	// File mode
	// Defaults for similar files
	if !flags.FilterSizes && !flags.FilterDurations && !flags.FilterNames {
		flags.FilterSizes = true
		flags.FilterDurations = true
	}

	groups := aggregate.ClusterByNumbers(flags, media)

	if flags.OnlyOriginals || flags.OnlyDuplicates {
		for i, g := range groups {
			if flags.OnlyOriginals {
				groups[i].Files = g.Files[:1]
			} else if flags.OnlyDuplicates {
				groups[i].Files = g.Files[1:]
			}
		}
	}

	if flags.JSON {
		return utils.PrintJSON(groups)
	}

	for _, g := range groups {
		fmt.Printf("Group: %s (%d files)\n", g.Path, len(g.Files))
		for _, m := range g.Files {
			fmt.Printf("  %s\n", m.Path)
		}
		fmt.Println()
	}

	fmt.Printf("%d groups\n", len(groups))
	return nil
}
