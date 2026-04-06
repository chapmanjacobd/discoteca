package models

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

// CoreFlags are essential flags shared across most binaries/commands
type CoreFlags struct {
	// Common options
	Verbose   int    `help:"Enable verbose logging (-v for info, -vv for debug)" short:"v" env:"DISCO_VERBOSE" type:"counter"`
	Simulate  bool   `help:"Dry run; don't actually do anything"`
	DryRun    bool   `                                                                                                        kong:"-"` // Alias for Simulate
	NoConfirm bool   `help:"Don't ask for confirmation"                          short:"y"`
	Yes       bool   `                                                                                                        kong:"-"` // Alias for NoConfirm
	Timeout   string `help:"Quit after N minutes/seconds"                        short:"T"`
}

type QueryFlags struct {
	Query  string `help:"Raw SQL query (overrides all query building)" short:"q" group:"Query"`
	Limit  int    `help:"Limit results per database"                   short:"L" group:"Query" default:"100"`
	All    bool   `help:"Return all results (no limit)"                short:"a" group:"Query"`
	Offset int    `help:"Skip N results"                                         group:"Query"`
}

type PathFilterFlags struct {
	Include      []string `help:"Include paths matching pattern"      short:"s" group:"PathFilter"`
	Exclude      []string `help:"Exclude paths matching pattern"      short:"E" group:"PathFilter"`
	Regex        string   `help:"Filter paths by regex pattern"                 group:"PathFilter"`
	PathContains []string `help:"Path must contain all these strings"           group:"PathFilter"`
	Paths        []string `help:"Exact paths to include"                        group:"PathFilter"`
}

type FilterFlags struct {
	Search           []string `help:"Search terms (space-separated for AND, | for OR)"                       group:"Filter"`
	Size             []string `help:"Size range (e.g., >100MB, 1GB%10)"                                      group:"Filter" short:"S"`
	Duration         []string `help:"Duration range (e.g., >1hour, 30min%10)"                                group:"Filter" short:"d"`
	Modified         []string `help:"Filter by modification time"                                            group:"Filter"`
	Created          []string `help:"Filter by creation time"                                                group:"Filter"`
	Downloaded       []string `help:"Filter by download time"                                                group:"Filter"`
	DurationFromSize string   `help:"Constrain media to duration of videos which match any size constraints" group:"Filter"`
	Watched          *bool    `help:"Filter by watched status (true/false)"                                  group:"Filter"`
	Unfinished       bool     `help:"Has playhead but not finished"                                          group:"Filter"`
	Partial          string   `help:"Filter by partial playback status"                                      group:"Filter" short:"P"`
	PlayCountMin     int      `help:"Minimum play count"                                                     group:"Filter"`
	PlayCountMax     int      `help:"Maximum play count"                                                     group:"Filter"`
	Completed        bool     `help:"Show only completed items"                                              group:"Filter"`
	InProgress       bool     `help:"Show only items in progress"                                            group:"Filter"`
	WithCaptions     bool     `help:"Show only items with captions"                                          group:"Filter"`
	FlexibleSearch   bool     `help:"Flexible search (fuzzy)"                                                group:"Filter"`
	Exact            bool     `help:"Exact match for search"                                                 group:"Filter"`
	Where            []string `help:"SQL where clause(s)"                                                    group:"Filter" short:"w"`
	Exists           bool     `help:"Filter out non-existent files"                                          group:"Filter"`
	FetchSiblings    string   `help:"Fetch siblings of matched files (each, all, if-audiobook)"              group:"Filter" short:"o"`
	FetchSiblingsMax int      `help:"Maximum number of siblings to fetch"                                    group:"Filter"`
}

