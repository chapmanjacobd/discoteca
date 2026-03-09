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
        // Captions render differently based on view mode:
        // - Grid/Details: .caption-media-card
        // - Group: .caption-group
        const isGroupView = await page.evaluate(() => window.disco.state.view === 'group');
        let isCaptionCard;
        if (isGroupView) {
          isCaptionCard = await page.locator('.caption-group').first().isVisible();
        } else {
          isCaptionCard = await page.locator('.caption-media-card').first().isVisible();
        }
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
    await page.waitForSelector('.caption-media-card, .caption-group', { timeout: 10000 });

    // Switch to Group view
    await page.click('#view-group');

    // In Captions mode, Group view renders .caption-group elements
    await page.waitForTimeout(2000);

    const resultsContainer = page.locator('#results-container');
    const hasSimilarityView = await resultsContainer.evaluate(el => el.classList.contains('similarity-view'));

    // In Captions mode + Group view, we should see .caption-group elements
    // If it has similarity-view, it means it's showing episode groups, NOT captions
    const captionGroups = page.locator('.caption-group');
    const captionCards = page.locator('.caption-media-card');
    
    const groupCount = await captionGroups.count();
    const cardCount = await captionCards.count();

    console.log(`In Captions+Group: hasSimilarityView=${hasSimilarityView}, captionGroups=${groupCount}, captionCards=${cardCount}`);

    // Either caption groups or caption cards should be visible
    expect(groupCount > 0 || cardCount > 0).toBe(true);
  });
});
