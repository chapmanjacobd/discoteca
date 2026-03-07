import { test, expect } from '../fixtures';

test.describe('Search and Filtering', () => {
  test('search filters media by title', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Get initial count
    const initialCards = page.locator('.media-card');
    const initialCount = await initialCards.count();
    
    // Search for "movie"
    await page.fill('#search-input', 'movie');
    await page.press('#search-input', 'Enter');
    
    // Wait for search results
    await page.waitForTimeout(1000);
    
    // Should have filtered results
    const filteredCards = page.locator('.media-card');
    const filteredCount = await filteredCards.count();
    
    // Count should be less than or equal to initial
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
    
    // All results should contain "movie" in title or path
    for (let i = 0; i < filteredCount; i++) {
      const title = await filteredCards.nth(i).locator('.media-title').textContent();
      expect(title?.toLowerCase()).toContain('movie');
    }
  });

  test('clears search when X button clicked', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Search for something
    await page.fill('#search-input', 'movie');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);
    
    // Clear search
    await page.fill('#search-input', '');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Results should be back to normal
    const cards = page.locator('.media-card');
    const count = await cards.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('filters by media type', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('#details-media-type', { timeout: 10000 });
    
    // Expand media type filter
    await page.locator('#details-media-type').evaluate((el: HTMLDetailsElement) => el.open = true);
    
    // Get initial count
    const initialCards = page.locator('.media-card');
    const initialCount = await initialCards.count();
    
    // Click video filter
    await page.locator('#media-type-list .category-btn[data-type="video"]').click();
    await page.waitForTimeout(1000);
    
    // Should have video results
    const videoCards = page.locator('.media-card');
    const videoCount = await videoCards.count();
    
    // Count should be less than or equal to initial
    expect(videoCount).toBeLessThanOrEqual(initialCount);
  });

  test('pagination works for large result sets', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Check if pagination is visible
    const pagination = page.locator('#pagination-container');
    
    // If there are results, pagination controls should exist
    const cards = page.locator('.media-card');
    const count = await cards.count();
    
    if (count > 0) {
      await expect(pagination).toBeVisible();
      
      // Page info should show current page
      const pageInfo = page.locator('#page-info');
      await expect(pageInfo).toBeVisible();
    }
  });

  test('sort options work', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Change sort option
    const sortBy = page.locator('#sort-by');
    await sortBy.selectOption('size');
    await page.waitForTimeout(500);
    
    // Verify sort changed
    await expect(sortBy).toHaveValue('size');
    
    // Change to another sort
    await sortBy.selectOption('duration');
    await page.waitForTimeout(500);
    await expect(sortBy).toHaveValue('duration');
  });

  test('reverse sort toggles correctly', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    const reverseBtn = page.locator('#sort-reverse-btn');

    // Click to toggle
    await reverseBtn.click();
    await page.waitForTimeout(300);

    // Should have active class
    await expect(reverseBtn).toHaveClass(/active/);

    // Click again to toggle off
    await reverseBtn.click();
    await page.waitForTimeout(300);

    // Should not have active class
    await expect(reverseBtn).not.toHaveClass(/active/);
  });

  test('filter bins (sliders) are visible in DU mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=du');
    
    await page.waitForSelector('#du-toolbar', { timeout: 10000 });
    
    // Open details elements
    await page.locator('#details-episodes').evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.locator('#details-size').evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.locator('#details-duration').evaluate((el: HTMLDetailsElement) => el.open = true);

    // Filter sliders should be visible
    await expect(page.locator('#episodes-slider-container')).toBeVisible();
    await expect(page.locator('#size-slider-container')).toBeVisible();
    await expect(page.locator('#duration-slider-container')).toBeVisible();
  });
});
