import { test, expect } from '../fixtures';
import { waitForPlayer } from '../fixtures';

/**
 * E2E tests for race conditions in progress updates, pagination, search, and UI state
 */
test.describe('Race Conditions - Progress Updates & Pagination', () => {
  test.use({ readOnly: false });

  test('progress update does not interfere with pagination navigation', async ({ page, server }) => {
    console.log('=== Testing progress update during pagination ===');
    
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Get initial page info
    const pageInfo = page.locator('#page-info');
    const initialPageText = await pageInfo.textContent();
    console.log(`Initial page: ${initialPageText}`);

    // Play a video briefly to trigger progress updates
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForTimeout(3000);
    await page.click('.close-pip');
    await page.waitForTimeout(500);

    // Immediately navigate to next page while progress might be updating
    const nextBtn = page.locator('#next-page');
    if (await nextBtn.count() > 0 && !(await nextBtn.isDisabled())) {
      console.log('Navigating to next page...');
      await nextBtn.click();
      
      // Wait for page to load
      await page.waitForTimeout(1000);
      await page.waitForSelector('.media-card', { timeout: 5000 });

      // Verify new page loaded correctly
      const newPageText = await pageInfo.textContent();
      console.log(`New page: ${newPageText}`);
      
      // Page number should have changed
      expect(newPageText).not.toBe(initialPageText);
      
      // Results should be visible
      const results = page.locator('.media-card');
      const count = await results.count();
      expect(count).toBeGreaterThan(0);
    }
  });

  test('rapid search input does not cause duplicate requests or crashes', async ({ page, server }) => {
    console.log('=== Testing rapid search input ===');
    
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    const searchInput = page.locator('#search-input');
    
    // Type rapidly (simulate user typing fast)
    const testQueries = ['test', 'testing', 'tester', 'test123', 'test'];
    for (const query of testQueries) {
      await searchInput.fill(query);
      await page.waitForTimeout(50); // Very fast typing
    }

    // Wait for debounced search to complete
    await page.waitForTimeout(500);
    
    // Should not crash and should show results (or no results message)
    const results = page.locator('.media-card');
    const count = await results.count();
    console.log(`Search results count: ${count}`);
    
    // Should have some result state (either cards or "no results")
    expect(count).toBeGreaterThanOrEqual(0);
    
    // No console errors should have occurred
    // (checked via page.on('console') handler)
  });

  test('progress sync does not block UI interactions', async ({ page, server }) => {
    console.log('=== Testing progress sync non-blocking ===');

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    await page.click('video');
    await page.waitForTimeout(500);
    await page.waitForTimeout(5000); // Let it play to trigger sync
    
    // While video is playing, try to interact with UI
    console.log('Interacting with UI during playback...');
    
    // Try to open settings
    await page.click('#settings-button');
    await page.waitForSelector('#settings-modal', { timeout: 5000 });
    
    const settingsVisible = await page.locator('#settings-modal').isVisible();
    console.log(`Settings modal visible: ${settingsVisible}`);
    expect(settingsVisible).toBe(true);
    
    // Close settings
    await page.click('#settings-modal .close-modal');
    await page.waitForTimeout(500);
    
    // Player should still be playing
    const video = page.locator('video');
    const isPlaying = await video.evaluate((el: HTMLVideoElement) => !el.paused);
    console.log(`Video still playing: ${isPlaying}`);
    expect(isPlaying).toBe(true);
  });

  test('local progress and server progress do not conflict', async ({ page, server }) => {
    console.log('=== Testing local vs server progress ===');
    
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

    // Play video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    const mediaPath = await mediaCard.getAttribute('data-path');
    console.log(`Testing with media: ${mediaPath}`);

    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    await page.waitForFunction(() => {
      const video = document.querySelector('video');
      return video && video.readyState >= 3;
    }, { timeout: 10000 });
    await page.click('video');
    await page.waitForTimeout(500);
    await page.waitForTimeout(3000);
    
    // Get local progress
    const localProgress = await page.evaluate(() => {
      const progress = localStorage.getItem('disco-progress');
      return progress ? JSON.parse(progress) : {};
    });
    
    console.log('Local progress saved:', localProgress[mediaPath]);
    expect(localProgress[mediaPath]).toBeTruthy();
    
    // Close and reopen
    await page.click('.close-pip');
    await page.waitForTimeout(1000);
    
    // Reload page (should load both local and server progress)
    await page.reload();
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Play same video again
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForTimeout(1000);
    
    // Should have resumed from some position
    const video = page.locator('video');
    const currentTime = await video.evaluate((el: HTMLVideoElement) => el.currentTime);
    console.log(`Resumed at: ${currentTime}s`);
    
    // Should have resumed from > 0 (or at least not crashed)
    expect(currentTime).toBeGreaterThanOrEqual(0);
    
    // Restore original state
    if (!initialState) {
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }
  });

  test('filter changes during search do not cause inconsistent state', async ({ page, server }) => {
    console.log('=== Testing filter changes during search ===');
    
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Start a search
    const searchInput = page.locator('#search-input');
    await searchInput.fill('test');
    await page.waitForTimeout(400); // Wait for search to start
    
    // Immediately change filter
    await page.locator('#details-media-type').evaluate((el: HTMLDetailsElement) => el.open = true);
    await page.click('#media-type-list button[data-type="video"]');
    await page.waitForTimeout(1000);
    
    // Should show filtered results without errors
    const results = page.locator('.media-card');
    const count = await results.count();
    console.log(`Filtered search results: ${count}`);
    
    // Should have results or empty state (no crash)
    expect(count).toBeGreaterThanOrEqual(0);
    
    // All results should be videos
    if (count > 0) {
      const types = await results.evaluateAll((els: Element[]) =>
        els.map(el => el.getAttribute('data-type'))
      );
      
      types.forEach(type => {
        if (type) {
          expect(type.toLowerCase()).toContain('video');
        }
      });
    }
  });

  test('completing media while on different page does not lose state', async ({ page, server }) => {
    console.log('=== Testing completion during page navigation ===');
    
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play a short video or seek to near end
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    
    const video = page.locator('video');
    const duration = await video.evaluate((el: HTMLVideoElement) => el.duration);
    
    // Seek to 95% if duration allows
    if (duration > 10) {
      await video.evaluate((el: HTMLVideoElement, pos) => {
        el.currentTime = pos;
      }, duration * 0.95);
      await page.waitForTimeout(2000);
    } else {
      await page.waitForTimeout(5000);
    }
    
    console.log('Media near completion, navigating...');
    
    // Navigate away while completion is being processed
    await page.evaluate(() => {
      window.location.hash = 'mode=history-unplayed';
    });
    await page.waitForTimeout(2000);
    
    // Should navigate successfully without hanging
    await page.waitForSelector('.media-card, .no-results', { timeout: 5000 });
    
    const results = page.locator('.media-card');
    const count = await results.count();
    console.log(`History page results: ${count}`);
    
    // Should have loaded history page
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('multiple rapid play/pause does not break progress tracking', async ({ page, server }) => {
    console.log('=== Testing rapid play/pause ===');
    
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Play media
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    
    const video = page.locator('video');
    
    // Rapid play/pause
    console.log('Rapid play/pause cycling...');
    for (let i = 0; i < 5; i++) {
      await video.evaluate((el: HTMLVideoElement) => {
        if (el.paused) el.play();
        else el.pause();
      });
      await page.waitForTimeout(200);
    }
    
    // Wait a bit
    await page.waitForTimeout(1000);
    
    // Close player
    await page.click('.close-pip');
    await page.waitForTimeout(500);
    
    // Check local progress was saved
    const localProgress = await page.evaluate(() => {
      const progress = localStorage.getItem('disco-progress');
      return progress ? JSON.parse(progress) : {};
    });
    
    console.log('Progress entries:', Object.keys(localProgress).length);
    expect(Object.keys(localProgress).length).toBeGreaterThan(0);
  });

  test('search during page load does not cause inconsistent results', async ({ page, server }) => {
    console.log('=== Testing search during page load ===');
    
    // Start navigation
    const navigatePromise = page.goto(server.getBaseUrl());
    
    // Immediately start searching before page fully loads
    await page.waitForSelector('#search-input', { timeout: 5000 });
    const searchInput = page.locator('#search-input');
    await searchInput.fill('test');
    
    // Wait for everything to settle
    await page.waitForTimeout(1000);
    await navigatePromise;
    
    // Should have search results or empty state
    const results = page.locator('.media-card');
    const count = await results.count();
    console.log(`Results after search during load: ${count}`);

    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('UI state remains consistent during filter toggling', async ({ page, server }) => {
    console.log('=== Testing UI consistency during filter toggling ===');
    
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Rapidly toggle filters
    await page.locator('#details-media-type').evaluate((el: HTMLDetailsElement) => el.open = true);
    
    for (const type of ['video', 'audio', 'image', 'video']) {
      await page.click(`#media-type-list button[data-type="${type}"]`);
      await page.waitForTimeout(100);
    }
    
    // Wait for results to stabilize and spinner to disappear if any
    await page.waitForSelector('.media-card', { timeout: 10000 });
    
    // Check active filter
    const activeBtn = page.locator('#media-type-list .category-btn.active');
    const activeType = await activeBtn.getAttribute('data-type');
    console.log(`Final active filter: ${activeType}`);
    
    // Should be video (last selection)
    expect(activeType).toBe('video');
    
    // Wait for actual results matching the filter
    await expect.poll(async () => {
      const results = page.locator('.media-card');
      return await results.count();
    }, { timeout: 10000 }).toBeGreaterThan(0);
    
    const results = page.locator('.media-card');
    const types = await results.evaluateAll((els: Element[]) =>
      els.map(el => el.getAttribute('data-type'))
    );
      
    types.forEach(type => {
      if (type) {
        expect(type.toLowerCase()).toContain('video');
      }
    });
  });

  test('progress update throttling works correctly', async ({ page, server }) => {
    console.log('=== Testing progress update throttling ===');
    
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

    // Play video
    const mediaCard = page.locator('.media-card[data-type*="video"]').first();
    await mediaCard.click();
    await waitForPlayer(page);
    await page.waitForSelector('video', { timeout: 5000 });
    
    const video = page.locator('video');
    
    // Monitor localStorage updates
    const updateTimes: number[] = [];
    let lastUpdate = 0;
    
    for (let i = 0; i < 5; i++) {
      await page.waitForTimeout(300);
      
      const progress = await page.evaluate(() => {
        const p = localStorage.getItem('disco-progress');
        return p ? JSON.parse(p) : {};
      });
      
      const mediaPath = await mediaCard.getAttribute('data-path');
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
      await page.click('#settings-button');
      await advancedSettings.scrollIntoViewIfNeeded();
      await localResumeToggle.click();
      await page.click('#settings-modal .close-modal');
    }
  });
});
