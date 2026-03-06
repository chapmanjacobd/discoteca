import { test as base } from '@playwright/test';
import { TestServer, startGlobalServer, stopGlobalServer } from './utils/test-server';
import { seedDatabase } from './utils/seed-db';
import * as path from 'path';
import * as fs from 'fs';

// Extend Playwright test with our fixtures
export const test = base.extend<{
  server: TestServer;
  testDbPath: string;
}>({
  // Test database path
  testDbPath: async ({}, use) => {
    const fixturesDir = path.join(__dirname, '../fixtures');
    if (!fs.existsSync(fixturesDir)) {
      fs.mkdirSync(fixturesDir, { recursive: true });
    }
    const dbPath = path.join(fixturesDir, 'test.db');
    await use(dbPath);
  },

  // Test server instance
  server: async ({ testDbPath }, use) => {
    // Seed database before starting server
    await seedDatabase({ databasePath: testDbPath, clean: true });

    // Start server
    const server = new TestServer({
      databasePath: testDbPath,
      port: 8080,
    });
    await server.start();

    await use(server);

    // Cleanup after tests
    await server.stop();
  },
});

export { expect } from '@playwright/test';
