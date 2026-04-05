package commands

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type SearchCaptionsCmd struct {
	models.CoreFlags        `embed:""`
	models.QueryFlags       `embed:""`
	models.FTSFlags         `embed:""`
	models.MediaFilterFlags `embed:""`
	models.PlaybackFlags    `embed:""`

	Database string   `help:"SQLite database file" required:"" arg:"" type:"existingfile"`
	Search   []string `help:"Search terms"         required:"" arg:""`

	Open    bool `help:"Open results in media player"`
	Overlap int  `help:"Overlap in seconds for merging captions" default:"8"`
}

type MergedCaption struct {
	Path  string
	Time  float64
	End   float64
	Text  string
	Title string
}

func (c *SearchCaptionsCmd) Run(ctx context.Context) error {
	models.SetupLogging(c.Verbose)

	sqlDB, err := db.Connect(ctx, c.Database)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	queries := db.New(sqlDB)
	queryStr := strings.Join(c.Search, " ")

	limit := int64(c.Limit)
	if c.All {
		limit = 1000000
	}

	rows, err := queries.SearchCaptions(ctx, db.SearchCaptionsParams{
		Query:     queryStr,
		VideoOnly: c.VideoOnly,
		AudioOnly: c.AudioOnly,
		ImageOnly: c.ImageOnly,
		TextOnly:  c.TextOnly,
		Limit:     limit,
	})
	if err != nil {
		return err
	}

	merged := c.mergeCaptions(rows)

	if c.Open {
		c.playCaptions(merged)
		return nil
	}

	c.printCaptions(merged)
	return nil
}

func (c *SearchCaptionsCmd) getEnd(t float64, text string) float64 {
	// Formula from original python: caption["time"] + (len(caption["text"]) / 4.2 / 220 * 60)
	return t + (float64(len(text)) / 4.2 / 220 * 60)
}

func (c *SearchCaptionsCmd) mergeCaptions(rows []db.SearchCaptionsRow) []MergedCaption {
	if len(rows) == 0 {
		return nil
	}

	var merged []MergedCaption
	var current *MergedCaption

	for _, row := range rows {
		timeVal := 0.0
		if row.Time.Valid {
			timeVal = row.Time.Float64
		}
		textVal := ""
		if row.Text.Valid {
			textVal = row.Text.String
		}
		titleVal := ""
		if row.Title.Valid {
			titleVal = row.Title.String
		}

		end := c.getEnd(timeVal, textVal)

		if current == nil {
			current = &MergedCaption{
				Path:  row.MediaPath,
				Time:  timeVal,
				End:   end,
				Text:  textVal,
				Title: titleVal,
			}
			continue
		}

		if current.Path == row.MediaPath &&
			(math.Abs(timeVal-current.End) <= float64(c.Overlap) ||
				math.Abs(timeVal-current.Time) <= float64(c.Overlap)) {

			current.End = end
			if !strings.Contains(current.Text, textVal) {
				current.Text += ". " + textVal
			}
		} else {
			merged = append(merged, *current)
			current = &MergedCaption{
				Path:  row.MediaPath,
				Time:  timeVal,
				End:   end,
				Text:  textVal,
				Title: titleVal,
			}
		}
	}

	if current != nil {
		merged = append(merged, *current)
	}

	return merged
}

func (c *SearchCaptionsCmd) printCaptions(captions []MergedCaption) {
	if len(captions) == 0 {
		fmt.Println("No captions found")
		return
	}

	fmt.Printf("%d captions\n", len(captions))
	lastPath := ""
	for _, cap := range captions {
		if cap.Path != lastPath {
			if lastPath != "" {
				fmt.Println()
			}
			displayTitle := cap.Path
			if cap.Title != "" {
				displayTitle = cap.Title + " - " + cap.Path
			}
			fmt.Printf("%s\n", displayTitle)
			lastPath = cap.Path
		}
		fmt.Printf("%s %s\n", utils.FormatDuration(int(cap.Time)), cap.Text)
	}
}

func (c *SearchCaptionsCmd) playCaptions(captions []MergedCaption) {
	for _, cap := range captions {
		fmt.Printf("Playing: %s at %s\n", cap.Path, utils.FormatDuration(int(cap.Time)))
		fmt.Printf("Text: %s\n", cap.Text)

		args := []string{
			"mpv",
			cap.Path,
			fmt.Sprintf("--start=%f", math.Max(0, cap.Time-2)),
			fmt.Sprintf("--end=%f", cap.End+1.5),
		}

		cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error playing %s: %v\n", cap.Path, err)
		}
	}
}
