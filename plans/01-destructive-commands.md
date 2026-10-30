# Phase 1: Destructive Commands and Silent Data Loss

Goal: every command that can delete, move, mark deleted, or write during a dry run either
performs exactly what it documents or fails before changing state.

## 1.1 Dedupe must honor simulation and report real failures

Files: `internal/commands/dedupe.go`, `internal/commands/dedupe_test.go`

Current problems:

- `DedupeCmd.Run` never reads `Simulate`/`DryRun`.
- `processDuplicates` calls `os.Remove` unconditionally and discards its error.
- The non-trash and `--dedupe-cmd` branches can mutate the filesystem during a dry run.
- `collectDuplicatesFromGroup` writes `is_deduped` during candidate collection, before confirmation.
- `updateDatabaseAfterDedupe` marks the row deleted even when removal failed.

Tasks:

- [ ] Add a shared `dryRun := c.Simulate || c.DryRun` check at the start of the mutating path.
- [ ] In dry-run mode, print each planned `keep`/`duplicate` action and return without:
  - calling `os.Remove`,
  - running `--dedupe-cmd`,
  - calling `utils.Trash`,
  - updating `media` rows.
- [ ] Move the `is_deduped` write out of `collectDuplicatesFromGroup`. Candidate collection must be read-only.
- [ ] Check every removal result. Only update the DB after the file is gone or trashed.
- [ ] If removal fails, keep the duplicate row active and report the error in the final summary.
- [ ] If `--dedupe-cmd` fails, do not mark the duplicate deleted.
- [ ] Make the `is_deduped` meaning explicit: “processed/keep candidate” vs “duplicate removed”. Rename the helper or add a comment at the write site and in the schema.
- [ ] Add tests:
  - `--simulate` leaves both files and all DB columns unchanged.
  - a failing remover leaves `time_deleted = 0`.
  - a successful remover sets `time_deleted` exactly once.
  - candidate collection with `--simulate` does not set `is_deduped`.
  - `--trash --simulate` does not invoke the trash command.

Acceptance:

- A dry run produces only output.
- A failed deletion cannot be reported as a successful dedupe.
- No `UPDATE media` occurs before the confirmation prompt.

## 1.2 Check must dry-run correctly and must not mass-delete on weak evidence

Files: `internal/commands/check.go`, `internal/commands/check_test.go`, `internal/models/flags.go`

Current problems:

- `CheckCmd.DryRun` shadows the embedded `CoreFlags.DryRun`; `--simulate` is ignored.
- Walk errors inside the callback are swallowed.
- An existing but empty scan root produces an empty presence set and marks every DB row under it deleted.
- Root membership uses `strings.HasPrefix` without a path separator boundary.

Tasks:

- [ ] Resolve the shadowing. Preferred: in `CheckCmd.AfterApply`, set `c.DryRun = c.DryRun || c.Simulate` after calling `CoreFlags.AfterApply`. Keep `--dry-run` as an alias.
- [ ] Alternatively, remove the outer field and expose `--dry-run` through a separate alias that maps to `CoreFlags.Simulate`.
- [ ] Make `buildPresenceSet` fail closed:
  - each check path must exist and be a directory,
  - a root that yields zero files is an error unless the caller explicitly passes an override,
  - callback errors are returned, not ignored.
- [ ] Replace `strings.HasPrefix(m.Path, root)` with a boundary-aware containment check (`filepath.Rel` or `root + string(os.PathSeparator)`).
- [ ] Do not cap the DB read at 1,000,000 rows. Stream rows or remove the magic limit.
- [ ] In dry-run mode, print missing paths and do not call `MarkDeleted`, `RefreshFolderStats`, or `RebuildFTS`.
- [ ] Add tests:
  - `--simulate` leaves all rows unchanged.
  - an empty but existing root returns an error and changes nothing.
  - `/media/foo` does not govern `/media/foobar`.
  - a walk permission error returns an error instead of deleting.
  - `--dry-run` and `--simulate` produce the same result.

Acceptance:

- A typo, empty mount, or unreadable directory cannot mark a library deleted.
- `check --simulate` is a true no-op.

## 1.3 SearchDB delete must require an explicit predicate

Files: `internal/commands/searchdb.go`, `internal/commands/searchdb_test.go`

Current problems:

- `FilterFlags.Where` is ignored.
- If the table has no TEXT columns, `whereClauses` is empty and the command runs `DELETE FROM <table>`.
- `--simulate` is ignored for delete and mark-deleted.
- `SearchDBCmd.Search` shadows the embedded `FilterFlags.Search`.

Tasks:

