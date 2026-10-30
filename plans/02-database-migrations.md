# Phase 2: Database Migrations, Schema Drift, and FTS Lifecycle

Goal: migrations preserve child rows, the production schema is the only schema, and FTS is
enabled consistently.

## 2.1 Add migration regression tests first

Files: new `internal/db/migrate_test.go`, `internal/db/schema/*.sql`, `internal/testutils/testutils.go`

Current problem: no test exercises `Migrate`, `cleanupMediaTable`, `convertHistoryMediaID`, or
`upgradeFTSTable`. The suite uses the legacy `schema_tables.sql`, so production schema changes are
largely untested.

Tasks:

- [ ] Add a test helper that builds a legacy database with:
  - `media` containing dead columns and old rowids,
  - `history`/`captions` with `media_id`,
  - `playlist_items` referencing `media.path`,
  - orphaned history/caption rows.
- [ ] Run `Migrate` and assert:
  - history/caption/playlist rows survive when their media survives,
  - orphan rows are handled explicitly,
  - `media` rowids are stable or the conversion never depends on them,
  - `media_fts` and `captions_fts` still map to the right rows.
- [ ] Run the same assertions against `InitDB` on a fresh file.
- [ ] Keep test DB creation on the production schema, not the legacy one.

## 2.2 Prevent cascade loss during media table recreation

Files: `internal/db/migrate.go`, `internal/db/connection.go`

Current problem: `recreateMediaTable` runs `DROP TABLE media` inside a transaction. `Connect`
sets `PRAGMA foreign_keys=ON`, and child tables use `ON DELETE CASCADE`, so the implicit delete
can remove history, captions, and playlist items.

Tasks:

- [ ] Acquire a single `*sql.Conn` for the migration.
- [ ] Disable foreign keys on that same connection before beginning the transaction, and re-enable
      them on the same connection afterward.
- [ ] Do not rely on `db.ExecContext("PRAGMA ...")` applying to the transaction connection.
- [ ] Alternatively, recreate without an implicit delete: copy child rows to temporary tables,
      recreate `media`, then restore and validate referential integrity.
- [ ] Rebuild the FTS tables after recreation.
- [ ] Assert `PRAGMA foreign_key_check` is clean after migration.
- [ ] Test the exact cascade case with a history row, a caption row, and a playlist item.

Acceptance:

- Migrating a legacy DB with child rows never loses those rows.
- A failed migration rolls back without leaving a half-recreated `media` table.

## 2.3 Convert legacy `media_id` before rowids can change

Files: `internal/db/migrate.go`

Current problem: `cleanupMediaTable` can recreate `media` before
`convertColumnsBeforeStrict`, but the history/caption converters join old `media_id` to
`m.rowid`.

Tasks:

- [ ] Reorder `Migrate` so `convertColumnsBeforeStrict` runs before any table recreation.
- [ ] Make the conversion join on a stable key (`path`) when possible.
- [ ] If `media_id` can no longer be resolved, preserve the row with a documented placeholder or
      move it to a quarantine table. Do not silently map it to an unrelated path.
- [ ] Add tests with non-contiguous rowids and a missing media row.

## 2.4 Make metadata consolidation schema-safe

Files: `internal/db/migrate.go`

Current problem: `needsCleanup` can return true for a STRICT-only migration or one dead column,
but `consolidateMetadata` always references `decade`, `mood`, `bpm`, and `key`.

Tasks:

- [ ] Inspect actual columns before building the `UPDATE`.
- [ ] Build the expression only from columns that exist.
- [ ] If there is no metadata to consolidate, skip the statement.
- [ ] Add tests for:
  - only `uploader` present,
  - only `webpath` present,
  - STRICT change with no dead columns,
  - all four metadata columns present.

## 2.5 Resolve the two schema sources

Files: `internal/db/schema/*.sql`, `internal/db/schema_tables.sql`,
`internal/db/schema_triggers.sql`, `internal/db/schema_fts.sql`,
`internal/testutils/testutils.go`, `internal/db/queries_test.go`

Current problems:

- Production `InitDB` uses `internal/db/schema`.
- Tests use the legacy root SQL files.
- Production `media` uses `id INTEGER PRIMARY KEY AUTOINCREMENT, path UNIQUE`; the legacy
  migration path creates `path TEXT PRIMARY KEY`.
- Production `playlists` lacks `time_created`, `time_modified`, and `hours_update_delay`;
  `migrateToStrict` returns early for already-STRICT tables, so they are never added.

Tasks:

- [ ] Choose `internal/db/schema` as the single source of truth.
- [ ] Delete or archive the root legacy schema files once tests use the production schema.
- [ ] Decide the canonical media primary key:
  - keep `id` and `path UNIQUE`, or
  - use `path` as the primary key.
- [ ] If keeping `id`, update `migrate.go` and all FTS/rowid assumptions to use it consistently.
- [ ] Add a migration that adds the missing playlist columns to existing production DBs.
- [ ] Add a test that compares `PRAGMA table_info` for a fresh `InitDB` database and a migrated
      legacy database. They should match the chosen schema.
- [ ] Remove the dead `GetSchema`/`SchemaFS` wrappers after the migration is complete (phase 6).

Acceptance:

- A fresh DB and a fully migrated DB have the same table columns and constraints.
- Tests no longer rely on a second schema definition.

## 2.6 Make FTS enabled by default

Files: `internal/db/config.go`, `internal/commands/add.go`, `internal/commands/serve.go`,
`internal/db/init_fts5.go`, `internal/db/migrate.go`

