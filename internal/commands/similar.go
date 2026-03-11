package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/query"
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

func (c *SimilarFilesCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
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
	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)

	// Defaults for similar files
	if !c.FilterSizes && !c.FilterDurations && !c.FilterNames {
		c.FilterSizes = true
		c.FilterDurations = true
	}

	groups := aggregate.ClusterByNumbers(flags, media)

	if c.OnlyOriginals || c.OnlyDuplicates {
		for i, g := range groups {
			if c.OnlyOriginals {
				groups[i].Files = g.Files[:1]
			} else if c.OnlyDuplicates {
				groups[i].Files = g.Files[1:]
			}
		}
	}

	if c.JSON {
		return json.NewEncoder(os.Stdout).Encode(groups)
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

func (c *SimilarFoldersCmd) Run(ctx *kong.Context) error {
	models.SetupLogging(c.Verbose)
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
	media, err := query.MediaQuery(context.Background(), c.Databases, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)

	// Defaults for similar folders
	if !c.FilterSizes && !c.FilterDurations && !c.FilterNames && !c.FilterCounts {
		c.FilterCounts = true
		c.FilterSizes = true
	}

	folders := query.AggregateMedia(media, flags)

	var groups []models.FolderStats
	if c.FilterNames {
		// First pass: group by name
		groups = aggregate.ClusterFoldersByName(flags, folders)

		if c.FilterSizes || c.FilterCounts || c.FilterDurations {
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

	return PrintFolders(c.DisplayFlags, c.Columns, groups)
}
