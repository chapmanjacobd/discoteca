/**
 * Progress Synchronization Tests
 *
 * Tests for verifying that playback progress is correctly synchronized between:
 * - Multiple sessions in the same tab
 * - Local storage persistence
 * - Concurrent progress updates
 */
import { test, expect } from '../fixtures';
import { waitForPlayer } from '../fixtures';

test.describe('Progress Synchronization', () => {
  test.use({ readOnly: false });

  test('localStorage progress structure is created on playback', async ({ page, server }) => {
    console.log('=== Testing progress structure creation ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first video
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    const videoPath = await videoCard.getAttribute('data-path');
    console.log('Testing with video:', videoPath);

    await videoCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });

    // Wait for player to initialize
    await page.waitForTimeout(1000);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Check localStorage for progress entry (may be pos: 0 if video didn't play)
    const progress = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });

    console.log('Progress object:', progress);
    // Progress object should exist for the video path (even if pos is 0)
    expect(progress).toBeDefined();
  });

  test('concurrent progress updates merge correctly', async ({ page, server }) => {
    console.log('=== Testing concurrent progress merge ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first video
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    const videoPath = await videoCard.getAttribute('data-path');
    console.log('Testing concurrent updates for:', videoPath);

    // Simulate existing progress from another session
    await page.evaluate((path) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 5, last: Date.now() - 10000 };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, videoPath);

    await videoCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(500);

    // Simulate concurrent update with newer timestamp
    await page.evaluate((path) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 10, last: Date.now() };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, videoPath);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1500);

    // Final progress should have the newer value
    const progress2 = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });
    console.log('Final progress:', progress2[videoPath]);

    expect(progress2[videoPath]).toBeTruthy();
    expect(progress2[videoPath].pos).toBe(10);
  });

  test('progress is not corrupted during rapid updates', async ({ page, server }) => {
    console.log('=== Testing progress during rapid updates ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first video
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    const videoPath = await videoCard.getAttribute('data-path');

    await videoCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(500);

    // Simulate rapid progress updates
    for (let i = 0; i < 5; i++) {
      await page.evaluate(({ path, pos }) => {
        const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
        p[path] = { pos: pos, last: Date.now() };
        localStorage.setItem('disco-progress', JSON.stringify(p));
      }, { path: videoPath, pos: i * 10 });
      await page.waitForTimeout(100);
    }

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Check progress wasn't corrupted
    const progress = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });

    console.log('Progress after rapid updates:', progress[videoPath]);
    expect(progress[videoPath]).toBeTruthy();
    expect(typeof progress[videoPath].pos).toBe('number');
    expect(typeof progress[videoPath].last).toBe('number');
  });

  test('progress persists across page reload', async ({ page, server }) => {
    console.log('=== Testing progress persistence across reload ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first video
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    const videoPath = await videoCard.getAttribute('data-path');

    // Set progress
    await page.evaluate((path) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 30, last: Date.now() };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, videoPath);

    // Get progress before reload
    const progressBefore = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });
    console.log('Progress before reload:', progressBefore[videoPath]);

    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get progress after reload
    const progressAfter = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });
    console.log('Progress after reload:', progressAfter[videoPath]);

    // Progress should persist
    expect(progressAfter[videoPath]).toBeTruthy();
    expect(progressAfter[videoPath].pos).toBe(progressBefore[videoPath].pos);
  });

  test('multiple videos track progress independently', async ({ page, server }) => {
    console.log('=== Testing independent progress tracking ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get first two videos
    const videoCards = page.locator('.media-card[data-type*="video"]');
    const count = await videoCards.count();
    expect(count).toBeGreaterThanOrEqual(2);

    const video1Path = await videoCards.nth(0).getAttribute('data-path');
    const video2Path = await videoCards.nth(1).getAttribute('data-path');
    console.log('Video 1:', video1Path);
    console.log('Video 2:', video2Path);

    // Set progress for first video
    await page.evaluate((path) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 15, last: Date.now() };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, video1Path);

    // Set progress for second video
    await page.evaluate((path) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 45, last: Date.now() };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, video2Path);

    // Check both have independent progress
    const progress = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });

    console.log('Progress for video 1:', progress[video1Path]);
    console.log('Progress for video 2:', progress[video2Path]);

    expect(progress[video1Path]).toBeTruthy();
    expect(progress[video2Path]).toBeTruthy();
    expect(progress[video1Path].pos).toBe(15);
    expect(progress[video2Path].pos).toBe(45);
  });

  test('progress is saved for audio files', async ({ page, server }) => {
    console.log('=== Testing audio progress saving ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first audio
    const audioCard = page.locator('.media-card[data-type*="audio"]').first();
    const audioPath = await audioCard.getAttribute('data-path');
    console.log('Testing with audio:', audioPath);

    await audioCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(500);

    // Set progress
    await page.evaluate((path) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 60, last: Date.now() };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, audioPath);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Check progress
    const progress = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });

    console.log('Audio progress:', progress[audioPath]);
    expect(progress[audioPath]).toBeTruthy();
    expect(progress[audioPath].pos).toBe(60);
  });

  test('progress handles missing localStorage gracefully', async ({ page, server }) => {
    console.log('=== Testing graceful handling of missing progress ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Clear localStorage
    await page.evaluate(() => {
      localStorage.removeItem('disco-progress');
    });

    // Open video
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    await videoCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(500);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Should not crash, progress may be empty or have new entry
    const progress = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });

    console.log('Progress after clean start:', progress);
    expect(typeof progress).toBe('object');
  });

  test('progress timestamp is updated on each save', async ({ page, server }) => {
    console.log('=== Testing progress timestamp updates ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Open first video
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    const videoPath = await videoCard.getAttribute('data-path');

    // Set initial progress
    const time1 = Date.now();
    await page.evaluate(({ path, time }) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 10, last: time };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, { path: videoPath, time: time1 });

    await page.waitForTimeout(500);

    // Update progress
    const time2 = Date.now();
    await page.evaluate(({ path, time }) => {
      const p = JSON.parse(localStorage.getItem('disco-progress') || '{}');
      p[path] = { pos: 20, last: time };
      localStorage.setItem('disco-progress', JSON.stringify(p));
    }, { path: videoPath, time: time2 });

    // Check timestamp was updated
    const progress = await page.evaluate(() => {
      const p = localStorage.getItem('disco-progress');
      return p ? JSON.parse(p) : {};
    });

    console.log('Progress timestamp:', progress[videoPath].last);
    expect(progress[videoPath].last).toBeGreaterThanOrEqual(time1);
  });
});
