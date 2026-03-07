# Client-Side Filtering Cleanup TODOs

This document tracks opportunities to move client-side filtering to the server for consistency and performance.

## ✅ Completed

### 1. Database Filtering
- **Status**: DONE - Moved to server-side
- **Change**: `excludedDbs` now sent as `db` query parameter to server
- **Benefit**: `X-Total-Count` now accurate, pagination works correctly, less bandwidth
- **Files**: `web/app.js`, `web/state.js`, `internal/query/query.go`

---

## 🔍 To Investigate

### 2. Media Type Filtering in Playlist/Episodes Views
**Location**: `web/app.js` lines 1243-1250 (playlist), lines 1894-1902 (episodes)

**Current behavior**:
```javascript
filtered = filtered.filter(item => {
    const mime = item.type || '';
    if (hasVideo && mime.startsWith('video')) return true;
    if (hasAudio && mime.startsWith('audio')) return true;
    // ...
});
```

**Issue**: Type filtering is applied client-side in playlist and episodes views, but the server already supports `type` parameter (see `parseFlags` in `serve.go`).

**TODO**: 
- [ ] Verify server `type` parameter works for playlist/episodes endpoints
- [ ] Add `type` params to `fetchPlaylistItems()` and `fetchEpisodes()` requests
- [ ] Remove client-side type filtering from `filterPlaylistItems()` and `renderEpisodes()`
- [ ] Update tests: `toolbar-filter.test.js`

**Complexity**: Low - Server already supports this

---

### 3. Search Text Filtering in Playlist/Episodes Views  
**Location**: `web/app.js` lines 1253-1260 (playlist), lines 1905-1910 (episodes)

**Current behavior**:
```javascript
// Filter by search text (client-side)
if (state.filters.search) {
    const query = state.filters.search.toLowerCase();
    filtered = filtered.filter(item => {
        const title = (item.title || '').toLowerCase();
        const path = (item.path || '').toLowerCase();
        return title.includes(query) || path.includes(query);
    });
}
```

**Issue**: Search is done via simple string matching client-side. Server has full FTS search capability.

**TODO**:
- [ ] Add `search` param to playlist/episodes API requests
- [ ] Verify server search works for these endpoints
- [ ] Remove client-side search filtering
- [ ] Consider: Should playlist search search only playlist items, or re-query with search + playlist constraint?

**Complexity**: Medium - Need to clarify search semantics for playlists

---

### 4. Progress Filtering in Episodes View
**Location**: `web/app.js` lines 1912-1922

**Current behavior**:
```javascript
// Client-side progress filtering
if (state.filters.unplayed) {
    if (getPlayCount(f) > 0 || (f.playhead || 0) > 0) return false;
} else if (state.filters.unfinished) { ... }
```

**Issue**: This is REDUNDANT with the main query filtering (lines 2468-2476). The server already filters by `unplayed`, `unfinished`, `completed` via `parseFlags`.

**TODO**:
- [ ] Verify server progress flags are being sent in `/api/episodes` request
- [ ] Check if `appendFilterParams()` is called for episodes fetch
- [ ] Remove redundant client-side progress filtering from `renderEpisodes()`
- [ ] Keep ONLY the local progress merging (localResume) which is still needed

**Complexity**: Low - Just need to ensure params are passed correctly

**Note**: Keep local progress merging (`state.localResume`) as this is for localStorage-based tracking that server doesn't know about.

---

### 5. Caption Search Text Filtering
**Location**: `web/app.js` lines 2480-2525

**Current behavior**: Client-side text matching on `caption_text` field with context expansion.

**Issue**: Server has FTS5 full-text search for captions, but client does additional filtering.

**TODO**:
- [ ] Investigate if server caption search already handles this correctly
- [ ] If server returns pre-filtered results, remove client-side filtering
- [ ] Keep context expansion logic if needed (2 before/after matches)

**Complexity**: Medium - Need to understand caption search flow

**Note**: This may be intentionally client-side for the "context" feature (showing 2 captions before/after match)

---

## 🚫 Should Stay Client-Side

### History/Progress Filtering (Main Query)
**Location**: `web/app.js` lines 2468-2476

**Reason**: This is NECESSARY for:
1. `localResume` feature - localStorage progress server doesn't know about
2. Read-only mode where server can't update play counts
3. Real-time UI updates without server round-trip

**Keep**: ✅ Yes - This is legitimate client-side state management

---

### Local Progress Merging
**Location**: `web/app.js` lines 1815-1845, 2435-2465

**Reason**: Merges localStorage progress with server data for:
- Offline playback tracking
- Multi-device sync gaps
- Read-only mode support

**Keep**: ✅ Yes - Server has no access to localStorage data

---

### Playlist Client-Side Filtering
**Location**: `web/app.js` `filterPlaylistItems()`

**Reason**: Playlists are stored client-side and filtered locally for instant response.

**Keep**: ✅ Yes - But could add server-side search for large playlists

---

## Tests to Update/Remove

After removing redundant client-side filtering, these tests may need updates:

### Tests that test client-side filtering behavior:
1. **`unplayed-filter.test.js`** - Tests client-side unplayed filtering
   - Action: Update to test server-side filtering + local merge only

2. **`history-pages.test.js`** - Tests client-side data merging
   - Action: Keep tests for local merge, remove redundant filter tests

3. **`toolbar-filter.test.js`** - Tests type filtering
   - Action: Update to verify server receives correct params

4. **`history-toggle.test.js`** - Tests toggle behavior
   - Action: Keep - tests UI state management, not filtering logic

5. **`trash-filter.test.js`** - Tests trash filtering
   - Action: Review - may need updates

### Tests that should remain unchanged:
- Tests for local progress merging (localStorage)
- Tests for UI state toggles
- Tests for read-only mode behavior

---

## Implementation Priority

1. **High Priority** (clear wins, low risk):
   - #4 Progress filtering in Episodes view (remove redundancy)
   - #2 Type filtering in Playlist/Episodes (server already supports)

2. **Medium Priority** (needs investigation):
   - #3 Search in Playlist/Episodes (clarify semantics first)
   - #5 Caption search (understand context feature)

3. **Low Priority**:
   - Optimization of existing client-side filters that are working correctly

---

## Verification Steps

After each change:
1. Run `go test ./...` - ensure backend tests pass
2. Run e2e tests: `cd e2e && npm test`
3. Manual testing:
   - Multi-database setup with filtering
   - Playlist with type filters
   - Episodes view with progress filters
   - Search in various views
4. Check network tab - verify correct params sent to server
5. Verify `X-Total-Count` matches displayed results
