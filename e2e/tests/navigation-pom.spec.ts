import { test, expect } from '../fixtures';

test.describe('Page Navigation with POM', () => {
  test.use({ readOnly: true });

  test('loads the home page', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Verify key elements are present using POM
    await expect(mediaPage.searchInput).toBeVisible();
    await expect(mediaPage.resultsContainer).toBeVisible();
    
    // Verify media cards are loaded
    await expect(mediaPage.mediaCards.first()).toBeVisible();
  });

  test('navigates to Disk Usage view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Use sidebar POM to navigate to DU view
    await sidebarPage.openDiskUsage();

    // Should show DU toolbar
    await expect(mediaPage.page.locator('#du-toolbar')).toBeVisible();
    // Path input is hidden on mobile in favour of breadcrumbs
    if (await mediaPage.page.locator('#du-path-input').isVisible()) {
      await expect(mediaPage.page.locator('#du-path-input')).toBeVisible();
    } else {
      await expect(mediaPage.page.locator('#du-breadcrumbs')).toBeVisible();
    }
  });

  test('navigates to Captions view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Use sidebar POM to navigate to Captions view
    await sidebarPage.openCaptions();

    // Should show captions - using POM locator
    await expect(mediaPage.page.locator('.caption-media-card').first()).toBeVisible();
  });

  test('opens and closes settings modal', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Use sidebar POM to open settings
    await sidebarPage.openSettings();
    
    const modal = mediaPage.page.locator('#settings-modal');
    await expect(modal).toBeVisible();

    // Use sidebar POM to close settings
    await sidebarPage.closeSettings();
    await expect(modal).not.toBeVisible();
  });

  test('toggles view modes (grid/details)', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Should start in grid view
    await expect(mediaPage.viewGridButton).toHaveClass(/active/);

    // Switch to details view using POM
    await mediaPage.switchToDetailsView();
    await expect(mediaPage.viewDetailsButton).toHaveClass(/active/);

    // Switch back to grid using POM
    await mediaPage.switchToGridView();
    await expect(mediaPage.viewGridButton).toHaveClass(/active/);
  });

  test('search functionality works', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count
    const initialCount = await mediaPage.getMediaCount();
    expect(initialCount).toBeGreaterThan(0);

    // Wait for API response when searching
    const [response] = await Promise.all([
      mediaPage.page.waitForResponse(resp => resp.url().includes('/api/query')),
      mediaPage.search('test'),
    ]);
    expect(response.ok()).toBe(true);
    
    // Results should be different (or same if nothing matches)
    const newCount = await mediaPage.getMediaCount();
    expect(newCount).toBeLessThanOrEqual(initialCount);

    // Clear search and wait for results to update
    const [clearResponse] = await Promise.all([
      mediaPage.page.waitForResponse(resp => resp.url().includes('/api/query')),
      mediaPage.clearSearch(),
    ]);
    expect(clearResponse.ok()).toBe(true);
  });

  test('media cards have correct data attributes', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // First card should have data-media_type attribute
    const firstCard = mediaPage.getMediaCard(0);
    await expect(firstCard).toHaveAttribute('data-media_type');
  });

  test('sidebar can be toggled on mobile', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Check if menu toggle is visible (mobile)
    const isMobile = await sidebarPage.menuToggle.isVisible();
    
    if (isMobile) {
      // Open sidebar
      await sidebarPage.open();
      expect(await sidebarPage.isVisible()).toBe(true);
      
      // Close sidebar
      await sidebarPage.close();
      expect(await sidebarPage.isVisible()).toBe(false);
    }
    // On desktop, sidebar is always visible
  });
});
