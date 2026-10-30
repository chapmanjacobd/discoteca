# Phase 4: CLI Feature Wiring and Flag Cleanup

Goal: a flag or positional argument either works as documented or is removed. No command should
accept a filter, sort, playback option, or post-action and silently ignore it.

## 4.1 Choose “implement” vs “remove” per flag

For each item below, record the decision in the command help text and in this plan. Default
principle: remove the flag if the feature is not part of the command’s job; implement it if it is.

| Command | Current no-op | Decision |
|---|---|---|
| `print` | positional files/directories (`ScanPaths`) | Implement via a shared scan helper, or reject non-`.db` args |
| `du` | filesystem probe results are discarded | Remove the duplicate walk; delegate to the database path |
| `check` | path/media filters | Implement file filtering before mark-deleted |
| `dedupe` | path/search/time/deleted filters | Implement in SQL candidate selection, or remove embedded flag structs |
| `dedupe` | most post-action flags | Support only trash/delete-cmd; remove the rest |
| `add` | search/duration/modified/created/downloaded/where filters | Implement, or reduce the embedded flag set to size/ext/path |
| `stats` | filter flags; `facet` without `--frequency` | Apply filters; require `--frequency` for facet output |
| `history` | `--limit/--offset/--all`; explicit `-u path` | Pass `QueryFlags`; make default sort explicit |
| `regexsort` | `SortFlags` | Implement or remove sort flags |
| `similar-*` | cluster/TFIDF/move/sort flags | Implement or remove |
| `mediacheck` | `AudioScan` | Remove until implemented |
| `searchcaptions` | playback flags | Use them in `playCaptions` or remove |
| `merge-dbs` | `--where` on non-media tables; `--simulate` | Apply only to compatible tables; implement simulate |
| `optimize` | `--simulate`; bulk-mark placeholder | Implement or remove the bulk-mark step |
| `playlists` | JSON only first DB | Fix loop |
| `readme` | invalid search examples | Correct examples |
| all | `--timeout` | Implement via context timeout or remove |
| all | `--threads`, `--ignore-errors` | Expose on commands that use them, or remove from `GlobalFlags` |
| all `SortFlags` users | `--re-rank` | Wire through `SortBuilder` or remove |
| most | `--frequency` | Only expose on commands that implement it |
| most | `--columns` | Validate column names and error on unknown values |

## 4.2 Print and DiskUsage

Files: `internal/commands/print.go`, `internal/commands/diskusage.go`,
`internal/commands/bigdirs.go`, `internal/commands/filesinfo.go`

Tasks:

- [ ] Extract one filesystem-scan helper from `filesinfo`/`diskusage` that walks a path and calls
      `metadata.Extract`.
- [ ] Use it in `print` when `ScanPaths` is non-empty.
- [ ] Remove the duplicate walk in `diskusage`; either pass scanned media into `PrintCmd` or query
      the DB once.
- [ ] If direct scanning is not intended, change the argument to `type:"existingfile"` for `.db`
      only and return a clear error for directories.
- [ ] Keep `bigdirs` as a thin alias for `print --big-dirs`; remove duplicated flag plumbing.
- [ ] Tests: `print file.mkv`, `print dir/`, `du dir/`, empty scan path, nonexistent path.

## 4.3 Check filters

Files: `internal/commands/check.go`, `internal/query/filter_builder.go`

Tasks:

- [ ] Apply `PathFilterFlags` and `MediaFilterFlags` to the DB rows before the presence check.
- [ ] Do not scan the whole DB when filters exclude most rows.
- [ ] Make the path filters apply to scan roots as well, or document that they only apply to DB
      rows.
- [ ] Add a test where only one of two media types is marked deleted.

## 4.4 Dedupe filters and post-actions

Files: `internal/commands/dedupe.go`, `internal/models/flags.go`

Tasks:

- [ ] Decide which of `PathFilter`, `Filter`, `MediaFilter`, `TimeFilter`, and `Deleted` make sense
      for each dedupe mode.
- [ ] Implement them in the candidate SQL (preferred) or remove the embedded structs.
- [ ] Remove `PostActionFlags` if only `--trash` is supported. Keep a narrow `--trash` field.
- [ ] Make `--extractor-id/--id` match the schema. Either add `extractor_id`, use an existing
      column, or remove the mode.
- [ ] Update help text so every mode names the columns it groups by.
- [ ] Tests: excluded path, video-only, time range, and `--only-deleted` behavior.

## 4.5 Add filters

Files: `internal/commands/add.go`

Tasks:

- [ ] Either wire the embedded `FilterFlags` into `matchesFilters`, or replace the embedded struct
      with a small `AddFilterFlags` containing only path/size/ext.
- [ ] If wiring, implement search, duration, modified/created/downloaded, watched, where, and
      captions consistently with `FilterBuilder`.
- [ ] Add tests for each supported filter.
- [ ] Keep dry-run behavior from phase 1.

## 4.6 Stats filters and facet

Files: `internal/commands/stats.go`, `internal/query/query.go`

Tasks:

- [ ] Apply filter flags to `GetStats`/`GetStatsByType`, or remove the embedded filter structs.
- [ ] Require `--frequency` when `facet` is used, or make `facet` select a real non-frequency
      statistic.
