package commands

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/formatutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type SearchCaptionsCmd struct {
	models.CoreFlags          `embed:""`
	models.FTSFlags           `embed:""`
	models.CaptionFilterFlags `embed:""`
	models.PlaybackFlags      `embed:""`

	Database string   `help:"SQLite database file"                    required:"true" arg:"" type:"existingfile"`
	Search   []string `help:"Search terms"                            required:"true" arg:""`
	Open     bool     `help:"Open results in media player"`
	Overlap  int      `help:"Overlap in seconds for merging captions"                                            default:"8"`
	Limit    int      `help:"Limit results"                                                                      default:"100" short:"L"`
	All      bool     `help:"Return all results (no limit)"                                                                    short:"a"`
	Offset   int      `help:"Skip N merged captions"`
}

type MergedCaption struct {
	Path  string
	Time  float64
	End   float64
	Text  string
	Title string
}

func (c *SearchCaptionsCmd) Run(ctx context.Context) error {
	logging.SetupLogging(c.Verbose)

	sqlDB, err := db.Connect(ctx, c.Database)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	queries := db.New(sqlDB)
	if schemaErr := requireCaptionSchema(ctx, sqlDB); schemaErr != nil {
		return schemaErr
	}
	queryStr := strings.Join(c.Search, " ")

	limit := int64(c.Limit)
	if c.All {
		limit = 1000000
	} else {
		if limit <= 0 {
			limit = 100
		}
		if c.Offset > 0 {
			// The captions query has no OFFSET; fetch enough rows so the offset
			// can be applied after merging.
			limit += int64(c.Offset)
		}
	}

	captionParams := db.SearchCaptionsParams{
		Query:     queryStr,
		VideoOnly: c.VideoOnly,
		AudioOnly: c.AudioOnly,
		ImageOnly: c.ImageOnly,
		TextOnly:  c.TextOnly,
		Limit:     limit,
	}
	var rows []db.SearchCaptionsRow
	if c.NoFTS {
		rows, err = queries.SearchCaptionsSubstring(ctx, captionParams)
	} else {
		rows, err = queries.SearchCaptions(ctx, captionParams)
	}
	if err != nil {
		return err
	}

	merged := c.mergeCaptions(rows)
	merged = applyOffset(merged, c.Offset)

	if c.Open {
		c.playCaptions(ctx, merged)
		return nil
	}

	c.printCaptions(merged)
	return nil
}

func applyOffset(captions []MergedCaption, offset int) []MergedCaption {
	if offset <= 0 {
		return captions
	}
	if offset >= len(captions) {
		return nil
	}
	return captions[offset:]
}

func (c *SearchCaptionsCmd) getEnd(t float64, text string) float64 {
	// Formula from original python: caption["time"] + (len(caption["text"]) / 4.2 / 220 * 60)
	return t + (float64(len(text)) / 4.2 / 220 * 60)
}

func (c *SearchCaptionsCmd) mergeCaptions(rows []db.SearchCaptionsRow) []MergedCaption {
	if len(rows) == 0 {
		return nil
	}

	// FTS returns rows in rank order, so matches from the same file are not
	// necessarily adjacent. Group by path first, then merge each file's hits in
	// timestamp order while preserving the rank order of files.
	order := make([]string, 0)
	byPath := make(map[string][]db.SearchCaptionsRow)
	for _, row := range rows {
		if _, ok := byPath[row.MediaPath]; !ok {
			order = append(order, row.MediaPath)
		}
		byPath[row.MediaPath] = append(byPath[row.MediaPath], row)
	}

	var merged []MergedCaption
	for _, path := range order {
		merged = append(merged, c.mergeCaptionGroup(byPath[path])...)
	}

	return merged
}

func (c *SearchCaptionsCmd) mergeCaptionGroup(group []db.SearchCaptionsRow) []MergedCaption {
	sort.SliceStable(group, func(i, j int) bool {
		left, right := 0.0, 0.0
		if group[i].Time.Valid {
			left = group[i].Time.Float64
		}
		if group[j].Time.Valid {
			right = group[j].Time.Float64
		}
		return left < right
	})

	var merged []MergedCaption
	var current *MergedCaption
	for _, row := range group {
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

		if math.Abs(timeVal-current.End) <= float64(c.Overlap) ||
			math.Abs(timeVal-current.Time) <= float64(c.Overlap) {

			current.End = max(current.End, end)
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
	for _, caption := range captions {
		if caption.Path != lastPath {
			if lastPath != "" {
				fmt.Println()
			}
			displayTitle := caption.Path
			if caption.Title != "" {
				displayTitle = caption.Title + " - " + caption.Path
			}
			fmt.Printf("%s\n", displayTitle)
			lastPath = caption.Path
		}
		fmt.Printf("%s %s\n", formatutil.FormatDuration(int(caption.Time)), caption.Text)
	}
}

func (c *SearchCaptionsCmd) playCaptions(ctx context.Context, captions []MergedCaption) {
	for _, caption := range captions {
		fmt.Printf("Playing: %s at %s\n", caption.Path, formatutil.FormatDuration(int(caption.Time)))
		fmt.Printf("Text: %s\n", caption.Text)

		args := c.buildPlayerArgs(caption)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error playing %s: %v\n", caption.Path, err)
		}
	}
}

func (c *SearchCaptionsCmd) buildPlayerArgs(caption MergedCaption) []string {
	player := c.OverridePlayer
	if player == "" {
		player = "mpv"
	}
	args := []string{player}

	if player == "mpv" {
		if c.Volume > 0 {
			args = append(args, fmt.Sprintf("--volume=%d", c.Volume))
		}
		if c.Mute {
			args = append(args, "--mute=yes")
		}
		if c.Fullscreen {
			args = append(args, "--fullscreen")
		}
		if c.Loop {
			args = append(args, "--loop-file=inf")
		}
		if c.Speed > 0 && c.Speed != 1.0 {
			args = append(args, fmt.Sprintf("--speed=%.2f", c.Speed))
		}
		if !c.SavePlayhead {
			args = append(args, "--no-resume-playback")
		}
		if c.WatchLaterDir != "" {
			args = append(args, "--watch-later-directory="+c.WatchLaterDir)
		}
		if c.MpvSocket != "" {
			args = append(args, "--input-ipc-server="+c.MpvSocket)
		}

		useSubs := !c.NoSubtitles
		if useSubs && c.SubtitleMix > 0 && utils.RandomFloat() < c.SubtitleMix {
			useSubs = false
		}
		if useSubs {
			args = append(args, c.PlayerArgsSub...)
		} else {
			args = append(args, "--no-sub")
			args = append(args, c.PlayerArgsNoSub...)
		}
	}

	start := fmt.Sprintf("%f", math.Max(0, caption.Time-2))
	end := fmt.Sprintf("%f", caption.End+1.5)
	if c.Start != "" {
		start = c.Start
	}
	if c.End != "" {
		end = c.End
	}
	args = append(args, "--start="+start, "--end="+end, caption.Path)
	return args
}

// requireCaptionSchema returns a clear error when a database was created
// without captions/FTS instead of leaking a raw "no such table" SQL error.
func requireCaptionSchema(ctx context.Context, sqlDB *sql.DB) error {
	for _, table := range []string{"captions", "captions_fts"} {
		var exists bool
		if err := sqlDB.QueryRowContext(
			ctx,
			"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?)",
			table,
		).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf(
				"caption search is unavailable: table %q is missing; this database was created without FTS support",
				table,
			)
		}
	}
	return nil
}
