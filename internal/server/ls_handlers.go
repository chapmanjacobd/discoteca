package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/logging"
)

func (c *Server) HandleLs(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	isPartial, searchDir, searchBase := c.parseLsPath(path)

	resultsMap := make(map[string]LsEntry)
	counts := make(map[string]int)

	for _, dbPath := range c.Databases {
		err := c.execDB(r.Context(), dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			rows, err := c.queryLsRows(ctx, lsQueryParams{
				SQLDB:      sqlDB,
				IsPartial:  isPartial,
				SearchDir:  searchDir,
				SearchBase: searchBase,
			})
			if err != nil {
				return err
			}
			defer rows.Close()

			c.processLsRows(lsRowProcessParams{
				Rows:       rows,
				ResultsMap: &resultsMap,
				Counts:     counts,
				IsPartial:  isPartial,
				OrigPath:   path,
				SearchDir:  searchDir,
			})
			return rows.Err()
		})
		if err != nil {
			logging.Log.Error("handleLs DB query failed", "db", dbPath, "error", err)
		}
	}

	results := c.buildLsResults(resultsMap, counts)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		logging.Log.Warn("Failed to encode results", "error", err)
	}
}

// parseLsPath parses the input path and returns isPartial flag, searchDir, and searchBase
func (c *Server) parseLsPath(path string) (isPartial bool, searchDir, searchBase string) {
	if strings.HasPrefix(path, "./") {
		isPartial = true
		path = strings.TrimPrefix(path, "./")
	} else if !strings.HasPrefix(path, "/") {
		isPartial = true
	}

	if isPartial {
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash != -1 {
			searchDir = path[:lastSlash+1]
			searchBase = path[lastSlash+1:]
		} else {
			searchBase = path
		}
	} else {
		searchDir = path
		searchBase = ""
		if !strings.HasSuffix(path, "/") {
			lastSlash := strings.LastIndex(path, "/")
			if lastSlash != -1 {
				searchDir = path[:lastSlash+1]
				searchBase = path[lastSlash+1:]
			} else {
				searchDir = "/"
				searchBase = path[1:]
			}
		}
	}
	return isPartial, searchDir, searchBase
}

type lsQueryParams struct {
	SQLDB      *sql.DB
	IsPartial  bool
	SearchDir  string
	SearchBase string
}

// queryLsRows executes the appropriate SQL query based on search mode
func (c *Server) queryLsRows(ctx context.Context, params lsQueryParams) (*sql.Rows, error) {
	if params.IsPartial {
		if params.SearchDir == "" {
			return params.SQLDB.QueryContext(ctx, `
				SELECT path, media_type FROM media
				WHERE time_deleted = 0
				  AND path LIKE '%' || ? || '%'
				LIMIT 500`, params.SearchBase)
		}
		return params.SQLDB.QueryContext(ctx, `
			SELECT path, media_type FROM media
			WHERE time_deleted = 0
			  AND path LIKE '%' || ? || '%' || ? || '%'
			LIMIT 500`, params.SearchDir, params.SearchBase)
	}

	if params.SearchBase == "" {
		return params.SQLDB.QueryContext(ctx, `
			SELECT path, media_type FROM media
			WHERE time_deleted = 0
			  AND path LIKE ? || '%'
			LIMIT 500`, params.SearchDir)
	}
	return params.SQLDB.QueryContext(ctx, `
		SELECT path, media_type FROM media
		WHERE time_deleted = 0
		  AND path LIKE ? || '%'
		  AND path LIKE '%' || ? || '%'
		LIMIT 500`, params.SearchDir, params.SearchBase)
}

type lsRowProcessParams struct {
	Rows       *sql.Rows
	ResultsMap *map[string]LsEntry
	Counts     map[string]int
	IsPartial  bool
	OrigPath   string
	SearchDir  string
}

// processLsRows processes SQL rows and populates resultsMap and counts
func (c *Server) processLsRows(params lsRowProcessParams) {
	for params.Rows.Next() {
		var p, t sql.NullString
		if err := params.Rows.Scan(&p, &t); err != nil || !p.Valid {
			continue
		}
		fullPath := p.String

		entryName, entryPath, isDir := c.extractLsEntry(fullPath, params.IsPartial, params.OrigPath, params.SearchDir)
		if entryName == "" {
			continue
		}

		c.mergeLsEntry(lsMergeParams{
			ResultsMap: params.ResultsMap,
			Counts:     params.Counts,
			EntryName:  entryName,
			EntryPath:  entryPath,
			IsDir:      isDir,
			MediaType:  t.String,
		})
	}
}

// extractLsEntry extracts the entry name, path, and isDir flag from a full path
func (c *Server) extractLsEntry(
	fullPath string,
	isPartial bool,
	origPath string,
	searchDir string,
) (entryName, entryPath string, isDir bool) {
	if isPartial {
		matchStr := origPath
		if searchDir != "" {
			matchStr = searchDir
		}

		idx := strings.Index(fullPath, matchStr)
		if idx == -1 {
			return "", "", false
		}

		var prefix string
		var remaining string

		if strings.HasSuffix(matchStr, "/") {
			prefix = fullPath[:idx+len(matchStr)]
			remaining = fullPath[idx+len(matchStr):]
		} else {
			lastSlash := strings.LastIndex(fullPath[:idx], "/")
			if lastSlash == -1 {
				lastSlash = 0
			}
			prefix = fullPath[:lastSlash+1]
			remaining = fullPath[lastSlash+1:]
		}

		if remaining == "" {
			return "", "", false
		}

		if before, _, ok := strings.Cut(remaining, "/"); ok {
			return before, prefix + before + "/", true
		}
		return remaining, prefix + remaining, false
	}

	// Absolute path
	if !strings.HasPrefix(fullPath, searchDir) {
		return "", "", false
	}
	suffix := strings.TrimPrefix(fullPath, searchDir)
	if suffix == "" {
		return "", "", false
	}
	if before, _, ok := strings.Cut(suffix, "/"); ok {
		return before, searchDir + before + "/", true
	}
	return suffix, searchDir + suffix, false
}

type lsMergeParams struct {
	ResultsMap *map[string]LsEntry
	Counts     map[string]int
	EntryName  string
	EntryPath  string
	IsDir      bool
	MediaType  string
}

// mergeLsEntry adds or updates an entry in resultsMap
func (c *Server) mergeLsEntry(params lsMergeParams) {
	params.Counts[params.EntryPath]++
	if existing, ok := (*params.ResultsMap)[params.EntryPath]; ok {
		if !existing.IsDir && params.IsDir {
			(*params.ResultsMap)[params.EntryPath] = LsEntry{
				Name:  params.EntryName,
				Path:  params.EntryPath,
				IsDir: true,
			}
		}
	} else {
		(*params.ResultsMap)[params.EntryPath] = LsEntry{
			Name:      params.EntryName,
			Path:      params.EntryPath,
			IsDir:     params.IsDir,
			MediaType: params.MediaType,
		}
	}
}

// buildLsResults sorts and truncates the results
func (c *Server) buildLsResults(
	resultsMap map[string]LsEntry,
	counts map[string]int,
) []LsEntry {
	results := make([]LsEntry, 0, len(resultsMap))
	for _, entry := range resultsMap {
		results = append(results, entry)
	}

	sort.Slice(results, func(i, j int) bool {
		countI := counts[results[i].Path]
		countJ := counts[results[j].Path]
		if countI != countJ {
			return countI > countJ
		}
		if results[i].IsDir != results[j].IsDir {
			return results[i].IsDir
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	if len(results) > 20 {
		results = results[:20]
	}
	return results
}
