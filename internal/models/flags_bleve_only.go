//go:build bleve && !fts5

package models

// FTSFlagsBuildSpecific contains FTS flags specific to Bleve-only builds
type FTSFlagsBuildSpecific struct {
	FTS      bool   `help:"Use FTS5 full-text search (not available in this build)" group:"FTS" hidden:""`
	FTSTable string `default:"media_fts" help:"FTS table name" group:"FTS" hidden:""`
	Bleve    bool   `help:"Use Bleve full-text search index" group:"FTS"`
}
