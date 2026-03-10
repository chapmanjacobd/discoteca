# Plan 1: Core Test Restoration

## Goal
Restore lost test coverage for critical API endpoints and CLI commands following the recent architectural refactoring into the `internal/commands` package.

## Context
The project recently moved logic from `main.go` into modular commands under `internal/commands`. Many tests were either commented out, deleted, or left in a failing state during this transition. We need to ensure that the core functionality remains stable.

## Tasks

### 1. API Handler Tests (`internal/commands/serve_test.go`)
Re-implement or repair unit tests for the following handlers in `serve_handlers.go`:
- **Metadata/Discovery:** `HandleCategories`, `HandleGenres`, `HandleRatings`.
- **Media Operations:** `HandleRate` (updating ratings), `HandleDelete` (moving to trash), `HandleProgress` (syncing playhead).
- **Advanced Views:** `HandleQuery` (arbitrary SQL), `HandleDU` (Disk Usage calculation), `HandleEpisodes` (TV show grouping).
- **Streaming:** `HandleSubtitles` (VTT conversion), `HandleThumbnail` (FFmpeg extraction).

### 2. CLI Command Integration Tests (`cmd/disco/e2e_test.go`)
Ensure the following commands are verified via the `cli-runner.ts` or Go-native integration tests:
- `Categorize`: Verify media is moved to correct subfolders.
- `Similar`: Test both file and folder similarity detection.
- `Dedupe`: Ensure duplicate files are identified and handled.
- `Stats/Print`: Validate output formats (Text, JSON, Table).
- `Media-Check`: Verify it identifies missing or corrupted files.

### 3. Progress Tracking Fix
- **File:** `web/tests/progress.test.js` (and corresponding Go handler)
- **Issue:** In read-only mode, the UI currently ignores `item.playhead`.
- **Fix:** Ensure the API sends the playhead even in read-only mode, and the UI respects it for visual progress without allowing updates.

## Verification
- Run `go test ./internal/commands/...`
- Run `go test ./cmd/disco/...`
- Ensure `npm run test` in the `web` directory passes for the progress tracking logic.
