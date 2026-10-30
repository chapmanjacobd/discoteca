# Discoteca Correctness and Cleanup Plan

Status: proposed, not started
Source: repository audit (tests pass; `deadcode -tags fts5 ./...` reports 186 unreachable functions)
Scope: correctness, data-loss prevention, feature wiring, frontend state, dead code, then security

## Rules for every phase

- Each phase must land independently and keep `go test -tags fts5 ./...` green.
- Add regression tests before or with each behavior change.
- Do not change the Git index. Do not commit unless explicitly asked.
- Prefer deleting a falsely advertised flag over leaving it as a no-op.
- When a behavior change is intentionally breaking, document it in the phase and in `README.md`.
- Run the phase exit checks before starting the next phase.

## Phase order

| # | File | Focus | Depends on |
|---|---|---|---|
| 1 | `01-destructive-commands.md` | Stop silent data loss in CLI write paths | — |
| 2 | `02-database-migrations.md` | Migrations, schema drift, FTS lifecycle | 1 |
| 3 | `03-query-sort-aggregation.md` | Query scanning, sorting, pagination, DU semantics | 2 |
| 4 | `04-cli-feature-wiring.md` | Wire or remove flags and advertised command behavior | 3 |
| 5 | `05-web-frontend.md` | Fix broken UI modes and state handling | 4 |
| 6 | `06-dead-code-redundancy.md` | Remove superseded implementations and dead exports | 5 |
| 7 | `07-security-and-zim.md` | Server hardening, then external ZIM server | 1–6 |

**Security work is intentionally last.** Do not start phase 7 until phases 1–6 have passing tests and no known data-loss regressions.

## Fixed design decisions

- **ZIM:** discoteca will not start, stop, wait for, or clean up `kiwix-serve`. An operator runs it
  externally and passes the port with a new `--kiwix-port` flag (default `8181`; `0` disables ZIM).
  Phase 7 removes the managed-process code and fixes the proxy to that one port.
- **Local by default:** the server should bind to `127.0.0.1` by default. Remote deployments should
  use a reverse proxy and an explicit `DISCO_API_TOKEN`.
- **One schema source:** `internal/db/schema/*.sql` is canonical; the legacy root schema files are
  test-only until phase 2 removes them.

## Exit checks

Run after each phase:

```bash
go test -tags fts5 ./...
go vet -tags fts5 ./...
go build -tags fts5 ./cmd/disco
```

Run after phases 1, 2, and 7:

```bash
go test -tags fts5 -race ./internal/commands ./internal/db ./internal/query
```

Run after phase 5:

```bash
npm test --prefix web
npm run build --prefix web
```

Run after phase 6:

```bash
deadcode -tags fts5 ./...
```

The dead-code count should be trending down. Test-only helpers in `internal/testutils` may remain; do not delete them merely to satisfy the count.

## Non-goals

- No database format rewrite beyond what is needed to remove ambiguity.
- No new external dependencies for the Go backend.
- No change to the required `fts5` build tag.
- No promise that the API token is a complete authentication system. Phase 7 defines the actual boundary.

## Manual verification assets

The repository root contains real databases (`audio.db`, `books.db`, `images.db`, `video.db`). Do not mutate them during development. Copy one to `/tmp/opencode/` when a migration or destructive-command test needs realistic data.
