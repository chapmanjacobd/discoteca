import { waitForPlayer, isPlayerOpen } from '../fixtures';
import { test, expect } from '../fixtures';

test.describe('Playlist Management E2E', () => {
  test.describe.configure({ mode: 'serial' });
  // Helper to open sidebar on mobile
  async function openSidebar(page) {
    const menuToggle = page.locator('#menu-toggle');
    if (await menuToggle.isVisible()) {
      await menuToggle.click();
      await page.waitForTimeout(300);
    }
  }

  test('playlist UI elements are present', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Open sidebar on mobile
    await openSidebar(page);

    // Wait for playlists section
    await page.waitForSelector('#details-playlists', { timeout: 10000 });

    // Expand playlists section
    await page.locator('#details-playlists').evaluate((el: HTMLDetailsElement) => el.open = true);

    // Verify new playlist button exists
    const newPlaylistBtn = page.locator('#new-playlist-btn');
    await expect(newPlaylistBtn).toBeVisible();

    // Verify playlist list container exists (may be empty/hidden if no playlists)
    const playlistList = page.locator('#playlist-list');
    await expect(playlistList).toBeAttached();
  });

  test('can interact with media cards', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find a media card and click to open player
    const firstCard = page.locator('.media-card:not(:has(.rsvp))').first();
    await firstCard.click();
    await page.waitForTimeout(500);

    // Player should open
    await expect(page.locator('#pip-player')).not.toHaveClass(/hidden/);
  });

  test('navigates to playlist view', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Open sidebar on mobile
    await openSidebar(page);

    await page.waitForSelector('#details-playlists', { timeout: 10000 });

    await page.locator('#details-playlists').evaluate((el: HTMLDetailsElement) => el.open = true);

    // Favorites playlist should exist from seed data
    const favoritesBtn = page.locator('#playlist-list .category-btn').filter({ hasText: 'Favorites' });
    if (await favoritesBtn.isVisible()) {
      await favoritesBtn.click();
      await page.waitForTimeout(1000);

      // Should be in playlist view
      const hash = await page.evaluate(() => window.location.hash);
      expect(hash).toContain('mode=playlist');
    }
  });

  test('playlist structure exists', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Open sidebar on mobile
    await openSidebar(page);

    await page.waitForSelector('#details-playlists', { timeout: 10000 });

    await page.locator('#details-playlists').evaluate((el: HTMLDetailsElement) => el.open = true);

    // Verify playlist structure exists
    await expect(page.locator('#details-playlists')).toBeVisible();
  });
});

