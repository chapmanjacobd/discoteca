# Discothèque E2E Tests

End-to-end tests for Discothèque using [Playwright](https://playwright.dev/).

## Quick Start

```bash
# Install dependencies
cd e2e
npm install

# Install browsers
npx playwright install

# Run tests (headless)
npm test

# Run tests with UI
npm run test:ui

# Run tests in headed mode (see browser)
npm run test:headed

# Debug tests
npm run test:debug

# View test report
npm run test:report
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
