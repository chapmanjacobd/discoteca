# Hybrid FTS5 Search with Phrase Support

## Overview

This implementation adds **phrase search support** to FTS5 while using `detail=none` for maximum index size reduction (~82% smaller).

## Problem

- FTS5 with `detail=none` doesn't support phrase queries (`"exact phrase"`) or tokens > 3 characters
- Your 10GB FTS index could be reduced to ~1.8GB
- But phrase searches are useful for exact matching

## Solution: Hybrid FTS + LIKE Search

Split search queries into two parts:
1. **FTS terms**: Individual words searched via FTS5 using 3-char trigrams (works with `detail=none`)
2. **Phrases**: Exact phrases searched via LIKE (optimized by trigram index)

### Example

```sql
-- User query: python "video tutorial"

-- FTS part (trigram-filtered, detail=none compatible):
-- "python" -> "pyt", "tutorial" -> "tut"
WHERE media_fts MATCH 'pyt OR tut'

-- Phrase part (trigram-optimized LIKE):
AND (path LIKE '%video tutorial%' 
  OR title LIKE '%video tutorial%' 
  OR description LIKE '%video tutorial%')
```

## Key Implementation Details

### Trigram Tokenizer + detail=none

With `tokenize='trigram'` and `detail=none`:
- Text is split into 3-character tokens automatically
- FTS MATCH queries can only use tokens ≤ 3 characters
- **Solution**: Convert search terms to first 3 chars: "video" → "vid"

### Query Building

```go
// BuildFTSQuery converts terms to trigram-compatible format
func (h *HybridSearchQuery) BuildFTSQuery(joinOp string) string {
    for _, term := range h.FTSTerms {
        if len(term) >= 3 {
            trigrams = append(trigrams, term[:3])  // "video" → "vid"
        }
    }
    return strings.Join(trigrams, " OR ")  // Loose filtering
}
```

### Phrase Search via LIKE

Phrases are searched using LIKE patterns which ARE optimized by the trigram index:
```sql
WHERE path LIKE '%exact phrase%'
```

The trigram index filters candidates efficiently before LIKE verification.

## Index Size Comparison

| Configuration | Index Size | Phrase Support | Term Length Limit |
|---------------|------------|----------------|-------------------|
| `detail=full` (old) | ~10 GB | ✅ Native FTS | Unlimited |
| `detail=none` (new) | ~1.8 GB | ✅ Hybrid (LIKE) | ≤3 chars in FTS |

## Performance Characteristics

### FTS Terms (Trigram Filter)
- **Speed**: Very fast - loose trigram filtering
- **Accuracy**: Approximate - finds documents containing the trigrams
- **Ranking**: FTS5 BM25 with trigram provides **limited differentiation**
  - **Solution**: In-memory Go ranking with field-weighted scoring
  - Title matches: 10 points per occurrence
  - Path matches: 5 points per occurrence
  - Description matches: 1 point per occurrence
  - Exact title match bonus: +5 points
- **Use case**: Fast candidate filtering, with meaningful relevance ranking in Go

### Phrase Searches (LIKE)
- **Speed**: Fast - trigram index filters before LIKE verification
- **Accuracy**: Exact - LIKE verifies full string match
- **Use case**: Exact phrase matching

## Limitations

1. **FTS term length**: Terms > 3 chars are truncated to first 3 chars for FTS filtering
2. **No NEAR queries**: `NEAR(term1 term2, 5)` not supported (wasn't being used)
3. **No column filters**: `title:video` syntax removed (made colons tedious)
4. **Two-stage search**: FTS provides candidates, LIKE verifies phrases

## Usage Examples

```bash
# Simple term search (FTS trigram filter)
disco ls "python tutorial"
# FTS: MATCH 'pyt OR tut'

# Phrase search (LIKE)
disco ls '"video tutorial"'
# LIKE: path LIKE '%video tutorial%'

# Mixed search (FTS + LIKE)
disco ls 'python "video tutorial" beginner'
# FTS: MATCH 'pyt OR tut OR beg'
# LIKE: path LIKE '%video tutorial%'
```

## Related Media Expansion

The `_related_media` sort marker now uses the hybrid search to find related content:

```go
// Sort config with related media expansion
flags.PlayInOrder = "play_count desc,_related_media,title asc"

// Expands results with media sharing search terms
// Uses trigram FTS for candidate filtering
```

## Files Changed

- `internal/utils/fts_hybrid.go` - Hybrid query parsing and building
- `internal/query/filter_builder.go` - Integration with filter builder + related media expansion
- `internal/db/fts_queries.go` - Manual FTS queries with **in-memory Go ranking**
  - `RankSearchResults()` - Field-weighted ranking for media search
  - `RankCaptionsResults()` - Field-weighted ranking for caption search
- `internal/db/schema.sql` - FTS table definitions with `detail=none`
- `internal/db/migrate.go` - Migration to upgrade existing FTS tables
- `internal/commands/serve_handlers.go` - Apply in-memory ranking to caption results
- `internal/commands/serve_metadata.go` - Apply in-memory ranking to caption results
- `queries.sql` - Commented out FTS queries (now implemented manually)

## Testing

```bash
# Test hybrid search utility
go test ./internal/utils -run TestHybrid -v

# Test related media expansion
go test -tags fts5 ./internal/query -run TestExpandRelatedMedia -v

# Test FTS queries
go test -tags fts5 ./internal/db -run TestQueries/FTSAndCaptions -v

# Full test suite
go test -tags fts5 ./internal/... -short
```
