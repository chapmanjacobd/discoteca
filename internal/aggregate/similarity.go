package aggregate

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
)

// IsSameGroup checks if two media items are similar enough to be grouped
func IsSameGroup(flags models.GlobalFlags, m0, m models.MediaWithDB) bool {
	if flags.FilterSizes {
		s0 := utils.Int64Value(m0.Size)
		s := utils.Int64Value(m.Size)
		if utils.PercentageDifference(float64(s0), float64(s)) >= flags.SizesDelta {
			return false
		}
	}

	if flags.FilterDurations {
		d0 := utils.Int64Value(m0.Duration)
		d := utils.Int64Value(m.Duration)
		if d > 0 {
			if utils.PercentageDifference(float64(d0), float64(d)) >= flags.DurationsDelta {
				return false
			}
		}
	}

	return true
}

// IsSameFolderGroup checks if two FolderStats are similar enough to be grouped
func IsSameFolderGroup(flags models.GlobalFlags, f0, f models.FolderStats) bool {
	if flags.FilterCounts {
		if utils.PercentageDifference(float64(f0.ExistsCount), float64(f.ExistsCount)) >= flags.CountsDelta {
			return false
		}
	}

	if flags.FilterSizes {
		s0 := f0.TotalSize
		s := f.TotalSize
		if !flags.TotalSizes {
			s0 = f0.MedianSize
			s = f.MedianSize
		}
		if utils.PercentageDifference(float64(s0), float64(s)) >= flags.SizesDelta {
			return false
		}
	}

	if flags.FilterDurations {
		d0 := f0.TotalDuration
		d := f.TotalDuration
		if !flags.TotalDurations {
			d0 = f0.MedianDuration
			d = f.MedianDuration
		}
		if utils.PercentageDifference(float64(d0), float64(d)) >= flags.DurationsDelta {
			return false
		}
	}

	return true
}

// ClusterByNumbers groups media items by numerical similarity
func ClusterByNumbers(flags models.GlobalFlags, media []models.MediaWithDB) []models.FolderStats {
	var groups [][]models.MediaWithDB

	for _, m := range media {
		found := false
		for i, group := range groups {
			if IsSameGroup(flags, group[0], m) {
				groups[i] = append(groups[i], m)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, []models.MediaWithDB{m})
		}
	}

	var result []models.FolderStats
	for _, group := range groups {
		if len(group) < 2 && (flags.OnlyDuplicates || flags.Similar) {
			continue
		}

		paths := make([]string, len(group))
		for i, m := range group {
			paths[i] = m.Path
		}
		common := utils.CommonPathFull(paths)

		stats := models.FolderStats{
			Path:  common,
			Files: group,
			Count: len(group),
		}
		// Finalize stats for the group
		for _, m := range group {
			if m.Size != nil {
				stats.TotalSize += *m.Size
			}
			if m.Duration != nil {
				stats.TotalDuration += *m.Duration
			}
		}
		if stats.Count > 0 {
			stats.AvgSize = stats.TotalSize / int64(stats.Count)
			stats.AvgDuration = stats.TotalDuration / int64(stats.Count)
		}

		result = append(result, stats)
	}

	return result
}

// ClusterFoldersByNumbers groups folder stats by numerical similarity
func ClusterFoldersByNumbers(flags models.GlobalFlags, folders []models.FolderStats) []models.FolderStats {
	var groups [][]models.FolderStats

	for _, f := range folders {
		found := false
		for i, group := range groups {
			if IsSameFolderGroup(flags, group[0], f) {
				groups[i] = append(groups[i], f)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, []models.FolderStats{f})
		}
	}

	return groupToStats(flags, groups)
}

func groupToStats(flags models.GlobalFlags, groups [][]models.FolderStats) []models.FolderStats {
	var result []models.FolderStats
	for _, group := range groups {
		if len(group) < 2 && (flags.OnlyDuplicates || flags.Similar) {
			continue
		}

		paths := make([]string, len(group))
		for i, f := range group {
			paths[i] = f.Path
		}
		common := utils.CommonPathFull(paths)

		merged := models.FolderStats{
			Path: common,
		}
		for _, f := range group {
			merged.Files = append(merged.Files, f.Files...)
			merged.Count += f.Count
			merged.TotalSize += f.TotalSize
			merged.TotalDuration += f.TotalDuration
			merged.ExistsCount += f.ExistsCount
			merged.DeletedCount += f.DeletedCount
			merged.PlayedCount += f.PlayedCount
			merged.FolderCount += f.FolderCount
		}
		if merged.Count > 0 {
			merged.AvgSize = merged.TotalSize / int64(merged.Count)
			merged.AvgDuration = merged.TotalDuration / int64(merged.Count)
		}

		result = append(result, merged)
	}

	return result
}

