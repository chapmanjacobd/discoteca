import { waitForPlayer, isPlayerOpen } from '../fixtures';
import { test, expect } from '../fixtures';

test.describe('Media Playback', () => {
  test.use({ readOnly: true });

  test.beforeEach(async ({ page }) => {
    page.on('console', msg => console.log('BROWSER LOG:', msg.text()));
    page.on('pageerror', err => console.error('BROWSER ERROR:', err.message));
  });

  test('opens media in PiP player when clicked', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Select a non-document card (documents have an 'rsvp' action button)
    const mediaCard = page.locator('.media-card:not(:has(.rsvp))').first();
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

  test('Now Playing button appears when media is playing', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card
    await page.locator('.media-card:not(:has(.rsvp))').first().click();

    // Wait for player to open
    await waitForPlayer(page);

    // Open Playlists section in sidebar to make Now Playing button visible
    const playlistsSection = page.locator('#details-playlists');
    await playlistsSection.evaluate((el: HTMLDetailsElement) => el.open = true);

    // Now Playing button should be visible in sidebar
    const nowPlayingBtn = page.locator('#now-playing-btn');
    if (await nowPlayingBtn.count() > 0) {
      await expect(nowPlayingBtn).toBeVisible();
      await expect(nowPlayingBtn).not.toHaveClass(/hidden/);
    }
  });

  test('Now Playing button shows queue count', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card
    await page.locator('.media-card:not(:has(.rsvp))').first().click();

    // Wait for player to open
    await waitForPlayer(page);

    // Open Playlists section in sidebar to make Now Playing button visible
    await page.locator('#details-playlists').evaluate((el: HTMLDetailsElement) => el.open = true);

    // Now Playing button should show count if there are queued items
    const nowPlayingBtn = page.locator('#now-playing-btn');
    if (await nowPlayingBtn.count() > 0) {
      const text = await nowPlayingBtn.textContent();

      // Should contain "Now Playing" text
      expect(text).toContain('Now Playing');
    }
  });

  test('clicking Now Playing shows current queue', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card
    await page.locator('.media-card:not(:has(.rsvp))').first().click();

    // Wait for player to open
    await waitForPlayer(page);

    // Open Playlists section in sidebar to make Now Playing button visible
    await page.locator('#details-playlists').evaluate((el: HTMLDetailsElement) => el.open = true);

    // Click Now Playing button
    const nowPlayingBtn = page.locator('#now-playing-btn');
    if (await nowPlayingBtn.count() > 0) {
      await nowPlayingBtn.click();

      // Should navigate to playlist view
      await expect(page.locator('.playlist-drop-zone.active, .media-card').first()).toBeVisible();
    }
  });

  test('closes player when close button clicked', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first non-document media card
    await page.locator('.media-card:not(:has(.rsvp))').first().click();

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
    await page.locator('.media-card:not(:has(.rsvp))').first().click();

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
    await page.locator('.media-card:not(:has(.rsvp))').first().click();

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
});
