# Build Modes

Discoteca supports two different build modes, each with different full-text search capabilities:

## Build Modes Overview

| Mode | Build Command | Binary Size | Search Capability | Best For |
|------|--------------|-------------|-------------------|----------|
| **FTS5** (default) | `make build-fts5` | ~22MB | Full-text search with trigram | Most users |
| **No-FTS** | `make build-nofts` | ~22MB | LIKE-based substring search | Minimal dependencies |

---

## 1. FTS5 Build (Default)

**Build command:**
```bash
make build-fts5
# or
go build -tags "fts5" -o disco ./cmd/disco
```

### Features
- SQLite FTS5 full-text search with trigram tokenizer
- Substring-like search capabilities
- Integrated with SQLite database
- No external dependencies beyond SQLite

### Search Performance (10k rows)
- Prefix search: ~500μs
- Substring search: ~2ms (via trigram)
- Phrase search: ~5.5ms

### Usage
```bash
# FTS5 is used automatically when available
disco print my_videos.db --fts --search "matrix"

# Or specify FTS table
disco search my_videos.db "matrix" --fts-table media_fts
```

### Pros
- ✅ Single database file (no separate index)
- ✅ Automatic index maintenance via triggers
- ✅ Good performance for most use cases
- ✅ No additional dependencies

### Cons
- ❌ Requires SQLite compiled with FTS5
- ❌ Trigram tokenizer may not match exact substrings

---

## 2. No-FTS Build

**Build command:**
```bash
make build-nofts
# or
go build -tags "" -o disco ./cmd/disco
```

### Features
- Basic LIKE-based substring search
- No full-text search capabilities
- Minimal dependencies
- Smallest feature set

### Search Performance (10k rows)
- Prefix search: ~500μs (with index)
- Substring search: ~1ms
- EndsWith search: ~660μs

### Usage
```bash
# Uses LIKE automatically (no --fts flag needed)
disco print my_videos.db --search "matrix"

# Explicit substring search
disco print my_videos.db --search "%matrix%"
```

### Pros
- ✅ Works with any SQLite build
- ✅ No FTS5 dependency
- ✅ Simple and predictable
- ✅ Good performance for small datasets (<100k rows)

### Cons
- ❌ No full-text search features
- ❌ Substring searches require full table scans
- ❌ Slower on large datasets
- ❌ No relevance ranking

---

## Choosing a Build Mode

### Use FTS5 (default) if:
- You want the best balance of features and simplicity
- Your SQLite has FTS5 support (most do)
- You want a single database file
- You need good substring search performance

### Use No-FTS if:
- You want minimal dependencies
- Your dataset is small (<100k rows)
- You only need basic substring search
- You're on a constrained environment

---

## Build Comparison

### Binary Sizes
```
disco-fts5:    22MB (default, recommended)
disco-nofts:   22MB (minimal)
```

### Dependencies
- **FTS5**: SQLite with FTS5 module
- **No-FTS**: None beyond SQLite

### Search Syntax

**FTS5:**
```bash
# Token-based search
disco search db "matrix"           # Finds "matrix", "matrices"
disco search db "mat*"             # Prefix wildcard
disco search db "\"matrix reloaded\""  # Exact phrase
disco search db "matrix AND neo"   # Boolean operators
```

**No-FTS:**
```bash
# LIKE-based search
disco print db --search "matrix"   # Becomes LIKE '%matrix%'
disco print db --exact --search "matrix"  # Exact match
```

---

## Switching Build Modes

You can have multiple builds installed simultaneously:

```bash
# Install with different names
make build-fts5 && cp disco disco-fts5
make build-nofts && cp disco disco-nofts

# Use the one you need
./disco-fts5 print db --search "test"
./disco-nofts print db --search "test"
```

Or install to GOPATH with different tags:
```bash
BUILD_TAGS=fts5 make install    # $GOPATH/bin/disco
BUILD_TAGS="" make install      # Overwrites previous
```

---

## Migration Between Build Modes

### FTS5 ↔ No-FTS
No migration needed - both use the same SQLite database schema.

---

## Technical Details

### Build Tags
- `fts5`: Enable FTS5 support
- No tags: Basic LIKE-only search

### File Structure
```
internal/db/
  init_fts5.go       # FTS5 build
  init_no_fts5.go    # Non-FTS build

internal/query/
  search_mode.go     # Search mode selection
```

### Conditional Compilation
Go build tags control which files are compiled:
- `//go:build fts5` - Only compiled with `-tags fts5`
- `//go:build !fts5` - Compiled when fts5 tag is not present
