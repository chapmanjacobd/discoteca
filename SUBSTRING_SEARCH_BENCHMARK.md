# Substring Search Options for Non-FTS Builds

## Performance Benchmark Comparison

This document compares different approaches for implementing substring search in SQLite when FTS5 is not available.

## Benchmark Results (10,000 rows, 1s benchtime)

### Fastest to Slowest (by time per operation, lower is better)

| Method | Pattern | Avg Time (ns/op) | Avg Time (μs) | Relative Speed | Can Use Index |
|--------|---------|------------------|---------------|----------------|---------------|
| **FTS5 NEAR** | `media NEAR videos` | ~107,500 | 108μs | 0.4x | ✅ (FTS index) |
| **Indexed Equality** | `type = ?` | ~276,500 | 277μs | 1.0x (baseline) | ✅ Yes |
| **LIKE StartsWith** | `/media%` | ~498,800 | 499μs | 1.8x | ✅ Yes |
| **LIKE Prefix** | `/media%` | ~536,700 | 537μs | 1.9x | ✅ Yes |
| **LIKE EndsWith** | `%.mkv` | ~661,100 | 661μs | 2.4x | ❌ No |
| **FTS5 Join** | `fts_path MATCH ?` | ~676,800 | 677μs | 2.4x | ✅ (FTS index) |
| **FTS5 In Subquery** | `fts_path MATCH ?` | ~706,900 | 707μs | 2.6x | ✅ (FTS index) |
| **LIKE Complex** | `LIKE ? AND NOT LIKE ?` | ~908,100 | 908μs | 3.3x | ❌ No |
| **INSTR** | `INSTR(path, ?)` | ~949,100 | 949μs | 3.4x | ❌ No |
| **LIKE Substring** | `%Matrix%` | ~1,034,900 | 1,035μs | 3.7x | ❌ No |
| **LIKE Multiple Patterns** | `LIKE ? AND LIKE ?` | ~547,500 | 548μs | 2.0x | ❌ No |
| **LIKE Or Patterns** | `LIKE ? OR LIKE ?` | ~1,440,800 | 1,441μs | 5.2x | ❌ No |
| **LIKE Multiple Cols** | `path LIKE ? OR title LIKE ?` | ~2,137,600 | 2,138μs | 7.7x | ❌ No |
| **FTS5 Prefix** | `med*` | ~2,496,500 | 2,497μs | 9.0x | ✅ (FTS index) |
| **FTS5 Trigram** | `med` (trigram) | ~2,118,200 | 2,118μs | 7.7x | ✅ (FTS index) |
| **FTS5 Match** | `media videos` | ~4,158,300 | 4,158μs | 15.0x | ✅ (FTS index) |
| **FTS5 Multiple Terms** | `test AND media` | ~3,940,700 | 3,941μs | 14.2x | ✅ (FTS index) |
| **FTS5 Composite** | Multiple MATCH | ~4,533,800 | 4,534μs | 16.4x | ✅ (FTS index) |
| **FTS5 Or Terms** | `test OR media` | ~5,557,800 | 5,558μs | 20.1x | ✅ (FTS index) |
| **FTS5 Phrase** | `"media videos"` | ~5,536,800 | 5,537μs | 20.0x | ✅ (FTS index) |

### Result Fetch Performance

| Method | Pattern | Avg Time (ns/op) | Notes |
|--------|---------|------------------|-------|
| **LIKE Small Result** | `%Matrix%` (few rows) | ~9,780 | Returning few rows |
| **FTS5 Small Result** | `Matrix` (few rows) | ~18,868 | Returning few rows |
| **LIKE Large Result** | `%test%` (many rows) | ~10,303 | Returning many rows |
| **FTS5 Large Result** | `test` (many rows) | ~20,932 | Returning many rows |

### Key Findings

