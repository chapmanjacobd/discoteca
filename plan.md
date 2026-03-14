# SQLite to Bleve Migration Plan

## Overview

This document outlines the migration strategy from SQLite FTS5 to Bleve full-text search engine for the discoteca media library project. The migration aims to improve search performance, enable advanced faceting/aggregation capabilities, and provide better scalability.

## Background

### Current State

- **Storage**: SQLite with FTS5 extension for full-text search
- **Search Fields**: `path`, `fts_path`, `title`, `description`
- **Aggregations**: SQL-based GROUP BY for `--big-dirs`, `--group-by-*` features
- **Sorting**: SQL ORDER BY with multiple fields
- **Pagination**: OFFSET/LIMIT based

### Target State

- **Storage**: SQLite for structured data + Bleve for full-text search
- **Search Fields**: All current fields plus enhanced tokenization
- **Aggregations**: Bleve facets for term, numeric range, and date range
- **Sorting**: Bleve sort with docValues optimization
- **Pagination**: Both offset and search_after (cursor-based)

---

## Phase 1: Index Mapping Design

### 1.1 Field Mapping Strategy

Based on docValues analysis, we need to carefully decide which fields should enable docValues:

| Field | Type | docValues | Rationale |
|-------|------|-----------|-----------|
| `path` | text (keyword) | ✅ | Sorting, faceting by directory |
| `path_tokenized` | text (standard) | ❌ | Full-text search only |
| `title` | text (edge_ngram) | ✅ | Sorting, autocomplete |
| `description` | text (standard) | ❌ | Full-text search only |
| `type` | text (keyword) | ✅ | Faceting (video/audio/image/text) |
| `size` | numeric | ✅ | Range facets, sorting, DU mode |
| `duration` | numeric | ✅ | Range facets, sorting |
| `time_created` | date/numeric | ✅ | Date range facets, sorting |
| `time_modified` | date/numeric | ✅ | Date range facets, sorting |
| `time_downloaded` | date/numeric | ✅ | Date range facets, sorting |
| `time_last_played` | date/numeric | ✅ | Sorting, range queries |
| `play_count` | numeric | ✅ | Sorting, range queries |
| `genre` | text (keyword) | ✅ | Term faceting |
| `artist` | text (keyword) | ✅ | Term faceting |
| `album` | text (keyword) | ✅ | Term faceting |
| `language` | text (keyword) | ✅ | Term faceting |
| `categories` | text (keyword) | ✅ | Term faceting |

### 1.2 Updated Index Mapping

```go
// internal/bleve/bleve.go - Updated createIndex function

func createIndex(path string) (bleve.Index, error) {
    // Path field - keyword analyzer for exact matching, docValues for faceting/sorting
    pathField := bleve.NewTextFieldMapping()
    pathField.Analyzer = keyword.Name
    pathField.DocValues = true
    
    // Path tokenized - standard analyzer for FTS, no docValues needed
    pathTokenizedField := bleve.NewTextFieldMapping()
    pathTokenizedField.Analyzer = standard.Name
    pathTokenizedField.DocValues = false
    
    // Title - edge_ngram for autocomplete, docValues for sorting
    titleField := bleve.NewTextFieldMapping()
    titleField.Analyzer = "title_edge_ngram"
    titleField.IncludeInAll = true
    titleField.DocValues = true
    
    // Description - standard analyzer, no docValues
    descriptionField := bleve.NewTextFieldMapping()
    descriptionField.Analyzer = standard.Name
    descriptionField.IncludeInAll = true
    descriptionField.DocValues = false
    
    // Type - keyword for exact matching, docValues for faceting
    typeField := bleve.NewTextFieldMapping()
    typeField.Analyzer = keyword.Name
    typeField.DocValues = true
    
    // Numeric fields with docValues enabled
    sizeField := bleve.NewNumericFieldMapping()
    sizeField.DocValues = true
    
    durationField := bleve.NewNumericFieldMapping()
    durationField.DocValues = true
    
    // Date/timestamp fields with docValues enabled
    timeCreatedField := bleve.NewNumericFieldMapping()
    timeCreatedField.DocValues = true
    
    timeModifiedField := bleve.NewNumericFieldMapping()
    timeModifiedField.DocValues = true
    
    timeDownloadedField := bleve.NewNumericFieldMapping()
    timeDownloadedField.DocValues = true
    
    timeLastPlayedField := bleve.NewNumericFieldMapping()
    timeLastPlayedField.DocValues = true
    
    playCountField := bleve.NewNumericFieldMapping()
    playCountField.DocValues = true
    
    // Text faceting fields with docValues
    genreField := bleve.NewTextFieldMapping()
    genreField.Analyzer = keyword.Name
    genreField.DocValues = true
    
    artistField := bleve.NewTextFieldMapping()
    artistField.Analyzer = keyword.Name
    artistField.DocValues = true
    
    languageField := bleve.NewTextFieldMapping()
    languageField.Analyzer = keyword.Name
    languageField.DocValues = true
    
    // Create document mapping
    docMapping := bleve.NewDocumentMapping()
    docMapping.AddFieldMappingsAt("path", pathField)
    docMapping.AddFieldMappingsAt("path_tokenized", pathTokenizedField)
    docMapping.AddFieldMappingsAt("title", titleField)
    docMapping.AddFieldMappingsAt("description", descriptionField)
    docMapping.AddFieldMappingsAt("type", typeField)
    docMapping.AddFieldMappingsAt("size", sizeField)
    docMapping.AddFieldMappingsAt("duration", durationField)
    docMapping.AddFieldMappingsAt("time_created", timeCreatedField)
    docMapping.AddFieldMappingsAt("time_modified", timeModifiedField)
    docMapping.AddFieldMappingsAt("time_downloaded", timeDownloadedField)
    docMapping.AddFieldMappingsAt("time_last_played", timeLastPlayedField)
    docMapping.AddFieldMappingsAt("play_count", playCountField)
    docMapping.AddFieldMappingsAt("genre", genreField)
    docMapping.AddFieldMappingsAt("artist", artistField)
    docMapping.AddFieldMappingsAt("language", languageField)
    
    // Create index mapping
    indexMapping := bleve.NewIndexMapping()
    indexMapping.DefaultMapping = docMapping
    indexMapping.DefaultAnalyzer = standard.Name
    indexMapping.ScoringModel = "bm25"
    
    // ... (rest of analyzer registration)
    
    return bleve.New(path, indexMapping)
}
```

