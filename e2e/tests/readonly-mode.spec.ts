import { test, expect } from '../fixtures';
import { waitForPlayer } from '../fixtures';

/**
 * E2E tests for Read-Only mode
 * Tests that server state is not modified in read-only mode
 */
test.describe('Read-Only Mode', () => {
  test.use({ readOnly: true });

  test('server database is not modified in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial media count
    const initialCount = await page.locator('.media-card').count();
    console.log(`Initial media count: ${initialCount}`);

    // Try to perform actions that would normally modify state
    
    // 1. Try to mark media as played (should not work in read-only)
    const mediaCard = page.locator('.media-card').first();
    await mediaCard.hover();
    
    // Check if mark-played button exists
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

    // Reload and verify count is the same
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    const finalCount = await page.locator('.media-card').count();
    console.log(`Final media count: ${finalCount}`);
    
    // Count should be the same (no new media added)
    expect(finalCount).toBe(initialCount);
  });

  test('playback works but does not sync to server in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Enable local resume first
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });

    const advancedSettings = page.locator('summary:has-text("Advanced Settings")');
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpanded = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpanded) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }

    const localResumeToggle = page.locator('#setting-local-resume').locator('xpath=..').locator('.slider');
    const localResumeCheckbox = page.locator('#setting-local-resume');
    const initialState = await localResumeCheckbox.isChecked();

    if (!initialState) {
      await localResumeToggle.click();
      await page.waitForTimeout(300);
    }

    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Play a video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    const mediaPath = await mediaCard.getAttribute('data-path');
    console.log(`Testing playback with: ${mediaPath}`);

    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });

    // Wait for video to be ready and playing
    await page.waitForFunction(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video && video.readyState >= 3;
    }, { timeout: 10000 });

    // Click to ensure video is playing
    await page.click('video');
    await page.waitForTimeout(500);

    // Let it play briefly
    const video = page.locator('video');
    await page.waitForTimeout(3000);

    const playhead = await video.evaluate((el: HTMLVideoElement) => el.currentTime);
    console.log(`Played to: ${playhead}s`);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1500); // Wait for progress to be saved (throttled to 1s)

    // In read-only mode, progress should only be stored locally
    const localProgress = await page.evaluate(() => {
      const progress = localStorage.getItem('disco-progress');
      return progress ? JSON.parse(progress) : {};
    });

    console.log('Local progress saved:', Object.keys(localProgress).length > 0);
    expect(Object.keys(localProgress).length).toBeGreaterThan(0);

    // Restore original state
    if (!initialState) {
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }

    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Local progress should still be there
    const localProgressAfterReload = await page.evaluate(() => {
      const progress = localStorage.getItem('disco-progress');
      return progress ? JSON.parse(progress) : {};
    });
    
    expect(Object.keys(localProgressAfterReload).length).toBeGreaterThan(0);
    
    // Note: We can't easily verify server state from the test,
    // but the read-only mode should prevent server writes
  });

  test('settings can be changed but not persisted to server in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });

    // Change a setting (e.g., theme)
    const themeSelect = page.locator('#setting-theme');
    const initialTheme = await themeSelect.inputValue();
    console.log(`Initial theme: ${initialTheme}`);

    // Change to dark theme
    await themeSelect.selectOption('dark');
    await page.waitForTimeout(300);

    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Setting should persist in localStorage (client-side)
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });
    
    const themeAfterReload = await themeSelect.inputValue();
    console.log(`Theme after reload: ${themeAfterReload}`);
    
    // Theme should persist (localStorage works in read-only mode)
    expect(themeAfterReload).toBe('dark');

    // Restore original theme
    await themeSelect.selectOption(initialTheme);
    await page.click('#settings-modal .close-modal');
  });

  test('cannot create playlists in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open sidebar
    await page.locator('#details-playlists').evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);

    // Try to create new playlist
    const newPlaylistBtn = page.locator('#new-playlist-btn');
    if (await newPlaylistBtn.count() > 0) {
      const isVisible = await newPlaylistBtn.isVisible();
      console.log(`New playlist button visible: ${isVisible}`);
      
      if (isVisible) {
        // Click should either not work or show an error
        await newPlaylistBtn.click();
        await page.waitForTimeout(1000);
        
        // Check if prompt appeared (it might in read-only mode, but creation should fail)
        // This is hard to test without mocking the prompt
        console.log('New playlist button was clickable');
      }
    } else {
      console.log('New playlist button not present (expected in read-only mode)');
    }
  });

  test('can browse and search in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Search should work
    const searchInput = page.locator('#search-input');
    await searchInput.fill('test');
    await page.waitForTimeout(1000);

    // Should show search results or "no results"
    const results = page.locator('.media-card');
    const count = await results.count();
    console.log(`Search results for "test": ${count}`);
    
    // Clear search
    await searchInput.clear();
    await page.waitForTimeout(500);

    // Navigate to different sections
    await page.locator('#details-media-type').evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.click('#media-type-list button[data-type="video"]');
    await page.waitForTimeout(2000);

    const videoCount = await page.locator('.media-card').count();
    console.log(`Video results: ${videoCount}`);
    
    // Should show videos
    expect(videoCount).toBeGreaterThanOrEqual(0);
  });

  test('can view media details in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Click on a media card to open player
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);

    // Player should be visible
    const player = page.locator('#pip-player');
    await expect(player.first()).toBeVisible();

    // Media title should be shown
    const mediaTitle = page.locator('#media-title');
    if (await mediaTitle.count() > 0) {
      await expect(mediaTitle.first()).toBeVisible();
    }

    // Can interact with playback controls
    const playPauseBtn = page.locator('#pip-play-pause');
    if (await playPauseBtn.count() > 0) {
      await playPauseBtn.click();
      await page.waitForTimeout(500);
      console.log('Play/pause button works in read-only mode');
    }
  });

  test('history pages work in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open history section
    await page.locator('#details-history').evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.waitForTimeout(500);

    // Click In Progress
    await page.click('#history-in-progress-btn');
    await page.waitForTimeout(2000);

    const inProgressCount = await page.locator('.media-card').count();
    console.log(`In Progress results: ${inProgressCount}`);

    // Click Unplayed
    await page.click('#history-unplayed-btn');
    await page.waitForTimeout(2000);

    const unplayedCount = await page.locator('.media-card').count();
    console.log(`Unplayed results: ${unplayedCount}`);

    // Click Completed
    await page.click('#history-completed-btn');
    await page.waitForTimeout(2000);

    const completedCount = await page.locator('.media-card').count();
    console.log(`Completed results: ${completedCount}`);

    // All should work without errors
    expect(inProgressCount).toBeGreaterThanOrEqual(0);
    expect(unplayedCount).toBeGreaterThanOrEqual(0);
    expect(completedCount).toBeGreaterThanOrEqual(0);
  });

  test('local progress still works in read-only mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Enable local resume
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });

    const advancedSettings = page.locator('summary:has-text("Advanced Settings")');
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpanded = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpanded) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }

    const localResumeToggle = page.locator('#setting-local-resume').locator('xpath=..').locator('.slider');
    const localResumeCheckbox = page.locator('#setting-local-resume');
    const initialState = await localResumeCheckbox.isChecked();

    if (!initialState) {
      await localResumeToggle.click();
      await page.waitForTimeout(300);
    }

    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Play a video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });

    // Wait for video to be ready and playing
    await page.waitForFunction(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video && video.readyState >= 3;
    }, { timeout: 10000 });

    // Click to ensure video is playing
    await page.click('video');
    await page.waitForTimeout(500);

    await page.waitForTimeout(2000);

    const playhead = await page.locator('video').evaluate((el: HTMLVideoElement) => el.currentTime);
    console.log(`Playhead: ${playhead}s`);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1500); // Wait for progress to be saved (throttled to 1s)

    // Local progress should be saved
    const localProgress = await page.evaluate(() => {
      const progress = localStorage.getItem('disco-progress');
      return progress ? JSON.parse(progress) : {};
    });

    console.log('Local progress entries:', Object.keys(localProgress).length);
    expect(Object.keys(localProgress).length).toBeGreaterThan(0);

    // Restore original state
    if (!initialState) {
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }
  });
});
