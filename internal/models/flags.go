package models

import (
	"log/slog"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

// CoreFlags are essential flags shared across most binaries/commands
type CoreFlags struct {
	// Common options
	Verbose   bool   `short:"v" help:"Enable verbose logging"`
	Simulate  bool   `help:"Dry run; don't actually do anything"`
	DryRun    bool   `kong:"-"` // Alias for Simulate
	NoConfirm bool   `short:"y" help:"Don't ask for confirmation"`
	Yes       bool   `kong:"-"` // Alias for NoConfirm
	Timeout   string `short:"T" help:"Quit after N minutes/seconds"`
}

type QueryFlags struct {
	Query  string `short:"q" help:"Raw SQL query (overrides all query building)" group:"Query"`
	Limit  int    `short:"L" default:"100" help:"Limit results per database" group:"Query"`
	All    bool   `short:"a" help:"Return all results (no limit)" group:"Query"`
	Offset int    `help:"Skip N results" group:"Query"`
}

type PathFilterFlags struct {
	Include      []string `short:"s" help:"Include paths matching pattern" group:"PathFilter"`
	Exclude      []string `short:"E" help:"Exclude paths matching pattern" group:"PathFilter"`
	Regex        string   `help:"Filter paths by regex pattern" group:"PathFilter"`
	PathContains []string `help:"Path must contain all these strings" group:"PathFilter"`
	Paths        []string `help:"Exact paths to include" group:"PathFilter"`
}

type FilterFlags struct {
	Search           []string `help:"Search terms (space-separated for AND, | for OR)" group:"Filter"`
	Size             []string `short:"S" help:"Size range (e.g., >100MB, 1GB%10)" group:"Filter"`
	Duration         []string `short:"d" help:"Duration range (e.g., >1hour, 30min%10)" group:"Filter"`
	Modified         []string `help:"Filter by modification time" group:"Filter"`
	Created          []string `help:"Filter by creation time" group:"Filter"`
	Downloaded       []string `help:"Filter by download time" group:"Filter"`
	DurationFromSize string   `help:"Constrain media to duration of videos which match any size constraints" group:"Filter"`
	Watched          *bool    `help:"Filter by watched status (true/false)" group:"Filter"`
	Unfinished       bool     `help:"Has playhead but not finished" group:"Filter"`
	Partial          string   `short:"P" help:"Filter by partial playback status" group:"Filter"`
	PlayCountMin     int      `help:"Minimum play count" group:"Filter"`
	PlayCountMax     int      `help:"Maximum play count" group:"Filter"`
	Completed        bool     `help:"Show only completed items" group:"Filter"`
	InProgress       bool     `help:"Show only items in progress" group:"Filter"`
	WithCaptions     bool     `help:"Show only items with captions" group:"Filter"`
	FlexibleSearch   bool     `help:"Flexible search (fuzzy)" group:"Filter"`
	Exact            bool     `help:"Exact match for search" group:"Filter"`
	Where            []string `short:"w" help:"SQL where clause(s)" group:"Filter"`
	Exists           bool     `help:"Filter out non-existent files" group:"Filter"`
	FetchSiblings    string   `short:"o" help:"Fetch siblings of matched files (each, all, if-audiobook)" group:"Filter"`
	FetchSiblingsMax int      `help:"Maximum number of siblings to fetch" group:"Filter"`
}

type MediaFilterFlags struct {
	Category            []string `help:"Filter by category" group:"MediaFilter"`
	Genre               string   `help:"Filter by genre" group:"MediaFilter"`
	Language            []string `help:"Filter by language" group:"MediaFilter"`
	Ext                 []string `short:"e" help:"Filter by extensions (e.g., .mp4,.mkv)" group:"MediaFilter"`
	VideoOnly           bool     `help:"Only video files" group:"MediaFilter"`
	AudioOnly           bool     `help:"Only audio files" group:"MediaFilter"`
	ImageOnly           bool     `help:"Only image files" group:"MediaFilter"`
	TextOnly            bool     `help:"Only text/ebook files" group:"MediaFilter"`
	Portrait            bool     `help:"Only portrait orientation files" group:"MediaFilter"`
	ScanSubtitles       bool     `help:"Scan for external subtitles during import" group:"MediaFilter"`
	OnlineMediaOnly     bool     `help:"Exclude local media" group:"MediaFilter"`
	LocalMediaOnly      bool     `help:"Exclude online media" group:"MediaFilter"`
	MimeType            []string `help:"Filter by mimetype substring (e.g., video, mp4)" group:"MediaFilter"`
	NoMimeType          []string `help:"Exclude by mimetype substring" group:"MediaFilter"`
	NoDefaultCategories bool     `help:"Disable default categories" group:"MediaFilter"`
}

type TimeFilterFlags struct {
	CreatedAfter     string `help:"Created after date (YYYY-MM-DD)" group:"Time"`
	CreatedBefore    string `help:"Created before date (YYYY-MM-DD)" group:"Time"`
	ModifiedAfter    string `help:"Modified after date (YYYY-MM-DD)" group:"Time"`
	ModifiedBefore   string `help:"Modified before date (YYYY-MM-DD)" group:"Time"`
	DownloadedAfter  string `help:"Downloaded after date (YYYY-MM-DD)" group:"Time"`
	DownloadedBefore string `help:"Downloaded before date (YYYY-MM-DD)" group:"Time"`
	DeletedAfter     string `help:"Deleted after date (YYYY-MM-DD)" group:"Time"`
	DeletedBefore    string `help:"Deleted before date (YYYY-MM-DD)" group:"Time"`
	PlayedAfter      string `help:"Last played after date (YYYY-MM-DD)" group:"Time"`
	PlayedBefore     string `help:"Last played before date (YYYY-MM-DD)" group:"Time"`
}

type DatabaseFlags struct {
	Databases []string `help:"Specific database paths to query (must be in server's allowed list). Can be specified multiple times." group:"Database"`
}

type DeletedFlags struct {
	HideDeleted bool `default:"true" help:"Exclude deleted files from results" group:"Deleted"`
	OnlyDeleted bool `help:"Include only deleted files in results" group:"Deleted"`
}

type SortFlags struct {
	SortBy  string `short:"u" default:"path" help:"Sort by field" group:"Sort"`
	Reverse bool   `short:"V" help:"Reverse sort order" group:"Sort"`
	NatSort bool   `short:"n" help:"Use natural sorting" group:"Sort"`
	Random  bool   `short:"r" help:"Random order" group:"Sort"`
	ReRank  string `short:"k" alias:"rerank" help:"Add key/value pairs re-rank sorting by multiple attributes (COLUMN=WEIGHT)" group:"Sort"`
}

type DisplayFlags struct {
	Columns   []string `short:"c" help:"Columns to display" group:"Display"`
	JSON      bool     `short:"j" help:"Output results as JSON" group:"Display"`
	Summarize bool     `help:"Print aggregate statistics" group:"Display"`
	Frequency string   `short:"f" help:"Group statistics by time frequency (daily, weekly, monthly, yearly)" group:"Display"`
	TUI       bool     `help:"Interactive TUI mode" group:"Display"`
}

type AggregateFlags struct {
	BigDirs           bool     `short:"B" help:"Aggregate by parent directory" group:"Aggregate"`
	FileCounts        string   `help:"Filter by number of files in directory (e.g., >5, 10%1)" group:"Aggregate"`
	GroupByExtensions bool     `help:"Group by file extensions" group:"Aggregate"`
	GroupByMimeTypes  bool     `help:"Group by mimetypes" group:"Aggregate"`
	GroupBySize       bool     `help:"Group by size buckets" group:"Aggregate"`
	GroupByParent     bool     `help:"Group media by parent directory with counts and totals" group:"Aggregate"`
	Depth             int      `short:"D" help:"Aggregate at specific directory depth" group:"Aggregate"`
	MinDepth          int      `default:"0" help:"Minimum depth for aggregation" group:"Aggregate"`
	MaxDepth          int      `help:"Maximum depth for aggregation" group:"Aggregate"`
	Parents           bool     `help:"Include parent directories in aggregation" group:"Aggregate"`
	FoldersOnly       bool     `help:"Only show folders" group:"Aggregate"`
	FilesOnly         bool     `help:"Only show files" group:"Aggregate"`
	FolderSizes       []string `help:"Filter folders by total size" group:"Aggregate"`
	FolderCounts      string   `help:"Filter folders by number of subfolders" group:"Aggregate"`
}

type TextFlags struct {
	RegexSort  bool     `help:"Sort by splitting lines and sorting words" alias:"rs" group:"Text"`
	Regexs     []string `help:"Regex patterns for line splitting" alias:"re" group:"Text"`
	WordSorts  []string `help:"Word sorting strategies" group:"Text"`
	LineSorts  []string `help:"Line sorting strategies" group:"Text"`
	Compat     bool     `help:"Use natsort compat mode" group:"Text"`
	Preprocess bool     `default:"true" help:"Remove junk common to filenames and URLs" group:"Text"`
	StopWords  []string `help:"List of words to ignore" group:"Text"`
	Duplicates *bool    `help:"Filter for duplicate words (true/false)" group:"Text"`
	UniqueOnly *bool    `help:"Filter for unique words (true/false)" group:"Text"`
}

type SimilarityFlags struct {
	Similar         bool    `help:"Find similar files or folders" group:"Similarity"`
	SizesDelta      float64 `default:"10.0" help:"Size difference threshold (%)" group:"Similarity"`
	CountsDelta     float64 `default:"3.0" help:"File count difference threshold (%)" group:"Similarity"`
	DurationsDelta  float64 `default:"5.0" help:"Duration difference threshold (%)" group:"Similarity"`
	FilterNames     bool    `help:"Cluster by name similarity" group:"Similarity"`
	FilterSizes     bool    `help:"Cluster by size similarity" group:"Similarity"`
	FilterCounts    bool    `help:"Cluster by count similarity" group:"Similarity"`
	FilterDurations bool    `help:"Cluster by duration similarity" group:"Similarity"`
	TotalSizes      bool    `help:"Compare total sizes (folders only)" group:"Similarity"`
	TotalDurations  bool    `help:"Compare total durations (folders only)" group:"Similarity"`
	OnlyDuplicates  bool    `help:"Only show duplicate items" group:"Similarity"`
	OnlyOriginals   bool    `help:"Only show original items" group:"Similarity"`
	ClusterSort     bool    `short:"C" help:"Group items by similarity" group:"Similarity"`
	Clusters        int     `help:"Number of clusters" group:"Similarity"`
	TFIDF           bool    `help:"Use TF-IDF for clustering" group:"Similarity"`
	MoveGroups      bool    `help:"Move grouped files into separate directories" group:"Similarity"`
	PrintGroups     bool    `help:"Print clusters as JSON" group:"Similarity"`
}

type DedupeFlags struct {
	Audio              bool    `help:"Dedupe database by artist + album + title" group:"Dedupe"`
	ExtractorID        bool    `alias:"id" help:"Dedupe database by extractor_id" group:"Dedupe"`
	TitleOnly          bool    `help:"Dedupe database by title" group:"Dedupe"`
	DurationOnly       bool    `help:"Dedupe database by duration" group:"Dedupe"`
	Filesystem         bool    `alias:"fs" help:"Dedupe filesystem database (hash)" group:"Dedupe"`
	CompareDirs        bool    `help:"Compare directories" group:"Dedupe"`
	Basename           bool    `help:"Match by basename similarity" group:"Dedupe"`
	Dirname            bool    `help:"Match by dirname similarity" group:"Dedupe"`
	MinSimilarityRatio float64 `default:"0.8" help:"Filter out matches with less than this ratio (0.7-0.9)" group:"Dedupe"`
	DedupeCmd          string  `help:"Command to run for deduplication (rmlint-style: cmd duplicate keep)" group:"Dedupe"`
}

type FTSFlags struct {
	FTS      bool   `help:"Use FTS5 full-text search" group:"FTS"`
	FTSTable string `default:"media_fts" help:"FTS table name" group:"FTS"`
	NoFTS    bool   `help:"Disable full-text search, use substring search only" group:"FTS"`
	Related  int    `short:"R" help:"Find media related to the first result" group:"FTS"`
}

type PlaybackFlags struct {
	PlayInOrder           string   `short:"O" default:"natural_ps" help:"Play media in order" group:"Playback"`
	NoPlayInOrder         bool     `help:"Don't play media in order" group:"Playback"`
	Loop                  bool     `help:"Loop playback" group:"Playback"`
	Mute                  bool     `short:"M" help:"Start playback muted" group:"Playback"`
	OverridePlayer        string   `help:"Override default player (e.g. --player 'vlc')" group:"Playback"`
	Start                 string   `help:"Start playback at specific time/percentage" group:"Playback"`
	End                   string   `help:"Stop playback at specific time/percentage" group:"Playback"`
	Volume                int      `help:"Set initial volume (0-100)" group:"Playback"`
	Fullscreen            bool     `help:"Start in fullscreen" group:"Playback"`
	NoSubtitles           bool     `help:"Disable subtitles" group:"Playback"`
	SubtitleMix           float64  `default:"0.35" help:"Probability to play no-subtitle content" group:"Playback"`
	InterdimensionalCable int      `short:"4" alias:"4dtv" help:"Duration to play (in seconds) while changing the channel" group:"Playback"`
	Speed                 float64  `default:"1.0" help:"Playback speed" group:"Playback"`
	SavePlayhead          bool     `default:"true" help:"Save playback position on quit" group:"Playback"`
	MpvSocket             string   `help:"Mpv socket path" group:"Playback"`
	WatchLaterDir         string   `help:"Mpv watch_later directory" group:"Playback"`
	PlayerArgsSub         []string `help:"Player arguments for videos with subtitles" group:"Playback"`
	PlayerArgsNoSub       []string `help:"Player arguments for videos without subtitles" group:"Playback"`
	Cast                  bool     `help:"Cast to chromecast groups" group:"Playback"`
	CastDevice            string   `alias:"cast-to" help:"Chromecast device name" group:"Playback"`
	CastWithLocal         bool     `help:"Play music locally at the same time as chromecast" group:"Playback"`
}

type MpvActionFlags struct {
	Cmd0        string `help:"Command to run if mpv exits with code 0" group:"MpvAction"`
	Cmd1        string `help:"Command to run if mpv exits with code 1" group:"MpvAction"`
	Cmd2        string `help:"Command to run if mpv exits with code 2" group:"MpvAction"`
	Cmd3        string `help:"Command to run if mpv exits with code 3" group:"MpvAction"`
	Cmd4        string `help:"Command to run if mpv exits with code 4" group:"MpvAction"`
	Cmd5        string `help:"Command to run if mpv exits with code 5" group:"MpvAction"`
	Cmd6        string `help:"Command to run if mpv exits with code 6" group:"MpvAction"`
	Cmd7        string `help:"Command to run if mpv exits with code 7" group:"MpvAction"`
	Cmd8        string `help:"Command to run if mpv exits with code 8" group:"MpvAction"`
	Cmd9        string `help:"Command to run if mpv exits with code 9" group:"MpvAction"`
	Cmd10       string `help:"Command to run if mpv exits with code 10" group:"MpvAction"`
	Cmd11       string `help:"Command to run if mpv exits with code 11" group:"MpvAction"`
	Cmd12       string `help:"Command to run if mpv exits with code 12" group:"MpvAction"`
	Cmd13       string `help:"Command to run if mpv exits with code 13" group:"MpvAction"`
	Cmd14       string `help:"Command to run if mpv exits with code 14" group:"MpvAction"`
	Cmd15       string `help:"Command to run if mpv exits with code 15" group:"MpvAction"`
	Cmd20       string `help:"Command to run if mpv exits with code 20" group:"MpvAction"`
	Cmd127      string `help:"Command to run if mpv exits with code 127" group:"MpvAction"`
	Interactive bool   `short:"I" help:"Interactive decision making after playback" group:"MpvAction"`
}

type PostActionFlags struct {
	Trash        bool   `help:"Trash files after action" group:"PostAction"`
	PostAction   string `help:"Post-action: none, delete, mark-deleted, move, copy" group:"PostAction"`
	DeleteFiles  bool   `help:"Delete files after action" group:"PostAction"`
	DeleteRows   bool   `help:"Delete rows from database" group:"PostAction"`
	MarkDeleted  bool   `help:"Mark as deleted in database" group:"PostAction"`
	MoveTo       string `help:"Move files to directory" group:"PostAction"`
	CopyTo       string `help:"Copy files to directory" group:"PostAction"`
	ActionLimit  int    `help:"Stop after N files" group:"PostAction"`
	ActionSize   string `help:"Stop after N bytes (e.g., 10GB)" group:"PostAction"`
	TrackHistory bool   `default:"true" help:"Track playback history" group:"PostAction"`
}

type HashingFlags struct {
	HashGap       float64 `default:"0.1" help:"Gap between segments (0.0-1.0 as percentage of file size, or absolute bytes if >1)" group:"Hashing"`
	HashChunkSize int64   `help:"Size of each segment to hash" group:"Hashing"`
	HashThreads   int     `default:"1" help:"Number of threads to use for hashing a single file" group:"Hashing"`
}

type MergeFlags struct {
	OnlyTables        []string `short:"t" help:"Comma separated specific table(s)" group:"Merge"`
	PrimaryKeys       []string `help:"Comma separated primary keys" group:"Merge"`
	BusinessKeys      []string `help:"Comma separated business keys" group:"Merge"`
	Upsert            bool     `help:"Upsert rows on conflict" group:"Merge"`
	Ignore            bool     `help:"Ignore rows on conflict (only-new-rows)" group:"Merge"`
	OnlyNewRows       bool     `kong:"-"` // Alias for Ignore
	OnlyTargetColumns bool     `help:"Only copy columns that exist in target" group:"Merge"`
	SkipColumns       []string `help:"Columns to skip during merge" group:"Merge"`
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
	IgnoreErrors bool `short:"i" help:"Ignore errors and continue to next file"`
}

// ControlFlags are a subset of flags for simple control commands
type ControlFlags struct {
	MpvSocket  string `help:"Mpv socket path" group:"Playback"`
	CastDevice string `alias:"cast-to" help:"Chromecast device name" group:"Playback"`
	Verbose    bool   `short:"v" help:"Enable verbose logging"`
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

var LogLevel = &slog.LevelVar{}

func SetupLogging(verbose bool) {
	if verbose {
		LogLevel.Set(slog.LevelDebug)
		db.SetDebugMode(true)
	} else {
		LogLevel.Set(slog.LevelInfo)
	}
}

// BuildQueryGlobalFlags constructs GlobalFlags for query-based commands
// This reduces boilerplate in command Run() methods
func BuildQueryGlobalFlags(
	core CoreFlags,
	query QueryFlags,
	pathFilter PathFilterFlags,
	filter FilterFlags,
	mediaFilter MediaFilterFlags,
	timeFilter TimeFilterFlags,
	deleted DeletedFlags,
	sort SortFlags,
	display DisplayFlags,
	fts FTSFlags,
) GlobalFlags {
	return GlobalFlags{
		CoreFlags:        core,
		QueryFlags:       query,
		PathFilterFlags:  pathFilter,
		FilterFlags:      filter,
		MediaFilterFlags: mediaFilter,
		TimeFilterFlags:  timeFilter,
		DeletedFlags:     deleted,
		SortFlags:        sort,
		DisplayFlags:     display,
		FTSFlags:         fts,
	}
}