### 1.3 Updated MediaDocument Structure

```go
type MediaDocument struct {
    ID              string  `json:"id"`
    Path            string  `json:"path"`
    PathTokenized   string  `json:"path_tokenized"`
    Title           string  `json:"title"`
    Description     string  `json:"description"`
    Type            string  `json:"type"`
    Size            int64   `json:"size"`
    Duration        int64   `json:"duration"`
    TimeCreated     int64   `json:"time_created"`
    TimeModified    int64   `json:"time_modified"`
    TimeDownloaded  int64   `json:"time_downloaded"`
    TimeLastPlayed  int64   `json:"time_last_played"`
    PlayCount       int64   `json:"play_count"`
    Genre           string  `json:"genre"`
    Artist          string  `json:"artist"`
    Album           string  `json:"album"`
    Language        string  `json:"language"`
    Categories      string  `json:"categories"`
    VideoCount      int64   `json:"video_count"`
    AudioCount      int64   `json:"audio_count"`
    SubtitleCount   int64   `json:"subtitle_count"`
    Width           int64   `json:"width"`
    Height          int64   `json:"height"`
    Score           float64 `json:"score"`
}
```

---

## Phase 2: Faceted Search Implementation

### 2.1 Facet Types

Bleve supports three facet types that map to our use cases:

1. **Term Facet**: Genre, artist, type, language, categories
2. **Numeric Range Facet**: Size, duration, play_count
3. **Date Range Facet**: time_created, time_modified, time_downloaded, time_last_played

### 2.2 Facet Query Examples

#### Term Facet (by type)

```go
func SearchWithFacets(queryStr string, limit int) (*bleve.SearchResult, error) {
    bleveQuery := bleve.NewMatchQuery(queryStr)
    
    searchRequest := bleve.NewSearchRequest(bleveQuery)
    searchRequest.Size = limit
    
    // Add term facet for media type
    searchRequest.AddFacet("media_type", bleve.NewFacetRequest("type", 10))
    
    // Add term facet for genre
    searchRequest.AddFacet("genre", bleve.NewFacetRequest("genre", 10))
    
    return GetIndex().Search(searchRequest)
}
```

#### Numeric Range Facet (by size)

