package commands_test

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/commands"
)

func TestServeCmd_ParseFlags(t *testing.T) {
	cmd := &commands.ServeCmd{}

	tests := []struct {
		name     string
		query    url.Values
		validate func(*testing.T, *commands.ServeCmd, *http.Request)
	}{
		{
			name: "Search",
			query: url.Values{
				"search": {"word1 word2"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				if len(flags.Search) != 2 || flags.Search[0] != "word1" || flags.Search[1] != "word2" {
					t.Errorf("Unexpected search flags: %v", flags.Search)
				}
			},
		},
		{
			name: "Rating",
			query: url.Values{
				"rating": {"0"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				found := slices.Contains(flags.Where, "(score IS NULL OR score = 0)")
				if !found {
					t.Errorf("Expected rating 0 where clause, got: %v", flags.Where)
				}
			},
		},
		{
			name: "RatingNonZero",
			query: url.Values{
				"rating": {"5"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				found := slices.Contains(flags.Where, "score = 5")
				if !found {
					t.Errorf("Expected rating 5 where clause, got: %v", flags.Where)
				}
			},
		},
		{
			name: "SortRandom",
			query: url.Values{
				"sort": {"random"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				if !flags.Random {
					t.Error("Expected Random to be true")
				}
			},
		},
		{
			name: "MultiCategory",
			query: url.Values{
				"category": {"comedy", "music"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				if len(flags.Category) != 2 || flags.Category[0] != "comedy" || flags.Category[1] != "music" {
					t.Errorf("Unexpected category flags: %v", flags.Category)
				}
			},
		},
		{
			name: "MultiRating",
			query: url.Values{
				"rating": {"5", "4"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				found := slices.Contains(flags.Where, "(score = 5 OR score = 4)")
				if !found {
					t.Errorf("Expected multi-rating where clause, got: %v", flags.Where)
				}
			},
		},
		{
			name: "MultiBins",
			query: url.Values{
				"size":     {"+100", "-200"},
				"duration": {"10-60", "300"},
				"episodes": {"1", "5-10"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				if len(flags.Size) != 2 || flags.Size[0] != "+100" || flags.Size[1] != "-200" {
					t.Errorf("Unexpected size bins: %v", flags.Size)
				}
				if len(flags.Duration) != 2 || flags.Duration[0] != "10-60" || flags.Duration[1] != "300" {
					t.Errorf("Unexpected duration bins: %v", flags.Duration)
				}
				if flags.FileCounts != "1,5-10" {
					t.Errorf("Unexpected episodes bins: %s", flags.FileCounts)
				}
			},
		},
		{
			name: "Percentiles",
			query: url.Values{
				"size":     {"p10-50"},
				"duration": {"p20-30"},
				"episodes": {"p1-1"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				if len(flags.Size) == 0 || flags.Size[0] != "p10-50" {
					t.Errorf("Unexpected size percentile: %v", flags.Size)
				}
				if len(flags.Duration) == 0 || flags.Duration[0] != "p20-30" {
					t.Errorf("Unexpected duration percentile: %v", flags.Duration)
				}
				if flags.FileCounts != "p1-1" {
					t.Errorf("Unexpected episodes percentile: %s", flags.FileCounts)
				}
			},
		},
		{
			name: "Ranges",
			query: url.Values{
				"min_size":     {"100"},
				"max_size":     {"500"},
				"min_duration": {"10"},
				"max_duration": {"60"},
				"min_score":    {"3"},
				"max_score":    {"5"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				if len(flags.Size) != 2 || flags.Size[0] != ">100MB" || flags.Size[1] != "<500MB" {
					t.Errorf("Unexpected size flags: %v", flags.Size)
				}
				if len(flags.Duration) != 2 || flags.Duration[0] != ">10min" || flags.Duration[1] != "<60min" {
					t.Errorf("Unexpected duration flags: %v", flags.Duration)
				}
				foundMinScore := false
				foundMaxScore := false
				for _, w := range flags.Where {
					if w == "score >= 3" {
						foundMinScore = true
					}
					if w == "score <= 5" {
						foundMaxScore = true
					}
				}
				if !foundMinScore || !foundMaxScore {
					t.Error("Expected score where clauses")
				}
			},
		},
		{
			name: "TypeFilters",
			query: url.Values{
				"video":      {"true"},
				"audio":      {"true"},
				"image":      {"true"},
				"text":       {"true"},
				"unplayed":   {"true"},
				"watched":    {"true"},
				"unfinished": {"true"},
				"completed":  {"true"},
				"trash":      {"true"},
			},
			validate: func(t *testing.T, c *commands.ServeCmd, r *http.Request) {
				flags := c.ParseFlags(r)
				if !flags.VideoOnly || !flags.AudioOnly || !flags.ImageOnly || !flags.TextOnly {
					t.Error("Expected media_type filters to be true")
				}
				if flags.Watched == nil || !*flags.Watched {
					t.Error("Expected Watched to be true")
				}
				if !flags.Unfinished || !flags.Completed || !flags.OnlyDeleted {
					t.Error("Expected playback/trash filters to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &url.URL{RawQuery: tt.query.Encode()}
			req := &http.Request{URL: u}
			tt.validate(t, cmd, req)
		})
	}
}
