package aggregate

import (
	"path/filepath"
	"sort"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

// GroupMediaByParent groups media items into FolderStats by parent directory.
// The query package has a separate GroupByParent method that returns synthetic
// MediaWithDB rows for the API's group-by-parent query mode.
func GroupMediaByParent(media []models.MediaWithDB) []models.FolderStats {
	groups := make(map[string]*models.FolderStats)

	for _, m := range media {
		parent := filepath.Dir(m.Path)
		if _, ok := groups[parent]; !ok {
			groups[parent] = &models.FolderStats{Path: parent}
		}

		stats := groups[parent]
		stats.Files = append(stats.Files, m)
		stats.Count++

		if m.TimeDeleted != nil && *m.TimeDeleted > 0 {
			stats.DeletedCount++
		} else {
			stats.ExistsCount++
			if m.Size != nil {
				stats.TotalSize += *m.Size
			}
			if m.Duration != nil {
				stats.TotalDuration += *m.Duration
			}
		}

		if m.PlayCount != nil && *m.PlayCount > 0 {
			stats.PlayedCount++
		}
	}

	result := make([]models.FolderStats, 0, len(groups))
	for _, stats := range groups {
		if stats.Count > 0 {
			stats.AvgSize = stats.TotalSize / int64(stats.Count)
			stats.AvgDuration = stats.TotalDuration / int64(stats.Count)
		}
		result = append(result, *stats)
	}

	// Sort by path
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}
