import { test, expect } from '../fixtures';

test.describe('Page Navigation', () => {
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

    // Should show DU toolbar
    await expect(mediaPage.getDUTToolbar()).toBeVisible();
    await expect(mediaPage.getDUPathInput()).toBeVisible();
  });

  test('navigates to Captions view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Use sidebar POM to navigate to Captions view
    await sidebarPage.openCaptions();

    // Should show captions - using POM locator
    await expect(mediaPage.getCaptionCards().first()).toBeVisible();
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

    // Should start in grid view
    await expect(mediaPage.viewGridButton).toHaveClass(/active/);

    // Switch to details view using POM
    await mediaPage.switchToDetailsView();
    await expect(mediaPage.viewDetailsButton).toHaveClass(/active/);

    // Switch back to grid using POM
    await mediaPage.switchToGridView();
    await expect(mediaPage.viewGridButton).toHaveClass(/active/);
  });
});
