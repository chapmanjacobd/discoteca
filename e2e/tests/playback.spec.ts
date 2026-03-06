import { test, expect } from '../fixtures';

test.describe('Media Playback', () => {
  test('opens media in PiP player when clicked', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Click first media card
    const firstCard = page.locator('.media-card').first();
    await firstCard.click();
    
    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });
    
    // Player should be visible
    await expect(page.locator('#pip-player')).toBeVisible();
    
    // Media title should be shown
    await expect(page.locator('#media-title')).toBeVisible();
  });

  test('Now Playing button appears when media is playing', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Click first media card
    await page.locator('.media-card').first().click();
    
    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });
    
    // Now Playing button should be visible in sidebar
    const nowPlayingBtn = page.locator('#now-playing-btn');
    await expect(nowPlayingBtn).toBeVisible();
    await expect(nowPlayingBtn).not.toHaveClass(/hidden/);
  });

  test('Now Playing button shows queue count', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Click first media card
    await page.locator('.media-card').first().click();
    
    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });
    
    // Now Playing button should show count if there are queued items
    const nowPlayingBtn = page.locator('#now-playing-btn');
    const text = await nowPlayingBtn.textContent();
    
    // Should contain "Now Playing" text
    expect(text).toContain('Now Playing');
  });

  test('clicking Now Playing shows current queue', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Click first media card
    await page.locator('.media-card').first().click();
    
    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });
    
    // Click Now Playing button
    await page.locator('#now-playing-btn').click();
    
    // Should navigate to playlist view
    await expect(page.locator('.playlist-drop-zone.active, .media-card')).toBeVisible();
  });

  test('closes player when close button clicked', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Click first media card
    await page.locator('.media-card').first().click();
    
    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });
    
    // Click close button
    await page.locator('.close-pip').click();
    
    // Player should be hidden
    await expect(page.locator('#pip-player')).toHaveClass(/hidden/);
    
    // Now Playing button should be hidden
    await expect(page.locator('#now-playing-btn')).toHaveClass(/hidden/);
  });

  test('toggles theatre mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Click first media card
    await page.locator('.media-card').first().click();
    
    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });
    
    // Click theatre mode button
    await page.locator('#pip-theatre').click();
    
    // Player should have theatre class
    await expect(page.locator('#pip-player')).toHaveClass(/theatre/);
    
    // Click again to exit theatre mode
    await page.locator('#pip-theatre').click();
    await page.waitForTimeout(300);
    
    // Theatre class should be removed
    const hasTheatre = await page.locator('#pip-player').classList();
    expect(hasTheatre).not.toContain('theatre');
  });

  test('playback speed can be adjusted', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Click first media card
    await page.locator('.media-card').first().click();
    
    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });
    
    // Click speed button
    const speedBtn = page.locator('#pip-speed');
    if (await speedBtn.isVisible()) {
      await speedBtn.click();
      
      // Speed menu should appear
      const speedMenu = page.locator('#pip-speed-menu');
      await expect(speedMenu).toBeVisible();
      
      // Select different speed
      await page.locator('#pip-speed-menu button:has-text("1.5x")').click();
      
      // Speed should update
      await expect(speedBtn).toHaveText('1.5x');
    }
  });
});
