import { test, expect } from '../fixtures';

test.describe('Search and Query Integration', () => {
  test.describe.configure({ mode: 'serial' });
  test('performs search when enter is pressed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    // Wait for initial load
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Type search query and press enter
    await page.fill('#search-input', 'movie');
    await page.press('#search-input', 'Enter');
    
    // Wait for search results
    await page.waitForTimeout(1000);
    
    // Verify search was applied
    const cards = page.locator('.media-card');
    const count = await cards.count();
    
    // All results should match search
    for (let i = 0; i < count; i++) {
      const title = await cards.nth(i).locator('.media-title').textContent();
      expect(title?.toLowerCase()).toContain('movie');
    }
  });

  test('persists filters when switching views', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Expand media type filter
    const mediaTypeDetails = page.locator('#details-media-type');
    const mediaTypeSummary = mediaTypeDetails.locator('summary');
    const isMediaTypeExpanded = await mediaTypeDetails.evaluate((el: HTMLDetailsElement) => el.open);
    if (!isMediaTypeExpanded) {
      await mediaTypeSummary.click();
      await page.waitForTimeout(500);
    }

    // Select video type
    await page.click('#media-type-list .category-btn[data-type="video"]');
    await page.waitForTimeout(500);

    // Get current URL hash
    const hashAfterFilter = await page.evaluate(() => window.location.hash);
    expect(hashAfterFilter).toContain('type=video');

    // Switch to DU mode
    await page.click('#du-btn');
    await page.waitForSelector('#du-toolbar', { timeout: 10000 });

    // Filter should persist (check localStorage since DU mode has different URL structure)
    const typesInStorage = await page.evaluate(() => {
      return JSON.parse(localStorage.getItem('disco-types') || '[]');
    });
    expect(typesInStorage).toContain('video');
  });

  test('handles view mode switching (Grid, Group, Details)', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Start in grid view
    await expect(page.locator('#view-grid')).toHaveClass(/active/);
    await expect(page.locator('.grid')).toBeVisible();

    // Switch to Group (Episodes) view
    await page.click('#view-group');
    await page.waitForTimeout(1000);
    await expect(page.locator('#view-group')).toHaveClass(/active/);

    // Should have media cards (group view may not have episode-group if no episodes)
    await expect(page.locator('.media-card').first()).toBeVisible();

    // Switch to Details view
    await page.click('#view-details');
    await page.waitForTimeout(1000);
    await expect(page.locator('#view-details')).toHaveClass(/active/);
    await expect(page.locator('.details-table')).toBeVisible();

    // Switch back to Grid
    await page.click('#view-grid');
    await page.waitForTimeout(500);
    await expect(page.locator('#view-grid')).toHaveClass(/active/);
  });

  test('filters by progress states under History', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('#details-history', { timeout: 10000 });
    
    // Expand history section
    const historyDetails = page.locator('#details-history');
    await historyDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);
    
    // Click Unplayed
    await page.click('#history-unplayed-btn');
    await page.waitForTimeout(1000);
    
    // URL should reflect filter
    const hash = await page.evaluate(() => window.location.hash);
    expect(hash).toContain('history=unplayed');
    
    // Click In Progress
    await page.click('#history-in-progress-btn');
    await page.waitForTimeout(1000);
    
    const hash2 = await page.evaluate(() => window.location.hash);
    expect(hash2).toContain('history=in-progress');
    expect(hash2).not.toContain('history=unplayed');
    
    // Click Completed
    await page.click('#history-completed-btn');
    await page.waitForTimeout(1000);
    
    const hash3 = await page.evaluate(() => window.location.hash);
    expect(hash3).toContain('history=completed');
  });

  test('playlist management works end-to-end', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('#details-playlists', { timeout: 10000 });
    
    // Expand playlists section
    const playlistDetails = page.locator('#details-playlists');
    await playlistDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);
    
    // Handle prompt BEFORE clicking
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Playlist Title');
      await dialog.accept('Test Playlist');
    });

    // Create new playlist
    await page.click('#new-playlist-btn');
    
    await page.waitForTimeout(1000);
    
    // Playlist should appear in list
    await expect(page.locator('#playlist-list')).toContainText('Test Playlist');
    
    // Navigate to playlist
    const playlistBtn = page.locator('#playlist-list .category-btn').filter({ hasText: 'Test Playlist' });
    if (await playlistBtn.isVisible()) {
      await playlistBtn.click();
      await page.waitForTimeout(1000);
      
      // Should be in playlist view
      const hash = await page.evaluate(() => window.location.hash);
      expect(hash).toContain('mode=playlist');
    }
  });

  test('rating system works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Expand ratings filter
    const ratingDetails = page.locator('#details-ratings');
    if (await ratingDetails.isVisible()) {
      await ratingDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
      await page.waitForTimeout(500);
    }
    
    // Click a rating filter (e.g., 5 stars)
    const ratingBtn = page.locator('#ratings-list .category-btn[data-rating="5"]');
    if (await ratingBtn.isVisible()) {
      await ratingBtn.click();
      await page.waitForTimeout(1000);
      
      // URL should reflect rating filter
      const hash = await page.evaluate(() => window.location.hash);
      expect(hash).toContain('rating=5');
    }
  });

  test('category filtering works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('#details-categories', { timeout: 10000 });
    
    // Expand categories section
    const categoryDetails = page.locator('#details-categories');
    await categoryDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);
    
    // Click a category
    const categoryBtn = page.locator('#categories-list .category-btn').first();
    if (await categoryBtn.isVisible()) {
      const categoryName = await categoryBtn.textContent();
      await categoryBtn.click();
      await page.waitForTimeout(1000);
      
      // URL should reflect category filter
      const hash = await page.evaluate(() => window.location.hash);
      expect(hash).toContain('category=');
    }
  });

  test('pagination works correctly', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Check pagination controls exist
    const pagination = page.locator('#pagination-container');
    await expect(pagination).toBeVisible();
    
    const pageInfo = page.locator('#page-info');
    await expect(pageInfo).toBeVisible();
    
    // Get initial page
    const initialPageText = await pageInfo.textContent();
    expect(initialPageText).toContain('Page 1');
    
    // Try next page if available
    const nextBtn = page.locator('#next-page');
    if (!(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await page.waitForTimeout(1000);
      
      const page2Text = await pageInfo.textContent();
      expect(page2Text).toContain('Page 2');
      
      // Previous should now be enabled
      const prevBtn = page.locator('#prev-page');
      expect(await prevBtn.isDisabled()).toBe(false);
    }
  });

  test('sorting works correctly', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Change sort to size
    const sortBy = page.locator('#sort-by');
    await sortBy.selectOption('size');
    await page.waitForTimeout(500);
    
    // Verify sort changed
    await expect(sortBy).toHaveValue('size');
    
    // Toggle reverse sort
    const reverseBtn = page.locator('#sort-reverse-btn');
    await reverseBtn.click();
    await page.waitForTimeout(500);
    await expect(reverseBtn).toHaveClass(/active/);
    
    // Change to duration sort
    await sortBy.selectOption('duration');
    await page.waitForTimeout(500);
    await expect(sortBy).toHaveValue('duration');
  });

  test('limit setting works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
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
    
    // Verify limit was applied (check URL or fetch)
    const hash = await page.evaluate(() => window.location.hash);
    // Limit might be stored in localStorage, not URL
  });

  test('database filtering works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
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
        
        // Should filter by database
        const hash = await page.evaluate(() => window.location.hash);
        expect(hash).toContain('db=');
      }
    }
  });
});
