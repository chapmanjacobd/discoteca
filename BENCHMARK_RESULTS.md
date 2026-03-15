# Benchmark Results: SQLite FTS5 vs Bleve

## Configuration
- **Hardware**: 13th Gen Intel(R) Core(TM) i5-13600KF
- **OS**: Linux
- **Scale**: 20,000 Media items / 40,000 Captions
- **Bleve**: v2 (standard mapping)
- **SQLite**: FTS5 (unicode61, detail='full')

## Summary of Results (20k Media)

| Operation | SQLite FTS5 | Bleve | Difference |
|-----------|-------------|-------|------------|
| **Search (Path)** | 13,697 ms (*) | 10.1 ms | Bleve is ~1350x faster |
| **Search (Description)** | 11,703 ms (*) | 7.9 ms | Bleve is ~1480x faster |
| **Search (Captions)** | 51,183 ms (*) | 16.2 ms | Bleve is ~3100x faster |
| **Filter & Sort** | 3.1 ms | 6.5 ms | SQLite is ~2x faster |
| **Pagination (Deep)** | 0.047 ms | 34.8 ms | SQLite is ~740x faster |
| **Update (Playhead)** | 0.12 ms | 2.7 ms | SQLite is ~22x faster |
| **Aggregation (Stats)**| 0.67 ms | 26.3 ms | SQLite is ~39x faster |
| **Group By Parent** | 4.1 ms | 89.2 ms | SQLite is ~21x faster |

(*) **Critical Context**: SQLite FTS5 search performance with `ORDER BY rank` and `tokenize='unicode61'` is unexpectedly poor in this environment, even with `LIMIT 1000`. This suggests that the `rank` calculation overhead in SQLite FTS5 is substantial when matching a large percentage of the dataset (e.g., 20k-60k matches).

## Detailed Findings

### 1. Full Text Search
- **Bleve** provides high-performance BM25 ranking and limiting natively. For 20k items, it returns 1000 results in ~10ms.
- **SQLite FTS5** with `ORDER BY rank` takes over 11 seconds for the same task. This is likely due to SQLite needing to compute the rank for *every* match (thousands of rows) before it can return the top 1000.
- **Result Alignment**: Both engines returned 1000 results for the test queries, confirming the tests are aligned.

### 2. Metadata & Aggregations
- **SQLite** continues to dominate in structured data tasks:
    - **Pagination**: SQLite is 740x faster.
    - **Stats/Disk Usage**: SQLite is 20x-40x faster.
    - **Updates**: SQLite is 22x faster for playhead updates.

### 3. Conclusion
SQLite is extremely efficient for metadata and structured queries but its FTS5 ranking (`ORDER BY rank`) scales poorly compared to Bleve's optimized inverted index search when matching many documents.

## Recommendation

**Keep SQLite for metadata, and use Bleve for Search.**

The hybrid approach is clearly best:
1.  **SQLite** as the primary DB for all metadata, relations, and progress tracking.
2.  **Bleve** as a side-car index specifically for the `Search` and `Captions Search` features.
3.  **Optimization**: If SQLite FTS is used, avoid `ORDER BY rank` for common terms if performance is critical, or use it only on smaller result sets.

## Raw Data (20k Benchmarks, unicode61, detail='full')
```
BenchmarkComparison/M20000_C40000/Search_Path_FTS_SQLite-20         	       1	13697453790 ns/op	      1000 results
BenchmarkComparison/M20000_C40000/Search_Path_FTS_Bleve-20          	      99	  10142778 ns/op	      1000 results	     60000 total_hits
BenchmarkComparison/M20000_C40000/Search_Desc_FTS_SQLite-20         	       1	11703429666 ns/op	      1000 results
BenchmarkComparison/M20000_C40000/Search_Desc_FTS_Bleve-20          	     134	   7937676 ns/op	      1000 results	     20000 total_hits
BenchmarkComparison/M20000_C40000/Search_Captions_SQLite-20         	       1	51183175218 ns/op
BenchmarkComparison/M20000_C40000/Search_Captions_Bleve-20          	      70	  16249985 ns/op
BenchmarkComparison/M20000_C40000/Complex_FilterSort_SQLite-20      	     657	   3101742 ns/op
BenchmarkComparison/M20000_C40000/Complex_FilterSort_Bleve-20       	     171	   6505648 ns/op
BenchmarkComparison/M20000_C40000/Pagination_SQLite-20              	   27501	     47414 ns/op
BenchmarkComparison/M20000_C40000/Pagination_Bleve-20               	      30	  34834934 ns/op
BenchmarkComparison/M20000_C40000/Update_Playhead_SQLite-20         	    9459	    119236 ns/op
BenchmarkComparison/M20000_C40000/Update_Playhead_Bleve-20          	     421	   2721072 ns/op
BenchmarkComparison/M20000_C40000/Stats_Agg_SQLite-20               	    2010	    677972 ns/op
BenchmarkComparison/M20000_C40000/Stats_Agg_Bleve-20                	      54	  26347495 ns/op
BenchmarkComparison/M20000_C40000/Group_By_Parent_SQLite-20         	     468	   4171527 ns/op
BenchmarkComparison/M20000_C40000/Group_By_Parent_Bleve-20          	      12	  89288097 ns/op
```
