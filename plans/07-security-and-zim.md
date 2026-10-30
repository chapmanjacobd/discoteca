# Phase 7: Security Hardening and External ZIM Server

**This is the final phase.** Do not start it until phases 1–6 pass their exit checks.

Goal: make the server’s trust boundary explicit, validate every destructive request, and replace
the managed `kiwix-serve` subprocess with an externally managed server on a configured port.

## 7.1 Bind to localhost by default

Files: `internal/commands/serve.go`, `README.md`, `docs/DEPLOYMENT.md`

Current problem: `Run` listens on `":<port>"`, so the server is reachable on every interface while
the API token is handed to any static client.

Tasks:

- [ ] Add `Host` to `ServeCmd` with `default:"127.0.0.1"`.
- [ ] Build the listen address from `Host` and `Port`.
- [ ] If `Host` is non-loopback and `DISCO_API_TOKEN` is unset, refuse to start with a clear error.
- [ ] Add `--allow-remote` only if there is a real reason to allow an unauthenticated remote bind;
      otherwise require the token.
- [ ] Update deployment docs: reverse proxy is the recommended remote deployment.
- [ ] Test `Mux`/`Run` address construction without binding a real port.

## 7.2 Stop handing the API token to every client

Files: `internal/commands/serve.go`, `web/src/main.ts`, tests

Current problem: `newStaticHandler` sets `disco_token` to `c.APIToken` on every static request.
`GET /` is unauthenticated, so the token is public.

Tasks:

- [ ] Remove the `disco_token` cookie write from `newStaticHandler`.
- [ ] Generate the token only when `DISCO_API_TOKEN` is unset, and print it once to stderr with a
      clear “use this to authenticate” message.
- [ ] Add a session endpoint:
  - `POST /api/session` accepts `Authorization: Bearer <token>`,
  - on success, sets an `HttpOnly`, `SameSite=Strict`, `Secure` when appropriate cookie,
  - returns 204.
- [ ] The web UI obtains the token from a user prompt or a `?token=` URL fragment, then calls
      `/api/session` and removes the token from the URL.
- [ ] Keep `X-Disco-Token` header support for scripts.
- [ ] `authMiddleware` remains for API writes and reads that need it.
- [ ] Add tests:
  - `GET /` does not set a token cookie,
  - `POST /api/session` with the wrong token returns 401,
  - with the right token it sets the cookie,
  - no token appears in static HTML or JS.

If a simpler interim design is preferred: keep the token as a CSRF guard, bind to localhost by
default, and document that remote deployments must authenticate at the reverse proxy. Do not
pretend the cookie is authentication when any client can fetch it.

## 7.3 Remove SQL injection surfaces

Files: `internal/commands/serve.go`, `internal/query/filter_builder.go`

Current problems:

- `min_score`/`max_score` are concatenated into `flags.Where`.
- `sort` is passed to `OverrideSort` and injected into `ORDER BY` after string replacements.

Tasks:

- [ ] Parse `min_score`/`max_score` with `strconv.ParseFloat`; reject non-finite or invalid values
      with 400.
- [ ] Format validated numbers with `strconv.FormatFloat`, or add a parameterized `WhereArgs`
      mechanism to `FilterBuilder`.
- [ ] Allowlist `sort` values. Accept only known fields, preset names, and the documented
      `field desc` syntax; reject anything else.
- [ ] Do the same for `sort_fields`/`sort_order`/`sort_desc` if they can reach SQL.
- [ ] Audit every other server query parameter that reaches `flags.Where`, `SortBy`, `Query`,
      `PlayInOrder`, or raw SQL.
- [ ] Add tests that inject `1; DROP TABLE media`, `1 OR 1=1`, and `(SELECT ...)` through each
      parameter and assert 400/no query execution.

## 7.4 Lock down `/api/empty-bin`

Files: `internal/commands/serve_extra_handlers.go`, tests

Current problem: request-provided paths are deleted without DB or trash membership checks.

Tasks:

- [ ] When `paths` is non-empty, load the matching rows with `OnlyDeleted=true` and require every
      requested path to be present.
- [ ] Reject paths that are not in the configured databases.
- [ ] Reject sensitive/blocklisted paths through the same helper used by raw/thumbnail.
- [ ] Use a transaction per database for the row delete.
- [ ] Count files removed, not databases touched.
- [ ] Return a structured result: `{deleted, failed, skipped}`.
- [ ] Add tests: unlisted path, listed path, path outside allowed DB, read-only mode, partial
      failure.

## 7.5 Lock down `/api/epub`

Files: `internal/commands/serve_epub.go`, tests

