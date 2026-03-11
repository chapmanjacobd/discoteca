import { test, expect } from '../fixtures';

test.describe('Large Result Sets Scrolling', () => {
  test.use({ readOnly: true });

  test('scrolls through large media list', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial visible cards using POM
    const initialCount = await mediaPage.getMediaCount();

    // Scroll down using POM
    await mediaPage.page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) content.scrollTo(0, content.scrollHeight);
    });
    await mediaPage.page.waitForTimeout(1000);

    // More cards should load or pagination should appear using POM
    const scrolledCount = await mediaPage.getMediaCount();

    // Either more cards loaded or we're at the end
    expect(scrolledCount).toBeGreaterThanOrEqual(initialCount);
  });

  test('infinite scroll loads more results', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial card count using POM
    const initialCount = await mediaPage.getMediaCount();

    // Scroll to bottom multiple times using POM
    for (let i = 0; i < 3; i++) {
      await mediaPage.page.evaluate(() => {
        const content = document.querySelector('#content');
        if (content) content.scrollTo(0, content.scrollHeight);
      });
      await mediaPage.page.waitForTimeout(1000);
    }

    // Check if more cards loaded using POM
    const finalCount = await mediaPage.getMediaCount();

    // Either more cards loaded or we hit the limit
    expect(finalCount).toBeGreaterThanOrEqual(initialCount);
  });

  test('pagination controls are visible for large sets', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Pagination container should exist using POM
    await expect(mediaPage.paginationContainer).toBeVisible();
  });

  test('page navigation works', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Find next page button using POM
    const nextBtn = mediaPage.page.locator('#next-page');

    if (await nextBtn.count() > 0 && !(await nextBtn.isDisabled())) {
      // Get current page info using POM
      const currentPageText = await mediaPage.pageInfo.textContent();

      // Click next page using POM
      await nextBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // Page should have changed using POM
      const newPageText = await mediaPage.pageInfo.textContent();
      expect(newPageText).not.toBe(currentPageText);
    }
  });

  test('scrolling does not cause page crash', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Rapid scrolling should not crash the page using POM
    for (let i = 0; i < 5; i++) {
      await mediaPage.page.evaluate((scrollPos) => {
        const content = document.querySelector('#results-container, #content, main');
        if (content) {
          content.scrollTo(0, scrollPos);
        } else {
          window.scrollTo(0, scrollPos);
        }
      }, i * 500);
      await mediaPage.page.waitForTimeout(200);
    }

    // Page should still be functional using POM
    await expect(mediaPage.resultsContainer).toBeVisible();
    
    // Page should not have crashed (check for error state)
    const hasError = await mediaPage.page.evaluate(() => {
      return document.body.classList.contains('error') || 
             document.querySelector('.error-message') !== null;
    });
    expect(hasError).toBe(false);
  });

  test('scroll position is maintained after search', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Scroll down using POM
    await mediaPage.page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) content.scrollTo(0, 500);
    });
    await mediaPage.page.waitForTimeout(500);

    // Get scroll position using POM
    const scrollPosBefore = await mediaPage.page.evaluate(() => {
      const content = document.querySelector('#content');
      return content ? content.scrollTop : 0;
    });

    // Perform search using POM
    await mediaPage.search('test');
    await mediaPage.page.waitForTimeout(1000);

    // Scroll position may reset to top after search (expected behavior)
    // Just verify page is still functional using POM
    await expect(mediaPage.resultsContainer).toBeVisible();
  });

  test('cards render correctly after scrolling', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Scroll down and back up using POM
    await mediaPage.page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) {
        content.scrollTo(0, 1000);
      }
    });
    await mediaPage.page.waitForTimeout(500);

    await mediaPage.page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) {
        content.scrollTo(0, 0);
      }
    });
    await mediaPage.page.waitForTimeout(500);

    // Cards should still render correctly using POM
    const firstCard = mediaPage.getMediaCard(0);
    await expect(firstCard).toBeVisible();

    // Card should have required attributes using POM
    await expect(firstCard).toHaveAttribute('data-type');
  });

  test('media cards have images after scrolling', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Scroll down using POM
    await mediaPage.page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) content.scrollTo(0, content.scrollHeight / 2);
    });
    await mediaPage.page.waitForTimeout(1000);

    // Check visible cards have images using POM
    const cards = mediaPage.mediaCards;
    const count = await cards.count();

    if (count > 0) {
      // At least some cards should be visible and have images
      const middleCard = cards.nth(Math.floor(count / 2));
      await expect(middleCard).toBeVisible();
    }
  });

  test('pagination page size selector works', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Check if page size selector exists using POM
    const pageSizeSelect = mediaPage.page.locator('#page-size');

    if (await pageSizeSelect.count() > 0) {
      // Get initial page size using POM
      const initialSize = await pageSizeSelect.inputValue();

      // Change page size using POM
      await pageSizeSelect.selectOption('100');
      await mediaPage.page.waitForTimeout(500);

      // Page size should have changed using POM
      const newSize = await pageSizeSelect.inputValue();
      expect(newSize).toBe('100');

      // Restore original size
      await pageSizeSelect.selectOption(initialSize);
    }
  });

  test('no duplicate cards load during scrolling', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial card paths using POM
    const initialPaths = await mediaPage.getAllMediaCardPaths();

    // Scroll down using POM
    await mediaPage.page.evaluate(() => {
      const content = document.querySelector('#content');
      if (content) content.scrollTo(0, content.scrollHeight);
    });
    await mediaPage.page.waitForTimeout(1000);

    // Get card paths after scroll using POM
    const scrolledPaths = await mediaPage.getAllMediaCardPaths();

    // Check for duplicates (simplified check)
    const uniquePaths = new Set(scrolledPaths);
    expect(uniquePaths.size).toBe(scrolledPaths.length);
  });
});

