//go:build !fts5 && !bleve

package models

// FTSFlagsBuildSpecific contains FTS flags for builds without FTS support
type FTSFlagsBuildSpecific struct {
	FTS      bool   `help:"Use FTS5 full-text search (not available in this build)" group:"FTS" hidden:""`
	FTSTable string `default:"media_fts" help:"FTS table name" group:"FTS" hidden:""`
	Bleve    bool   `help:"Use Bleve full-text search index (not available in this build)" group:"FTS" hidden:""`
}
