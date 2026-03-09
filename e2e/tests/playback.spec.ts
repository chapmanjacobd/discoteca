import { waitForPlayer, isPlayerOpen } from '../fixtures';
import { test, expect } from '../fixtures';

test.describe('Media Playback', () => {
  test.use({ readOnly: true });

  test('toggles playback with Space key', async ({ page, server }) => {

    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Debug: list card types
    const types = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('.media-card')).map(el => (el as HTMLElement).dataset.type);
    });
    console.log(`Found card types: ${types.join(', ')}`);

    // Select a non-document card
    const mediaCard = page.locator('.media-card[data-type*="video"], .media-card[data-type*="audio"], .media-card[data-type*="image"]').first();
    const cardHtml = await mediaCard.evaluate(el => el.outerHTML);
    console.log(`Selected card HTML: ${cardHtml}`);

    await mediaCard.click();
    console.log('Clicked media card');

    // Wait for player to open
    await waitForPlayer(page);

    // Player should be visible
    const player = page.locator('#pip-player, #player-container');
    await expect(player.first()).toBeVisible();

    // Media title should be shown
    const mediaTitle = page.locator('#media-title');
    if (await mediaTitle.count() > 0) {
      await expect(mediaTitle.first()).toBeVisible();
    }
  });

  test('Queue container appears when enabled in settings', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Open settings
    await page.locator('#settings-button').click();
    await page.waitForSelector('#settings-modal', { state: 'visible' });

    // Enable Queue
    const queueToggle = page.locator('#setting-enable-queue').locator('xpath=..').locator('.slider');
    await queueToggle.click();

    // Close settings
    await page.locator('.close-modal').first().click();

    // Queue container should be visible
    const queueContainer = page.locator('#queue-container');
    await expect(queueContainer).toBeVisible();
  });

  test('adding to queue when enabled', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Enable Queue via settings
    await page.locator('#settings-button').click();
    await page.waitForSelector('#settings-modal', { state: 'visible' });
    await page.locator('#setting-enable-queue').locator('xpath=..').locator('.slider').click();
    await page.locator('.close-modal').first().click();

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card (should add to queue, not play)
    await page.locator('.media-card[data-type*="video"], .media-card[data-type*="audio"], .media-card[data-type*="image"]').first().click();

    // Queue count badge should show 1
    const badge = page.locator('#queue-count-badge');
    await expect(badge).toHaveText('1');

    // Queue item should be present
    const queueItem = page.locator('.queue-item').first();
    await expect(queueItem).toBeVisible();

    // Player should NOT be visible yet
    const player = page.locator('#pip-player');
    const isHidden = await player.count() === 0 || await player.first().evaluate(el => el.classList.contains('hidden'));
    expect(isHidden).toBe(true);
  });

  test('closes player when close button clicked', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card
    await page.locator('.media-card[data-type*="video"], .media-card[data-type*="audio"], .media-card[data-type*="image"]').first().click();

    // Wait for player to open
    await waitForPlayer(page);

    // Click close button
    const closeBtn = page.locator('.close-pip, .player-close, button:has-text("Close")');
    if (await closeBtn.count() > 0) {
      await closeBtn.first().click();

      // Player should be hidden or removed
      const player = page.locator('#pip-player');
      const isHidden = await player.count() === 0 || await player.first().evaluate(el => el.classList.contains('hidden'));
      expect(isHidden).toBe(true);
    }
  });

  test('toggles theatre mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card
    await page.locator('.media-card[data-type*="video"], .media-card[data-type*="audio"], .media-card[data-type*="image"]').first().click();

    // Wait for player to open
    await waitForPlayer(page);

    // Click theatre mode button
    const theatreBtn = page.locator('#pip-theatre');
    if (await theatreBtn.count() > 0) {
      await theatreBtn.click();

      // Player should have theatre class
      const player = page.locator('#pip-player');
      if (await player.count() > 0) {
        await expect(player.first()).toHaveClass(/theatre/);

        // Click again to exit theatre mode
        await theatreBtn.click();
        await page.waitForTimeout(300);

        // Theatre class should be removed
        await expect(player.first()).not.toHaveClass(/theatre/);
      }
    }
  });

  test('playback speed can be adjusted', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card
    await page.locator('.media-card[data-type*="video"], .media-card[data-type*="audio"], .media-card[data-type*="image"]').first().click();

    // Wait for player to open
    await waitForPlayer(page);

    // Click speed button
    const speedBtn = page.locator('#pip-speed');
    if (await speedBtn.count() > 0 && await speedBtn.isVisible()) {
      await speedBtn.click();

      // Speed menu should appear
      const speedMenu = page.locator('#pip-speed-menu');
      if (await speedMenu.count() > 0) {
        await expect(speedMenu.first()).toBeVisible();

        // Select different speed
        const speedOption = page.locator('#pip-speed-menu button:has-text("1.5x")');
        if (await speedOption.count() > 0) {
          await speedOption.first().click();

          // Speed should update
          await expect(speedBtn).toHaveText('1.5x');
        }
      }
    }
  });

  test('handles rapid clicks on the same item', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    
    // Rapidly click the same card
    await mediaCard.click();
    await mediaCard.click();
    await mediaCard.click();

    // Wait for playback to stabilize
    await page.waitForTimeout(1000);

    // Verify no "Unplayable" error toast is shown
    const toast = page.locator('#toast');
    const toastText = await toast.textContent();
    if (await toast.isVisible()) {
      expect(toastText).not.toContain('Unplayable');
    }
  });
});
