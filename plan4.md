# Plan 4: Database & Media Fixture Factory

## Goal
Accelerate testing by removing the dependency on the `disco add` command for every test case.

## Context
Currently, most tests start by running `disco add <folder>`, which is slow and involves actual filesystem I/O and media probing. We need a way to inject state directly into the SQLite database.

## Tasks

### 1. SQLite Seeding Factory
- **File:** `e2e/utils/db-factory.ts` (or a Go equivalent in `internal/testutils`).
- **Functionality:** Generate a temporary `.db` file and insert rows into the `media` table with specific attributes (path, size, duration, play_count).
- **Integration:** Update `fixtures.ts` to allow tests to request a pre-seeded database.

### 2. Dynamic Media Stub Generator
- **Goal:** Replace real MP4/MP3 files in `e2e/fixtures/` with minimal valid headers.
- **Action:** Use a script to generate 1KB files that "look" like media to `ffmpeg` or the browser to avoid committing large binaries.

### 3. Isolated DB Templates
- Create a set of "Gold Master" databases for common scenarios:
    - `empty.db`
    - `large-collection.db` (1000+ items)
    - `duplicates.db` (items with same size/hash)

## Verification
- Measure the time of `cli-history-stats.spec.ts` before and after switching to direct seeding.
- Target: 50% reduction in test setup time.
