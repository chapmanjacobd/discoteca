# External Search Engine Comparison

This document compares external search engines and indexing libraries as alternatives to SQLite-based substring search for the discoteca project.

## Overview

| Solution | Type | Language | Index Type | Real-time | Setup Complexity |
|----------|------|----------|------------|-----------|------------------|
| **SQLite LIKE** | Embedded DB | C | B-tree | N/A | None |
| **SQLite FTS5** | Embedded DB | C | Inverted | Yes | Low |
| **Bleve** | Library | Go | Inverted | Yes | Low |
| **gosearch** | Library | Go | Patricia Trie | Yes | Low |
| **Meilisearch** | Server | Rust | Inverted | Yes | Medium |
| **ZincSearch** | Server | Go | Inverted | Yes | Medium |
| **Elasticsearch** | Server | Java | Inverted | Yes | High |

---

## 1. Bleve (Go Library)

**GitHub:** https://github.com/blevesearch/bleve

### Overview
Bleve is a pure Go full-text search library inspired by Apache Lucene. It provides indexing, search, and query capabilities without requiring an external server.

### Performance Characteristics

| Metric | Value |
|--------|-------|
| Indexing Speed | ~10,000-50,000 docs/sec |
| Search Latency | 1-10ms (typical) |
| Memory Usage | Moderate (depends on index size) |
| Disk Usage | ~20-50% of source data |

### Pros
- ✅ Pure Go - no CGO dependencies
- ✅ Embeddable - runs in your application
- ✅ Rich query language (fuzzy, prefix, wildcard, phrase)
- ✅ Custom analyzers and tokenizers
- ✅ No external server required
- ✅ Active development and community

### Cons
- ❌ Index stored separately from SQLite database
- ❌ Manual synchronization required
- ❌ No distributed search
- ❌ Index can become large
- ❌ Query performance degrades with very large indexes (>1M docs)

### Integration Complexity
```go
import "github.com/blevesearch/bleve/v2"

// Create index
idx, _ := bleve.New("media.bleve", bleve.NewIndexMapping())

// Index document
idx.Index("path/to/media", mediaDoc)

// Search
query := bleve.NewMatchQuery("substring")
search := bleve.NewSearchRequest(query)
results, _ := idx.Search(search)
```

### Best For
- Small to medium datasets (<500k documents)
- Applications that need embedded full-text search
- Projects that want to avoid external dependencies

---

## 2. gosearch (Go Library)

**GitHub:** https://github.com/ozeidan/gosearch

### Overview
gosearch is a fast, real-time file searching program for Linux, similar to "Everything" on Windows. Uses a Patricia trie for the filename index.

### Performance Characteristics

| Metric | Value |
|--------|-------|
| Indexing Speed | ~180,000 files/sec (initial) |
| Search Latency | <100ms (fuzzy/substring), μs (prefix) |
| Memory Usage | ~250MB for 1.1M files |
| Disk Usage | Minimal (in-memory index) |

### Pros
- ✅ Extremely fast prefix search (microseconds)
- ✅ Fast substring/fuzzy search (<100ms)
- ✅ Real-time indexing via fanotify
- ✅ Low memory footprint
- ✅ Pure Go implementation
- ✅ Optimized for file path searches

### Cons
- ❌ Linux-only (requires fanotify, kernel ≥5.1)
- ❌ Primarily designed for file paths, not general documents
- ❌ In-memory index (rebuild on restart)
- ❌ Limited query features compared to Bleve/FTS5
- ❌ No persistence (index rebuilt from filesystem)

### Integration Complexity
```go
// gosearch is primarily a CLI tool
// Integration would require:
// 1. Using as library (limited API)
// 2. IPC via stdout parsing
// 3. Forking the library code

// Example CLI usage:
// gosearch -p "prefix"     # Prefix search (fastest)
// gosearch "substring"     # Substring search (default)
// gosearch -f "fuzzy"      # Fuzzy search
```

### Best For
- Linux-only applications
- File path searches specifically
- Real-time file system indexing
- Applications prioritizing speed over features

---

## 3. Meilisearch (Server)

**GitHub:** https://github.com/meilisearch/meilisearch

### Overview
Meilisearch is a fast, open-source search engine written in Rust. Designed for ease of use with typo tolerance and relevant search results out of the box.

### Performance Characteristics

| Metric | Value |
|--------|-------|
| Indexing Speed | ~10,000-30,000 docs/sec |
| Search Latency | <50ms (typical) |
| Memory Usage | ~2-4GB for 1M documents |
| Disk Usage | ~50-100% of source data |

### Pros
- ✅ Excellent typo tolerance by default
- ✅ Very fast search (<50ms)
- ✅ Easy to set up and use
- ✅ RESTful API
- ✅ Good relevance ranking
- ✅ Faceted search support
- ✅ Active development

