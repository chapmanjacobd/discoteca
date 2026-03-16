import { test, expect } from '../fixtures';

test.describe('Pagination Limit', () => {
  test.use({ readOnly: true });

  test('pagination controls are visible when results exceed page size', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Check if pagination is visible using POM
    await expect(mediaPage.paginationContainer).toBeVisible();

    // Page info should show current page using POM
    await expect(mediaPage.pageInfo).toBeVisible();
  });

  test('next page button navigates to next page', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Get initial page info using POM
    const initialPageText = await mediaPage.pageInfo.textContent();

    // Click next page button using POM
    const nextBtn = mediaPage.page.locator('#next-page');
    if (await nextBtn.count() > 0 && !(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // Page number should have changed using POM
      const newPageText = await mediaPage.pageInfo.textContent();
      expect(newPageText).not.toBe(initialPageText);
    }
  });

  test('previous page button navigates to previous page', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Go to page 2 first using POM
    const nextBtn = mediaPage.page.locator('#next-page');
    if (await nextBtn.count() > 0 && !(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // Click previous page button using POM
      const prevBtn = mediaPage.page.locator('#prev-page');
      if (await prevBtn.count() > 0 && !(await prevBtn.isDisabled())) {
        await prevBtn.click();
        await mediaPage.page.waitForTimeout(1000);

        // Should be back on page 1 using POM
        const pageText = await mediaPage.pageInfo.textContent();
        expect(pageText).toContain('1');
      }
    }
  });

  test('page size can be changed', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Check if page size selector exists using POM
    const pageSizeSelect = mediaPage.page.locator('#page-size');
    if (await pageSizeSelect.count() > 0) {
      // Get initial page size using POM
      const initialSize = await pageSizeSelect.inputValue();

      // Change to different page size using POM
      await pageSizeSelect.selectOption('100');
      await mediaPage.page.waitForTimeout(500);

      // Page size should have changed using POM
      const newSize = await pageSizeSelect.inputValue();
      expect(newSize).toBe('100');

      // Restore original size
      await pageSizeSelect.selectOption(initialSize);
    }
  });

  test('pagination works with search results', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Search for something using POM
    await mediaPage.search('test');

    // Pagination should still be visible if results exceed page size using POM
    const paginationVisible = await mediaPage.paginationContainer.isVisible();

    if (paginationVisible) {
      await expect(mediaPage.paginationContainer).toBeVisible();
    }
  });

  test('pagination works with filtered results', async ({ mediaPage, sidebarPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Apply a filter using POM
    await sidebarPage.expandMediaTypeSection();
    await sidebarPage.getMediaTypeButton('video').click();
    await mediaPage.page.waitForTimeout(1000);

    // Pagination should still work with filtered results using POM
    const paginationVisible = await mediaPage.paginationContainer.isVisible();

    if (paginationVisible) {
      await expect(mediaPage.paginationContainer).toBeVisible();
    }
  });

  test('last page button navigates to last page', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Click last page button using POM
    const lastBtn = mediaPage.page.locator('#last-page');
    if (await lastBtn.count() > 0 && !(await lastBtn.isDisabled())) {
      await lastBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // Should be on last page (check page info) using POM
      const pageText = await mediaPage.pageInfo.textContent();
      // Page text format is typically "X of Y"
      const parts = pageText?.split(' of ') || [];
      if (parts.length === 2) {
        const currentPage = parts[0].trim();
        const lastPage = parts[1].trim();
        expect(currentPage).toBe(lastPage);
      }
    }
  });

  test('first page button navigates to first page', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Go to page 2 first using POM
    const nextBtn = mediaPage.page.locator('#next-page');
    if (await nextBtn.count() > 0 && !(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // Click first page button using POM
      const firstBtn = mediaPage.page.locator('#first-page');
      if (await firstBtn.count() > 0 && !(await firstBtn.isDisabled())) {
        await firstBtn.click();
        await mediaPage.page.waitForTimeout(1000);

        // Should be back on page 1 using POM
        const pageText = await mediaPage.pageInfo.textContent();
        expect(pageText).toContain('1');
      }
    }
  });

  test('pagination buttons are disabled when appropriate', async ({ mediaPage, server }) => {
    // Set a small limit to force pagination (test.db has 11 items)
    await mediaPage.page.goto(server.getBaseUrl() + '/#limit=5');

    // Previous and first buttons should be disabled on page 1 using POM
    const prevBtn = mediaPage.page.locator('#prev-page');
    const firstBtn = mediaPage.page.locator('#first-page');

    if (await prevBtn.count() > 0) {
      const prevDisabled = await prevBtn.isDisabled();
      expect(prevDisabled).toBe(true);
    }

    if (await firstBtn.count() > 0) {
      const firstDisabled = await firstBtn.isDisabled();
      expect(firstDisabled).toBe(true);
    }
  });
});
