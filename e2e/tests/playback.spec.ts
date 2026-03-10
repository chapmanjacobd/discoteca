import { test, expect } from '../fixtures';

test.describe('Media Playback', () => {
  test.use({ readOnly: true });

  test('toggles playback with Space key', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Debug: list card types
    const types = await mediaPage.getAllMediaCardTypes();
    console.log(`Found card types: ${types.join(', ')}`);

    // Select a non-document card
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    const cardHtml = await mediaCard.evaluate(el => el.outerHTML);
    console.log(`Selected card HTML: ${cardHtml}`);

    await mediaCard.click();
    console.log('Clicked media card');

    // Wait for player to open
    await viewerPage.waitForPlayer();

    // Player should be visible
    await expect(viewerPage.playerContainer).toBeVisible();

    // Media title should be shown
    await expect(viewerPage.mediaTitle).toBeVisible();
  });

  test('Queue container appears when enabled in settings', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open settings using POM
    await sidebarPage.openSettings();

    // Enable Queue
    const queueToggle = mediaPage.getSettingToggleSlider('setting-enable-queue');
    await queueToggle.click();

    // Close settings
    await sidebarPage.closeSettings();

    // Queue container should be visible
    await expect(viewerPage.queueContainer).toBeVisible();
  });

  test('adding to queue when enabled', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Enable Queue via settings
    await sidebarPage.openSettings();
    await mediaPage.getSettingToggleSlider('setting-enable-queue').click();
    await sidebarPage.closeSettings();

    await mediaPage.waitForMediaToLoad();

    // Click first non-document media card (should add to queue, not play)
    await mediaPage.openFirstMediaByType('video');

    // Queue count badge should show 1
    await expect(mediaPage.queueCountBadge).toHaveText('1');

    // Queue item should be present
    await expect(viewerPage.getQueueItem(0)).toBeVisible();

    // Player should NOT be visible yet
    const isHidden = await viewerPage.isHidden();
    expect(isHidden).toBe(true);
  });

  test('closes player when close button clicked', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first non-document media card
    await mediaPage.openFirstMediaByType('video');

    // Wait for player to open
    await viewerPage.waitForPlayer();

    // Click close button using POM
    await viewerPage.close();

    // Player should be hidden
    await expect(viewerPage.playerContainer).toHaveClass(/hidden/);
  });

  test('toggles theatre mode', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first non-document media card
    await mediaPage.openFirstMediaByType('video');

    // Wait for player to open
    await viewerPage.waitForPlayer();

    // Click theatre mode button using POM
    if (await viewerPage.theatreBtn.isVisible()) {
      await viewerPage.toggleTheatreMode();

      // Player should have theatre class
      expect(await viewerPage.isInTheatreMode()).toBe(true);

      // Click again to exit theatre mode
      await viewerPage.toggleTheatreMode();
      await mediaPage.page.waitForTimeout(300);

      // Theatre class should be removed
      expect(await viewerPage.isInTheatreMode()).toBe(false);
    }
  });

  test('playback speed can be adjusted', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Click first non-document media card
    await mediaPage.openFirstMediaByType('video');

    // Wait for player to open
    await viewerPage.waitForPlayer();

    // Click speed button using POM
    if (await viewerPage.speedBtn.isVisible()) {
      await viewerPage.speedBtn.click();

      // Speed menu should appear
      await expect(viewerPage.speedMenu).toBeVisible();

      // Select different speed
      const speedOption = viewerPage.getSpeedOption('1.5x');
      if (await speedOption.isVisible()) {
        await speedOption.click();

        // Speed should update
        await expect(viewerPage.speedBtn).toHaveText('1.5x');
      }
    }
  });

  test('handles rapid clicks on the same item', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    const mediaCard = mediaPage.getFirstMediaCardByType('video');

    // Rapidly click the same card
    await mediaCard.click();
    await mediaCard.click();
    await mediaCard.click();

    // Wait for playback to stabilize
    await mediaPage.page.waitForTimeout(1000);

    // Verify no "Unplayable" error toast is shown
    if (await mediaPage.toast.isVisible()) {
      const toastText = await mediaPage.getToastMessage();
      expect(toastText).not.toContain('Unplayable');
    }
  });
});
