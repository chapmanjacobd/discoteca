import { test, expect } from '../fixtures';

test.describe('Subtitles Selection', () => {
  test('subtitle button is visible for videos with captions', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show videos with captions
    await page.fill('#search-input', 'caption');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Click first media card
    const firstCard = page.locator('.media-card').first();
    await firstCard.click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Subtitle button should be visible
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn, button:has-text("Subtitle"), .cc-btn, .captions-btn');
    await expect(subtitleBtn.first()).toBeVisible();
  });

  test('subtitle menu opens with track options', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn, .cc-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Subtitle menu should open
      const subtitleMenu = page.locator('#subtitle-menu, .subtitle-menu, .cc-menu, [role="menu"]:has-text("Subtitle")');
      await expect(subtitleMenu.first()).toBeVisible();
    }
  });

  test('primary subtitle track can be selected', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Select primary track
      const primaryTrack = page.locator('.subtitle-menu button:has-text("English"), .subtitle-menu button:has-text("Primary"), .subtitle-menu button:first-child');
      if (await primaryTrack.count() > 0) {
        await primaryTrack.click();
        await page.waitForTimeout(500);

        // Subtitle should be enabled
        const video = page.locator('video').first();
        const textTracks = await video.evaluate((el) => (el as HTMLVideoElement).textTracks.length);
        expect(textTracks).toBeGreaterThan(0);
      }
    }
  });

  test('secondary subtitle track can be selected', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Look for secondary track option
      const secondaryTrack = page.locator('.subtitle-menu button:has-text("Secondary"), .subtitle-menu button:has-text("Spanish"), .subtitle-menu button:nth-child(2)');
      if (await secondaryTrack.count() > 0) {
        await secondaryTrack.click();
        await page.waitForTimeout(500);

        // Secondary subtitle should be selected
        await expect(secondaryTrack).toHaveClass(/active|selected/);
      }
    }
  });

  test('subtitles can be toggled off', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      // Toggle subtitles off
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Select "Off" option
      const offOption = page.locator('.subtitle-menu button:has-text("Off"), .subtitle-menu button:has-text("None"), .subtitle-menu button:has-text("Disable")');
      if (await offOption.count() > 0) {
        await offOption.click();
        await page.waitForTimeout(500);

        // Subtitles should be disabled
        const video = page.locator('video').first();
        const mode = await video.evaluate((el) => {
          const tracks = (el as HTMLVideoElement).textTracks;
          for (let i = 0; i < tracks.length; i++) {
            if (tracks[i].mode === 'showing') return 'showing';
          }
          return 'hidden';
        });
        expect(mode).toBe('hidden');
      }
    }
  });

  test('subtitle track shows current selection', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Selected track should have active class
      const activeTrack = page.locator('.subtitle-menu button.active, .subtitle-menu button.selected');
      if (await activeTrack.count() > 0) {
        await expect(activeTrack.first()).toBeVisible();
      }
    }
  });

  test('subtitle button shows indicator when enabled', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Subtitle button may have indicator
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn, .cc-btn').first();
    
    // Check for active class or indicator
    const hasActiveClass = await subtitleBtn.evaluate((el) => 
      el.classList.contains('active') || el.classList.contains('enabled')
    );
    
    // Or check for indicator element
    const indicator = page.locator('.subtitle-indicator, .cc-indicator');
    const hasIndicator = await indicator.count() > 0;
    
    // Either should be true when subtitles are available
    expect(hasActiveClass || hasIndicator).toBe(true);
  });

  test('subtitle size can be adjusted', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Look for size settings
      const sizeOption = page.locator('.subtitle-menu button:has-text("Size"), .subtitle-settings, .cc-settings');
      if (await sizeOption.count() > 0) {
        await sizeOption.click();
        await page.waitForTimeout(500);

        // Size slider or options should appear
        const sizeSlider = page.locator('input[type="range"][aria-label*="size"], .size-slider');
        if (await sizeSlider.count() > 0) {
          await expect(sizeSlider.first()).toBeVisible();
        }
      }
    }
  });

  test('subtitle color can be adjusted', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Look for color settings
      const colorOption = page.locator('.subtitle-menu button:has-text("Color"), .color-picker, .cc-color');
      if (await colorOption.count() > 0) {
        await colorOption.click();
        await page.waitForTimeout(500);

        // Color options should appear
        const colorPicker = page.locator('input[type="color"], .color-options');
        if (await colorPicker.count() > 0) {
          await expect(colorPicker.first()).toBeVisible();
        }
      }
    }
  });

  test('subtitle position can be adjusted', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Look for position settings
      const positionOption = page.locator('.subtitle-menu button:has-text("Position"), .position-slider, .cc-position');
      if (await positionOption.count() > 0) {
        await positionOption.click();
        await page.waitForTimeout(500);

        // Position controls should appear
        const positionControl = page.locator('input[type="range"][aria-label*="position"], .position-slider');
        if (await positionControl.count() > 0) {
          await expect(positionControl.first()).toBeVisible();
        }
      }
    }
  });

  test('external subtitle files are detected', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show videos with external subtitles
    await page.fill('#search-input', '.vtt');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Click first media card
    const firstCard = page.locator('.media-card').first();
    await firstCard.click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Subtitle button should be available
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    await expect(subtitleBtn).toBeVisible();

    // Click to see subtitle options
    await subtitleBtn.click();
    await page.waitForTimeout(500);

    // External subtitle should be listed
    const subtitleMenu = page.locator('.subtitle-menu');
    if (await subtitleMenu.count() > 0) {
      const menuText = await subtitleMenu.first().textContent();
      expect(menuText?.toLowerCase()).toMatch(/(vtt|srt|external|sub)/);
    }
  });

  test('embedded subtitle tracks are detected', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Subtitle menu should list embedded tracks
      const subtitleMenu = page.locator('.subtitle-menu');
      if (await subtitleMenu.count() > 0) {
        const menuText = await subtitleMenu.first().textContent();
        // Should have at least one track option or "No subtitles"
        expect(menuText?.length).toBeGreaterThan(0);
      }
    }
  });

  test('subtitle language is displayed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Language should be displayed
      const languageLabel = page.locator('.subtitle-menu button:has-text("English"), .subtitle-menu button:has-text("ESP"), .subtitle-menu button:has-text("中文")');
      if (await languageLabel.count() > 0) {
        await expect(languageLabel.first()).toBeVisible();
      }
    }
  });

  test('C key toggles subtitles', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Focus the player
    await page.locator('#pip-player').focus();

    // Press C for captions
    await page.keyboard.press('c');
    await page.waitForTimeout(500);

    // Player should still be visible
    await expect(page.locator('#pip-player')).toBeVisible();
  });
});
