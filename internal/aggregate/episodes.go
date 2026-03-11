package aggregate

import (
	"path/filepath"
	"sort"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

// GroupByParent groups media items by their parent directory
func GroupByParent(media []models.MediaWithDB) []models.FolderStats {
	groups := make(map[string]*models.FolderStats)

	for _, m := range media {
		parent := filepath.Dir(m.Path)
		if _, ok := groups[parent]; !ok {
			groups[parent] = &models.FolderStats{Path: parent}
		}

		stats := groups[parent]
		stats.Files = append(stats.Files, m)
		stats.Count++

		if m.Size != nil {
			stats.TotalSize += *m.Size
		}
		if m.Duration != nil {
			stats.TotalDuration += *m.Duration
		}
		stats.ExistsCount++ // Assuming exist if they came from query

		if m.PlayCount != nil && *m.PlayCount > 0 {
			stats.PlayedCount++
		}
	}

	var result []models.FolderStats
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
