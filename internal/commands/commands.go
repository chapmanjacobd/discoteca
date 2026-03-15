package commands

import (
	"embed"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// SchemaFS provides access to the database schema
func SchemaFS() embed.FS {
	return db.SchemaFS
}

// GetSchema returns the database schema SQL
func GetSchema() string {
	return db.GetSchema()
}

// ParseDatabaseAndScanPaths separates .db files from scan paths
// Returns databases, scan paths, and any error
func ParseDatabaseAndScanPaths(args []string, coreFlags *models.CoreFlags, mediaFlags *models.MediaFilterFlags) ([]string, []string, error) {
	if err := coreFlags.AfterApply(); err != nil {
		return nil, nil, err
	}
	if err := mediaFlags.AfterApply(); err != nil {
		return nil, nil, err
	}

	var databases, scanPaths []string
	for _, arg := range args {
		if strings.HasSuffix(arg, ".db") && utils.IsSQLite(arg) {
			databases = append(databases, arg)
		} else {
			scanPaths = append(scanPaths, arg)
		}
	}
	return databases, scanPaths, nil
}