type MediaFilterFlags struct {
	Category        []string `help:"Filter by category"                         group:"MediaFilter"`
	Genre           string   `help:"Filter by genre"                            group:"MediaFilter"`
	Language        []string `help:"Filter by language"                         group:"MediaFilter"`
	Ext             []string `help:"Filter by extensions (e.g., .mp4,.mkv)"     group:"MediaFilter" short:"e"`
	VideoOnly       bool     `help:"Only video files"                           group:"MediaFilter"`
	AudioOnly       bool     `help:"Only audio files"                           group:"MediaFilter"`
	ImageOnly       bool     `help:"Only image files"                           group:"MediaFilter"`
	TextOnly        bool     `help:"Only text/ebook files"                      group:"MediaFilter"`
	Portrait        bool     `help:"Only portrait orientation files"            group:"MediaFilter"`
	ScanSubtitles   bool     `help:"Scan for external subtitles during import"  group:"MediaFilter"`
	OnlineMediaOnly bool     `help:"Exclude local media"                        group:"MediaFilter"`
	LocalMediaOnly  bool     `help:"Exclude online media"                       group:"MediaFilter"`
	ProbeImages     bool     `help:"Run ffprobe on image files (default: skip)" group:"MediaFilter"`
}

type TimeFilterFlags struct {
	CreatedAfter     string `help:"Created after date (YYYY-MM-DD)"      group:"Time"`
	CreatedBefore    string `help:"Created before date (YYYY-MM-DD)"     group:"Time"`
	ModifiedAfter    string `help:"Modified after date (YYYY-MM-DD)"     group:"Time"`
	ModifiedBefore   string `help:"Modified before date (YYYY-MM-DD)"    group:"Time"`
	DownloadedAfter  string `help:"Downloaded after date (YYYY-MM-DD)"   group:"Time"`
	DownloadedBefore string `help:"Downloaded before date (YYYY-MM-DD)"  group:"Time"`
	DeletedAfter     string `help:"Deleted after date (YYYY-MM-DD)"      group:"Time"`
	DeletedBefore    string `help:"Deleted before date (YYYY-MM-DD)"     group:"Time"`
	PlayedAfter      string `help:"Last played after date (YYYY-MM-DD)"  group:"Time"`
	PlayedBefore     string `help:"Last played before date (YYYY-MM-DD)" group:"Time"`
}

type DatabaseFlags struct {
	Databases []string `help:"Specific database paths to query (must be in server's allowed list). Can be specified multiple times." group:"Database"`
}

type DeletedFlags struct {
	HideDeleted bool `help:"Exclude deleted files from results"    default:"true" group:"Deleted"`
	OnlyDeleted bool `help:"Include only deleted files in results"                group:"Deleted"`
}

type SortFlags struct {
	SortBy  string `help:"Sort by field"                                                              default:"path" short:"u" group:"Sort"`
	Reverse bool   `help:"Reverse sort order"                                                                        short:"V" group:"Sort"`
	NatSort bool   `help:"Use natural sorting"                                                                       short:"n" group:"Sort"`
	Random  bool   `help:"Random order"                                                                              short:"r" group:"Sort"`
	ReRank  string `help:"Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)"                short:"k" group:"Sort" alias:"rerank"`
}

type DisplayFlags struct {
	Columns   []string `help:"Columns to display"                                                  short:"c" group:"Display"`
	JSON      bool     `help:"Output results as JSON"                                              short:"j" group:"Display"`
	Summarize bool     `help:"Print aggregate statistics"                                                    group:"Display"`
	Frequency string   `help:"Group statistics by time frequency (daily, weekly, monthly, yearly)" short:"f" group:"Display"`
}

