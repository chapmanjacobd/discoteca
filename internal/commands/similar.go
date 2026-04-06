package commands

import (
	"context"
	"fmt"

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

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
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

	Databases []string `help:"SQLite database files" required:"true" arg:"" type:"existingfile"`
}

func (c *SimilarFilesCmd) Run(ctx context.Context) error {
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
	return runSimilar(ctx, flags, c.Databases, false)
}

func (c *SimilarFoldersCmd) Run(ctx context.Context) error {
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
	return runSimilar(ctx, flags, c.Databases, true)
}

func runSimilar(ctx context.Context, flags models.GlobalFlags, dbs []string, folderMode bool) error {
	models.SetupLogging(flags.Verbose)

	media, err := query.MediaQuery(ctx, dbs, flags)
	if err != nil {
		return err
	}

	media = query.FilterMedia(media, flags)

	if folderMode {
		return runSimilarFolders(flags, media)
	}

	return runSimilarFiles(flags, media)
}

func runSimilarFolders(flags models.GlobalFlags, media []models.MediaWithDB) error {
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
			groups = refineFolderGroups(flags, groups)
		}
	} else {
		groups = aggregate.ClusterFoldersByNumbers(flags, folders)
	}

	return PrintFolders(flags.DisplayFlags, flags.Columns, groups)
}

func refineFolderGroups(flags models.GlobalFlags, groups []models.FolderStats) []models.FolderStats {
	var refined []models.FolderStats
	for _, group := range groups {
		if len(group.Files) < 2 {
			continue
		}
		subFolders := query.AggregateMedia(group.Files, flags)
		subGroups := aggregate.ClusterFoldersByNumbers(flags, subFolders)
		refined = append(refined, subGroups...)
	}
	return refined
}

func runSimilarFiles(flags models.GlobalFlags, media []models.MediaWithDB) error {
	// Defaults for similar files
	if !flags.FilterSizes && !flags.FilterDurations && !flags.FilterNames {
		flags.FilterSizes = true
		flags.FilterDurations = true
	}

	groups := aggregate.ClusterByNumbers(flags, media)

	if flags.OnlyOriginals || flags.OnlyDuplicates {
		groups = filterOriginalsOrDuplicates(groups, flags)
	}

	if flags.JSON {
		return utils.PrintJSON(groups)
	}

	printFileGroups(groups)
	return nil
}

func filterOriginalsOrDuplicates(groups []models.FolderStats, flags models.GlobalFlags) []models.FolderStats {
	for i, g := range groups {
		if flags.OnlyOriginals {
			groups[i].Files = g.Files[:1]
		} else if flags.OnlyDuplicates {
			groups[i].Files = g.Files[1:]
		}
	}
	return groups
}

func printFileGroups(groups []models.FolderStats) {
	for _, g := range groups {
		fmt.Printf("Group: %s (%d files)\n", g.Path, len(g.Files))
		for _, m := range g.Files {
			fmt.Printf("  %s\n", m.Path)
		}
		fmt.Println()
	}
	fmt.Printf("%d groups\n", len(groups))
}
