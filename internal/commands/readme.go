package commands

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

type ReadmeCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.PathFilterFlags  `embed:""`
	models.FilterFlags      `embed:""`
	models.MediaFilterFlags `embed:""`
	models.TimeFilterFlags  `embed:""`
	models.DeletedFlags     `embed:""`
	models.SortFlags        `embed:""`
	models.DisplayFlags     `embed:""`
	models.AggregateFlags   `embed:""`
	models.TextFlags        `embed:""`
	models.SimilarityFlags  `embed:""`
	models.DedupeFlags      `embed:""`
	models.FTSFlags         `embed:""`
	models.PlaybackFlags    `embed:""`
	models.MpvActionFlags   `embed:""`
	models.PostActionFlags  `embed:""`
	models.HashingFlags     `embed:""`
	models.MergeFlags       `embed:""`
	models.DatabaseFlags    `embed:""`
}

func (c *ReadmeCmd) Run(ctx *kong.Context) error {
	var sb strings.Builder

	sb.WriteString("# discoteca\n\n")
	sb.WriteString("Golang implementation of xklb/library\n\n")
	sb.WriteString("## Install\n\n")
	sb.WriteString("    go install github.com/chapmanjacobd/discoteca/cmd/disco@latest\n\n")
	sb.WriteString("## Usage\n\n")

	examples := map[string][]string{
		"add": {
			"disco add my_videos.db ~/Videos",
			"disco add --video-only my_videos.db /mnt/media",
		},
		"print": {
			"disco print my_videos.db",
			"disco print my_videos.db -u size --reverse",
			"disco print my_videos.db --big-dirs -u count",
		},
		"search": {
			"disco search my_videos.db 'matrix'",
			"disco search my_videos.db 'cyberpunk' --video-only",
		},
		"watch": {
			"disco watch my_videos.db",
			"disco watch my_videos.db -r --limit 10",
			"disco watch my_videos.db --size '>1GB'",
		},
		"listen": {
			"disco listen my_music.db",
			"disco listen my_music.db --random",
		},
		"serve": {
			"disco serve my_videos.db my_music.db",
			"disco serve --trashcan my_videos.db",
		},
		"disk-usage": {
			"disco du my_videos.db",
			"disco du my_videos.db --depth 2",
		},
		"history": {
			"disco history my_videos.db",
			"disco history my_videos.db --inprogress",
		},
		"optimize": {
			"disco optimize my_videos.db",
		},
	}

	// Iterate through subcommands
	for _, node := range ctx.Model.Children {
		if node.Hidden {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s\n\n", node.Name))
		sb.WriteString(fmt.Sprintf("%s\n\n", node.Help))

		if ex, ok := examples[node.Name]; ok {
			sb.WriteString("Examples:\n\n```bash\n")
			for _, line := range ex {
				sb.WriteString(fmt.Sprintf("$ %s\n", line))
			}
			sb.WriteString("```\n\n")
		}

		sb.WriteString("<details><summary>All Options</summary>\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString(fmt.Sprintf("$ disco %s --help\n", node.Name))

		if len(node.Flags) > 0 {
			sb.WriteString("\nFlags:\n")
			for _, flag := range node.Flags {
				if flag.Hidden {
					continue
				}
				short := ""
				if flag.Short != 0 {
					short = fmt.Sprintf("-%c, ", flag.Short)
				}
				sb.WriteString(fmt.Sprintf("  %s--%s\n", short, flag.Name))
				sb.WriteString(fmt.Sprintf("        %s\n", flag.Help))
			}
		}

		sb.WriteString("```\n\n")
		sb.WriteString("</details>\n\n")
	}

	fmt.Print(sb.String())
	return nil
}
