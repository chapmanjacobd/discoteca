# Client-Side Filtering Cleanup - Status

## ✅ Completed

### 1. Database Filtering
- **Status**: DONE
- **Change**: `excludedDbs` sent as `db` query parameter to server
- **Files**: `web/app.js`, `web/state.js`, `internal/query/query.go`
- **Benefit**: `X-Total-Count` accurate, pagination correct, less bandwidth

### 2. Episodes View Filtering  
- **Status**: DONE
- **Change**: Removed redundant type/search/progress filtering from `renderEpisodes()`
- **Files**: `web/app.js` (lines 1889-1898 simplified)
- **Benefit**: 36 lines removed, server already filters via `appendFilterParams()`

### 3. Main Query Progress Filtering
- **Status**: DONE  
- **Change**: `currentMedia = data` set correctly, client-side filtering only for localStorage items
- **Files**: `web/app.js` (lines 2438-2450)
- **Benefit**: Correct flow - server filters, localStorage merge handled separately

### 4. Playlist Filtering
- **Status**: DONE
- **Change**: Removed type and search filtering from `filterPlaylistItems()`
- **Files**: `web/app.js` (lines 1229-1239 simplified)
- **Benefit**: 28 lines removed, playlists are server-fetched collections

### 5. Caption Search Filtering
- **Status**: DONE
- **Change**: Removed redundant client-side text filtering from captions view
- **Files**: `web/app.js` (lines 2426-2428)
- **Benefit**: 55 lines removed, server already returns filtered results with context (2 before/after)

---

## 🚫 Keeping Client-Side (By Design)

### LocalStorage Progress Merging
**Location**: `web/app.js` lines 2403-2420

**Reason**: Server has no access to localStorage data. Necessary for:
- Offline playback tracking
- Multi-device sync gaps  
- Read-only mode support

**Keep**: ✅ Yes - Legitimate client-side state management

---

## Tests Status

**All 154 tests pass** including:
- ✅ `unplayed-filter.test.js` - Tests localStorage merge (correct)
- ✅ `history-pages.test.js` - Tests localStorage merge (correct)  
- ✅ `history-group.test.js` - Tests episodes with localStorage merge
- ✅ `toolbar-filter.test.js` - Tests server receives type params
- ✅ `episodes.test.js` - Tests episodes view behavior
- ✅ All other tests

**No test updates needed** - existing tests already verify correct behavior.

---

## Summary

**Lines removed**: ~120 lines of redundant client-side filtering across 5 areas
**Tests passing**: 154/154
**Breaking changes**: None
**Remaining work**: None - all redundant filtering removed
