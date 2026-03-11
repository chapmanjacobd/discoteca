package aggregate

import (
	"path/filepath"
	"sort"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

// FolderStats aggregates media by folder
type FolderStats struct {
	Path          string         `json:"path"`
	Count         int            `json:"count"`
	TotalSize     int64          `json:"total_size"`
	TotalDuration int64          `json:"total_duration"`
	AvgSize       int64          `json:"avg_size"`
	AvgDuration   int64          `json:"avg_duration"`
	Files         []models.Media `json:"files,omitempty"`
}

// ByFolder groups media by parent directory
func ByFolder(media []models.Media) []FolderStats {
	folders := make(map[string]*FolderStats)

	for _, m := range media {
		parent := filepath.Dir(m.Path)
		if _, exists := folders[parent]; !exists {
			folders[parent] = &FolderStats{
				Path:  parent,
				Files: []models.Media{},
			}
		}

		f := folders[parent]
		f.Count++
		if m.Size != nil {
			f.TotalSize += *m.Size
		}
		if m.Duration != nil {
			f.TotalDuration += *m.Duration
		}
		f.Files = append(f.Files, m)
	}

	var result []FolderStats
	for _, f := range folders {
		if f.Count > 0 {
			f.AvgSize = f.TotalSize / int64(f.Count)
			f.AvgDuration = f.TotalDuration / int64(f.Count)
		}
		result = append(result, *f)
	}

	return result
}

// SortFolders sorts folder stats
func SortFolders(folders []FolderStats, sortBy string, reverse bool) {
	less := func(i, j int) bool {
		switch sortBy {
		case "count":
			return folders[i].Count < folders[j].Count
		case "size":
			return folders[i].TotalSize < folders[j].TotalSize
		case "duration":
			return folders[i].TotalDuration < folders[j].TotalDuration
		default:
			return folders[i].Path < folders[j].Path
		}
	}

	if reverse {
		sort.Slice(folders, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.Slice(folders, less)
	}
}
