import { test, expect } from '../fixtures';

test.describe('Document Viewer (PDF/EPUB)', () => {
  test.use({ readOnly: true });

  test('opens PDF in fullscreen modal', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Modal header should be at top of page
      const header = viewerPage.documentModal.locator('.modal-header');
      await expect(header.first()).toBeVisible();

      // Check that header is at the top (y position should be 0 or very close)
      const headerBox = await header.first().boundingBox();
      expect(headerBox).toBeTruthy();
      if (headerBox) {
        expect(headerBox.y).toBeLessThanOrEqual(5);
      }
    }
  });

  test('iframe area is at least 70% of display area', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Get viewport dimensions using POM
      const viewport = await mediaPage.getViewportSize();
      expect(viewport).toBeTruthy();
      if (viewport) {
        const viewportArea = viewport.width * viewport.height;

        // Get iframe dimensions using POM
        const iframe = viewerPage.getDocumentIframe();
        await expect(iframe.first()).toBeVisible();

        const iframeBox = await iframe.first().boundingBox();
        expect(iframeBox).toBeTruthy();
        if (iframeBox) {
          const iframeArea = iframeBox.width * iframeBox.height;
          const areaRatio = iframeArea / viewportArea;

          // Iframe should take at least 70% of viewport area
          expect(areaRatio).toBeGreaterThanOrEqual(0.65);
        }
      }
    }
  });

  test('document viewer has fullscreen button', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Fullscreen button should exist using POM
      await expect(viewerPage.documentFullscreenBtn.first()).toBeVisible();
    }
  });

  test('fullscreen button toggles fullscreen mode', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Click fullscreen button using POM
      await viewerPage.documentFullscreenBtn.first().click({ force: true });

      // Wait for fullscreen change using POM
      await viewerPage.waitForFullscreenChange();

      // Should enter fullscreen
      expect(await viewerPage.isFullscreenActive()).toBe(true);

      // Use Escape key to exit fullscreen
      await mediaPage.page.keyboard.press('Escape');
      await mediaPage.page.waitForTimeout(300);

      // Wait for fullscreen to exit using POM
      await viewerPage.waitForFullscreenExit();

      // Should exit fullscreen
      expect(await viewerPage.isFullscreenActive()).toBe(false);
    }
  });

  test('f key toggles fullscreen', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Press 'f' key
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(500);

      // Should enter fullscreen using POM
      expect(await viewerPage.isFullscreenActive()).toBe(true);

      // Press 'f' again
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(500);

      // Should exit fullscreen using POM
      expect(await viewerPage.isFullscreenActive()).toBe(false);
    }
  });

  test('document viewer can be closed with Escape', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Press Escape
      await mediaPage.page.keyboard.press('Escape');
      await mediaPage.page.waitForTimeout(500);

      // Modal should be hidden using POM
      expect(await viewerPage.isDocumentModalHidden()).toBe(true);
    }
  });

  test('document viewer has close button', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Close button should exist using POM
      const closeBtn = viewerPage.documentModal.locator('.close-modal');
      await expect(closeBtn.first()).toBeVisible();

      // Click to close using POM
      await closeBtn.first().click();
      await mediaPage.page.waitForTimeout(500);

      // Modal should be hidden using POM
      expect(await viewerPage.isDocumentModalHidden()).toBe(true);
    }
  });

  test('document viewer shows document title', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test PDF using POM
    await mediaPage.search('test-document.pdf');

    const pdfCard = mediaPage.getMediaCardByPath('test-document.pdf');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await viewerPage.waitForDocumentModal();

      // Title should show filename using POM
      await expect(viewerPage.documentTitle.first()).toContainText('test-document.pdf');
    }
  });
});

test.describe('Image Viewer', () => {
  test('opens image in viewer', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only images using POM
    await mediaPage.search('.png');

    // Find and clicking an image media card using POM
    const imageCards = mediaPage.page.locator('.media-card:has-text(".png")');
    if (await imageCards.count() > 0) {
      await imageCards.first().click();

      // Image viewer (PiP player with img) should open using POM
      await viewerPage.waitForImageLoad();

      // Image should be visible using POM
      await expect(viewerPage.getImageElement()).toBeVisible();
    }
  });

  test('image viewer can be closed with Escape key', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to images using POM
    await mediaPage.search('.png');

    const imageCards = mediaPage.page.locator('.media-card:has-text(".png")');
    if (await imageCards.count() > 0) {
      await imageCards.first().click();
      await mediaPage.page.waitForTimeout(1000);

      // Press Escape
      await mediaPage.page.keyboard.press('Escape');
      await mediaPage.page.waitForTimeout(500);

      // Viewer should be hidden using POM
      await expect(viewerPage.playerContainer.first()).not.toBeVisible();
    }
  });

  test('image viewer supports keyboard navigation', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to images using POM
    await mediaPage.search('.png');

    const imageCards = mediaPage.page.locator('.media-card:has-text(".png")');
    if (await imageCards.count() > 1) {
      await imageCards.first().click();
      await mediaPage.page.waitForTimeout(1000);

      // Press right arrow for next
      await mediaPage.page.keyboard.press('ArrowRight');
      await mediaPage.page.waitForTimeout(500);

      // Should navigate to next image - viewer should still be visible using POM
      await expect(viewerPage.playerContainer.first()).toBeVisible();
    }
  });
});

test.describe('Audio Playback', () => {
  test('opens audio file in player', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only audio files using POM
    await mediaPage.search('.mp3');

    // Find and clicking an audio media card using POM
    const audioCards = mediaPage.page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();

      // Audio player should open using POM
      await viewerPage.playerContainer.first().waitFor({ state: 'visible', timeout: 10000 });

      // Player should be visible using POM
      await expect(viewerPage.playerContainer.first()).toBeVisible();
    }
  });

  test('audio player shows duration', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to audio using POM
    await mediaPage.search('.mp3');

    const audioCards = mediaPage.page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();
      await mediaPage.page.waitForTimeout(1000);

      // Duration should be available on the audio element using POM
      const audio = viewerPage.audioElement.first();
      await expect(audio).toBeVisible();

      // Check that the audio element has a valid duration using POM
      const duration = await viewerPage.getDuration();
      expect(duration).toBeGreaterThan(0);
      expect(duration).toBeLessThan(1000);
    }
  });

  test('audio player can be closed', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to audio using POM
    await mediaPage.search('.mp3');

    const audioCards = mediaPage.page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();
      await mediaPage.page.waitForTimeout(1000);

      // Close player using Escape using POM
      await mediaPage.page.keyboard.press('Escape');
      await mediaPage.page.waitForTimeout(500);

      // Player should be hidden using POM
      await expect(viewerPage.playerContainer).toHaveClass(/hidden/);
    }
  });

  test('audio player supports keyboard shortcuts', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to audio using POM
    await mediaPage.search('.mp3');

    const audioCards = mediaPage.page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();
      await mediaPage.page.waitForTimeout(1000);

      // Press space for play/pause using POM
      await mediaPage.page.keyboard.press(' ');
      await mediaPage.page.waitForTimeout(500);

      // Player should still be visible using POM
      await expect(viewerPage.playerContainer.first()).toBeVisible();
    }
  });
});
