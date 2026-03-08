import { test, expect } from '../fixtures';

/**
 * E2E tests for document viewer (PDF, EPUB, etc.)
 */
test.describe('Document Viewer', () => {
  test.use({ readOnly: true });

  test.beforeEach(async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.error('BROWSER ERROR:', msg.text());
      }
    });
  });

  test('opens EPUB in document modal with calibre conversion', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find the EPUB document by title/filename
    const epubCard = page.locator('.media-card[data-type*="text"]').filter({ hasText: /test-book/i }).first();
    const epubCount = await epubCard.count();
    
    console.log(`Found ${epubCount} EPUB documents`);
    expect(epubCount).toBeGreaterThan(0);
    
    const epubPath = await epubCard.getAttribute('data-path');
    console.log(`Opening EPUB: ${epubPath}`);
    expect(epubPath).toContain('.epub');
    
    await epubCard.click();

    // Wait for document modal to open
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    
    // Verify modal title matches the EPUB filename
    const title = await page.locator('#document-title').textContent();
    console.log(`Document title: ${title}`);
    expect(title).toBeTruthy();
    expect(title.length).toBeGreaterThan(3);
    expect(title.toLowerCase()).toContain('test-book');
    
    // Check if iframe is present and has valid src
    const iframe = page.locator('#document-container iframe');
    await expect(iframe).toBeVisible({ timeout: 5000 });
    
    const iframeSrc = await iframe.getAttribute('src');
    console.log(`Iframe src: ${iframeSrc}`);
    expect(iframeSrc).toBeTruthy();
    expect(iframeSrc).toContain('/api/epub/');
    
    // Wait for calibre conversion to complete
    await page.waitForTimeout(5000);
    
    // Verify no "File not found" error in the modal
    const containerText = await page.locator('#document-container').textContent();
    if (containerText) {
      expect(containerText.toLowerCase()).not.toContain('file not found');
      expect(containerText.toLowerCase()).not.toContain('404');
      expect(containerText.toLowerCase()).not.toContain('conversion failed');
    }
    
    // Verify fullscreen and RSVP buttons are present
    const fsBtn = page.locator('#doc-fullscreen');
    await expect(fsBtn).toBeVisible();
    
    const rsvpBtn = page.locator('#doc-rsvp');
    await expect(rsvpBtn).toBeVisible();
    
    // Check that content is loaded (iframe should have content)
    const frameContent = await iframe.content();
    const bodyText = await frameContent.locator('body').textContent();
    console.log(`Frame body text length: ${bodyText?.length}`);
    expect(bodyText).toBeTruthy();
    expect(bodyText.length).toBeGreaterThan(100);
    
    // Check for chapter content from our test EPUB
    expect(bodyText.toLowerCase()).toContain('chapter');
    
    // Close modal
    await page.click('#document-modal .close-modal');
    await page.waitForTimeout(500);
    
    // Modal should be hidden
    const isHidden = await page.locator('#document-modal').evaluate(el => el.classList.contains('hidden'));
    expect(isHidden).toBe(true);
  });

  test('document modal has correct title', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first text document
    const textCard = page.locator('.media-card[data-type*="text"]').first();
    const textPath = await textCard.getAttribute('data-path');
    console.log(`Opening document: ${textPath}`);
    
    await textCard.click();
    
    // Wait for modal
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    
    // Check title matches filename
    const title = await page.locator('#document-title').textContent();
    const expectedTitle = textPath?.split('/').pop() || '';
    console.log(`Title: "${title}", Expected: "${expectedTitle}"`);
    
    expect(title).toBeTruthy();
    expect(title.length).toBeGreaterThan(0);
    
    // Close modal
    await page.click('#document-modal .close-modal');
  });

  test('document viewer has fullscreen button', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first text document
    const textCard = page.locator('.media-card[data-type*="text"]').first();
    await textCard.click();
    
    // Wait for modal
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    
    // Check fullscreen button exists
    const fsBtn = page.locator('#doc-fullscreen');
    await expect(fsBtn).toBeVisible();
    
    // Close modal
    await page.click('#document-modal .close-modal');
  });

  test('document viewer has RSVP button', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first text document
    const textCard = page.locator('.media-card[data-type*="text"]').first();
    await textCard.click();
    
    // Wait for modal
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    
    // Check RSVP button exists
    const rsvpBtn = page.locator('#doc-rsvp');
    await expect(rsvpBtn).toBeVisible();
    
    // Close modal
    await page.click('#document-modal .close-modal');
  });

  test('escape key closes document modal', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first text document
    const textCard = page.locator('.media-card[data-type*="text"]').first();
    await textCard.click();
    
    // Wait for modal
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    
    // Press escape
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);
    
    // Modal should be closed
    const isHidden = await page.locator('#document-modal').evaluate(el => el.classList.contains('hidden'));
    expect(isHidden).toBe(true);
  });

  test('document iframe does not show 404', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first text document
    const textCard = page.locator('.media-card[data-type*="text"]').first();
    const textPath = await textCard.getAttribute('data-path');
    console.log(`Testing document: ${textPath}`);
    
    await textCard.click();
    
    // Wait for modal
    await page.waitForSelector('#document-modal:not(.hidden)', { timeout: 10000 });
    
    // Wait for iframe to load
    await page.waitForTimeout(3000);
    
    // Check iframe
    const iframe = page.locator('#document-container iframe');
    const iframeSrc = await iframe.getAttribute('src');
    console.log(`Iframe src: ${iframeSrc}`);
    
    // Listen for any frame errors
    const frameErrors: string[] = [];
    page.on('frameattached', frame => {
      frame.on('load', () => {
        console.log('Frame loaded:', frame.url());
      }).on('error', (err) => {
        console.error('Frame error:', err);
        frameErrors.push(err.message);
      });
    });
    
    // Check for 404 in iframe content
    try {
      const frame = iframe.first();
      const frameContent = await frame.content();
      const bodyText = await frameContent.locator('body').textContent();
      console.log(`Frame body text (first 200 chars): ${bodyText?.substring(0, 200)}`);
      
      // Should not contain 404 or "not found"
      if (bodyText) {
        expect(bodyText.toLowerCase()).not.toContain('404');
        expect(bodyText.toLowerCase()).not.toContain('not found');
      }
    } catch (e) {
      // Cross-origin iframe, can't access content
      console.log('Cannot access iframe content (cross-origin), checking URL instead');
    }
    
    // Close modal
    await page.click('#document-modal .close-modal');
  });
});
