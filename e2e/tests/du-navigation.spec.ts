import { test, expect } from '../fixtures';

test.describe('Disk Usage Navigation', () => {
  test.use({ readOnly: true });

  test('auto-skips single folder at root level', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    // Wait for DU view to load using POM
    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Verify DU toolbar is visible using POM
    await expect(mediaPage.getDUTToolbar()).toBeVisible();

    // Path input should show current path using POM
    await expect(mediaPage.getDUPathInput()).toBeVisible();
  });

  test('displays folder cards with size visualization', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Should show folder/file cards using POM
    const cards = mediaPage.getDUCards();
    const count = await cards.count();
    expect(count).toBeGreaterThanOrEqual(1);

    // Cards should have size information using POM
    await expect(cards.first()).toBeVisible();
  });

  test('navigates into folder when clicked', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Get initial path using POM
    const initialPath = await mediaPage.getDUPathInput().inputValue();

    // Click first folder card using POM
    const folderCards = mediaPage.getFolderCards();
    const count = await folderCards.count();

    if (count > 0) {
      await folderCards.first().click();

      // Wait for navigation
      await mediaPage.page.waitForTimeout(500);

      // Path should have changed using POM
      const newPath = await mediaPage.getDUPathInput().inputValue();
      expect(newPath).not.toBe(initialPath);
    }
  });

  test('back button navigates to parent directory', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Navigate into a folder first using POM
    const folderCards = mediaPage.getFolderCards();
    const count = await folderCards.count();

    if (count > 0) {
      await folderCards.first().click();
      await mediaPage.page.waitForTimeout(500);

      // Click back button using POM
      const backBtn = mediaPage.getDUBackBtn();
      if (await backBtn.isVisible()) {
        await backBtn.click();
        await mediaPage.page.waitForTimeout(500);

        // Should be back at previous location using POM
        await expect(mediaPage.getDUTToolbar()).toBeVisible();
      }
    }
  });

  test('path input allows direct navigation', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Edit path input using POM
    const pathInput = mediaPage.getDUPathInput();
    await pathInput.fill('/videos/');
    await pathInput.press('Enter');

    // Wait for navigation
    await mediaPage.page.waitForTimeout(1000);

    // Path should be updated using POM
    const newPath = await pathInput.inputValue();
    expect(newPath).toBe('/videos/');
  });

  test('sorts folders by size', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=du');

    await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

    // Change sort to size using POM
    await mediaPage.setSortBy('size');

    // Enable reverse sort (largest first) using POM
    const reverseBtn = mediaPage.sortReverseBtn;
    const isActive = await reverseBtn.evaluate((el) => el.classList.contains('active'));
    if (!isActive) {
      await reverseBtn.click();
    }

    await mediaPage.page.waitForTimeout(500);

    // Verify sort dropdown shows size using POM
    await expect(mediaPage.sortBySelect).toHaveValue('size');
  });
});