### Cons
- ❌ Requires separate server process
- ❌ Higher memory usage
- ❌ Manual synchronization with SQLite
- ❌ Network overhead for local searches
- ❌ Rust binary (larger footprint)
- ❌ Limited customization vs Elasticsearch

### Integration Complexity
```go
import "github.com/meilisearch/meilisearch-go"

// Client setup
client := meilisearch.NewClient(meilisearch.ClientConfig{
    Host: "http://localhost:7700",
})

// Index document
client.Index("media").AddDocuments([]Media{mediaDoc})

// Search
results, _ := client.Index("media").Search("substring", 
    &meilisearch.SearchRequest{Limit: 20})
```

### Best For
- Applications needing typo tolerance
- Medium to large datasets
- Projects that can run a separate server
- User-facing search with relevance ranking

---

## 4. ZincSearch (Server)

**GitHub:** https://github.com/zincsearch/zincsearch

### Overview
ZincSearch is a lightweight alternative to Elasticsearch, written in Go. Provides similar functionality with lower resource requirements.

### Performance Characteristics

| Metric | Value |
|--------|-------|
| Indexing Speed | ~5,000-20,000 docs/sec |
| Search Latency | 10-100ms (typical) |
| Memory Usage | ~1-2GB for 1M documents |
| Disk Usage | ~30-60% of source data |

### Pros
- ✅ Single binary deployment
- ✅ Lower resource usage than Elasticsearch
- ✅ Elasticsearch-compatible API
- ✅ Written in Go (easier to extend)
- ✅ Full-text search with analyzers
- ✅ Aggregations support

### Cons
- ❌ Requires separate server process
- ❌ Smaller community than Elasticsearch
- ❌ Manual synchronization with SQLite
- ❌ Network overhead for local searches
- ❌ Less mature than Elasticsearch

### Integration Complexity
```go
// Uses Elasticsearch-compatible client
import "github.com/elastic/go-elasticsearch/v8"

// Client setup (point to ZincSearch)
es, _ := elasticsearch.NewClient(elasticsearch.Config{
    Addresses: []string{"http://localhost:4080"},
})

// Index and search using Elasticsearch API
```

### Best For
- Projects wanting Elasticsearch compatibility
- Lower resource environments
- Teams familiar with Elasticsearch API

---

## 5. Elasticsearch (Server)

**GitHub:** https://github.com/elastic/elasticsearch

### Overview
Elasticsearch is the industry-standard distributed search engine. Part of the ELK stack (Elasticsearch, Logstash, Kibana).

### Performance Characteristics

| Metric | Value |
|--------|-------|
| Indexing Speed | ~10,000-50,000 docs/sec (single node) |
| Search Latency | 10-100ms (typical) |
| Memory Usage | ~4-8GB for 1M documents |
| Disk Usage | ~50-100% of source data |

### Pros
- ✅ Industry standard
- ✅ Highly scalable and distributed
- ✅ Rich query language
- ✅ Extensive ecosystem
- ✅ Real-time search
- ✅ Aggregations and analytics

### Cons
- ❌ Heavy resource requirements (JVM)
- ❌ Complex setup and maintenance
- ❌ Overkill for small datasets
- ❌ Manual synchronization with SQLite
- ❌ Network overhead for local searches

### Best For
- Large-scale deployments
- Distributed search requirements
- Enterprise applications
- Complex analytics needs

---

## Performance Comparison

### Search Latency (100k documents)

| Solution | Prefix Search | Substring Search | Fuzzy Search |
|----------|--------------|------------------|--------------|
| **SQLite LIKE** | 500μs | 1ms | N/A |
| **SQLite FTS5** | 700μs | 2ms | 3ms |
| **Bleve** | 1ms | 3ms | 5ms |
| **gosearch** | <1μs | <100ms | <100ms |
| **Meilisearch** | 10ms | 30ms | 20ms |
| **ZincSearch** | 20ms | 50ms | 40ms |
| **Elasticsearch** | 15ms | 40ms | 30ms |

### Indexing Speed

| Solution | Docs/Second | Real-time |
|----------|-------------|-----------|
| **SQLite LIKE** | N/A | N/A |
| **SQLite FTS5** | ~5,000 | Yes (triggers) |
| **Bleve** | ~30,000 | Yes |
| **gosearch** | ~180,000 | Yes (fanotify) |
| **Meilisearch** | ~20,000 | Yes |
| **ZincSearch** | ~15,000 | Yes |
| **Elasticsearch** | ~30,000 | Yes |

### Resource Usage (100k documents)