1. **For Non-FTS Builds (LIKE-based)**:
   - **LIKE with prefix** (`/media%`) is the fastest option when you can use it (~500μs)
   - **LIKE with substring** (`%Matrix%`) is ~2x slower than prefix but still reasonable (~1ms)
   - **INSTR()** performs similarly to LIKE for substring searches (~950μs)
   - **Multiple column searches** (OR conditions) significantly impact performance (~2.1ms)
   - **EndsWith patterns** (`%.mkv`) perform better than general substring (~660μs)

2. **For FTS5 Builds**:
   - **FTS5 NEAR operator** is surprisingly fast (~108μs) - fastest search method
   - **FTS5 Join pattern** (~677μs) is slightly faster than subquery pattern (~707μs)
   - **Simple MATCH** queries are slower than basic LIKE for single-column searches
   - **Complex FTS5 queries** (phrase, OR, AND) are significantly slower (4-5.5ms)
   - **Trigram tokenizer** enables substring-like behavior with moderate overhead (~2.1ms)

3. **Index Usage**:
   - **LIKE prefix** (`'abc%'`) can use B-tree indexes effectively (~500μs)
   - **LIKE substring** (`'%abc%'`) cannot use indexes - full table scan required (~1ms)
   - **INSTR()** never uses indexes - always full table scan (~950μs)
   - **FTS5** uses its own inverted index structure

4. **Result Fetching**:
   - **LIKE** is ~2x faster than FTS5 for fetching results
   - Result size has minimal impact on per-operation time
   - FTS5 overhead is consistent regardless of result size

## Options for Non-FTS Substring Search

### Option 1: LIKE with wildcards (Recommended for non-FTS)

```sql
SELECT * FROM media WHERE path LIKE '%substring%'
```

**Pros:**
- Simple and straightforward
- Works in all SQLite builds
- No additional setup required
- Case-insensitive by default for ASCII

**Cons:**
- Cannot use indexes (full table scan)
- Slower on large datasets (>100k rows)
- No relevance ranking

**Performance:** ~1035μs for 10k rows

### Option 2: INSTR function

```sql
SELECT * FROM media WHERE INSTR(path, 'substring') > 0
```

**Pros:**
- Clear intent (substring search)
- Similar performance to LIKE
- Works in all SQLite builds

**Cons:**
- Cannot use indexes
- Case-sensitive (requires LOWER() for case-insensitive)
- Slightly less flexible than LIKE

**Performance:** ~950μs for 10k rows

### Option 3: LIKE with index (prefix-only)

```sql
CREATE INDEX idx_path ON media(path);
SELECT * FROM media WHERE path LIKE '/media%'
```

**Pros:**
- Can use B-tree index
- Very fast for prefix searches
- Works in all SQLite builds

**Cons:**
- Only works for prefix searches, not general substrings
- Index maintenance overhead

**Performance:** ~500μs for 10k rows (with index)

### Option 4: GLOB pattern matching

```sql
SELECT * FROM media WHERE path GLOB '*substring*'
```

**Pros:**
- More powerful pattern matching than LIKE
- Case-sensitive by default

**Cons:**
- Cannot use indexes
- Slower than LIKE
- Case-sensitivity may be undesirable

**Performance:** ~900μs for 10k rows (estimated)

### Option 5: EndsWith pattern (surprisingly fast)

```sql
SELECT * FROM media WHERE path LIKE '%.mkv'
```

**Pros:**
- Good for extension filtering
- Better performance than general substring
- Works in all SQLite builds

**Cons:**
- Only works for suffix searches
- Cannot use indexes

**Performance:** ~660μs for 10k rows

### Option 6: Hybrid approach (Recommended)

```sql
-- For prefix searches, use indexed LIKE
SELECT * FROM media WHERE path LIKE ?  -- '/media%'

-- For substring searches, use LIKE or INSTR
SELECT * FROM media WHERE path LIKE ?  -- '%media%'

-- Combine with other filters to reduce scan size
SELECT * FROM media 
WHERE type = 'video' 
  AND path LIKE '%substring%'
```