- [ ] Rename the positional field to `Terms` or add a separate Kong name so `--search` is not shadowed.
- [ ] Include `FilterFlags.Where` in the built predicate.
- [ ] Refuse `--delete-rows` and `--mark-deleted` when the final predicate is empty. Require an explicit `--all-rows` flag for a table-wide action.
- [ ] Implement `--simulate` for both actions. Print the predicate and affected row count without executing.
- [ ] Run delete in a transaction and report the real affected count.
- [ ] Add tests:
  - `--where` limits the deleted rows.
  - no searchable columns plus no predicate refuses to delete.
  - `--simulate` leaves rows intact.
  - `--search` is actually accepted and applied.

Acceptance:

- No path can execute an unqualified `DELETE FROM` without an explicit opt-in.
- A user-provided safety predicate is honored.

## 1.4 Add must not write during simulation

Files: `internal/commands/add.go`, `internal/commands/add_test.go`

Current problems:

- The scan-root playlist insert happens before the simulate check.
- `RebuildFTS` and `RefreshFolderStats` run regardless.

Tasks:

- [ ] Move the playlist insert after the simulate decision, or guard it with `if !c.Simulate`.
- [ ] Skip `RebuildFTS` and `RefreshFolderStats` when simulating.
- [ ] Keep the existing “would process N new files” output, but make it the only side effect.
- [ ] Add a test that creates a temp DB, runs add with `--simulate`, and verifies:
  - no playlists row,
  - no media row,
  - no FTS/folder-stats writes.

Acceptance:

- `add --simulate` changes no database state.

## 1.5 MediaCheck must distinguish “scan failed” from “measured corruption”

Files: `internal/metadata/media_check.go`, `internal/commands/mediacheck.go`, tests

Current problems:

- `DecodeFullScan` returns `0.5, err` for all failures.
- `checkSingleMedia` overwrites failures with `0.5`.
- A threshold at 50% can therefore delete a valid file because ffprobe failed.

Tasks:

- [ ] Return `(0, err)` or a separate sentinel/typed error for scan failures.
- [ ] Do not overwrite the error with a numeric corruption value in `checkSingleMedia`.
- [ ] Only call `handleCorruptFile` when the scan succeeded and the measured value crosses the threshold.
- [ ] Report skipped files in the summary.
- [ ] Fix the help text for `--delete-corrupt`/`--full-scan-if-corrupt`: values are ratios, not seconds.
- [ ] Either implement `AudioScan` or remove it. Preferred: remove it until there is a real audio-only check.
- [ ] Add tests for ffprobe-not-found, invalid JSON, zero streams, and a genuine above-threshold result.

Acceptance:

- A tool failure can never trigger deletion.
- The flag help matches the comparison logic.

## 1.6 Post-actions must not report false success

Files: `internal/commands/actions.go`, tests

Current problems:

- Unknown `--post-action` strings silently count as success.
- `CopyMediaItem` loads the entire file into memory.
- `MoveMediaItem` uses `os.Rename` without a cross-filesystem fallback.

Tasks:

- [ ] Add a `default` case that returns an error for unknown actions.
- [ ] Return an error when the action is non-empty but unrecognized.
- [ ] Replace `os.ReadFile`/`os.WriteFile` in `CopyMediaItem` with `io.Copy` and a streaming source/destination.
- [ ] Add a cross-device fallback for `MoveMediaItem`, preserving the existing `utils.MoveFile` behavior if it is reliable.
- [ ] After a move, update `path` and `path_tokenized`, and rebuild/update FTS.
- [ ] Guard the post-action loop with `flags.Simulate`: print plans, change nothing.
- [ ] Add tests for unknown action, simulated delete/move/copy, and a copy of a file large enough that streaming is observable.

Acceptance:

- The command cannot print “<action> N files” when no action happened.
- Post-actions honor `--simulate`.

## 1.7 History timestamps must not move backwards

Files: `internal/history/history.go`, `internal/db/queries.go`, tests

Current problem:

- The comment claims `time_last_played` is only updated if newer; the SQL overwrites unconditionally.

Tasks:

- [ ] Change `UpdatePlayHistory` to use `MAX(time_last_played, ?)` or a conditional update.
- [ ] Do the same for `time_first_played` with `MIN`/`COALESCE` so imports cannot move it forward.
- [ ] Preserve explicit “rewrite this timestamp” behavior if a caller needs it; add a separate method rather than weakening the normal path.
- [ ] Add tests for older, equal, and newer timestamps, including the mpv watch-later import path.

Acceptance:

- Importing an old session cannot regress a media item’s last-played timestamp.

## Phase exit criteria

- `dedupe`, `check`, `searchdb`, `add`, `mediacheck`, and post-actions all have dry-run tests.
- No destructive command discards the result of its destructive operation.
- `go test -tags fts5 -race ./internal/commands ./internal/history` passes.