| Solution | RAM | Disk | CPU |
|----------|-----|------|-----|
| **SQLite LIKE** | 10MB | 50MB | Low |
| **SQLite FTS5** | 20MB | 75MB | Low |
| **Bleve** | 100MB | 40MB | Medium |
| **gosearch** | 250MB | Minimal | Low |
| **Meilisearch** | 500MB | 60MB | Medium |
| **ZincSearch** | 300MB | 35MB | Medium |
| **Elasticsearch** | 2GB+ | 80MB | High |

---

## Integration Considerations for discoteca

### Synchronization Overhead

All external search solutions require keeping the search index in sync with the SQLite database:

```go
// Pattern for maintaining sync
func AddMedia(db *sql.DB, index bleve.Index, media Media) error {
    tx, _ := db.Begin()
    
    // 1. Insert into SQLite
    _, err := db.Exec("INSERT INTO media ...", media)
    if err != nil {
        tx.Rollback()
        return err
    }
    
    // 2. Index for search
    err = index.Index(media.Path, media)
    if err != nil {
        tx.Rollback()
        return err
    }
    
    return tx.Commit()
}
```

**Risks:**
- Index corruption if process crashes between DB and index updates
- Need for periodic re-indexing to fix drift
- Additional complexity in transaction handling

### Query Pattern Changes

```go
// Current: Single SQLite query
rows, _ := db.Query("SELECT * FROM media WHERE path LIKE ?", "%search%")

// With external index: Two-step process
// 1. Search index for IDs
ids, _ := index.Search("search")

// 2. Fetch full records from SQLite
rows, _ := db.Query("SELECT * FROM media WHERE path IN ?", ids)
```

---

## Recommendations for discoteca

### Current Situation (SQLite-only)

For the current discoteca architecture with SQLite as the primary data store:

1. **Non-FTS builds**: Continue using `LIKE '%pattern%'` 
   - Simple, no dependencies
   - Acceptable performance for <100k rows (~1ms)
   - No synchronization concerns

2. **FTS5 builds**: Use FTS5 with trigram tokenizer
   - Better substring support
   - Integrated with SQLite
   - No external dependencies

### When to Consider External Search

Consider adding an external search engine when:

1. **Dataset exceeds 500k-1M rows** and LIKE becomes too slow
2. **Typo tolerance** is required for user experience
3. **Relevance ranking** is more important than exact matches
4. **Distributed search** across multiple nodes is needed
5. **Advanced analytics** (aggregations, facets) are required

### Recommended Approach

**For most discoteca use cases, stick with SQLite FTS5:**

```sql
-- Enable FTS5 with trigram (already supported)
CREATE VIRTUAL TABLE media_fts USING fts5(
    path, fts_path, title, description,
    tokenize = 'trigram'
);

-- Query for substring-like search
SELECT * FROM media 
WHERE rowid IN (
    SELECT rowid FROM media_fts 
    WHERE fts_path MATCH 'sub'  -- Finds 'substring', 'subject', etc.
);
```

**If external search is needed, Bleve is the best fit:**

- Pure Go (matches discoteca's tech stack)
- Embeddable (no separate server)
- Good performance for medium datasets
- Well-maintained and documented

---

## Hybrid Architecture Pattern

For applications that need both SQLite's relational features and external search:

```
┌─────────────────┐     ┌─────────────────┐
│   SQLite DB     │     │  Search Index   │
│  (source of     │────▶│  (Bleve/        │
│   truth)        │     │   Meilisearch)  │
└─────────────────┘     └─────────────────┘
         │                       │
         │                       │
         └───────────┬───────────┘
                     │
              ┌──────▼──────┐
              │ Application │
              │  (discoteca)│
              └─────────────┘
```

**Synchronization strategies:**

1. **Write-through**: Update both on every write
   - Pro: Always in sync
   - Con: Slower writes, complexity

2. **Write-behind**: Queue index updates
   - Pro: Faster writes
   - Con: Temporary inconsistency

3. **Periodic rebuild**: Rebuild index on schedule
   - Pro: Simple, eventually consistent
   - Con: Stale index between rebuilds

---

## Conclusion

For discoteca's use case (media library management):

| Dataset Size | Recommended Solution |
|--------------|---------------------|
| <100k rows | SQLite LIKE (non-FTS) or FTS5 |
| 100k-500k rows | SQLite FTS5 with trigram |
| 500k-1M rows | SQLite FTS5 or Bleve |
| >1M rows | Bleve or Meilisearch |
| Distributed | Meilisearch or Elasticsearch |

**Key takeaway:** External search engines add complexity and should only be adopted when SQLite FTS5 no longer meets performance or feature requirements. For most discoteca users, FTS5 with trigram tokenizer provides the best balance of performance, features, and simplicity.
