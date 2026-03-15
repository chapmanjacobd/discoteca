/**
 * Read-Only Mode Tests
 *
 * Tests for verifying that modification functionality is properly disabled
 * when the server is started with --read-only flag
 */
import { test, expect } from '../fixtures';

test.describe('Read-Only Mode', () => {
  // Override server options to enable read-only mode for these tests
  test.use({
    serverOptions: {
      readOnly: true,
    },
  });

  test('delete API returns error in read-only mode', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get the first media card
    const firstCard = mediaPage.getFirstMediaCardByType('audio');
    await expect(firstCard).toBeVisible();

    // Click first audio to open player
    await firstCard.click();
    await mediaPage.page.waitForTimeout(500);

    // Press Delete - should fail with read-only error
    await mediaPage.page.keyboard.press('Delete');
    await mediaPage.page.waitForTimeout(1500);

    // Check that no "Trashed" toast appeared
    const toast = mediaPage.toast;
    const toastVisible = await toast.isVisible().catch(() => false);

    if (toastVisible) {
      const toastText = await mediaPage.getToastMessage();
      // Should NOT contain "Trashed" since read-only mode is enabled
      expect(toastText).not.toContain('Trashed');
      // Should contain "Read-only" or "Access Denied"
      expect(toastText).toMatch(/Read-only|Access Denied/);
    }
  });

  test('trash button not visible in read-only mode', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Hover over first media card
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.hover();
    await mediaPage.page.waitForTimeout(500);

    // Trash button should NOT be visible in read-only mode
    const trashBtn = mediaPage.page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
    const isVisible = await trashBtn.isVisible().catch(() => false);
    expect(isVisible).toBe(false);
  });

  test('playlist modification buttons not visible in read-only mode', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Hover over first media card
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.hover();
    await mediaPage.page.waitForTimeout(500);

    // Add to playlist button should NOT be visible in read-only mode
    const addPlaylistBtn = mediaPage.page.locator('.media-action-btn.add-playlist').first();
    const isAddVisible = await addPlaylistBtn.isVisible().catch(() => false);
    expect(isAddVisible).toBe(false);
  });

  test('mark played/unplayed buttons still visible in read-only mode', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Hover over first media card
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.hover();
    await mediaPage.page.waitForTimeout(500);

    // Mark played/unplayed buttons should still be visible (client-side only)
    // Note: The server will reject the request, but the UI shows the button
    const markBtn = mediaPage.page.locator('.media-action-btn.mark-played, .media-action-btn.mark-unplayed').first();
    const isVisible = await markBtn.isVisible().catch(() => false);
    expect(isVisible).toBe(true);
  });
});
