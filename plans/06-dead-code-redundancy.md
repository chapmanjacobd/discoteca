# Phase 6: Dead Code and Redundant Implementations

Goal: remove production-unreachable code and collapse duplicate implementations so future fixes
have one place to land.

Baseline: `deadcode -tags fts5 ./...` reports 186 unreachable functions. This phase does not need
to reach zero immediately, but it should remove the large superseded subsystems and establish a
ratchet so new dead code does not accumulate.

## 6.1 Establish a dead-code baseline

Files: `Makefile`, `.github/workflows/ci.yml` (or the current CI config)

Tasks:

- [ ] Add a `deadcode` make target.
- [ ] Capture the current output as an allowlist file (`plans/deadcode-allowlist.txt` or
      `scripts/deadcode-allowlist.txt`).
- [ ] Add a CI check that fails when a new unreachable function appears outside the allowlist.
- [ ] Keep test helpers in `internal/testutils` exempt from the ratchet.
- [ ] Remove entries from the allowlist as they are deleted.

## 6.2 Delete superseded sorting/folder implementations

Files: `internal/sort/`, `internal/aggregate/aggregate.go`, tests

Tasks:

- [ ] Delete the `internal/sort` package entirely. It has no production callers and duplicates
      `SortBuilder`.
- [ ] Delete `aggregate.FolderStats`, `aggregate.ByFolder`, and `aggregate.SortFolders`.
- [ ] Update `internal/aggregate/aggregate_test.go` to remove tests for deleted code, keeping
      tests for the clustering functions that remain.
- [ ] Confirm `query.AggregateMedia`/`query.SortFolders` are the only folder aggregation path.
- [ ] Remove the duplicate `!less` reverse logic with the deleted code.

## 6.3 Delete unfiltered DU implementations

Files: `internal/query/aggregation.go`, tests

Tasks:

- [ ] Delete `AggregateDUByPath`, `AggregateDUByPathMultiDB`, and `FetchDUDirectFiles` if they
      remain unused after phase 3.
- [ ] Keep only the `...WithFilters` variants.
- [ ] Factor the repeated path-extraction SQL into one helper so it exists in one place.
- [ ] Add a test that the filtered variant is used by the server and by any CLI caller.

## 6.4 Delete dead schema/query wrappers

Files: `internal/commands/commands.go`, `internal/db/schema.go`, `internal/db/schema/schema.go`,
`internal/testutils/testutils.go`

Tasks:

- [ ] Delete `commands.SchemaFS`, `commands.GetSchema`, `db.SchemaFS`, `db.GetSchemaTables`,
      `db.GetSchemaTriggers`, and `db.GetSchemaFTS` once phase 2 makes the production schema the
      only schema.
- [ ] Delete `schema.GetSchema` unless a test needs it; if needed, move it to `testutils`.
- [ ] Remove the dead `db.SetLogger` if the slog logger is the only path.
- [ ] Remove dead generated row types in `internal/db/models.go` after confirming no query uses
      them.

## 6.5 Delete dead history and FTS APIs

Files: `internal/history/history.go`, `internal/db/fts_queries.go`

Tasks:

- [ ] Remove `history.Tracker`, `NewTracker`, `Tracker.UpdatePlayback`, and
      `Tracker.MarkDeleted`; production uses `UpdateHistorySimple`/`UpdateHistoryWithTime`.
- [ ] Remove `db.SearchMediaFTS` and `db.RankSearchResults` if they remain test-only. Move any
      still-useful ranking test to the production SQL path.
- [ ] Remove test-only media helpers that duplicate query scanner logic.

## 6.6 Delete duplicate migration helper

Files: `internal/db/micro_migrate.go`, `internal/commands/dedupe.go`

Tasks:

- [ ] Delete `micro_migrate.go` and its `EnsureColumns`/`EnsureIndexes` usage once `Migrate`
      guarantees `is_deduped` and its indexes.
- [ ] Keep one migration mechanism. Do not maintain a second ad-hoc schema path.

## 6.7 Collapse duplicate CLI wrappers

Files: `internal/commands/bigdirs.go`, `internal/commands/diskusage.go`,
`internal/commands/print.go`, `internal/commands/watch.go`

Tasks:

- [ ] Make `bigdirs` a thin alias for `print --big-dirs` or remove it.
- [ ] Make `du` call the shared print/aggregate path after phase 4; do not duplicate query or scan
      code.
- [ ] Consolidate `watch` and `listen` shared logic into helpers:
  - query flags/media query,
  - player argument building,
  - history update,
  - post-action handling.
- [ ] Preserve intentional differences (`--save-playhead`, fullscreen, player args) as explicit
      options, not silent divergence.
- [ ] Remove duplicate `buildFlags`/`queryMedia`/`updateHistory` code.

