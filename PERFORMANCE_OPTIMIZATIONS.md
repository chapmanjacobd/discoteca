# Frontend Performance Optimization Opportunities

## Current Data-Heavy Operations

### 1. Disk Usage View (CRITICAL - HIGH PRIORITY)
**Current behavior:**
- Fetches ALL media files recursively under the current path
- Aggregates in backend but sends all files to frontend
- For deep directory trees, this can be thousands/millions of rows

**Optimization:**
- ✅ Add SQL-level depth filtering to only fetch files at target depth
- ✅ Aggregate folder sizes in SQL using GROUP BY
- ✅ Only return immediate children with pre-calculated sizes

**Implementation:**
```sql
-- For path "/media" at depth 2, only get files at depth 3
SELECT 
    substr(path, 1, length(?) + instr(substr(path, length(?) + 1), '/')) as parent_path,
    COUNT(*) as file_count,
    SUM(size) as total_size,
    SUM(duration) as total_duration
FROM media 
WHERE path LIKE ? || '%' 
  AND path NOT LIKE ?
  AND time_deleted = 0
GROUP BY parent_path
ORDER BY total_size DESC
```

### 2. Captions View (MEDIUM PRIORITY)
**Current behavior:**
- Fetches all caption rows individually
- Frontend groups by media path
- No aggregation of caption counts/sizes

**Optimization:**
- Add `?aggregate=true` parameter to captions endpoint
- Backend returns grouped data with:
  - Total caption count per media
  - Total duration of captions
  - First/last caption timestamps
  - Match highlighting info for searches

**New endpoint or parameter:**
```
GET /api/query?captions=true&aggregate=true&search=keyword
```

### 3. Search with Filters (MEDIUM PRIORITY)
**Current behavior:**
- Fetches all matching media with full metadata
- Frontend applies some filters client-side
- No pre-aggregated counts for filter UI

**Optimization:**
- Add `?include_counts=true` for filter bin counts
- Backend returns both results AND available filter counts
- Eliminates separate `/api/filter-bins` call

**Response format:**
```json
{
  "items": [...],
  "counts": {
    "types": {"video": 100, "audio": 50},
    "extensions": {".mp4": 80, ".mkv": 20},
    "sizes": [{"label": "<100MB", "count": 50}],
    ...
  }
}
```

### 4. Group/Episodes View (LOW-MEDIUM PRIORITY)
**Current behavior:**
- Fetches all media, frontend groups by parent directory
- No pre-calculated episode counts or totals

**Optimization:**
- Add `?group_by=parent` parameter
- Backend returns pre-grouped data with counts
- Similar to DU aggregation but for episode display

### 5. Random Clip / Channel Surf (LOW PRIORITY)
**Current behavior:**
- Fetches random clip with full media metadata
- Works well currently, minor optimization possible

**Optimization:**
- Add `?fields=path,type,duration,start,end` to limit response
- Only fetch necessary fields for random clip

### 6. Thumbnail Generation (LOW PRIORITY)
**Current behavior:**
- Each media card requests thumbnail individually
- Many small HTTP requests

**Optimization:**
- Add bulk thumbnail endpoint: `POST /api/thumbnails` with paths array
- Or use data URIs in list responses for small thumbnails
- Consider sprite sheets for grid view

### 7. Progress/Play Count Updates (LOW PRIORITY)
**Current behavior:**
- Individual API calls for each progress update
- Could batch multiple updates

**Optimization:**
- Add `POST /api/progress/batch` for multiple updates
- Queue updates and send in batches

---

## Implementation Priority

### Phase 1 (Immediate - DU View)
1. Add depth-based SQL filtering for DU endpoint
2. Add SQL aggregation for folder sizes
3. Test with large directory trees (10k+ files)

### Phase 2 (Short-term - Captions)
1. Add aggregation parameter to captions endpoint
2. Update frontend to use aggregated data
3. Add search result highlighting info

### Phase 3 (Medium-term - Search/Filters)
1. Add filter counts to search response
2. Remove separate filter-bins endpoint calls
3. Optimize filter UI updates

### Phase 4 (Long-term - Other)
1. Group/Episodes aggregation
2. Bulk thumbnail endpoint
3. Batch progress updates
