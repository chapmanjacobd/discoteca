/**
 * E2E tests for Disk Usage mode navigation bug
 * Tests that next/previous navigation uses DU mode siblings, not search mode siblings
 * 
 * Bug: When playing media from DU mode, pressing next/delete uses search mode siblings
 * instead of DU mode siblings.
 */
import { test, expect } from '../fixtures';

test.describe('DU Mode Navigation Bug', () => {
  test('next/previous navigation uses DU siblings, not search siblings', async ({ mediaPage, viewerPage, server }) => {
    // Navigate to DU mode
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    // Wait for DU view to load
    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
    
    // Get the base path from the path input
    const pathInput = mediaPage.getDUPathInput();
    const basePath = await pathInput.inputValue();
    console.log(`[DU Bug Test] Base path: ${basePath}`);
    
    // Navigate to images folder using full path (append to base path)
    const imagesPath = basePath + 'images/';
    console.log(`[DU Bug Test] Navigating to: ${imagesPath}`);
    
    await pathInput.fill(imagesPath);
    await pathInput.press('Enter');
    await mediaPage.page.waitForTimeout(2000);
    
    // In DU mode, files are rendered as .media-card (not .du-card which is for folders)
    // Use results container to get all cards
    const resultsContainer = mediaPage.page.locator('#results-container');
    const duCards = resultsContainer.locator('.media-card');
    let count = await duCards.count();
    console.log(`[DU Bug Test] After navigating to images, cards count: ${count}`);
    
    // Get all card details for debugging
    for (let i = 0; i < count && i < 5; i++) {
      const card = duCards.nth(i);
      const cardPath = await card.getAttribute('data-path');
      const cardClass = await card.getAttribute('class');
      console.log(`[DU Bug Test] Card ${i}: class="${cardClass}", path="${cardPath}"`);
    }

    // Assert we have enough media items
    expect(count).toBeGreaterThanOrEqual(2);

    // Get the paths of the first few items in DU mode
    const duPaths = [];
    for (let i = 0; i < Math.min(3, count); i++) {
      const card = duCards.nth(i);
      const cardPath = await card.getAttribute('data-path');
      if (cardPath) {
        duPaths.push(cardPath);
        console.log(`[DU Bug Test] DU path ${i}: ${cardPath}`);
      }
    }
    
    expect(duPaths.length).toBeGreaterThanOrEqual(2);
    
    // Click on the first media item to play it
    console.log(`[DU Bug Test] Clicking first card: ${duPaths[0]}`);
    await duCards.nth(0).click();
    await viewerPage.waitForPlayer();
    
    // Verify we're playing the first item
    const firstItemName = duPaths[0].split('/').pop();
    console.log(`[DU Bug Test] Expected to play: ${firstItemName}`);
    await expect(viewerPage.mediaTitle).toContainText(firstItemName || '');
    
    // Press 'n' to go to next media
    console.log(`[DU Bug Test] Pressing 'n' for next`);
    await mediaPage.page.keyboard.press('n');
    await mediaPage.page.waitForTimeout(500);
    
    // Verify we're now playing the second item from DU mode (not search mode)
    const secondItemName = duPaths[1].split('/').pop();
    console.log(`[DU Bug Test] Expected next: ${secondItemName}`);
    const currentTitle = await viewerPage.mediaTitle.textContent();
    console.log(`[DU Bug Test] Actual title after next: ${currentTitle}`);
    await expect(viewerPage.mediaTitle).toContainText(secondItemName || '');
    
    // Press 'p' to go to previous media
    console.log(`[DU Bug Test] Pressing 'p' for previous`);
    await mediaPage.page.keyboard.press('p');
    await mediaPage.page.waitForTimeout(500);
    
    // Verify we're back to the first item from DU mode
    const prevTitle = await viewerPage.mediaTitle.textContent();
    console.log(`[DU Bug Test] Actual title after prev: ${prevTitle}`);
    await expect(viewerPage.mediaTitle).toContainText(firstItemName || '');
  });

  test('delete in DU mode navigates to DU sibling', async ({ mediaPage, viewerPage, server }) => {
    // Navigate to DU mode
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    // Wait for DU view to load
    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
    
    // Get the base path
    const pathInput = mediaPage.getDUPathInput();
    const basePath = await pathInput.inputValue();
    
    // Navigate to images folder using full path
    const imagesPath = basePath + 'images/';
    await pathInput.fill(imagesPath);
    await pathInput.press('Enter');
    await mediaPage.page.waitForTimeout(2000);
    
    // In DU mode, files are rendered as .media-card (not .du-card which is for folders)
    const resultsContainer = mediaPage.page.locator('#results-container');
    const duCards = resultsContainer.locator('.media-card');
    let count = await duCards.count();
    console.log(`[DU Delete Test] Cards in images: ${count}`);

    // Assert we have enough media items
    expect(count).toBeGreaterThanOrEqual(2);

    // Get the paths of the first few items in DU mode
    const duPaths = [];
    for (let i = 0; i < Math.min(3, count); i++) {
      const card = duCards.nth(i);
      const cardPath = await card.getAttribute('data-path');
      if (cardPath) {
        duPaths.push(cardPath);
      }
    }
    
    expect(duPaths.length).toBeGreaterThanOrEqual(2);
    
    // Click on the first media item to play it
    await duCards.nth(0).click();
    await viewerPage.waitForPlayer();
    
    // Verify we're playing the first item
    const firstItemName = duPaths[0].split('/').pop();
    await expect(viewerPage.mediaTitle).toContainText(firstItemName || '');
    
    // Press Delete to delete current media and go to next
    await mediaPage.page.keyboard.press('Delete');
    await mediaPage.page.waitForTimeout(1000);
    
    // Verify we're now playing the second item from DU mode (not search mode)
    const secondItemName = duPaths[1].split('/').pop();
    await expect(viewerPage.mediaTitle).toContainText(secondItemName || '');
  });

  test('arrow keys navigation uses DU siblings', async ({ mediaPage, viewerPage, server }) => {
    // Navigate to DU mode
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    // Wait for DU view to load
    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
    
    // Get the base path
    const pathInput = mediaPage.getDUPathInput();
    const basePath = await pathInput.inputValue();
    
    // Navigate to images folder using full path
    const imagesPath = basePath + 'images/';
    await pathInput.fill(imagesPath);
    await pathInput.press('Enter');
    await mediaPage.page.waitForTimeout(2000);
    
    // In DU mode, files are rendered as .media-card (not .du-card which is for folders)
    const resultsContainer = mediaPage.page.locator('#results-container');
    const duCards = resultsContainer.locator('.media-card');
    let count = await duCards.count();
    console.log(`[DU Arrow Test] Cards in images: ${count}`);

    // Assert we have enough media items
    expect(count).toBeGreaterThanOrEqual(2);

    // Get the paths of the first few items in DU mode
    const duPaths = [];
    for (let i = 0; i < Math.min(3, count); i++) {
      const card = duCards.nth(i);
      const cardPath = await card.getAttribute('data-path');
      if (cardPath) {
        duPaths.push(cardPath);
      }
    }
    
    expect(duPaths.length).toBeGreaterThanOrEqual(2);
    
    // Click on the first media item to play it
    await duCards.nth(0).click();
    await viewerPage.waitForPlayer();
    
    // Verify we're playing the first item
    const firstItemName = duPaths[0].split('/').pop();
    await expect(viewerPage.mediaTitle).toContainText(firstItemName || '');
    
    // Press ArrowRight to go to next media
    await mediaPage.page.keyboard.press('ArrowRight');
    await mediaPage.page.waitForTimeout(500);
    
    // Verify we're now playing the second item from DU mode (not search mode)
    const secondItemName = duPaths[1].split('/').pop();
    await expect(viewerPage.mediaTitle).toContainText(secondItemName || '');
    
    // Press ArrowLeft to go to previous media
    await mediaPage.page.keyboard.press('ArrowLeft');
    await mediaPage.page.waitForTimeout(500);
    
    // Verify we're back to the first item from DU mode
    await expect(viewerPage.mediaTitle).toContainText(firstItemName || '');
  });
});
