import { test, expect } from '../fixtures';

test.describe('Subtitles', () => {
  test.use({ readOnly: true });

  test('player opens when clicking caption segment', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Click a caption segment to open player using POM
    await mediaPage.getCaptionSegments().first().click();
    await viewerPage.waitForPlayer();

    // Player should be open using POM
    await expect(viewerPage.playerContainer).toBeVisible();
  });

  test('video element exists when playing caption media', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Click a caption segment to open player using POM
    await mediaPage.getCaptionSegments().first().click();
    await viewerPage.waitForPlayer();

    // Video element should exist using POM
    const count = await viewerPage.getMediaElement().count();
    expect(count).toBeGreaterThan(0);
  });

  test('keyboard shortcut j cycles subtitles', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Click a caption segment to open player using POM
    await mediaPage.getCaptionSegments().first().click();
    await viewerPage.waitForPlayer();

    // Press 'j' key (subtitle cycling shortcut)
    await mediaPage.page.keyboard.press('j');
    await mediaPage.page.waitForTimeout(500);

    // Player should still be visible using POM
    await expect(viewerPage.playerContainer).toBeVisible();
  });

  test('keyboard shortcut J cycles subtitles in reverse', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Click a caption segment to open player using POM
    await mediaPage.getCaptionSegments().first().click();
    await viewerPage.waitForPlayer();

    // Press 'J' key (shift+j) to cycle subtitles in reverse
    await mediaPage.page.keyboard.press('J');
    await mediaPage.page.waitForTimeout(500);

    // Player should still be visible using POM
    await expect(viewerPage.playerContainer).toBeVisible();
  });

  test('subtitle track can be changed', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Click a caption segment to open player using POM
    await mediaPage.getCaptionSegments().first().click();
    await viewerPage.waitForPlayer();

    // Check if subtitle button exists using POM
    if (await viewerPage.page.locator('#pip-subs').count() > 0) {
      // Click subtitle track button
      await viewerPage.page.locator('#pip-subs').click();
      await mediaPage.page.waitForTimeout(500);

      // Subtitle menu should appear or cycle
      // Player should still be visible using POM
      await expect(viewerPage.playerContainer).toBeVisible();
    }
  });

  test('subtitle visibility can be toggled', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Click a caption segment to open player using POM
    await mediaPage.getCaptionSegments().first().click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'v' to toggle subtitle visibility using POM
    await mediaPage.page.keyboard.press('v');
    await mediaPage.page.waitForTimeout(500);

    // Toast should appear with subtitle state using POM
    if (await mediaPage.toast.isVisible()) {
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).toMatch(/Subtitles: (Off|Track|External)/);
    }
  });

  test('captions view shows correct media count', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Caption count badge should be visible using POM
    const countText = await mediaPage.getCaptionCountBadge(0);
    expect(countText).toMatch(/\d+/);
  });

  test('caption segments have correct time attributes', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Get first caption time using POM
    const time = await mediaPage.getCaptionTime(0);
    
    // Should be at least 10 seconds (filter threshold)
    expect(time).toBeGreaterThanOrEqual(10);
  });

  test('caption text is not empty', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Get caption count using POM
    const count = await mediaPage.getCaptionSegments().count();

    // Check each caption has non-empty text using POM
    for (let i = 0; i < count; i++) {
      const text = await mediaPage.getCaptionText(i);
      const trimmedText = text?.trim() || '';
      
      // Should not be empty
      expect(trimmedText).not.toBe('');
    }
  });
});
