package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// ErrUserQuit is returned when the user chooses to quit during interactive decision.
var ErrUserQuit = errors.New("user requested quit")

func HideRedundantFirstPlayed(media []models.MediaWithDB) {
	for i := range media {
		if media[i].PlayCount != nil && *media[i].PlayCount <= 1 {
			media[i].TimeFirstPlayed = nil
		}
	}
}

func PrintMedia(flags models.DisplayFlags, columns []string, media []models.MediaWithDB) error {
	if flags.JSON {
		return utils.PrintJSON(media)
	}

	if len(columns) == 0 {
		columns = []string{"path", "duration", "size"}
	}

	// Print header
	fmt.Println(strings.Join(columns, "\t"))

	for _, m := range media {
		var row []string
		for _, col := range columns {
			switch col {
			case "path":
				row = append(row, m.Path)
			case "title":
				row = append(row, utils.StringValue(m.Title))
			case "duration":
				row = append(row, utils.FormatDuration(int(utils.Int64Value(m.Duration))))
			case "size":
				row = append(row, utils.FormatSize(utils.Int64Value(m.Size)))
			case "play_count":
				row = append(row, strconv.FormatInt(utils.Int64Value(m.PlayCount), 10))
			case "playhead":
				row = append(row, utils.FormatDuration(int(utils.Int64Value(m.Playhead))))
			case "time_last_played":
				row = append(row, utils.FormatTime(utils.Int64Value(m.TimeLastPlayed)))
			case "db":
				row = append(row, filepath.Base(m.DB))
			}
		}
		fmt.Println(strings.Join(row, "\t"))
	}

	fmt.Printf("\n%d media files\n", len(media))
	return nil
}

func PrintFolders(flags models.DisplayFlags, columns []string, folders []models.FolderStats) error {
	if flags.JSON {
		return utils.PrintJSON(folders)
	}

	if len(columns) == 0 {
		columns = []string{"path", "exists_count", "size", "duration"}
	}

	fmt.Println(strings.Join(columns, "\t"))

	for _, f := range folders {
		var row []string
		for _, col := range columns {
			switch col {
			case "path":
				row = append(row, f.Path)
			case "count":
				row = append(row, strconv.Itoa(f.Count))
			case "exists_count":
				row = append(row, strconv.Itoa(f.ExistsCount))
			case "deleted_count":
				row = append(row, strconv.Itoa(f.DeletedCount))
			case "played_count":
				row = append(row, strconv.Itoa(f.PlayedCount))
			case "size":
				row = append(row, utils.FormatSize(f.TotalSize))
			case "duration":
				row = append(row, utils.FormatDuration(int(f.TotalDuration)))
			case "avg_size":
				row = append(row, utils.FormatSize(f.AvgSize))
			case "avg_duration":
				row = append(row, utils.FormatDuration(int(f.AvgDuration)))
			case "median_size":
				row = append(row, utils.FormatSize(f.MedianSize))
			case "median_duration":
				row = append(row, utils.FormatDuration(int(f.MedianDuration)))
			case "folder_count":
				row = append(row, strconv.Itoa(f.FolderCount))
			}
		}
		fmt.Println(strings.Join(row, "\t"))
	}

	fmt.Printf("\n%d groups\n", len(folders))
	return nil
}

func InteractiveDecision(ctx context.Context, flags models.GlobalFlags, m models.MediaWithDB) error {
	fmt.Printf("\nAction for %s?\n", m.Path)
	fmt.Println("  [k]eep (default)")
	fmt.Println("  [d]elete")
	fmt.Println("  [t]rash")
	fmt.Println("  [m]ark-deleted")
	fmt.Println("  [q]uit")

	var input string
	fmt.Print("> ")
	fmt.Scanln(&input)

	switch strings.ToLower(input) {
	case "d":
		return DeleteMediaItem(m)
	case "t":
		return utils.Trash(flags, m.Path)
	case "m":
		return MarkDeletedItem(ctx, m)
	case "q":
		return ErrUserQuit
	}

	return nil
}
