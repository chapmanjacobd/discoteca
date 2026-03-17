import { test, expect } from '../fixtures';
import * as path from 'path';

test.describe('Disk Usage Recursive Navigation', () => {
  test.use({ readOnly: true });

  // Calculate the absolute path to fixtures/media at runtime
  const fixturesMediaPath = path.resolve(__dirname, '../fixtures/media');

  test('auto-skips multiple levels of single folders', async ({ mediaPage, server }) => {
    // Navigate to the fixtures/media root
    await mediaPage.goto(server.getBaseUrl() + `/#mode=du&path=${encodeURIComponent(fixturesMediaPath)}`);
    await mediaPage.page.waitForTimeout(1000);

    // Find and click the 'recursive' folder ONCE
    // Structure: /recursive/level1/level2/deep_video.mp4
    // If auto-skip works, clicking 'recursive' should skip level1 and level2
    // and land directly at /recursive/level1/level2/ with the file visible
    const folderCards = mediaPage.getFolderCards();
    const folderCount = await folderCards.count();
    
    let foundRecursive = false;
    for (let i = 0; i < folderCount; i++) {
      const card = folderCards.nth(i);
      const title = await card.locator('.media-title').textContent();
      if (title && title.includes('recursive')) {
        await card.click();
        foundRecursive = true;
        break;
      }
    }
    
    expect(foundRecursive).toBe(true);
    
    // Wait for auto-navigation to complete
    await mediaPage.page.waitForTimeout(2000);

    // Get current path - should be at deepest level (level2)
    const pathInput = mediaPage.getDUPathInput();
    const currentPath = await pathInput.inputValue();
    console.log(`[DU Recursive Test] Current path: ${currentPath}`);
    
    // Verify we're at level2 (auto-skip should have skipped level1)
    expect(currentPath).toContain('level2');

    // Should have exactly 1 file at this level (deep_video.mp4)
    const fileCards = mediaPage.getDUFileCards();
    const fileCount = await fileCards.count();
    console.log(`[DU Recursive Test] File count: ${fileCount}`);
    expect(fileCount).toBe(1);
    
    // Should have NO folders at this level (we're at the deepest level)
    const finalFolderCards = mediaPage.getFolderCards();
    const finalFolderCount = await finalFolderCards.count();
    console.log(`[DU Recursive Test] Folder count: ${finalFolderCount}`);
    expect(finalFolderCount).toBe(0);
  });
});