type AggregateFlags struct {
	BigDirs           bool     `help:"Aggregate by parent directory"                           short:"B" group:"Aggregate"`
	FileCounts        string   `help:"Filter by number of files in directory (e.g., >5, 10%1)"           group:"Aggregate"`
	GroupByExtensions bool     `help:"Group by file extensions"                                          group:"Aggregate"`
	GroupBySize       bool     `help:"Group by size buckets"                                             group:"Aggregate"`
	GroupByParent     bool     `help:"Group media by parent directory with counts and totals"            group:"Aggregate"`
	Depth             int      `help:"Aggregate at specific directory depth"                   short:"D" group:"Aggregate"`
	MinDepth          int      `help:"Minimum depth for aggregation"                                     group:"Aggregate" default:"0"`
	MaxDepth          int      `help:"Maximum depth for aggregation"                                     group:"Aggregate"`
	Parents           bool     `help:"Include parent directories in aggregation"                         group:"Aggregate"`
	FoldersOnly       bool     `help:"Only show folders"                                                 group:"Aggregate"`
	FilesOnly         bool     `help:"Only show files"                                                   group:"Aggregate"`
	FolderSizes       []string `help:"Filter folders by total size"                                      group:"Aggregate"`
	FolderCounts      string   `help:"Filter folders by number of subfolders"                            group:"Aggregate"`
}

type TextFlags struct {
	RegexSort  bool     `help:"Sort by splitting lines and sorting words" alias:"rs" group:"Text"`
	Regexs     []string `help:"Regex patterns for line splitting"         alias:"re" group:"Text"`
	WordSorts  []string `help:"Word sorting strategies"                              group:"Text"`
	LineSorts  []string `help:"Line sorting strategies"                              group:"Text"`
	Compat     bool     `help:"Use natsort compat mode"                              group:"Text"`
	Preprocess bool     `help:"Remove junk common to filenames and URLs"             group:"Text" default:"true"`
	StopWords  []string `help:"List of words to ignore"                              group:"Text"`
	Duplicates *bool    `help:"Filter for duplicate words (true/false)"              group:"Text"`
	UniqueOnly *bool    `help:"Filter for unique words (true/false)"                 group:"Text"`
}

type SimilarityFlags struct {
	Similar         bool    `help:"Find similar files or folders"                group:"Similarity"`
	SizesDelta      float64 `help:"Size difference threshold (%)"                group:"Similarity" default:"10.0"`
	CountsDelta     float64 `help:"File count difference threshold (%)"          group:"Similarity" default:"3.0"`
	DurationsDelta  float64 `help:"Duration difference threshold (%)"            group:"Similarity" default:"5.0"`
	FilterNames     bool    `help:"Cluster by name similarity"                   group:"Similarity"`
	FilterSizes     bool    `help:"Cluster by size similarity"                   group:"Similarity"`
	FilterCounts    bool    `help:"Cluster by count similarity"                  group:"Similarity"`
	FilterDurations bool    `help:"Cluster by duration similarity"               group:"Similarity"`
	TotalSizes      bool    `help:"Compare total sizes (folders only)"           group:"Similarity"`
	TotalDurations  bool    `help:"Compare total durations (folders only)"       group:"Similarity"`
	OnlyDuplicates  bool    `help:"Only show duplicate items"                    group:"Similarity"`
	OnlyOriginals   bool    `help:"Only show original items"                     group:"Similarity"`
	ClusterSort     bool    `help:"Group items by similarity"                    group:"Similarity"                short:"C"`
	Clusters        int     `help:"Number of clusters"                           group:"Similarity"`
	TFIDF           bool    `help:"Use TF-IDF for clustering"                    group:"Similarity"`
	MoveGroups      bool    `help:"Move grouped files into separate directories" group:"Similarity"`
	PrintGroups     bool    `help:"Print clusters as JSON"                       group:"Similarity"`
}

type DedupeFlags struct {
	Audio              bool    `help:"Dedupe database by artist + album + title"                           group:"Dedupe"`
	ExtractorID        bool    `help:"Dedupe database by extractor_id"                                     group:"Dedupe" alias:"id"`
	TitleOnly          bool    `help:"Dedupe database by title"                                            group:"Dedupe"`
	DurationOnly       bool    `help:"Dedupe database by duration"                                         group:"Dedupe"`
	Filesystem         bool    `help:"Dedupe filesystem database (hash)"                                   group:"Dedupe" alias:"fs"`
	CompareDirs        bool    `help:"Compare directories"                                                 group:"Dedupe"`
	Basename           bool    `help:"Match by basename similarity"                                        group:"Dedupe"`
	Dirname            bool    `help:"Match by dirname similarity"                                         group:"Dedupe"`
	MinSimilarityRatio float64 `help:"Filter out matches with less than this ratio (0.7-0.9)"              group:"Dedupe"            default:"0.8"`
	DedupeCmd          string  `help:"Command to run for deduplication (rmlint-style: cmd duplicate keep)" group:"Dedupe"`
}

