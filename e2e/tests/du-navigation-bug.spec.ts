/**
 * E2E tests for Disk Usage mode navigation
 * Tests that next/previous/delete navigation uses DU mode siblings
 */
import { test, expect } from '../fixtures';
import * as path from 'path';

test.describe('DU Mode Navigation', () => {
  test.use({ readOnly: true });

  // Calculate the absolute path to fixtures/media at runtime
  // This works in CI because it's relative to the test file location
  const fixturesMediaPath = path.resolve(__dirname, '../fixtures/media');

  test('next/previous navigation uses DU siblings', async ({ mediaPage, viewerPage, server }) => {
    // Navigate directly to the images folder using absolute path
    const imagesPath = path.join(fixturesMediaPath, 'images');
    await mediaPage.goto(server.getBaseUrl() + `/#mode=du&path=${encodeURIComponent(imagesPath)}`);
    await mediaPage.page.waitForTimeout(1000);

    // Get file cards and ensure we have at least 2 files
    const fileCards = mediaPage.getDUFileCards();
    const fileCount = await fileCards.count();
    console.log(`[DU Test] Found ${fileCount} files in images folder`);
    
    expect(fileCount).toBeGreaterThanOrEqual(2);

    // Get the first two file paths
    const files = await mediaPage.getDUFiles();
    const firstName = files[0].split('/').pop() || '';
    const secondName = files[1].split('/').pop() || '';
    console.log(`[DU Test] First: ${firstName}, Second: ${secondName}`);

    // Click first file to play it
    await fileCards.nth(0).click();
    await viewerPage.waitForPlayer();

    // Verify we're playing the first item
    await expect(viewerPage.mediaTitle).toContainText(firstName);

    // Press 'n' to go to next
    await mediaPage.page.keyboard.press('n');
    await mediaPage.page.waitForTimeout(500);

    // Verify we're now playing the second item (DU sibling, not search sibling)
    const titleAfterNext = await viewerPage.mediaTitle.textContent() || '';
    console.log(`[DU Test] After next: ${titleAfterNext}`);
    await expect(viewerPage.mediaTitle).toContainText(secondName);

    // Press 'p' to go to previous
    await mediaPage.page.keyboard.press('p');
    await mediaPage.page.waitForTimeout(500);

    // Verify we're back to the first item
    const titleAfterPrev = await viewerPage.mediaTitle.textContent() || '';
    console.log(`[DU Test] After prev: ${titleAfterPrev}`);
    await expect(viewerPage.mediaTitle).toContainText(firstName);
  });

  test('delete in DU mode navigates to DU sibling', async ({ mediaPage, viewerPage, server }) => {
    // Navigate directly to the images folder using absolute path
    const imagesPath = path.join(fixturesMediaPath, 'images');
    await mediaPage.goto(server.getBaseUrl() + `/#mode=du&path=${encodeURIComponent(imagesPath)}`);
    await mediaPage.page.waitForTimeout(1000);

    // Get file cards and ensure we have at least 2 files
    const fileCards = mediaPage.getDUFileCards();
    const fileCount = await fileCards.count();
    console.log(`[DU Delete Test] Found ${fileCount} files in images folder`);
    
    expect(fileCount).toBeGreaterThanOrEqual(2);

    // Get the first two file paths
    const files = await mediaPage.getDUFiles();
    const firstName = files[0].split('/').pop() || '';
    const secondName = files[1].split('/').pop() || '';
    console.log(`[DU Delete Test] First: ${firstName}, Second: ${secondName}`);

    // Click first file to play it
    await fileCards.nth(0).click();
    await viewerPage.waitForPlayer();

    // Verify we're playing the first item
    await expect(viewerPage.mediaTitle).toContainText(firstName);

    // Press Delete to go to next (in read-only mode, it navigates without deleting)
    await mediaPage.page.keyboard.press('Delete');
    await mediaPage.page.waitForTimeout(1000);

    // Verify we're now playing the second item (DU sibling)
    const titleAfterDelete = await viewerPage.mediaTitle.textContent() || '';
    console.log(`[DU Delete Test] After delete: ${titleAfterDelete}`);
    await expect(viewerPage.mediaTitle).toContainText(secondName);
  });

  test('keyboard navigation (n/p keys) uses DU siblings', async ({ mediaPage, viewerPage, server }) => {
    // Navigate directly to the images folder using absolute path
    const imagesPath = path.join(fixturesMediaPath, 'images');
    await mediaPage.goto(server.getBaseUrl() + `/#mode=du&path=${encodeURIComponent(imagesPath)}`);
    await mediaPage.page.waitForTimeout(1000);

    // Get file cards and ensure we have at least 2 files
    const fileCards = mediaPage.getDUFileCards();
    const fileCount = await fileCards.count();
    console.log(`[DU Keyboard Test] Found ${fileCount} files in images folder`);
    
    expect(fileCount).toBeGreaterThanOrEqual(2);

    // Get the first two file paths
    const files = await mediaPage.getDUFiles();
    const firstName = files[0].split('/').pop() || '';
    const secondName = files[1].split('/').pop() || '';
    console.log(`[DU Keyboard Test] First: ${firstName}, Second: ${secondName}`);

    // Click first file to play it
    await fileCards.nth(0).click();
    await viewerPage.waitForPlayer();

    // Verify we're playing the first item
    await expect(viewerPage.mediaTitle).toContainText(firstName);

    // Press 'n' to go to next
    await mediaPage.page.keyboard.press('n');
    await mediaPage.page.waitForTimeout(500);

    // Verify we're now playing the second item (DU sibling)
    const titleAfterNext = await viewerPage.mediaTitle.textContent() || '';
    console.log(`[DU Keyboard Test] After next: ${titleAfterNext}`);
    await expect(viewerPage.mediaTitle).toContainText(secondName);

    // Press 'p' to go to previous
    await mediaPage.page.keyboard.press('p');
    await mediaPage.page.waitForTimeout(500);

    // Verify we're back to the first item
    const titleAfterPrev = await viewerPage.mediaTitle.textContent() || '';
    console.log(`[DU Keyboard Test] After prev: ${titleAfterPrev}`);
    await expect(viewerPage.mediaTitle).toContainText(firstName);
  });
});
