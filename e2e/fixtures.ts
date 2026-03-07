import { test as base } from '@playwright/test';
import { TestServer } from './utils/test-server';
import * as path from 'path';
import * as fs from 'fs';

// Extend Playwright test with our fixtures
export const test = base.extend<{
  server: TestServer;
  testDbPath: string;
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

  // Test server instance
  server: async ({ testDbPath }, use) => {
    // Start server with pre-existing database (uses dynamic port)
    const server = new TestServer({
      databasePath: testDbPath,
    });
    await server.start();
    
    // Set base URL for Playwright
    process.env.DISCO_BASE_URL = server.getBaseUrl();

    await use(server);

    // Cleanup after tests
    await server.stop();
  },
});

export { expect } from '@playwright/test';
