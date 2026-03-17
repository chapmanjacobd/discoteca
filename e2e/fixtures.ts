import { test as base, expect, Page } from '@playwright/test';
import { TestServer, TestServerOptions } from './utils/test-server';
import { globalServers } from './global-setup';
import { MediaPage } from './pages/media-page';
import { SidebarPage } from './pages/sidebar-page';
import { ViewerPage } from './pages/viewer-page';
import * as path from 'path';
import * as fs from 'fs';

// Custom matchers
import {
  toHaveMediaCount,
  toBeInMode,
  toHaveJsonOutput,
  toHaveProgress,
  toBePlaying,
  toBePaused,
  toHaveDataAttribute,
  toHaveToast,
  toHaveNoErrorToast,
} from './utils/matchers';

// Extend expect with custom matchers
expect.extend({
  toHaveMediaCount,
  toBeInMode,
  toHaveJsonOutput,
  toHaveProgress,
  toBePlaying,
  toBePaused,
  toHaveDataAttribute,
  toHaveToast,
  toHaveNoErrorToast,
});

/**
 * Wait for media player to be ready
 */
export async function waitForPlayer(page: Page, timeout: number = 10000): Promise<void> {
  const player = page.locator('#pip-player');
  await player.waitFor({ state: 'visible', timeout });
  const media = page.locator('#pip-player video, #pip-player audio');
  await media.waitFor({ state: 'visible', timeout });
  await page.waitForFunction(() => {
    const media = document.querySelector('#pip-player video, #pip-player audio') as HTMLMediaElement;
    return media && media.readyState >= 1;
  }, { timeout });
}

/**
 * Check if player is open
 */
export async function isPlayerOpen(page: Page): Promise<boolean> {
  const player = page.locator('#pip-player, #player-container');
  if (await player.count() > 0) {
    return await player.first().isVisible();
  }
  const videoCount = await page.locator('video').count();
  const audioCount = await page.locator('audio').count();
  return videoCount > 0 || audioCount > 0;
}

// Test fixtures
export const test = base.extend<{
  server: TestServer;
  testDbPath: string;
  readOnly: boolean;
  serverOptions: TestServerOptions;
  mediaPage: MediaPage;
  sidebarPage: SidebarPage;
  viewerPage: ViewerPage;
}>({
  testDbPath: async ({}, use) => {
    const fixturesDir = path.join(__dirname, './fixtures');
    const dbPath = path.join(fixturesDir, 'test.db');
    if (!fs.existsSync(dbPath)) {
      throw new Error(`Test database not found at ${dbPath}. Run 'make e2e-init'.`);
    }
    await use(dbPath);
  },

  mediaPage: async ({ page }, use) => {
    await use(new MediaPage(page));
  },

  sidebarPage: async ({ page }, use) => {
    await use(new SidebarPage(page));
  },

  viewerPage: async ({ page }, use) => {
    await use(new ViewerPage(page));
  },

  readOnly: [false, { option: true }],

  serverOptions: async ({}, use) => {
    await use({});
  },

  server: async ({ testDbPath, readOnly, page, serverOptions }, use) => {
    const workerId = process.env.TEST_WORKER_INDEX || 'default';
    const project = process.env.PLAYWRIGHT_PROJECT || 'desktop';
    const serverKey = `${project}-${workerId}`;
    const tmpDir = path.join(__dirname, '../tmp');

    let server: TestServer;
    let tempDbPath: string | null = null;

    if (readOnly) {
      if (!globalServers.has(serverKey)) {
        server = new TestServer({ databasePath: testDbPath, readOnly: true, ...serverOptions });
        await server.start();
        globalServers.set(serverKey, server);
      } else {
        server = globalServers.get(serverKey)!;
      }
    } else {
      fs.mkdirSync(tmpDir, { recursive: true });
      tempDbPath = path.join(tmpDir, `test-${process.pid}-${Date.now()}.db`);
      fs.copyFileSync(testDbPath, tempDbPath);
      server = new TestServer({ databasePath: tempDbPath, readOnly: false, ...serverOptions });
      await server.start();
    }

    process.env.DISCO_BASE_URL = server.getBaseUrl();

    const url = new URL(server.getBaseUrl());
    await page.context().addCookies([{
      name: 'disco_token',
      value: 'e2e-test-token',
      domain: url.hostname,
      path: '/',
    }]);

    // page.on('console', msg => {
    //   const msgText = msg.text();
    //   if (msg.type() === 'error' && !['Failed to load resource: net::ERR_INCOMPLETE_CHUNKED_ENCODING'].includes(msgText)) {
    //     console.error(`console.error:`, msgText);
    //   } else {
    //     console.log(`console.log:`, msgText);
    //   }
    // });

    page.on('requestfailed', request => {
      const errorText = request.failure()?.errorText;
      if (errorText && !['net::ERR_ABORTED', 'net::ERR_INCOMPLETE_CHUNKED_ENCODING'].includes(errorText)) {
        console.error(`request.failed: ${request.url()} - ${errorText}`);
      }
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

    if (!readOnly) {
      await server.stop();
      if (tempDbPath && fs.existsSync(tempDbPath)) {
        try {
          fs.unlinkSync(tempDbPath);
          fs.unlinkSync(tempDbPath + '-wal');
        } catch (e) { /* ignore */ }
        try {
          fs.unlinkSync(tempDbPath + '-shm');
        } catch (e) { /* ignore */ }
      }
    }
  },
});

export { expect };
