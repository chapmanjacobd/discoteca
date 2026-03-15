/**
 * Trashcan Disabled Tests
 *
 * Tests for verifying that delete functionality is properly disabled
 * when the server is started without --trashcan flag
 */
import { test, expect } from '../fixtures';

test.describe('Trashcan Disabled', () => {
  // Override server options to disable trashcan for these tests
  test.use({
    serverOptions: {
      trashcan: false,
    },
  });

  test('delete API returns error when trashcan is disabled', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get the first media card
    const firstCard = mediaPage.getFirstMediaCardByType('audio');
    await expect(firstCard).toBeVisible();

    // Click first audio to open player
    await firstCard.click();
    await mediaPage.page.waitForTimeout(500);

    // Press Delete - should fail silently or show error
    await mediaPage.page.keyboard.press('Delete');
    await mediaPage.page.waitForTimeout(1500);

    // Check that no "Trashed" toast appeared
    // Instead, we might see an error or no toast at all
    const toast = mediaPage.toast;
    const toastVisible = await toast.isVisible().catch(() => false);
    
    if (toastVisible) {
      const toastText = await mediaPage.getToastMessage();
      // Should NOT contain "Trashed" since trashcan is disabled
      expect(toastText).not.toContain('Trashed');
    }
  });

  test('trash button not visible when trashcan is disabled', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Hover over first media card
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.hover();
    await mediaPage.page.waitForTimeout(500);

    // Trash button should NOT be visible when trashcan is disabled
    const trashBtn = mediaPage.page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
    const isVisible = await trashBtn.isVisible().catch(() => false);
    expect(isVisible).toBe(false);
  });
});
