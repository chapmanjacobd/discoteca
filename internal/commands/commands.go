package commands

import (
	"fmt"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

// ParseDatabaseAndScanPaths separates .db files from scan paths
// Returns databases, scan paths, and any error
func ParseDatabaseAndScanPaths(
	args []string,
	coreFlags *models.CoreFlags,
	mediaFlags *models.MediaFilterFlags,
) (databases, scanPaths []string, err error) {
	if err = coreFlags.AfterApply(); err != nil {
		return nil, nil, err
	}
	if err = mediaFlags.AfterApply(); err != nil {
		return nil, nil, err
	}

	for _, arg := range args {
		if strings.HasSuffix(arg, ".db") {
			if !utils.IsSQLite(arg) {
				return nil, nil, fmt.Errorf("database file not found: %s", arg)
			}
			databases = append(databases, arg)
		} else {
			scanPaths = append(scanPaths, arg)
		}
	}
	return databases, scanPaths, nil
}
