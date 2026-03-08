import { test, expect } from '../fixtures';

test.describe('Group View Error Handling', () => {
  test('Group view should not reset to Grid view on media error', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // 1. Switch to Group view
    await page.click('#view-group');
    
    // Wait for group view to load (similarity-view class)
    await page.waitForSelector('#results-container.similarity-view', { timeout: 10000 });
    
    // Verify we have some group headers
    const groupHeaders = page.locator('.similarity-header');
    await expect(groupHeaders.first()).toBeVisible();

    // 2. Mock a 404 error for any media request
    // We want to trigger handleMediaError
    await page.route('**/api/raw*', route => {
      route.fulfill({
        status: 404,
        body: 'Not Found'
      });
    });

    // 3. Click a media card in Group view
    const mediaCard = page.locator('.media-card').first();
    await mediaCard.click();

    // 4. Wait for error toast
    const toast = page.locator('#toast');
    await expect(toast).toBeVisible({ timeout: 10000 });
    
    // 5. Verify the view is STILL Group view
    // This is where it currently fails (it resets to 'grid')
    const resultsContainer = page.locator('#results-container');
    await expect(resultsContainer).toHaveClass(/similarity-view/);
    
    // Also verify group headers are still there
    await expect(groupHeaders.first()).toBeVisible();
  });
});
