package main

import (
	"github.com/chapmanjacobd/discoteca/internal/commands"
)

// CLI defines the command-line interface
type CLI struct {
	Add            commands.AddCmd            `cmd:"" help:"Add media to database"`
	Check          commands.CheckCmd          `cmd:"" help:"Check for missing files and mark as deleted"`
	Print          commands.PrintCmd          `cmd:"" help:"Print media information"`
	Search         commands.SearchCmd         `cmd:"" help:"Search media using FTS"`
	SearchCaptions commands.SearchCaptionsCmd `cmd:"" help:"Search captions using FTS" aliases:"sc"`
	Playlists      commands.PlaylistsCmd      `cmd:"" help:"List scan roots (playlists)"`
	SearchDB       commands.SearchDBCmd       `cmd:"" help:"Search arbitrary database table" aliases:"sdb"`
	MediaCheck     commands.MediaCheckCmd     `cmd:"" help:"Check media files for corruption" aliases:"mc"`
	FilesInfo      commands.FilesInfoCmd      `cmd:"" help:"Show information about files" aliases:"fs"`
	DiskUsage      commands.DiskUsageCmd      `cmd:"" help:"Show disk usage aggregation" aliases:"du"`
	Dedupe         commands.DedupeCmd         `cmd:"" name:"dedupe" help:"Dedupe similar media" aliases:"dedupe-media"`
	Shrink         commands.ShrinkCmd         `cmd:"" name:"shrink" help:"Shrink media to efficient formats (AV1/Opus/AVIF)" aliases:"media-shrink"`
	BigDirs        commands.BigDirsCmd        `cmd:"" help:"Show big directories aggregation" aliases:"bigdirs,bd"`
	Categorize     commands.CategorizeCmd     `cmd:"" help:"Auto-group media into categories"`
	SimilarFiles   commands.SimilarFilesCmd   `cmd:"" help:"Find similar files" aliases:"sf"`
	SimilarFolders commands.SimilarFoldersCmd `cmd:"" help:"Find similar folders" aliases:"sh"`
	Watch          commands.WatchCmd          `cmd:"" help:"Watch videos with mpv"`
	Listen         commands.ListenCmd         `cmd:"" help:"Listen to audio with mpv"`
	Stats          commands.StatsCmd          `cmd:"" help:"Show library statistics"`
	History        commands.HistoryCmd        `cmd:"" help:"Show playback history"`
	HistoryAdd     commands.HistoryAddCmd     `cmd:"" help:"Add paths to playback history"`
	MpvWatchlater  commands.MpvWatchlaterCmd  `cmd:"" name:"mpv-watchlater" help:"Import mpv watchlater files to history"`
	Serve          commands.ServeCmd          `cmd:"" help:"Start Web UI server"`
	Optimize       commands.OptimizeCmd       `cmd:"" help:"Optimize database (VACUUM, ANALYZE, FTS optimize)"`
	Repair         commands.RepairCmd         `cmd:"" help:"Repair malformed database using sqlite3"`
	Tui            commands.TuiCmd            `cmd:"" help:"Interactive TUI media picker"`
	Readme         commands.ReadmeCmd         `cmd:"" help:"Generate README.md content"`
	RegexSort      commands.RegexSortCmd      `cmd:"" help:"Sort by splitting lines and sorting words" aliases:"rs"`
	ClusterSort    commands.ClusterSortCmd    `cmd:"" help:"Group items by similarity" aliases:"cs"`
	SampleHash     commands.SampleHashCmd     `cmd:"" name:"sample-hash" help:"Calculate a hash based on small file segments" aliases:"hash"`
	Open           commands.OpenCmd           `cmd:"" help:"Open files with default application"`
	Browse         commands.BrowseCmd         `cmd:"" help:"Open URLs in browser"`
	Now            commands.NowCmd            `cmd:"" help:"Show current mpv playback status"`
	Next           commands.NextCmd           `cmd:"" help:"Skip to next file in mpv"`
	Stop           commands.StopCmd           `cmd:"" help:"Stop mpv playback"`
	Pause          commands.PauseCmd          `cmd:"" help:"Toggle mpv pause state" aliases:"play"`
	Seek           commands.SeekCmd           `cmd:"" help:"Seek mpv playback" aliases:"ffwd,rewind"`
	MergeDBs       commands.MergeDBsCmd       `cmd:"" name:"merge-dbs" help:"Merge multiple SQLite databases" aliases:"mergedbs"`
	Explode        commands.ExplodeCmd        `cmd:"" help:"Create symlinks for all subcommands (busybox-style)"`
	Update         commands.UpdateCmd         `cmd:"" help:"Check for and install updates from GitHub"`
	Version        commands.VersionCmd        `cmd:"" help:"Show version and build information"`

	ExitCalled bool `kong:"-"`
}

func (c *CLI) Terminate(code int) {
	c.ExitCalled = true
}
