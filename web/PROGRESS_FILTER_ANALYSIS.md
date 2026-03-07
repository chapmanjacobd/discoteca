# Progress Filtering in Episodes View - Analysis

## Current Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    User clicks "In Progress"                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  fetchEpisodes() is called                                       │
│  - Calls appendFilterParams()                                    │
│  - Sends: /api/episodes?unfinished=true&type=video              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  SERVER receives request                                         │
│  - parseFlags() extracts unfinished=true                         │
│  - MediaQuery() builds query with:                               │
│    WHERE COALESCE(playhead, 0) > 0                               │
│  - Returns ONLY items with playhead > 0                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Frontend receives server results                                │
│  - Server already filtered by progress                           │
│  - Results: [{path: "video1.mp4", playhead: 10, ...}]           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  LOCAL RESUME: Merge localStorage items                          │
│  - Check localStorage for items not in server results            │
│  - Fetch missing items via fetchMediaByPaths()                   │
│  - Filter THESE localStorage items (lines 1824-1830):            │
│    if unfinished: keep if playhead > 0                           │
│  - Merge into groups                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  renderEpisodes() is called                                      │
│  - CURRENTLY: Filters AGAIN (lines 1912-1922) ❌ REDUNDANT       │
│    if unfinished: filter out if playhead === 0                   │
│  - This is filtering SERVER results that are already filtered!   │
└─────────────────────────────────────────────────────────────────┘
```

## What Can Be Removed

### ❌ REMOVE - Lines 1912-1922 in renderEpisodes()
```javascript
// Client-side progress filtering
if (state.filters.unplayed) {
    if (getPlayCount(f) > 0 || (f.playhead || 0) > 0) return false;
} else if (state.filters.unfinished) {
    if (getPlayCount(f) > 0 || (f.playhead || 0) === 0) return false;
} else if (state.filters.completed) {
    if (getPlayCount(f) === 0) return false;
} else if (state.page === 'history') {
    if ((f.time_last_played || 0) === 0) return false;
}
```

**Why remove?** Server already filtered by these criteria. This is redundant.

---

## What Must Stay

### ✅ KEEP - Lines 1824-1830 (localStorage merge filtering)
```javascript
// Client-side filtering for merged items
if (state.filters.unfinished) {
    missingData = missingData.filter(item => 
        getPlayCount(item) === 0 && (localProgress[item.path]?.pos > 0)
    );
} else if (state.filters.completed) {
    missingData = missingData.filter(item => getPlayCount(item) > 0);
}
```

**Why keep?** This filters items from localStorage that the server doesn't know about.

### ✅ KEEP - Lines 1855-1865 (localStorage progress overlay)
```javascript
// Update playhead and time_last_played from localStorage
if (state.localResume) {
    const localProgress = JSON.parse(localStorage.getItem('disco-progress'));
    groups.forEach(group => {
        group.files.forEach(item => {
            const local = localProgress[item.path];
            if (local) {
                // Update item with local progress
                item.playhead = localPlayhead;
                item.time_last_played = localTime;
            }
        });
    });
}
```

**Why keep?** This overlays localStorage progress on server results for accurate UI display.

---

## Edge Case Analysis

### Q: "Without client filtering, may we get results which conflict with localStorage?"

**A: No, because:**

1. **Server results** are already filtered by the server based on the same criteria
2. **LocalStorage items** go through separate filtering (lines 1824-1830) which we KEEP
3. **Progress overlay** (lines 1855-1865) updates server results with localStorage data

### Example Scenario

```javascript
// localStorage has:
{
  "video1.mp4": { pos: 50, last: 1234567890 },  // In progress locally
  "video2.mp4": { pos: 0, last: 1234567890 }    // Not started locally
}

// Server returns (filtered by unfinished=true):
[
  { path: "video1.mp4", playhead: 45, ... }  // Has playhead on server
]
// Note: video2.mp4 NOT returned because server playhead = 0

// After localStorage merge (lines 1824-1830):
// - video2.mp4 is fetched via fetchMediaByPaths()
// - But filtered OUT because pos=0 doesn't match unfinished criteria
// - Only items with pos > 0 are merged

// After progress overlay (lines 1855-1865):
[
  { path: "video1.mp4", playhead: 50, ... }  // Updated with localStorage
]

// Result: Correct! Only video1.mp4 shown as in-progress
```

---

## What If Server Doesn't Receive the Params?

**Check**: Does `fetchEpisodes()` call `appendFilterParams()`?

**Answer**: YES (line 1789):
```javascript
async function fetchEpisodes() {
    const params = new URLSearchParams();
    appendFilterParams(params);  // ✅ Sends unplayed/unfinished/completed
    // ...
    const resp = await fetchAPI(`/api/episodes?${params.toString()}`);
}
```

And `appendFilterParams()` includes (lines 680-683):
```javascript
if (state.filters.unplayed) params.append('unplayed', 'true');
if (state.filters.unfinished) params.append('unfinished', 'true');
if (state.filters.completed) params.append('completed', 'true');
```

---

## Verification Steps

Before removing, verify:

1. ✅ `fetchEpisodes()` calls `appendFilterParams()` - YES (line 1789)
2. ✅ `appendFilterParams()` sends progress params - YES (lines 680-683)
3. ✅ Server handles progress params - YES (query.go lines 281-295)
4. ✅ LocalStorage merge has separate filtering - YES (lines 1824-1830)

**Conclusion**: Safe to remove lines 1912-1922.

---

## Similar Analysis for Other Views

### Playlist View (filterPlaylistItems)

Same pattern:
- Server receives type/search params
- Client filters redundantly
- Can remove client-side filtering

### Main Query (performSearch)

Different pattern:
- Server receives progress params
- Client-side filtering REMAINS for localStorage merge
- This is CORRECT - keep as-is

---

## Implementation Plan

1. **Remove** lines 1912-1922 from `renderEpisodes()`
2. **Remove** similar filtering from `filterPlaylistItems()` (lines 1243-1260)
3. **Keep** localStorage merge filtering (lines 1824-1830)
4. **Keep** progress overlay (lines 1855-1865)
5. **Update tests** to verify server receives correct params
6. **Test manually** with localStorage data to ensure no regression
