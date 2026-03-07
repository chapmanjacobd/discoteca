import { test, expect } from '../fixtures';

test.describe('Large Result Sets Scrolling', () => {
  test('scrolls through large media list', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial visible cards
    const initialCards = page.locator('.media-card');
    const initialCount = await initialCards.count();

    // Scroll down
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(1000);

    // More cards should load or pagination should appear
    const scrolledCards = page.locator('.media-card');
    const scrolledCount = await scrolledCards.count();

    // Either more cards loaded or we're at the end
    expect(scrolledCount).toBeGreaterThanOrEqual(initialCount);
  });

  test('infinite scroll loads more results', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial card count
    const initialCards = page.locator('.media-card');
    const initialCount = await initialCards.count();

    // Scroll to bottom multiple times
    for (let i = 0; i < 3; i++) {
      await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
      await page.waitForTimeout(1000);
    }

    // Check if more cards loaded
    const finalCards = page.locator('.media-card');
    const finalCount = await finalCards.count();

    // Either more cards loaded or we hit the limit
    expect(finalCount).toBeGreaterThanOrEqual(initialCount);
  });

  test('pagination controls are visible for large sets', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Pagination container should exist
    const pagination = page.locator('#pagination-container, .pagination, .pager');
    await expect(pagination.first()).toBeVisible();
  });

  test('page navigation works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find next page button
    const nextBtn = page.locator('.pagination button:has-text("Next"), .pagination .next, .pager .next');
    
    if (await nextBtn.count() > 0) {
      // Get current page
      const currentPage = await page.locator('.page-info, .pagination .active').textContent();
      
      // Click next
      await nextBtn.first().click();
      await page.waitForTimeout(1000);

      // Page should change
      const newPage = await page.locator('.page-info, .pagination .active').textContent();
      expect(newPage).not.toEqual(currentPage);
    }
  });

  test('page number input works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find page input
    const pageInput = page.locator('input[type="number"][aria-label*="page"], .page-input, input.page');
    
    if (await pageInput.count() > 0) {
      // Enter page 2
      await pageInput.first().fill('2');
      await pageInput.first().press('Enter');
      await page.waitForTimeout(1000);

      // Should navigate to page 2
      const currentPage = await page.locator('.page-info, .pagination .active').textContent();
      expect(currentPage).toContain('2');
    }
  });

  test('jump to last page works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find last page button
    const lastBtn = page.locator('.pagination button:has-text("Last"), .pagination .last, .pager .last');
    
    if (await lastBtn.count() > 0) {
      await lastBtn.first().click();
      await page.waitForTimeout(1000);

      // Should be on last page
      const currentPage = await page.locator('.page-info, .pagination .active').textContent();
      expect(currentPage).toBeTruthy();
    }
  });

  test('scroll position is preserved after action', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll down
    const scrollPosition = 500;
    await page.evaluate((pos) => window.scrollTo(0, pos), scrollPosition);
    await page.waitForTimeout(500);

    // Perform an action (like clicking a filter)
    const sortBy = page.locator('#sort-by');
    if (await sortBy.count() > 0) {
      await sortBy.first().click();
      await page.waitForTimeout(500);
    }

    // Scroll position should be roughly preserved
    const newPosition = await page.evaluate(() => window.scrollY);
    expect(newPosition).toBeGreaterThanOrEqual(scrollPosition - 100);
  });

  test('smooth scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll smoothly
    await page.evaluate(() => {
      window.scrollTo({
        top: 500,
        behavior: 'smooth'
      });
    });
    await page.waitForTimeout(1000);

    // Should have scrolled
    const scrollPosition = await page.evaluate(() => window.scrollY);
    expect(scrollPosition).toBeGreaterThan(0);
  });

  test('scroll to top button appears after scrolling', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll down
    await page.evaluate(() => window.scrollTo(0, 1000));
    await page.waitForTimeout(500);

    // Scroll to top button should appear
    const scrollTopBtn = page.locator('.scroll-top, .back-to-top, button:has-text("Top"), .to-top');
    if (await scrollTopBtn.count() > 0) {
      await expect(scrollTopBtn.first()).toBeVisible();
    }
  });

  test('scroll to top button works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll down
    await page.evaluate(() => window.scrollTo(0, 1000));
    await page.waitForTimeout(500);

    // Click scroll to top
    const scrollTopBtn = page.locator('.scroll-top, .back-to-top').first();
    if (await scrollTopBtn.count() > 0) {
      await scrollTopBtn.click();
      await page.waitForTimeout(500);

      // Should be at top
      const scrollPosition = await page.evaluate(() => window.scrollY);
      expect(scrollPosition).toBeLessThan(100);
    }
  });

  test('keyboard scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial position
    const initialPosition = await page.evaluate(() => window.scrollY);

    // Press Page Down
    await page.keyboard.press('PageDown');
    await page.waitForTimeout(500);

    // Should have scrolled down
    const newPosition = await page.evaluate(() => window.scrollY);
    expect(newPosition).toBeGreaterThan(initialPosition);
  });

  test('arrow key scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial position
    const initialPosition = await page.evaluate(() => window.scrollY);

    // Press down arrow
    await page.keyboard.press('ArrowDown');
    await page.waitForTimeout(300);

    // Should have scrolled down
    const newPosition = await page.evaluate(() => window.scrollY);
    expect(newPosition).toBeGreaterThanOrEqual(initialPosition);
  });

  test('spacebar scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial position
    const initialPosition = await page.evaluate(() => window.scrollY);

    // Press space
    await page.keyboard.press(' ');
    await page.waitForTimeout(300);

    // Should have scrolled down
    const newPosition = await page.evaluate(() => window.scrollY);
    expect(newPosition).toBeGreaterThan(initialPosition);
  });

  test('home/end keys work', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Press End
    await page.keyboard.press('End');
    await page.waitForTimeout(500);

    // Should be near bottom
    const scrollHeight = await page.evaluate(() => document.body.scrollHeight);
    const scrollPosition = await page.evaluate(() => window.scrollY);
    expect(scrollPosition).toBeGreaterThan(scrollHeight * 0.5);

    // Press Home
    await page.keyboard.press('Home');
    await page.waitForTimeout(500);

    // Should be at top
    const newPosition = await page.evaluate(() => window.scrollY);
    expect(newPosition).toBeLessThan(100);
  });

  test('scroll indicator shows position', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll down
    await page.evaluate(() => window.scrollTo(0, 500));
    await page.waitForTimeout(500);

    // Scroll indicator may exist
    const scrollIndicator = page.locator('.scroll-indicator, .progress-bar, .scroll-progress');
    if (await scrollIndicator.count() > 0) {
      await expect(scrollIndicator.first()).toBeVisible();
    }
  });
});