type FTSFlags struct {
	FTS      bool   `help:"Use FTS5 full-text search"                           group:"FTS"`
	FTSTable string `help:"FTS table name"                                      group:"FTS" default:"media_fts"`
	NoFTS    bool   `help:"Disable full-text search, use substring search only" group:"FTS"`
	Related  int    `help:"Find media related to the first result"              group:"FTS"                     short:"R"`
}

type PlaybackFlags struct {
	PlayInOrder           string   `help:"Play media in order"                                      default:"natural_ps" short:"O" group:"Playback"`
	NoPlayInOrder         bool     `help:"Don't play media in order"                                                               group:"Playback"`
	Loop                  bool     `help:"Loop playback"                                                                           group:"Playback"`
	Mute                  bool     `help:"Start playback muted"                                                          short:"M" group:"Playback"`
	OverridePlayer        string   `help:"Override default player (e.g. --player 'vlc')"                                           group:"Playback"`
	Start                 string   `help:"Start playback at specific time/percentage"                                              group:"Playback"`
	End                   string   `help:"Stop playback at specific time/percentage"                                               group:"Playback"`
	Volume                int      `help:"Set initial volume (0-100)"                                                              group:"Playback"`
	Fullscreen            bool     `help:"Start in fullscreen"                                                                     group:"Playback"`
	NoSubtitles           bool     `help:"Disable subtitles"                                                                       group:"Playback"`
	SubtitleMix           float64  `help:"Probability to play no-subtitle content"                  default:"0.35"                 group:"Playback"`
	InterdimensionalCable int      `help:"Duration to play (in seconds) while changing the channel"                      short:"4" group:"Playback" alias:"4dtv"`
	Speed                 float64  `help:"Playback speed"                                           default:"1.0"                  group:"Playback"`
	SavePlayhead          bool     `help:"Save playback position on quit"                           default:"true"                 group:"Playback"`
	MpvSocket             string   `help:"Mpv socket path"                                                                         group:"Playback"`
	WatchLaterDir         string   `help:"Mpv watch_later directory"                                                               group:"Playback"`
	PlayerArgsSub         []string `help:"Player arguments for videos with subtitles"                                              group:"Playback"`
	PlayerArgsNoSub       []string `help:"Player arguments for videos without subtitles"                                           group:"Playback"`
	Cast                  bool     `help:"Cast to chromecast groups"                                                               group:"Playback"`
	CastDevice            string   `help:"Chromecast device name"                                                                  group:"Playback" alias:"cast-to"`
	CastWithLocal         bool     `help:"Play music locally at the same time as chromecast"                                       group:"Playback"`
}

type MpvActionFlags struct {
	Cmd0        string `help:"Command to run if mpv exits with code 0"    group:"MpvAction"`
	Cmd1        string `help:"Command to run if mpv exits with code 1"    group:"MpvAction"`
	Cmd2        string `help:"Command to run if mpv exits with code 2"    group:"MpvAction"`
	Cmd3        string `help:"Command to run if mpv exits with code 3"    group:"MpvAction"`
	Cmd4        string `help:"Command to run if mpv exits with code 4"    group:"MpvAction"`
	Cmd5        string `help:"Command to run if mpv exits with code 5"    group:"MpvAction"`
	Cmd6        string `help:"Command to run if mpv exits with code 6"    group:"MpvAction"`
	Cmd7        string `help:"Command to run if mpv exits with code 7"    group:"MpvAction"`
	Cmd8        string `help:"Command to run if mpv exits with code 8"    group:"MpvAction"`
	Cmd9        string `help:"Command to run if mpv exits with code 9"    group:"MpvAction"`
	Cmd10       string `help:"Command to run if mpv exits with code 10"   group:"MpvAction"`
	Cmd11       string `help:"Command to run if mpv exits with code 11"   group:"MpvAction"`
	Cmd12       string `help:"Command to run if mpv exits with code 12"   group:"MpvAction"`
	Cmd13       string `help:"Command to run if mpv exits with code 13"   group:"MpvAction"`
	Cmd14       string `help:"Command to run if mpv exits with code 14"   group:"MpvAction"`
	Cmd15       string `help:"Command to run if mpv exits with code 15"   group:"MpvAction"`
	Cmd20       string `help:"Command to run if mpv exits with code 20"   group:"MpvAction"`
	Cmd127      string `help:"Command to run if mpv exits with code 127"  group:"MpvAction"`
	Interactive bool   `help:"Interactive decision making after playback" group:"MpvAction" short:"I"`
}

