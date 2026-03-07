# Test Cleanup Guide

This document identifies tests that need updates after moving filtering from client-side to server-side.

## Background

We're progressively moving redundant client-side filtering to the server to:
- Ensure `X-Total-Count` accuracy
- Reduce bandwidth (don't fetch filtered-out items)
- Improve pagination correctness
- Simplify client code

**What stays client-side:**
- Local progress merging (localStorage-based tracking)
- Read-only mode progress tracking
- Real-time UI state management

---

## Tests Requiring Updates

### 1. `unplayed-filter.test.js` - UPDATE

**Current behavior tested**: Client-side filtering of played items from server results

**Issue**: Tests the REDUNDANT client-side filtering (lines 2468-2476 in app.js) that we're removing from main query flow.

**What to keep testing**:
- Local progress merging (localStorage) still works
- `localResume` feature filters correctly

**Action**:
```javascript
// UPDATE test to verify:
// 1. Server receives unplayed=true parameter
// 2. Local progress merging still works for localStorage items
// 3. Remove assertion about client-side filtering of server results

// Keep this behavior (local merge):
localStorage.setItem('disco-progress', JSON.stringify({...}));
// Server should receive unplayed=true param
expect(fetch).toHaveBeenCalledWith(expect.stringContaining('unplayed=true'), ...);
```

---

### 2. `history-pages.test.js` - PARTIAL UPDATE

**Tests that should stay** (testing local merge):
- ✅ "In Progress page merges local progress and respects type filters" - Tests localStorage merge
- ✅ "Completed page merges local play counts and respects type filters" - Tests localStorage merge

**Tests to check**:
- Any test that verifies client-side filtering of server results (not local merge)

**Action**:
- Keep tests for localStorage merging
- Remove/update tests that verify redundant client-side filtering
- Add tests verifying server receives correct params

---

### 3. `toolbar-filter.test.js` - MINOR UPDATE

**Current tests**: Verify type filtering sends correct params to server

**Status**: ✅ Mostly correct - already tests server-side filtering

**Action**:
- Verify tests check for `type` param in ALL relevant endpoints (including `/api/episodes`)
- Add test for database filter param (`db=`)

---

### 4. `history-toggle.test.js` - NO CHANGE

**Tests**: UI toggle state management

**Status**: ✅ Keep as-is - tests UI state, not filtering logic

---

### 5. `trash-filter.test.js` - REVIEW

**Tests**: Trash page filtering behavior

**Action**:
- Review for redundant client-side filtering tests
- Keep tests for trash-specific behavior

---

### 6. `history-group.test.js` - REVIEW

**Tests**: Episodes view with history filters

**Action**:
- Verify tests check server receives `unfinished=true` etc. params
- Remove tests for redundant client-side filtering in `renderEpisodes()`

---

### 7. `pagination-limit.test.js` - UPDATE

**Tests**: Pagination behavior

**Action**:
- Add test for database filtering pagination
- Verify `X-Total-Count` matches displayed results after DB filter
- Already added in `e2e/tests/pagination-limit.spec.ts`

---

## Tests to Add

### 1. Server-Side Database Filtering

```javascript
it('sends db filter to server when databases are toggled', async () => {
    // Setup with multiple databases
    window.disco.state.databases = ['/db1.db', '/db2.db'];
    
    // Deselect one database
    const dbBtn = document.querySelector('[data-db="/db1.db"]');
    dbBtn.click();
    
    await vi.waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
            expect.stringContaining('db=/db2.db'),
            expect.any(Object)
        );
    });
});
```

### 2. Type Filter in Episodes View

```javascript
it('sends type param to /api/episodes when type filter is selected', async () => {
    const viewGroupBtn = document.getElementById('view-group');
    viewGroupBtn.click();
    
    const audioBtn = document.querySelector('[data-type="audio"]');
    audioBtn.click();
    
    await vi.waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
            expect.stringContaining('/api/episodes'),
            expect.stringContaining('type=audio')
        );
    });
});
```

### 3. Search Param in Playlist View

```javascript
it('sends search param when searching in playlist view', async () => {
    // Navigate to playlist
    // Type in search
    expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('search=...'),
        expect.stringContaining('title=...')
    );
});
```

---

## Test File Status Summary

| File | Action | Priority | Notes |
|------|--------|----------|-------|
| `unplayed-filter.test.js` | UPDATE | High | Remove redundant client filter tests |
| `history-pages.test.js` | PARTIAL UPDATE | High | Keep local merge, remove redundant |
| `toolbar-filter.test.js` | MINOR UPDATE | Medium | Add DB filter test |
| `history-toggle.test.js` | NO CHANGE | - | UI state tests are fine |
| `trash-filter.test.js` | REVIEW | Medium | Check for redundant filtering |
| `history-group.test.js` | REVIEW | Medium | Check episodes rendering |
| `pagination-limit.test.js` | UPDATE | High | Add DB filter tests |
| `race.test.js` | REVIEW | Low | Check unplayed filter usage |
| `progress.test.js` | NO CHANGE | - | Tests progress API, not filtering |

---

## Implementation Checklist

After moving each filter server-side:

1. **Update backend** (if needed)
   - [ ] Add query param support
   - [ ] Add tests for new param

2. **Update frontend**
   - [ ] Add param to API request
   - [ ] Remove redundant client-side filtering
   - [ ] Update display count logic

3. **Update tests**
   - [ ] Remove tests for client-side filtering
   - [ ] Add tests for server param
   - [ ] Verify `X-Total-Count` accuracy
   - [ ] Run all tests

4. **Manual verification**
   - [ ] Test in browser with dev tools
   - [ ] Verify network requests have correct params
   - [ ] Check pagination works correctly
   - [ ] Verify count matches results

---

## Quick Reference: What Stays Client-Side

```javascript
// ✅ KEEP - Local progress merging (server doesn't know about localStorage)
if (state.localResume) {
    const localProgress = JSON.parse(localStorage.getItem('disco-progress'));
    // Merge logic...
}

// ✅ KEEP - Read-only mode progress tracking
if (state.readOnly) {
    // Local updates only
}

// ✅ KEEP - UI state management
state.filters.unplayed = true; // UI state
state.currentPage = 2; // Pagination state

// ❌ REMOVE - Redundant filtering of server results
currentMedia = currentMedia.filter(item => {
    // If server already filters by this, remove
});
```
