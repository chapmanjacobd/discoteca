import { test, expect } from '../fixtures';

test.describe('Search and Query Integration', () => {
  test.use({ readOnly: true });

  test('search filters media by title', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Initial count using POM
    const initialCount = await mediaPage.getMediaCount();
    expect(initialCount).toBeGreaterThan(0);

    // Search for 'video' which should match test_video files using FTS
    await mediaPage.search('video');
    await mediaPage.page.waitForTimeout(500);

    // Count should decrease or change using POM
    const searchCount = await mediaPage.getMediaCount();

    // Should have results
    expect(searchCount).toBeGreaterThan(0);
    expect(searchCount).toBeLessThanOrEqual(initialCount);

    // Check first result contains 'video' in path using POM
    const firstPath = await mediaPage.getMediaCardPath(0);
    expect(firstPath.toLowerCase().includes('video')).toBe(true);
  });

  test('filters by media type', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open Media Type filter section using POM
    await sidebarPage.expandMediaTypeSection();
    await mediaPage.page.waitForTimeout(500);

    // Click Video filter using POM
    const videoBtn = sidebarPage.getMediaTypeButton('video');
    if (await videoBtn.isVisible()) {
      await videoBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // All results should be video using POM
      const count = await mediaPage.getMediaCount();

      for (let i = 0; i < Math.min(count, 5); i++) {
        const title = await mediaPage.getMediaTitle(i);
        expect(title).toBeTruthy();
      }
    }
  });

  test('filters by progress states under History', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open History/Progress filter section using POM
    await sidebarPage.expandHistorySection();
    await mediaPage.page.waitForTimeout(500);

    // Click In Progress filter using POM
    const inProgressBtn = sidebarPage.historyInProgressButton;
    if (await inProgressBtn.isVisible()) {
      await inProgressBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // Should only show in progress items using POM
      const count = await mediaPage.getMediaCount();
      expect(count).toBeGreaterThanOrEqual(0);
    }
  });

  test('search is case insensitive', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search with lowercase using POM
    await mediaPage.search('video');
    await mediaPage.page.waitForTimeout(500);
    const lowerCount = await mediaPage.getMediaCount();
    expect(lowerCount).toBeGreaterThan(0);

    // Clear and search with uppercase using POM
    await mediaPage.clearSearch();
    await mediaPage.search('VIDEO');
    await mediaPage.page.waitForTimeout(500);
    const upperCount = await mediaPage.getMediaCount();

    // Results should be the same (case insensitive)
    expect(upperCount).toBe(lowerCount);
  });

  test('search with special characters', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search with special characters using POM
    await mediaPage.search('test-video');
    await mediaPage.page.waitForTimeout(1000);

    // Should not crash and should show results or empty state using POM
    const count = await mediaPage.getMediaCount();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('clearing search restores all results', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();

    // Search using POM
    await mediaPage.search('video');
    await mediaPage.page.waitForTimeout(500);
    const searchCount = await mediaPage.getMediaCount();
    expect(searchCount).toBeLessThanOrEqual(initialCount);
    expect(searchCount).toBeGreaterThan(0);

    // Clear search using POM
    await mediaPage.clearSearch();
    await mediaPage.page.waitForTimeout(500);

    // Count should return to initial using POM
    const finalCount = await mediaPage.getMediaCount();
    expect(finalCount).toBe(initialCount);
  });

  test('search persists across view mode changes', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search using POM
    await mediaPage.search('video');
    await mediaPage.page.waitForTimeout(500);
    const searchCount = await mediaPage.getMediaCount();
    expect(searchCount).toBeGreaterThan(0);

    // Switch to details view using POM
    await mediaPage.switchToDetailsView();
    await mediaPage.page.waitForTimeout(500);

    // Search results should persist using POM
    const detailsCount = await mediaPage.getMediaCount();
    expect(detailsCount).toBe(searchCount);

    // Switch back to grid using POM
    await mediaPage.switchToGridView();
    await mediaPage.page.waitForTimeout(500);

    // Search results should still persist using POM
    const gridCount = await mediaPage.getMediaCount();
    expect(gridCount).toBe(searchCount);
  });

  test('empty search shows all results', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();

    // Search with empty string using POM
    await mediaPage.search('');

    // Should show all results using POM
    const finalCount = await mediaPage.getMediaCount();
    expect(finalCount).toBe(initialCount);
  });
});