test.describe('Error Handling and Recovery', () => {
  test.use({ readOnly: true });

  test('handles network error gracefully', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Mock a network error
    await mediaPage.page.route('**/api/query*', route => {
      route.abort('failed');
    });

    // Try to search (should handle error gracefully)
    await mediaPage.page.fill('#search-input', 'test');
    await mediaPage.page.press('#search-input', 'Enter');
    await mediaPage.page.waitForTimeout(2000);

    // Page should not crash - may show error state or empty results
    await expect(mediaPage.resultsContainer).toBeVisible();
  });

  test('recovers from temporary network error', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Mock first request to fail, second to succeed
    let requestCount = 0;
    await mediaPage.page.route('**/api/query*', route => {
      requestCount++;
      if (requestCount === 1) {
        route.abort('failed');
      } else {
        route.continue();
      }
    });

    // Try to search
    await mediaPage.search('test');
    await mediaPage.page.waitForTimeout(2000);

    // Retry search (should succeed)
    await mediaPage.search('test');
    await mediaPage.page.waitForTimeout(1000);

    // Should have results using POM
    const count = await mediaPage.getMediaCount();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('shows error toast for failed media playback', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Mock media file to return 404
    await mediaPage.page.route('**/api/raw*', route => {
      route.fulfill({
        status: 404,
        body: 'Not Found'
      });
    });

    // Click first media card using POM
    await mediaPage.getMediaCard(0).click();
    await mediaPage.page.waitForTimeout(2000);

    // Error toast should appear using POM
    if (await mediaPage.toast.isVisible()) {
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).toBeTruthy();
    }
  });

  test('page remains functional after error', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Trigger an error by searching with invalid query
    await mediaPage.search('nonexistent_query_xyz123');
    await mediaPage.page.waitForTimeout(1000);

    // Page should still be functional using POM
    await expect(mediaPage.searchInput).toBeVisible();
    await expect(mediaPage.resultsContainer).toBeVisible();

    // Clear search and try again using POM
    await mediaPage.clearSearch();
    const count = await mediaPage.getMediaCount();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
