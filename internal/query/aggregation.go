package query

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
	"github.com/chapmanjacobd/discoteca/internal/utils/pathutil"
)

func AggregateMedia(media []models.MediaWithDB, flags models.GlobalFlags) []models.FolderStats {
	var stats []models.FolderStats
	if flags.GroupByExtensions {
		stats = AggregateExtensions(media)
	} else if flags.GroupByMimeTypes {
		stats = AggregateMimeTypes(media)
	} else if flags.GroupBySize {
		stats = AggregateSizeBuckets(media)
	} else {
		stats = AggregateByDepth(media, flags)
	}

	// Post-aggregation filtering
	if flags.FoldersOnly || flags.FilesOnly || flags.FolderSizes != nil || flags.FileCounts != "" || flags.FolderCounts != "" {
		var filtered []models.FolderStats
		for _, f := range stats {
			keep := true
			if flags.FoldersOnly && f.Count == 0 {
				keep = false
			} else if flags.FilesOnly && f.Count > 0 {
				keep = false
			}

			if keep && len(flags.FolderSizes) > 0 {
				for _, fs := range flags.FolderSizes {
					if r, err := utils.ParseRange(fs, utils.HumanToBytes); err == nil {
						if !r.Matches(f.TotalSize) {
							keep = false
							break
						}
					}
				}
			}
			if keep && flags.FileCounts != "" {
				if r, err := utils.ParseRange(flags.FileCounts, func(s string) (int64, error) {
					return strconv.ParseInt(s, 10, 64)
				}); err == nil {
					// In Python, file_counts applies to 'exists' or 'count'
					if !r.Matches(int64(utils.Max(f.ExistsCount, f.Count))) {
						keep = false
					}
				}
			}
			if keep && flags.FolderCounts != "" {
				if r, err := utils.ParseRange(flags.FolderCounts, func(s string) (int64, error) {
					return strconv.ParseInt(s, 10, 64)
				}); err == nil {
					if !r.Matches(int64(f.FolderCount)) {
						keep = false
					}
				}
			}
			if keep {
				filtered = append(filtered, f)
			}
		}
		stats = filtered
	}

	return stats
}

func AggregateExtensions(media []models.MediaWithDB) []models.FolderStats {
	groups := make(map[string]*models.FolderStats)
	for _, m := range media {
		ext := strings.ToLower(filepath.Ext(m.Path))
		if ext == "" {
			ext = "no extension"
		}
		if _, ok := groups[ext]; !ok {
			groups[ext] = &models.FolderStats{Path: ext}
		}
		updateStats(groups[ext], m, true)
	}
	return finalizeStats(groups)
}

func AggregateMimeTypes(media []models.MediaWithDB) []models.FolderStats {
	groups := make(map[string]*models.FolderStats)
	for _, m := range media {
		mime := "unknown"
		if m.Type != nil {
			mime = *m.Type
		}
		if _, ok := groups[mime]; !ok {
			groups[mime] = &models.FolderStats{Path: mime}
		}
		updateStats(groups[mime], m, true)
	}
	return finalizeStats(groups)
}

func AggregateSizeBuckets(media []models.MediaWithDB) []models.FolderStats {
	baseEdges := []int64{2, 5, 10}
	var multipliers []int64
	multipliers = append(multipliers, baseEdges...)
	for _, n := range baseEdges {
		multipliers = append(multipliers, n*10)
	}
	for _, n := range baseEdges {
		multipliers = append(multipliers, n*100)
	}

	unitMultiplier := int64(1024)
	units := []int64{
		1,
		unitMultiplier,
		unitMultiplier * unitMultiplier,
		unitMultiplier * unitMultiplier * unitMultiplier,
		unitMultiplier * unitMultiplier * unitMultiplier * unitMultiplier,
	}

	var binEdges []float64
	binEdges = append(binEdges, 0.0)
	for _, unit := range units {
		for _, mMult := range multipliers {
			binEdges = append(binEdges, float64(mMult*unit))
		}
	}
	binEdges = append(binEdges, math.Inf(1))

	groups := make(map[string]*models.FolderStats)
	for _, m := range media {
		size := float64(0)
		if m.Size != nil {
			size = float64(*m.Size)
		}

		var label string
		for i := 0; i < len(binEdges)-1; i++ {
			if size >= binEdges[i] && size < binEdges[i+1] {
				label = fmt.Sprintf("%s-%s", utils.FormatSize(int64(binEdges[i])), utils.FormatSize(int64(binEdges[i+1])))
				if binEdges[i+1] == math.Inf(1) {
					label = fmt.Sprintf(">%s", utils.FormatSize(int64(binEdges[i])))
				}
				break
			}
		}

		if _, ok := groups[label]; !ok {
			groups[label] = &models.FolderStats{Path: label}
		}
		updateStats(groups[label], m, true)
	}
	return finalizeStats(groups)
}

