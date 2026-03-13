# Hybrid FTS5 Search with Phrase Support

## Overview

This implementation adds **phrase search support** to FTS5 while using `detail=none` for maximum index size reduction (~82% smaller).

## Problem

- FTS5 with `detail=none` doesn't support phrase queries (`"exact phrase"`)
- Your 10GB FTS index could be reduced to ~1.8GB
- But phrase searches are useful for exact matching

## Solution: Hybrid FTS + LIKE Search

Split search queries into two parts:
1. **FTS terms**: Individual words searched via FTS5 (works with `detail=none`)
2. **Phrases**: Exact phrases searched via LIKE (optimized by trigram index)

### Example

```sql
-- User query: python "video tutorial" beginner

-- FTS part (detail=none compatible):
WHERE media_fts MATCH '"python" OR "beginner"'

-- Phrase part (trigram-optimized LIKE):
AND (path LIKE '%video tutorial%' OR title LIKE '%video tutorial%' OR description LIKE '%video tutorial%')
```

## Changes Made

### 1. New Utility: `internal/utils/fts_hybrid.go`

```go
// ParseHybridSearchQuery splits query into FTS terms and phrases
hybrid := utils.ParseHybridSearchQuery(`python "video tutorial"`)

// hybrid.FTSTerms = ["python"]
// hybrid.Phrases = ["video tutorial"]
```

**Features:**
- Extracts quoted phrases (`"..."` or `'...'`)
- Handles boolean operators (OR, AND, NOT)
- Skips phrases < 3 characters (trigram requirement)
- Removes FTS5 operators incompatible with `detail=none`

### 2. Updated Filter Builder: `internal/query/filter_builder.go`

FTS search now uses hybrid approach:
```go
hybrid := utils.ParseHybridSearchQuery(queryStr)

// FTS terms
if hybrid.HasFTSTerms() {
    whereClauses = append(whereClauses, "media_fts MATCH ?", hybrid.BuildFTSQuery(joinOp))
}

// Phrases via LIKE
for _, phrase := range hybrid.Phrases {
    whereClauses = append(whereClauses, "(path LIKE ? OR title LIKE ? OR description LIKE ?)")
}
```

### 3. Schema Updates

**`internal/db/schema.sql`**, **`e2e/fixtures/schema.sql`**, **`internal/db/migrate.go`**:

```sql
CREATE VIRTUAL TABLE media_fts USING fts5(
    path,
    fts_path,
    title,
    description,
    content='media',
    content_rowid='rowid',
    tokenize = 'trigram',
    detail = 'none'  -- NEW: 82% index size reduction
);
```

## Index Size Comparison

| Configuration | Index Size | Phrase Support |
|---------------|------------|----------------|
| `detail=full` (old) | ~10 GB | ✅ Native FTS |
| `detail=none` (new) | ~1.8 GB | ✅ Hybrid (LIKE) |

## Performance Characteristics

### FTS Terms (Individual Words)
- **Speed**: Fast - direct FTS index lookup
- **Ranking**: Basic (bm25 works but less accurate without column/offset data)

### Phrase Searches (LIKE)
- **Speed**: Moderate - trigram filtering + verification
- **Accuracy**: Exact - LIKE verifies full string match
- **Optimization**: Trigram index filters candidates before LIKE verification

## Usage Examples

```bash
# Simple term search (FTS only)
disco ls "python tutorial"

# Phrase search (LIKE)
disco ls '"video tutorial"'

# Mixed search (FTS + LIKE)
disco ls 'python "video tutorial" beginner'

# Boolean operators
disco ls 'python OR golang "machine learning"'
```

## Limitations

1. **No NEAR queries**: `NEAR(term1 term2, 5)` not supported (wasn't being used)
2. **No column filters**: `title:video` syntax removed (made colons tedious)
3. **Phrase minimum 3 chars**: `"ab"` ignored (trigram requirement)
4. **Slightly slower phrase search**: LIKE vs native FTS phrase query

## Migration

Existing databases will be automatically upgraded:
1. Old FTS table dropped
2. New table created with `detail=none`
3. Rebuilt from `media` and `captions` tables

**Note**: First search after migration may be slower while index rebuilds.

## Testing

```bash
# Test hybrid search utility
go test ./internal/utils -run TestParseHybridSearchQuery -v

# Test filter builder integration
go test ./internal/query -run TestFilterBuilder_Build -v

# Full test suite
go test ./internal/... -short
```

## Future Enhancements

1. **Ranking improvement**: Boost results that match phrases higher than term-only matches
2. **Caption search**: Apply same hybrid approach to `captions_fts`
3. **Highlighting**: Support snippet/highlight generation for phrase matches
