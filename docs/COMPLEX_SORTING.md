# Complex Sorting Guide

This document describes the advanced sorting capabilities available in discoteca, including the xklb-style default sorting and custom multi-field sorting.

## Overview

Discoteca now supports complex multi-field sorting with drag-and-drop reordering capabilities. The sorting system is designed to be compatible with xklb's lambda sorting while providing additional flexibility.

## Default Sorting (xklb-style)

When using the "Default" sort option, discoteca applies xklb-style sorting that prioritizes media in the following order:

1. **video_count DESC** - Videos before audio-only files
2. **audio_count DESC** - Files with audio before silent ones
3. **path_is_remote ASC** - Local files before remote URLs (http://)
4. **subtitle_count DESC** - Files with subtitles first
5. **play_count ASC** - Unplayed/least-played files first
6. **playhead DESC** - Furthest along in playback first (resume progress)
7. **time_last_played ASC** - Least-recently played first
8. **title_is_null ASC** - Titled entries before untitled ones
9. **path ASC** - Alphabetical tiebreak

### Usage

```bash
# Use xklb default sorting
disco print --sort=default

# Or explicitly
disco print --play-in-order="xklb"
```

## DU Mode Default Sorting

For disk usage (DU) mode, the default sorting optimizes for finding large directories:

1. **size_per_count DESC** - Average size per file (size/count)
2. **size DESC** - Total size
3. **count DESC** - File count
4. **folders DESC** - Folder count
5. **path DESC** - Reverse alphabetical path

### Usage

```bash
# Use DU default sorting
disco du --play-in-order="du"
```

## Custom Multi-Field Sorting

You can specify custom sorting with multiple fields using comma-separated values.

### Syntax

```
field1 [direction],field2 [direction],...
```

**Directions:**
- `asc` or no suffix - Ascending order
- `desc` - Descending order
- `-` prefix - Reverses the field direction
- `reverse_` prefix - Reverses the field direction

### Examples

```bash
# Sort by video count (desc), then audio count (desc), then path (asc)
disco print --play-in-order="video_count desc,audio_count desc,path asc"

# Using minus prefix for descending
disco print --play-in-order="-video_count,-audio_count,path"

# Mix algorithms with fields
disco print --play-in-order="natural_path,title desc,-play_count"

# Python-style string comparison
disco print --play-in-order="python_title,-size"
```

## API Usage

The web API supports complex sorting through the `sort_fields` parameter:

### Query Parameters

- `sort` - Simple sort field (e.g., `path`, `title`, `default`)
- `sort_fields` - Complex sorting (JSON array or comma-separated string)
- `sort_desc` - Comma-separated list of fields to sort descending

### Examples

```bash
# Using comma-separated string
curl "http://localhost:8080/api/query?sort_fields=video_count%20desc,audio_count%20desc,path%20asc"

# Using JSON array
curl "http://localhost:8080/api/query?sort_fields=[\"video_count%20desc\",\"audio_count%20desc\",\"path%20asc\"]"

# Using sort_desc modifier
curl "http://localhost:8080/api/query?sort_fields=video_count,audio_count,path&sort_desc=video_count,audio_count"
```

## Supported Fields

### Numeric Fields

- `video_count` - Number of video streams
- `audio_count` - Number of audio streams
- `subtitle_count` - Number of subtitle streams
- `play_count` - Number of times played
- `playhead` - Current playback position (seconds)
- `time_last_played` - Last played timestamp
- `time_created` - Creation timestamp
- `time_modified` - Modification timestamp
- `time_downloaded` - Download timestamp
- `time_deleted` - Deletion timestamp
- `duration` - Duration in seconds
- `size` - File size in bytes
- `width` - Video width
- `height` - Video height
- `fps` - Frames per second
- `score` - User rating/score
- `track_number` - Track number

### String Fields

- `path` - Full file path
- `title` - Media title
- `parent` - Parent directory
- `stem` - Filename without extension
- `ps` - Parent + stem (default)
- `pts` - Parent + title + stem
- `type` - Media type (video, audio, etc.)
- `genre` - Genre
- `artist` - Artist
- `album` - Album
- `language` - Language
- `categories` - Categories
- `video_codecs` - Video codecs
- `audio_codecs` - Audio codecs
- `extension` - File extension

### Computed Fields

- `path_is_remote` - 1 if path starts with "http", 0 otherwise
- `title_is_null` - 1 if title is null/empty, 0 otherwise
- `size_per_count` - Size divided by count (for folder stats)

## Algorithm Modifiers

You can prefix fields with sorting algorithms:

- `natural_` - Natural sorting (default)
- `python_` - Python-style lexicographic sorting
- `ignorecase_` - Case-insensitive sorting
- `locale_` - Locale-aware sorting

### Example

```bash
# Use natural sorting for path, Python sorting for title
disco print --play-in-order="natural_path,python_title desc"
```

## Drag-and-Drop Reordering

The sorting system is designed to support drag-and-drop reordering in the frontend. The frontend can:

1. Display current sort order as a list of fields
2. Allow users to reorder fields via drag-and-drop
3. Send the reordered list to the API via `sort_fields` parameter

### Frontend Integration Example

```javascript
// Current sort configuration
const sortConfig = [
  "video_count desc",
  "audio_count desc",
  "path asc"
];

// After user reorders via drag-and-drop
const newSortConfig = [
  "path asc",
  "video_count desc",
  "audio_count desc"
];

// Send to API
fetch(`/api/query?sort_fields=${encodeURIComponent(JSON.stringify(newSortConfig))}`);
```

## Preset Keywords

Use these keywords for common sorting patterns:

- `xklb` or `xklb_default` - xklb-style default sorting
- `du` or `du_default` - DU mode default sorting
- `default` - Uses xklb default sorting

## Migration from xklb Lambda Sorting

If you're familiar with xklb's lambda sorting, here's the equivalent:

**xklb:**
```python
PlayInOrder(lambda x: (
    -(int(bool(x.get("video_count")))),
    -(int(bool(x.get("audio_count")))),
    int((x.get("path") or "").startswith("http")),
    -(int(bool(x.get("subtitle_count")))),
    x.get("play_count") or 0,
    -((x.get("playhead") or 0)),
    x.get("time_last_played") or 0,
    int(x.get("title") is None),
    x.get("path") or "",
))
```

**discoteca:**
```bash
disco print --play-in-order="video_count desc,audio_count desc,path_is_remote asc,subtitle_count desc,play_count asc,playhead desc,time_last_played asc,title_is_null asc,path asc"
```

Or simply:
```bash
disco print --sort=default
```

## Testing

Run the sorting tests:

```bash
go test ./internal/query/... -run TestSort -v
```

## Implementation Details

The sorting system uses a stable sort algorithm, which means that when two items are equal according to the current sort field, their relative order from previous sort fields is preserved. This is essential for multi-field sorting to work correctly.

The `compareSortFields` function compares two media items across all specified fields in order, returning as soon as a difference is found. This provides efficient sorting even with many fields.
