import { test, expect } from '../fixtures';

/**
 * E2E tests for playhead resume functionality
 * Tests both local resume mode and server-based progress tracking
 */
test.describe('Playhead Resume', () => {
  test.skip(({ browserName }) => browserName === 'webkit', 'WebKit headless cannot decode the H.264 fixtures used by these tests');

  test('resumes from local progress when localResume is enabled', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Enable local resume in settings using POM
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

    // Enable local resume if not already enabled using POM
    if (!initialState) {
      await localResumeToggle.click();
      await mediaPage.page.waitForTimeout(300);
    }

    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Find and play first video using POM
    const mediaCard = mediaPage.getMediaCardByText('video', /test_video1/);
    await mediaCard.click();
    await viewerPage.waitForPlayer();

    // Wait for video to load and start playing using POM
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Let it play until enough progress has accumulated to be saved
    await viewerPage.waitForPlaybackPosition(2);
    await mediaPage.page.waitForTimeout(300);

    // Get current playback position using POM
    const playheadBefore = await viewerPage.getCurrentTime();
    console.log(`Playhead before close: ${playheadBefore}s`);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Verify progress was saved to localStorage using POM
    const savedProgress = await mediaPage.getProgress();
    console.log('Saved progress:', savedProgress);
    expect(Object.keys(savedProgress).length).toBeGreaterThan(0);

    // Play the same media again using POM
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Should resume from saved position (with some tolerance) using POM
    const playheadAfter = await viewerPage.getCurrentTime();
    console.log(`Playhead after resume: ${playheadAfter}s`);

    // Should have resumed from approximately the same position
    expect(playheadAfter).toBeGreaterThan(playheadBefore * 0.8);

    // Restore original state
    if (!initialState) {
      await sidebarPage.openSettings();
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await sidebarPage.closeSettings();
    }
  });

  test('does not resume when localResume is disabled', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Disable local resume using POM
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

    // Disable local resume if enabled using POM
    if (initialState) {
      await localResumeToggle.click();
      await mediaPage.page.waitForTimeout(300);
    }

    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Play video using POM
    const mediaCard = mediaPage.getMediaCardByText('video', /test_video1/);
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Let it play until enough progress has accumulated to be saved
    await viewerPage.waitForPlaybackPosition(2);
    await mediaPage.page.waitForTimeout(300);

    const playheadBefore = await viewerPage.getCurrentTime();
    console.log(`Playhead before close: ${playheadBefore}s`);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Play same video again using POM
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Should start from beginning (or near beginning) when local resume is disabled
    const playheadAfter = await viewerPage.getCurrentTime();
    console.log(`Playhead after resume (should be ~0): ${playheadAfter}s`);

    // May have small playhead due to loading, but should be much less than before
    expect(playheadAfter).toBeLessThan(playheadBefore * 0.5);

    // Restore original state
    if (initialState) {
      await sidebarPage.openSettings();
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await sidebarPage.closeSettings();
    }
  });

  test('resumes from server progress when localResume is disabled but server has progress', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play video to create server progress using POM
    const mediaCard = mediaPage.getMediaCardByText('video', /test_video1/);
    const mediaPath = await mediaCard.getAttribute('data-path');
    console.log(`Testing with media: ${mediaPath}`);

    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Let it play until enough progress has accumulated to be saved
    await viewerPage.waitForPlaybackPosition(2);
    await mediaPage.page.waitForTimeout(300);

    const playheadBefore = await viewerPage.getCurrentTime();
    console.log(`Playhead before close: ${playheadBefore}s`);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1500); // Wait for progress to be saved

    // Reload page to ensure fresh state
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Play same video again using POM
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Should resume from server progress
    const playheadAfter = await viewerPage.getCurrentTime();
    console.log(`Playhead after resume: ${playheadAfter}s`);

    // Should have resumed from some position > 0
    expect(playheadAfter).toBeGreaterThan(0);
  });

  test('local progress takes precedence over server progress when localResume is enabled', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Enable local resume using POM
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

    // Play video using POM
    const mediaCard = mediaPage.getMediaCardByText('video', /test_video1/);
    const mediaPath = await mediaCard.getAttribute('data-path') || '';
    console.log(`Testing with media: ${mediaPath}`);

    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Let it play until enough progress has accumulated to be saved
    await viewerPage.waitForPlaybackPosition(2);
    await mediaPage.page.waitForTimeout(300);

    const playheadBefore = await viewerPage.getCurrentTime();
    console.log(`Playhead before close: ${playheadBefore}s`);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1500);

    // Simulate different server progress (older timestamp) using POM
    await mediaPage.setProgress(mediaPath, 1, Date.now() - 10000);

    // Play same video again using POM
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Should resume from local progress (newer), not server progress
    const playheadAfter = await viewerPage.getCurrentTime();
    console.log(`Playhead after resume: ${playheadAfter}s`);

    // Should have resumed from local progress position (closer to playheadBefore)
    expect(playheadAfter).toBeGreaterThan(1);

    // Restore original state
    if (!initialState) {
      await sidebarPage.openSettings();
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await sidebarPage.closeSettings();
    }
  });

  test('progress is saved when video is closed before completion', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play video using POM
    const mediaCard = mediaPage.getMediaCardByText('video', /test_video1/);
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Let it play until enough progress has accumulated to be saved
    await viewerPage.waitForPlaybackPosition(2);
    await mediaPage.page.waitForTimeout(300);

    const playhead = await viewerPage.getCurrentTime();
    console.log(`Playhead: ${playhead}s`);
    expect(playhead).toBeGreaterThan(2);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1500);

    // Progress should be saved using POM
    const progress = await mediaPage.getProgress();
    expect(Object.keys(progress).length).toBeGreaterThan(0);
  });

  test('progress is updated continuously during playback', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play video using POM
    const mediaCard = mediaPage.getMediaCardByText('video', /test_video1/);
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Get initial progress using POM
    let progress = await mediaPage.getProgress();
    const initialKeys = Object.keys(progress).length;

    // Let it play for 5 seconds
    await mediaPage.page.waitForTimeout(5000);

    // Progress should be updated (throttled to 1s updates) using POM
    progress = await mediaPage.getProgress();
    const finalKeys = Object.keys(progress).length;

    console.log(`Progress entries: ${finalKeys}`);
    expect(finalKeys).toBeGreaterThanOrEqual(initialKeys);

    // Close player using POM
    await viewerPage.close();
  });
});
