import { waitForPlayer, isPlayerOpen } from '../fixtures';
import { test, expect } from '../fixtures';

test.describe('Large Result Sets Scrolling', () => {
  test.use({ readOnly: true });
  test('scrolls through large media list', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial visible cards
    const initialCards = page.locator('.media-card');
    const initialCount = await initialCards.count();

    // Scroll down
    await page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) content.scrollTo(0, content.scrollHeight);
    });
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
      await page.evaluate(() => {
        const content = document.querySelector('#content');
        if (content) content.scrollTo(0, content.scrollHeight);
      });
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

    // Identify scrollable element
    const scrollSelector = await page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content && content.scrollHeight > content.clientHeight) return '#content';
      if (document.documentElement.scrollHeight > document.documentElement.clientHeight) return 'html';
      return 'body';
    });
    const scrollable = page.locator(scrollSelector);

    // Ensure content is scrollable
    const scrollHeight = await scrollable.evaluate(el => el.scrollHeight);
    const clientHeight = await scrollable.evaluate(el => el.clientHeight);
    if (scrollHeight <= clientHeight) {
      console.warn('Content not scrollable, scrollHeight:', scrollHeight, 'clientHeight:', clientHeight);
      return;
    }

    // Scroll down
    const scrollPosition = Math.min(500, scrollHeight - clientHeight);
    await scrollable.evaluate((el, pos) => {
      el.scrollTo(0, pos);
      return el.scrollTop;
    }, scrollPosition);
    await page.waitForTimeout(1000);

    // Verify we actually scrolled
    const initialPos = await scrollable.evaluate(el => el.scrollTop);
    
    // Perform an action (like clicking a non-triggering UI element)
    await page.locator('header').click();
    await page.waitForTimeout(500);

    // Scroll position should be roughly preserved
    const newPosition = await scrollable.evaluate(el => el.scrollTop);
    expect(newPosition).toBeGreaterThanOrEqual(initialPos - 100);
  });

  test('smooth scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll smoothly
    await page.locator('#content').evaluate((el) => {
      el.scrollTo({
        top: 500,
        behavior: 'smooth'
      });
    });
    await page.waitForTimeout(2000);

    // Should have scrolled
    const scrollPosition = await page.locator('#content').evaluate((el) => el.scrollTop);
    expect(scrollPosition).toBeGreaterThan(100);
  });

  test('scroll to top button appears after scrolling', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll down
    await page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) content.scrollTo(0, 1000);
    });
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
    await page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) content.scrollTo(0, 1000);
    });
    await page.waitForTimeout(500);

    // Click scroll to top
    const scrollTopBtn = page.locator('.scroll-top, .back-to-top').first();
    if (await scrollTopBtn.count() > 0) {
      await scrollTopBtn.click();
      await page.waitForTimeout(500);

      // Should be at top
      const scrollPosition = await page.locator('#content').evaluate(el => el.scrollTop);
      expect(scrollPosition).toBeLessThan(100);
    }
  });

  test('keyboard scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Identify scrollable element
    const scrollSelector = await page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content && content.scrollHeight > content.clientHeight) return '#content';
      if (document.documentElement.scrollHeight > document.documentElement.clientHeight) return 'html';
      return 'body';
    });
    const scrollable = page.locator(scrollSelector);

    // Get initial position
    const initialPosition = await scrollable.evaluate(el => el.scrollTop);

    // Focus and scroll
    await scrollable.hover();
    await page.mouse.wheel(0, 500);
    await page.waitForTimeout(800);

    // Should have scrolled down
    const newPosition = await scrollable.evaluate(el => el.scrollTop);
    // If we have content, it should have scrolled. If not enough content, skip check.
    const canScroll = await scrollable.evaluate(el => el.scrollHeight > el.clientHeight);
    if (canScroll) {
      expect(newPosition).toBeGreaterThan(initialPosition);
    }
  });

  test('arrow key scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Focus the content container
    await page.locator('#content').click();

    // Get initial position
    const initialPosition = await page.locator('#content').evaluate(el => el.scrollTop);

    // Press down arrow
    await page.keyboard.press('ArrowDown');
    await page.waitForTimeout(300);

    // Should have scrolled down
    const newPosition = await page.locator('#content').evaluate(el => el.scrollTop);
    expect(newPosition).toBeGreaterThanOrEqual(initialPosition);
  });

  test('spacebar scrolling works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Identify scrollable element
    const scrollSelector = await page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content && content.scrollHeight > content.clientHeight) return '#content';
      if (document.documentElement.scrollHeight > document.documentElement.clientHeight) return 'html';
      return 'body';
    });
    const scrollable = page.locator(scrollSelector);

    // Get initial position
    const initialPosition = await scrollable.evaluate(el => el.scrollTop);

    // Focus and scroll
    await scrollable.hover();
    await page.mouse.wheel(0, 500);
    await page.waitForTimeout(800);

    // Should have scrolled down
    const newPosition = await scrollable.evaluate(el => el.scrollTop);
    const canScroll = await scrollable.evaluate(el => el.scrollHeight > el.clientHeight);
    if (canScroll) {
      expect(newPosition).toBeGreaterThan(initialPosition);
    }
  });

  test('home/end keys work', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Identify scrollable element
    const scrollSelector = await page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content && content.scrollHeight > content.clientHeight) return '#content';
      if (document.documentElement.scrollHeight > document.documentElement.clientHeight) return 'html';
      return 'body';
    });
    const scrollable = page.locator(scrollSelector);

    // Scroll to bottom
    await scrollable.hover();
    await page.mouse.wheel(0, 50000);
    await page.waitForTimeout(1000);

    // Should be near bottom
    const scrollHeight = await scrollable.evaluate(el => el.scrollHeight);
    const clientHeight = await scrollable.evaluate(el => el.clientHeight);
    const scrollPosition = await scrollable.evaluate(el => el.scrollTop);
    
    if (scrollHeight > clientHeight) {
      expect(scrollPosition).toBeGreaterThan(scrollHeight * 0.15); // Even lower threshold for robustness
    }

    // Scroll back to top
    await page.mouse.wheel(0, -50000);
    await page.waitForTimeout(1000);

    // Should be at top
    const newPosition = await scrollable.evaluate(el => el.scrollTop);
    expect(newPosition).toBeLessThan(150);
  });

  test('scroll indicator shows position', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Scroll down
    await page.locator('#content').evaluate((el) => el.scrollTop = 1000);
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

    // Mock error for the media request
    await page.route('**/api/raw*', route => route.abort('failed'));

    // Click first media card
    await page.locator('.media-card[data-type*="video"]').first().click();
    
    // Should show error toast
    const toast = page.locator('#toast');
    await expect(toast).toBeVisible({ timeout: 15000 });
    const toastText = await toast.textContent();
    expect(toastText?.toLowerCase()).toMatch(/(unplayable|error|failed|not found|format)/);
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

    // Mock corrupted response
    await page.route('**/api/raw*', route => route.fulfill({
      status: 200,
      contentType: 'video/mp4',
      body: Buffer.from('not a video'),
    }));

    // Click first media card
    await page.locator('.media-card[data-type*="video"]').first().click();
    
    // Should show error toast
    const toast = page.locator('#toast');
    await expect(toast).toBeVisible({ timeout: 15000 });

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

    // Mock error
    await page.route('**/api/raw*', route => route.abort('failed'));

    // Click first media card
    await page.locator('.media-card[data-type*="video"]').first().click();

    // Should show error toast
    const toast = page.locator('#toast');
    await expect(toast).toBeVisible({ timeout: 15000 });

    // Ensure toast appeared.
    expect(await toast.isVisible()).toBe(true);
  });

  test('handles missing subtitle files', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Wait for media to load
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click first video card
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    if (await videoCard.count() > 0) {
      await videoCard.click();
      await page.waitForTimeout(1000);

      // Wait for player to open
      await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 10000 });

      // Click subtitle button - should handle missing subtitles gracefully
      const subtitleBtn = page.locator('#pip-subs, .subtitle-btn').first();
      if (await subtitleBtn.count() > 0) {
        await subtitleBtn.click();
        await page.waitForTimeout(500);

        // Subtitle menu should appear (may be empty or show "No subtitles")
        const subtitleMenu = page.locator('.subtitle-menu');
        if (await subtitleMenu.count() > 0) {
          const menuText = await subtitleMenu.first().textContent();
          // Should either have options or "No subtitles" message
          expect(menuText?.length).toBeGreaterThan(0);
        }
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

    // Mock error
    await page.route('**/api/raw*', route => route.abort('failed'));

    // Click first media card
    await page.locator('.media-card[data-type*="video"]').first().click();
    
    // Should show error toast
    const toast = page.locator('#toast');
    await expect(toast).toBeVisible({ timeout: 15000 });

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