type PostActionFlags struct {
	Trash        bool   `help:"Trash files after action"                            group:"PostAction"`
	PostAction   string `help:"Post-action: none, delete, mark-deleted, move, copy" group:"PostAction"`
	DeleteFiles  bool   `help:"Delete files after action"                           group:"PostAction"`
	DeleteRows   bool   `help:"Delete rows from database"                           group:"PostAction"`
	MarkDeleted  bool   `help:"Mark as deleted in database"                         group:"PostAction"`
	MoveTo       string `help:"Move files to directory"                             group:"PostAction"`
	CopyTo       string `help:"Copy files to directory"                             group:"PostAction"`
	ActionLimit  int    `help:"Stop after N files"                                  group:"PostAction"`
	ActionSize   string `help:"Stop after N bytes (e.g., 10GB)"                     group:"PostAction"`
	TrackHistory bool   `help:"Track playback history"                              group:"PostAction" default:"true"`
}

type HashingFlags struct {
	HashGap       float64 `help:"Gap between segments (0.0-1.0 as percentage of file size, or absolute bytes if >1)" default:"0.1" group:"Hashing"`
	HashChunkSize int64   `help:"Size of each segment to hash"                                                                     group:"Hashing"`
	HashThreads   int     `help:"Number of threads to use for hashing a single file"                                 default:"1"   group:"Hashing"`
}

type MergeFlags struct {
	OnlyTables        []string `help:"Comma separated specific table(s)"       short:"t" group:"Merge"`
	PrimaryKeys       []string `help:"Comma separated primary keys"                      group:"Merge"`
	BusinessKeys      []string `help:"Comma separated business keys"                     group:"Merge"`
	Upsert            bool     `help:"Upsert rows on conflict"                           group:"Merge"`
	Ignore            bool     `help:"Ignore rows on conflict (only-new-rows)"           group:"Merge"`
	OnlyNewRows       bool     `                                                                       kong:"-"` // Alias for Ignore
	OnlyTargetColumns bool     `help:"Only copy columns that exist in target"            group:"Merge"`
	SkipColumns       []string `help:"Columns to skip during merge"                      group:"Merge"`
}

// GlobalFlags are flags available to disco data commands (print, search, du, etc)
// This struct is used for passing flags to query and utility functions.
// Command structs should embed only the flag structs they need.
type GlobalFlags struct {
	CoreFlags        `embed:""`
	QueryFlags       `embed:""`
	PathFilterFlags  `embed:""`
	FilterFlags      `embed:""`
	MediaFilterFlags `embed:""`
	TimeFilterFlags  `embed:""`
	DeletedFlags     `embed:""`
	SortFlags        `embed:""`
	DisplayFlags     `embed:""`
	AggregateFlags   `embed:""`
	TextFlags        `embed:""`
	SimilarityFlags  `embed:""`
	DedupeFlags      `embed:""`
	FTSFlags         `embed:""`
	PlaybackFlags    `embed:""`
	MpvActionFlags   `embed:""`
	PostActionFlags  `embed:""`
	HashingFlags     `embed:""`
	MergeFlags       `embed:""`
	DatabaseFlags    `embed:""`

	Threads      int  `help:"Use N threads for parallel processing"`
	IgnoreErrors bool `help:"Ignore errors and continue to next file" short:"i"`
}

