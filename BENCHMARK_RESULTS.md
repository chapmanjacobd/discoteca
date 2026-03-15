# Benchmark Results: SQLite FTS5 vs Bleve

## Configuration
- **Hardware**: 13th Gen Intel(R) Core(TM) i5-13600KF
- **OS**: Linux
- **Scale**: 200,000 Media items / 400,000 Captions
- **Bleve**: v2 (standard mapping)
- **SQLite**: FTS5 (trigram, detail='full')

## Summary of Results (200k Media)

| Operation | SQLite FTS5 | Bleve | Difference |
|-----------|-------------|-------|------------|
| **Search (Common Term)** | 5,208 ms (*) | 25.2 ms | Bleve is ~200x faster |
| **Search (Captions)** | 7,727 ms (*) | 119.4 ms | Bleve is ~65x faster |
| **Filter & Sort** | 56.8 ms | 61.6 ms | Comparable |
| **Pagination (Deep)** | 0.05 ms | 259.2 ms | SQLite is ~5000x faster |
| **Update (Playhead)** | 0.46 ms | 2.7 ms | SQLite is ~5.8x faster |
| **Aggregation (Stats)**| 6.7 ms | 202.2 ms | SQLite is ~30x faster |
| **Group By Parent** | 65.1 ms | 530.7 ms | SQLite is ~8x faster |

(*) **Critical Context**: The high latency for SQLite Search operations is because the current implementation **fetches ALL matching rows** (e.g., 200,000 rows for "common term") to perform in-memory ranking in Go. It explicitly **ignores the `LIMIT` parameter** in the SQL query. If `LIMIT` were applied at the database level (as in the raw SQL benchmark in previous tests), SQLite returns results in microseconds (0.004ms).

## Detailed Findings

### 1. Full Text Search & Captions
- **Bleve** is significantly faster for broad queries because it natively handles ranking and limiting (BM25) efficiently.
- **SQLite** (as implemented) is slow for broad queries because it retrieves the entire result set to rank them manually.
- **Optimization Opportunity**: If we accept "random" or "boolean-only" ranking for common terms, applying `LIMIT` to the SQLite query would make it orders of magnitude faster than Bleve.

### 2. Disk Usage & Aggregation
- **Group By Parent**: SQLite (65ms) is ~8x faster than Bleve (530ms) for aggregating disk usage by directory, even with complex string manipulation in SQL. Bleve requires client-side aggregation after fetching documents or facets.
- **Stats**: SQLite (6.7ms) is ~30x faster for standard metadata aggregations.

### 3. Updates (Progress Tracking)
- **SQLite** handles high-frequency updates (e.g., `playhead` tracking) efficiently (0.46ms).
- **Bleve** requires re-indexing the document (2.7ms), which adds load at high concurrency.

### 4. Metadata & Pagination
- **Pagination**: SQLite is instant (50µs) for deep pagination, while Bleve struggles (259ms).
- **Filter & Sort**: For complex queries, SQLite and Bleve showed comparable performance (~60ms) in this run, likely due to the specific index usage plan for the "video" type filter.

## Recommendation

**Do NOT remove SQLite.**

The benchmarks confirm that SQLite is the optimal engine for:
- Metadata management
- Filtering and sorting
- Aggregations and Disk Usage analysis
- Progress tracking
- Deep pagination

**Bleve Integration Strategy**:
- Use **Bleve** *only* for the specific `Search` endpoints where full-text relevance (ranking) is critical.
- Keep **SQLite** for everything else.
- **Action Item**: Consider optimizing the SQLite FTS implementation to use `LIMIT` for queries where strict ranking is less important, or pre-filter results to avoid fetching 200k rows.

## Raw Data (200k Benchmarks, detail='full')
```
BenchmarkComparison/M200000_C400000/Search_Media_FTS_SQLite-20                 1        5208839976 ns/op
BenchmarkComparison/M200000_C400000/Search_Media_FTS_Bleve-20                 42          25176828 ns/op
BenchmarkComparison/M200000_C400000/Search_Captions_SQLite-20                  1        7727777484 ns/op
BenchmarkComparison/M200000_C400000/Search_Captions_Bleve-20                  10         119412696 ns/op
BenchmarkComparison/M200000_C400000/Complex_FilterSort_SQLite-20              24          56811385 ns/op
BenchmarkComparison/M200000_C400000/Complex_FilterSort_Bleve-20               21          61578553 ns/op
BenchmarkComparison/M200000_C400000/Pagination_SQLite-20                   25047             51938 ns/op
BenchmarkComparison/M200000_C400000/Pagination_Bleve-20                        4         259159589 ns/op
BenchmarkComparison/M200000_C400000/Update_Playhead_SQLite-20               2784            462576 ns/op
BenchmarkComparison/M200000_C400000/Update_Playhead_Bleve-20                 466           2730647 ns/op
BenchmarkComparison/M200000_C400000/Stats_Agg_SQLite-20                      169           6650656 ns/op
BenchmarkComparison/M200000_C400000/Stats_Agg_Bleve-20                         6         202171805 ns/op
BenchmarkComparison/M200000_C400000/Group_By_Parent_SQLite-20                 18          65138286 ns/op
BenchmarkComparison/M200000_C400000/Group_By_Parent_Bleve-20                   2         530730085 ns/op
```
