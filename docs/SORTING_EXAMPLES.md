# Complex Sorting Examples

This document provides practical examples of using the complex sorting feature.

## Quick Start

### Use xklb-style default sorting (recommended for most cases)

```bash
# Search mode with xklb default sorting
disco print --sort=default

# Or explicitly
disco print --play-in-order="xklb"
```

This sorts by:
1. Videos before audio
2. Files with audio before silent
3. Local files before remote URLs
4. Files with subtitles first
5. Unplayed/least-played first
6. Resume progress (furthest along first)
7. Least-recently played first
8. Titled entries before untitled
9. Alphabetical tiebreak

### Use DU mode default sorting

```bash
# Disk usage mode with optimized sorting
disco du --play-in-order="du"
```

This sorts folders by:
1. Average size per file (descending)
2. Total size (descending)
3. File count (descending)
4. Folder count (descending)
5. Reverse alphabetical path

## Custom Sorting Examples

### Prioritize by content type

```bash
# Videos first, then by size
disco print --play-in-order="video_count desc,size desc"

# Audio files with lyrics (subtitles) first
disco print --play-in-order="audio_count desc,subtitle_count desc"

# Local media before streaming URLs
disco print --play-in-order="path_is_remote asc"
```

### Playback-based sorting

```bash
# Unplayed content first, then by resume progress
disco print --play-in-order="play_count asc,playhead desc"

# Recently played last (rediscover old content)
disco print --play-in-order="time_last_played asc"

# Binge-watch continuation (most progress first)
disco print --play-in-order="-playhead,play_count desc"
```

### Quality and metadata sorting

```bash
# Titled entries first, then by quality indicators
disco print --play-in-order="title_is_null asc,video_count desc,size desc"

# High-definition content first
disco print --play-in-order="-width,height desc"

# Complete series/collections (has all metadata)
disco print --play-in-order="title desc,artist desc,album desc"
```

### Time-based sorting

```bash
# Newest additions first
disco print --play-in-order="-time_created"

# Oldest unwatched content first
disco print --play-in-order="play_count asc,time_created asc"

# Recently modified files
disco print --play-in-order="-time_modified"
```

### Multi-criteria sorting

```bash
# Smart sorting: quality + unplayed + resume
disco print --play-in-order="video_count desc,audio_count desc,play_count asc,playhead desc"

# Discover hidden gems (unplayed, titled, high quality)
disco print --play-in-order="play_count asc,title_is_null asc,video_count desc,score desc"

# Marathon mode (longest content, unplayed, with subtitles)
disco print --play-in-order="play_count asc,duration desc,subtitle_count desc"
```

## Algorithm Examples

### Natural vs Python sorting

```bash
# Natural sorting (handles numbers intuitively)
disco print --play-in-order="natural_path"
# Result: file1, file2, file10, file100

# Python sorting (lexicographic)
disco print --play-in-order="python_path"
# Result: file1, file10, file100, file2
```

### Case handling

```bash
# Case-insensitive title sorting
disco print --play-in-order="ignorecase_title"

# Locale-aware sorting
disco print --play-in-order="locale_title"
```

## API Examples

### REST API with complex sorting

```bash
# Comma-separated format
curl "http://localhost:8080/api/query?sort_fields=video_count%20desc,play_count%20asc,path%20asc"

# JSON array format
curl "http://localhost:8080/api/query?sort_fields=%5B%22video_count%20desc%22%2C%22play_count%20asc%22%2C%22path%20asc%22%5D"

# With sort_desc modifier
curl "http://localhost:8080/api/query?sort_fields=video_count,play_count,path&sort_desc=video_count,play_count"

# Combine with filters
curl "http://localhost:8080/api/query?type=video&min_duration=30&sort_fields=play_count%20asc,time_last_played%20asc"
```

### Frontend drag-and-drop integration

```javascript
// Initial sort configuration
let sortConfig = [
  "video_count desc",
  "audio_count desc", 
  "play_count asc",
  "playhead desc",
  "path asc"
];

// User reorders via drag-and-drop
function onSortReorder(newOrder) {
  sortConfig = newOrder;
  refreshMediaList();
}

// Apply sorting to API request
function fetchMedia() {
  const sortParam = encodeURIComponent(JSON.stringify(sortConfig));
  fetch(`/api/query?sort_fields=${sortParam}`)
    .then(res => res.json())
    .then(data => renderMedia(data));
}

// Save user preference
function saveSortPreference() {
  localStorage.setItem('sortConfig', JSON.stringify(sortConfig));
}

// Load saved preference
function loadSortPreference() {
  const saved = localStorage.getItem('sortConfig');
  if (saved) {
    sortConfig = JSON.parse(saved);
  }
}
```

## Advanced Use Cases

### Podcast/Audiobook sorting

```bash
# Unplayed episodes first, then by release date
disco print --play-in-order="play_count asc,time_created asc"

# In-progress episodes (resume listening)
disco print --play-in-order="-playhead,play_count asc"

# Series order (by track number if available)
disco print --play-in-order="track_number asc"
```

### Music video sorting

```bash
# By artist, then album, then track
disco print --play-in-order="artist asc,album asc,track_number asc"

# High-quality audio first
disco print --play-in-order="-audio_count,size desc"
```

### Educational content sorting

```bash
# Course order (by title/number)
disco print --play-in-order="natural_title"

# Unwatched lectures first
disco print --play-in-order="play_count asc,time_created asc"
```

### Documentary sorting

```bash
# By category, then unwatched, then newest
disco print --play-in-order="categories asc,play_count asc,-time_created"
```

## Combining with Filters

```bash
# Unplayed videos only, sorted by quality
disco print -w "COALESCE(play_count, 0) = 0" --play-in-order="video_count desc,size desc"

# In-progress content (resume playback)
disco print -w "COALESCE(playhead, 0) > 0" --play-in-order="-playhead"

# High-rated unplayed content
disco print -w "score >= 4" -w "COALESCE(play_count, 0) = 0" --play-in-order="score desc"

# Large files with subtitles (for language learning)
disco print -S ">1GB" --play-in-order="subtitle_count desc,size desc"
```

## Performance Tips

1. **Use indexes**: Sorting by indexed fields (path, time_created, etc.) is faster
2. **Limit results**: Use `-L` to limit results when testing sort orders
3. **Avoid excessive fields**: 3-5 sort fields is usually sufficient
4. **Cache preferences**: Save your favorite sort configurations

```bash
# Test sort on limited dataset
disco print -L 100 --play-in-order="video_count desc,play_count asc"

# Create alias for frequently used sort
alias disco-unplayed='disco print -w "COALESCE(play_count, 0) = 0" --play-in-order="time_created asc"'
```

## Troubleshooting

### Sort not working as expected?

1. Check field names are correct (use `--help` to see available fields)
2. Verify direction modifiers (desc/asc, -prefix)
3. Try simpler sort configurations first
4. Check for typos in field names

### Performance issues?

1. Reduce number of sort fields
2. Use indexed fields when possible
3. Limit result set size
4. Consider database optimization

## See Also

- [COMPLEX_SORTING.md](COMPLEX_SORTING.md) - Complete technical documentation
- `disco print --help` - Command-line help
- `/api/query` endpoint documentation - API reference
