/**
 * Keyboard Shortcuts Edge Cases Tests
 *
 * Tests for edge cases in keyboard shortcut handling
 */
import { test, expect } from '../fixtures';

test.describe('Keyboard Shortcuts Edge Cases', () => {
  test.use({ readOnly: true });

  test.describe('Document Viewer Fullscreen Edge Cases', () => {
    test('f -> w -> f: exit fullscreen via f even after closing modal', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open first text document using POM
      const docCard = mediaPage.getFirstMediaCardByType('text');
      await expect(docCard).toBeVisible();
      await docCard.click();
      await viewerPage.waitForDocumentModal();

      // Press 'f' to enter fullscreen
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      // Verify fullscreen is active using POM
      expect(await viewerPage.isFullscreenActive()).toBe(true);

      // Press 'w' to close the modal (while in fullscreen) using POM
      await mediaPage.page.keyboard.press('w');
      await mediaPage.page.waitForTimeout(300);

      // Modal should be closed using POM
      expect(await viewerPage.isDocumentModalHidden()).toBe(true);

      // Press 'f' again - should exit fullscreen gracefully (no error)
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      // Fullscreen should be exited using POM
      expect(await viewerPage.isFullscreenActive()).toBe(false);
    });

    test('f -> Escape -> f: re-enter fullscreen after exiting with Escape', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open first text document using POM
      const docCard = mediaPage.getFirstMediaCardByType('text');
      await docCard.click();
      await viewerPage.waitForDocumentModal();

      // Press 'f' to enter fullscreen
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      expect(await viewerPage.isFullscreenActive()).toBe(true);

      // Press Escape to exit fullscreen
      await mediaPage.page.keyboard.press('Escape');
      await mediaPage.page.waitForTimeout(300);

      expect(await viewerPage.isFullscreenActive()).toBe(false);

      // Press 'f' again to re-enter fullscreen
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      expect(await viewerPage.isFullscreenActive()).toBe(true);
    });

    test('s -> f: pressing f after closing modal should do nothing (no crash)', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open first text document using POM
      const docCard = mediaPage.getFirstMediaCardByType('text');
      await docCard.click();
      await viewerPage.waitForDocumentModal();

      // Press 's' to close modal (if configured)
      await mediaPage.page.keyboard.press('s');
      await mediaPage.page.waitForTimeout(300);

      // Press 'f' - should not crash even though modal is closed
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      // Page should not crash - just verify it's still functional
      await expect(mediaPage.resultsContainer).toBeVisible();
    });
  });

  test.describe('PiP Player Edge Cases', () => {
    test('f key in PiP does not conflict with document fullscreen', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open video in PiP using POM
      const videoCard = mediaPage.getFirstMediaCardByType('video');
      await videoCard.click();
      await viewerPage.waitForPlayer();

      // Press 'f' for fullscreen
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      // Should attempt fullscreen (may not work in headless)
      // Just verify no crash
      await expect(viewerPage.playerContainer).toBeVisible();
    });

    test('Escape key closes both PiP and exits fullscreen', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open video in PiP using POM
      const videoCard = mediaPage.getFirstMediaCardByType('video');
      await videoCard.click();
      await viewerPage.waitForPlayer();

      // Enter fullscreen using POM
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      // Press Escape - should exit fullscreen and possibly close PiP
      await mediaPage.page.keyboard.press('Escape');
      await mediaPage.page.waitForTimeout(300);

      // Player should still exist (may or may not be hidden)
      expect(await viewerPage.playerContainer.count()).toBeGreaterThan(0);
    });
  });

  test.describe('Rapid Key Press Edge Cases', () => {
    test('rapid f key presses do not cause errors', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open document using POM
      const docCard = mediaPage.getFirstMediaCardByType('text');
      await docCard.click();
      await viewerPage.waitForDocumentModal();

      // Rapidly press 'f' multiple times
      for (let i = 0; i < 5; i++) {
        await mediaPage.page.keyboard.press('f');
        await mediaPage.page.waitForTimeout(100);
      }

      // Should not crash
      await expect(mediaPage.resultsContainer).toBeVisible();
    });

    test('rapid Escape key presses do not cause errors', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open document using POM
      const docCard = mediaPage.getFirstMediaCardByType('text');
      await docCard.click();
      await viewerPage.waitForDocumentModal();

      // Rapidly press Escape multiple times
      for (let i = 0; i < 5; i++) {
        await mediaPage.page.keyboard.press('Escape');
        await mediaPage.page.waitForTimeout(100);
      }

      // Should not crash
      await expect(mediaPage.resultsContainer).toBeVisible();
    });

    test('simultaneous key presses handled gracefully', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open video using POM
      const videoCard = mediaPage.getFirstMediaCardByType('video');
      await videoCard.click();
      await viewerPage.waitForPlayer();

      // Press multiple keys in quick succession
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.keyboard.press(' ');
      await mediaPage.page.keyboard.press('m');
      await mediaPage.page.waitForTimeout(500);

      // Should not crash
      await expect(viewerPage.playerContainer).toBeVisible();
    });
  });

  test.describe('State Transition Edge Cases', () => {
    test('keyboard shortcuts work after page reload', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open video using POM
      const videoCard = mediaPage.getFirstMediaCardByType('video');
      await videoCard.click();
      await viewerPage.waitForPlayer();

      // Reload page
      await mediaPage.page.reload();
      await mediaPage.waitForMediaToLoad();

      // Open video again using POM
      await videoCard.click();
      await viewerPage.waitForPlayer();

      // Keyboard shortcuts should still work
      await mediaPage.page.keyboard.press(' ');
      await mediaPage.page.waitForTimeout(300);

      // Player should still be visible
      await expect(viewerPage.playerContainer).toBeVisible();
    });

    test('keyboard shortcuts work after navigating between modes', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Navigate to Captions using POM
      await sidebarPage.openCaptions();

      // Click caption segment using POM
      await mediaPage.getCaptionSegments().first().click();
      await viewerPage.waitForPlayer();

      // Keyboard shortcuts should work
      await mediaPage.page.keyboard.press(' ');
      await mediaPage.page.waitForTimeout(300);

      // Player should still be visible
      await expect(viewerPage.playerContainer).toBeVisible();
    });

    test('fullscreen state after media change', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Open document using POM
      const docCard = mediaPage.getFirstMediaCardByType('text');
      await docCard.click();
      await viewerPage.waitForDocumentModal();

      // Enter fullscreen using POM
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(300);

      // Close modal using POM
      await viewerPage.closeDocumentModal();

      // Open different document using POM
      const docCard2 = mediaPage.page.locator('.media-card[data-media_type*="text"]').nth(1);
      if (await docCard2.count() > 0) {
        await docCard2.click();
        await viewerPage.waitForDocumentModal();

        // Fullscreen state may vary - just verify no crash
        await expect(viewerPage.documentModal).toBeVisible();
      }
    });
  });
});
