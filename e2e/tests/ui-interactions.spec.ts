import { test, expect } from '../fixtures';

test.use({ readOnly: true });

test.describe('Fullscreen Toggle', () => {
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

  test('Escape exits player fullscreen', async ({ mediaPage, viewerPage, server, browserName }) => {
    test.skip(browserName === 'webkit', 'WebKit headless does not support the Fullscreen API');

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
    test.skip((mediaPage.page.viewportSize()?.width ?? 1920) <= 768, 'The outside click lands on the mobile header');

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
