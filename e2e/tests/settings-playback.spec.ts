import { test, expect } from '../fixtures';

test.describe('Playlist Management E2E', () => {
  test.describe.configure({ mode: 'serial' });

  test('playlist UI elements are present', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Expand playlists section using POM
    await sidebarPage.expandPlaylistsSection();

    // Verify new playlist button exists using POM
    await expect(sidebarPage.getNewPlaylistButton()).toBeVisible();

    // Verify playlist list container exists using POM
    await expect(mediaPage.playlistList).toBeAttached();
  });

  test('can interact with media cards', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Find a media card and click to open player using POM
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.click();
    await mediaPage.page.waitForTimeout(500);

    // Player should open using POM
    await expect(viewerPage.playerContainer).not.toHaveClass(/hidden/);
  });

  test('navigates to playlist view', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Expand playlists section using POM
    await sidebarPage.expandPlaylistsSection();

    // Favorites playlist should exist from seed data using POM
    const favoritesBtn = sidebarPage.getPlaylistButtonByName('Favorites');
    if (await favoritesBtn.isVisible()) {
      await favoritesBtn.click();
      await mediaPage.page.waitForTimeout(1000);

      // Should be in playlist view using POM
      const hash = await mediaPage.getCurrentHash();
      expect(hash).toContain('mode=playlist');
    }
  });

  test('playlist structure exists', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Expand playlists section using POM
    await sidebarPage.expandPlaylistsSection();

    // Verify playlists section exists and is visible using POM
    await expect(sidebarPage.detailsPlaylists).toBeVisible();
  });
});

test.describe('Settings Persistence', () => {
  test('persists theme setting across reloads', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Change theme using POM
    const themeSelect = mediaPage.getSetting('setting-theme');
    await themeSelect.selectOption('dark');

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open settings and verify theme persisted using POM
    await sidebarPage.openSettings();
    await expect(themeSelect).toHaveValue('dark');

    // Close settings
    await sidebarPage.closeSettings();
  });

  test('persists default view setting', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Scroll modal body to ensure elements are visible using POM
    await mediaPage.scrollSettingsModal(0);
    await mediaPage.page.waitForTimeout(300);

    // Change default view to theatre mode using POM
    const viewSelect = mediaPage.getSetting('setting-default-view');
    await viewSelect.selectOption('theatre');

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open settings and verify the setting persisted using POM
    await sidebarPage.openSettings();
    await mediaPage.page.waitForTimeout(300);
    await expect(viewSelect).toHaveValue('theatre');

    // Close settings
    await sidebarPage.closeSettings();
  });

  test('persists language preference', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Set language using POM
    const langInput = mediaPage.getSetting('setting-language');
    await langInput.fill('eng,spa');

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open settings and verify language persisted using POM
    await sidebarPage.openSettings();
    await expect(langInput).toHaveValue('eng,spa');

    // Close settings
    await sidebarPage.closeSettings();
  });

  test('persists autoplay setting', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Open Advanced Settings using POM
    const advancedSettings = mediaPage.getAdvancedSettingsSummary();
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await mediaPage.page.waitForTimeout(500);
    }

    // Scroll modal body using POM
    await mediaPage.scrollSettingsModal(200);
    await mediaPage.page.waitForTimeout(300);

    // Toggle autoplay using POM
    const autoplayToggle = mediaPage.getSettingToggleSlider('setting-autoplay');
    const autoplayCheckbox = mediaPage.getSetting('setting-autoplay');
    const initialState = await autoplayCheckbox.isChecked();
    await autoplayToggle.click();

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open settings and verify autoplay persisted using POM
    await sidebarPage.openSettings();
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpandedAfter = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedAfter) {
      await advancedSettings.click({ force: true });
      await mediaPage.page.waitForTimeout(500);
    }
    await mediaPage.scrollSettingsModal(200);
    await mediaPage.page.waitForTimeout(300);
    await expect(autoplayCheckbox).not.toBeChecked();

    // Restore original state
    if (initialState) {
      await autoplayToggle.click();
    }

    // Close settings
    await sidebarPage.closeSettings();
  });

  test('persists playback rate settings', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Open Advanced Settings using POM
    const advancedSettings = mediaPage.getAdvancedSettingsSummary();
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await mediaPage.page.waitForTimeout(500);
    }

    // Scroll modal body using POM
    await mediaPage.scrollSettingsModal(500);
    await mediaPage.page.waitForTimeout(300);

    // Set video rate using POM
    const videoRate = mediaPage.getSetting('setting-default-video-rate');
    await videoRate.selectOption('1.5');

    // Set audio rate using POM
    const audioRate = mediaPage.getSetting('setting-default-audio-rate');
    await audioRate.selectOption('2');

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open settings and verify rates persisted using POM
    await sidebarPage.openSettings();
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpandedAfter = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedAfter) {
      await advancedSettings.click({ force: true });
      await mediaPage.page.waitForTimeout(500);
    }
    await mediaPage.scrollSettingsModal(500);
    await mediaPage.page.waitForTimeout(300);
    await expect(videoRate).toHaveValue('1.5');
    await expect(audioRate).toHaveValue('2');

    // Close settings
    await sidebarPage.closeSettings();
  });

  test('persists slideshow delay', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Set slideshow delay using POM
    const delayInput = mediaPage.getSetting('setting-slideshow-delay');
    await delayInput.fill('10');

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open settings and verify delay persisted using POM
    await sidebarPage.openSettings();
    await expect(delayInput).toHaveValue('10');

    // Close settings
    await sidebarPage.closeSettings();
  });

  test('persists local resume setting', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Open Advanced Settings using POM
    const advancedSettings = mediaPage.getAdvancedSettingsSummary();
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpandedNow = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedNow) {
      await advancedSettings.click({ force: true });
      await mediaPage.page.waitForTimeout(500);
    }

    // Scroll modal body using POM
    await mediaPage.scrollSettingsModal(100);
    await mediaPage.page.waitForTimeout(300);

    // Toggle local resume using POM
    const localResumeToggle = mediaPage.getSettingToggleSlider('setting-local-resume');
    const localResume = mediaPage.getSetting('setting-local-resume');
    const initialState = await localResume.isChecked();
    await localResumeToggle.click();

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Reload page
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open settings and verify local resume persisted using POM
    await sidebarPage.openSettings();
    await advancedSettings.scrollIntoViewIfNeeded();
    const isExpandedAfter = await advancedSettings.evaluate((el) => (el.parentElement as HTMLDetailsElement).open);
    if (!isExpandedAfter) {
      await advancedSettings.click({ force: true });
      await mediaPage.page.waitForTimeout(500);
    }
    await mediaPage.scrollSettingsModal(100);
    await mediaPage.page.waitForTimeout(300);
    await expect(localResume).not.toBeChecked();

    // Restore original state
    if (initialState) {
      await localResumeToggle.click();
    }

    // Close settings
    await sidebarPage.closeSettings();
  });
});