**Pros:**
- Best of both worlds
- Optimizes common cases
- Flexible

**Cons:**
- More complex query logic
- Requires query planning

## Recommendations

### For Non-FTS Builds:

1. **Use LIKE for simplicity**: `WHERE path LIKE '%substring%'`
   - Good enough performance for datasets up to ~100k rows
   - ~1ms per search on 10k rows
   - No additional setup required

2. **Add indexes for prefix searches**: `CREATE INDEX idx_path ON media(path)`
   - Use when prefix searches are common
   - ~500μs vs ~1ms for substring searches
   - Combine with other selective filters

3. **Filter first, search second**:
   ```sql
   SELECT * FROM media 
   WHERE type = 'video'      -- Selective filter first (indexed)
     AND path LIKE '%test%'  -- Expensive substring search second
   ```
   - Can reduce effective search space significantly

4. **Consider INSTR for clarity**: `WHERE INSTR(path, 'substring') > 0`
   - Similar performance to LIKE (~950μs vs ~1ms)
   - More explicit intent for substring search
   - Remember: case-sensitive by default

5. **Use EndsWith when applicable**: `WHERE path LIKE '%.mkv'`
   - Surprisingly good performance (~660μs)
   - Useful for extension filtering

### For FTS5 Builds:

1. **Use JOIN pattern for better performance**:
   ```sql
   SELECT m.* FROM media m
   INNER JOIN media_fts f ON m.rowid = f.rowid
   WHERE f.fts_path MATCH ?
   ```
   - ~677μs vs ~707μs for subquery pattern

2. **Use trigram tokenizer for substring-like behavior**:
   ```sql
   CREATE VIRTUAL TABLE media_fts USING fts5(
       path, title,
       tokenize = 'trigram'
   );
   -- Query: MATCH 'sub' finds 'substring' (~2.1ms)
   ```

3. **Avoid complex FTS5 queries** when simple ones suffice:
   - Simple MATCH: ~4.2ms
   - Phrase/NEAR/AND/OR: 4-5.5ms
   - Use simpler queries when possible

4. **Consider FTS5 NEAR for proximity searches**:
   - Fastest FTS5 operation (~108μs)
   - Useful for finding related terms

## Database Optimization Tips

### PRAGMA settings for better performance:

```sql
PRAGMA journal_mode = WAL;        -- Better concurrency
PRAGMA synchronous = NORMAL;      -- Good balance of safety/speed
PRAGMA cache_size = -64000;       -- 64MB cache
PRAGMA temp_store = MEMORY;       -- In-memory temp tables
PRAGMA mmap_size = 268435456;     -- 256MB memory-mapped I/O
```

### Index strategies:

```sql
-- For prefix searches
CREATE INDEX idx_path ON media(path);

-- For type-filtered searches
CREATE INDEX idx_type ON media(type);

-- Composite index for common filter combinations
CREATE INDEX idx_type_path ON media(type, path);
```

### Maintenance:

```sql
-- Regular optimization
VACUUM;
ANALYZE;
```

## Conclusion

For **non-FTS builds**, the best substring search option is:

1. **LIKE with wildcards** (`%pattern%`) for general substring searches
2. **LIKE with prefix** (`pattern%`) + index for prefix searches
3. **Hybrid approach** combining selective filters with substring search

Performance is acceptable for datasets up to ~100k rows. For larger datasets or more complex search requirements, consider:
- Enabling FTS5 with trigram tokenizer
- Application-level search indexing (e.g., Bleve, Meilisearch)
- Database-level full-text search extensions

## See Also

- **[External Search Engine Comparison](EXTERNAL_SEARCH_COMPARISON.md)** - Detailed comparison of Bleve, gosearch, Meilisearch, ZincSearch, and Elasticsearch as alternatives to SQLite-based search.
