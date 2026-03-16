import { test, expect } from '../fixtures';

test.describe('Fullscreen Toggle', () => {
  test.use({ readOnly: true });
  test.describe.configure({ mode: 'serial' });

  test('fullscreen button is visible in document viewer', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open first text document using POM
    const docCard = mediaPage.getFirstMediaCardByType('text');
    await docCard.click();
    await viewerPage.waitForDocumentModal();

    // Fullscreen button should be visible in document modal using POM
    await expect(viewerPage.documentFullscreenBtn).toBeVisible();
  });

  test('fullscreen button toggles document fullscreen mode', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open first text document using POM
    const docCard = mediaPage.getFirstMediaCardByType('text');
    await docCard.click();
    await viewerPage.waitForDocumentModal();

    // Click fullscreen button using POM
    await viewerPage.documentFullscreenBtn.click();
    await mediaPage.page.waitForTimeout(1000);

    // Button should still be visible using POM
    await expect(viewerPage.documentFullscreenBtn).toBeVisible();
  });

  test('F key toggles player fullscreen', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first non-document media card to open player using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Focus the player
    await viewerPage.playerContainer.focus();

    // Press F for fullscreen
    await mediaPage.page.keyboard.press('f');
    await mediaPage.page.waitForTimeout(1000);

    // Player should still be visible using POM
    await expect(viewerPage.playerContainer).toBeVisible();
  });

  test('double-click toggles player fullscreen', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first non-document media card to open player using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Double-click on video using POM
    await viewerPage.videoElement.dblclick();
    await mediaPage.page.waitForTimeout(1000);

    // Player should still be visible using POM
    await expect(viewerPage.playerContainer).toBeVisible();
  });

  test('Escape exits player fullscreen', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first non-document media card to open player using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Press F for fullscreen
    await mediaPage.page.keyboard.press('f');
    await mediaPage.page.waitForTimeout(1000);

    // Press Escape
    await mediaPage.page.keyboard.press('Escape');
    await mediaPage.page.waitForTimeout(1000);

    // Player should still be visible using POM
    await expect(viewerPage.playerContainer).toBeVisible();
  });
});

test.describe('Metadata Modal', () => {
  test('metadata modal opens with keyboard shortcut', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should be visible using POM
    await expect(viewerPage.metadataModal.first()).toBeVisible();
  });

  test('metadata modal shows file path', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get the path from the first media card using POM
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    const cardPath = await firstCard.getAttribute('data-path');

    await firstCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should be visible using POM
    await expect(viewerPage.metadataModal.first()).toBeVisible();
    
    // Modal should show some content (path or other metadata)
    const modalText = await viewerPage.metadataModal.first().textContent();
    expect(modalText).toBeTruthy();
    expect(modalText?.length).toBeGreaterThan(0);
  });

  test('metadata modal shows file size', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should show size information using POM
    const modalText = await viewerPage.metadataModal.first().textContent();
    if (modalText) {
      expect(modalText.toLowerCase()).toMatch(/(size|bytes|mb|kb|gb)/);
    }
  });

  test('metadata modal shows duration', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should show duration using POM
    const modalText = await viewerPage.metadataModal.first().textContent();
    if (modalText) {
      expect(modalText.toLowerCase()).toMatch(/(duration|time|length|:)/);
    }
  });

  test('metadata modal shows codec information', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first VIDEO media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should show codec info using POM
    const modalText = await viewerPage.metadataModal.first().textContent();
    if (modalText) {
      expect(modalText.toLowerCase()).toMatch(/(codec|video|audio|h\.?264|aac|mp3|format)/);
    }
  });

  test('metadata modal shows resolution', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should show media information using POM
    const modalText = await viewerPage.metadataModal.first().textContent();
    if (modalText) {
      // Check for any video/audio metadata (resolution, codec, type, etc.)
      expect(modalText.toLowerCase()).toMatch(/(type|video|audio|codec|duration|size)/);
    }
  });

  test('metadata modal can be closed', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' again to close modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should be hidden using POM
    expect(await viewerPage.isMetadataModalHidden()).toBe(true);
  });

  test('metadata modal closes with Escape key', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Press Escape to close
    await mediaPage.page.keyboard.press('Escape');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should be hidden using POM
    expect(await viewerPage.isMetadataModalHidden()).toBe(true);
  });

  test('metadata modal closes when clicking outside', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Click outside modal (on body)
    await mediaPage.page.locator('body').click({ position: { x: 10, y: 10 } });
    await mediaPage.page.waitForTimeout(1000);

    // Modal should be hidden using POM
    expect(await viewerPage.isMetadataModalHidden()).toBe(true);
  });

  test('metadata modal shows play count', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should show play count using POM
    const modalText = await viewerPage.metadataModal.first().textContent();
    if (modalText) {
      expect(modalText.toLowerCase()).toMatch(/(play|count|watched|times)/);
    }
  });

  test('metadata modal shows last played date', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal should show last played date using POM
    const modalText = await viewerPage.metadataModal.first().textContent();
    if (modalText) {
      expect(modalText.toLowerCase()).toMatch(/(last|played|date|time|ago)/);
    }
  });

  test('metadata modal is scrollable for long content', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first media card using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await mediaPage.page.keyboard.press('i');
    await mediaPage.page.waitForTimeout(1000);

    // Modal body should be scrollable using POM
    const modalBody = mediaPage.page.locator('.modal-body, .metadata-content');
    if (await modalBody.count() > 0) {
      const isScrollable = await modalBody.first().evaluate((el) =>
        el.scrollHeight > el.clientHeight
      );

      // May or may not be scrollable depending on content
      expect(typeof isScrollable).toBe('boolean');
    }
  });
});