Current problems:

- Arbitrary local paths are served after only a small blocklist.
- Asset containment uses a string prefix.
- Book HTML runs unsandboxed in the application origin.

Tasks:

- [ ] Require every document path to be present in a configured database (reuse
      `getMediaTypeFromDB` or an exact-path query).
- [ ] Replace `strings.HasPrefix(assetFile, docPath)` with a `filepath.Rel` containment check.
- [ ] Reject paths that escape the document directory after cleaning.
- [ ] Remove the double `PathUnescape`; decode exactly once.
- [ ] Serve book HTML with:
  - a sandboxed iframe (`sandbox="allow-same-origin"` only if necessary; prefer no scripts),
  - a restrictive CSP for the wrapper,
  - `X-Content-Type-Options: nosniff`.
- [ ] Ensure `/api/epub` cannot be used to run `ebook-convert` on arbitrary files.
- [ ] Add tests for `../`, encoded `..`, sibling-prefix paths, unlisted files, and a listed
      EPUB/PDF.

## 7.6 Require DB membership for streaming endpoints

Files: `internal/commands/serve_streaming.go`, tests

Current problems:

- `/api/hls/segment` only checks file existence.
- `/api/play` accepts any existing local file.
- Fuzzy subtitle lookup only scans the first 1000 rows.

Tasks:

- [ ] Add the same exact-path DB check used by `/api/raw` to `HandleHLSSegment` and `HandlePlay`.
- [ ] Return 403 for paths not in any configured DB.
- [ ] Replace the fuzzy subtitle `GetMedia(ctx, 1000)` with an indexed path/sidecar lookup.
- [ ] Do not mark a file deleted merely because it is missing unless the caller is an explicit
      library action; a playback probe should not mutate the DB.
- [ ] Add tests for listed, unlisted, missing, and sidecar paths.

## 7.7 Validate `/api/metadata?db=`

Files: `internal/commands/serve_metadata.go`

Current problem: the query parameter bypasses `filterDatabases`.

Tasks:

- [ ] Run `db` through `filterDatabases` and return 400 when it is not in the allowed list.
- [ ] Do not create a new SQLite file as a side effect of opening an arbitrary path.
- [ ] Add a test with an allowed DB and a path outside the list.

## 7.8 Make write handlers return real failures

Files: `internal/commands/serve_handlers.go`, `serve_extra_handlers.go`, `serve_reorder.go`

Current problem: handlers log `execDB` errors and still return 200/201.

Tasks:

- [ ] For each mutating handler, collect errors across databases.
- [ ] Return non-2xx when every write fails; return a partial-success body when some succeed.
- [ ] Verify rows affected for rating/progress/mark-played/mark-unplayed; unknown paths return 404.
- [ ] Wrap playlist create/delete/reorder in transactions.
- [ ] Verify reorder updates the expected number of tracks.
- [ ] Add tests for total failure, partial failure, and success.

## 7.9 Require the right HTTP methods

Files: `internal/commands/serve_extra_handlers.go`, `serve_queries.go`

Tasks:

- [ ] `/api/categorize/apply`: require `POST`.
- [ ] `/api/categorize/keyword`: require `POST` for insert and `DELETE` for delete.
- [ ] `/api/queries?action=clear|toggle`: require `POST`; keep `GET` for read-only listing.
- [ ] Reject unknown methods with 405.
- [ ] Add tests for each method.

## 7.10 Harden forwarded-prefix handling

Files: `internal/commands/serve_basepath.go`

Current problem: `X-Forwarded-Prefix` is trusted from any client when `--base-path` is unset.

Tasks:

- [ ] Trust the header only when explicitly enabled (`--trust-forwarded-prefix`) or when the
      immediate peer is loopback.
- [ ] Validate/normalize the prefix as a path segment set; reject schemes, hosts, `..`, and
      backslashes.
- [ ] Do not let the header change the auth cookie path for an unauthenticated request.
- [ ] Add tests for normal prefix, `//evil.example`, `http://evil`, and `../`.

## 7.11 Replace managed Kiwix with an external server

Files: `internal/commands/zim.go`, `internal/commands/serve.go`, `internal/commands/zim_test.go`,
`README.md`, `docs/DEPLOYMENT.md`

Decision: **discoteca does not start, stop, wait for, or clean up `kiwix-serve`.** An operator runs
it externally and tells discoteca which port to proxy to.

### Configuration

- [ ] Add `KiwixPort int` to `ServeCmd`:
      `help:"Port of the externally managed kiwix-serve (0 disables ZIM support)" default:"8181"`.
