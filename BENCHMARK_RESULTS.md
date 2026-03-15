# Benchmark Results: SQLite FTS5 vs Bleve

## Configuration
- **Hardware**: 13th Gen Intel(R) Core(TM) i5-13600KF
- **OS**: Linux
- **Scale**: 10,000 and 100,000 Media items (20k/200k Captions)
- **Bleve**: v2 (standard mapping)
- **SQLite**: FTS5 (trigram, detail=none)

## Summary of Results (100k Media)

| Operation | SQLite FTS5 | Bleve | Difference |
|-----------|-------------|-------|------------|
| **Search (Common Term)** | 12,900 ms (*) | 13.5 ms | Bleve is ~1000x faster |
| **Search (Captions)** | 0.004 ms | 93.8 ms | SQLite is ~20,000x faster |
| **Filter & Sort** | 0.30 ms | 41.2 ms | SQLite is ~130x faster |
| **Pagination (Deep)** | 0.28 ms | 208.4 ms | SQLite is ~700x faster |
| **Update (Playhead)** | 0.36 ms | 7.2 ms | SQLite is ~20x faster |
| **Aggregation (Stats)**| 0.007 ms | 258.5 ms | SQLite is ~36,000x faster |

(*) *SQLite FTS5 search time is high because the current implementation fetches ALL matching rows (100k) to perform in-memory ranking. It does not use `LIMIT` at the database level for full-text search.*

## Detailed Findings

### 1. Full Text Search
- **Bleve** excels at full-text search relevance and performance for common terms due to its inverted index and efficient BM25 scoring.
- **SQLite FTS5** (as implemented) is slow for common terms because it lacks native BM25 ranking with `detail=none` (trigram) indexes, forcing the application to fetch all results and rank them in Go.
- **Fix**: SQLite FTS5 performance could be improved by using `LIMIT` in the SQL query, but this would sacrifice relevance ranking (random results).

### 2. Captions
- **SQLite** is incredibly fast (microseconds) because it uses `LIMIT` effectively and the index covers the query.
- **Bleve** is significantly slower (93ms), likely due to the overhead of searching a separate index or document set and joining results.

### 3. Updates (Progress Tracking)
- **SQLite** handles updates (playhead, time_last_played) with negligible cost (0.36ms).
- **Bleve** requires re-indexing the entire document for any field update, taking ~7ms per update. At high concurrency (e.g., 5 users/sec), this would add significant CPU load (35ms/sec) but is manageable. However, at 100 updates/sec, it would consume a full core.

### 4. Aggregation & Metadata
- **SQLite** is instant for aggregations (`GROUP BY`, `COUNT`) using standard indices.
- **Bleve** faceting is slow (250ms) for large datasets as it requires processing docValues for the entire result set (or all docs).

## Recommendation

**Do NOT remove SQLite.**

SQLite outperforms Bleve by orders of magnitude in:
- Metadata filtering and sorting
- Aggregations (Stats)
- Updates (Progress tracking)
- Caption search

Bleve is only superior for:
- Full-text search relevance (BM25)
- Fuzzy/Prefix search (if configured)
- "Common term" queries where `LIMIT` cannot be applied blindly.

**Hybrid Approach**:
- Keep **SQLite** as the primary source of truth, metadata storage, and aggregation engine.
- Keep **SQLite** for progress tracking (it is much more efficient).
- Use **Bleve** (optionally) for the specific `Search` API route to provide better relevance ranking, but consider the complexity/maintenance cost.
- **Fix**: The SQLite FTS5 search implementation needs to handle query tokenization correctly (as fixed in `internal/db/fts_queries.go`) to avoid "phrase query" errors with `detail=none`.

## Raw Data (100k Benchmarks)
```
BenchmarkComparison/M100000_C200000/Search_Media_FTS_SQLite-20                 1        12914595450 ns/op
BenchmarkComparison/M100000_C200000/Search_Media_FTS_Bleve-20                103          13463369 ns/op
BenchmarkComparison/M100000_C200000/Search_Captions_SQLite-20             323138              4387 ns/op
BenchmarkComparison/M100000_C200000/Search_Captions_Bleve-20                  12          93828875 ns/op
BenchmarkComparison/M100000_C200000/Complex_FilterSort_SQLite-20            3484            299129 ns/op
BenchmarkComparison/M100000_C200000/Complex_FilterSort_Bleve-20               28          41182543 ns/op
BenchmarkComparison/M100000_C200000/Pagination_SQLite-20                    3706            281797 ns/op
BenchmarkComparison/M100000_C200000/Pagination_Bleve-20                        7         208372186 ns/op
BenchmarkComparison/M100000_C200000/Update_Playhead_SQLite-20               3187            360108 ns/op
BenchmarkComparison/M100000_C200000/Update_Playhead_Bleve-20                 158           7223079 ns/op
BenchmarkComparison/M100000_C200000/Stats_Agg_SQLite-20                   215115              7070 ns/op
BenchmarkComparison/M100000_C200000/Stats_Agg_Bleve-20                         5         258515510 ns/op
```
