package schema

import (
	"embed"
	"strings"
)

//go:embed *.sql
var SchemaFS embed.FS

// GetCoreTables returns the SQL to create all core tables (media, playlists, history, meta)
func GetCoreTables() string {
	var sb strings.Builder
	sb.WriteString(GetMediaTable())
	sb.WriteString("\n")
	sb.WriteString(GetPlaylistsTables())
	sb.WriteString("\n")
	sb.WriteString(GetHistoryTable())
	sb.WriteString("\n")
	sb.WriteString(GetMetaTables())
	return sb.String()
}

// GetCoreIndexes returns the SQL to create core indexes (media indexes)
func GetCoreIndexes() string {
	return GetMediaIndexes()
}

// GetFTSTables returns the SQL to create FTS tables (media, captions)
// Note: Call GetCaptionsTable first if you want captions FTS
func GetFTSTables() string {
	var sb strings.Builder
	sb.WriteString(GetMediaFTS())
	sb.WriteString("\n")
	sb.WriteString(GetCaptionsFTS())
	return sb.String()
}
