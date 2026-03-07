import { TestServer } from './utils/test-server';

// Global server instances (one per worker process)
// These are shared across tests tagged as 'readOnly'
const globalServers = new Map<string, TestServer>();

// Global setup - runs once before all tests
export default async function globalSetup() {
  // We don't pre-start servers here because Playwright runs workers in separate processes
  // Instead, each worker will start its own server on first test
  return () => {
    globalServers.forEach(async (server, key) => {
      await server.stop();
    });
    globalServers.clear();
  };
}

// Export for use in fixtures
export { globalServers };
