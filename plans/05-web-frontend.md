# Phase 5: Web Frontend Correctness

Goal: every visible control is connected, every URL state round-trips, and no feature is routed to
the wrong viewer or endpoint.

## 5.1 Merge the two search input handlers

Files: `web/src/main.ts`

Current problem: the path-suggestion/backslash handler at `7638-7652` is overwritten by the
throttled search handler at `8416-8418`. `/` and `./` suggestions never run; pasted Windows paths
are no longer normalized.

Tasks:

- [ ] Create one `onSearchInput` function that:
  - normalizes backslashes,
  - calls `fetchSuggestions` for `/` and `./`,
  - otherwise hides suggestions and calls the debounced search.
- [ ] Assign it exactly once.
- [ ] Keep `Enter` and focus behavior in the same place.
- [ ] Add a frontend test for backslash normalization, `/` suggestions, and debounce.

## 5.2 Fix RSVP routing

Files: `web/src/main.ts`

Current problem: `playRSVP` sets `type: 'video/webm'`, but `openActivePlayer` routes on
`media_type`. The original document type remains, so RSVP opens the document viewer.

Tasks:

- [ ] Set `media_type: 'video'` (and any other fields the player expects) in the RSVP item.
- [ ] Add a unit test that `playRSVP` reaches `openInPiP` for a PDF/text item.
- [ ] Remove the `type` field if it is not used anywhere else.

## 5.3 Fix URL/history state

Files: `web/src/main.ts`

Current problems:

- `mode=search` is never written, so every search/filter pushes a new history entry.
- DU navigation switches to `replaceState` once `path=` exists, so deep navigation cannot go back.
- `language` is written by neither `syncUrl` nor restored consistently.
- Search/category/rating state is written for trash/history/DU/captions but not restored.

Tasks:

- [ ] Define one URL schema: `mode`, `path`, `search`, `sort`, `reverse`, `limit`, `page`,
      `language`, `category`, `rating`, `min_score`, `max_score`, `db`, `view`.
- [ ] Write `mode=search` explicitly.
- [ ] Use `pushState` for mode changes and explicit navigation; use `replaceState` for filter
      changes within the same mode.
- [ ] Restore all written params in `readUrl`, including search text and mode-specific filters.
- [ ] Add round-trip tests: set filters -> sync -> read -> same state.

## 5.4 Await database state before the first query

Files: `web/src/main.ts`

Current problems:

- `fetchDatabases()` is not awaited before `onUrlChange()`.
- Excluding every database sends no `db` params, which the backend interprets as “all databases”.
- On first load `state.readOnly` is false until databases resolve, so delete controls can render.

Tasks:

- [ ] Await `fetchDatabases()` before calling `onUrlChange`/`performSearch`.
- [ ] When the user excludes every DB, render an explicit empty state and do not query.
- [ ] If a sentinel is needed, send `db=__none__` and have the backend return an empty result, or
      block the request client-side.
- [ ] Keep delete/mark-played controls hidden until `readOnly` is known.

## 5.5 Fix DU mode

Files: `web/src/main.ts`, `internal/commands/serve_handlers.go`

Current problems:

- `fetchDU` always sends `limit`, even when `filters.all` is true, while pagination is hidden.
- Custom DU sort fields fall through to the DU default.
- `reverse` is discarded for the default sort.

Tasks:

- [ ] Do not send `limit` when `filters.all` is true.
- [ ] Either implement the advertised DU sort fields in `sortDUFiles`/`SortFolders`, or remove
      them from the dropdown.
- [ ] Preserve `reverse` for the default DU sort.
- [ ] Show total counts separately from page counts.
- [ ] Add tests for all/limited, sort, reverse, and pagination.

## 5.6 Fix playback and progress

Files: `web/src/main.ts`

Current problems:

- Default video/audio rates are reset only when `!state.playback.item`, but the item is set before
  `openInPiP`; the reset never runs.
- Unplayed-filter refresh compares `prevItem` after the item has already been replaced.
- Document scroll percentage is stored in the same `pos` namespace as media seconds.
- Queue index and seek history are not cleared when switching media.
- Theatre mode resets on every track change.
- Modal close paths bypass `closeActivePlayer`.

Tasks:

- [ ] Apply default rates in `openInPiP` before assigning the new item, or pass the previous item
      explicitly.
- [ ] Capture `prevItem` before `openActivePlayer` replaces `state.playback.item`.
- [ ] Use a separate `documentProgress` field or unit; do not mix it with media `playhead`.
- [ ] Clear `queueIndex`, seek history, and marked position on item change.
- [ ] Preserve `playerMode` when changing tracks; reset only on explicit close.
- [ ] Route every modal close through a single `closeActivePlayer`/`closeModal` path.
- [ ] Add tests for rate selection, unplayed refresh, document progress, queue highlight, and
      theatre persistence.

## 5.7 Fix subtitles and captions

Files: `web/src/main.ts`, `web/index.html`

Current problems:

- `parseWebVTT` skips blank lines before cue finalization, so it returns only the last cue.
- The timestamp regex rejects valid `MM:SS.mmm` timestamps.
- `filter-captions` does not exist in `index.html`; the captions filter is always off.
- Captions details is routed to the generic details table.

Tasks:

- [ ] Rewrite the WebVTT parser as a state machine:
  - preserve blank-line cue termination,
  - support `HH:MM:SS.mmm` and `MM:SS.mmm`,
  - ignore `NOTE` blocks,
  - collect multi-line cue text.
- [ ] Add fixture tests for multi-cue, no trailing blank line, and `MM:SS` timestamps.
- [ ] Add the captions filter control or remove the state and fetch code.
- [ ] Handle captions details before the generic details branch, or make
      `renderCaptionsDetails` the details renderer for that mode.
- [ ] Add tests for `seekToSubtitleCue` across multiple cues.

## 5.8 Fix playlist and episode pagination

Files: `web/src/main.ts`

Current problems:

- Playlist items never update `state.totalCount`/`currentPage`, so pagination uses stale counts.
- Playlist search/filter controls are visible but ignored.
- Episodes/group view requests `limit=99` and always hides pagination.

Tasks:

- [ ] Set `state.totalCount` from the playlist response header/count.
- [ ] Reset `currentPage` when switching playlists.
- [ ] Disable or hide search/filter controls in playlist mode if not supported.
- [ ] Add pagination to episodes/group view, or remove the limit and render all groups with a
      clear count.
- [ ] Tests for playlist page 1/2 and group pagination.

## 5.9 Fix “Empty Bin”

Files: `web/src/main.ts`, `internal/commands/serve_extra_handlers.go`

Current problem: the UI sends the current visible page, but the button label implies the whole bin.

Tasks:

- [ ] Decide semantics:
  - “Empty visible items” = send current page,
  - “Empty bin” = send empty `paths`.
- [ ] Align the label and confirmation text with the chosen behavior.
- [ ] After deletion, refresh from page 1 and update counts.
- [ ] Add a test for both modes.

## 5.10 Fix thumbnail and media-type checks

Files: `web/src/main.ts`

Current problems:

- The DU inline `onerror` calls `generateClientThumbnail`, which is module-scoped and not on
  `window.disco`.
- Several checks use `item.type` instead of `media_type`.

Tasks:

- [ ] Expose the thumbnail helper on `window.disco` or attach the handler with
      `addEventListener` at render time.
- [ ] Replace all `item.type` checks with `media_type` (preload, playlist sort, captions details).
- [ ] Add tests for image preload and playlist Type sort.

## 5.11 Fix missing/dead UI modes

Files: `web/src/main.ts`, `web/index.html`, `web/src/types.ts`

Current problems:

- Document reader settings target nonexistent `#document-reader`/`.reader-content`.
- `trackReaderProgress`/`applyReaderProgress` are never called; `'remove'` is not a DOM event.
- Browse/genre controls are absent from `index.html`.
- `state.page='episodes'` is never set.
- History mode has no navigation button; mobile code references missing IDs.
- `curationBtn`, `isForwardNav`, `isBackwardNav`, `fetchFilterBins`, `renderFilterBins` are dead.

Tasks:

- [ ] For each item, either add the missing DOM/wiring or delete the code and state.
- [ ] Remove `state.filters.browseCol/browseVal` if the browse UI is not being built.
- [ ] Remove `state.filterBins.media_type` if unused.
- [ ] Add or remove the history button and `#filter-unplayed`.
- [ ] Add a dead-ID check to the frontend tests: every `getElementById` target used by a mode must
      exist in `index.html` or be created dynamically.

## 5.12 Clean listener lifecycle

Files: `web/src/main.ts`

Current problem: `setupViewerZoomPan` and `setupCursorAutoHide` add shared listeners on every
opened item and never remove them.

Tasks:

- [ ] Register listeners once at startup, or store and remove them in `closeActivePlayer`.
- [ ] Add a test that opening/closing N items does not increase listener count.
- [ ] Ensure fullscreen change listeners are also removed.

## 5.13 Remove debug logging and dead code

Files: `web/src/main.ts`

Tasks:

- [ ] Remove `console.log` from `showToast`, “Not an HTML folder”, HLS autoplay, and server-restart
      paths. Keep `console.error` only for real errors.
- [ ] Remove the dead functions listed in 5.11.
- [ ] Remove `state.filters.captions` if the UI is not added.
- [ ] Update `web/tests/advanced-features.test.js` to stop using
      `if (!element) expect(true).toBe(true)`. Tests must fail when a control is missing.

## Phase exit criteria

- `/`, `./`, backslash input, and normal search all work from one handler.
- RSVP opens the video player.
- Back/forward walks mode changes, not every keystroke.
- Excluding all DBs shows an empty state.
- Every visible filter/sort control has an effect.
- WebVTT exposes all cues.
- `npm test --prefix web` and `npm run build --prefix web` pass.