Current problem: `InitFtsConfig` stores `false` despite the comment saying true. Only `add` and
`serve` call `SetFtsEnabled(true)`. Other `ConnectWithInit` callers create DBs without captions/FTS
and can drop `path_tokenized` during cleanup.

Tasks:

- [ ] Change `InitFtsConfig` to store `true`.
- [ ] Remove the now-redundant `SetFtsEnabled(true)` calls from `add` and `serve`.
- [ ] Keep the compile-time `FtsEnabled` guard; the unsupported no-`fts5` build should still fail
      clearly.
- [ ] Make `searchcaptions` fail with a clear message when the captions table is absent, instead
      of a raw SQL error.
- [ ] Add tests that a fresh DB from each `ConnectWithInit` command contains:
  - `captions`,
  - `media_fts`,
  - `captions_fts`,
  - `path_tokenized`.
- [ ] Verify `--no-fts` still changes query behavior without changing schema creation.

## 2.7 Fix FTS upgrade and rebuild behavior

Files: `internal/db/migrate.go`, `internal/db/maintenance.go`, `internal/db/repair.go`

Current problems:

- `upgradeFTSTable` only checks for the substrings `trigram` and `description`; an old table can
  pass and then triggers reference missing columns.
- `CREATE TRIGGER IF NOT EXISTS` silently keeps outdated trigger bodies.
- `RebuildFTS` rebuilds only `media_fts`, not `captions_fts`.
- `captions_fts` has no UPDATE trigger.
- `rebuildFTSTable` drops a table before confirming it can recreate it.

Tasks:

- [ ] Compare the full expected FTS schema (columns, tokenizer, `detail`) rather than substrings.
- [ ] Drop and recreate triggers when their SQL differs from the expected SQL.
- [ ] Rebuild both `media_fts` and `captions_fts` after media recreation.
- [ ] Add a `captions_fts` UPDATE trigger or explicitly document that captions are immutable.
- [ ] Make `rebuildFTSTable` validate the create SQL before dropping.
- [ ] Add tests for upgrading from:
  - no trigram,
  - trigram with missing `path_tokenized`,
  - trigram with `detail=none`,
  - outdated trigger bodies.

## 2.8 Replace or remove folder-stats dirty-bit machinery

Files: `internal/db/migrate.go`, `internal/db/maintenance.go`, `internal/commands/serve_filters.go`

Current problems:

- Triggers set `folder_stats_meta.dirty`, but nothing reads or clears it.
- `PopulateFolderStatsInGo` only inserts/replaces current folders; it does not delete stale rows.
- `Migrate` runs a full folder-stats scan and per-folder writes on nearly every CLI invocation.
- `folder_stats.depth` is written but never read.

Tasks:

- [ ] Choose one design:
  - **Dirty-bit design:** `NeedsRefresh` reads `dirty`; `RefreshFolderStats` clears it; writes happen
    only when dirty.
  - **Timestamp design:** remove the dirty table and keep interval-based refresh.
- [ ] If the dirty-bit design is chosen, reset `dirty = 0` after a successful refresh.
- [ ] Make population delete stale `parent` rows (or use `DELETE` + insert in a transaction).
- [ ] Stop running the full population during ordinary `Migrate`; run it only when the table is
      missing/empty or when maintenance is requested.
- [ ] Remove `depth` or use it in at least one query; do not keep an unused indexed column.
- [ ] Add tests for delete-all, add-folder, rename-folder, and no-op refresh.

## 2.9 Remove duplicate and redundant indexes

Files: `internal/db/schema/media_indexes.sql`, `internal/db/migrate.go`

Tasks:

- [ ] Remove either `idx_media_type` or `idx_type`; keep one.
- [ ] Remove `idx_path` if the `path` UNIQUE/PK index already covers it.
- [ ] Confirm no query relies on the removed index names in `EXPLAIN QUERY PLAN`.
- [ ] Add a test that lists indexes and asserts there are no duplicate column sets.

## 2.10 Fix the repair lock and Windows recovery path

Files: `internal/db/repair.go`, `internal/db/repair_windows.go`

Current problems:

- `acquireCrossProcessLock` takes `LOCK_EX` and immediately releases `LOCK_UN`.
- The deferred cleanup unlinks the lock file.
- Windows skips cross-process locking and uses POSIX shell quoting with `cmd /c`.

Tasks:

- [ ] Hold the flock until `Repair` finishes; release it in a deferred function after all work.
- [ ] Do not remove the lock file before unlocking. Prefer leaving a stable lock file.
- [ ] On Windows, implement a real lock (LockFileEx or a named mutex) or fail closed with a clear
      message when another repair is active.
- [ ] Use Windows-native quoting for `cmd /c` or invoke sqlite3 with `exec.Command` arguments.
- [ ] Make backup directory names collision-resistant (nanoseconds + PID).
- [ ] Add a test that two concurrent repair attempts cannot both rename the DB.

## 2.11 Decide the legacy pre-STRICT path

Files: `internal/db/migrate.go`, `internal/db/init_fts5.go`

Current problem: `migrateLegacyTables` only creates `custom_keywords`, and `InitDB` runs STRICT SQL
before the fallback can help.

Tasks:

- [ ] Either declare SQLite >= 3.37 required and delete the unreachable path, or
- [ ] Create the core tables without STRICT first when the version is too old, then migrate.
- [ ] Document the minimum SQLite version in `README.md`.
- [ ] Add a version-gated test if the dual path remains.

## Phase exit criteria

- Migration tests cover child-row preservation and legacy `media_id` conversion.
- A fresh DB and a migrated DB have matching schemas.
- FTS is enabled for every `ConnectWithInit` caller.
- `PRAGMA foreign_key_check` is clean after migration.
- Repair cannot run twice concurrently.
