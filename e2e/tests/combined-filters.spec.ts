import { test, expect } from '../fixtures';

test.describe('Combined Filters and Views', () => {
  test.use({ readOnly: true });

  const modes = [
    { name: 'Search', hash: '', selector: '#all-media-btn' },
    { name: 'Trash', hash: 'mode=trash', selector: '#trash-btn' },
    { name: 'History', hash: 'mode=history', selector: '#history-completed-btn' },
    { name: 'Captions', hash: 'mode=captions', selector: '#captions-btn' }
  ];

  for (const mode of modes) {
    test(`mode: ${mode.name} - switching views and filtering`, async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + (mode.hash ? `#${mode.hash}` : ''));
      
      // Wait for results container
      await page.waitForSelector('#results-container', { timeout: 10000 });

      // Expand history details if we are in history mode to make button visible
      if (mode.name === 'History') {
        await page.evaluate(() => {
          const det = document.getElementById('details-history');
          if (det) (det as HTMLDetailsElement).open = true;
        });
      }

      // 1. Switch to Details view
      await page.click('#view-details');
      
      // Verify we are still in the same mode via URL
      if (mode.hash) {
        expect(page.url()).toContain(mode.hash);
      }
      
      // Verify we are still in the same mode via state evaluation
      const currentPage = await page.evaluate(() => (window as any).disco.state.page);
      const expectedPage = mode.name.toLowerCase();
      if (expectedPage !== 'search') {
        expect(currentPage).toBe(expectedPage);
      }

      // Verify active button in sidebar if it's supposed to be active
      if (mode.selector && mode.name !== 'History') {
        const activeBtn = page.locator(mode.selector);
        await expect(activeBtn).toHaveClass(/active/);
      }

      // 2. Switch to Group view
      await page.click('#view-group');
      
      // Special check for captions + group
      if (mode.name === 'Captions') {
        await page.waitForTimeout(1000);
        const resultsContainer = page.locator('#results-container');
        // Captions should still render as caption-media-cards even in "group" view
        const isCaptionCard = await page.locator('.caption-media-card').first().isVisible();
        expect(isCaptionCard).toBe(true);
      }

      // Verify we are still in the same mode after view change
      const pageAfterViewChange = await page.evaluate(() => (window as any).disco.state.page);
      if (expectedPage !== 'search') {
        expect(pageAfterViewChange).toBe(expectedPage);
      }

      // 3. Apply a search filter
      await page.fill('#search-input', 'test');
      await page.press('#search-input', 'Enter');
      await page.waitForTimeout(500);

      // Verify we are STILL in the same mode
      const pageAfterFilter = await page.evaluate(() => (window as any).disco.state.page);
      if (expectedPage !== 'search') {
        expect(pageAfterFilter).toBe(expectedPage);
      }

      // 4. Switch back to Grid view
      await page.click('#view-grid');
      
      // 5. Test pagination (if visible)
      const pagination = page.locator('#pagination-container');
      if (await pagination.isVisible()) {
        const nextPage = page.locator('#next-page');
        if (await nextPage.isEnabled()) {
          await nextPage.click();
          await page.waitForTimeout(500);
          
          // Verify we are STILL in the same mode
          const pageAfterPagination = await page.evaluate(() => (window as any).disco.state.page);
          if (expectedPage !== 'search') {
            expect(pageAfterPagination).toBe(expectedPage);
          }
        }
      }

      // Verify final state
      const finalPage = await page.evaluate(() => (window as any).disco.state.page);
      if (expectedPage !== 'search') {
        expect(finalPage).toBe(expectedPage);
      }
    });
  }

  test('Captions mode + Group view specific failure detection', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');
    await page.waitForSelector('.caption-media-card', { timeout: 10000 });

    // Switch to Group view
    await page.click('#view-group');
    
    // In current implementation, Group view triggers fetchEpisodes() which doesn't know about captions
    // It will likely show regular episodic groups instead of captions, or just be empty
    await page.waitForTimeout(2000);
    
    const resultsContainer = page.locator('#results-container');
    const hasSimilarityView = await resultsContainer.evaluate(el => el.classList.contains('similarity-view'));
    
    // If it has similarity-view, it means it's showing episode groups, NOT captions.
    // Captions mode should probably not even allow Group view if it's not implemented,
    // or Group view should group captions by something.
    
    const mediaCards = page.locator('.media-card');
    const firstCard = mediaCards.first();
    const isCaptionCard = await firstCard.evaluate(el => el.classList.contains('caption-media-card'));
    
    console.log(`In Captions+Group: hasSimilarityView=${hasSimilarityView}, isCaptionCard=${isCaptionCard}`);
    
    // If we are in Captions mode, every card should be a caption card
    // If it's broken, they might be regular media cards
    expect(isCaptionCard).toBe(true);
  });
});
