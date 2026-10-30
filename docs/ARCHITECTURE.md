# Architecture

discoteca is organized by domain. Packages point inward: the CLI and HTTP
server depend on domain packages, and domain packages depend on the database,
models, and small shared helpers. Nothing in `internal/models` imports
`internal/db`; persistence converts rows with `db.Media.ToModel()`.

## Packages

| Package | Responsibility |
| --- | --- |
| `cmd/disco` | Process entry point, kong CLI wiring, profiling. |
| `internal/commands` | One file per CLI command; no HTTP handlers. |
| `internal/server` | HTTP server lifecycle, routing, flags parsing, and API handlers. |
| `internal/query` | SQL/filter/sort construction, FTS syntax, and aggregation entry points. |
| `internal/aggregate` | In-memory grouping, statistics, and similarity logic. |
| `internal/db` | SQLite connections, schema/migrations, row types, and queries. |
| `internal/db/schema` | Embedded SQL schema files. |
| `internal/models` | Shared domain and API types plus CLI flag definitions. |
| `internal/metadata` | ffprobe-driven media metadata extraction. |
| `internal/fs` | Filesystem walking and media discovery. |
| `internal/history` | Playback history persistence. |
| `internal/categorize` | Keyword-to-category matching shared by CLI and server. |
| `internal/media` | Media probing, playback (mpv), transcoding, images, and subtitles. |
| `internal/reading` | Document text extraction, word counts, RSVP, and TTS. |
| `internal/textutil` | String, path-token, and corpus processing. |
| `internal/formatutil` | Human-readable sizes/durations/dates and date parsing. |
| `internal/fileutil` | Filesystem, hashing, process, and terminal I/O helpers. |
| `internal/selfupdate` | Release checking, download, and version information. |
| `internal/config` | Platform-specific configuration and cache paths. |
| `internal/logging` | Process-wide structured logger. |
| `internal/pathutil` | Cross-platform path splitting and joining. |
| `internal/utils` | Small generic numeric, range, nullable-SQL, and iterator helpers. |
| `internal/shellquote` | POSIX shell quoting. |
| `internal/testutils` | Shared test fixtures. |
| `web` | Embedded frontend assets. |

## Dependency rules

- `models` must not import `db`.
- In production code, `commands` and `server` must not import each other.
- Domain packages must not import `commands` or `server`.
- `logging` owns the logger; do not add per-package loggers or `models.Log`.
- Keep tests next to the package they exercise. Name test files after the
  behavior, not a project phase.

## Build tags

SQLite FTS5 support is enabled with `-tags fts5`, which is the default in the
`Makefile` targets.
