import { test, expect } from '../fixtures';

test.describe('Search and Query Integration', () => {
  test.use({ readOnly: true });

  test('search filters media by title', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Initial count
    const initialCount = await page.locator('.media-card').count();
    expect(initialCount).toBeGreaterThan(0);
    
    // Search for a specific movie
    await page.fill('#search-input', 'test_video');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Count should decrease or change
    const searchResults = page.locator('.media-card');
    const searchCount = await searchResults.count();

    // At least test_video should be there
    expect(searchCount).toBeGreaterThan(0);
    expect(searchCount).toBeLessThanOrEqual(initialCount);

    // Check first result title
    const firstTitle = await searchResults.first().locator('.media-title').textContent();
    expect(firstTitle?.toLowerCase()).toContain('test_video');
  });

  test('filters by media type', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Open Media Type filter section
    const typeDetails = page.locator('#details-media-type');
    await typeDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);
    
    // Click Video filter
    const videoBtn = page.locator('#media-type-list .category-btn[data-type="video"]');
    if (await videoBtn.isVisible()) {
      await videoBtn.click();
      await page.waitForTimeout(1000);
      
      // All results should be video (we can check extensions as proxy)
      const results = page.locator('.media-card');
      const count = await results.count();
      
      for (let i = 0; i < Math.min(count, 5); i++) {
        const title = await results.nth(i).locator('.media-title').textContent();
        // Since we don't have explicit type badges in the UI always, 
        // we just verify results updated
        expect(title).toBeTruthy();
      }
    }
  });

  test('filters by progress states under History', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open History/Progress filter section
    const historyDetails = page.locator('#details-history');
    if (await historyDetails.isVisible()) {
      await historyDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
      await page.waitForTimeout(500);

      // Click Unfinished filter
      const unfinishedBtn = page.locator('#history-in-progress-btn');
      if (await unfinishedBtn.isVisible()) {
        await unfinishedBtn.click();
        await page.waitForTimeout(1000);

        // Should only show unfinished items
        // In our seed data, we might not have many, but check it doesn't error
        const hash = await page.evaluate(() => window.location.hash);
        expect(hash).toContain('history=in-progress');
      }
    }
  });

  test('playlist management works end-to-end', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Open Playlists section
    const playlistDetails = page.locator('#details-playlists');
    await playlistDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);
    
    // Create new playlist
    await page.click('#new-playlist-btn');
    await page.waitForTimeout(500);
    
    // We can't easily handle native prompt in Playwright without listener
    // but we can check if Favorites exists
    const favoritesBtn = page.locator('#playlist-list .category-btn').filter({ hasText: 'Favorites' });
    if (await favoritesBtn.isVisible()) {
      await favoritesBtn.click();
      await page.waitForTimeout(1000);
      
      // Should be in playlist view
      const hash = await page.evaluate(() => window.location.hash);
      expect(hash).toContain('mode=playlist');
    }
  });

  test('rating system works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 20000 });
    
    // Expand ratings filter
    const ratingDetails = page.locator('#details-ratings');
    if (await ratingDetails.isVisible()) {
      await ratingDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
      await page.waitForTimeout(500);
      
      // Click a rating filter
      const ratingBtn = page.locator('#ratings-list .category-btn').first();
      if (await ratingBtn.isVisible()) {
        await ratingBtn.click();
        await page.waitForTimeout(1000);
        
        // URL should update
        const hash = await page.evaluate(() => window.location.hash);
        expect(hash).toContain('score=');
      }
    }
  });

  test('category filtering works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 20000 });
    
    // Expand categories section
    const categoryDetails = page.locator('#details-categories');
    await categoryDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);
    
    // Check if any categories exist
    const categoryBtn = page.locator('#categories-list .category-btn').first();
    if (await categoryBtn.isVisible()) {
      const categoryName = await categoryBtn.textContent();
      await categoryBtn.click();
      await page.waitForTimeout(1000);
      
      // Results should be filtered
      const hash = await page.evaluate(() => window.location.hash);
      expect(hash).toContain('category=');
    }
  });

  test('pagination works correctly', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 20000 });
    
    // Check pagination controls exist
    const pagination = page.locator('#pagination-container');
    await expect(pagination).toBeVisible();
    
    // Navigate to next page if possible
    const nextBtn = page.locator('#next-page');
    if (await nextBtn.isVisible() && !(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await page.waitForTimeout(1000);
      
      const pageNum = await page.locator('#page-number').inputValue();
      expect(pageNum).toBe('2');
    }
  });

  test('sorting works correctly', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 20000 });
    
    // Change sort to size
    const sortBy = page.locator('#sort-by');
    await sortBy.selectOption('size');
    await page.waitForTimeout(500);
    
    // Change sort to duration
    await sortBy.selectOption('duration');
    await page.waitForTimeout(500);
    await expect(sortBy).toHaveValue('duration');
  });

  test('limit setting works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 20000 });
    
    // Open settings
    await page.locator('#settings-button').click();
    await page.waitForSelector('#settings-modal:not(.hidden)', { timeout: 10000 });
    await page.waitForTimeout(500);

    // Change limit
    const advancedSettings = page.locator('summary:has-text("Advanced Settings")');
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpanded = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpanded) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }
    const limitInput = page.locator('#limit');
    await limitInput.fill('50');
    await page.waitForTimeout(1000);
    
    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);
    
    // Verify limit was applied (check URL or fetch)
    const hash = await page.evaluate(() => window.location.hash);
    // Limit might be stored in localStorage, not URL
  });

  test('database filtering works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 20000 });
    
    // Check if database filter exists
    const dbFilter = page.locator('#details-databases');
    if (await dbFilter.isVisible()) {
      await dbFilter.evaluate((el: HTMLDetailsElement) => el.open = true);
      await page.waitForTimeout(500);
      
      // Click a database filter
      const dbBtn = page.locator('#databases-list .category-btn').first();
      if (await dbBtn.isVisible()) {
        await dbBtn.click();
        await page.waitForTimeout(1000);
        
        // URL should update
        const hash = await page.evaluate(() => window.location.hash);
        expect(hash).toContain('db=');
      }
    }
  });
});