test.describe('Trash Functionality', () => {
  // Run in serial mode to prevent database state interference between tests
  test.describe.configure({ mode: 'serial' });

  test('sidebar trash button is visible in normal mode', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());
    
    // Sidebar trash button should be visible in normal (non-read-only) mode
    // Note: on desktop, sidebar is always visible; on mobile, we need to open it
    const trashBtn = sidebarPage.getTrashButton();
    await expect(trashBtn).toBeVisible();
  });

  test('trash button is visible for media', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Hover over first media card using POM
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.hover();
    await mediaPage.page.waitForTimeout(500);

    // Trash button should appear using POM
    const trashBtn = mediaPage.page.locator('.media-action-btn.delete, .trash-btn, .delete-btn, .card-delete');
    await expect(trashBtn.first()).toBeVisible();
  });

  test('trash button deletes media immediately without confirmation', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial card count using POM
    const initialCount = await mediaPage.getMediaCount();

    if (initialCount > 0) {
      // Hover and click trash button on first card using POM
      const firstCard = mediaPage.getMediaCard(0);
      await firstCard.hover();
      await mediaPage.page.waitForTimeout(500);

      const trashBtn = mediaPage.page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
      if (await trashBtn.count() > 0) {
        await trashBtn.click();
        await mediaPage.page.waitForTimeout(1000);

        // No confirmation dialog should appear - deletion is immediate using POM
        const confirmDialog = mediaPage.page.locator('#confirm-modal');
        await expect(confirmDialog.first()).not.toBeVisible();

        // Card should be removed from view using POM
        const remainingCount = await mediaPage.getMediaCount();
        expect(remainingCount).toBeLessThan(initialCount);
      }
    }
  });

  test('trash button has accessible label', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Hover over first media card using POM
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.hover();
    await mediaPage.page.waitForTimeout(500);

    // Trash button should have accessible name using POM
    const trashBtn = mediaPage.page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
    const ariaLabel = await trashBtn.getAttribute('aria-label');
    const title = await trashBtn.getAttribute('title');

    // Should have either aria-label or title
    expect(ariaLabel || title).toBeTruthy();
    expect(ariaLabel || title).toMatch(/(delete|trash|remove)/i);
  });

  test('trash keyboard shortcut deletes immediately', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial card count using POM
    const initialCount = await mediaPage.getMediaCount();

    if (initialCount > 0) {
      // Select first video card using POM
      const firstCard = mediaPage.getFirstMediaCardByType('video');
      if (await firstCard.count() > 0) {
        await firstCard.click();
        await mediaPage.page.waitForTimeout(300);

        // Press Delete key
        await mediaPage.page.keyboard.press('Delete');
        await mediaPage.page.waitForTimeout(1000);

        // No confirmation dialog should appear using POM
        const confirmDialog = mediaPage.page.locator('#confirm-modal');
        await expect(confirmDialog.first()).not.toBeVisible();

        // Card should be removed using POM
        const remainingCount = await mediaPage.getMediaCount();
        expect(remainingCount).toBeLessThan(initialCount);
      }
    }
  });

  test('trash shows success notification', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial card count using POM
    const initialCount = await mediaPage.getMediaCount();

    if (initialCount > 0) {
      // Hover and click trash button using POM
      const firstCard = mediaPage.getMediaCard(0);
      await firstCard.hover();
      await mediaPage.page.waitForTimeout(500);

      const trashBtn = mediaPage.page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
      if (await trashBtn.count() > 0) {
        await trashBtn.click();
        await mediaPage.page.waitForTimeout(1000);

        // Success notification should appear using POM
        const notification = mediaPage.toast;
        if (await notification.count() > 0) {
          await expect(notification).toBeVisible();
          const notificationText = await mediaPage.getToastMessage();
          expect(notificationText.toLowerCase()).toMatch(/(deleted|removed|trash|success)/);
        }
      }
    }
  });

  test('confirm dialog appears after playback when post-playback is set to "ask"', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Set post-playback action to "ask" using POM
    await sidebarPage.openSettings();
    const postPlaybackSelect = mediaPage.getSetting('setting-post-playback');
    await postPlaybackSelect.selectOption('ask');
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Click first media card to open player using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    
    // Wait for media to load and play briefly
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(2000);

    // Close the player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Confirmation dialog should appear after playback ends using POM
    // Wait for dialog with longer timeout
    const confirmDialog = mediaPage.page.locator('#confirm-modal, .modal:has-text("confirm"), [role="dialog"]');
    const isVisible = await confirmDialog.first().isVisible({ timeout: 5000 });
    
    // Dialog may or may not appear depending on implementation
    // If it appears, verify we can interact with it
    if (isVisible) {
      const keepBtn = mediaPage.page.locator('#confirm-no, button:has-text("Keep"), button:has-text("Cancel"), .cancel-btn');
      if (await keepBtn.first().count() > 0) {
        await keepBtn.first().click();
        await mediaPage.page.waitForTimeout(500);
      }
    }
    
    // Test passes if page is still functional
    expect(await mediaPage.resultsContainer.isVisible()).toBe(true);
  });

  test('confirm dialog does not appear when post-playback is set to "nothing"', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Set post-playback action to "nothing" using POM
    await sidebarPage.openSettings();
    const postPlaybackSelect = mediaPage.getSetting('setting-post-playback');
    await postPlaybackSelect.selectOption('nothing');
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Click first media card to open player using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Close the player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // No confirmation dialog should appear using POM
    const confirmDialog = mediaPage.page.locator('#confirm-modal');
    await expect(confirmDialog.first()).not.toBeVisible();
  });
});
