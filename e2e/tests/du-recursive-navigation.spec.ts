import { test, expect } from '../fixtures';

test.describe('Disk Usage Recursive Navigation', () => {
  test.use({ readOnly: true });

  test('auto-skips multiple levels of single folders', async ({ mediaPage, server }) => {
    // Navigate to DU mode
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
    await mediaPage.getDUTToolbar().waitFor({ state: 'visible' });

    // Find the 'recursive' folder and click it
    // The structure is /recursive/level1/level2/deep_video.mp4
    // So clicking 'recursive' should skip 'level1' and 'level2' and go straight to 'recursive/level1/level2/'

    // Find folder card with text 'recursive'
    const folderCard = mediaPage.page.locator('.is-folder', { hasText: 'recursive' });
    await expect(folderCard).toBeVisible();

    await folderCard.click();
    
    // Wait for potential multiple navigations
    // We can't rely on a single timeout if there are multiple fetches, but 2s should be enough for local
    await mediaPage.page.waitForTimeout(2000);

    // Expect path to be '/recursive/level1/level2/'
    const pathInput = mediaPage.getDUPathInput();
    const currentPath = await pathInput.inputValue();

    // If bug exists, it will probably be '/recursive/level1/'
    expect(currentPath.endsWith('/recursive/level1/level2/')).toBe(true);
  });
});
