package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/aggregate"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/pathutil"
	"github.com/chapmanjacobd/discoteca/internal/query"
)

func (c *Server) HandleDU(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	if rejectInvalidFlags(w, flags) {
		return
	}
	// DU aggregation uses custom SQL without an FTS join, so it must use the
	// substring path even when the client asks for FTS.
	flags.FTS = false
	flags.NoFTS = true
	flags.SearchModeExplicit = true
	path := r.URL.Query().Get("path")
	includeCounts := r.URL.Query().Get("include_counts") == "true"

	cleanPath := c.normalizeDUPath(path)
	targetDepth := c.calculateDUTargetDepth(cleanPath)

	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}

	resolvedFlags, err := query.ResolvePercentileFlags(r.Context(), dbs, flags)
	if err != nil {
		logging.Log.Warn("Failed to resolve percentile filters", "error", err)
		resolvedFlags = flags
	}

	folderResults, err := query.AggregateDUByPathMultiDBWithFilters(
		r.Context(),
		dbs,
		cleanPath,
		targetDepth,
		resolvedFlags,
	)
	if err != nil {
		logging.Log.Error("Failed to fetch DU folders", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	directFiles, err := query.FetchDUDirectFilesWithFilters(
		r.Context(),
		dbs,
		cleanPath,
		targetDepth,
		resolvedFlags,
	)
	if err != nil {
		logging.Log.Error("Failed to fetch DU files", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	// Re-check every filter against the final per-file rows. SQL covers the
	// filterable predicates; this catches filesystem-only checks such as Exists.
	directFiles = query.FilterMedia(directFiles, resolvedFlags)

	folders := c.convertDUFolderResults(folderResults)

	sortBy, reverse := c.parseDUSortParams(r)
	query.SortFolders(folders, sortBy, reverse)
	c.sortDUFiles(directFiles, sortBy, reverse)

	limit, offset := c.parseDUPagination(r)
	totalCount := len(folders) + len(directFiles)
	folders, directFiles = c.applyDUPagination(folders, directFiles, offset, limit)

	response := models.DUResponse{
		Folders:     folders,
		Files:       directFiles,
		FolderCount: len(folders),
		FileCount:   len(directFiles),
		TotalCount:  totalCount,
	}

	if includeCounts {
		response.Counts = c.calculateFilterCounts(r.Context(), resolvedFlags, dbs)
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		if isClientDisconnect(err) {
			logging.Log.Debug("Client disconnected while encoding response", "error", err)
		} else {
			logging.Log.Warn("Failed to encode response", "error", err)
		}
	}
}

// normalizeDUPath converts URL path to filesystem path
func (c *Server) normalizeDUPath(path string) string {
	cleanPath := pathutil.FromURL(path)
	if cleanPath == "." || cleanPath == "/" || cleanPath == "\\" {
		cleanPath = ""
	}
	return cleanPath
}

// calculateDUTargetDepth calculates the depth of the path and returns targetDepth (currentDepth + 1)
func (c *Server) calculateDUTargetDepth(cleanPath string) int {
	currentDepth := 0
	if cleanPath != "" {
		parts := strings.FieldsFunc(cleanPath, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		for _, p := range parts {
			if p != "" {
				currentDepth++
			}
		}
	}
	return currentDepth + 1
}

// convertDUFolderResults converts folder query results to FolderStats slice
func (c *Server) convertDUFolderResults(folderResults []query.DUQueryResult) []models.FolderStats {
	folders := make([]models.FolderStats, 0, len(folderResults))
	for _, r := range folderResults {
		folders = append(folders, models.FolderStats{
			Path:          r.Path,
			Count:         r.Count,
			TotalSize:     r.TotalSize,
			TotalDuration: r.TotalDuration,
		})
	}
	return folders
}

// parseDUSortParams extracts sort and reverse parameters from request.
// The default DU order is itself a meaningful sort, so reverse is preserved
// even when no explicit sort key is requested.
func (c *Server) parseDUSortParams(r *http.Request) (sortBy string, reverse bool) {
	sortBy = r.URL.Query().Get("sort")
	reverse = r.URL.Query().Get("reverse") == "true"

	if sortBy == "default" {
		sortBy = ""
	}
	return sortBy, reverse
}

// sortDUFiles sorts direct files using the given sort parameters
func (c *Server) sortDUFiles(directFiles []models.MediaWithDB, sortBy string, reverse bool) {
	less := func(i, j int) bool {
		switch sortBy {
		case "size":
			iSize := int64(0)
			jSize := int64(0)
			if directFiles[i].Size != nil {
				iSize = *directFiles[i].Size
			}
			if directFiles[j].Size != nil {
				jSize = *directFiles[j].Size
			}
			return iSize < jSize
		case "duration":
			iDur := int64(0)
			jDur := int64(0)
			if directFiles[i].Duration != nil {
				iDur = *directFiles[i].Duration
			}
			if directFiles[j].Duration != nil {
				jDur = *directFiles[j].Duration
			}
			return iDur < jDur
		case "path", "name":
			return directFiles[i].Path < directFiles[j].Path
		case "title":
			iTitle := ""
			jTitle := ""
			if directFiles[i].Title != nil {
				iTitle = *directFiles[i].Title
			}
			if directFiles[j].Title != nil {
				jTitle = *directFiles[j].Title
			}
			return iTitle < jTitle
		default:
			return directFiles[i].Path < directFiles[j].Path
		}
	}

	if reverse {
		sort.SliceStable(directFiles, func(i, j int) bool { return less(j, i) })
		return
	}
	sort.SliceStable(directFiles, less)
}

// parseDUPagination extracts limit and offset from request.
// all=true (or a missing/zero limit from the caller) disables pagination.
func (c *Server) parseDUPagination(r *http.Request) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	if r.URL.Query().Get("all") == "true" {
		return 0, 0
	}

	limit = 100
	offset = 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if c.MaxLimit > 0 && l > c.MaxLimit {
				l = c.MaxLimit
			}
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	return limit, offset
}

// applyDUPagination applies pagination to folders and files slices
func (c *Server) applyDUPagination(
	folders []models.FolderStats,
	directFiles []models.MediaWithDB,
	offset int,
	limit int,
) ([]models.FolderStats, []models.MediaWithDB) {
	if limit <= 0 {
		return folders, directFiles
	}
	if offset >= len(folders) {
		fileStart := offset - len(folders)
		fileEnd := fileStart + limit
		if fileStart >= len(directFiles) {
			return folders[len(folders):], directFiles[len(directFiles):]
		}
		if fileEnd > len(directFiles) {
			fileEnd = len(directFiles)
		}
		return folders[len(folders):], directFiles[fileStart:fileEnd]
	}

	folderEnd := offset + limit
	if folderEnd > len(folders) {
		fileEnd := min(folderEnd-len(folders), len(directFiles))
		return folders[offset:], directFiles[0:fileEnd]
	}
	return folders[offset:folderEnd], directFiles
}

func (c *Server) HandleEpisodes(w http.ResponseWriter, r *http.Request) {
	flags := c.ParseFlags(r)
	if rejectInvalidFlags(w, flags) {
		return
	}
	// Group view renders playable media cards, so sidecar subtitle files belong
	// in captions, not as media cards or folder counts.
	flags.HideSubtitles = true

	dbs, err := c.getDBs(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database filter: %v", err), http.StatusBadRequest)
		return
	}
	c.applySearchMode(r.Context(), &flags, dbs)

	// Grouping happens after the media query, so fetch the complete filtered
	// set and paginate the resulting folder groups below. Paginating media
	// rows first would drop whole groups or split their counts.
	fetchFlags := flags
	fetchFlags.All = true
	fetchFlags.Limit = 0
	fetchFlags.Offset = 0

	allMedia, err := query.MediaQuery(r.Context(), dbs, fetchFlags)
	if err != nil {
		logging.Log.Error("Failed to fetch media for episodes", "error", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	results := aggregate.GroupMediaByParent(allMedia)

	// GroupMediaByParent sorts folders by path. Apply the selected sort to the
	// folder sequence as well as to the files inside each folder.
	switch flags.SortBy {
	case "size", "duration":
		sort.SliceStable(results, func(i, j int) bool {
			left, right := results[i].TotalSize, results[j].TotalSize
			if flags.SortBy == "duration" {
				left, right = results[i].TotalDuration, results[j].TotalDuration
			}
			if left == right {
				return results[i].Path < results[j].Path
			}
			if flags.Reverse {
				return left > right
			}
			return left < right
		})
	default:
		// Use the position of each folder's first file in the already-sorted
		// media list. This covers path, random, and the remaining fields.
		firstSeen := make(map[string]int, len(results))
		for i, item := range allMedia {
			parent := item.Parent()
			if _, ok := firstSeen[parent]; !ok {
				firstSeen[parent] = i
			}
		}
		sort.SliceStable(results, func(i, j int) bool {
			left, leftOK := firstSeen[results[i].Path]
			right, rightOK := firstSeen[results[j].Path]
			if !leftOK || !rightOK {
				return results[i].Path < results[j].Path
			}
			return left < right
		})
	}

	totalCount := len(results)
	if !flags.All && flags.Limit > 0 {
		start := flags.Offset
		if start > len(results) {
			results = []models.FolderStats{}
		} else {
			end := min(start+flags.Limit, len(results))
			results = results[start:end]
		}
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.Log.Warn("Failed to encode results", "error", err)
	}
}
