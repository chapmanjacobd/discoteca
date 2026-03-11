import { test, expect } from '../fixtures';

test.describe('Search and Filtering', () => {
  test.use({ readOnly: true });

  test('search filters media by title', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();

    // Search for "video" which should match test_video files using POM
    await mediaPage.search('video');

    // Should have filtered results using POM
    const filteredCount = await mediaPage.getMediaCount();

    // Count should be less than or equal to initial
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
    expect(filteredCount).toBeGreaterThan(0);

    // All results should contain "video" in title or path using POM
    for (let i = 0; i < filteredCount; i++) {
      const card = mediaPage.getMediaCard(i);
      const path = await card.getAttribute('data-path');
      expect(path?.toLowerCase().includes('video')).toBe(true);
    }
  });

  test('clears search when X button clicked', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for something using POM
    await mediaPage.search('video');

    // Clear search using POM
    await mediaPage.clearSearch();

    // Results should be back to normal using POM
    const count = await mediaPage.getMediaCount();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('filters by media type', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Expand media type filter using POM
    await sidebarPage.expandMediaTypeSection();

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();

    // Click video filter using POM
    await sidebarPage.getMediaTypeButton('video').click();
    await mediaPage.page.waitForTimeout(1000);

    // Should have video results using POM
    const videoCount = await mediaPage.getMediaCount();

    // Count should be less than or equal to initial
    expect(videoCount).toBeLessThanOrEqual(initialCount);
  });

  test('pagination works for large result sets', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Check if pagination is visible using POM
    const pagination = mediaPage.paginationContainer;

    // If there are results, pagination controls should exist using POM
    const count = await mediaPage.getMediaCount();

    if (count > 0) {
      await expect(pagination).toBeVisible();

      // Page info should show current page using POM
      await expect(mediaPage.pageInfo).toBeVisible();
    }
  });

  test('sort options work', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Change sort option using POM
    await mediaPage.setSortBy('size');
    await mediaPage.page.waitForTimeout(500);

    // Verify sort changed using POM
    await expect(mediaPage.sortBySelect).toHaveValue('size');

    // Change to another sort using POM
    await mediaPage.setSortBy('duration');
    await mediaPage.page.waitForTimeout(500);
    await expect(mediaPage.sortBySelect).toHaveValue('duration');
  });

  test('reverse sort toggles correctly', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click to toggle using POM
    await mediaPage.sortReverseBtn.click();
    await mediaPage.page.waitForTimeout(300);

    // Should have active class using POM
    await expect(mediaPage.sortReverseBtn).toHaveClass(/active/);

    // Click again to toggle off using POM
    await mediaPage.sortReverseBtn.click();
    await mediaPage.page.waitForTimeout(300);

    // Should not have active class using POM
    await expect(mediaPage.sortReverseBtn).not.toHaveClass(/active/);
  });

  test('filter bins (sliders) are visible in DU mode', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Open details elements using POM
    await sidebarPage.expandEpisodesSection();
    await sidebarPage.expandSizeSection();
    await sidebarPage.expandDurationSection();

    // Filter sliders should be visible using POM
    await expect(mediaPage.episodesSliderContainer).toBeVisible();
    await expect(mediaPage.sizeSliderContainer).toBeVisible();
    await expect(mediaPage.durationSliderContainer).toBeVisible();
  });
});
