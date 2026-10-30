# Phase 3: Query Scanning, Sorting, Pagination, and Aggregation

Goal: the same flags produce the same ordering and counts regardless of command, database count,
or whether the result came from SQL or in-memory post-processing.

## 3.1 Stop dropping sortable columns while scanning

Files: `internal/query/filter_builder.go`, `internal/query/query_test.go`

Current problem: `mapInt64Field` does not map `video_count`, `audio_count`, `subtitle_count`,
`fps`, or `is_deduped`; `mapStringField` does not map `path_tokenized`, `fasthash`, or `sha256`.
`MediaQuery` selects `*`, so those fields are always zero/nil in memory. The xklb default sort on
`video_count/audio_count/subtitle_count` therefore sorts constants.

Tasks:

- [ ] Add all missing fields to `mapInt64Field`, `mapFloat64Field`, and `mapStringField`.
- [ ] Add a table-driven test that scans a row with every media column set and asserts every
      `models.Media` field is populated.
- [ ] Add a sort test that proves `video_count desc,audio_count desc,subtitle_count desc` reorders
      rows using real values.
- [ ] Verify `FetchSiblings` and any explicit column list still populate the same fields.

Acceptance:

- No documented sort key resolves to a zero/nil value solely because the scanner dropped it.

## 3.2 Define one sort contract

Files: `internal/query/filter_builder.go`, `internal/query/query.go`,
`internal/models/flags.go`, `internal/sort/sort.go` (deleted in phase 6)

Current problems:

- `SortBy` defaults to `"path"`, but `SortBuilder` treats `"path"` as “not explicitly set” and
  falls back to xklb.
- `--reverse`/`--natsort` take a different branch, changing the keys instead of reversing them.
- Single-DB queries use SQL ordering; multi-DB queries also run in-memory sorting.
- Unsupported sort fields silently fall back to path.
- Reverse comparators use `!less`, which is not a strict weak ordering.

Tasks:

- [ ] Decide and document the sort contract:
  - `default` = xklb for media, DU order for folders,
  - `path` = literal path order,
  - every other supported key = that key.
- [ ] Change the default `SortBy` from `"path"` to `"default"` if possible. If this is too
      breaking, add a `SortBySet`/`sort_explicit` boolean populated by Kong or the server parser.
- [ ] Make `--reverse` reverse the chosen comparator, not the key selection.
- [ ] Replace every `!less(i, j)` reverse comparator with `less(j, i)` (or `sort.Reverse` on a
      proper `Less`).
- [ ] Make unsupported sort fields return an error or log a warning; do not silently sort by path.
- [ ] Apply the same in-memory sort for one DB and multiple DBs.
- [ ] Add tests for:
  - one DB vs two DBs with identical data,
  - `default`, `path`, `size`, `play_count`, `video_count`, `title`, and `duration`,
  - `--reverse` with ties,
  - `--natsort`,
  - unknown sort key.

## 3.3 Apply pagination after the final ordering

Files: `internal/query/filter_builder.go`

Current problem: SQL `LIMIT` is applied before the in-memory sort. Callers then sort only the
already-limited page, so `default`/re-rank/play-order can miss the true top-N.

Tasks:

- [ ] Separate “fetch enough rows” from “return the page”.
- [ ] If the requested sort is representable in SQL, keep the SQL path and apply `LIMIT/OFFSET`
      after `ORDER BY`.
- [ ] If the sort is in-memory only, fetch at least `Limit+Offset` rows (or all rows when `All`),
      sort, then slice.
- [ ] Apply the same rule to multi-DB queries.
- [ ] Emit `LIMIT -1` whenever `OFFSET > 0` and no positive `LIMIT` exists. Confirm with a test
      that `--all --offset N` no longer returns a SQL syntax error.
- [ ] Add tests for top-N with a conflicting SQL pre-sort and for `--all --offset`.

Acceptance:

- `ORDER BY` and the returned page agree.
- `OFFSET` without `LIMIT` works in both single-DB and multi-DB paths.

## 3.4 Validate percentile input

Files: `internal/query/filter_builder.go`, `internal/utils/nums.go`, tests

Current problem: `ParsePercentileRange` accepts any floats; indexing a 101-element mapping can
panic.

Tasks:

- [ ] Reject values outside `[0, 100]`.
- [ ] Reject `min > max`.
- [ ] Return an error from the resolver rather than silently leaving the percentile string in place.
- [ ] Do not drop per-DB percentile query errors in goroutines; aggregate and return them.
- [ ] Fix `cleanPercentileFlags` to clear `Offset`, or let it work with `LIMIT -1`.
- [ ] Add tests for `p200-300`, `p100-200`, `p50-10`, non-numeric input, and a DB query failure.