// ControlFlags are a subset of flags for simple control commands
type ControlFlags struct {
	MpvSocket  string `help:"Mpv socket path"                                     group:"Playback"`
	CastDevice string `help:"Chromecast device name"                              group:"Playback" alias:"cast-to"`
	Verbose    int    `help:"Enable verbose logging (-v for info, -vv for debug)"                                  short:"v" env:"DISCO_VERBOSE" type:"counter"`
}

func (c *CoreFlags) AfterApply() error {
	if c.Simulate {
		c.DryRun = true
	}
	if c.NoConfirm {
		c.Yes = true
	}
	return nil
}

func (m *MediaFilterFlags) AfterApply() error {
	if m.Ext != nil {
		for i, ext := range m.Ext {
			if !strings.HasPrefix(ext, ".") {
				m.Ext[i] = "." + ext
			}
		}
	}
	return nil
}

func (m *MergeFlags) AfterApply() error {
	if m.Ignore {
		m.OnlyNewRows = true
	}
	return nil
}

// Logger interface for structured logging
type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

var Log Logger

func SetupLogging(verbosity int) {
	// Create handler with appropriate level
	var level slog.Level
	if verbosity >= 2 {
		level = slog.LevelDebug
		db.SetDebugMode(true)
	} else if verbosity == 1 {
		level = slog.LevelInfo
	} else {
		// Default to Warn (hides Info and Debug, shows Warn and Error)
		level = slog.LevelWarn
	}

	// Use a simple slog logger as default
	handler := &plainHandler{
		level: level,
		out:   os.Stderr,
	}
	logger := slog.New(handler)
	Log = &slogLogger{logger}
}

// plainHandler implements [slog.Handler] with plain output
type plainHandler struct {
	level slog.Level
	out   io.Writer
	attrs []slog.Attr
}

func (h *plainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *plainHandler) Handle(_ context.Context, r slog.Record) error {
	var msg strings.Builder
	msg.WriteString(r.Message)
	for _, a := range h.attrs {
		fmt.Fprintf(&msg, "\n    %s=%v", a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&msg, "\n    %s=%v", a.Key, a.Value.Any())
		return true
	})
	_, err := fmt.Fprintln(h.out, msg.String())
	return err
}

func (h *plainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &plainHandler{
		level: h.level,
		out:   h.out,
		attrs: append(h.attrs, attrs...),
	}
}

func (h *plainHandler) WithGroup(_ string) slog.Handler {
	return h
}

// slogLogger wraps [slog.Logger] and implements Logger interface
type slogLogger struct {
	*slog.Logger
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{l.Logger.With(args...)}
}

// BuildQueryOptions contains all flags needed to build GlobalFlags
type BuildQueryOptions struct {
	Core        CoreFlags
	Query       QueryFlags
	PathFilter  PathFilterFlags
	Filter      FilterFlags
	MediaFilter MediaFilterFlags
	TimeFilter  TimeFilterFlags
	Deleted     DeletedFlags
	Sort        SortFlags
	Display     DisplayFlags
	FTS         FTSFlags
}

// BuildQueryGlobalFlags constructs GlobalFlags for query-based commands
// This reduces boilerplate in command Run() methods
func BuildQueryGlobalFlags(opts BuildQueryOptions) GlobalFlags {
	return GlobalFlags{
		CoreFlags:        opts.Core,
		QueryFlags:       opts.Query,
		PathFilterFlags:  opts.PathFilter,
		FilterFlags:      opts.Filter,
		MediaFilterFlags: opts.MediaFilter,
		TimeFilterFlags:  opts.TimeFilter,
		DeletedFlags:     opts.Deleted,
		SortFlags:        opts.Sort,
		DisplayFlags:     opts.Display,
		FTSFlags:         opts.FTS,
	}
}
