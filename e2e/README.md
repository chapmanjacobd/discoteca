# Discothèque E2E Tests

End-to-end tests for Discothèque using [Playwright](https://playwright.dev/).

## Test Strategy

### E2E Tests (Playwright)
Test complete user workflows with a real server:

#### Web UI Tests
- Navigation and routing
- Media playback (video, audio, images)
- Document viewing (PDF/EPUB)
- Caption display and subtitle selection
- Fullscreen toggle
- Metadata modal
- Trash functionality
- Large result sets scrolling
- Broken media handling
- Range sliders (Size, Duration, Episodes)
- Disk Usage navigation
- Search and filtering
- Playlist management
- Settings persistence

#### CLI E2E Tests
Test CLI commands end-to-end:
- `add` - Add media to database
- `check` - Check for missing files
- `print` - Print media information
- `search` - Search media using FTS
- `search-captions` - Search captions
- `playlists` - List scan roots
- `search-db` - Search arbitrary tables
- `media-check` - Check media corruption
- `files-info` - Show file information
- `disk-usage` - Show disk usage aggregation
- `dedupe` - Deduplicate similar media
- `categorize` - Auto-group media
- `similar-files` / `similar-folders` - Find similar items
- `watch` / `listen` - Play media with mpv
- `stats` - Show library statistics
- `history` / `history-add` - Playback history
- `mpv-watchlater` - Import mpv watchlater
- `serve` - Start Web UI server
- `optimize` / `repair` - Database maintenance
- `tui` - Interactive TUI mode
- `now` / `next` / `stop` / `pause` / `seek` - MPV control
- `regex-sort` / `cluster-sort` - Sorting commands
- `sample-hash` - Calculate file hash
- `open` / `browse` - Open files/URLs
- `update` / `version` - Version management
- `merge-dbs` - Merge databases
- `explode` - Create symlinks

### Unit/Integration Tests (Vitest)
Keep in `web/tests/` for fast feedback on:
- Component rendering
- State management
- Utility functions
- Event handlers
- Mocked API interactions

## Quick Start

```bash
# First time setup - build binary and create test database
make e2e-init

# Install Playwright browsers
cd e2e && npm install && npx playwright install

# Run all E2E tests (headless)
make e2e

# Run only Web UI tests
make e2e-web

# Run only CLI E2E tests
make e2e-cli

# Run tests with UI
cd e2e && npm run test:ui

# Run tests in headed mode (see browser)
cd e2e && npm run test:headed

# Debug tests
cd e2e && npm run test:debug

# View test report
cd e2e && npm run test:report
```

## Test Database

The test database is generated dynamically for each test run to ensure:
- **Portability**: No absolute paths tied to specific machines
- **Schema tracking**: Full SQL schema committed to `e2e/fixtures/schema.sql`
- **Reproducibility**: Same test data generated every time

### Test Data

The database is seeded with:
- 10 sample media files (videos, audio, images, documents)
- 7 caption entries from sidecar VTT files
- Pre-computed metadata for fast test execution

### Schema Migrations

When the database schema changes:

1. **Regenerate the database:**
   ```bash
   make e2e-init
   ```

2. **Verify tests still pass:**
   ```bash
   make e2e
   ```

3. **Review and commit schema changes:**
   ```bash
   git diff e2e/fixtures/schema.sql
   git add e2e/fixtures/schema.sql
   git commit -m "Update E2E schema: added column XYZ"
   ```

The `schema.sql` file contains the complete database schema, allowing you to:
- Track schema evolution over time
- Review changes in pull requests
- Understand the database structure without running code

### Manual Database Inspection

```bash
# View schema
sqlite3 e2e/fixtures/test.db ".schema"

# View test data
sqlite3 e2e/fixtures/test.db "SELECT path, type, size FROM media LIMIT 5;"

# Compare schema with committed version
diff <(sqlite3 e2e/fixtures/test.db ".schema") e2e/fixtures/schema.sql
```

## Prerequisites

1. **Go** - To build the Disco server
2. **Node.js 18+** - To run Playwright tests
3. **Disco binary** - Built automatically by the test runner

## Test Structure

```
e2e/
├── fixtures.ts           # Web UI test fixtures (server, DB)
├── fixtures-cli.ts       # CLI test fixtures (CLI runner, temp dirs)
├── playwright.config.ts  # Playwright configuration
├── utils/
│   ├── test-server.ts    # Disco server management
│   ├── cli-runner.ts     # CLI command runner
│   └── seed-db.ts        # Database seeding
└── tests/
    # CLI E2E Tests
    ├── cli-add.spec.ts                    # Add command tests
    ├── cli-check-print-search.spec.ts     # Check, Print, Search commands
    ├── cli-history-stats.spec.ts          # History, Stats, Playlists, Optimize
    ├── cli-media-check-files-info-du.spec.ts  # Media-Check, Files-Info, Disk-Usage
    ├── cli-categorize-similar-dedupe.spec.ts  # Categorize, Similar, Dedupe, Big-Dirs
    ├── cli-watch-listen-serve.spec.ts     # Watch, Listen, Serve commands
    ├── cli-mpv-control-sort.spec.ts       # MPV control, Regex/Cluster sort, Sample-hash
    └── cli-open-browse-etc.spec.ts        # Open, Browse, Update, Version, etc.
    
    # Web UI E2E Tests
    ├── navigation.spec.ts                 # Basic navigation tests
    ├── navigation-optimized.spec.ts       # Optimized navigation
    ├── captions.spec.ts                   # Caption functionality
    ├── du-navigation.spec.ts              # Disk Usage navigation
    ├── playback.spec.ts                   # Media playback
    ├── search-filter.spec.ts              # Search and filtering
    ├── settings-playback.spec.ts          # Settings and playback controls
    ├── query-integration.spec.ts          # Query integration tests
    ├── media-viewers.spec.ts              # PDF/EPUB, Image, Audio viewers
    ├── ui-interactions.spec.ts            # Fullscreen, Metadata, Trash
    ├── subtitles.spec.ts                  # Subtitle selection
    ├── scrolling-error-handling.spec.ts   # Scrolling, Error handling
    └── range-sliders.spec.ts              # Size, Duration, Episodes sliders
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DISCO_BASE_URL` | `http://localhost:8080` | Base URL for the Disco server |
| `DISCO_BINARY` | `./cmd/disco/disco` | Path to the Disco binary |

### Running Against Existing Server

If you already have a Disco server running:

```bash
DISCO_BASE_URL=http://localhost:8080 npx playwright test
```

### Running Specific Tests

```bash
# Run specific test file
npx playwright test tests/captions.spec.ts

# Run tests by name pattern
npx playwright test -g "caption"

# Run specific browser
npx playwright test --project=firefox

# Run with retries
npx playwright test --retries=3
```

## Writing Tests

### Web UI Tests

```typescript
import { test, expect } from '../fixtures';

test.describe('My Feature', () => {
  test('does something', async ({ page, server }) => {
    // Navigate to the app
    await page.goto(server.getBaseUrl());

    // Interact with the page
    await page.click('#some-button');

    // Make assertions
    await expect(page.locator('.result')).toBeVisible();
  });
});
```

### CLI E2E Tests

```typescript
import { test, expect } from '../fixtures-cli';

test.describe('CLI: My Command', () => {
  test('does something', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create a dummy video file
    const videoPath = createDummyVideo('test.mp4');
    
    // Run add command
    await cli.runAndVerify(['add', testDbPath, videoPath]);
    
    // Verify file is in database
    const result = await cli.runJson(['print', testDbPath, '--all']);
    expect(result.length).toBe(1);
  });
});
```

### Available Fixtures

#### Web UI Fixtures (`fixtures.ts`)
- `page` - Playwright Page object
- `server` - Running Disco server instance
- `testDbPath` - Path to the test database

#### CLI Fixtures (`fixtures-cli.ts`)
- `cli` - CLI runner instance
- `tempDir` - Temporary directory for test files
- `testDbPath` - Path to the test database
- `createDummyFile` - Helper to create dummy files
- `createDummyVideo` - Helper to create dummy video files
- `createDummyAudio` - Helper to create dummy audio files
- `createDummyImage` - Helper to create dummy image files
- `createDummyDocument` - Helper to create dummy document files (PDF/EPUB)
- `createDummyVtt` - Helper to create dummy VTT subtitle files

## CI/CD

Tests run automatically on:
- Push to `main` branch
- Pull requests to `main` branch

Test artifacts (screenshots, videos, traces) are uploaded as GitHub Actions artifacts for failed tests.

## Debugging

### Using Playwright Inspector

```bash
npm run test:debug
```

### Using VS Code

Install the [Playwright Test for VS Code](https://marketplace.visualstudio.com/items?itemName=ms-playwright.playwright) extension.

### Trace Viewer

After a test run, view traces with:

```bash
npx playwright show-trace test-results/<test-name>/trace.zip
```

## Test Database

The test database is automatically seeded before each test run with:
- 10 sample media files (videos, audio, images, documents)
- 7 caption entries (all after 10 seconds)
- 3 categories with keywords
- 1 playlist with 3 items

The database is located at `e2e/fixtures/test.db` and is cleaned before each test run.

## Troubleshooting

### Server fails to start

Ensure the Disco binary exists:
```bash
go build -o cmd/disco/disco ./cmd/disco
```

### Tests timeout

Increase timeout in `playwright.config.ts`:
```typescript
use: {
  timeout: 120000, // 2 minutes
}
```

### Browser not found

Install browsers:
```bash
npx playwright install
```

### Port already in use

Change the port:
```bash
DISCO_BASE_URL=http://localhost:8081 npx playwright test
```
