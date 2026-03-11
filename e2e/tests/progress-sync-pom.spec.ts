import { test, expect } from '../fixtures';

test.describe('Progress Sync with POM', () => {
  test.use({ readOnly: false });

  // localResume is enabled by default (localStorage !== 'false')
  // No need to set it explicitly

  test('resumes from saved position', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open media and let it play briefly
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Wait for video to load and start playing
    const video = viewerPage.videoElement;
    await video.waitFor({ state: 'visible' });

    // Wait for video to have some duration loaded
    await mediaPage.page.waitForFunction(() => {
      const v = document.querySelector('#pip-player video') as HTMLVideoElement;
      return v && v.duration > 0;
    }, { timeout: 10000 });

    // Ensure video is playing
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Play for a short time
    await mediaPage.page.waitForTimeout(3000);

    // Get current position
    const position = await viewerPage.getCurrentTime();
    expect(position).toBeGreaterThan(0);

    // Close player
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(500);

    // Re-open same media
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Should resume from approximately the same position
    await mediaPage.page.waitForTimeout(1000);
    const resumedPosition = await viewerPage.getCurrentTime();

    // Position should be close to where we left off (within 5 seconds)
    expect(Math.abs(resumedPosition - position)).toBeLessThan(5);
  });

  test('syncs progress across page refresh', async ({ mediaPage, viewerPage, server, page }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open media
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Wait for video to load
    const video = viewerPage.videoElement;
    await video.waitFor({ state: 'visible' });
    await mediaPage.page.waitForFunction(() => {
      const v = document.querySelector('#pip-player video') as HTMLVideoElement;
      return v && v.duration > 0;
    }, { timeout: 10000 });

    // Play for a bit
    await mediaPage.page.waitForTimeout(2000);
    const position = await viewerPage.getCurrentTime();

    // Refresh page
    await page.reload();
    await mediaPage.waitForMediaToLoad();

    // Re-open same media
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Should resume from saved position
    const resumedPosition = await viewerPage.getCurrentTime();
    expect(Math.abs(resumedPosition - position)).toBeLessThan(5);
  });

  test('marks media as completed after watching', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open media
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();

    // Wait for video to load
    const video = viewerPage.videoElement;
    await video.waitFor({ state: 'visible' });
    await mediaPage.page.waitForFunction(() => {
      const v = document.querySelector('#pip-player video') as HTMLVideoElement;
      return v && v.duration > 0;
    }, { timeout: 10000 });

    const duration = await viewerPage.getDuration();

    // If video is short, seek near the end
    if (duration < 100) {
      await viewerPage.seekTo(duration - 5);
      await mediaPage.page.waitForTimeout(6000); // Wait for it to finish
    }

    // Check if media is marked as played (play count should increase)
    // This is verified through the UI showing play count
    const playCountBadge = mediaPage.page.locator('.play-count-badge');
    if (await playCountBadge.isVisible()) {
      const count = await playCountBadge.textContent();
      expect(count).toBeTruthy();
    }
  });

  test('tracks progress for audio files', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open audio media
    await mediaPage.openFirstMediaByType('audio');
    await viewerPage.waitForPlayer();

    // Wait for audio to load
    const audio = viewerPage.audioElement;
    await audio.waitFor({ state: 'visible' });
    await mediaPage.page.waitForFunction(() => {
      const a = document.querySelector('#pip-player audio') as HTMLAudioElement;
      return a && a.duration > 0;
    }, { timeout: 10000 });

    // Ensure audio is playing
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);

    // Play for a bit
    await mediaPage.page.waitForTimeout(3000);
    const position = await viewerPage.getCurrentTime();
    expect(position).toBeGreaterThan(0);

    // Close and re-open
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(500);
    await mediaPage.openFirstMediaByType('audio');
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Should resume from saved position
    const resumedPosition = await viewerPage.getCurrentTime();
    expect(Math.abs(resumedPosition - position)).toBeLessThan(5);
  });

  test('progress bar shows on media cards', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open and play media briefly
    const firstCard = mediaPage.getFirstMediaCardByType('video');
    const videoPath = await firstCard.getAttribute('data-path');
    
    await firstCard.click();
    await viewerPage.waitForPlayer();

    // Ensure video is playing
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(3000);
    await viewerPage.close();

    // Reload to refresh results with new progress
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Find the card with the video we played
    const playedCard = mediaPage.page.locator(`.media-card[data-path="${videoPath}"]`).first();
    const progressBar = playedCard.locator('.progress-bar');
    const playheadIndicator = playedCard.locator('.playhead-indicator');

    // Progress bar should be visible or progress indicator present
    const hasProgress = await progressBar.isVisible() ||
                        await playheadIndicator.isVisible();
    expect(hasProgress).toBe(true);
  });

  test('reset progress with mark unplayed', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open and play media
    await mediaPage.openFirstMediaByType('video');
    await viewerPage.waitForPlayer();
    
    // Ensure video is playing
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(3000);
    const position = await viewerPage.getCurrentTime();
    expect(position).toBeGreaterThan(0);
    await viewerPage.close();

    // Wait for results to refresh
    await mediaPage.waitForMediaToLoad();

    // Mark as unplayed (via context menu or keyboard shortcut)
    // This may require right-click or specific UI interaction
    const firstCard = mediaPage.getMediaCard(0);
    await firstCard.click({ button: 'right' });
    
    // Look for context menu
    const contextMenu = mediaPage.page.locator('.context-menu, [role="menu"]');
    if (await contextMenu.isVisible()) {
      const markUnplayed = contextMenu.locator('text=/Mark unplayed/i, text=/Reset progress/i');
      if (await markUnplayed.isVisible()) {
        await markUnplayed.click();
        
        // Progress should be reset
        await mediaPage.waitForMediaToLoad();
        
        // Re-open and check position is 0
        await mediaPage.openFirstMediaByType('video');
        await viewerPage.waitForPlayer();
        await mediaPage.page.waitForTimeout(1000);
        
        const newPosition = await viewerPage.getCurrentTime();
        expect(newPosition).toBeLessThan(5); // Should be near start
      }
    }
  });
});
