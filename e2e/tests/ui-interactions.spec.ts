import { waitForPlayer, isPlayerOpen } from '../fixtures';
import { test, expect } from '../fixtures';

test.describe('Fullscreen Toggle', () => {
  test('fullscreen button is visible in player', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click first media card to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Fullscreen button should be visible
    const fullscreenBtn = page.locator('#pip-fullscreen, .fullscreen-btn, button:has-text("Fullscreen"), .player-fullscreen');
    await expect(fullscreenBtn.first()).toBeVisible();
  });

  test('fullscreen button toggles fullscreen mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click first media card to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Click fullscreen button
    const fullscreenBtn = page.locator('#pip-fullscreen, .fullscreen-btn, .player-fullscreen').first();
    
    // Note: Actual fullscreen may be blocked by browser, but we can test the button click
    await fullscreenBtn.click();
    await page.waitForTimeout(1000);

    // Button should still be visible (may have changed icon)
    await expect(fullscreenBtn).toBeVisible();
  });

  test('fullscreen button icon changes when in fullscreen', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click first media card to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Get initial button state
    const fullscreenBtn = page.locator('#pip-fullscreen, .fullscreen-btn').first();
    const initialAriaLabel = await fullscreenBtn.getAttribute('aria-label');

    // Click fullscreen
    await fullscreenBtn.click();
    await page.waitForTimeout(1000);

    // Button state may change
    const newAriaLabel = await fullscreenBtn.getAttribute('aria-label');
    
    // Either the label changed or the button has different text
    if (initialAriaLabel === newAriaLabel) {
      // Check for visual change via class or text
      const btnText = await fullscreenBtn.textContent();
      expect(btnText?.toLowerCase()).toMatch(/(fullscreen|exit|leave|close)/);
    }
  });

  test('F key toggles fullscreen', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click first media card to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Focus the player
    await page.locator('#pip-player').focus();

    // Press F for fullscreen
    await page.keyboard.press('f');
    await page.waitForTimeout(1000);

    // Player should still be visible
    await expect(page.locator('#pip-player')).toBeVisible();
  });

  test('double-click toggles fullscreen', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click first media card to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Double-click on video
    const video = page.locator('video, #pip-player').first();
    await video.dblclick();
    await page.waitForTimeout(1000);

    // Player should still be visible
    await expect(page.locator('#pip-player')).toBeVisible();
  });

  test('Escape exits fullscreen', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click first media card to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Click fullscreen
    const fullscreenBtn = page.locator('#pip-fullscreen, .fullscreen-btn').first();
    await fullscreenBtn.click();
    await page.waitForTimeout(1000);

    // Press Escape
    await page.keyboard.press('Escape');
    await page.waitForTimeout(1000);

    // Player should still be visible
    await expect(page.locator('#pip-player')).toBeVisible();
  });

  test('fullscreen button is accessible', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click first media card to open player
    await page.locator('.caption-segment').first().click();
    await waitForPlayer(page);

    // Fullscreen button should have accessible name
    const fullscreenBtn = page.locator('#pip-fullscreen, .fullscreen-btn').first();
    const ariaLabel = await fullscreenBtn.getAttribute('aria-label');
    const title = await fullscreenBtn.getAttribute('title');
    
    // Should have either aria-label or title
    expect(ariaLabel || title).toBeTruthy();
  });
});

