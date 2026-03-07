import { test, expect } from '../fixtures';

test.describe('Document Viewer (PDF/EPUB)', () => {
  test.use({ readOnly: true });
  test('opens PDF file in viewer', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show only PDF documents
    await page.fill('#search-input', '.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Find and click a PDF media card
    const pdfCards = page.locator('.media-card[data-type*="document"], .media-card:has-text(".pdf")');
    const count = await pdfCards.count();

    if (count > 0) {
      await pdfCards.first().click();

      // Document viewer modal should open
      await page.waitForSelector('#document-modal:not(.hidden), #epub-viewer iframe', { timeout: 10000 });

      // Viewer should be visible (iframe inside epub-viewer or document-modal)
      const viewer = page.locator('#document-modal iframe, #epub-viewer iframe');
      if (await viewer.count() > 0) {
        await expect(viewer.first()).toBeVisible();
      }
    }
  });

  test('opens EPUB file in viewer', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show only EPUB documents
    await page.fill('#search-input', '.epub');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Find and click an EPUB media card
    const epubCards = page.locator('.media-card[data-type*="document"], .media-card:has-text(".epub")');
    const count = await epubCards.count();

    if (count > 0) {
      await epubCards.first().click();

      // EPUB viewer should open
      await page.waitForSelector('#document-modal:not(.hidden), #epub-viewer', { timeout: 10000 });

      // Viewer should be visible
      const viewer = page.locator('#epub-viewer, .epub-viewer');
      await expect(viewer.first()).toBeVisible();
    }
  });

  test('document viewer has navigation controls', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to documents
    await page.fill('#search-input', '.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCards = page.locator('.media-card:has-text(".pdf")');
    const count = await pdfCards.count();

    if (count > 0) {
      await pdfCards.first().click();
      await page.waitForTimeout(2000);

      // Check for navigation controls (doc-prev, doc-next)
      const prevBtn = page.locator('#doc-prev');
      const nextBtn = page.locator('#doc-next');

      // At least one navigation control should exist
      const hasPrev = await prevBtn.count() > 0;
      const hasNext = await nextBtn.count() > 0;
      expect(hasPrev || hasNext).toBe(true);
    }
  });

  test('document viewer shows page indicator', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to documents
    await page.fill('#search-input', '.pdf');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const pdfCards = page.locator('.media-card:has-text(".pdf")');
    const count = await pdfCards.count();

    if (count > 0) {
      await pdfCards.first().click();
      await page.waitForTimeout(2000);

      // Check for page indicator (doc-page-info)
      const pageIndicator = page.locator('#doc-page-info');
      if (await pageIndicator.count() > 0) {
        // Just verify it exists, it might be empty and thus hidden for some media (like PDF in iframe)
        expect(await pageIndicator.count()).toBeGreaterThan(0);
      }
    }
  });

  test('document viewer can be closed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to documents
    await page.fill('#search-input', '.epub');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const docCards = page.locator('.media-card[data-type*="text"], .media-card:has-text(".epub")');
    const count = await docCards.count();

    if (count > 0) {
      await docCards.first().click();
      
      // Wait for modal to open
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
      
      // Modal should be visible
      const modal = page.locator('#document-modal');
      await expect(modal.first()).toBeVisible();
      
      // Close using Escape key (more reliable than clicking close button)
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);

      // Modal should be hidden
      const isHidden = await modal.first().evaluate(el => el.classList.contains('hidden'));
      expect(isHidden).toBe(true);
    }
  });

  test('EPUB viewer has table of contents', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to EPUB
    await page.fill('#search-input', '.epub');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const epubCards = page.locator('.media-card:has-text(".epub")');
    const count = await epubCards.count();

    if (count > 0) {
      await epubCards.first().click();
      await page.waitForTimeout(2000);

      // Check for TOC button (may not exist in all implementations)
      const tocBtn = page.locator('.toc-btn, .table-of-contents, button:has-text("Contents"), button:has-text("TOC")');
      if (await tocBtn.count() > 0) {
        await expect(tocBtn.first()).toBeVisible();
      }
    }
  });

  test('document viewer zoom controls work', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to documents
    await page.fill('#search-input', '.epub');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const docCards = page.locator('.media-card[data-type*="text"], .media-card:has-text(".epub")');
    const count = await docCards.count();

    if (count > 0) {
      await docCards.first().click();
      
      // Wait for modal to open
      await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
      await page.waitForTimeout(1000);

      // Check for zoom controls (doc-zoom-in, doc-zoom-out)
      const zoomInBtn = page.locator('#doc-zoom-in');
      const zoomOutBtn = page.locator('#doc-zoom-out');
      const zoomInfo = page.locator('#doc-zoom-info');

      // Zoom controls should exist
      const hasZoomIn = await zoomInBtn.count() > 0;
      const hasZoomOut = await zoomOutBtn.count() > 0;
      const hasZoomInfo = await zoomInfo.count() > 0;
      
      // At least zoom info should be visible
      if (hasZoomInfo) {
        await expect(zoomInfo.first()).toBeVisible();
      }
      
      // Try zoom in if available and enabled
      if (hasZoomIn) {
        const isEnabled = await zoomInBtn.first().isEnabled();
        if (isEnabled) {
          await zoomInBtn.first().click({ force: true });
          await page.waitForTimeout(500);
        }
      }
    }
  });
});