## 6.8 Delete duplicate server helpers

Files: `internal/commands/serve_epub.go`, `internal/commands/serve.go`,
`internal/commands/serve_queries.go`

Tasks:

- [ ] Collapse `serveHTMLWithTOC` and `serveHTMLFolderWithTOC` into one function.
- [ ] Remove the empty `/favicon.ico` handler or serve the real favicon.
- [ ] Remove `LsEntry.Local` if nothing sets or reads it.
- [ ] Either wire the slow-query dashboard to real query timing or remove `/api/queries`,
      `TimedQuery`, `RecordSlowQuery`, and the query-stats types.
- [ ] Remove dead/unreachable branches in `parseCategoryFlags`, `parseStatusFlags`,
      `filterDatabases`, and `processLsEmptySearchPath` after verifying they cannot run.
- [ ] Remove `Mux`'s hardcoded `"test-token"` fallback; tests should set a token explicitly.

## 6.9 Delete dead utility surface

Files: `internal/utils/` and `internal/utils/pathutil/`

The inventory is large. Remove in batches, with `deadcode` after each batch:

- [ ] `utils/db.go`: `MostSimilarSchema`, `BlockDictsLikeSQL`, `AllowDictsLikeSQL`,
      `ConstructSearchBindings`.
- [ ] `utils/processes.go`: `Cmd`, `CmdInteractive`, `CmdWithTimeout`, `FzfSelect`,
      `AdjustDuration`, `SizeTimeout` (keep `CmdDetach` if used).
- [ ] `utils/path_utils.go`: `RandomFilename`, `TrimPathSegments`, `SafeJoin`, `Relativize`,
      `StripMountSyntax`, `IsEmptyFolder`, `FolderSize`, `PathTupleFromURL`, `CleanPath`.
- [ ] `utils/iterables.go`, `utils/strings.go`, `utils/nums.go`: remove the long tail of unused
      Python-parity helpers. Do this file by file; do not delete a function only because one grep
      missed a dynamic reference.
- [ ] `utils/formatting.go`, `utils/time.go`, `utils/printing.go`, `utils/files.go`: remove
      `RelativeDatetime`, `FormatPlaybackDuration`, `IsTZAware`, `TubeDate`,
      `UtcFromLocalTimestamp`, `ColNaturalDate`, `ColFilesize`, `ColDuration`,
      `FilterDeleted`, `DetectMimeType`, `GetMountPoint`, `MoveFile`, `Rename`, `Unlink`,
      `Rmtree`, `AltName` if still unused.
- [ ] `utils/mpv.go`: remove unused wrappers (`MpvSetProperty`, `MpvGetProperty`, `MpvPause`,
      `MpvSeek`, `MpvLoadFile`, `MpvArgsToMap`) or use them in production.
- [ ] `utils/terminal_common.go`, `utils/terminal_unix.go`: remove unused command/terminal helpers.
- [ ] `utils/pathutil`: remove `Depth`, `Parent`, `EnsureTrailingSep`, `HasTrailingSep`,
      `HasLeadingSep`, `StripTrailingSep`, `ToURL` if still unused. Keep `FromURL` only if a
      production caller remains.
- [ ] Remove unused constants in `utils/consts.go`.

Guardrail:

- [ ] For platform-split files (`atime_*`, `is_file_open_*`, `repair_*`), verify all build tags
      before deleting. A function may be reachable only on another OS.
- [ ] Do not remove code used by tests unless the test is also removed.
- [ ] After each batch, run `go test -tags fts5 ./...` and `deadcode -tags fts5 ./...`.

## 6.10 Remove dead frontend helpers

Deferred to phase 5 for behavior. This phase only deletes what phase 5 explicitly marks dead:

- [ ] `fetchFilterBins`, `renderFilterBins`.
- [ ] `trackReaderProgress`, `applyReaderProgress`.
- [ ] `curationBtn`, `isForwardNav`, `isBackwardNav`.
- [ ] `state.filters.browseCol/browseVal`, `state.filterBins.media_type`.
- [ ] Unreachable `episodes` page support if not implemented.
- [ ] Stale test selectors in `web/tests`.

## 6.11 Do not delete security-relevant code early

`internal/commands/zim.go` process management is handled in phase 7. Do not delete it here, even
though parts become dead once the external-server design lands. Keeping it until phase 7 makes the
security diff reviewable on its own.

## Phase exit criteria

- `internal/sort` is gone.
- `aggregate.ByFolder`/`SortFolders` are gone.
- Unfiltered DU implementations are gone.
- Only one schema source and one migration mechanism remain.
- `watch`/`listen` share helpers.
- `deadcode` output is at or below the allowlist, and the allowlist only shrinks.
- `go test -tags fts5 ./...` still passes.
