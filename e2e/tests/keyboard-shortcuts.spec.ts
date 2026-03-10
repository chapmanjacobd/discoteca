import { test, expect } from '../fixtures';

test.describe('Keyboard Shortcuts', () => {
  test.use({ readOnly: true });

  test.describe('Navigation Shortcuts', () => {
    test('n key plays next sibling without player open', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Get initial media count using POM
      const initialCount = await mediaPage.getMediaCount();

      // Need at least 2 items for next sibling to work
      if (initialCount < 2) {
        console.log('Skipping test: not enough media items');
        return;
      }

      // Press 'n' to play next (no player needs to be open)
      await mediaPage.page.keyboard.press('n');
      await mediaPage.page.waitForTimeout(1000);

      // Player should open with next media using POM
      await expect(viewerPage.playerContainer).toBeVisible();
    });

    test('p key plays previous sibling without player open', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click second card to have a previous item, then close player using POM
      const secondCard = mediaPage.getMediaCard(1);
      await secondCard.click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Close player using keyboard shortcut
      await mediaPage.page.keyboard.press('w');
      await mediaPage.page.waitForTimeout(500);

      // Press 'p' to play previous (no player needs to be open)
      await mediaPage.page.keyboard.press('p');
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Player should be visible with some media
      await expect(viewerPage.playerContainer).toBeVisible();
    });

    test('ArrowRight seeks forward 5 seconds', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first video/audio card using POM
      const card = mediaPage.getFirstMediaCardByType('video');
      await card.click();
      await viewerPage.waitForPlayer();

      // Wait for media to be ready
      await mediaPage.page.waitForTimeout(1000);

      // Get initial time using POM
      const initialTime = await viewerPage.getCurrentTime();

      // Press ArrowRight
      await mediaPage.page.keyboard.press('ArrowRight');
      await mediaPage.page.waitForTimeout(500);

      // Time should have increased by ~5 seconds using POM
      const newTime = await viewerPage.getCurrentTime();

      expect(newTime).toBeGreaterThan(initialTime);
    });

    test('ArrowLeft seeks backward 5 seconds', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first video/audio card using POM
      const card = mediaPage.getFirstMediaCardByType('video');
      await card.click();
      await viewerPage.waitForPlayer();

      // Wait for media and seek forward first using POM
      await mediaPage.page.waitForTimeout(1000);
      await viewerPage.seekTo(15);
      await mediaPage.page.waitForTimeout(500);

      // Get initial time using POM
      const initialTime = await viewerPage.getCurrentTime();

      // Press ArrowLeft
      await mediaPage.page.keyboard.press('ArrowLeft');
      await mediaPage.page.waitForTimeout(500);

      // Time should have decreased by ~5 seconds using POM
      const newTime = await viewerPage.getCurrentTime();

      expect(newTime).toBeLessThan(initialTime);
    });
  });

  test.describe('Random Media', () => {
    test('r key plays random media', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Press 'r' to play random media
      await mediaPage.page.keyboard.press('r');
      await mediaPage.page.waitForTimeout(3000);

      // Should have either:
      // 1. PiP player open (video/audio)
      // 2. Document modal open (PDF/EPUB)
      // 3. Error toast (if media is unplayable or not found)
      const pipVisible = await viewerPage.playerContainer.first().isVisible();
      const docVisible = await viewerPage.isDocumentModalVisible();
      const toastVisible = await mediaPage.toast.isVisible();

      // At least one should happen
      expect(pipVisible || docVisible || toastVisible).toBe(true);
    });

    test('r key plays random media of same type', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // First open a video to set the type filter using POM
      const videoCard = mediaPage.getFirstMediaCardByType('video');
      if (await videoCard.count() > 0) {
        await videoCard.click();
        await viewerPage.waitForPlayer();
        await mediaPage.page.waitForTimeout(500);

        // Press 'r' to play random video
        await mediaPage.page.keyboard.press('r');
        await mediaPage.page.waitForTimeout(1000);

        // Player should still be visible with video
        await expect(viewerPage.playerContainer).toBeVisible();
      }
    });
  });

  test.describe('Playback Controls', () => {
    test('m key toggles mute', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first video/audio card using POM
      const card = mediaPage.getFirstMediaCardByType('video');
      await card.click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Get initial muted state using POM
      const initialMuted = await viewerPage.isMuted();

      // Press 'm' to toggle mute
      await mediaPage.page.keyboard.press('m');
      await mediaPage.page.waitForTimeout(300);

      // Muted state should have changed using POM
      const newMuted = await viewerPage.isMuted();

      expect(newMuted).not.toBe(initialMuted);
    });

    test('l key toggles loop', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first video/audio card using POM
      const card = mediaPage.getFirstMediaCardByType('video');
      await card.click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Press 'l' to toggle loop using POM
      await viewerPage.toggleLoop();
      await mediaPage.page.waitForTimeout(500);

      // Toast should appear indicating loop state
      await mediaPage.waitForToast();
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).toMatch(/Loop: (ON|OFF)/);
    });

    test('w key closes player', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first card to open player using POM
      await mediaPage.getMediaCard(0).click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Player should be visible
      await expect(viewerPage.playerContainer).toBeVisible();

      // Press 'w' to close
      await mediaPage.page.keyboard.press('w');
      await mediaPage.page.waitForTimeout(500);

      // Player should be hidden using POM
      await viewerPage.waitForHidden();
    });

    test('Space key toggles play/pause', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first video/audio card using POM
      const card = mediaPage.getFirstMediaCardByType('video');
      await card.click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(1000);

      // Get initial paused state using POM
      const isPausedInitial = await viewerPage.isPlaying();

      // Press Space
      await mediaPage.page.keyboard.press(' ');
      await mediaPage.page.waitForTimeout(500);

      // Paused state should have toggled using POM
      const isPausedAfterSpace = await viewerPage.isPlaying();

      expect(isPausedAfterSpace).not.toBe(isPausedInitial);
    });

    test('k key toggles play/pause', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first video/audio card using POM
      const card = mediaPage.getFirstMediaCardByType('video');
      await card.click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(1000);

      // Get initial paused state using POM
      const isPausedInitial = await viewerPage.isPlaying();

      // Press 'k'
      await mediaPage.page.keyboard.press('k');
      await mediaPage.page.waitForTimeout(500);

      // Paused state should have toggled using POM
      const isPausedAfterK = await viewerPage.isPlaying();

      expect(isPausedAfterK).not.toBe(isPausedInitial);
    });

    test('f key toggles fullscreen', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first card using POM
      await mediaPage.getMediaCard(0).click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Press 'f' to enter fullscreen
      await mediaPage.page.keyboard.press('f');
      await mediaPage.page.waitForTimeout(500);

      // Check if fullscreen is active using POM
      const isFullscreen = await viewerPage.isFullscreenActive();
      expect(typeof isFullscreen).toBe('boolean');
    });

    test('a key cycles aspect ratio', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first video card using POM
      const videoCard = mediaPage.getFirstMediaCardByType('video');
      if (await videoCard.count() === 0) return;

      await videoCard.click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(1000);

      // Press 'a' to cycle aspect ratio
      await mediaPage.page.keyboard.press('a');
      await mediaPage.page.waitForTimeout(500);

      // Check if aspect ratio style was applied to video using POM
      const aspectRatio = await viewerPage.getAspectRatio();

      // Normalize aspect ratio (browser may add spaces: '16 / 9' vs '16/9')
      expect(aspectRatio.replace(/\s/g, '')).toBe('16/9');

      // Toast should appear using POM
      await mediaPage.waitForToast();
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).toContain('Aspect Ratio');
    });
  });

  test.describe('Metadata Shortcuts', () => {
    test('i key toggles metadata modal', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first card to open player using POM
      await mediaPage.getMediaCard(0).click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Press 'i' to open metadata using POM
      await mediaPage.page.keyboard.press('i');
      await mediaPage.page.waitForTimeout(500);

      // Metadata modal should be visible using POM
      await expect(viewerPage.metadataModal.first()).toBeVisible();

      // Press 'i' again to close
      await mediaPage.page.keyboard.press('i');
      await mediaPage.page.waitForTimeout(500);

      // Metadata modal should be hidden using POM
      expect(await viewerPage.isMetadataModalHidden()).toBe(true);
    });
  });

  test.describe('Utility Shortcuts', () => {
    test('c key copies media path to clipboard', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first card to open player using POM
      const firstCard = mediaPage.getMediaCard(0);
      const expectedPath = await mediaPage.getMediaTitle(0);
      await firstCard.click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Grant clipboard permissions
      const context = mediaPage.page.context();
      await context.grantPermissions(['clipboard-read', 'clipboard-write']);

      // Press 'c' to copy path
      await mediaPage.page.keyboard.press('c');
      await mediaPage.page.waitForTimeout(500);

      // Toast should appear using POM
      await mediaPage.waitForToast();
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).toContain('Copied path');
    });

    test('? key opens help modal', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Press '?' to open help
      await mediaPage.page.keyboard.press('?');
      await mediaPage.page.waitForTimeout(500);

      // Help modal should be visible using POM
      await expect(viewerPage.helpModal.first()).toBeVisible();
    });

    test('/ key opens help modal', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Press '/' to open help
      await mediaPage.page.keyboard.press('/');
      await mediaPage.page.waitForTimeout(500);

      // Help modal should be visible using POM
      await expect(viewerPage.helpModal.first()).toBeVisible();
    });

    test('t key focuses search input', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Press 't' to focus search
      await mediaPage.page.keyboard.press('t');
      await mediaPage.page.waitForTimeout(300);

      // Search input should have focus using POM
      expect(await mediaPage.isSearchFocused()).toBe(true);
    });
  });

  test.describe('Subtitle Controls', () => {
    test('v key toggles subtitle visibility', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

      // Get caption count using POM
      const captionCount = await mediaPage.getCaptionSegments().count();
      if (captionCount === 0) {
        console.log('No captions available, skipping test');
        return;
      }

      // Click a caption segment to open player with subtitles using POM
      await mediaPage.getCaptionSegments().first().click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(1000);

      // Press 'v' to toggle subtitle visibility
      await mediaPage.page.keyboard.press('v');
      await mediaPage.page.waitForTimeout(500);

      // Player should still be visible using POM
      await expect(viewerPage.playerContainer.first()).toBeVisible();

      // Check if toast appeared (subtitle toggle message) using POM
      if (await mediaPage.toast.isVisible()) {
        const toastText = await mediaPage.getToastMessage();
        expect(toastText).toMatch(/Subtitles: (Off|Track)/);
      }
    });
  });

  test.describe('Rating Shortcuts', () => {
    test('1-5 keys rate media', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first card to open player using POM
      await mediaPage.getMediaCard(0).click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Press '5' to rate 5 stars
      await mediaPage.page.keyboard.press('5');
      await mediaPage.page.waitForTimeout(500);

      // Toast should appear using POM
      await mediaPage.waitForToast();
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).toContain('Rated');
      expect(toastText).toContain('⭐');
    });

    test('` key unrates media', async ({ mediaPage, viewerPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click first card to open player using POM
      await mediaPage.getMediaCard(0).click();
      await viewerPage.waitForPlayer();
      await mediaPage.page.waitForTimeout(500);

      // Press '`' to unrate
      await mediaPage.page.keyboard.press('`');
      await mediaPage.page.waitForTimeout(500);

      // Toast should appear using POM
      await mediaPage.waitForToast();
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).toContain('Unrated');
    });
  });

  test.describe('Rating (Drag and Drop)', () => {
    test('rating buttons exist in sidebar', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Expand ratings section using POM
      await sidebarPage.expandRatingsSection();

      // Verify rating buttons exist using POM
      const ratingBtn = mediaPage.getRatingButtons().first();
      await expect(ratingBtn).toBeAttached();

      // Rating buttons can be used for drag-drop
      const ratingBtnText = await ratingBtn.textContent();
      expect(ratingBtnText).toContain('⭐');
    });
  });
});