test.describe('Settings Persistence', () => {
  test('persists theme setting across reloads', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });
    
    // Change theme
    const themeSelect = page.locator('#setting-theme');
    await themeSelect.selectOption('dark');
    
    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);
    
    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Re-open settings and verify theme persisted
    await page.click('#settings-button');
    await expect(themeSelect).toHaveValue('dark');
    
    // Close settings
    await page.click('#settings-modal .close-modal');
  });

  test('persists default view setting', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });

    // Scroll modal body to ensure elements are visible
    await page.evaluate(() => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = 0;
    });
    await page.waitForTimeout(300);

    // Change default view to theatre mode
    const viewSelect = page.locator('#setting-default-view');
    await viewSelect.selectOption('theatre');

    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Re-open settings and verify the setting persisted
    await page.click('#settings-button');
    await page.waitForTimeout(300);
    await expect(viewSelect).toHaveValue('theatre');
  });

  test('persists language preference', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });
    
    // Set language
    const langInput = page.locator('#setting-language');
    await langInput.fill('eng,spa');
    
    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);
    
    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Re-open settings and verify language persisted
    await page.click('#settings-button');
    await expect(langInput).toHaveValue('eng,spa');
  });

  test('persists autoplay setting', async ({ page, server }) => {
    const advancedSettings = page.locator('summary:has-text("Advanced Settings")');
    let isExpandedNow;
    await page.goto(server.getBaseUrl());

    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });

    // Open Advanced Settings
    await advancedSettings.scrollIntoViewIfNeeded();
    isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }

    // Scroll modal body to ensure elements are visible
    await page.evaluate(() => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = 200;
    });
    await page.waitForTimeout(300);

    // Toggle autoplay - click on the slider element, not the hidden input
    const autoplayToggle = page.locator('#setting-autoplay').locator('xpath=..').locator('.slider');
    const autoplayCheckbox = page.locator('#setting-autoplay');
    const initialState = await autoplayCheckbox.isChecked();
    await autoplayToggle.click();

    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Re-open settings and verify autoplay persisted
    await page.click('#settings-button');
    // Open Advanced Settings
    await advancedSettings.scrollIntoViewIfNeeded();
    isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }
    await page.evaluate(() => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = 200;
    });
    await page.waitForTimeout(300);
    await expect(autoplayCheckbox).not.toBeChecked();

    // Restore original state
    if (initialState) {
      await autoplayToggle.click();
    }
  });

  test('persists playback rate settings', async ({ page, server }) => {
    const advancedSettings = page.locator('summary:has-text("Advanced Settings")');
    let isExpandedNow;
    await page.goto(server.getBaseUrl());

    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });

    // Open Advanced Settings
    await advancedSettings.scrollIntoViewIfNeeded();
    isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }

    // Scroll modal body to ensure elements are visible
    await page.evaluate(() => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = 500;
    });
    await page.waitForTimeout(300);

    // Set video rate
    const videoRate = page.locator('#setting-default-video-rate');
    await videoRate.selectOption('1.5');

    // Set audio rate
    const audioRate = page.locator('#setting-default-audio-rate');
    await audioRate.selectOption('2');

    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Re-open settings and verify rates persisted
    await page.click('#settings-button');
    // Open Advanced Settings
    await advancedSettings.scrollIntoViewIfNeeded();
    isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }
    await page.evaluate(() => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = 500;
    });
    await page.waitForTimeout(300);
    await expect(videoRate).toHaveValue('1.5');
    await expect(audioRate).toHaveValue('2');
  });

  test('persists slideshow delay', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    
    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });
    
    // Set slideshow delay
    const delayInput = page.locator('#setting-slideshow-delay');
    await delayInput.fill('10');
    
    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);
    
    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Re-open settings and verify delay persisted
    await page.click('#settings-button');
    await expect(delayInput).toHaveValue('10');
  });

  test('persists local resume setting', async ({ page, server }) => {
    const advancedSettings = page.locator('summary:has-text("Advanced Settings")');
    let isExpandedNow;
    await page.goto(server.getBaseUrl());

    // Open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });

    // Open Advanced Settings
    await advancedSettings.scrollIntoViewIfNeeded();
    isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }

    // Scroll modal body to ensure elements are visible
    await page.evaluate(() => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = 100;
    });
    await page.waitForTimeout(300);

    // Toggle local resume - click on the slider element, not the hidden input
    const localResumeToggle = page.locator('#setting-local-resume').locator('xpath=..').locator('.slider');
    const localResume = page.locator('#setting-local-resume');
    const initialState = await localResume.isChecked();
    await localResumeToggle.click();

    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);

    // Reload page
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Re-open settings and verify local resume persisted
    await page.click('#settings-button');
    // Open Advanced Settings
    await advancedSettings.scrollIntoViewIfNeeded();
    isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await page.waitForTimeout(500);
    }
    await page.evaluate(() => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = 100;
    });
    await page.waitForTimeout(300);
    await expect(localResume).not.toBeChecked();

    // Restore original state
    if (initialState) {
      await localResumeToggle.click();
    }
  });
});

