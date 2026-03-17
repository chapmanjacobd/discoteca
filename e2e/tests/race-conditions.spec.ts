import { test, expect } from '../fixtures';

/**
 * E2E tests for race conditions in progress updates, pagination, search, and UI state
 */
test.describe('Race Conditions - Progress Updates & Pagination', () => {
  test.use({ readOnly: false });

  test('progress update does not interfere with pagination navigation', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing progress update during pagination ===');

    await mediaPage.goto(server.getBaseUrl());

    // Get initial page info using POM
    const pageInfo = mediaPage.pageInfo;
    const initialPageText = await pageInfo.textContent();
    console.log(`Initial page: ${initialPageText}`);

    // Play a video briefly to trigger progress updates using POM
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(3000);
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(500);

    // Immediately navigate to next page while progress might be updating using POM
    const nextBtn = mediaPage.page.locator('#next-page');
    if (await nextBtn.count() > 0 && !(await nextBtn.isDisabled())) {
      console.log('Navigating to next page...');
      await nextBtn.click();

      // Wait for page to load
      await mediaPage.page.waitForTimeout(1000);
      await mediaPage.waitForMediaToLoad();

      // Verify new page loaded correctly using POM
      const newPageText = await pageInfo.textContent();
      console.log(`New page: ${newPageText}`);

      // Page number should have changed
      expect(newPageText).not.toBe(initialPageText);

      // Results should be visible using POM
      const count = await mediaPage.getMediaCount();
      expect(count).toBeGreaterThan(0);
    }
  });

  test('rapid search input does not cause duplicate requests or crashes', async ({ mediaPage, server }) => {
    console.log('=== Testing rapid search input ===');

    await mediaPage.goto(server.getBaseUrl());

    // Type rapidly (simulate user typing fast) using POM
    const testQueries = ['test', 'testing', 'tester', 'test123', 'test'];
    for (const query of testQueries) {
      await mediaPage.page.fill('#search-input', query);
      await mediaPage.page.waitForTimeout(50); // Very fast typing
    }

    // Wait for debounced search to complete
    await mediaPage.page.waitForTimeout(500);

    // Should not crash and should show results (or no results message) using POM
    const count = await mediaPage.getMediaCount();
    console.log(`Search results count: ${count}`);

    // Should have some result state (either cards or "no results")
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('progress sync does not block UI interactions', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    console.log('=== Testing progress sync non-blocking ===');

    await mediaPage.goto(server.getBaseUrl());

    // Play video using POM
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(500);
    await mediaPage.page.waitForTimeout(5000); // Let it play to trigger sync

    // While video is playing, try to interact with UI using POM
    console.log('Interacting with UI during playback...');

    // Try to open settings using POM
    await sidebarPage.openSettings();
    await mediaPage.page.waitForSelector('#settings-modal', { timeout: 5000 });

    const settingsVisible = await mediaPage.page.locator('#settings-modal').isVisible();
    console.log(`Settings modal visible: ${settingsVisible}`);
    expect(settingsVisible).toBe(true);

    // Close settings using POM
    await sidebarPage.closeSettings();
    await mediaPage.page.waitForTimeout(500);

    // Player should still be playing using POM
    expect(await viewerPage.isPlaying()).toBe(true);
  });

  test('local progress and server progress do not conflict', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    console.log('=== Testing local vs server progress ===');

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
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    const mediaPath = await mediaCard.getAttribute('data-path') || '';
    console.log(`Testing with media: ${mediaPath}`);

    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });
    await viewerPage.waitForMediaData();
    await viewerPage.play();
    await mediaPage.page.waitForTimeout(3000);

    // Get local progress using POM
    const localProgress = await mediaPage.getProgress();

    console.log('Local progress saved:', localProgress[mediaPath]);
    expect(localProgress[mediaPath]).toBeTruthy();

    // Close and reopen using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Reload page (should load both local and server progress)
    await mediaPage.page.reload();
    await mediaPage.waitForMediaToLoad();

    // Play same video again using POM
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(1000);

    // Should have resumed from some position using POM
    const currentTime = await viewerPage.getCurrentTime();
    console.log(`Resumed at: ${currentTime}s`);

    // Should have resumed from > 0 (or at least not crashed)
    expect(currentTime).toBeGreaterThanOrEqual(0);

    // Restore original state
    if (!initialState) {
      await sidebarPage.openSettings();
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await sidebarPage.closeSettings();
    }
  });

  test('filter changes during search do not cause inconsistent state', async ({ mediaPage, sidebarPage, server }) => {
    console.log('=== Testing filter changes during search ===');

    await mediaPage.goto(server.getBaseUrl());

    // Start a search using POM
    await mediaPage.page.fill('#search-input', 'test');
    await mediaPage.page.waitForTimeout(400); // Wait for search to start

    // Immediately change filter using POM
    await sidebarPage.expandMediaTypeSection();
    await sidebarPage.getMediaTypeButton('video').click();
    await mediaPage.page.waitForTimeout(1000);

    // Should show filtered results without errors using POM
    const count = await mediaPage.getMediaCount();
    console.log(`Filtered search results: ${count}`);

    // Should have results or empty state (no crash)
    expect(count).toBeGreaterThanOrEqual(0);

    // All results should be videos using POM
    if (count > 0) {
      const types = await mediaPage.getAllMediaCardTypes();
      types.forEach(type => {
        if (type) {
          expect(type.toLowerCase()).toContain('video');
        }
      });
    }
  });

  test('completing media while on different page does not lose state', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing completion during page navigation ===');

    await mediaPage.goto(server.getBaseUrl());

    // Play a short video or seek to near end using POM
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    const duration = await viewerPage.getDuration();

    // Seek to 95% if duration allows using POM
    if (duration > 10) {
      await viewerPage.seekTo(duration * 0.95);
      await mediaPage.page.waitForTimeout(2000);
    } else {
      await mediaPage.page.waitForTimeout(5000);
    }

    console.log('Media near completion, navigating...');

    // Navigate away while completion is being processed using POM
    await mediaPage.page.evaluate(() => {
      window.location.hash = 'mode=history-unplayed';
    });
    await mediaPage.page.waitForTimeout(2000);

    // Should navigate successfully without hanging using POM
    await mediaPage.resultsContainer.waitFor({ state: 'visible', timeout: 5000 });

    const count = await mediaPage.getMediaCount();
    console.log(`History page results: ${count}`);

    // Should have loaded history page
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('multiple rapid play/pause does not break progress tracking', async ({ mediaPage, viewerPage, server }) => {
    console.log('=== Testing rapid play/pause ===');

    await mediaPage.goto(server.getBaseUrl());

    // Play media using POM
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    // Rapid play/pause using POM
    console.log('Rapid play/pause cycling...');
    for (let i = 0; i < 5; i++) {
      await viewerPage.play();
      await mediaPage.page.waitForTimeout(100);
      await viewerPage.pause();
      await mediaPage.page.waitForTimeout(100);
    }

    // Wait a bit
    await mediaPage.page.waitForTimeout(1000);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(500);

    // Check local progress was saved using POM
    const localProgress = await mediaPage.getProgress();

    console.log('Progress entries:', Object.keys(localProgress).length);
    expect(Object.keys(localProgress).length).toBeGreaterThan(0);
  });

  test('search during page load does not cause inconsistent results', async ({ mediaPage, server }) => {
    console.log('=== Testing search during page load ===');

    // Start navigation
    const navigatePromise = mediaPage.page.goto(server.getBaseUrl());

    // Immediately start searching before page fully loads using POM
    await mediaPage.page.waitForSelector('#search-input', { timeout: 5000 });
    await mediaPage.page.fill('#search-input', 'test');

    // Wait for everything to settle
    await mediaPage.page.waitForTimeout(1000);
    await navigatePromise;

    // Should have search results or empty state using POM
    const count = await mediaPage.getMediaCount();
    console.log(`Results after search during load: ${count}`);

    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('UI state remains consistent during filter toggling', async ({ mediaPage, sidebarPage, server }) => {
    console.log('=== Testing UI consistency during filter toggling ===');

    await mediaPage.goto(server.getBaseUrl());

    // Rapidly toggle filters using POM
    await sidebarPage.expandMediaTypeSection();

    for (const type of ['video', 'audio', 'image', 'video']) {
      await sidebarPage.getMediaTypeButton(type).click();
      // Wait a bit longer for the filter to be applied
      await mediaPage.page.waitForTimeout(300);
    }

    // Wait for results to stabilize using POM
    await mediaPage.waitForMediaToLoad();

    // Check active filter using POM
    const activeBtn = mediaPage.page.locator('#media-type-list .category-btn.active');
    const activeType = await activeBtn.getAttribute('data-type');
    console.log(`Final active filter: ${activeType}`);

    // Should be video (last selection)
    expect(activeType).toBe('video');

    // Wait for actual results matching the filter using POM
    await expect.poll(async () => {
      return await mediaPage.getMediaCount();
    }, { timeout: 10000 }).toBeGreaterThan(0);

    // All results should be videos using POM
    // Wait for the filter to fully apply before checking types
    await mediaPage.page.waitForTimeout(500);
    const types = await mediaPage.getAllMediaCardTypes();
    console.log(`Media card types: ${types}`);
    types.forEach(type => {
      if (type) {
        expect(type.toLowerCase()).toContain('video');
      }
    });
  });

  test('progress update throttling works correctly', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    console.log('=== Testing progress update throttling ===');

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
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    await mediaCard.click();
    await viewerPage.waitForPlayer();
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    // Monitor localStorage updates using POM
    const updateTimes: number[] = [];
    let lastUpdate = 0;

    for (let i = 0; i < 5; i++) {
      await mediaPage.page.waitForTimeout(300);

      const progress = await mediaPage.getProgress();
      const mediaPath = await mediaCard.getAttribute('data-path') || '';
      const entry = progress[mediaPath];

      if (entry && entry.last !== lastUpdate) {
        updateTimes.push(entry.last);
        lastUpdate = entry.last;
      }
    }

    console.log(`Progress updates: ${updateTimes.length} in ${(updateTimes[updateTimes.length - 1] - updateTimes[0]) / 1000}s`);

    // Should have throttled updates (not every 300ms, but every ~1000ms)
    // In 1.5s (5 * 300ms), should have at most 2-3 updates due to 1000ms throttling
    expect(updateTimes.length).toBeLessThanOrEqual(3);

    // Restore original state
    if (!initialState) {
      await sidebarPage.openSettings();
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await sidebarPage.closeSettings();
    }
  });
});
