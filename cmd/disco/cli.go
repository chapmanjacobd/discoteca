package main

import (
	"github.com/chapmanjacobd/discoteca/internal/commands"
)

// CLI defines the command-line interface
type CLI struct {
	Add            commands.AddCmd            `help:"Add media to database"                               cmd:""`
	Check          commands.CheckCmd          `help:"Check for missing files and mark as deleted"         cmd:""`
	Print          commands.PrintCmd          `help:"Print media information"                             cmd:""`
	Search         commands.SearchCmd         `help:"Search media using FTS"                              cmd:""`
	SearchCaptions commands.SearchCaptionsCmd `help:"Search captions using FTS"                           cmd:"" aliases:"sc"`
	Playlists      commands.PlaylistsCmd      `help:"List scan roots (playlists)"                         cmd:""`
	SearchDB       commands.SearchDBCmd       `help:"Search arbitrary database table"                     cmd:"" aliases:"sdb"`
	MediaCheck     commands.MediaCheckCmd     `help:"Check media files for corruption"                    cmd:"" aliases:"mc"`
	FilesInfo      commands.FilesInfoCmd      `help:"Show information about files"                        cmd:"" aliases:"fs"`
	DiskUsage      commands.DiskUsageCmd      `help:"Show disk usage aggregation"                         cmd:"" aliases:"du"`
	Dedupe         commands.DedupeCmd         `help:"Dedupe similar media"                                cmd:"" aliases:"dedupe-media" name:"dedupe"`
	BigDirs        commands.BigDirsCmd        `help:"Show big directories aggregation"                    cmd:"" aliases:"bigdirs,bd"`
	Categorize     commands.CategorizeCmd     `help:"Auto-group media into categories"                    cmd:""`
	SimilarFiles   commands.SimilarFilesCmd   `help:"Find similar files"                                  cmd:"" aliases:"sf"`
	SimilarFolders commands.SimilarFoldersCmd `help:"Find similar folders"                                cmd:"" aliases:"sh"`
	Watch          commands.WatchCmd          `help:"Watch videos with mpv"                               cmd:""`
	Listen         commands.ListenCmd         `help:"Listen to audio with mpv"                            cmd:""`
	Stats          commands.StatsCmd          `help:"Show library statistics"                             cmd:""`
	History        commands.HistoryCmd        `help:"Show playback history"                               cmd:""`
	HistoryAdd     commands.HistoryAddCmd     `help:"Add paths to playback history"                       cmd:""`
	MpvWatchlater  commands.MpvWatchlaterCmd  `help:"Import mpv watchlater files to history"              cmd:""                        name:"mpv-watchlater"`
	Serve          commands.ServeCmd          `help:"Start Web UI server"                                 cmd:""`
	Optimize       commands.OptimizeCmd       `help:"Optimize database (VACUUM, ANALYZE, FTS optimize)"   cmd:""`
	Repair         commands.RepairCmd         `help:"Repair malformed database using sqlite3"             cmd:""`
	Readme         commands.ReadmeCmd         `help:"Generate README.md content"                          cmd:""`
	RegexSort      commands.RegexSortCmd      `help:"Sort by splitting lines and sorting words"           cmd:"" aliases:"rs"`
	ClusterSort    commands.ClusterSortCmd    `help:"Group items by similarity"                           cmd:"" aliases:"cs"`
	SampleHash     commands.SampleHashCmd     `help:"Calculate a hash based on small file segments"       cmd:"" aliases:"hash"         name:"sample-hash"`
	Open           commands.OpenCmd           `help:"Open files with default application"                 cmd:""`
	Browse         commands.BrowseCmd         `help:"Open URLs in browser"                                cmd:""`
	Now            commands.NowCmd            `help:"Show current mpv playback status"                    cmd:""`
	Next           commands.NextCmd           `help:"Skip to next file in mpv"                            cmd:""`
	Stop           commands.StopCmd           `help:"Stop mpv playback"                                   cmd:""`
	Pause          commands.PauseCmd          `help:"Toggle mpv pause state"                              cmd:"" aliases:"play"`
	Seek           commands.SeekCmd           `help:"Seek mpv playback"                                   cmd:"" aliases:"ffwd,rewind"`
	MergeDBs       commands.MergeDBsCmd       `help:"Merge multiple SQLite databases"                     cmd:"" aliases:"mergedbs"     name:"merge-dbs"`
	Explode        commands.ExplodeCmd        `help:"Create symlinks for all subcommands (busybox-style)" cmd:""`
	Update         commands.UpdateCmd         `help:"Check for and install updates from GitHub"           cmd:""`
	Version        commands.VersionCmd        `help:"Show version and build information"                  cmd:""`

	ExitCalled bool `kong:"-"`
	ExitCode   int  `kong:"-"`
}

func (c *CLI) Terminate(code int) {
	c.ExitCalled = true
	c.ExitCode = code
}
