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

  test('delete in PiP deletes and plays next media', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial count
    const initialCount = await page.locator('.media-card').count();
    expect(initialCount).toBeGreaterThanOrEqual(2);

    // Get the first audio card title
    const firstCard = page.locator('.media-card[data-type*="audio"]').first();
    await expect(firstCard).toBeVisible();
    const firstTitle = await firstCard.textContent();

    // Click first audio to open player
    await firstCard.click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });
    await page.waitForTimeout(500);

    // Press Delete (without shift) - should delete and play next
    await page.keyboard.press('Delete');

    // Wait for delete toast and search refresh
    await page.waitForTimeout(2000);

    // PiP should still be visible (playing next media)
    const isPipVisible = await page.locator('#pip-player').isVisible();
    expect(isPipVisible).toBe(true);

    // Title should have changed to next media
    const newTitle = await page.locator('#media-title').textContent();
    expect(newTitle).not.toContain(firstTitle.split('/').pop());
  });

  test('shift+delete in PiP deletes and stops playback', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial count
    const initialCount = await page.locator('.media-card').count();
    expect(initialCount).toBeGreaterThanOrEqual(2);

    // Click first audio to open player
    const firstCard = page.locator('.media-card[data-type*="audio"]').first();
    await firstCard.click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });
    await page.waitForTimeout(500);

    // Press Shift+Delete - should delete and stop (close PiP)
    await page.keyboard.press('Shift+Delete');
    await page.waitForTimeout(1500);

    // PiP should be closed
    const isPipVisible = await page.locator('#pip-player').isVisible();
    expect(isPipVisible).toBe(false);

    // Check that toast appeared
    const toastVisible = await page.locator('#toast').isVisible();
    expect(toastVisible).toBe(true);
    const toastText = await page.locator('#toast').textContent();
    expect(toastText).toContain('Trashed');
  });

  test('delete in document modal deletes and plays next media', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial count
    const initialCount = await page.locator('.media-card').count();
    expect(initialCount).toBeGreaterThanOrEqual(2);

    // Find a document card in the main view (not search)
    const docCard = page.locator('.media-card[data-type*="document"], .media-card[data-type="text"]').first();

    // Click to open document modal
    await docCard.click();
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    await page.waitForTimeout(500);

    // Click on the modal header (outside iframe) to ensure focus
    await page.click('#document-modal .modal-header');
    await page.waitForTimeout(200);

    // Press Delete (without shift) - should delete and play next
    await page.keyboard.press('Delete');
    await page.waitForTimeout(2000);

    // Either PiP or document modal should be visible (modal stays open if next item is also a document)
    const isPipVisible = await page.locator('#pip-player').isVisible();
    const isModalVisible = await page.locator('#document-modal').isVisible();
    expect(isPipVisible || isModalVisible).toBe(true);

    // Check that toast appeared (delete was triggered)
    const toastVisible = await page.locator('#toast').isVisible();
    expect(toastVisible).toBe(true);
    const toastText = await page.locator('#toast').textContent();
    expect(toastText).toContain('Trashed');
  });

  test('shift+delete in document modal deletes and closes', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first text document (PDF/EPUB)
    const docCard = page.locator('.media-card[data-type*="text"]').first();
    await expect(docCard).toBeVisible();
    
    // Click to open document modal
    await docCard.click();
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

    // Modal should be visible
    expect(await page.locator('#document-modal').isVisible()).toBe(true);

    // Press Shift+Delete - should delete and close (not play next)
    await page.keyboard.press('Shift+Delete');
    await page.waitForTimeout(1500);

    // Document modal should be closed
    const isModalVisible = await page.locator('#document-modal').isVisible();
    expect(isModalVisible).toBe(false);

    // PiP should NOT be visible (stopped, not playing next)
    const isPipVisible = await page.locator('#pip-player').isVisible();
    expect(isPipVisible).toBe(false);

    // Check that toast appeared
    const toastVisible = await page.locator('#toast').isVisible();
    expect(toastVisible).toBe(true);
  });
});
