package db

import (
	"embed"
)

//go:embed schema_tables.sql schema_triggers.sql schema_fts.sql
var SchemaFS embed.FS

// GetSchemaTables returns the core database tables SQL
func GetSchemaTables() string {
	data, err := SchemaFS.ReadFile("schema_tables.sql")
	if err != nil {
		panic("schema_tables.sql not found: " + err.Error())
	}
	return string(data)
}

// GetSchemaTriggers returns the core database triggers and indexes SQL
func GetSchemaTriggers() string {
	data, err := SchemaFS.ReadFile("schema_triggers.sql")
	if err != nil {
		panic("schema_triggers.sql not found: " + err.Error())
	}
	return string(data)
}

// GetSchemaFTS returns the FTS database schema SQL
func GetSchemaFTS() string {
	data, err := SchemaFS.ReadFile("schema_fts.sql")
	if err != nil {
		panic("schema_fts.sql not found: " + err.Error())
	}
	return string(data)
}

// GetSchema returns the full schema (for backward compatibility if needed, but InitDB logic changed)
// Deprecated: Use GetSchemaTables, GetSchemaTriggers, GetSchemaFTS instead
func GetSchema() string {
	return GetSchemaTables() + "\n" + GetSchemaTriggers() + "\n" + GetSchemaFTS()
}