- [ ] Reject invalid facets instead of defaulting to `time_last_played`.
- [ ] Add tests for each facet and for a filtered total.

## 4.7 History limit and sort

Files: `internal/commands/history.go`, `internal/query/query.go`

Tasks:

- [ ] Pass `c.QueryFlags` instead of `models.QueryFlags{}`.
- [ ] Add an explicit `sort_explicit` signal so `history -u path` is not overwritten by the
      default rewrite.
- [ ] Fix `SortHistory` to return a new slice or mutate in place. Current `partial == "s"` path
      reassigns the local parameter and loses the filter.
- [ ] Add tests for `--limit`, `--offset`, `--all`, `-u path`, and `--partial s`.

## 4.8 RegexSort

Files: `internal/commands/regexsort.go`, `internal/utils/text_processor.go`

Tasks:

- [ ] Remove `SortFlags` if `TextProcessor` cannot support them; otherwise teach the processor
      about sort fields.
- [ ] Replace `map[sentence]line` with a slice plus a seen counter, matching
      `query.RegexSortMedia`. Duplicate normalized lines must not drop input.
- [ ] Fix default `LineSorts`: either implement `-allunique`, `alldup`, and `dupmode`, or remove
      them from the default list.
- [ ] Fix `-index` to precompute positions rather than call `indexOf` on a slice being sorted.
- [ ] Tests: duplicate lines, no-input, `--preprocess`, and each sort strategy.

## 4.9 Similarity and cluster commands

Files: `internal/commands/similar.go`, `internal/commands/clustersort.go`,
`internal/models/flags.go`

Tasks:

- [ ] Implement `ClusterSort`, `TFIDF`, and `MoveGroups`, or remove them.
- [ ] Either sort `similar-*` output using `SortFlags`, or remove sort flags from those commands.
- [ ] Make `PrintGroups` behavior consistent between `similar-*` and `clustersort`.
- [ ] Replace the `TotalSizes` reuse in `ClusterFoldersByName` with a dedicated
      `NameMatchMode`/`UseFullPath` flag.
- [ ] Fix `FilterNearDuplicates` to compare basenames when the feature is “near-duplicate names”.
- [ ] Tests: each flag has an observable effect.

## 4.10 MediaCheck and SearchCaptions

Files: `internal/commands/mediacheck.go`, `internal/commands/searchcaptions.go`

Tasks:

- [ ] Remove `AudioScan` or implement an audio-only corruption scan.
- [ ] Fix threshold help text from phase 1.
- [ ] In `searchcaptions`, either pass `PlaybackFlags` into `playCaptions` or remove them from the
      command struct.
- [ ] Use `--offset` or remove it.
- [ ] Tests: `--override-player`, `--volume`, `--start/--end`, and simulate where applicable.

## 4.11 MergeDBs

Files: `internal/commands/mergedbs.go`

Tasks:

- [ ] Apply `--where` only to tables whose columns are known to support it.
- [ ] Add a per-table `--where-table TABLE=CLAUSE` or a `media`-only `--where`. Do not silently
      fail and continue.
- [ ] Return a non-zero error if any requested table fails to merge.
- [ ] Implement `--simulate` (report table/row counts, no writes).
- [ ] Fix the upsert fallback: if `--upsert` cannot build an upsert query, fail before copying
      rather than falling back to a conflict-erroring `INSERT`.
- [ ] Tests: where on media, where on playlists, failed table, upsert with skipped PK, simulate.

## 4.12 Optimize

Files: `internal/commands/optimize.go`

Tasks:

- [ ] Add `--simulate`; in simulate mode print the planned VACUUM/ANALYZE/FTS operations.
- [ ] Rename `BulkMarkOptimizedExtensions`: it backfills `media_type`, not extensions.
- [ ] Either implement the backfill correctly or remove it from `optimize`.
- [ ] Remove the unused `*db.Queries` parameter if it stays.
- [ ] Tests: simulate changes nothing; backfill fills only NULL media types.

## 4.13 Playlists JSON and Readme examples

Files: `internal/commands/playlists.go`, `internal/commands/readme.go`

Tasks:

- [ ] Accumulate playlists from every database, then print JSON once.
- [ ] Correct the `search` examples to use the actual positional signature.
- [ ] Either make `readme` write `README.md` or change its help text to “print README content”.
- [ ] Tests: two DBs in JSON mode, and generated examples parse with Kong.

## 4.14 Global flags

Files: `internal/models/flags.go`

Tasks:

- [ ] `Timeout`: implement a context deadline in commands that embed `CoreFlags`, or remove it.
- [ ] `Threads`/`IgnoreErrors`: these live on `GlobalFlags`, which is never embedded by a command.
      Move them to a command struct that uses them, or remove them.
- [ ] `DryRun`/`Yes` aliases: expose them or remove the comments claiming they are aliases.
- [ ] Ensure every command that exposes `--simulate` actually honors it.
- [ ] Add a test that iterates over registered commands and fails if a documented destructive
      flag has no corresponding read in the command package. (A simple grep-based test is fine.)

## Phase exit criteria

- Every row in the decision table is implemented or removed.
- No command accepts a filter/sort option that it cannot apply.
- `--help` for each command matches actual behavior.
- Tests cover each newly wired flag.
