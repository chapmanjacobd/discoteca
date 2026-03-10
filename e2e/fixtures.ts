import { test as base, expect, Page } from '@playwright/test';
import { TestServer } from './utils/test-server';
import { globalServers } from './global-setup';
import * as path from 'path';
import * as fs from 'fs';

/**
 * Wait for media player to be ready
 * Tries multiple selector patterns to handle different player implementations
 */
export async function waitForPlayer(page: Page, timeout: number = 10000): Promise<void> {
  try {
    // Try waiting for any player element
    await page.waitForSelector('#pip-player, #player-container, .player, video, audio', {
      timeout,
      state: 'visible'
    });
  } catch (e) {
    // If specific player not found, check if any media element exists
    const videoCount = await page.locator('video').count();
    const audioCount = await page.locator('audio').count();
    if (videoCount === 0 && audioCount === 0) {
      throw e;
    }
  }
}

/**
 * Check if player is open/visible
 */
export async function isPlayerOpen(page: Page): Promise<boolean> {
  const player = page.locator('#pip-player, #player-container');
  if (await player.count() > 0) {
    return await player.first().isVisible();
  }

  // Check for video/audio elements
  const videoCount = await page.locator('video').count();
  const audioCount = await page.locator('audio').count();
  return videoCount > 0 || audioCount > 0;
}

// Extended test fixture with server management
export const test = base.extend<{
  server: TestServer;
  testDbPath: string;
  readOnly: boolean;
}>({
  // Test database path (pre-committed to repo)
  testDbPath: async ({}, use) => {
    const fixturesDir = path.join(__dirname, './fixtures');
    const dbPath = path.join(fixturesDir, 'test.db');

    // Verify database exists
    if (!fs.existsSync(dbPath)) {
      throw new Error(`Test database not found at ${dbPath}. Run 'make e2e-init' to create it.`);
    }

    await use(dbPath);
  },

  // Whether this test is read-only (doesn't modify server state)
  readOnly: [false, { option: true }],

  // Test server instance - shared for readOnly tests, isolated for others
  server: async ({ testDbPath, readOnly, page }, use) => {
    const workerId = process.env.TEST_WORKER_INDEX || 'default';
    const project = process.env.PLAYWRIGHT_PROJECT || 'chromium';
    const serverKey = `${project}-${workerId}`;
    const tmpDir = path.join(__dirname, '../tmp');

    let server: TestServer;
    let tempDbPath: string | null = null;

    if (readOnly) {
      // Use shared global server for read-only tests
      if (!globalServers.has(serverKey)) {
        server = new TestServer({
          databasePath: testDbPath,
        });
        await server.start();
        globalServers.set(serverKey, server);
      } else {
        server = globalServers.get(serverKey)!;
      }
    } else {
      // Create isolated server with a temporary copy of the database for state-modifying tests
      // This ensures each test starts with a clean database state
      try {
        fs.mkdirSync(tmpDir, { recursive: true });
      } catch (e) {
        // Directory may already exist
      }
      tempDbPath = path.join(tmpDir, `test-${process.pid}-${Date.now()}.db`);
      fs.copyFileSync(testDbPath, tempDbPath);

      server = new TestServer({
        databasePath: tempDbPath,
      });
      await server.start();
    }

    // Set base URL for Playwright
    process.env.DISCO_BASE_URL = server.getBaseUrl();

    // Set authentication cookie before the test starts
    const url = new URL(server.getBaseUrl());
    await page.context().addCookies([{
      name: 'disco_token',
      value: 'e2e-test-token',
      domain: url.hostname,
      path: '/',
    }]);

    page.on('console', msg => {
      const msgText = msg.text();

      if (msg.type() === 'error') {
        if (!['Failed to load resource: net::ERR_INCOMPLETE_CHUNKED_ENCODING'].includes(msgText)) {
          console.error(`console.error:`, msgText);
        }
      } else {
        console.log(`console.log:`, msgText);
      }
    });

    page.on('requestfailed', request => {
      const errorText = request.failure()?.errorText
      if (errorText && ['net::ERR_ABORTED', 'net::ERR_INCOMPLETE_CHUNKED_ENCODING'].includes(errorText)) {
        return
      }

      console.error(`request.failed: ${request.url()} - ${errorText}`);
    });

    page.on('response', response => {
      if (response.status() >= 400) {
        console.error(`response.error: ${response.url()} - status ${response.status()}`);
      }
    });

    page.on('pageerror', err => {
      console.error(`page.error:`, err.message);
    });

    await use(server);

    // Cleanup: stop isolated servers and remove temporary database
    if (!readOnly) {
      await server.stop();
      // Remove temporary database copy
      if (tempDbPath && fs.existsSync(tempDbPath)) {
        try {
          fs.unlinkSync(tempDbPath);
          // Also remove WAL and SHM files if they exist
          fs.unlinkSync(tempDbPath + '-wal');
        } catch (e) {
          // Files may already be deleted
        }
        try {
          fs.unlinkSync(tempDbPath + '-shm');
        } catch (e) {
          // File may not exist
        }
      }
    }
  },
});

export { expect };
