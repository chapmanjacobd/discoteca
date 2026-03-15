import { test, expect } from '../fixtures';
import * as fs from 'fs';
import * as path from 'path';

/**
 * E2E tests to capture screenshots for README documentation
 * Run with: npx playwright test screenshots --project=chromium
 */
test.describe('README Screenshots', () => {
  test.use({
    readOnly: true,
    viewport: { width: 1280, height: 720 }
  });

  const screenshotsDir = path.join(__dirname, '../../docs/screenshots');

  test.beforeAll(async () => {
    // Create screenshots directory
    fs.mkdirSync(screenshotsDir, { recursive: true });
  });

  test('capture home page - details view', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Switch to details view
    await mediaPage.switchToDetailsView();
    await mediaPage.page.waitForTimeout(1000);

    // Take screenshot
    await mediaPage.page.screenshot({
      path: path.join(screenshotsDir, 'home-details-view.png'),
      fullPage: false,
      animations: 'disabled'
    });

    console.log('✓ Captured: home-details-view.png');
  });

  test('capture video player', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open first video
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    await videoCard.click();

    // Wait for player to be ready
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    // Wait for media to load
    await viewerPage.waitForMediaData();

    // Ensure video is playing
    const isPlaying = await viewerPage.isPlaying();
    if (!isPlaying) {
      await viewerPage.play();
    }

    // Let it play briefly
    await mediaPage.page.waitForTimeout(1000);

    // Take screenshot
    await mediaPage.page.screenshot({
      path: path.join(screenshotsDir, 'video-player.png'),
      fullPage: false,
      animations: 'disabled'
    });

    console.log('✓ Captured: video-player.png');

    // Close player
    await viewerPage.close();
  });

  test('capture audio player', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open first audio file
    const audioCard = mediaPage.getFirstMediaCardByType('audio');
    if (await audioCard.count() > 0) {
      await audioCard.click();

      // Wait for player
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(1000);

      // Take screenshot
      await mediaPage.page.screenshot({
        path: path.join(screenshotsDir, 'audio-player.png'),
        fullPage: false,
        animations: 'disabled'
      });

      console.log('✓ Captured: audio-player.png');

      // Close player
      await viewerPage.close();
    } else {
      console.log('⚠ No audio files found, skipping audio player screenshot');
    }
  });

  test('capture EPUB viewer', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only EPUB documents
    await mediaPage.search('.epub');

    // Find and open EPUB
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    if (await epubCard.count() > 0) {
      await epubCard.click();

      // Wait for document modal
      await viewerPage.waitForDocumentModal();

      // Wait for calibre conversion
      const frame = mediaPage.page.frameLocator('#document-container iframe');
      const tocHeader = frame.locator('.toc-header');
      await expect(tocHeader).toBeVisible({ timeout: 15000 });

      await mediaPage.page.waitForTimeout(500);

      // Take screenshot
      await mediaPage.page.screenshot({
        path: path.join(screenshotsDir, 'epub-viewer.png'),
        fullPage: false,
        animations: 'disabled'
      });

      console.log('✓ Captured: epub-viewer.png');

      // Close modal
      await viewerPage.closeDocumentModal();
    } else {
      console.log('⚠ No EPUB files found, skipping EPUB viewer screenshot');
    }
  });

  test('capture search functionality', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Perform a search
    await mediaPage.search('test');
    await mediaPage.page.waitForTimeout(500);

    // Take screenshot
    await mediaPage.page.screenshot({
      path: path.join(screenshotsDir, 'search-results.png'),
      fullPage: false,
      animations: 'disabled'
    });

    console.log('✓ Captured: search-results.png');

    // Clear search
    await mediaPage.clearSearch();
  });

  test('capture settings modal', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings
    await sidebarPage.openSettings();
    await mediaPage.page.waitForTimeout(500);

    // Take screenshot
    await mediaPage.page.screenshot({
      path: path.join(screenshotsDir, 'settings-modal.png'),
      fullPage: false,
      animations: 'disabled'
    });

    console.log('✓ Captured: settings-modal.png');

    // Close settings
    await sidebarPage.closeSettings();
  });

  test('capture disk usage view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Navigate to disk usage view
    await sidebarPage.openDiskUsage();
    await mediaPage.page.waitForTimeout(2000);

    // Take screenshot
    await mediaPage.page.screenshot({
      path: path.join(screenshotsDir, 'disk-usage-view.png'),
      fullPage: false,
      animations: 'disabled'
    });

    console.log('✓ Captured: disk-usage-view.png');
  });

  test('capture group view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Switch to group view
    if (await mediaPage.viewGroupButton.count() > 0) {
      await mediaPage.viewGroupButton.click();
      await mediaPage.page.waitForTimeout(2000);

      // Take screenshot
      await mediaPage.page.screenshot({
        path: path.join(screenshotsDir, 'group-view.png'),
        fullPage: false,
        animations: 'disabled'
      });

      console.log('✓ Captured: group-view.png');
    } else {
      console.log('⚠ Group view not available, skipping screenshot');
    }
  });
});
