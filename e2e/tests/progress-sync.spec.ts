/**
 * Progress Synchronization Tests
 *
 * Tests for verifying that playback progress is correctly synchronized between:
 * - Multiple sessions in the same tab
 * - Local storage persistence
 * - Concurrent progress updates
 */
import { test, expect } from '../fixtures';

test.describe('Progress Synchronization', () => {
  test.use({ readOnly: false });

  test('localStorage progress structure is created on playback', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing progress structure creation ===');

    await mediaPage.goto(server.getBaseUrl());

    // Open first video using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    const videoPath = await videoCard.getAttribute('data-path');
    console.log('Testing with video:', videoPath);

    await videoCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    // Wait for player to initialize
    await mediaPage.page.waitForTimeout(1000);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Check localStorage for progress entry using POM
    const progress = await mediaPage.getProgress();

    console.log('Progress object:', progress);
    // Progress object should exist for the video path (even if pos is 0)
    expect(progress).toBeDefined();
  });

  test('concurrent progress updates merge correctly', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing concurrent progress merge ===');

    await mediaPage.goto(server.getBaseUrl());

    // Open first video using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    const videoPath = await videoCard.getAttribute('data-path');
    console.log('Testing concurrent updates for:', videoPath);

    // Simulate existing progress from another session using POM
    await mediaPage.setProgress(videoPath, 5, Date.now() - 10000);

    await videoCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(500);

    // Simulate concurrent update with newer timestamp using POM
    await mediaPage.setProgress(videoPath, 10, Date.now());

    // Wait for player to sync with the concurrent update
    await mediaPage.page.waitForTimeout(2000);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1500);

    // Final progress should have the newer value using POM
    const progress = await mediaPage.getProgress();
    console.log('Final progress:', progress[videoPath]);

    expect(progress[videoPath]).toBeTruthy();
    expect(progress[videoPath].pos).toBe(10);
  });

  test('progress is not corrupted during rapid updates', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing progress during rapid updates ===');

    await mediaPage.goto(server.getBaseUrl());

    // Open first video using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    const videoPath = await videoCard.getAttribute('data-path');

    await videoCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(500);

    // Simulate rapid progress updates using POM
    for (let i = 0; i < 5; i++) {
      await mediaPage.setProgress(videoPath, i * 10, Date.now());
      await mediaPage.page.waitForTimeout(100);
    }

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Check progress wasn't corrupted using POM
    const progress = await mediaPage.getProgress();

    console.log('Progress after rapid updates:', progress[videoPath]);
    expect(progress[videoPath]).toBeTruthy();
    expect(typeof progress[videoPath].pos).toBe('number');
    expect(typeof progress[videoPath].last).toBe('number');
  });

  test('progress persists across page reload', async ({ mediaPage, viewerPage, server, page }) => {
    console.log('=== Testing progress persistence across reload ===');

    await mediaPage.goto(server.getBaseUrl());

    // Open first video using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    const videoPath = await videoCard.getAttribute('data-path');

    // Set progress using POM
    await mediaPage.setProgress(videoPath, 30, Date.now());

    // Get progress before reload using POM
    const progressBefore = await mediaPage.getProgress();
    console.log('Progress before reload:', progressBefore[videoPath]);

    // Reload page
    await page.reload();
    await mediaPage.waitForMediaToLoad();

    // Get progress after reload using POM
    const progressAfter = await mediaPage.getProgress();
    console.log('Progress after reload:', progressAfter[videoPath]);

    // Progress should persist
    expect(progressAfter[videoPath]).toBeTruthy();
    expect(progressAfter[videoPath].pos).toBe(progressBefore[videoPath].pos);
  });

  test('multiple videos track progress independently', async ({ mediaPage, server }) => {
    console.log('=== Testing independent progress tracking ===');

    await mediaPage.goto(server.getBaseUrl());

    // Get first two videos using POM
    const videoCards = mediaPage.page.locator('.media-card[data-type*="video"]');
    const count = await videoCards.count();
    expect(count).toBeGreaterThanOrEqual(2);

    const video1Path = await videoCards.nth(0).getAttribute('data-path');
    const video2Path = await videoCards.nth(1).getAttribute('data-path');
    console.log('Video 1:', video1Path);
    console.log('Video 2:', video2Path);

    // Set progress for first video using POM
    await mediaPage.setProgress(video1Path, 15, Date.now());

    // Set progress for second video using POM
    await mediaPage.setProgress(video2Path, 45, Date.now());

    // Check both have independent progress using POM
    const progress = await mediaPage.getProgress();

    console.log('Progress for video 1:', progress[video1Path]);
    console.log('Progress for video 2:', progress[video2Path]);

    expect(progress[video1Path]).toBeTruthy();
    expect(progress[video2Path]).toBeTruthy();
    expect(progress[video1Path].pos).toBe(15);
    expect(progress[video2Path].pos).toBe(45);
  });

  test('progress is saved for audio files', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing audio progress saving ===');

    await mediaPage.goto(server.getBaseUrl());

    // Open first audio using POM
    const audioCard = mediaPage.getFirstMediaCardByType('audio');
    const audioPath = await audioCard.getAttribute('data-path');
    console.log('Testing with audio:', audioPath);

    await audioCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(500);

    // Set progress using POM
    await mediaPage.setProgress(audioPath, 60, Date.now());

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Check progress using POM
    const progress = await mediaPage.getProgress();

    console.log('Audio progress:', progress[audioPath]);
    expect(progress[audioPath]).toBeTruthy();
    expect(progress[audioPath].pos).toBe(60);
  });

  test('progress handles missing localStorage gracefully', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing graceful handling of missing progress ===');

    await mediaPage.goto(server.getBaseUrl());

    // Clear localStorage using POM
    await mediaPage.removeLocalStorageItem('disco-progress');

    // Open video using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    await videoCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(500);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Should not crash, progress may be empty or have new entry using POM
    const progress = await mediaPage.getProgress();

    console.log('Progress after clean start:', progress);
    expect(typeof progress).toBe('object');
  });

  test('progress timestamp is updated on each save', async ({ mediaPage, server }) => {
    console.log('=== Testing progress timestamp updates ===');

    await mediaPage.goto(server.getBaseUrl());

    // Open first video using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    const videoPath = await videoCard.getAttribute('data-path');

    // Set initial progress using POM
    const time1 = Date.now();
    await mediaPage.setProgress(videoPath, 10, time1);

    await mediaPage.page.waitForTimeout(500);

    // Update progress using POM
    const time2 = Date.now();
    await mediaPage.setProgress(videoPath, 20, time2);

    // Check timestamp was updated using POM
    const progress = await mediaPage.getProgress();

    console.log('Progress timestamp:', progress[videoPath].last);
    expect(progress[videoPath].last).toBeGreaterThanOrEqual(time1);
  });
});