func AggregateByDepth(media []models.MediaWithDB, flags models.GlobalFlags) []models.FolderStats {
	groups := make(map[string]*models.FolderStats)

	for _, m := range media {
		path := filepath.Clean(m.Path)
		// Use pathutil for proper cross-platform path handling
		parts, isAbs := pathutil.Split(path)

		// 1. Add the file itself if it matches depth
		if flags.Depth > 0 && len(parts) == flags.Depth {
			if _, ok := groups[path]; !ok {
				groups[path] = &models.FolderStats{Path: path}
			}
			updateStats(groups[path], m, false)
		}

		// 2. Add folders
		if flags.Parents {
			// MinDepth refers to the number of path components, so MinDepth=1 means start from first component
			start := max(flags.MinDepth-1, 0)
			for d := start; d < len(parts); d++ {
				if flags.MaxDepth > 0 && d+1 > flags.MaxDepth {
					break
				}
				parent := pathutil.Join(parts[:d+1], isAbs)
				if _, ok := groups[parent]; !ok {
					groups[parent] = &models.FolderStats{Path: parent}
				}
				updateStats(groups[parent], m, true)
			}
		} else if flags.BigDirs {
			// BigDirs: group by immediate parent directory
			parent := m.Parent()
			if _, ok := groups[parent]; !ok {
				groups[parent] = &models.FolderStats{Path: parent}
			}
			updateStats(groups[parent], m, true)
		} else if flags.Depth > 0 && len(parts) > flags.Depth {
			// Group at depth (e.g., depth=1 -> "/media" or "media", depth=2 -> "/media/video")
			var parent string
			if flags.Depth < len(parts) {
				parent = pathutil.Join(parts[:flags.Depth], isAbs)
			} else {
				parent = pathutil.Join(parts, isAbs)
			}
			if _, ok := groups[parent]; !ok {
				groups[parent] = &models.FolderStats{Path: parent}
			}
			updateStats(groups[parent], m, true)
		} else {
			// Default to immediate parent (when Depth=0 or file is at/below target depth)
			parent := m.Parent()
			if _, ok := groups[parent]; !ok {
				groups[parent] = &models.FolderStats{Path: parent}
			}
			updateStats(groups[parent], m, true)
		}
	}

	return finalizeStats(groups)
}

func updateStats(f *models.FolderStats, m models.MediaWithDB, isFolder bool) {
	if isFolder {
		f.Count++
	}
	isDeleted := m.TimeDeleted != nil && *m.TimeDeleted > 0
	if isDeleted {
		f.DeletedCount++
	} else {
		f.ExistsCount++
		if m.Size != nil {
			f.TotalSize += *m.Size
		}
		if m.Duration != nil {
			f.TotalDuration += *m.Duration
		}
	}

	if m.PlayCount != nil && *m.PlayCount > 0 {
		f.PlayedCount++
	}
	f.Files = append(f.Files, m)
}

func finalizeStats(groups map[string]*models.FolderStats) []models.FolderStats {
	// Identify parents to count subdirectories
	for path := range groups {
		parts, isAbs := pathutil.Split(path)
		if len(parts) > 1 {
			for i := len(parts) - 1; i > 0; i-- {
				p := pathutil.Join(parts[:i], isAbs)
				if _, ok := groups[p]; ok {
					groups[p].FolderCount++
				}
			}
		}
		// Special case for root (isAbs with no components, or drive letter root)
		if isAbs {
			root := pathutil.Join(nil, true)
			if path != root {
				if _, ok := groups[root]; ok {
					groups[root].FolderCount++
				}
			}
		}
	}

	var result []models.FolderStats
	for _, f := range groups {
		if f.ExistsCount > 0 {
			f.AvgSize = f.TotalSize / int64(f.ExistsCount)
			f.AvgDuration = f.TotalDuration / int64(f.ExistsCount)

			sizes := make([]int64, 0, f.ExistsCount)
			durations := make([]int64, 0, f.ExistsCount)
			for _, m := range f.Files {
				isDeleted := m.TimeDeleted != nil && *m.TimeDeleted > 0
				if !isDeleted {
					if m.Size != nil {
						sizes = append(sizes, *m.Size)
					}
					if m.Duration != nil {
						durations = append(durations, *m.Duration)
					}
				}
			}
			f.MedianSize = int64(utils.SafeMedian(sizes))
			f.MedianDuration = int64(utils.SafeMedian(durations))
		}
		result = append(result, *f)
	}
	return result
}
