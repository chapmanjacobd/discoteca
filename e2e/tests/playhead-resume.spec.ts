import { test, expect } from '../fixtures';
import { waitForPlayer, isPlayerOpen } from '../fixtures';

/**
 * E2E tests for playhead resume functionality
 * Tests both local resume mode and server-based progress tracking
 */
test.describe('Playhead Resume', () => {
  test.use({ readOnly: false });

  test.beforeEach(async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.error('BROWSER ERROR:', msg.text());
      }
    });
  });

  test('resumes from local progress when localResume is enabled', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Enable local resume in settings
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
    
    // Enable local resume if not already enabled
    if (!initialState) {
      await localResumeToggle.click();
      await page.waitForTimeout(300);
    }
    
    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Find and play first video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);

    // Wait for video to load and start playing
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    
    // Click to ensure video is playing
    await page.click('video');
    await page.waitForTimeout(500);
    
    // Let it play for 3 seconds
    await page.waitForTimeout(3000);

    // Get current playback position
    const playheadBefore = await page.evaluate(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video ? video.currentTime : 0;
    });
    console.log(`Playhead before close: ${playheadBefore}s`);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Verify progress was saved to localStorage
    const savedProgress = await page.evaluate(() => {
      const progress = localStorage.getItem('disco-progress');
      return progress ? JSON.parse(progress) : null;
    });
    console.log('Saved progress:', savedProgress);
    expect(savedProgress).toBeTruthy();

    // Play the same media again
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(1000);

    // Should resume from saved position (with some tolerance)
    const playheadAfter = await page.evaluate(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video ? video.currentTime : 0;
    });
    console.log(`Playhead after resume: ${playheadAfter}s`);
    
    // Should have resumed from approximately the same position
    // Allow some tolerance for network delay and playback buffering
    expect(playheadAfter).toBeGreaterThan(playheadBefore * 0.8);

    // Restore original state
    if (!initialState) {
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }
  });

  test('does not resume when localResume is disabled', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Disable local resume in settings
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
    
    // Disable local resume if enabled
    if (initialState) {
      await localResumeToggle.click();
      await page.waitForTimeout(300);
    }
    
    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Find and play first video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);

    // Wait for video to load and start playing
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    
    // Click to ensure video is playing
    await page.click('video');
    await page.waitForTimeout(500);
    await page.waitForTimeout(3000);

    const playheadBefore = await page.evaluate(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video ? video.currentTime : 0;
    });
    console.log(`Playhead before close: ${playheadBefore}s`);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Play the same media again
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(1000);

    // Should start from beginning (or very close to it)
    const playheadAfter = await page.evaluate(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video ? video.currentTime : 0;
    });
    console.log(`Playhead after resume: ${playheadAfter}s`);
    
    // Should have started from beginning (allow small tolerance for loading)
    expect(playheadAfter).toBeLessThan(5);

    // Restore original state
    if (initialState) {
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }
  });

  test('syncs progress to server in non-readonly mode', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find and play first video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    const path = await mediaCard.getAttribute('data-path');
    console.log(`Testing with media: ${path}`);
    
    await mediaCard.click();
    await waitForPlayer(page);

    // Wait for video to load and start playing
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    
    // Click to ensure video is playing
    await page.click('video');
    await page.waitForTimeout(500);
    await page.waitForTimeout(5000);

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(2000);

    // Progress should be synced to server
    // We can verify by reloading and checking if progress persists
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play the same media again
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(1000);

    const playhead = await page.evaluate(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video ? video.currentTime : 0;
    });
    console.log(`Playhead after reload: ${playhead}s`);

    // Should have some progress (not starting from 0)
    expect(playhead).toBeGreaterThan(0);
  });

  test('handles progress for multiple media items', async ({ page, server }) => {
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

    // Play first video
    const video1 = page.locator('.media-card[data-type*="video"]').nth(0);
    await video1.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    await page.click('video');
    await page.waitForTimeout(500);
    await page.waitForTimeout(2000);
    await page.click('.close-pip');
    await page.waitForTimeout(500);

    // Play second video
    const video2 = page.locator('.media-card[data-type*="video"]').nth(1);
    await video2.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    await page.click('video');
    await page.waitForTimeout(500);
    await page.waitForTimeout(2000);
    await page.click('.close-pip');
    await page.waitForTimeout(500);

    // Play third video
    const video3 = page.locator('.media-card[data-type*="video"]').nth(2);
    await video3.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    await page.click('video');
    await page.waitForTimeout(500);
    await page.waitForTimeout(2000);
    await page.click('.close-pip');
    await page.waitForTimeout(500);

    // Check localStorage has progress for all three
    const savedProgress = await page.evaluate(() => {
      const progress = localStorage.getItem('disco-progress');
      return progress ? JSON.parse(progress) : {};
    });

    console.log('Saved progress for multiple items:', Object.keys(savedProgress).length);
    expect(Object.keys(savedProgress).length).toBeGreaterThanOrEqual(3);

    // Restore original state
    if (!initialState) {
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }
  });

  test('clears progress when media is marked as complete', async ({ page, server }) => {
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

    // Play a short video or play until near the end
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);

    // Wait for video to load and start playing
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    await page.click('video');
    await page.waitForTimeout(500);

    // Get duration
    const duration = await page.evaluate(() => {
      const video = document.querySelector('video') as HTMLVideoElement;
      return video ? video.duration : 0;
    });
    console.log(`Video duration: ${duration}s`);
    
    // Seek to near the end (95%)
    if (duration > 10) {
      await page.evaluate((pos) => {
        const video = document.querySelector('video') as HTMLVideoElement;
        if (video) video.currentTime = pos;
      }, duration * 0.95);
      await page.waitForTimeout(2000);
    } else {
      // For short videos, just wait
      await page.waitForTimeout(5000);
    }

    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Progress should be cleared or marked as complete
    const savedProgress = await page.evaluate((path: string) => {
      const progress = localStorage.getItem('disco-progress');
      if (!progress) return null;
      const parsed = JSON.parse(progress);
      return parsed[path];
    }, await mediaCard.getAttribute('data-path'));

    console.log('Progress after completion:', savedProgress);
    // Progress should be cleared or very small for completed media
    expect(savedProgress === null || (savedProgress && savedProgress.pos < 10)).toBe(true);

    // Restore original state
    if (!initialState) {
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }
  });
});
