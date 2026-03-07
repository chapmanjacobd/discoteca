import { defineConfig, devices } from '@playwright/test';

/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
// require('dotenv').config();

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: './tests',
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,
  maxFailures: 10,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['html', { open: 'never' }],
    ['list'],
    ['json', { outputFile: 'test-results/results.json' }]
  ],
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: process.env.DISCO_BASE_URL || 'http://localhost:8080',
    viewport: { width: 1280, height: 720 },

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',

    /* Screenshot on failure */
    screenshot: 'only-on-failure',

    /* Video on failure */
    video: 'retain-on-failure',

    /* Maximum time each action can take */
    actionTimeout: 10000,

    /* Maximum time each test can take */
    timeout: 60000,

    /* Mute audio for all browsers */
    contextOptions: {
      reducedMotion: 'reduce',
    },
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'firefox',
      use: {
        ...devices['Desktop Firefox'],
        // Mute audio and configure for testing
        launchOptions: {
          firefoxUserPrefs: {
            'media.volume_scale': '0.0', // Mute all audio
            'media.autoplay.default': 5, // Allow autoplay with user gesture
            'media.autoplay.blocking_policy': 0, // Don't block autoplay
            'media.block-autoplay-until-in-foreground': false, // Allow autoplay
            'dom.disable_open_during_load': false, // Allow popups
            'privacy.trackingprotection.enabled': false, // Disable tracking protection
            'browser.safebrowsing.malware.enabled': false, // Disable safe browsing
            'browser.safebrowsing.phishing.enabled': false, // Disable phishing protection
          },
          headless: true, // Run in headless mode
        },
      },
    },

    /* Temporarily disabled: Mobile and Safari tests */
    // {
    //   name: 'Mobile Chrome',
    //   use: {
    //     ...devices['Pixel 5'],
    //     // Mute audio for Chrome
    //     launchOptions: {
    //       args: [
    //         '--mute-audio',
    //         '--autoplay-policy=user-gesture-required',
    //         '--disable-background-media-suspend',
    //       ],
    //       headless: true,
    //     },
    //     contextOptions: {
    //       permissions: [],
    //     },
    //   },
    // },

    {
      name: 'webkit',
      use: {
        ...devices['Desktop Safari'],
        // Mute audio for Safari/WebKit
        launchOptions: {
          headless: true,
        },
        contextOptions: {
          // WebKit doesn't have a direct mute option, but we can reduce motion
          reducedMotion: 'reduce',
        },
      },
    },

    /* Test against branded browsers. */
    // {
    //   name: 'Microsoft Edge',
    //   use: { ...devices['Desktop Edge'], channel: 'msedge' },
    // },
    // {
    //   name: 'Google Chrome',
    //   use: { ...devices['Desktop Chrome'], channel: 'chrome' },
    // },
  ],

  /* Ignore CLI tests temporarily */
  testIgnore: /cli-.*\.spec\.ts/,

  /* Folder for test artifacts such as screenshots, videos, traces, etc. */
  outputDir: 'test-results/',

  /* Global setup and teardown */
  globalSetup: require.resolve('./global-setup'),
});
