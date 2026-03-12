//go:build fts5 && !bleve

package models

// FTSFlagsBuildSpecific contains FTS flags specific to FTS5-only builds
type FTSFlagsBuildSpecific struct {
	FTS      bool   `help:"Use FTS5 full-text search" group:"FTS"`
	FTSTable string `default:"media_fts" help:"FTS table name" group:"FTS"`
	Bleve    bool   `help:"Use Bleve full-text search index (not available in this build)" group:"FTS" hidden:""`
}