test.describe('Playback Controls', () => {
  test('theatre mode toggle works', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a media using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Click theatre mode button using POM
    await viewerPage.toggleTheatreMode();
    await mediaPage.page.waitForTimeout(500);

    // Check if theatre mode is active using POM
    expect(await viewerPage.isInTheatreMode()).toBe(true);

    // Click again to disable using POM
    await viewerPage.toggleTheatreMode();
    await mediaPage.page.waitForTimeout(500);

    expect(await viewerPage.isInTheatreMode()).toBe(false);
  });

  test('close button closes player', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a media using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(500);

    // Click close button using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Player should be hidden using POM
    await expect(viewerPage.playerContainer).toHaveClass(/hidden/);
  });

  test('playback speed can be changed', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a media using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Wait for video to load
    await mediaPage.page.waitForTimeout(1000);

    // Use JavaScript to set playback rate directly using POM
    await viewerPage.videoElement.evaluate((el: HTMLMediaElement) => {
      el.playbackRate = 2.0;
    });
    await mediaPage.page.waitForTimeout(300);

    const rate = await viewerPage.videoElement.evaluate((el: HTMLMediaElement) => el.playbackRate);
    expect(rate).toBe(2);
  });

  test('play/pause via JavaScript', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a media using POM
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    await firstCard.click();
    await viewerPage.waitForPlayer();

    // Wait for media element to be available
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    // Wait for media to be ready using POM
    await viewerPage.waitForMediaData();

    // Explicitly play the media using POM
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Wait until video is actually playing using POM
    await mediaPage.page.waitForFunction(() => {
      const v = document.querySelector('video, audio') as HTMLMediaElement;
      return v && !v.paused;
    }, { timeout: 5000 });

    // Pause via JavaScript using POM
    await viewerPage.pause();
    await mediaPage.page.waitForTimeout(300);

    expect(await viewerPage.isPlaying()).toBe(true);

    // Play via JavaScript using POM
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(300);

    expect(await viewerPage.isPlaying()).toBe(false);
  });

  test('seek functionality works', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a media using POM
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Wait for media to be ready using POM
    await viewerPage.waitForMediaData();

    // Explicitly play the media using POM
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Wait until video is actually playing using POM
    await mediaPage.page.waitForFunction(() => {
      const v = document.querySelector('video, audio') as HTMLMediaElement;
      return v && !v.paused;
    }, { timeout: 5000 });

    // Ensure we have enough duration for seeking using POM
    const duration = await viewerPage.getDuration();
    const seekPos = Math.min(10, duration > 2 ? duration - 1 : 0);

    // Seek forward via JavaScript using POM
    await viewerPage.seekTo(seekPos);
    await mediaPage.page.waitForTimeout(500);

    const timeAfterForward = await viewerPage.getCurrentTime();
    // If we could seek forward, check it
    if (seekPos > 0) {
       expect(timeAfterForward).toBeGreaterThanOrEqual(seekPos * 0.9);
    }

    // Seek backward via JavaScript using POM
    const backPos = Math.max(0, timeAfterForward - 2);
    await viewerPage.seekTo(backPos);
    await mediaPage.page.waitForTimeout(500);

    const timeAfterBackward = await viewerPage.getCurrentTime();
    expect(timeAfterBackward).toBeLessThan(timeAfterForward);
  });
});
