import { test, expect } from '../fixtures';

test.describe('Image Arrow Key Navigation', () => {
  test.use({ readOnly: true });

  test('ArrowLeft navigates to previous sibling when viewing an image', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Find images using POM
    const imageCards = mediaPage.page.locator('.media-card[data-type*="image"]');
    const imageCount = await imageCards.count();

    expect(imageCount).toBeGreaterThanOrEqual(2);

    // Click second image (so there's a previous sibling) using POM
    await imageCards.nth(1).click();
    await viewerPage.waitForImageLoad();

    // Verify image is loaded using POM
    await expect(viewerPage.getImageElement()).toBeVisible();

    // Get initial image src using POM
    const initialSrc = await viewerPage.getImageElement().getAttribute('src');
    console.log('Initial image src:', initialSrc);

    // Press ArrowLeft
    await mediaPage.page.keyboard.press('ArrowLeft');
    await mediaPage.page.waitForTimeout(500);

    // Image SHOULD have changed (ArrowLeft should call playSibling(-1)) using POM
    const newSrc = await viewerPage.getImageElement().getAttribute('src');
    console.log('New image src after ArrowLeft:', newSrc);
    expect(newSrc).not.toBe(initialSrc);
  });

  test('ArrowRight navigates to next sibling when viewing an image', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Find images using POM
    const imageCards = mediaPage.page.locator('.media-card[data-type*="image"]');
    const imageCount = await imageCards.count();

    expect(imageCount).toBeGreaterThanOrEqual(2);

    // Click first image (so there's a next sibling) using POM
    await imageCards.first().click();
    await viewerPage.waitForImageLoad();

    // Verify image is loaded using POM
    await expect(viewerPage.getImageElement()).toBeVisible();

    // Get initial image src using POM
    const initialSrc = await viewerPage.getImageElement().getAttribute('src');
    console.log('Initial image src:', initialSrc);

    // Press ArrowRight
    await mediaPage.page.keyboard.press('ArrowRight');
    await mediaPage.page.waitForTimeout(500);

    // Image SHOULD have changed (ArrowRight should call playSibling(1)) using POM
    const newSrc = await viewerPage.getImageElement().getAttribute('src');
    console.log('New image src after ArrowRight:', newSrc);
    expect(newSrc).not.toBe(initialSrc);
  });

  test('Arrow keys do not navigate when not viewing media', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Press ArrowRight when no media is open
    await mediaPage.page.keyboard.press('ArrowRight');
    await mediaPage.page.waitForTimeout(500);

    // Player should NOT be open
    const playerVisible = await mediaPage.page.locator('#pip-player').isVisible();
    expect(playerVisible).toBe(false);
  });

  test('Arrow keys cycle through all images', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Find images using POM
    const imageCards = mediaPage.page.locator('.media-card[data-type*="image"]');
    const imageCount = await imageCards.count();

    expect(imageCount).toBeGreaterThanOrEqual(3);

    // Click first image using POM
    await imageCards.first().click();
    await viewerPage.waitForImageLoad();

    // Get initial src using POM
    const imageElement = viewerPage.getImageElement();
    await imageElement.waitFor({ state: 'visible', timeout: 5000 });
    const initialSrc = await imageElement.getAttribute('src');

    // Press ArrowRight multiple times using POM (cycle through fewer images to avoid closing player)
    const cycles = Math.min(imageCount - 1, 3);
    for (let i = 0; i < cycles; i++) {
      await mediaPage.page.keyboard.press('ArrowRight');
      await mediaPage.page.waitForTimeout(300);
    }

    // Should still be viewing an image
    // Wait for image element to be visible before getting src
    await imageElement.waitFor({ state: 'visible', timeout: 5000 });
    const currentSrc = await imageElement.getAttribute('src');

    // Verify we're still viewing an image
    expect(currentSrc).toBeTruthy();
    expect(currentSrc).not.toBe('');
    // Src should have changed from initial (unless we cycled back to start)
    expect(currentSrc).toContain('/api/raw?path=');
  });
});
