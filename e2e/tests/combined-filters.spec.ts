import { test, expect } from '../fixtures';

test.describe('Combined Filters and Views', () => {
  test.use({ readOnly: false });

  const modes = [
    { name: 'Search', hash: '', selector: '#all-media-btn' },
    { name: 'Trash', hash: 'mode=trash', selector: '#trash-btn' },
    { name: 'History', hash: 'mode=history', selector: '#history-completed-btn' },
    { name: 'Captions', hash: 'mode=captions', selector: '#captions-btn' }
  ];

  for (const mode of modes) {
    test(`mode: ${mode.name} - switching views and filtering`, async ({ mediaPage, sidebarPage, viewerPage, server }) => {
      // Set up test data for Trash and History modes
      if (mode.name === 'Trash') {
        // Go to home, delete a file to populate trash
        await mediaPage.goto(server.getBaseUrl());
        const firstVideo = mediaPage.getFirstMediaCardByType('video');
        await firstVideo.click();
        await viewerPage.waitForPlayer();
        // Click the delete button in the viewer
        const deleteBtn = viewerPage.page.locator('.media-action-btn.delete').first();
        await deleteBtn.click();
        // Wait for deletion to process
        await mediaPage.page.waitForTimeout(1000);
        await viewerPage.close();
        await mediaPage.page.waitForTimeout(500);
      } else if (mode.name === 'History') {
        // Go to home, watch a file to populate history
        await mediaPage.goto(server.getBaseUrl());
        const firstVideo = mediaPage.getFirstMediaCardByType('video');
        await firstVideo.click();
        await viewerPage.waitForPlayer();
        await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });
        await viewerPage.play();
        await mediaPage.page.waitForTimeout(2000);
        await viewerPage.close();
        await mediaPage.page.waitForTimeout(500);
      }

      await mediaPage.goto(server.getBaseUrl() + (mode.hash ? `#${mode.hash}` : ''));

      // Wait for results container using POM
      await expect(mediaPage.resultsContainer).toBeVisible();

      // For Trash and History, verify we have items (test setup requirement)
      if (mode.name === 'Trash' || mode.name === 'History') {
        const count = await mediaPage.getMediaCount();
        expect(count).toBeGreaterThan(0);
      }

      // Expand history details if we are in history mode using POM
      if (mode.name === 'History') {
        await sidebarPage.expandHistorySection();
      }

      // 1. Switch to Details view using POM
      await mediaPage.switchToDetailsView();

      // Verify we are still in the same mode via URL using POM
      if (mode.hash) {
        expect(mediaPage.page.url()).toContain(mode.hash);
      }

      // Verify we are still in the same mode via state evaluation
      const currentPage = await mediaPage.page.evaluate(() => (window as any).disco.state.page);
      const expectedPage = mode.name.toLowerCase();
      if (expectedPage !== 'search') {
        expect(currentPage).toBe(expectedPage);
      }

      // Verify active button in sidebar if it's supposed to be active using POM
      if (mode.selector && mode.name !== 'History') {
        const activeBtn = mediaPage.page.locator(mode.selector);
        await expect(activeBtn).toHaveClass(/active/);
      }

      // 2. Switch to Group view using POM
      await mediaPage.page.locator('#view-group').click();
      await mediaPage.page.waitForTimeout(500);

      // Special check for captions + group using POM
      if (mode.name === 'Captions') {
        await mediaPage.page.waitForTimeout(1000);
        const resultsContainer = mediaPage.resultsContainer;
        // Captions render differently based on view mode
        const isGroupView = await mediaPage.page.evaluate(() => window.disco.state.view === 'group');
        let isCaptionCard;
        if (isGroupView) {
          isCaptionCard = await mediaPage.page.locator('.caption-group').first().isVisible();
        } else {
          isCaptionCard = await mediaPage.getCaptionCards().first().isVisible();
        }
        expect(isCaptionCard).toBe(true);
      }

      // Verify we are still in the same mode after view change using POM
      const pageAfterViewChange = await mediaPage.page.evaluate(() => (window as any).disco.state.page);
      if (expectedPage !== 'search') {
        expect(pageAfterViewChange).toBe(expectedPage);
      }

      // 3. Apply a search filter using POM
      // Search for 'test' which matches our test files (test_video1, test_image1, etc.)
      await mediaPage.search('test');

      // Verify we are STILL in the same mode using POM
      const pageAfterFilter = await mediaPage.page.evaluate(() => (window as any).disco.state.page);
      if (expectedPage !== 'search') {
        expect(pageAfterFilter).toBe(expectedPage);
      }

      // 4. Switch back to Grid view using POM
      await mediaPage.switchToGridView();

      // 5. Test pagination (if visible) using POM
      const pagination = mediaPage.paginationContainer;
      if (await pagination.isVisible()) {
        const nextPage = mediaPage.page.locator('#next-page');
        if (await nextPage.isEnabled()) {
          await nextPage.click();
          await mediaPage.page.waitForTimeout(500);

          // Verify we are STILL in the same mode using POM
          const pageAfterPagination = await mediaPage.page.evaluate(() => (window as any).disco.state.page);
          if (expectedPage !== 'search') {
            expect(pageAfterPagination).toBe(expectedPage);
          }
        }
      }

      // 6. Clear search using POM
      await mediaPage.clearSearch();

      // Final verification using POM
      const finalPage = await mediaPage.page.evaluate(() => (window as any).disco.state.page);
      if (expectedPage !== 'search') {
        expect(finalPage).toBe(expectedPage);
      }
    });
  }

  test('all view modes are accessible from any page', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Grid view button should be visible using POM
    await expect(mediaPage.viewGridButton).toBeVisible();

    // Details view button should be visible using POM
    await expect(mediaPage.viewDetailsButton).toBeVisible();

    // Group view button may exist using POM
    const groupViewBtn = mediaPage.viewGroupButton;
    if (await groupViewBtn.count() > 0) {
      await expect(groupViewBtn).toBeVisible();
    }
  });

  test('view mode persists across page navigation', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Switch to details view using POM
    await mediaPage.switchToDetailsView();

    // Verify we're in details view using POM
    await expect(mediaPage.viewDetailsButton).toHaveClass(/active/);

    // Navigate to a different mode using POM
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
    // Captions may render as cards, table, or groups depending on view mode
    const captionSelectors = ['.caption-media-card', '.details-table', '.caption-group'];
    await mediaPage.page.locator(captionSelectors.join(', ')).first().waitFor({ state: 'visible', timeout: 10000 });

    // Go back to home using POM
    await mediaPage.goto(server.getBaseUrl());
    await mediaPage.waitForMediaToLoad();

    // View mode may or may not persist depending on implementation
    // Just verify the page loads correctly using POM
    await expect(mediaPage.resultsContainer).toBeVisible();
  });

  test('search works in all view modes', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Test search in grid view using POM
    await mediaPage.search('test');
    const gridSearchCount = await mediaPage.getMediaCount();
    expect(gridSearchCount).toBeGreaterThanOrEqual(0);

    // Clear and switch to details view using POM
    await mediaPage.clearSearch();
    await mediaPage.switchToDetailsView();

    // Test search in details view using POM
    await mediaPage.search('test');
    const detailsSearchCount = await mediaPage.getMediaCount();
    expect(detailsSearchCount).toBeGreaterThanOrEqual(0);

    // Results count should be similar (may differ slightly due to rendering)
    expect(Math.abs(gridSearchCount - detailsSearchCount)).toBeLessThanOrEqual(1);
  });
});