- [ ] Optionally add `KiwixHost` defaulting to `127.0.0.1`.
- [ ] Validate the port range (`1..65535`) and return a startup error for invalid values.
- [ ] If `KiwixPort == 0`, ZIM routes return 503 “ZIM support disabled” instead of panicking.

### Remove process management

- [ ] Delete `KiwixBin` and `KiwixPortStart`.
- [ ] Delete `KiwixInstance`, `KiwixManager`, `ZimManager`, `StartZimManager`.
- [ ] Delete `EnsureKiwixServing`, `FindAvailablePort`, `cleanupOldInstances`, `IsPortAvailable`.
- [ ] Delete `WaitForKiwixReady`, `DefaultWaitForKiwixReady`, `defaultWaitForKiwixReadyAt`, and
      `waitForKiwixReady`.
- [ ] Remove `StartZimManager()` from `ServeCmd.Run`.
- [ ] Remove unused imports (`os/exec`, `net`, `strconv`, `sync`, `time`, etc.) and the
      `KiwixBin` constant.
- [ ] Verify `grep -R "exec.Command" internal/commands/zim.go` returns nothing.

### Fixed-port proxy

- [ ] Change the route from `/api/zim/proxy/{port}/{rest...}` to
      `/api/zim/proxy/{rest...}`.
- [ ] Build the proxy target from `c.KiwixHost` and `c.KiwixPort` only.
- [ ] Never read a port from the request path or query; a request must not be able to redirect the
      proxy to another port or host.
- [ ] Preserve the base-path/`urlRootLocation` behavior with one documented convention.
- [ ] Update `getKiwixContentURL` to use the configured port and the new route.
- [ ] `HandleZimView`:
  - verify the `.zim` path exists and is listed in a configured database,
  - use the configured port,
  - if the external server is unreachable, return 503 with a clear “kiwix-serve is not reachable”
    message,
  - do not start or wait for a process.
- [ ] Update the generated iframe/TOC URLs to the new proxy path.
- [ ] Update `web/src/main.ts` only if it constructs proxy URLs directly (currently it opens
      `/api/zim/view`).

### Document the external invocation

Add to `README.md` and `docs/DEPLOYMENT.md`:

```bash
# Terminal 1: run one or more ZIM files in an external kiwix-serve
kiwix-serve --port=8181 \
  --urlRootLocation=/api/zim/proxy/ \
  /path/to/library.zim

# Terminal 2: point discoteca at it
disco serve --kiwix-port 8181 library.db
```

- [ ] Document that the external server must use the same `--urlRootLocation` as the proxy route.
- [ ] Document that the ZIM file must also be present in the discoteca database for the view
      endpoint to accept it.
- [ ] Document that `--kiwix-port 0` disables ZIM support.
- [ ] Remove the “kiwix-serve is started by discoteca” wording from the README.

### Tests

- [ ] Update `zim_test.go` to test only:
  - configured port parsing,
  - fixed-port proxy target,
  - rejection of a request-supplied port,
  - disabled mode,
  - unreachable upstream returns 503,
  - valid `.zim` path in DB returns HTML.
- [ ] Delete tests for port allocation, instance caching, cleanup, and readiness.
- [ ] Add a test that no child process is spawned (inject a fake runner only if any process code
      remains; the goal is that none does).

## 7.12 Final security review

- [ ] Run `go test -tags fts5 -race ./internal/commands ./internal/query ./internal/db`.
- [ ] Run a manual check with the server bound to `127.0.0.1`:
  - `GET /` does not expose a token,
  - `POST /api/session` works with the printed token,
  - unauthenticated writes return 401,
  - `/api/empty-bin` rejects an unlisted path,
  - `/api/epub/../../etc/passwd` is rejected,
  - `/api/hls/segment?path=/etc/passwd` is rejected,
  - `/api/metadata?db=/tmp/other.db` is rejected,
  - `/api/zim/proxy/9999/` does not reach port 9999,
  - `min_score=1;DROP TABLE media` returns 400.
- [ ] Confirm the server does not create files outside configured DB paths.
- [ ] Review the final diff for any remaining `flags.Where` concatenation or `exec.Command` with
      request-controlled arguments.

## Phase exit criteria

- The server binds to localhost by default.
- The token is not available from an unauthenticated static response.
- Every destructive endpoint validates DB membership and path containment.
- SQL sort/score parameters are allowlisted or parameterized.
- No `kiwix-serve` process is started, waited for, or killed by discoteca.
- ZIM proxying uses exactly one configured host/port.
- Security tests pass before the phase is considered complete.