```go
func SearchWithSizeFacet(queryStr string, limit int) (*bleve.SearchResult, error) {
    bleveQuery := bleve.NewMatchQuery(queryStr)
    
    searchRequest := bleve.NewSearchRequest(bleveQuery)
    searchRequest.Size = limit
    
    // Size ranges: 0-100MB, 100MB-1GB, 1GB-10GB, 10GB+
    sizeFacet := bleve.NewFacetRequest("size", 4)
    sizeFacet.AddRange(bleve.NewNumericRangeFacet("0-100MB", 0, 100*1024*1024))
    sizeFacet.AddRange(bleve.NewNumericRangeFacet("100MB-1GB", 100*1024*1024, 1024*1024*1024))
    sizeFacet.AddRange(bleve.NewNumericRangeFacet("1GB-10GB", 1024*1024*1024, 10*1024*1024*1024))
    sizeFacet.AddRange(bleve.NewNumericRangeFacet("10GB+", 10*1024*1024*1024, math.MaxInt64))
    
    searchRequest.AddFacet("size_ranges", sizeFacet)
    
    return GetIndex().Search(searchRequest)
}
```

#### Date Range Facet (by time_created)

```go
func SearchWithDateFacet(queryStr string, limit int) (*bleve.SearchResult, error) {
    bleveQuery := bleve.NewMatchQuery(queryStr)
    
    searchRequest := bleve.NewSearchRequest(bleveQuery)
    searchRequest.Size = limit
    
    // Date ranges: Last 7 days, Last 30 days, Last year, Older
    now := time.Now().Unix()
    dateFacet := bleve.NewFacetRequest("time_created", 4)
    dateFacet.AddRange(bleve.NewNumericRangeFacet("Last 7 days", now-7*24*3600, now))
    dateFacet.AddRange(bleve.NewNumericRangeFacet("Last 30 days", now-30*24*3600, now))
    dateFacet.AddRange(bleve.NewNumericRangeFacet("Last year", now-365*24*3600, now))
    dateFacet.AddRange(bleve.NewNumericRangeFacet("Older", 0, now-365*24*3600))
    
    searchRequest.AddFacet("date_ranges", dateFacet)
    
    return GetIndex().Search(searchRequest)
}
```

### 2.3 Disk Usage Aggregation (DU Mode)

DU mode requires aggregating by directory and calculating size/count ratios. This can be achieved using path-based term faceting:

```go
func DiskUsageByDirectory(prefix string, limit int) (*bleve.SearchResult, error) {
    // Match all documents under prefix
    query := bleve.NewPrefixQuery(prefix)
    query.SetField("path")
    
    searchRequest := bleve.NewSearchRequest(query)
    searchRequest.Size = 0 // No hits needed, just facets
    
    // Extract directory from path using term facet
    // This requires path to be indexed with directory-level tokens
    dirFacet := bleve.NewFacetRequest("path", limit)
    
    // For true DU mode, we need custom aggregation
    // Consider using search + client-side aggregation for complex calculations
    
    return GetIndex().Search(searchRequest)
}
```

**Note**: Complex aggregations like `size/count` ratio may require:
1. Client-side aggregation after fetching results
2. Custom Bleve aggregator extension
3. Hybrid approach: Bleve for filtering + SQLite for aggregation

---

## Phase 3: Sorting Implementation

### 3.1 Default Sort Order (Web UI)

Based on xklb sorting logic:

```go
func DefaultSortOrder() []string {
    return []string{
        "-video_count",  // videos before audio-only (desc)
        "-audio_count",  // files with audio before silent (desc)
        "path",          // local files before remote URLs (asc)
        "-subtitle_count", // subtitled content first (desc)
        "play_count",    // unplayed/least-played first (asc)
        "-playhead",     // furthest progress first (desc)
        "time_last_played", // least-recently played first (asc)
        "title",         // titled entries first (asc)
        "path",          // alphabetical tiebreak (asc)
    }
}
```

### 3.2 DU Mode Sort Order

```go
func DUSortOrder() []string {
    // For directory aggregation, sort by:
    // (size/count) desc, size desc, count desc, path desc
    return []string{
        "-size_per_count", // Custom field or client-side calculation
        "-size",
        "-count",
        "-path",
    }
}
```

### 3.3 Search Request with Sort

```go
func SearchWithSort(queryStr string, limit, offset int, sortFields []string) (*bleve.SearchResult, error) {
    bleveQuery := bleve.NewMatchQuery(queryStr)
    
    searchRequest := bleve.NewSearchRequest(bleveQuery)
    searchRequest.Size = limit
    searchRequest.From = offset
    
    // Parse sort fields into Bleve sort order
    searchRequest.Sort = search.ParseSortOrderStrings(sortFields)
    
    return GetIndex().Search(searchRequest)
}
```

---

## Phase 4: Pagination Implementation

