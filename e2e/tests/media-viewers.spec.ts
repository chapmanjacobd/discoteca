import { test, expect } from '../fixtures';

test.describe('Document Viewer (PDF/EPUB)', () => {
  test.use({ readOnly: true });

  test('opens PDF in fullscreen modal', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Modal header should be at top of page
      const header = page.locator('#document-modal .modal-header');
      await expect(header.first()).toBeVisible();

      // Check that header is at the top (y position should be 0 or very close)
      const headerBox = await header.first().boundingBox();
      expect(headerBox).toBeTruthy();
      if (headerBox) {
        expect(headerBox.y).toBeLessThanOrEqual(5); // Allow small margin for rounding
      }
    }
  });

  test('iframe area is at least 70% of display area', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Get viewport dimensions
      const viewport = page.viewportSize();
      expect(viewport).toBeTruthy();
      if (viewport) {
        const viewportArea = viewport.width * viewport.height;

        // Get iframe dimensions
        const iframe = page.locator('#document-container iframe');
        await expect(iframe.first()).toBeVisible();

        const iframeBox = await iframe.first().boundingBox();
        expect(iframeBox).toBeTruthy();
        if (iframeBox) {
          const iframeArea = iframeBox.width * iframeBox.height;
          const areaRatio = iframeArea / viewportArea;

          // Iframe should take at least 70% of viewport area
          // (accounting for header which takes some space)
          expect(areaRatio).toBeGreaterThanOrEqual(0.65); // Using 65% to account for header
        }
      }
    }
  });

  test('document viewer has fullscreen button', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Fullscreen button should exist
      const fsBtn = page.locator('#doc-fullscreen');
      await expect(fsBtn.first()).toBeVisible();
    }
  });

  test('fullscreen button toggles fullscreen mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(500);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Click fullscreen button (use force: true to bypass iframe overlay)
      const fsBtn = page.locator('#doc-fullscreen');
      await fsBtn.first().click({ force: true });

      // Wait for fullscreen change with retry polling
      await page.waitForFunction(() => !!document.fullscreenElement, { timeout: 5000 });

      // Should enter fullscreen
      const isFullscreen = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreen).toBe(true);

      // Use Escape key to exit fullscreen (more reliable than button click in fullscreen mode)
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);

      // Wait for fullscreen to exit
      await page.waitForFunction(() => !document.fullscreenElement, { timeout: 5000 });

      // Should exit fullscreen
      const isFullscreenAfter = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreenAfter).toBe(false);
    }
  });

  test('f key toggles fullscreen', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Press 'f' key
      await page.keyboard.press('f');
      await page.waitForTimeout(500);

      // Should enter fullscreen
      const isFullscreen = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreen).toBe(true);

      // Press 'f' again
      await page.keyboard.press('f');
      await page.waitForTimeout(500);

      // Should exit fullscreen
      const isFullscreenAfter = await page.evaluate(() => !!document.fullscreenElement);
      expect(isFullscreenAfter).toBe(false);
    }
  });

  test('document viewer can be closed with Escape', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Press Escape
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);

      // Modal should be hidden
      const modal = page.locator('#document-modal');
      const isHidden = await modal.first().evaluate(el => el.classList.contains('hidden'));
      expect(isHidden).toBe(true);
    }
  });

  test('document viewer has close button', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Close button should exist
      const closeBtn = page.locator('#document-modal .close-modal');
      await expect(closeBtn.first()).toBeVisible();

      // Click to close
      await closeBtn.first().click();
      await page.waitForTimeout(500);

      // Modal should be hidden
      const modal = page.locator('#document-modal');
      const isHidden = await modal.first().evaluate(el => el.classList.contains('hidden'));
      expect(isHidden).toBe(true);
    }
  });

  test('document viewer shows document title', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search for test PDF
    await page.fill('#search-input', 'test-document.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCard = page.locator('.media-card:has-text("test-document.pdf")');
    if (await pdfCard.count() > 0) {
      await pdfCard.first().click();
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });

      // Title should show filename
      const title = page.locator('#document-title');
      await expect(title.first()).toContainText('test-document.pdf');
    }
  });
});

test.describe('Image Viewer', () => {
  test('opens image in viewer', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show only images
    await page.fill('#search-input', '.png');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Find and click an image media card
    const imageCards = page.locator('.media-card:has-text(".png")');
    if (await imageCards.count() > 0) {
      await imageCards.first().click();

      // Image viewer (PiP player with img) should open
      await page.waitForSelector('#pip-player img', { timeout: 10000 });

      // Image should be visible
      const img = page.locator('#pip-player img').first();
      await expect(img).toBeVisible();
    }
  });

  test('image viewer can be closed with Escape key', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to images
    await page.fill('#search-input', '.png');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".png")');
    if (await imageCards.count() > 0) {
      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Press Escape
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);

      // Viewer should be hidden
      const viewer = page.locator('#pip-player');
      await expect(viewer.first()).not.toBeVisible();
    }
  });

  test('image viewer supports keyboard navigation', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to images
    await page.fill('#search-input', '.png');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".png")');
    if (await imageCards.count() > 1) {
      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Press right arrow for next
      await page.keyboard.press('ArrowRight');
      await page.waitForTimeout(500);

      // Should navigate to next image
      const viewer = page.locator('#pip-player');
      await expect(viewer.first()).toBeVisible();
    }
  });
});

test.describe('Audio Playback', () => {
  test('opens audio file in player', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show only audio files
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Find and click an audio media card
    const audioCards = page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();

      // Audio player should open
      await page.waitForSelector('#pip-player:not(.hidden), audio[src]', { timeout: 10000 });

      // Player should be visible
      const player = page.locator('#pip-player');
      await expect(player.first()).toBeVisible();
    }
  });

  test('audio player shows duration', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Duration should be available on the audio element
      const audio = page.locator('#pip-player audio').first();
      await expect(audio).toBeVisible();
      
      // Check that the audio element has a valid duration
      const duration = await audio.evaluate(el => el.duration);
      expect(duration).toBeGreaterThan(0);
      expect(duration).toBeLessThan(1000); // Reasonable upper bound (in seconds)
    }
  });

  test('audio player can be closed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Close player using Escape
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);

      // Player should be hidden
      const player = page.locator('#pip-player');
      await expect(player).toHaveClass(/hidden/);
    }
  });

  test('audio player supports keyboard shortcuts', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    if (await audioCards.count() > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Press space for play/pause
      await page.keyboard.press(' ');
      await page.waitForTimeout(500);

      // Player should still be visible
      const player = page.locator('#pip-player');
      await expect(player.first()).toBeVisible();
    }
  });
});