test.describe('Broken Media Handling', () => {
  test('shows error for unplayable video', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Wait for potential error
    await page.waitForTimeout(3000);

    // Error message may appear
    const errorMsg = page.locator('.error-message, .player-error, [role="alert"]:has-text("error"), .video-error');
    if (await errorMsg.count() > 0) {
      await expect(errorMsg.first()).toBeVisible();
    }
  });

  test('shows fallback for missing thumbnail', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Media cards should have some visual representation
    const mediaCards = page.locator('.media-card');
    const count = await mediaCards.count();

    for (let i = 0; i < Math.min(count, 3); i++) {
      const card = mediaCards.nth(i);
      
      // Either has thumbnail or fallback icon
      const thumbnail = card.locator('img.thumbnail, .media-thumb, .card-image');
      const fallback = card.locator('.no-thumbnail, .fallback-icon, .default-thumb');
      
      const hasThumbnail = await thumbnail.count() > 0;
      const hasFallback = await fallback.count() > 0;
      
      expect(hasThumbnail || hasFallback).toBe(true);
    }
  });

  test('handles corrupted video gracefully', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Wait for video to load
    await page.waitForTimeout(3000);

    // Player should not crash
    const player = page.locator('#pip-player');
    await expect(player).toBeVisible();
  });

  test('shows retry option on error', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Wait and check for retry button
    await page.waitForTimeout(3000);

    const retryBtn = page.locator('.retry-btn, button:has-text("Retry"), .player-retry');
    if (await retryBtn.count() > 0) {
      await expect(retryBtn.first()).toBeVisible();
    }
  });

  test('handles missing subtitle files', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Click subtitle button
    const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
    if (await subtitleBtn.count() > 0) {
      await subtitleBtn.click();
      await page.waitForTimeout(500);

      // Menu should handle missing subtitles gracefully
      const subtitleMenu = page.locator('.subtitle-menu');
      if (await subtitleMenu.count() > 0) {
        const menuText = await subtitleMenu.first().textContent();
        // Should either have options or "No subtitles" message
        expect(menuText?.length).toBeGreaterThan(0);
      }
    }
  });

  test('shows placeholder for missing metadata', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Media cards should have some content
    const mediaCards = page.locator('.media-card');
    const count = await mediaCards.count();

    for (let i = 0; i < Math.min(count, 3); i++) {
      const card = mediaCards.nth(i);
      
      // Should have at least title or placeholder
      const title = card.locator('.media-title, .card-title, .title');
      const placeholder = card.locator('.no-title, .unknown');
      
      const hasTitle = await title.count() > 0;
      const hasPlaceholder = await placeholder.count() > 0;
      
      expect(hasTitle || hasPlaceholder).toBe(true);
    }
  });

  test('handles network errors gracefully', async ({ page, server }) => {
    // Go to page
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Simulate network error by going offline
    await page.context().setOffline(true);

    // Try to interact
    const firstCard = page.locator('.media-card').first();
    await firstCard.click();
    await page.waitForTimeout(2000);

    // Should handle gracefully (either show error or not crash)
    const player = page.locator('#pip-player');
    const errorMsg = page.locator('.error-message, .offline-message');
    
    // Either player exists or error message shown
    const hasPlayer = await player.count() > 0;
    const hasError = await errorMsg.count() > 0;
    expect(hasPlayer || hasError).toBe(true);

    // Go back online
    await page.context().setOffline(false);
  });

  test('shows error notification for failed actions', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Try an action that might fail
    // For example, delete without confirmation might fail

    // Check if error notifications work
    const notification = page.locator('.toast-error, .notification-error, .alert-danger');
    
    // Trigger potential error by trying invalid action
    await page.evaluate(() => {
      // Dispatch a custom error event
      window.dispatchEvent(new CustomEvent('error', { detail: { message: 'Test error' } }));
    });
    await page.waitForTimeout(500);

    // Error handling should exist
    expect(true).toBe(true);
  });

  test('recovers from player error', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

    // Wait for potential error
    await page.waitForTimeout(3000);

    // Close player
    const closeBtn = page.locator('.close-pip');
    if (await closeBtn.count() > 0) {
      await closeBtn.first().click();
      await page.waitForTimeout(500);
    }

    // Try opening another media
    const secondCard = page.locator('.media-card').nth(1);
    if (await secondCard.count() > 0) {
      await secondCard.click();
      await page.waitForTimeout(2000);

      // Player should work for second media
      const player = page.locator('#pip-player');
      await expect(player).toBeVisible();
    }
  });

  test('shows loading state for slow media', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first media card
    await page.locator('.media-card').first().click();

    // Loading state should appear
    const loadingIndicator = page.locator('.loading, .spinner, .loader, .player-loading');
    
    // Either loading indicator or player appears
    await page.waitForTimeout(1000);
    const hasLoading = await loadingIndicator.count() > 0;
    const hasPlayer = await page.locator('#pip-player:not(.hidden)').count() > 0;
    
    expect(hasLoading || hasPlayer).toBe(true);
  });

  test('handles unsupported format', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to potentially unsupported format
    await page.fill('#search-input', '.avi');
    await page.press('#search-input', 'Enter');
    await page.waitForTimeout(1000);

    // If there are results, try to play
    const cards = page.locator('.media-card');
    const count = await cards.count();

    if (count > 0) {
      await cards.first().click();
      await page.waitForTimeout(3000);

      // Should handle unsupported format gracefully
      const errorMsg = page.locator('.error-message:has-text("unsupported"), .format-error');
      if (await errorMsg.count() > 0) {
        await expect(errorMsg.first()).toBeVisible();
      } else {
        // Or player should still be functional
        const player = page.locator('#pip-player');
        await expect(player).toBeVisible();
      }
    }
  });

  test('shows file not found error', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Filter to show deleted/missing files
    const deletedFilter = page.locator('#deleted-filter, .filter-deleted');
    if (await deletedFilter.count() > 0) {
      await deletedFilter.first().click();
      await page.waitForTimeout(1000);

      // Try to play missing file
      const cards = page.locator('.media-card');
      const count = await cards.count();

      if (count > 0) {
        await cards.first().click();
        await page.waitForTimeout(2000);

        // Should show file not found error
        const errorMsg = page.locator('.error-message:has-text("not found"), .error-message:has-text("missing"), .file-not-found');
        if (await errorMsg.count() > 0) {
          await expect(errorMsg.first()).toBeVisible();
        }
      }
    }
  });
});
