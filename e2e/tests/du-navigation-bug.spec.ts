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

    // Navigate to fixtures/media folder where test media exists
    // The path should be something like /home/xk/github/xk/discoteca/e2e/fixtures/media/
    const mediaPath = basePath.includes('fixtures') 
      ? basePath + 'media/'
      : '/home/xk/github/xk/discoteca/e2e/fixtures/media/';
    console.log(`[DU Bug Test] Navigating to: ${mediaPath}`);

    await pathInput.fill(mediaPath);
    await pathInput.press('Enter');
    await mediaPage.page.waitForTimeout(2000);

    // Check current path
    const currentPath = await pathInput.inputValue();
    console.log(`[DU Bug Test] Current path: ${currentPath}`);

    // In DU mode, files are rendered as .media-card (not .du-card which is for folders)
    const resultsContainer = mediaPage.page.locator('#results-container');
    // Media file cards have data-path attribute, folder cards don't
    const duCards = resultsContainer.locator('.media-card[data-path]');
    let count = await duCards.count();
    console.log(`[DU Bug Test] Initial media cards count: ${count}`);

    // If no media cards found, check if we have folders to navigate into
    if (count === 0) {
      console.log('[DU Bug Test] No media cards found, checking for folders...');
      // Folder cards are .media-card without data-path attribute
      const duFolders = resultsContainer.locator('.media-card:not([data-path])');
      const folderCount = await duFolders.count();
      console.log(`[DU Bug Test] Found ${folderCount} folders`);

      // Navigate into images or videos folder
      if (folderCount > 0) {
        // Try to find images or videos folder specifically by text content
        let targetFolder;
        for (let i = 0; i < folderCount; i++) {
          const folder = duFolders.nth(i);
          const folderText = await folder.textContent();
          console.log(`[DU Bug Test] Folder ${i}: ${folderText}`);
          if (folderText && (folderText.includes('images') || folderText.includes('videos'))) {
            targetFolder = folder;
            break;
          }
        }
        
        if (!targetFolder) {
          targetFolder = duFolders.first();
        }
        
        console.log('[DU Bug Test] Clicking folder...');
        await targetFolder.click();
        await mediaPage.page.waitForTimeout(3000);

        // Re-check for media cards
        count = await duCards.count();
        console.log(`[DU Bug Test] After navigating into folder, cards count: ${count}`);
        
        // If still no cards, go back and try another folder
        if (count === 0 && folderCount > 1) {
          console.log('[DU Bug Test] Still no cards, trying another folder...');
          await mediaPage.page.keyboard.press('Backspace');
          await mediaPage.page.waitForTimeout(1000);
          
          for (let i = 0; i < folderCount; i++) {
            const folder = duFolders.nth(i);
            if (folder !== targetFolder) {
              await folder.click();
              await mediaPage.page.waitForTimeout(3000);
              count = await duCards.count();
              console.log(`[DU Bug Test] After trying folder ${i}, cards count: ${count}`);
              if (count > 0) break;
            }
          }
        }
      }
    }

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

    // Navigate to fixtures/media folder where test media exists
    const mediaPath = basePath.includes('fixtures') 
      ? basePath + 'media/'
      : '/home/xk/github/xk/discoteca/e2e/fixtures/media/';
    await pathInput.fill(mediaPath);
    await pathInput.press('Enter');
    await mediaPage.page.waitForTimeout(2000);

    // Find media cards with fallback logic
    const resultsContainer = mediaPage.page.locator('#results-container');
    // Media file cards have data-path attribute, folder cards don't
    const duCards = resultsContainer.locator('.media-card[data-path]');
    let count = await duCards.count();
    console.log(`[DU Delete Test] Initial media cards count: ${count}`);

    // If no media cards found, check if we have folders to navigate into
    if (count === 0) {
      // Folder cards are .media-card without data-path attribute
      const duFolders = resultsContainer.locator('.media-card:not([data-path])');
      const folderCount = await duFolders.count();
      console.log(`[DU Delete Test] Found ${folderCount} folders`);

      if (folderCount > 0) {
        let targetFolder;
        for (let i = 0; i < folderCount; i++) {
          const folder = duFolders.nth(i);
          const folderText = await folder.textContent();
          console.log(`[DU Delete Test] Folder ${i}: ${folderText}`);
          if (folderText && (folderText.includes('images') || folderText.includes('videos'))) {
            targetFolder = folder;
            break;
          }
        }
        
        if (!targetFolder) {
          targetFolder = duFolders.first();
        }
        
        await targetFolder.click();
        await mediaPage.page.waitForTimeout(3000);
        count = await duCards.count();
        console.log(`[DU Delete Test] After navigating into folder, cards count: ${count}`);
      }
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

  test('keyboard navigation (n/p keys) uses DU siblings', async ({ mediaPage, viewerPage, server }) => {
    // Navigate to DU mode
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    // Wait for DU view to load
    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Get the base path
    const pathInput = mediaPage.getDUPathInput();
    const basePath = await pathInput.inputValue();

    // Navigate to fixtures/media folder where test media exists
    const mediaPath = basePath.includes('fixtures') 
      ? basePath + 'media/'
      : '/home/xk/github/xk/discoteca/e2e/fixtures/media/';
    await pathInput.fill(mediaPath);
    await pathInput.press('Enter');
    await mediaPage.page.waitForTimeout(2000);

    // Find media cards with fallback logic
    const resultsContainer = mediaPage.page.locator('#results-container');
    // Media file cards have data-path attribute, folder cards don't
    const duCards = resultsContainer.locator('.media-card[data-path]');
    let count = await duCards.count();
    console.log(`[DU Arrow Test] Initial media cards count: ${count}`);

    // If no media cards found, check if we have folders to navigate into
    if (count === 0) {
      // Folder cards are .media-card without data-path attribute
      const duFolders = resultsContainer.locator('.media-card:not([data-path])');
      const folderCount = await duFolders.count();
      console.log(`[DU Arrow Test] Found ${folderCount} folders`);

      if (folderCount > 0) {
        let targetFolder;
        for (let i = 0; i < folderCount; i++) {
          const folder = duFolders.nth(i);
          const folderText = await folder.textContent();
          console.log(`[DU Arrow Test] Folder ${i}: ${folderText}`);
          if (folderText && (folderText.includes('images') || folderText.includes('videos'))) {
            targetFolder = folder;
            break;
          }
        }
        
        if (!targetFolder) {
          targetFolder = duFolders.first();
        }
        
        await targetFolder.click();
        await mediaPage.page.waitForTimeout(3000);
        count = await duCards.count();
        console.log(`[DU Arrow Test] After navigating into folder, cards count: ${count}`);
      }
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
      }
    }

    expect(duPaths.length).toBeGreaterThanOrEqual(2);

    // Click on the first media item to play it
    await duCards.nth(0).click();
    await viewerPage.waitForPlayer();

    // Verify we're playing the first item
    const firstItemName = duPaths[0].split('/').pop();
    await expect(viewerPage.mediaTitle).toContainText(firstItemName || '');

    // Press 'n' to go to next media
    await mediaPage.page.keyboard.press('n');
    await mediaPage.page.waitForTimeout(500);

    // Verify we're now playing the second item from DU mode (not search mode)
    const secondItemName = duPaths[1].split('/').pop();
    await expect(viewerPage.mediaTitle).toContainText(secondItemName || '');

    // Press 'p' to go to previous media
    await mediaPage.page.keyboard.press('p');
    await mediaPage.page.waitForTimeout(500);

    // Verify we're back to the first item from DU mode
    await expect(viewerPage.mediaTitle).toContainText(firstItemName || '');
  });
});
