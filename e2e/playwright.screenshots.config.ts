import { defineConfig, devices } from '@playwright/test';
import path from 'path';

/**
 * Playwright config for screenshot tests
 * Run with: npx playwright test -c playwright.screenshots.config.ts
 */
export default defineConfig({
  testDir: './screenshots',
  fullyParallel: false,
  workers: 1,
  maxFailures: 10,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,

  reporter: [
    ['html', { open: 'never' }],
    ['list'],
  ],

  use: {
    baseURL: process.env.DISCO_BASE_URL || 'http://localhost:8080',
    viewport: { width: 1280, height: 720 },
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10000,
    contextOptions: {
      reducedMotion: 'reduce',
    },
  },

  projects: [
    {
      name: 'desktop',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: [
            '--mute-audio',
            '--autoplay-policy=user-gesture-required',
            '--disable-background-media-suspend',
          ],
          headless: true,
        },
        contextOptions: {
          permissions: [],
        },
      },
    },
  ],

  outputDir: path.join(__dirname, 'test-results', 'screenshots'),
  globalSetup: require.resolve('./global-setup'),
});
