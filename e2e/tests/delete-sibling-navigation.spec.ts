/**
 * Delete Shortcut - Sibling Navigation Tests
 *
 * Tests for Delete shortcut behavior when navigating between media:
 * - Delete should play next sibling (or previous if no next)
 * - Shift+Delete should stop playback
 */
import { test, expect } from '../fixtures';

test.describe('Delete Shortcut - Sibling Navigation', () => {
  test('delete in PiP plays next sibling', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get the first audio card
    const firstCard = page.locator('.media-card[data-type*="audio"]').first();
    const firstTitleText = await firstCard.textContent();
    const firstFileName = firstTitleText.split('/').pop()?.trim();
    expect(firstFileName).toBeTruthy();

    // Click first audio to open player
    await firstCard.click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });
    await page.waitForSelector('audio', { timeout: 5000 });
    await page.waitForFunction(() => {
      const audio = document.querySelector('audio');
      return audio && audio.readyState >= 3;
    }, { timeout: 10000 });
    await page.click('audio');
    await page.waitForTimeout(500);
    await page.waitForTimeout(500);

    // Press Delete (without shift) - should delete and play next
    await page.keyboard.press('Delete');
    await page.waitForTimeout(2000);

    // PiP should still be visible (playing next media)
    const isPipVisible = await page.locator('#pip-player').isVisible();
    expect(isPipVisible).toBe(true);

    // Title should have changed to different media
    const newTitle = await page.locator('#media-title').textContent();
    expect(newTitle).not.toContain(firstFileName);
  });

  test('delete last item plays previous sibling', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get total media count
    const allCards = page.locator('.media-card');
    const totalCount = await allCards.count();
    expect(totalCount).toBeGreaterThanOrEqual(2);

    // Click the LAST media card (any type)
    const lastCard = allCards.last();
    const lastTitleText = await lastCard.textContent();
    const lastFileName = lastTitleText.split('/').pop()?.trim();
    expect(lastFileName).toBeTruthy();

    // Click last card to open player
    await lastCard.click();
    await page.waitForTimeout(1000);

    // Press Delete (without shift) - should delete and play previous (since no next)
    await page.keyboard.press('Delete');
    await page.waitForTimeout(2000);

    // Either PiP or modal should be visible (playing previous media)
    const isPipVisible = await page.locator('#pip-player').isVisible().catch(() => false);
    const isModalVisible = await page.locator('#document-modal').isVisible().catch(() => false);
    expect(isPipVisible || isModalVisible).toBe(true);

    // Title should have changed (not the deleted file)
    let newTitle = '';
    if (isPipVisible) {
      newTitle = await page.locator('#media-title').textContent();
    } else if (isModalVisible) {
      newTitle = await page.locator('#document-title').textContent();
    }
    expect(newTitle).not.toContain(lastFileName);
  });

  test('delete in document modal plays next sibling', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find first document card
    const firstDoc = page.locator('.media-card[data-type*="document"], .media-card[data-type="text"]').first();
    expect(await firstDoc.count()).toBeGreaterThan(0);

    const firstTitleText = await firstDoc.textContent();
    const firstFileName = firstTitleText.split('/').pop()?.trim();
    expect(firstFileName).toBeTruthy();

    // Click to open modal
    await firstDoc.click();
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    await page.waitForTimeout(500);

    // Click on the modal header (outside iframe) to ensure focus
    await page.click('#document-modal .modal-header');
    await page.waitForTimeout(100);

    // Press Delete (without shift) - should delete and play next sibling
    await page.keyboard.press('Delete');
    await page.waitForTimeout(2000);

    // Either PiP or document modal should be visible
    const isPipVisible = await page.locator('#pip-player').isVisible().catch(() => false);
    const isModalVisible = await page.locator('#document-modal').isVisible().catch(() => false);
    expect(isPipVisible || isModalVisible).toBe(true);

    // Title should have changed
    let newTitle = '';
    if (isPipVisible) {
      newTitle = await page.locator('#media-title').textContent();
    } else if (isModalVisible) {
      newTitle = await page.locator('#document-title').textContent();
    }
    expect(newTitle).not.toContain(firstFileName);

    // Close modal if open
    if (isModalVisible) {
      await page.click('#document-modal .close-modal, #document-modal .modal-close, button[aria-label="Close"]');
    }
  });

  test('shift+delete in PiP stops playback', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first audio to open player
    const firstCard = page.locator('.media-card[data-type*="audio"]').first();
    await firstCard.click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });
    await page.waitForTimeout(500);

    // Press Shift+Delete - should delete and stop
    await page.keyboard.press('Shift+Delete');
    await page.waitForTimeout(1500);

    // PiP should be closed
    const isPipVisible = await page.locator('#pip-player').isVisible();
    expect(isPipVisible).toBe(false);

    // Toast should appear
    const toastVisible = await page.locator('#toast').isVisible();
    expect(toastVisible).toBe(true);
  });

  test('shift+delete in document modal closes without playing', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first document to open modal
    const firstDoc = page.locator('.media-card[data-type*="document"], .media-card[data-type="text"]').first();
    await firstDoc.click();
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    await page.waitForTimeout(500);

    // Press Shift+Delete - should delete and close (no playback)
    await page.keyboard.press('Shift+Delete');
    await page.waitForTimeout(1500);

    // Document modal should be closed
    const isModalVisible = await page.locator('#document-modal').isVisible();
    expect(isModalVisible).toBe(false);

    // PiP should NOT be visible
    const isPipVisible = await page.locator('#pip-player').isVisible();
    expect(isPipVisible).toBe(false);

    // Toast should appear
    const toastVisible = await page.locator('#toast').isVisible();
    expect(toastVisible).toBe(true);
  });
});
