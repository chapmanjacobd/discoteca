# Benchmark Results: SQLite FTS5 vs Bleve

## Configuration
- **Hardware**: 13th Gen Intel(R) Core(TM) i5-13600KF
- **OS**: Linux
- **Bleve**: v2 (standard mapping)
- **SQLite**: FTS5 (trigram, detail='full')

---

## Results (March 15, 2026) - Scale: 80,000 Media / 160,000 Captions

| Operation | SQLite (avg) | Bleve (avg) | Winner | Mem SQLite | Mem Bleve | Thrash SQLite | Thrash Bleve |
|-----------|--------------|-------------|--------|------------|-----------|---------------|--------------|
| **Search (Path)** | 182.2 ms | 17.4 ms | Bleve ~10x | 21.8 MB | 2.9 MB | 457k | 82k |
| **Search (Description)** | 382.5 ms | 9.7 ms | Bleve ~39x | 3.3 MB | 0.3 MB | 19k | 2k |
| **Search (Captions)** | 313.1 ms | 68.8 ms | Bleve ~4.5x | 20 KB | 24.5 MB | 283 | 322k |
| **Filter & Sort** | 15.0 ms | 28.8 ms | SQLite ~2x | 1.4 KB | 5.6 MB | 54 | 60k |
| **Aggregation (Stats)**| 3.7 ms | 90.9 ms | SQLite ~24x | 0.6 KB | 46.3 MB | 27 | 800k |
| **Group By Parent** | 22.6 ms | 344.1 ms | SQLite ~15x | 0.6 KB | 199.5 MB | 18 | 2M |

### Analysis (M80000_C160000)

**Search Operations:**
- Bleve dominates full-text search at scale
- Description search: Bleve is 39x faster (9.7ms vs 382ms)
- Path search: Bleve is 10x faster (17ms vs 182ms)
- Caption search: Bleve is 4.5x faster (69ms vs 313ms)

**Metadata Operations:**
- SQLite is the clear winner for all non-search workloads
- Aggregations: SQLite is 24x faster (3.7ms vs 91ms)
- Grouping: SQLite is 15x faster (23ms vs 344ms)
- Filter & Sort: SQLite is 2x faster (15ms vs 29ms)

**Memory Efficiency:**
- SQLite remains extremely memory-efficient at scale
- Bleve Group By Parent allocates 199.5 MB vs SQLite's 0.6 KB (330,000x more!)
- Bleve Stats Aggregation allocates 46.3 MB vs SQLite's 0.6 KB

**Allocation Thrashing (allocs/op):**
- SQLite: 18-457k allocs/op (scales with query complexity)
- Bleve: 2k-2M allocs/op (extreme GC pressure for aggregations)
- Bleve Group By Parent: 2 million allocations per operation

---

## Recommendation

**Hybrid Approach Mandatory at Scale:**

1. **Use Bleve for search queries** - 4.5-39x faster, essential for responsive UX
2. **Use SQLite for everything else** - 2-24x faster with dramatically lower memory pressure
3. **Avoid Bleve aggregations** - 199MB allocations and 2M allocs/op will cause severe GC pauses

**Implementation Strategy:**
- Store all data in SQLite
- Index search fields (Path, Title, Description, Captions) in Bleve
- Route search queries to Bleve, all metadata/aggregation queries to SQLite
- Consider caching frequently-accessed aggregations in SQLite

---

## Raw Benchmark Output

```
BenchmarkComparison/M80000_C160000/Search_Path_FTS_SQLite-20         	       6	 182206318 ns/op	    1000 results	21846680 B/op	     457070 allocs/op
BenchmarkComparison/M80000_C160000/Search_Path_FTS_Bleve-20          	      66	  17423104 ns/op	    1000 results	     80000 total_hits	 2865491 B/op	      82095 allocs/op
BenchmarkComparison/M80000_C160000/Search_Desc_FTS_SQLite-20         	       4	 382523386 ns/op	    1000 results	 3293036 B/op	      19095 allocs/op
BenchmarkComparison/M80000_C160000/Search_Desc_FTS_Bleve-20          	     127	   9658244 ns/op	    1000 results	     80000 total_hits	  321194 B/op	       2162 allocs/op
BenchmarkComparison/M80000_C160000/Search_Captions_SQLite-20         	       4	 313103643 ns/op	   20032 B/op	        283 allocs/op
BenchmarkComparison/M80000_C160000/Search_Captions_Bleve-20          	      16	  68846315 ns/op	24529950 B/op	     321524 allocs/op
BenchmarkComparison/M80000_C160000/Complex_FilterSort_SQLite-20      	      98	  15045123 ns/op	    1392 B/op	         54 allocs/op
BenchmarkComparison/M80000_C160000/Complex_FilterSort_Bleve-20       	      36	  28825688 ns/op	 5562818 B/op	      60190 allocs/op
BenchmarkComparison/M80000_C160000/Stats_Agg_SQLite-20               	     334	   3744830 ns/op	     624 B/op	         27 allocs/op
BenchmarkComparison/M80000_C160000/Stats_Agg_Bleve-20                	      12	  90933318 ns/op	46292184 B/op	     800262 allocs/op
BenchmarkComparison/M80000_C160000/Group_By_Parent_SQLite-20         	      48	  22551142 ns/op	     592 B/op	         18 allocs/op
BenchmarkComparison/M80000_C160000/Group_By_Parent_Bleve-20          	       3	 344138438 ns/op	199524394 B/op	    2020529 allocs/op
```
