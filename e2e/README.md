# Discothèque E2E Tests

End-to-end tests using [Playwright](https://playwright.dev/) with Page Object Model (POM) architecture.

## Quick Start

```bash
# First time setup
make e2e-init

# Run all tests (headless)
make e2e

# Run with UI
cd e2e && npm run test:ui

# Debug tests
cd e2e && npm run test:debug

# View test report
cd e2e && npm run test:report
```

> **Important:** All `npx playwright` commands must be run from within the `e2e/` directory.
> The Playwright configuration and dependencies are located in `e2e/`, not the project root.

## Architecture

### Page Objects

Reusable page-specific logic in `e2e/pages/`:

| Page Object | Description |
|-------------|-------------|
| `MediaPage` | Media grid/list interactions |
| `SidebarPage` | Sidebar navigation and filters |
| `ViewerPage` | Media player controls |

### Custom Matchers

Extended assertions in `e2e/utils/matchers.ts`:

- `toHaveMediaCount(count)` - Verify media card count
- `toBeInMode(mode)` - Verify URL hash mode
- `toHaveProgress(expected)` - Verify progress bar
- `toBePlaying()` / `toBePaused()` - Verify playback state

### Test Fixtures

| Fixture | Description |
|---------|-------------|
| `mediaPage` | MediaPage instance |
| `sidebarPage` | SidebarPage instance |
| `viewerPage` | ViewerPage instance |
| `server` | Running Disco server |
| `readOnly` | Test modifies server state |

## Writing Tests

```typescript
import { test, expect } from '../fixtures';

test.describe('My Feature', () => {
  test.use({ readOnly: true });

  test('opens media', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await expect(viewerPage.playerContainer).toBeVisible();
  });
});
```

### Waiting Strategies

```typescript
// ✅ Wait for element
await element.waitFor({ state: 'visible' });

// ✅ Wait for API response
const [response] = await Promise.all([
  page.waitForResponse(resp => resp.url().includes('/api/query')),
  page.click('#search-button'),
]);

// ✅ Wait for condition
await page.waitForFunction(() => {
  const video = document.querySelector('video');
  return video && video.duration > 0;
});
```

## Running Tests

```bash
# All tests
npx playwright test --project=desktop

# Specific file
npx playwright test tests/navigation-pom.spec.ts

# Pattern match
npx playwright test -g "playback"

# Headed mode (visible browser)
npx playwright test --headed

# Check for flakes (5 runs)
for i in 1 2 3 4 5; do npx playwright test --project=desktop; done

# Generate README screenshots (strips metadata for deterministic output)
make screenshots
```

> **Note:** Always run `npx playwright` commands from the `e2e/` directory.

## Test Database

Auto-generated at `e2e/fixtures/test.db` with:
- 10 sample media files (video, audio, images, documents)
- 7 caption entries from VTT sidecars
- Pre-computed metadata

### Schema Changes

```bash
# Regenerate database
make e2e-init

# Review schema changes
git diff e2e/fixtures/schema.sql

# Commit
git add e2e/fixtures/schema.sql
git commit -m "Update E2E schema"
```

## File Structure

```
e2e/
├── pages/
│   ├── media-page.ts      # Media grid POM
│   ├── sidebar-page.ts    # Sidebar POM
│   └── viewer-page.ts     # Player POM
├── utils/
│   ├── matchers.ts        # Custom matchers
│   ├── test-server.ts     # Server management
│   └── cli-runner.ts      # CLI runner
├── tests/
│   ├── navigation-pom.spec.ts   # Navigation (POM example)
│   ├── playback-pom.spec.ts     # Playback (POM example)
│   ├── cli-*.spec.ts            # CLI E2E tests
│   └── *.spec.ts                # Web UI tests
├── fixtures/
│   ├── test.db            # Test database
│   └── schema.sql         # Database schema
├── fixtures.ts            # Web UI fixtures
├── fixtures-cli.ts        # CLI fixtures
├── playwright.config.ts   # Configuration
└── index.ts               # Exports
```

## CLI E2E Tests

Test CLI commands end-to-end using `fixtures-cli.ts`:

```typescript
import { test, expect } from '../fixtures-cli';

test('add command', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
  const videoPath = createDummyVideo('test.mp4');
  await cli.runAndVerify(['add', testDbPath, videoPath]);
  
  const result = await cli.runJson(['print', testDbPath, '--all']);
  expect(result.length).toBe(1);
});
```

### Available CLI Commands

| Command | Test File |
|---------|-----------|
| `add`, `check`, `print`, `search` | `cli-add.spec.ts`, `cli-check-print-search.spec.ts` |
| `history`, `stats`, `playlists` | `cli-history-stats.spec.ts` |
| `media-check`, `files-info`, `disk-usage` | `cli-media-check-files-info-du.spec.ts` |
| `categorize`, `similar-*`, `dedupe` | `cli-categorize-similar-dedupe.spec.ts` |
| `watch`, `listen`, `serve` | `cli-watch-listen-serve.spec.ts` |
| `mpv-*`, `regex-sort`, `cluster-sort` | `cli-mpv-control-sort.spec.ts` |

## Debugging

### Playwright Inspector
```bash
npm run test:debug
```

### Trace Viewer
```bash
npx playwright show-trace test-results/<test-name>/trace.zip
```

### VS Code
Install [Playwright Test extension](https://marketplace.visualstudio.com/items?itemName=ms-playwright.playwright)

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Server fails to start | `go build -o cmd/disco/disco ./cmd/disco` |
| Tests timeout | Increase `timeout` in `playwright.config.ts` |
| Browser not found | `npx playwright install` |
| Port in use | `DISCO_BASE_URL=http://localhost:8081 npx playwright test` |

## CI/CD

Tests run on:
- Push to `main`
- Pull requests to `main`

Artifacts (screenshots, videos, traces) uploaded for failed tests.
