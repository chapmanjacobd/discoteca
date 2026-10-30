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

	result := make([]FolderStats, 0, len(folders))
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
		case "priority", "size_per_count", "size_per_file":
			p1 := float64(folders[i].TotalSize) / float64(max(1, folders[i].Count))
			p2 := float64(folders[j].TotalSize) / float64(max(1, folders[j].Count))
			if p1 != p2 {
				return p1 < p2
			}
			if folders[i].TotalSize != folders[j].TotalSize {
				return folders[i].TotalSize < folders[j].TotalSize
			}
			if folders[i].Count != folders[j].Count {
				return folders[i].Count < folders[j].Count
			}
			return folders[i].Path > folders[j].Path
		default:
			// Default to DU-style sorting
			p1 := float64(folders[i].TotalSize) / float64(max(1, folders[i].Count))
			p2 := float64(folders[j].TotalSize) / float64(max(1, folders[j].Count))
			if p1 != p2 {
				return p1 < p2
			}
			if folders[i].TotalSize != folders[j].TotalSize {
				return folders[i].TotalSize < folders[j].TotalSize
			}
			if folders[i].Count != folders[j].Count {
				return folders[i].Count < folders[j].Count
			}
			return folders[i].Path > folders[j].Path
		}
	}

	if reverse {
		sort.SliceStable(folders, func(i, j int) bool { return less(j, i) })
		return
	}
	sort.SliceStable(folders, less)
}
