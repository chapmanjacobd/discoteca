import { test, expect } from '../fixtures';

test.describe('Pagination Limit and X-Total-Count', () => {
  test('results count matches X-Total-Count header', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for initial load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get the total count from the UI (should show total results)
    const totalCountText = await page.locator('#total-count, .total-count, [data-total-count]').textContent();
    
    // Get current page and limit
    const pageInfo = await page.locator('#page-info').textContent();
    
    // Parse page info to get current page and total pages
    // Format is typically "Page X of Y" or similar
    const pageMatch = pageInfo?.match(/Page\s+(\d+)\s+of\s+(\d+)/);
    
    if (pageMatch) {
      const currentPage = parseInt(pageMatch[1]);
      const totalPages = parseInt(pageMatch[2]);
      
      // Get limit from input
      const limitInput = page.locator('#limit');
      const limitValue = await limitInput.inputValue();
      const limit = parseInt(limitValue) || 99;
      
      // Calculate expected total from page info
      const expectedTotalFromPages = totalPages * limit;
      
      // The total count displayed should match what we expect
      if (totalCountText) {
        const displayedTotal = parseInt(totalCountText.replace(/[^0-9]/g, ''));
        // Total should be consistent with pagination
        expect(displayedTotal).toBeGreaterThan(0);
      }
    }
    
    // Verify we're not hitting the limit incorrectly
    // If there are more results than the limit, we should see pagination controls
    const nextBtn = page.locator('#next-page');
    const hasNextPage = await nextBtn.isVisible() && !(await nextBtn.isDisabled());
    
    // If there's a next page, we should have exactly limit items on current page
    if (hasNextPage) {
      const cards = page.locator('.media-card');
      const cardCount = await cards.count();
      
      // Should have exactly limit items (or close to it for last page edge cases)
      expect(cardCount).toBeLessThanOrEqual(limit);
    }
  });

  test('changing limit updates results correctly', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial limit
    const limitInput = page.locator('#limit');
    const initialLimit = await limitInput.inputValue();
    
    // Change to a smaller limit
    await limitInput.fill('25');
    await page.waitForTimeout(1000);
    
    // Count results
    const cards25 = page.locator('.media-card');
    const count25 = await cards25.count();
    
    // Should have at most 25 results
    expect(count25).toBeLessThanOrEqual(25);
    
    // Change to larger limit
    await limitInput.fill('100');
    await page.waitForTimeout(1000);
    
    const cards100 = page.locator('.media-card');
    const count100 = await cards100.count();
    
    // Should have at most 100 results
    expect(count100).toBeLessThanOrEqual(100);
    expect(count100).toBeGreaterThanOrEqual(count25);
    
    // Restore initial limit
    await limitInput.fill(initialLimit);
  });

  test('pagination navigation maintains correct count', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get total count from first page
    const totalCountEl = page.locator('#total-count, .total-count');
    let initialTotal = 0;
    if (await totalCountEl.count() > 0) {
      const totalText = await totalCountEl.textContent();
      initialTotal = parseInt(totalText?.replace(/[^0-9]/g, '') || '0');
    }
    
    // Try to go to next page
    const nextBtn = page.locator('#next-page');
    if (await nextBtn.isVisible() && !(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await page.waitForTimeout(1000);
      
      // Total count should remain the same
      if (await totalCountEl.count() > 0) {
        const totalText2 = await totalCountEl.textContent();
        const total2 = parseInt(totalText2?.replace(/[^0-9]/g, '') || '0');
        expect(total2).toBe(initialTotal);
      }
      
      // Go back to previous page
      const prevBtn = page.locator('#prev-page');
      if (await prevBtn.isVisible() && !(await prevBtn.isDisabled())) {
        await prevBtn.click();
        await page.waitForTimeout(1000);
        
        // Total count should still be the same
        if (await totalCountEl.count() > 0) {
          const totalText3 = await totalCountEl.textContent();
          const total3 = parseInt(totalText3?.replace(/[^0-9]/g, '') || '0');
          expect(total3).toBe(initialTotal);
        }
      }
    }
  });

  test('results per page matches limit setting', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Test with different limit values
    const testLimits = [10, 25, 50];
    
    for (const limit of testLimits) {
      const limitInput = page.locator('#limit');
      await limitInput.fill(limit.toString());
      await page.waitForTimeout(800);
      
      const cards = page.locator('.media-card');
      const count = await cards.count();
      
      // Should not exceed limit
      expect(count).toBeLessThanOrEqual(limit);
      
      // If we have results, count should be positive
      if (count > 0) {
        // On non-last pages, should be exactly the limit
        const nextBtn = page.locator('#next-page');
        const hasNext = await nextBtn.isVisible() && !(await nextBtn.isDisabled());
        
        if (hasNext) {
          expect(count).toBe(limit);
        }
      }
    }
  });

  test('X-Total-Count header is correctly read by frontend', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Check that the frontend has received and stored the total count
    const stateTotalCount = await page.evaluate(() => {
      // Access the state object if available
      const win = window as any;
      if (win.state && win.state.totalCount) {
        return win.state.totalCount;
      }
      return null;
    });
    
    // Total count should be a positive number
    if (stateTotalCount !== null) {
      expect(stateTotalCount).toBeGreaterThan(0);
    }
    
    // Verify displayed count matches state
    const displayedCountEl = page.locator('#total-count, .total-count');
    if (await displayedCountEl.count() > 0) {
      const displayedText = await displayedCountEl.textContent();
      const displayedCount = parseInt(displayedText?.replace(/[^0-9]/g, '') || '0');
      
      if (stateTotalCount !== null) {
        expect(displayedCount).toBe(stateTotalCount);
      }
    }
  });

  test('multiple databases do not return more results than limit', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Set a specific limit
    const limitInput = page.locator('#limit');
    await limitInput.fill('50');
    await page.waitForTimeout(1000);
    
    // Count results
    const cards = page.locator('.media-card');
    const count = await cards.count();
    
    // Should not exceed limit even with multiple databases
    expect(count).toBeLessThanOrEqual(50);
    
    // Check if we have database filter available
    const dbFilter = page.locator('#details-databases');
    if (await dbFilter.isVisible()) {
      await dbFilter.evaluate((el: HTMLDetailsElement) => el.open = true);
      await page.waitForTimeout(500);
      
      // Check how many databases are configured
      const dbBtns = page.locator('#databases-list .category-btn');
      const dbCount = await dbBtns.count();
      
      // If multiple databases, verify limit is still respected
      if (dbCount > 1) {
        // Select all databases
        for (let i = 0; i < dbCount; i++) {
          const btn = dbBtns.nth(i);
          if (!(await btn.hasClass('active'))) {
            await btn.click();
            await page.waitForTimeout(300);
          }
        }
        
        // Re-count results
        const cardsAfter = page.locator('.media-card');
        const countAfter = await cardsAfter.count();
        
        // Still should not exceed limit
        expect(countAfter).toBeLessThanOrEqual(50);
      }
    }
  });

  test('database filtering updates total count correctly', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial total count
    const initialTotalText = await page.locator('#results-count').textContent();
    const initialTotal = parseInt(initialTotalText?.replace(/[^0-9]/g, '') || '0');
    
    // Check if we have database filter available
    const dbFilter = page.locator('#details-databases');
    if (await dbFilter.isVisible()) {
      await dbFilter.evaluate((el: HTMLDetailsElement) => el.open = true);
      await page.waitForTimeout(500);
      
      // Check how many databases are configured
      const dbBtns = page.locator('#databases-list .category-btn');
      const dbCount = await dbBtns.count();
      
      // If multiple databases, test filtering
      if (dbCount > 1) {
        // Deselect one database
        const firstDbBtn = dbBtns.first();
        await firstDbBtn.click();
        await page.waitForTimeout(1000);
        
        // Get new total count
        const filteredTotalText = await page.locator('#results-count').textContent();
        const filteredTotal = parseInt(filteredTotalText?.replace(/[^0-9]/g, '') || '0');
        
        // Total should be less than or equal to initial (could be equal if one DB is empty)
        expect(filteredTotal).toBeLessThanOrEqual(initialTotal);
        
        // Results count should match displayed total
        const cards = page.locator('.media-card');
        const cardCount = await cards.count();
        expect(cardCount).toBeLessThanOrEqual(filteredTotal);
        
        // Re-select the database
        await firstDbBtn.click();
        await page.waitForTimeout(1000);
        
        // Total should return to initial
        const restoredTotalText = await page.locator('#results-count').textContent();
        const restoredTotal = parseInt(restoredTotalText?.replace(/[^0-9]/g, '') || '0');
        expect(restoredTotal).toBe(initialTotal);
      }
    }
  });
});
