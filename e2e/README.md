# Discothèque E2E Tests

End-to-end tests for Discothèque using [Playwright](https://playwright.dev/).

## Test Strategy

### E2E Tests (Playwright)
Test complete user workflows with a real server:
- Navigation and routing
- Media playback
- Caption display and jumping
- Disk Usage navigation
- Search and filtering
- Playlist management
- Settings persistence

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

# Run tests (headless)
make e2e

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
├── fixtures.ts           # Test fixtures (server, DB)
├── playwright.config.ts  # Playwright configuration
├── utils/
│   ├── test-server.ts    # Disco server management
│   └── seed-db.ts        # Database seeding
└── tests/
    ├── navigation.spec.ts       # Basic navigation tests
    ├── captions.spec.ts         # Caption functionality
    ├── du-navigation.spec.ts    # Disk Usage navigation
    ├── playback.spec.ts         # Media playback
    └── search-filter.spec.ts    # Search and filtering
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

### Available Fixtures

- `page` - Playwright Page object
- `server` - Running Disco server instance
- `testDbPath` - Path to the test database

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
