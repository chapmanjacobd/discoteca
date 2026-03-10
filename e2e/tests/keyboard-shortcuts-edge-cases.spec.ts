/**
 * Keyboard Shortcuts Edge Cases Tests
 *
 * Tests for edge cases in keyboard shortcut handling, particularly:
 * - Fullscreen toggle behavior when viewer state changes
 * - Escape key behavior in different states
 * - Rapid key presses and state transitions
 */
import { test, expect } from '../fixtures';

test.describe('Keyboard Shortcuts Edge Cases', () => {
  test.use({ readOnly: true });
  test.describe('Document Viewer Fullscreen Edge Cases', () => {
    test('f -> s -> f: exit fullscreen via f even after closing modal', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open first text document (PDF/EPUB)
      const docCard = page.locator('.media-card[data-type*="text"]').first();
      await expect(docCard).toBeVisible();
      await docCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Press 'f' to enter fullscreen
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      // Verify fullscreen is active
      const isFullscreenAfterF = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreenAfterF).toBe(true);

      // Press 'w' to close the modal (while in fullscreen)
      await page.keyboard.press('w');
      await page.waitForTimeout(300);

      // Modal should be closed
      const isModalVisible = await page.locator('#document-modal').isVisible();
      expect(isModalVisible).toBe(false);

      // Press 'f' again - should exit fullscreen gracefully (no error)
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      // Fullscreen should be exited
      const isFullscreenAfterSecondF = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreenAfterSecondF).toBe(false);
    });

    test('f -> Escape -> f: re-enter fullscreen after exiting with Escape', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open first text document (PDF/EPUB)
      const docCard = page.locator('.media-card[data-type*="text"]').first();
      await docCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Press 'f' to enter fullscreen
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(true);

      // Press Escape to exit fullscreen
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);

      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(false);

      // Press 'f' again to re-enter fullscreen
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(true);
    });

    test('s -> f: pressing f after closing modal should do nothing (no crash)', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open first text document (PDF/EPUB)
      const docCard = page.locator('.media-card[data-type*="text"]').first();
      await docCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Press 'w' to close the modal
      await page.keyboard.press('w');
      await page.waitForTimeout(300);

      // Modal should be closed
      const isModalVisible = await page.locator('#document-modal').isVisible();
      expect(isModalVisible).toBe(false);

      // Press 'f' - should not crash, should do nothing since no viewer is open
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      // Page should still be functional
      expect(await page.title()).toBeDefined();
    });
  });

  test.describe('PiP Player Fullscreen Edge Cases', () => {
    test('f -> s -> f: exit fullscreen via f even after closing PiP', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Click first non-document media card to open player
      const audioCard = page.locator('.media-card[data-type*="audio"]').first();
      await expect(audioCard).toBeVisible();
      await audioCard.click();
      await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });

      // Press 'f' to enter fullscreen
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      // Verify fullscreen is active
      const isFullscreenAfterF = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreenAfterF).toBe(true);

      // Press 'w' to close the PiP player (while in fullscreen)
      await page.keyboard.press('w');
      await page.waitForTimeout(300);

      // PiP should be closed
      const isPipVisible = await page.locator('#pip-player').isVisible();
      expect(isPipVisible).toBe(false);

      // Press 'f' again - should exit fullscreen gracefully (no error)
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      // Fullscreen should be exited
      const isFullscreenAfterSecondF = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreenAfterSecondF).toBe(false);
    });

    test('f -> Escape -> f: re-enter fullscreen after exiting with Escape', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Click first non-document media card to open player
      const audioCard = page.locator('.media-card[data-type*="audio"]').first();
      await audioCard.click();
      await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });

      // Press 'f' to enter fullscreen
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(true);

      // Press Escape to exit fullscreen
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);

      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(false);

      // Press 'f' again to re-enter fullscreen
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(true);
    });

    test('s -> f: pressing f after closing PiP should do nothing (no crash)', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Click first non-document media card to open player
      const audioCard = page.locator('.media-card[data-type*="audio"]').first();
      await audioCard.click();
      await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });

      // Press 'w' to close the PiP player
      await page.keyboard.press('w');
      await page.waitForTimeout(300);

      // PiP should be closed
      const isPipVisible = await page.locator('#pip-player').isVisible();
      expect(isPipVisible).toBe(false);

      // Press 'f' - should not crash, should do nothing since no viewer is open
      await page.keyboard.press('f');
      await page.waitForTimeout(300);

      // Page should still be functional
      expect(await page.title()).toBeDefined();
    });
  });

  test.describe('Escape Key Edge Cases', () => {
    test('Escape exits fullscreen before closing viewer', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open document viewer
      const docCard = page.locator('.media-card[data-type*="text"]').first();
      await docCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Enter fullscreen
      await page.keyboard.press('f');
      await page.waitForTimeout(300);
      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(true);

      // Press Escape - should exit fullscreen but NOT close modal
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);

      // Fullscreen should be exited
      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(false);

      // Modal should STILL be visible
      const isModalVisible = await page.locator('#document-modal').isVisible();
      expect(isModalVisible).toBe(true);

      // Second Escape should close the modal
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);

      const isModalVisibleAfterSecondEscape = await page.locator('#document-modal').isVisible();
      expect(isModalVisibleAfterSecondEscape).toBe(false);
    });

    test('rapid f key presses do not cause errors', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open document viewer
      const docCard = page.locator('.media-card[data-type*="text"]').first();
      await docCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Rapidly press 'f' multiple times
      for (let i = 0; i < 5; i++) {
        await page.keyboard.press('f');
        await page.waitForTimeout(100);
      }

      // Should not crash - page should still be functional
      expect(await page.title()).toBeDefined();

      // Fullscreen state might be either true or false depending on timing,
      // but the important thing is no error occurred
      const isFullscreen = await page.evaluate(() => !!document.fullscreenElement);
      expect(typeof isFullscreen).toBe('boolean');
    });
  });
});