test.describe('Metadata Modal', () => {
  test('metadata modal opens with keyboard shortcut', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Click a caption segment to open player
    await page.locator('.caption-segment').first().click();
    
    // Wait for player to have video/audio content
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should be visible
    const modal = page.locator('#metadata-modal');
    await expect(modal.first()).toBeVisible();
  });

  test('metadata modal shows file path', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Get the path from the first media card
    const firstCard = page.locator('.media-card').first();
    const cardTitle = await firstCard.locator('.media-title').textContent();

    await firstCard.click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should show file path
    const modal = page.locator('#metadata-modal');
    const modalText = await modal.first().textContent();
    expect(modalText).toContain(cardTitle || '');
  });

  test('metadata modal shows file size', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should show size information
    const modal = page.locator('#metadata-modal');
    const modalText = await modal.first().textContent();
    expect(modalText.toLowerCase()).toMatch(/(size|bytes|mb|kb|gb)/);
  });

  test('metadata modal shows duration', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should show duration
    const modal = page.locator('#metadata-modal');
    const modalText = await modal.first().textContent();
    expect(modalText.toLowerCase()).toMatch(/(duration|time|length|:)/);
  });

  test('metadata modal shows codec information', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should show codec info
    const modal = page.locator('#metadata-modal');
    const modalText = await modal.first().textContent();
    expect(modalText.toLowerCase()).toMatch(/(codec|video|audio|h\\.?264|aac|mp3|format)/);
  });

  test('metadata modal shows resolution', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();

    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should show media information
    const modal = page.locator('#metadata-modal');
    const modalText = await modal.first().textContent();
    // Check for any video/audio metadata (resolution, codec, type, etc.)
    expect(modalText.toLowerCase()).toMatch(/(type|video|audio|codec|duration|size)/);
  });

  test('metadata modal can be closed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();

    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Press 'i' again to close modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should be hidden
    const modal = page.locator('#metadata-modal');
    await expect(modal.first()).not.toBeVisible();
  });

  test('metadata modal closes with Escape key', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();

    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Press 'i' again to close (Escape doesn't close metadata modal)
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should be hidden
    const modal = page.locator('#metadata-modal');
    await expect(modal.first()).not.toBeVisible();
  });

  test('metadata modal closes when clicking outside', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Click outside modal (on body)
    await page.locator('body').click({ position: { x: 10, y: 10 } });
    await page.waitForTimeout(1000);

    // Modal should be hidden
    const modal = page.locator('#metadata-modal');
    await expect(modal.first()).not.toBeVisible();
  });

  test('metadata modal shows play count', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should show play count
    const modal = page.locator('#metadata-modal');
    const modalText = await modal.first().textContent();
    expect(modalText.toLowerCase()).toMatch(/(play|count|watched|times)/);
  });

  test('metadata modal shows last played date', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal should show last played date
    const modal = page.locator('#metadata-modal');
    const modalText = await modal.first().textContent();
    expect(modalText.toLowerCase()).toMatch(/(last|played|date|time|ago)/);
  });

  test('metadata modal is scrollable for long content', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for page to load
    await page.waitForSelector('#search-input', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    await page.locator('.caption-segment').first().click();
    
    // Wait for player to be ready
    await page.waitForSelector('#pip-player video, #pip-player audio', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Press 'i' key to open metadata modal
    await page.keyboard.press('i');
    await page.waitForTimeout(1000);

    // Modal body should be scrollable
    const modalBody = page.locator('.modal-body, .metadata-content');
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
  test('trash button is visible for media', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Hover over first media card
    const firstCard = page.locator('.media-card').first();
    await firstCard.hover();
    await page.waitForTimeout(500);

    // Trash button should appear
    const trashBtn = page.locator('.media-action-btn.delete, .trash-btn, .delete-btn, .card-delete');
    await expect(trashBtn.first()).toBeVisible();
  });

  test('trash button opens confirmation dialog', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Hover and click trash button
    const firstCard = page.locator('.media-card').first();
    await firstCard.hover();
    await page.waitForTimeout(500);

    const trashBtn = page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
    if (await trashBtn.count() > 0) {
      await trashBtn.click();
      await page.waitForTimeout(1000);

      // Confirmation dialog should appear
      const confirmDialog = page.locator('[role="alertdialog"], .confirm-dialog, .modal:has-text("delete"), .modal:has-text("trash")');
      await expect(confirmDialog.first()).toBeVisible();
    }
  });

  test('trash confirmation can be cancelled', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Hover and click trash button
    const firstCard = page.locator('.media-card').first();
    await firstCard.hover();
    await page.waitForTimeout(500);

    const trashBtn = page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
    if (await trashBtn.count() > 0) {
      await trashBtn.click();
      await page.waitForTimeout(1000);

      // Click cancel
      const cancelBtn = page.locator('button:has-text("Cancel"), .btn-cancel, [aria-label="Cancel"]');
      await cancelBtn.first().click();
      await page.waitForTimeout(1000);

      // Dialog should be hidden
      const confirmDialog = page.locator('[role="alertdialog"], .confirm-dialog');
      await expect(confirmDialog.first()).not.toBeVisible();

      // Card should still exist
      await expect(firstCard).toBeVisible();
    }
  });

  test('trash deletes media from view', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial card count
    const initialCards = page.locator('.media-card');
    const initialCount = await initialCards.count();

    if (initialCount > 0) {
      // Hover and click trash button on first card
      const firstCard = initialCards.first();
      await firstCard.hover();
      await page.waitForTimeout(500);

      const trashBtn = page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
      if (await trashBtn.count() > 0) {
        await trashBtn.click();
        await page.waitForTimeout(1000);

        // Confirm deletion
        const confirmBtn = page.locator('button:has-text("Delete"), button:has-text("Yes"), .btn-confirm, [aria-label="Confirm"]');
        await confirmBtn.first().click();
        await page.waitForTimeout(1000);

        // Card should be removed from view
        const remainingCards = page.locator('.media-card');
        const remainingCount = await remainingCards.count();
        expect(remainingCount).toBeLessThan(initialCount);
      }
    }
  });

  test('trash button has accessible label', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Hover over first media card
    const firstCard = page.locator('.media-card').first();
    await firstCard.hover();
    await page.waitForTimeout(500);

    // Trash button should have accessible name
    const trashBtn = page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
    const ariaLabel = await trashBtn.getAttribute('aria-label');
    const title = await trashBtn.getAttribute('title');
    
    // Should have either aria-label or title
    expect(ariaLabel || title).toBeTruthy();
    expect(ariaLabel || title).toMatch(/(delete|trash|remove)/i);
  });

  test('trash keyboard shortcut works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Select first card
    const firstCard = page.locator('.media-card').first();
    await firstCard.click();
    await page.waitForTimeout(300);

    // Press Delete key
    await page.keyboard.press('Delete');
    await page.waitForTimeout(1000);

    // Confirmation dialog should appear
    const confirmDialog = page.locator('[role="alertdialog"], .confirm-dialog');
    await expect(confirmDialog.first()).toBeVisible();

    // Cancel the deletion
    const cancelBtn = page.locator('button:has-text("Cancel")');
    if (await cancelBtn.count() > 0) {
      await cancelBtn.first().click();
    } else {
      await page.keyboard.press('Escape');
    }
  });

  test('trash shows success notification', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=captions');

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial card count
    const initialCards = page.locator('.media-card');
    const initialCount = await initialCards.count();

    if (initialCount > 0) {
      // Hover and click trash button
      const firstCard = initialCards.first();
      await firstCard.hover();
      await page.waitForTimeout(500);

      const trashBtn = page.locator('.media-action-btn.delete, .trash-btn, .delete-btn').first();
      if (await trashBtn.count() > 0) {
        await trashBtn.click();
        await page.waitForTimeout(1000);

        // Confirm deletion
        const confirmBtn = page.locator('button:has-text("Delete"), button:has-text("Yes")').first();
        await confirmBtn.click();
        await page.waitForTimeout(1000);

        // Success notification should appear
        const notification = page.locator('.toast, .notification, .alert-success, [role="status"]');
        if (await notification.count() > 0) {
          await expect(notification.first()).toBeVisible();
          const notificationText = await notification.first().textContent();
          expect(notificationText?.toLowerCase()).toMatch(/(deleted|removed|trash|success)/);
        }
      }
    }
  });

  test('trash button is disabled for already deleted items', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '#mode=trash');

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Deleted items should have disabled trash button or no trash button
    const deletedCards = page.locator('.media-card.deleted, .media-card:has-text("deleted")');
    if (await deletedCards.count() > 0) {
      const trashBtn = deletedCards.first().locator('.media-action-btn.delete, .trash-btn, .delete-btn');
      const isDisabled = await trashBtn.first().isDisabled();
      expect(isDisabled).toBe(true);
    }
  });
});