user note: Let's just implement Offset-Based Pagination for now--not searchBefore/searchAfter


### 4.1 Offset-Based Pagination (Simple)

```go
func SearchPaginated(queryStr string, page, pageSize int) (*SearchResponse, error) {
    offset := (page - 1) * pageSize
    
    bleveQuery := bleve.NewMatchQuery(queryStr)
    searchRequest := bleve.NewSearchRequest(bleveQuery)
    searchRequest.Size = pageSize
    searchRequest.From = offset
    searchRequest.Sort = search.ParseSortOrderStrings([]string{"-_score", "id"})
    
    results, err := GetIndex().Search(searchRequest)
    if err != nil {
        return nil, err
    }
    
    return &SearchResponse{
        Hits:   extractHits(results),
        Total:  results.Total,
        Page:   page,
        Pages:  (results.Total + int64(pageSize) - 1) / int64(pageSize),
    }, nil
}
```

---

## Phase 5: Batch Indexing

### 5.1 Batch Import for Bulk Operations

```go
// internal/bleve/bleve.go

func BatchIndexDocuments(docs []*MediaDocument, batchSize int) error {
    indexMutex.RLock()
    defer indexMutex.RUnlock()
    
    if indexInstance == nil {
        return fmt.Errorf("bleve index not initialized")
    }
    
    for i := 0; i < len(docs); i += batchSize {
        batch := indexInstance.NewBatch()
        
        end := i + batchSize
        if end > len(docs) {
            end = len(docs)
        }
        
        for j := i; j < end; j++ {
            err := batch.Index(docs[j].ID, docs[j])
            if err != nil {
                return fmt.Errorf("failed to index document %s: %w", docs[j].ID, err)
            }
        }
        
        err := indexInstance.Batch(batch)
        if err != nil {
            return fmt.Errorf("failed to execute batch: %w", err)
        }
        
        // Optional: Force merge after large batches
        if i%10000 == 0 && i > 0 {
            // Consider periodic force merge for segment optimization
        }
    }
    
    return nil
}
```

### 5.2 Integration with Add Command

```go
// internal/commands/add.go

func AddMedia(dbPath string, paths []string, options AddOptions) error {
    // ... existing setup ...
    
    var docs []*bleve.MediaDocument
    batchSize := 1000
    
    for _, path := range paths {
        // ... extract metadata ...
        
        doc := bleve.ToBleveDocFromUpsert(params)
        docs = append(docs, doc)
        
        if len(docs) >= batchSize {
            if err := bleve.BatchIndexDocuments(docs, batchSize); err != nil {
                log.Printf("Warning: batch indexing error: %v", err)
            }
            docs = docs[:0]
        }
    }
    
    // Final batch
    if len(docs) > 0 {
        if err := bleve.BatchIndexDocuments(docs, batchSize); err != nil {
            return err
        }
    }
    
    // Optional: Force merge after bulk import
    if options.ForceMerge {
        return bleve.ForceMerge()
    }
    
    return nil
}
```

---

## Phase 6: Feature Parity Checklist

### 6.1 Features Requiring Bleve

- [x] Full-text search (path, title, description)
- [ ] Term faceting (type, genre, artist, language)
- [ ] Numeric range faceting (size, duration)
- [ ] Date range faceting (time_created, time_modified, time_downloaded)
- [ ] Sorting by all docValues-enabled fields

### 6.2 Features Requiring Hybrid Approach

These features may need SQLite + Bleve combination to work well (investigation and experimentation is needed):

- [ ] `--big-dirs` (directory aggregation with size and count)
- [ ] `--group-by-extensions`
- [ ] `--group-by-mime-types`
- [ ] `--group-by-size` (bucketed aggregation)
- [ ] `--frequency` (time-based grouping)
- [ ] Complex re-ranking with multiple weights
- [ ] Sibling fetching (requires relational queries)

**Hybrid Strategy**:
1. Use Bleve for filtering and sorting and initial result set
2. Perform complex aggregations and sorting in Go (if not possible with Bleve)
3. Compare speed, memory usage, and results with the existing SQLite implementation

### 6.3 Features SQLite-Only

- [ ] Playback history tracking
- [ ] Watched/unwatched filtering (requires history JOIN)
- [ ] Custom keyword categories
- [ ] Playlist management

---

## Phase 7: Web UI Integration

### 7.2 Facet Sidebar

Display facet counts in filters sidebar:
- Media Type (video/audio/image/text)
- Genre
- Artist
- Language
- Size ranges
- Date ranges

### 7.3 Search Mode Enhancements

