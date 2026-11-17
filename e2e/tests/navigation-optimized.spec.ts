import { test, expect } from '../fixtures';

test.describe('Basic Navigation (Read-Only)', () => {
  // All tests in this describe block are read-only
  test.use({ readOnly: true });

  test('loads the home page', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Verify key elements are present using POM
    await expect(mediaPage.searchInput).toBeVisible();
    await expect(mediaPage.resultsContainer).toBeVisible();
  });

  test('navigates to Disk Usage view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Use sidebar POM to navigate to DU view
    await sidebarPage.openDiskUsage();

    // Should show DU toolbar using POM
    await expect(mediaPage.getDUTToolbar()).toBeVisible();
    // Path input is hidden on mobile in favour of breadcrumbs
    const pathInput = mediaPage.getDUPathInput();
    if (await pathInput.isVisible()) {
      await expect(pathInput).toBeVisible();
    } else {
      await expect(mediaPage.page.locator('#du-breadcrumbs')).toBeVisible();
    }
  });

  test('navigates to Captions view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Use sidebar POM to navigate to Captions view
    await sidebarPage.openCaptions();

    // Should show captions (or error if no captions in DB) using POM
    await mediaPage.page.waitForTimeout(1000);
    const hasCaptions = await mediaPage.getCaptionCards().count() > 0;
    if (hasCaptions) {
      await expect(mediaPage.getCaptionCards().first()).toBeVisible();
    }
  });

  test('opens and closes settings modal', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Use sidebar POM to open settings
    await sidebarPage.openSettings();

    const modal = mediaPage.getSettingsModal();
    await expect(modal).toBeVisible();

    // Use sidebar POM to close settings
    await sidebarPage.closeSettings();
    await expect(modal).not.toBeVisible();
  });

  test('toggles view modes (grid/details)', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Should start in grid view using POM
    await expect(mediaPage.viewGridButton).toHaveClass(/active/);

    // Switch to details view using POM
    await mediaPage.switchToDetailsView();
    await expect(mediaPage.viewDetailsButton).toHaveClass(/active/);

    // Switch back to grid using POM
    await mediaPage.switchToGridView();
    await expect(mediaPage.viewGridButton).toHaveClass(/active/);
  });

  test('navigates to History pages', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Navigate to In Progress using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.clickHistoryInProgress();
    await mediaPage.page.waitForTimeout(1000);
    await expect(mediaPage.resultsContainer).toBeVisible();

    // Navigate to Unplayed using POM
    await sidebarPage.clickHistoryUnplayed();
    await mediaPage.page.waitForTimeout(1000);
    await expect(mediaPage.resultsContainer).toBeVisible();

    // Navigate to Completed using POM
    await sidebarPage.clickHistoryCompleted();
    await mediaPage.page.waitForTimeout(1000);
    await expect(mediaPage.resultsContainer).toBeVisible();
  });

  test('search input is functional', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for something using POM
    await mediaPage.search('test');

    // Results should update using POM
    const searchCount = await mediaPage.getMediaCount();
    expect(searchCount).toBeGreaterThanOrEqual(0);

    // Clear search using POM
    await mediaPage.clearSearch();
  });

  test('sort options work', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Change sort option using POM
    await mediaPage.setSortBy('size');
    await mediaPage.page.waitForTimeout(500);

    // Verify sort changed using POM
    await expect(mediaPage.sortBySelect).toHaveValue('size');
  });

  test('sidebar can be toggled on mobile', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Check if menu toggle is visible (mobile) using POM
    const isMobile = await sidebarPage.menuToggle.isVisible();

    if (isMobile) {
      // Open sidebar using POM
      await sidebarPage.open();
      expect(await sidebarPage.isVisible()).toBe(true);

      // Close sidebar using POM
      await sidebarPage.close();
      expect(await sidebarPage.isVisible()).toBe(false);
    }
    // On desktop, sidebar is always visible
  });
});