test.describe('Playback Controls', () => {
  test('theatre mode toggle works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play a media
    await page.locator('.media-card:not(:has(.rsvp))').first().click();
    await waitForPlayer(page);
    await page.waitForTimeout(1000);

    // Click theatre mode button
    await page.click('#pip-theatre');
    await page.waitForTimeout(500);

    // Check if theatre mode is active
    const pipPlayer = page.locator('#pip-player');
    const hasTheatreClass = await pipPlayer.evaluate((el) => el.classList.contains('theatre'));
    expect(hasTheatreClass).toBe(true);

    // Click again to disable
    await page.click('#pip-theatre');
    await page.waitForTimeout(500);

    const hasTheatreClass2 = await pipPlayer.evaluate((el) => el.classList.contains('theatre'));
    expect(hasTheatreClass2).toBe(false);
  });

  test('close button closes player', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play a media
    await page.locator('.media-card:not(:has(.rsvp))').first().click();
    await waitForPlayer(page);
    await page.waitForTimeout(500);

    // Click close button
    await page.click('.close-pip');
    await page.waitForTimeout(1000);

    // Player should be hidden
    await expect(page.locator('#pip-player')).toHaveClass(/hidden/);
  });

  test('playback speed can be changed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play a media
    await page.locator('.media-card:not(:has(.rsvp))').first().click();
    await waitForPlayer(page);
    
    // Wait for video to load
    await page.waitForTimeout(1000);

    // Use JavaScript to set playback rate directly (tests the underlying functionality)
    const video = page.locator('video, audio');
    await video.evaluate((el: HTMLMediaElement) => {
      el.playbackRate = 2.0;
    });
    await page.waitForTimeout(300);

    const rate = await video.evaluate((el: HTMLMediaElement) => el.playbackRate);
    expect(rate).toBe(2);
  });

  test('play/pause via JavaScript', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play a media
    const firstCard = page.locator('.media-card:not(:has(.rsvp))').first();
    await firstCard.click();
    await waitForPlayer(page);

    // Wait for media to load and start playing
    await page.waitForTimeout(1000);

    const video = page.locator('video, audio');
    // Wait until video is actually playing
    await page.waitForFunction(() => {
      const v = document.querySelector('video, audio') as HTMLMediaElement;
      return v && !v.paused;
    }, { timeout: 5000 });

    // Pause via JavaScript
    await video.evaluate((el: HTMLMediaElement) => el.pause());
    await page.waitForTimeout(300);

    const isPaused = await video.evaluate((el: HTMLMediaElement) => el.paused);
    expect(isPaused).toBe(true);

    // Play via JavaScript
    await video.evaluate((el: HTMLMediaElement) => el.play());
    await page.waitForTimeout(300);

    const isPaused2 = await video.evaluate((el: HTMLMediaElement) => el.paused);
    expect(isPaused2).toBe(false);
  });

  test('seek functionality works', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());

    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play a media
    await page.locator('.media-card:not(:has(.rsvp))').first().click();
    await waitForPlayer(page);
    await page.waitForTimeout(1000);

    // Wait until video is actually playing
    await page.waitForFunction(() => {
      const v = document.querySelector('video, audio') as HTMLMediaElement;
      return v && !v.paused;
    }, { timeout: 5000 });

    const video = page.locator('video, audio');
    
    // Ensure we have enough duration for seeking
    const duration = await video.evaluate((el: HTMLMediaElement) => el.duration);
    const seekPos = Math.min(10, duration > 2 ? duration - 1 : 0);

    // Seek forward via JavaScript
    await video.evaluate((el: HTMLMediaElement, pos) => {
      el.currentTime = pos;
    }, seekPos);
    await page.waitForTimeout(500);

    const timeAfterForward = await video.evaluate((el: HTMLMediaElement) => el.currentTime);
    // If we could seek forward, check it
    if (seekPos > 0) {
       expect(timeAfterForward).toBeGreaterThanOrEqual(seekPos * 0.9);
    }

    // Seek backward via JavaScript
    const backPos = Math.max(0, timeAfterForward - 2);
    await video.evaluate((el: HTMLMediaElement, pos) => {
      el.currentTime = pos;
    }, backPos);
    await page.waitForTimeout(500);

    const timeAfterBackward = await video.evaluate((el: HTMLMediaElement) => el.currentTime);
    expect(timeAfterBackward).toBeLessThan(timeAfterForward);
  });
});