- Implement xklb default sort order
- Add facet-based filtering

### 7.4 DU Mode Enhancements

- Directory-based aggregation view
- Sort by size/count ratio
- Visual size representation

---

## Phase 8: Testing & Benchmarking

### 8.1 Test Cases

1. **Index Mapping Tests**
   - Verify docValues enabled for correct fields
   - Verify analyzers working correctly
   - Test edge_ngram for autocomplete

2. **Faceting Tests**
   - Term facet accuracy
   - Numeric range facet boundaries
   - Date range facet calculations

3. **Sorting Tests**
   - Default sort order matches xklb
   - DU mode sort order
   - Multi-field sort with tie-breaking

4. **Pagination Tests**
   - Offset pagination correctness
   - SearchAfter cursor pagination
   - Deep pagination performance

5. **Batch Indexing Tests**
   - Large batch handling
   - Error recovery
   - Index consistency

### 8.2 Performance Benchmarks

Compare FTS5 vs Bleve:

1. **Search Latency**
   - Simple term search
   - Multi-field search
   - Filtered search with facets

2. **Index Size**
   - FTS5-only index
   - Bleve index with docValues
   - Bleve index without docValues

3. **Memory Usage**
   - Query memory footprint
   - Faceting memory overhead
   - Sorting memory overhead

4. **Indexing Speed**
   - Single document indexing
   - Batch indexing throughput
   - Reindex full database

### 8.3 Benchmark Test Structure

```go
// internal/bleve/benchmark_test.go

func BenchmarkFTS5Search(b *testing.B) {
    // SQLite FTS5 search benchmark
}

func BenchmarkBleveSearch(b *testing.B) {
    // Bleve search benchmark
}

func BenchmarkBleveSearchWithFacets(b *testing.B) {
    // Bleve search with faceting benchmark
}

func BenchmarkBleveSearchWithSort(b *testing.B) {
    // Bleve search with sorting benchmark
}

func BenchmarkBleveBatchIndex(b *testing.B) {
    // Batch indexing benchmark
}
```

---

## Phase 9: Migration Strategy

### 9.1 Performance driven Migration

1. **Phase 1**: Run Bleve alongside FTS5 (dual indexing)
2. **Phase 2**: Migrate read queries to Bleve
3. **Phase 3**: Compare scaling performance across many metrics (5000 rows, 500000 rows)
4. **Phase 4**: Remove FTS5 dependency (optional)

Data and schema migration is not necessary but it would be helpful to have a --bleve feature flag

---

## Phase 10: Documentation Updates

### 10.1 User Documentation

- Update README.go with Bleve capabilities
- Add troubleshooting guide

### 10.2 Developer Documentation

- Index mapping documentation
- docValues field selection guide
- Faceting API reference
- Performance tuning guide

---

## Appendix A: Disk Usage Trade-off Analysis

Based on empirical analysis from the background document:

| Metric | Without docValues | With docValues | Delta |
|--------|------------------|----------------|-------|
| Index Size | Baseline | +7,762 bytes per 1000 docs | +7.7 KB |
| Query Time (1000 queries) | Baseline | -27ms | -27ms |
| Memory Usage | Higher (doc fetch) | Lower (direct access) | Variable |

**Recommendation**: Enable docValues for all fields used in:
- Sorting (SortField, SortGeoDistance)
- Faceting (all facet types)

**Disk space increase is marginal compared to performance gains.**

---

## Appendix B: Implementation Priority

### High Priority (Core Functionality)

1. ✅ Updated index mapping with docValues
2. ✅ Basic faceting (term, numeric, date)
3. ✅ Sorting implementation
4. ✅ Batch indexing
5. ✅ Pagination (both modes)

### Medium Priority (Feature Enhancements)

1. DU mode implementation
2. Hybrid aggregation approach
3. Performance benchmarking

### Low Priority (Optimization)

1. Force merge strategies
2. Advanced caching
3. Custom aggregators

---

## Appendix C: Known Limitations

1. **Complex Aggregations**: Bleve facets don't support custom calculations (e.g., size/count ratio). May require hybrid approach (Bleve + Go).

2. **Categorization Mode**: Web UI categorization may not benefit from Bleve. Keep SQLite-based.

---

## Next Steps

1. **Implement Phase 1**: Update index mapping with docValues
2. **Implement Phase 2**: Add faceting support
3. **Implement Phase 5**: Batch indexing for bulk operations
4. **Test**: Run benchmarks comparing FTS5 vs Bleve
5. **Iterate**: Refine based on performance results
