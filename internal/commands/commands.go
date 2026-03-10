package commands

import (
	"embed"

	"github.com/chapmanjacobd/discotheque/internal/db"
)

// SchemaFS provides access to the database schema
// Deprecated: Use db.SchemaFS directly
var SchemaFS = db.SchemaFS

//go:embed schema.sql
var _legacySchemaFS embed.FS

// init runs migration to ensure schema is available from central location
func init() {
	// Keep legacy embed for backward compatibility during transition
	// New code should use db.SchemaFS
	_ = _legacySchemaFS
}

// GetSchema returns the database schema SQL
// Deprecated: Use db.GetSchema directly
func GetSchema() string {
	return db.GetSchema()
}
