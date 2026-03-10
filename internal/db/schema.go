package db

import (
	"embed"
)

//go:embed schema.sql
var SchemaFS embed.FS

// GetSchema returns the complete database schema SQL
func GetSchema() string {
	data, err := SchemaFS.ReadFile("schema.sql")
	if err != nil {
		// This should never happen as the schema is embedded
		panic("schema.sql not found: " + err.Error())
	}
	return string(data)
}
