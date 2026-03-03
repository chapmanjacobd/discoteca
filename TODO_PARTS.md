# Discotheque TODO List by Component

## `cmd/` (Entry Points)
- [ ] `cmd/disco/main.go`: Improve error handling and logging configuration
- [ ] `cmd/syncweb/`: Ensure consistency between standalone `syncweb` and the one integrated into `disco`.

## `internal/db/` & `internal/query/` (Database & Queries)
- [ ] `schema.sql`: Review and optimize indexes for frequent queries.
- [ ] `migrations.go`: Ensure migrations are idempotent and robust against partial failures.
- [ ] `query.go`: Improve FTS5 search performance and add more complex search filters.
- [ ] Add more comprehensive tests for complex query builders in `query_test.go`.

## `internal/metadata/` (Media Extraction)
- [ ] Enhance metadata extraction to include more detailed media information (e.g., codec details, subtitle tracks).
- [ ] Improve handling of corrupted or unusual media files.
- [ ] Benchmark and optimize extraction speed for large media libraries.

## `web/` (Frontend)
- [ ] `app.js`: Refactor frontend logic into smaller modules or components.
- [ ] Implement better state management (e.g., using a lightweight store instead of global variables).
- [ ] Improve UI responsiveness and mobile experience.
- [ ] Increase test coverage for frontend components (currently 20+ `.test.js` files, but verify depth).
- [ ] Optimize loading performance for large media lists.

## `internal/utils/` (Utilities)
- [ ] `mpv.go`: Enhance MPV integration (e.g., better event handling, more control options).
- [ ] `shell_utils.go`: Audit for shell injection vulnerabilities and improve error reporting.
- [ ] General: Add more unit tests for core utility functions in `*_test.go`.