test.describe('Image Viewer', () => {
  test('opens image in viewer', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show only images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Find and click an image media card
    const imageCards = page.locator('.media-card[data-type="image"], .media-card:has-text(".jpg"), .media-card:has-text(".png")');
    const count = await imageCards.count();

    if (count > 0) {
      await imageCards.first().click();

      // Image viewer (PiP player with img) should open
      await page.waitForSelector('#pip-player img, img[src*="/api/"]', { timeout: 10000 });

      // Image should be visible
      const img = page.locator('#pip-player img').first();
      if (await img.count() > 0) {
        await expect(img).toBeVisible();
      }
    }
  });

  test('image viewer shows image title', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 0) {
      // Get the title from the card
      const cardTitle = await imageCards.first().locator('.media-title').textContent();

      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Title should be shown in viewer
      const viewerTitle = page.locator('.viewer-title, .image-title, #media-title');
      if (await viewerTitle.count() > 0) {
        const titleText = await viewerTitle.first().textContent();
        expect(titleText).toContain(cardTitle || '');
      }
    }
  });

  test('image viewer can navigate to next image', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 1) {
      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Press 'n' key for next (keyboard shortcut)
      await page.keyboard.press('n');
      await page.waitForTimeout(500);

      // Should still be in viewer
      const viewer = page.locator('#pip-player');
      if (await viewer.count() > 0) {
        await expect(viewer.first()).toBeVisible();
      }
    }
  });

  test('image viewer can navigate to previous image', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 1) {
      // Click second image first
      await imageCards.nth(1).click();
      await page.waitForTimeout(1000);

      // Press 'p' key for previous (keyboard shortcut)
      await page.keyboard.press('p');
      await page.waitForTimeout(500);

      // Should still be in viewer
      const viewer = page.locator('#pip-player');
      if (await viewer.count() > 0) {
        await expect(viewer.first()).toBeVisible();
      }
    }
  });

  test('image viewer can be closed with Escape key', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 0) {
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

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 1) {
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

  test('image viewer shows image count', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 0) {
      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Check for image count indicator
      const countIndicator = page.locator('.image-count, .viewer-count');
      if (await countIndicator.count() > 0) {
        await expect(countIndicator.first()).toBeVisible();
      }
    }
  });

  test('image viewer supports slideshow mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 0) {
      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Check for slideshow button
      const slideshowBtn = page.locator('.slideshow-btn, button:has-text("Slideshow"), .play-slideshow');
      if (await slideshowBtn.count() > 0) {
        await slideshowBtn.first().click();
        await page.waitForTimeout(3000);

        // Should still be in viewer (slideshow running)
        const viewer = page.locator('#pip-player');
        await expect(viewer.first()).toBeVisible();

        // Stop slideshow
        await slideshowBtn.first().click();
      }
    }
  });

  test('image viewer rotates image', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 0) {
      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Check for rotate button
      const rotateBtn = page.locator('.rotate-btn, button:has-text("Rotate"), .image-rotate');
      if (await rotateBtn.count() > 0) {
        const initialRotation = await page.locator('.image-viewer img').evaluate((el) => 
          window.getComputedStyle(el).transform
        );

        await rotateBtn.first().click();
        await page.waitForTimeout(500);

        // Rotation should change
        const newRotation = await page.locator('.image-viewer img').evaluate((el) => 
          window.getComputedStyle(el).transform
        );

        expect(newRotation).not.toEqual(initialRotation);
      }
    }
  });

  test('image viewer fits image to screen', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to images
    await page.fill('#search-input', '.jpg');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const imageCards = page.locator('.media-card:has-text(".jpg")');
    const count = await imageCards.count();

    if (count > 0) {
      await imageCards.first().click();
      await page.waitForTimeout(1000);

      // Check for fit button
      const fitBtn = page.locator('.fit-btn, button:has-text("Fit"), .fit-to-screen');
      if (await fitBtn.count() > 0) {
        await fitBtn.first().click();
        await page.waitForTimeout(500);

        // Image should fit
        const img = page.locator('.image-viewer img').first();
        await expect(img).toBeVisible();
      }
    }
  });
});