// ClusterFoldersByName groups folder stats by name similarity
func ClusterFoldersByName(flags models.GlobalFlags, folders []models.FolderStats) []models.FolderStats {
	var groups [][]models.FolderStats
	metric := metrics.NewSorensenDice()

	for _, f := range folders {
		found := false
		name := filepath.Base(f.Path)
		if flags.TotalSizes { // Re-using flag for name clustering mode for now, or we can use another one
			name = f.Path
		}

		for i, group := range groups {
			prevName := filepath.Base(group[0].Path)
			if flags.TotalSizes {
				prevName = group[0].Path
			}

			if strutil.Similarity(name, prevName, metric) > 0.8 {
				groups[i] = append(groups[i], f)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, []models.FolderStats{f})
		}
	}

	return groupToStats(flags, groups)
}

// FilterNearDuplicates breaks down existing groups further by string similarity
func FilterNearDuplicates(groups []models.FolderStats) []models.FolderStats {
	var regrouped []models.FolderStats
	metric := metrics.NewSorensenDice()

	for _, group := range groups {
		tempGroups := make(map[string][]models.MediaWithDB)
		var keys []string

		for _, m := range group.Files {
			curr := strings.TrimSpace(m.Path)
			if curr == "" {
				continue
			}

			isDuplicate := false
			for _, prev := range keys {
				if strutil.Similarity(curr, prev, metric) > 0.8 { // 0.8 ratio
					tempGroups[prev] = append(tempGroups[prev], m)
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				tempGroups[curr] = []models.MediaWithDB{m}
				keys = append(keys, curr)
			}
		}

		for i, key := range keys {
			groupFiles := tempGroups[key]
			paths := make([]string, len(groupFiles))
			for j, f := range groupFiles {
				paths[j] = f.Path
			}
			common := utils.CommonPathFull(paths)
			if len(keys) > 1 {
				common = fmt.Sprintf("%s#%d", group.Path, i)
			}

			regrouped = append(regrouped, models.FolderStats{
				Path:  common,
				Files: groupFiles,
				Count: len(groupFiles),
			})
		}
	}

	return regrouped
}

type PathObservation struct {
	Path   string
	Vector clusters.Coordinates
}

func (p PathObservation) Coordinates() clusters.Coordinates {
	return p.Vector
}

func (p PathObservation) Distance(o clusters.Coordinates) float64 {
	return p.Vector.Distance(o)
}

// ClusterPaths groups lines of text using TF-IDF and KMeans
func ClusterPaths(flags models.GlobalFlags, lines []string) []models.FolderStats {
	if len(lines) < 2 {
		return []models.FolderStats{{Path: utils.CommonPathFull(lines), Files: wrapLines(lines), Count: len(lines)}}
	}

	k := flags.Clusters
	if k <= 0 {
		k = max(int(math.Sqrt(float64(len(lines)))), 2)
	}

	// Simple TF-IDF Vectorization
	corpus := make([][]string, len(lines))
	for i, line := range lines {
		corpus[i] = strings.Fields(utils.PathToSentence(line))
	}

	vocab := make(map[string]int)
	df := make(map[string]int)
	for _, doc := range corpus {
		seen := make(map[string]bool)
		for _, word := range doc {
			if _, ok := vocab[word]; !ok {
				vocab[word] = len(vocab)
			}
			if !seen[word] {
				df[word]++
				seen[word] = true
			}
		}
	}

	numDocs := float64(len(lines))
	var observations clusters.Observations
	for i, doc := range corpus {
		vec := make([]float64, len(vocab))
		tf := make(map[string]int)
		for _, word := range doc {
			tf[word]++
		}
		for word, count := range tf {
			tfScore := float64(count) / float64(len(doc))
			idfScore := math.Log(numDocs / float64(df[word]))
			vec[vocab[word]] = tfScore * idfScore
		}
		observations = append(observations, PathObservation{
			Path:   lines[i],
			Vector: clusters.Coordinates(vec),
		})
	}

	km := kmeans.New()
	clusters, err := km.Partition(observations, k)
	if err != nil {
		// Fallback to single group on error
		return []models.FolderStats{{Path: utils.CommonPathFull(lines), Files: wrapLines(lines), Count: len(lines)}}
	}

	var result []models.FolderStats
	for _, c := range clusters {
		var groupLines []string
		for _, obs := range c.Observations {
			if po, ok := obs.(PathObservation); ok {
				groupLines = append(groupLines, po.Path)
			}
		}
		if len(groupLines) > 0 {
			result = append(result, models.FolderStats{
				Path:  utils.CommonPathFull(groupLines),
				Files: wrapLines(groupLines),
				Count: len(groupLines),
			})
		}
	}

	return result
}

func wrapLines(lines []string) []models.MediaWithDB {
	res := make([]models.MediaWithDB, len(lines))
	for i, l := range lines {
		res[i] = models.MediaWithDB{Media: models.Media{Path: l}}
	}
	return res
}
