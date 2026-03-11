package commands

import (
	"embed"

	"github.com/chapmanjacobd/discotheque/internal/db"
)

// SchemaFS provides access to the database schema
func SchemaFS() embed.FS {
	return db.SchemaFS
}

// GetSchema returns the database schema SQL
func GetSchema() string {
	return db.GetSchema()
}