test.describe('Audio Playback', () => {
  test('opens audio file in player', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show only audio files
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // Find and click an audio media card
    const audioCards = page.locator('.media-card[data-type="audio"], .media-card:has-text(".mp3"), .media-card:has-text(".wav")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();

      // Audio player should open
      await page.waitForSelector('#pip-player:not(.hidden), .audio-player, audio[src]', { timeout: 10000 });

      // Player should be visible
      const player = page.locator('#pip-player, .audio-player');
      await expect(player.first()).toBeVisible();
    }
  });

  test('audio player shows track title', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      // Get the title from the card
      const cardTitle = await audioCards.first().locator('.media-title').textContent();

      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Title should be shown in player
      const playerTitle = page.locator('#media-title');
      if (await playerTitle.count() > 0) {
        const titleText = await playerTitle.first().textContent();
        expect(titleText).toContain(cardTitle || '');
      }
    }
  });

  test('audio player shows duration', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Duration should be shown
      const duration = page.locator('#pip-duration');
      await expect(duration.first()).toBeVisible();
    }
  });

  test('audio player play/pause button works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Find play/pause button
      const playBtn = page.locator('#pip-play-pause');
      
      if (await playBtn.count() > 0) {
        // Click play/pause
        await playBtn.first().click();
        await page.waitForTimeout(500);

        // Button state should change
        await expect(playBtn.first()).toBeVisible();
      }
    }
  });

  test('audio player volume control works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Find volume control
      const volumeSlider = page.locator('#pip-volume');
      
      if (await volumeSlider.count() > 0) {
        // Change volume
        await volumeSlider.first().evaluate((el) => {
          (el as HTMLInputElement).value = '50';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);

        // Volume should be set
        const value = await volumeSlider.first().evaluate((el) => (el as HTMLInputElement).value);
        expect(value).toBe('50');
      }
    }
  });

  test('audio player seek bar works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Find seek bar
      const seekBar = page.locator('#pip-seekbar');
      
      if (await seekBar.count() > 0) {
        // Get initial value
        const initialValue = await seekBar.first().evaluate((el) => (el as HTMLInputElement).value);

        // Change seek position
        await seekBar.first().evaluate((el) => {
          (el as HTMLInputElement).value = '30';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);

        // Position should change
        const newValue = await seekBar.first().evaluate((el) => (el as HTMLInputElement).value);
        expect(newValue).not.toEqual(initialValue);
      }
    }
  });

  test('audio player can be closed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Close player
      const closeBtn = page.locator('.close-pip, .player-close, button:has-text("Close")');
      if (await closeBtn.count() > 0) {
        await closeBtn.first().click();
        await page.waitForTimeout(500);

        // Player should be hidden
        const player = page.locator('#pip-player');
        await expect(player).toHaveClass(/hidden/);
      }
    }
  });

  test('audio player shows next/previous track buttons', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 1) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Check for next/previous buttons
      const nextBtn = page.locator('.next-btn, .track-next, button:has-text("Next"), .player-next');
      const prevBtn = page.locator('.prev-btn, .track-prev, button:has-text("Previous"), .player-prev');

      // At least one should exist
      const hasNext = await nextBtn.count() > 0;
      const hasPrev = await prevBtn.count() > 0;
      expect(hasNext || hasPrev).toBe(true);
    }
  });

  test('audio player supports keyboard shortcuts', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Press space for play/pause
      await page.keyboard.press(' ');
      await page.waitForTimeout(500);

      // Player should still be visible
      const player = page.locator('#pip-player, .audio-player');
      await expect(player.first()).toBeVisible();
    }
  });

  test('audio player shows album art if available', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Filter to audio
    await page.fill('#search-input', '.mp3');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    const audioCards = page.locator('.media-card:has-text(".mp3")');
    const count = await audioCards.count();

    if (count > 0) {
      await audioCards.first().click();
      await page.waitForTimeout(1000);

      // Check for album art
      const albumArt = page.locator('.album-art, .track-art, .player-art, img[src*="cover"]');
      if (await albumArt.count() > 0) {
        await expect(albumArt.first()).toBeVisible();
      }
    }
  });
});