## 3.5 Centralize DU filter predicates

Files: `internal/query/aggregation.go`, `internal/commands/serve_handlers.go`, tests

Current problem: `hasBasicFilters` omits `Where`, `Regex`, `Paths`, `Exists`, `Portrait`,
`WithCaptions`, `DurationFromSize`, `DeletedAfter/Before`, and `PlayedAfter/Before`. If one of
those is the only filter, DU returns unfiltered data.

Tasks:

- [ ] Create one function that reports whether any filter is active.
- [ ] Use that function in the fast path, the slow path, and direct-file fetching.
- [ ] Ensure the final per-file aggregation re-checks every filter, not a subset.
- [ ] Make `FolderCounts` consistent:
  - decide whether it means files or subfolders,
  - use the same definition in SQL and in-memory,
  - update help text.
- [ ] Add table-driven tests: one test per filter as the sole filter, asserting DU excludes
      non-matching rows.

## 3.6 Make counts match the returned result set

Files: `internal/query/filter_builder.go`, `internal/commands/serve_handlers.go`

Current problem: `MediaQueryCount` is a plain `COUNT(*)`. It ignores `GroupByParent`,
`FetchSiblings`, and in-memory `FilterMedia`, so `X-Total-Count` can differ from the response.

Tasks:

- [ ] For grouped/sibling/related modes, compute the count from the same post-processed data used
      for the response.
- [ ] For plain modes, keep the SQL count but apply `FilterMedia`-equivalent predicates in SQL.
- [ ] Add tests for:
  - grouped results,
  - sibling expansion,
  - include/exclude path filters,
  - captions view,
  - DU total counts after pagination.

## 3.7 Fix filter semantics that currently collapse together

Files: `internal/query/filter_builder.go`, `internal/models/flags.go`, tests

Tasks:

- [ ] Define “finished” once (for example `playhead >= duration * 0.95` or an explicit `done`
      marker). Use it for both `Unfinished` and `InProgress`:
  - `Unfinished` = started but not finished,
  - `InProgress` = same if that is the intended UI, otherwise distinguish them explicitly.
- [ ] Change play-count bounds to support zero. Use `*int` flags or a separate `PlayCountMaxSet`.
- [ ] Make `--include` have one meaning. Preferred: SQL and in-memory both use path glob/prefix
      matching; do not also treat include as free-text search.
- [ ] Make `--no-fts` effectively disable the FTS join, not only the `MATCH` predicate.
- [ ] Order related-media FTS results by `bm25(...) ASC`.
- [ ] Fix `GroupByParent` to preserve each row’s actual `DB`.
- [ ] Fix `HistoricalUsage`:
  - do not add `time_deleted = 0` when the selected facet is `time_deleted`,
  - compute quarters without subtracting three months first,
  - reject unknown facets instead of silently defaulting.
- [ ] Make random sampling include all active filters, and do not force non-deleted rows when
      `HideDeleted=false`.
- [ ] Make `FetchSiblings` honor `FetchSiblingsMax` for `all`/`always`, sort output, and respect the
      deleted filter.
- [ ] Make `--re-rank` work in every command that embeds `SortFlags`, or remove the flag from
      commands that cannot support it.
- [ ] Define multi-group sort behavior. If groups are tie-breakers, apply them as a single
      comparator. If they are successive full sorts, document that and remove misleading comments.
- [ ] Add tests for each changed semantic.

## 3.8 Fix search-mode detection

Files: `internal/query/search_mode.go`, `internal/commands/search.go`

Current problems:

- `resolveSearchMode` always calls `DetectSearchMode(ctx, nil)`, so detection always selects
  substring.
- `GetSearchMode`/`IsSearchAvailable`/`ResetSearchModeDetection` are test-only.
- `search` checks only the first DB for FTS and then enables FTS for all DBs.

Tasks:

- [ ] Pass the actual `*sql.DB` to detection, or remove runtime detection and rely on explicit
      `--fts`/`--no-fts`.
- [ ] If detection remains, make it per query/per database, not a process-global `sync.Once`.
- [ ] In `search`, either require all DBs to have FTS or enable FTS per database.
- [ ] Remove the unused getters or use them in a real diagnostics path.
- [ ] Add tests for mixed FTS/non-FTS database lists.

## Phase exit criteria

- The same query with one and two databases returns the same order.
- `--sort-by path` sorts by path; `--reverse` reverses that order.
- `--all --offset` works.
- Invalid percentiles return an error, not a panic.
- DU honors every documented filter.
- `X-Total-Count` matches the response for grouped and sibling modes.
