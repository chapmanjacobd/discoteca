/**
 * Delete Shortcut Tests
 *
 * Tests for the Delete keyboard shortcut behavior in different contexts:
 * - PiP player: Delete deletes and plays next, Shift+Delete deletes and stops
 * - Document modal: Delete deletes and plays next, Shift+Delete deletes and closes
 */
import { test, expect } from '../fixtures';

test.describe('Delete Shortcut', () => {
  test.describe.configure({ mode: 'serial' });

  test('delete in PiP deletes and plays next media', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();
    expect(initialCount).toBeGreaterThanOrEqual(2);

    // Get the first audio card title using POM
    const firstCard = mediaPage.getFirstMediaCardByType('audio');
    await expect(firstCard).toBeVisible();
    const firstTitle = await firstCard.textContent();

    // Click first audio to open player using POM
    await firstCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(500);

    // Press Delete (without shift) - should delete and play next
    await mediaPage.page.keyboard.press('Delete');

    // Wait for delete toast and search refresh using POM
    await mediaPage.page.waitForTimeout(2000);

    // PiP should still be visible (playing next media) using POM
    expect(await viewerPage.isOpen()).toBe(true);

    // Title should have changed to next media using POM
    const newTitle = await viewerPage.getTitle();
    expect(newTitle).not.toContain(firstTitle?.split('/').pop());
  });

  test('shift+delete in PiP deletes and stops playback', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();
    expect(initialCount).toBeGreaterThanOrEqual(2);

    // Click first audio to open player using POM
    const firstCard = mediaPage.getFirstMediaCardByType('audio');
    await firstCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(500);

    // Press Shift+Delete - should delete and stop (close PiP)
    await mediaPage.page.keyboard.press('Shift+Delete');
    await mediaPage.page.waitForTimeout(1500);

    // PiP should be closed using POM
    expect(await viewerPage.isHidden()).toBe(true);

    // Check that toast appeared using POM
    await mediaPage.waitForToast();
    const toastText = await mediaPage.getToastMessage();
    expect(toastText).toContain('Trashed');
  });

  test('delete in document modal deletes and plays next media', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();
    expect(initialCount).toBeGreaterThanOrEqual(2);

    // Find a document card in the main view using POM
    const docCard = mediaPage.page.locator('.media-card[data-type*="document"], .media-card[data-type="text"]').first();

    // Click to open document modal using POM
    await docCard.click();
    await viewerPage.waitForDocumentModal();
    await mediaPage.page.waitForTimeout(500);

    // Click on the modal header (outside iframe) to ensure focus
    await viewerPage.documentModal.locator('.modal-header').click();
    await mediaPage.page.waitForTimeout(200);

    // Press Delete (without shift) - should delete and play next
    await mediaPage.page.keyboard.press('Delete');
    await mediaPage.page.waitForTimeout(2000);

    // Either PiP or document modal should be visible using POM
    const isPipVisible = await viewerPage.isOpen();
    const isModalVisible = await viewerPage.isDocumentModalVisible();
    expect(isPipVisible || isModalVisible).toBe(true);

    // Check that toast appeared (delete was triggered) using POM
    await mediaPage.waitForToast();
    const toastText = await mediaPage.getToastMessage();
    expect(toastText).toContain('Trashed');
  });

  test('shift+delete in document modal deletes and closes', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open first text document (PDF/EPUB) using POM
    const docCard = mediaPage.getFirstMediaCardByType('text');
    await expect(docCard).toBeVisible();

    // Click to open document modal using POM
    await docCard.click();
    await viewerPage.waitForDocumentModal();

    // Modal should be visible using POM
    expect(await viewerPage.isDocumentModalVisible()).toBe(true);

    // Press Shift+Delete - should delete and close (not play next)
    await mediaPage.page.keyboard.press('Shift+Delete');
    await mediaPage.page.waitForTimeout(1500);

    // Document modal should be closed using POM
    expect(await viewerPage.isDocumentModalHidden()).toBe(true);

    // PiP should NOT be visible (stopped, not playing next) using POM
    expect(await viewerPage.isHidden()).toBe(true);
  });
});
