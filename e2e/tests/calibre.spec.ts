import { test, expect } from '../fixtures';

/**
 * E2E tests for calibre-based EPUB conversion and viewing
 */
test.describe('Calibre EPUB Viewer', () => {
  test.use({ readOnly: true });

  test('opens EPUB in document modal with calibre conversion', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only EPUB documents using POM
    await mediaPage.search('.epub');

    // Find and clicking the test-book EPUB media card using POM
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    const epubCount = await epubCard.count();

    console.log(`Found ${epubCount} EPUB documents`);
    expect(epubCount).toBeGreaterThan(0);

    const epubPath = await epubCard.getAttribute('data-path');
    console.log(`Opening EPUB: ${epubPath}`);
    expect(epubPath).toContain('.epub');

    await epubCard.click();

    // Wait for document modal to open using POM
    await viewerPage.waitForDocumentModal();

    // Verify modal title matches the EPUB filename using POM
    const title = await viewerPage.getTitle();
    console.log(`Document title: ${title}`);
    expect(title).toBeTruthy();
    expect(title.toLowerCase()).toContain('test-book');

    // Check if iframe is present and has valid src using POM
    const iframe = viewerPage.getDocumentIframe();
    await expect(iframe).toBeVisible({ timeout: 10000 });

    const iframeSrc = await iframe.getAttribute('src');
    console.log(`Iframe src: ${iframeSrc}`);
    expect(iframeSrc).toBeTruthy();
    expect(iframeSrc).toContain('/api/epub/');

    // Wait for calibre conversion and content load
    await mediaPage.page.waitForTimeout(5000);

    // Verify no "File not found" or "Conversion failed" error in the modal using POM
    const containerText = await mediaPage.page.locator('#document-container').textContent();
    if (containerText) {
      expect(containerText.toLowerCase()).not.toContain('file not found');
      expect(containerText.toLowerCase()).not.toContain('404');
      expect(containerText.toLowerCase()).not.toContain('conversion failed');
    }

    // Verify fullscreen and RSVP buttons are present using POM
    await expect(viewerPage.documentFullscreenBtn).toBeVisible();
    await expect(viewerPage.page.locator('#doc-rsvp')).toBeVisible();

    // --- TOC AND CONTENT VERIFICATION ---

    // The main iframe contains the wrapper HTML which has its own iframe (#content-frame)
    const frame = mediaPage.page.frameLocator('#document-container iframe');

    // Check for the sticky TOC header in the wrapper HTML using POM
    const tocHeader = frame.locator('.toc-header');
    await expect(tocHeader).toBeVisible();

    const tocTitle = await tocHeader.locator('h1').textContent();
    console.log(`TOC Title: ${tocTitle}`);
    expect(tocTitle).toBeTruthy();

    // Check if TOC select element is present and has readable options using POM
    const tocSelect = frame.locator('.toc-nav select');
    await expect(tocSelect).toBeVisible();

    const options = await tocSelect.locator('option').all();
    console.log(`TOC options count: ${options.length}`);
    expect(options.length).toBeGreaterThan(1); // "Select chapter..." + actual chapters

    const firstChapterOption = options[1];
    const firstChapterText = await firstChapterOption.textContent();
    console.log(`First chapter in TOC: ${firstChapterText}`);
    expect(firstChapterText).toBeTruthy();

    // Verify content frame exists and loads book content using POM
    const contentFrame = frame.frameLocator('#content-frame');

    // Access content-frame via nested iframe check using POM
    const bodyText = await contentFrame.locator('body').textContent();
    console.log(`Inner frame body text length: ${bodyText?.length}`);
    expect(bodyText).toBeTruthy();
    if (bodyText) {
      expect(bodyText.length).toBeGreaterThan(100);

      // Check for chapter content from our test EPUB using POM
      expect(bodyText.toLowerCase()).toContain('chapter 1');
      expect(bodyText.toLowerCase()).toContain('introduction');
    }

    // Test TOC navigation using POM
    // Select the second chapter from TOC
    if (options.length > 2) {
      const secondChapterText = await options[2].textContent();
      console.log(`Navigating to: ${secondChapterText}`);

      await tocSelect.selectOption({ index: 2 });
      await mediaPage.page.waitForTimeout(1000);

      // Verify content changed to Chapter 2 using POM
      const newBodyText = await contentFrame.locator('body').textContent();
      if (newBodyText) {
        expect(newBodyText.toLowerCase()).toContain('chapter 2');
      }
    }

    // Close modal using POM
    await viewerPage.closeDocumentModal();
    await mediaPage.page.waitForTimeout(500);

    // Modal should be hidden using POM
    expect(await viewerPage.isDocumentModalHidden()).toBe(true);
  });

  test('original test files are browsable and viewable', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only EPUB documents using POM
    await mediaPage.search('.epub');

    // Find and clicking the test-book EPUB media card using POM
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    await epubCard.click();

    // Wait for document modal to open using POM
    await viewerPage.waitForDocumentModal();

    // Verify that we can see the content using POM
    const frame = mediaPage.page.frameLocator('#document-container iframe');
    const contentFrame = frame.frameLocator('#content-frame');

    await expect(contentFrame.locator('body')).toBeVisible({ timeout: 10000 });

    // Get some text content to verify it loaded using POM
    const bodyText = await contentFrame.locator('body').textContent();
    if (bodyText) {
      expect(bodyText.length).toBeGreaterThan(100);
    }

    // Close modal using POM
    await viewerPage.closeDocumentModal();
  });

  test('EPUB viewer has navigation controls', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only EPUB documents using POM
    await mediaPage.search('.epub');

    // Find and click the test-book EPUB using POM
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    await epubCard.click();

    // Wait for document modal using POM
    await viewerPage.waitForDocumentModal();

    // Navigation controls should be visible using POM
    const tocHeader = mediaPage.page.frameLocator('#document-container iframe').locator('.toc-header');
    await expect(tocHeader).toBeVisible();

    const tocSelect = mediaPage.page.frameLocator('#document-container iframe').locator('.toc-nav select');
    await expect(tocSelect).toBeVisible();

    // Close modal using POM
    await viewerPage.closeDocumentModal();
  });

  test('EPUB viewer handles errors gracefully', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Mock a 404 for EPUB conversion
    await mediaPage.page.route('**/api/epub/*', route => {
      route.fulfill({
        status: 404,
        body: 'Not Found'
      });
    });

    // Filter to show only EPUB documents using POM
    await mediaPage.search('.epub');

    // Find and click an EPUB using POM
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    if (await epubCard.count() > 0) {
      await epubCard.click();

      // Wait for document modal using POM
      await viewerPage.waitForDocumentModal();

      // Should show error state using POM
      const containerText = await mediaPage.page.locator('#document-container').textContent();
      if (containerText) {
        expect(containerText.toLowerCase()).toContain('file not found');
      }

      // Close modal using POM
      await viewerPage.closeDocumentModal();
    }
  });

  test('can search for EPUB files and open them', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search for test-book EPUB file using POM
    await mediaPage.search('test-book');

    // Should have at least one result using POM
    const epubCount = await mediaPage.getMediaCount();
    expect(epubCount).toBeGreaterThan(0);

    // Open first EPUB using POM
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').first();
    await epubCard.click();

    // Document modal should open using POM
    await viewerPage.waitForDocumentModal();

    // Verify it's an EPUB using POM
    const title = await viewerPage.getTitle();
    expect(title.toLowerCase()).toContain('epub');

    // Close modal using POM
    await viewerPage.closeDocumentModal();
  });

  test('EPUB viewer supports fullscreen', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only EPUB documents using POM
    await mediaPage.search('.epub');

    // Find and click an EPUB using POM
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    await epubCard.click();

    // Wait for document modal using POM
    await viewerPage.waitForDocumentModal();

    // Fullscreen button should be visible using POM
    await expect(viewerPage.documentFullscreenBtn).toBeVisible();

    // Click fullscreen using POM
    await viewerPage.documentFullscreenBtn.click();
    await mediaPage.page.waitForTimeout(500);

    // Fullscreen may or may not work in headless, but button should still exist
    await expect(viewerPage.documentFullscreenBtn).toBeVisible();

    // Close modal using POM
    await viewerPage.closeDocumentModal();
  });

  test('EPUB viewer has RSVP button', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Filter to show only EPUB documents using POM
    await mediaPage.search('.epub');

    // Find and click an EPUB using POM
    const epubCard = mediaPage.page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    await epubCard.click();

    // Wait for document modal using POM
    await viewerPage.waitForDocumentModal();

    // RSVP button should be visible using POM
    const rsvpBtn = mediaPage.page.locator('#doc-rsvp');
    await expect(rsvpBtn).toBeVisible();

    // Close modal using POM
    await viewerPage.closeDocumentModal();
  });
});
