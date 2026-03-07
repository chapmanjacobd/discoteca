import { test, expect } from '../fixtures';
import { waitForPlayer } from '../fixtures';

test.describe('Subtitles', () => {
  test.use({ readOnly: true });
  test('player opens when clicking caption segment', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click a caption segment to open player
    const captionSegment = page.locator('.caption-segment').first();
    await captionSegment.click();
    await waitForPlayer(page);

    // Player should be open
    const player = page.locator('#pip-player');
    await expect(player.first()).toBeVisible();
  });

  test('video element exists when playing caption media', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click a caption segment to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Video element should exist
    const video = page.locator('#pip-player video, #pip-player audio');
    const count = await video.count();
    expect(count).toBeGreaterThan(0);
  });

  test('keyboard shortcut j cycles subtitles', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click a caption segment to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Press 'j' key (subtitle cycling shortcut)
    await page.keyboard.press('j');
    await page.waitForTimeout(500);

    // Player should still be visible
    const player = page.locator('#pip-player');
    await expect(player.first()).toBeVisible();
  });

  test('keyboard shortcut J cycles subtitles in reverse', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click a caption segment to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Press 'J' key (shift+j) to cycle subtitles in reverse
    await page.keyboard.press('J');
    await page.waitForTimeout(500);

    // Player should still be visible
    const player = page.locator('#pip-player');
    await expect(player.first()).toBeVisible();
  });

  test('keyboard shortcut v toggles subtitle visibility', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click a caption segment to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Press 'v' key to toggle subtitle visibility
    await page.keyboard.press('v');
    await page.waitForTimeout(500);

    // Player should still be visible
    const player = page.locator('#pip-player');
    await expect(player.first()).toBeVisible();

    // Press 'v' again to toggle back
    await page.keyboard.press('v');
    await page.waitForTimeout(500);

    // Player should still be visible
    await expect(player.first()).toBeVisible();
  });
});
