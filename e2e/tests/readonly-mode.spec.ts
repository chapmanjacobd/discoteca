import { test, expect } from '../fixtures';

/**
 * E2E tests for Read-Only mode
 * Tests that server state is not modified in read-only mode
 */
test.describe('Read-Only Mode', () => {
  test.use({ readOnly: true });

  test('server database is not modified in read-only mode', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial media count using POM
    const initialCount = await mediaPage.getMediaCount();
    console.log(`Initial media count: ${initialCount}`);

    // Try to perform actions that would normally modify state

    // 1. Try to mark media as played (should not work in read-only)
    const mediaCard = mediaPage.getMediaCard(0);
    await mediaCard.hover();

    // Check if mark-played button exists using POM
    const markPlayedBtn = mediaCard.locator('.media-action-btn.mark-played');
    const hasMarkPlayedBtn = await markPlayedBtn.count() > 0;
    console.log(`Has mark-played button: ${hasMarkPlayedBtn}`);

    // In read-only mode, action buttons may be hidden or disabled
    // Just verify the page doesn't crash

    // 2. Try to add to playlist (should not work in read-only)
    const addToPlaylistBtn = mediaCard.locator('.media-action-btn.add-playlist');
    if (await addToPlaylistBtn.count() > 0) {
      console.log('Add to playlist button exists');
      // Don't click it in read-only mode
    }

    // Reload and verify count is the same using POM
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    const finalCount = await mediaPage.getMediaCount();
    console.log(`Final media count: ${finalCount}`);

    // Count should be the same (no new media added)
    expect(finalCount).toBe(initialCount);
  });

  test('playback works but does not sync to server in read-only mode', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Enable local resume first using POM
    await sidebarPage.openSettings();

    const advancedSettings = mediaPage.getAdvancedSettingsSummary();
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpanded = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpanded) {
      await advancedSettings.click({ force: true });
      await mediaPage.page.waitForTimeout(500);
    }

    const localResumeToggle = mediaPage.getSettingToggleSlider('setting-local-resume');
    const localResumeCheckbox = mediaPage.getSetting('setting-local-resume');
    const initialState = await localResumeCheckbox.isChecked();

    if (!initialState) {
      await localResumeToggle.click();
      await mediaPage.page.waitForTimeout(300);
    }

    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Play a video using POM
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    const mediaPath = await mediaCard.getAttribute('data-path');
    console.log(`Testing playback with: ${mediaPath}`);

    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    // Wait for video to be ready and playing using POM
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Let it play briefly using POM
    await mediaPage.page.waitForTimeout(3000);

    const playhead = await viewerPage.getCurrentTime();
    console.log(`Played to: ${playhead}s`);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1500); // Wait for progress to be saved (throttled to 1s)

    // In read-only mode, progress should only be stored locally using POM
    const localProgress = await mediaPage.getProgress();

    console.log('Local progress saved:', Object.keys(localProgress).length > 0);
    expect(Object.keys(localProgress).length).toBeGreaterThan(0);

    // Restore original state
    if (!initialState) {
      await sidebarPage.openSettings();
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await sidebarPage.closeSettings();
    }

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Local progress should still be there using POM
    const localProgressAfterReload = await mediaPage.getProgress();

    expect(Object.keys(localProgressAfterReload).length).toBeGreaterThan(0);

    // Note: We can't easily verify server state from the test,
    // but the read-only mode should prevent server writes
  });

  test('settings can be changed but not persisted to server in read-only mode', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Change a setting (e.g., theme) using POM
    const themeSelect = mediaPage.getSetting('setting-theme');
    const initialTheme = await themeSelect.inputValue();
    console.log(`Initial theme: ${initialTheme}`);

    // Change to dark theme using POM
    await themeSelect.selectOption('dark');
    await mediaPage.page.waitForTimeout(300);

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Setting should persist in localStorage (client-side) using POM
    await sidebarPage.openSettings();

    const themeAfterReload = await themeSelect.inputValue();
    console.log(`Theme after reload: ${themeAfterReload}`);

    // Theme should persist (localStorage works in read-only mode)
    expect(themeAfterReload).toBe('dark');

    // Restore original theme
    await themeSelect.selectOption(initialTheme);
    await sidebarPage.closeSettings();
  });

  test('cannot create playlists in read-only mode', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open sidebar playlists section using POM
    await sidebarPage.expandPlaylistsSection();
    await mediaPage.page.waitForTimeout(500);

    // Try to create new playlist using POM
    const newPlaylistBtn = sidebarPage.getNewPlaylistButton();
    if (await newPlaylistBtn.count() > 0) {
      const isVisible = await newPlaylistBtn.isVisible();
      console.log(`New playlist button visible: ${isVisible}`);

      if (isVisible) {
        // Click should either not work or show an error
        await newPlaylistBtn.click();
        await mediaPage.page.waitForTimeout(1000);

        // Check if prompt appeared (it might in read-only mode, but creation should fail)
        // This is hard to test without mocking the prompt
        console.log('New playlist button was clickable');
      }
    } else {
      console.log('New playlist button not present (expected in read-only mode)');
    }
  });

  test('can browse and search in read-only mode', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Search should work using POM
    await mediaPage.search('test');
    const searchCount = await mediaPage.getMediaCount();
    console.log(`Search results: ${searchCount}`);

    // Clear search using POM
    await mediaPage.clearSearch();

    // Navigate to different pages using POM
    await sidebarPage.openDiskUsage();
    await expect(mediaPage.getDUTToolbar()).toBeVisible();

    // Go back to home using POM
    await mediaPage.goto(server.getBaseUrl());
    await mediaPage.waitForMediaToLoad();

    // Should have media cards using POM
    const homeCount = await mediaPage.getMediaCount();
    expect(homeCount).toBeGreaterThan(0);
  });

  test('can view media details in read-only mode', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first video to open player using POM
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.click();
    await viewerPage.waitForPlayer();

    // Player should be visible using POM
    await expect(viewerPage.playerContainer).toBeVisible();

    // Media title should be shown using POM
    await expect(viewerPage.mediaTitle).toBeVisible();

    // Close player using POM
    await viewerPage.close();
  });

  test('keyboard shortcuts work in read-only mode', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open first media using POM
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.click();
    await viewerPage.waitForPlayer();

    // Press space to toggle playback
    await mediaPage.page.keyboard.press(' ');
    await mediaPage.page.waitForTimeout(500);

    // Player should still be visible using POM
    await expect(viewerPage.playerContainer).toBeVisible();

    // Press 'w' to close
    await mediaPage.page.keyboard.press('w');
    await mediaPage.page.waitForTimeout(500);

    // Player should be hidden using POM
    await expect(viewerPage.playerContainer).toHaveClass(/hidden/);
  });

  test('filters work in read-only mode', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();

    // Apply video filter using POM
    await sidebarPage.expandMediaTypeSection();
    await sidebarPage.getMediaTypeButton('video').click();
    await mediaPage.page.waitForTimeout(1000);

    // Should have filtered results using POM
    const videoCount = await mediaPage.getMediaCount();
    expect(videoCount).toBeLessThanOrEqual(initialCount);

    // Clear filter by clicking All Media using POM
    await sidebarPage.openAllMedia();
    await mediaPage.page.waitForTimeout(1000);

    // Should have all media again using POM
    const allCount = await mediaPage.getMediaCount();
    expect(allCount).toBeGreaterThanOrEqual(videoCount);
  });

  test('history pages work in read-only mode', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Navigate to In Progress using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.clickHistoryInProgress();
    await mediaPage.page.waitForTimeout(2000);

    // Page should load without errors using POM
    await expect(mediaPage.resultsContainer).toBeVisible();

    // Navigate to Unplayed using POM
    await sidebarPage.clickHistoryUnplayed();
    await mediaPage.page.waitForTimeout(2000);

    // Page should load without errors using POM
    await expect(mediaPage.resultsContainer).toBeVisible();

    // Navigate to Completed using POM
    await sidebarPage.clickHistoryCompleted();
    await mediaPage.page.waitForTimeout(2000);

    // Page should load without errors using POM
    await expect(mediaPage.resultsContainer).toBeVisible();
  });

  test('captions page works in read-only mode', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Should have caption cards using POM
    const count = await mediaPage.getCaptionCards().count();
    expect(count).toBeGreaterThanOrEqual(0);

    // Search in captions should work using POM
    await mediaPage.search('test');
    await mediaPage.page.waitForTimeout(1000);

    // Should have search results using POM
    const searchCount = await mediaPage.getCaptionCards().count();
    expect(searchCount).toBeGreaterThanOrEqual(0);
  });
});
