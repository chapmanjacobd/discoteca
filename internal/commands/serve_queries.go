package commands

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/models"
)

// QueryStats tracks slow query statistics
type QueryStats struct {
	mu         sync.RWMutex
	queries    []SlowQueryEntry
	enabled    bool
	maxEntries int
}

// SlowQueryEntry represents a single slow query record
type SlowQueryEntry struct {
	Query        string        `json:"query"`
	Args         []any         `json:"args,omitempty"`
	Duration     time.Duration `json:"duration_ms"`
	Timestamp    time.Time     `json:"timestamp"`
	DB           string        `json:"db"`
	RowsAffected int64         `json:"rows_affected,omitempty"`
}

// globalQueryStats holds application-wide query statistics
var globalQueryStats = &QueryStats{
	queries:    make([]SlowQueryEntry, 0, 1000),
	enabled:    true,
	maxEntries: 1000,
}

// SetQueryStatsEnabled enables or disables query statistics collection
func SetQueryStatsEnabled(enabled bool) {
	globalQueryStats.mu.Lock()
	defer globalQueryStats.mu.Unlock()
	globalQueryStats.enabled = enabled
}

// IsQueryStatsEnabled returns true if query statistics collection is enabled
func IsQueryStatsEnabled() bool {
	globalQueryStats.mu.RLock()
	defer globalQueryStats.mu.RUnlock()
	return globalQueryStats.enabled
}

// RecordSlowQuery records a slow query entry
func RecordSlowQuery(query string, args []any, duration time.Duration, dbPath string, rowsAffected int64) {
	globalQueryStats.mu.Lock()
	defer globalQueryStats.mu.Unlock()

	if !globalQueryStats.enabled {
		return
	}

	entry := SlowQueryEntry{
		Query:        query,
		Args:         args,
		Duration:     duration,
		Timestamp:    time.Now(),
		DB:           dbPath,
		RowsAffected: rowsAffected,
	}

	globalQueryStats.queries = append(globalQueryStats.queries, entry)

	// Trim old entries if we exceed max
	if len(globalQueryStats.queries) > globalQueryStats.maxEntries {
		globalQueryStats.queries = globalQueryStats.queries[len(globalQueryStats.queries)-globalQueryStats.maxEntries:]
	}
}

// GetQueryStats returns current query statistics
func GetQueryStats() []SlowQueryEntry {
	globalQueryStats.mu.RLock()
	defer globalQueryStats.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]SlowQueryEntry, len(globalQueryStats.queries))
	copy(result, globalQueryStats.queries)
	return result
}

// ClearQueryStats clears all recorded query statistics
func ClearQueryStats() {
	globalQueryStats.mu.Lock()
	defer globalQueryStats.mu.Unlock()
	globalQueryStats.queries = make([]SlowQueryEntry, 0, 1000)
}

// QueryStatsResponse is the response for the /api/queries endpoint
type QueryStatsResponse struct {
	Queries      []SlowQueryEntry `json:"queries"`
	TotalCount   int              `json:"total_count"`
	SlowestQuery *SlowQueryEntry  `json:"slowest_query,omitempty"`
	AvgDuration  float64          `json:"avg_duration_ms"`
	StartTime    int64            `json:"start_time"` // Unix timestamp to avoid JSON marshaling issues
}

// handleQueries handles the /api/queries endpoint for slow query dashboard
func (c *ServeCmd) handleQueries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	action := q.Get("action")

	switch action {
	case "clear":
		ClearQueryStats()
		sendJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
		return

	case "toggle":
		enabled := IsQueryStatsEnabled()
		SetQueryStatsEnabled(!enabled)
		sendJSON(w, http.StatusOK, map[string]bool{"enabled": !enabled})
		return

	default:
		// Return query statistics
		entries := GetQueryStats()

		// Sort by duration descending (slowest first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Duration > entries[j].Duration
		})

		// Calculate statistics
		var totalDuration time.Duration
		var slowestQuery *SlowQueryEntry
		if len(entries) > 0 {
			slowestQuery = &entries[0]
			for _, e := range entries {
				totalDuration += e.Duration
			}
		}

		avgDuration := float64(0)
		if len(entries) > 0 {
			avgDuration = float64(totalDuration) / float64(len(entries)) / float64(time.Millisecond)
		}

		resp := QueryStatsResponse{
			Queries:      entries,
			TotalCount:   len(entries),
			SlowestQuery: slowestQuery,
			AvgDuration:  avgDuration,
			StartTime:    c.ApplicationStartTime,
		}

		sendJSON(w, http.StatusOK, resp)
	}
}

// analyzeIndexUsage returns index usage statistics for the media table
func (c *ServeCmd) analyzeIndexUsage(ctx context.Context, dbPath string) ([]models.IndexStat, error) {
	var results []models.IndexStat

	err := c.execDB(ctx, dbPath, func(sqlDB *sql.DB) error {
		// Get index list
		rows, err := sqlDB.Query("SELECT name, tbl_name FROM sqlite_master WHERE type='index' AND tbl_name='media' AND name NOT LIKE 'sqlite_%'")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var indexName, tableName string
			if err := rows.Scan(&indexName, &tableName); err != nil {
				return err
			}

			// Get index stats using sqlite_stat1 if available
			var sizeEst int64
			statRows, err := sqlDB.Query("SELECT stat FROM sqlite_stat1 WHERE idx=?", indexName)
			if err == nil {
				defer statRows.Close()
				if statRows.Next() {
					var statStr string
					if err := statRows.Scan(&statStr); err == nil {
						// stat format is "rows pages leaf_pages"
						// We'll just store it as-is for now
						sizeEst = 1 // Placeholder
					}
				}
			}

			results = append(results, models.IndexStat{
				IndexName: indexName,
				TableName: tableName,
				SizeEst:   sizeEst,
			})
		}

		return nil
	})

	return results, err
}

// TimedQuery executes a query function and records timing if it exceeds the threshold
func TimedQuery[T any](ctx context.Context, dbPath string, query string, args []any, fn func() (T, error)) (T, error) {
	start := time.Now()
	result, err := fn()
	duration := time.Since(start)

	// Record slow queries
	if duration > db.SlowQueryThreshold && IsQueryStatsEnabled() {
		var rowsAffected int64
		if err == nil {
			// For queries that return rows, we can't easily count them here
			// This is best effort - the caller could provide this info
			rowsAffected = 0
		}
		RecordSlowQuery(query, args, duration, dbPath, rowsAffected)
	}

	return result, err
}
