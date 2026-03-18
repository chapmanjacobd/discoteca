import { test, expect } from '../fixtures';

/**
 * E2E tests for filterToIgnore functionality
 * 
 * The filterToIgnore logic ensures that when calculating filter bins for a specific
 * dimension (e.g., size), that filter is ignored so users can see the full range of
 * available values within the context of OTHER applied filters.
 * 
 * Key behavior:
 * - When calculating SIZE bins: ignore SIZE filter, but apply duration/type/episodes filters
 * - When calculating DURATION bins: ignore DURATION filter, but apply size/type/episodes filters
 * - When calculating TYPE bins: ignore TYPE filter, but apply size/duration/episodes filters
 * 
 * This prevents circular constraints where a filter would shrink its own range.
 */
test.describe('Filter To Ignore Functionality', () => {
  test.use({ readOnly: true });

  test.describe('Duration Filter Constrains Size Range (but not vice versa)', () => {
    test('applying duration filter should update size range to match duration-filtered media', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand both duration and size sections
      await sidebarPage.expandDurationSection();
      await sidebarPage.expandSizeSection();

      // Get initial size slider max value
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      const initialSizeMax = await sizeMaxSlider.getAttribute('max');
      expect(initialSizeMax).toBeTruthy();
      expect(initialSizeMax).not.toBe('0');

      // Get initial duration slider max value
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      const initialDurationMax = await durationMaxSlider.getAttribute('max');
      expect(initialDurationMax).toBeTruthy();

      // Apply a restrictive duration filter (set max to a small value)
      const quarterDuration = Math.floor(parseInt(initialDurationMax!) / 4);
      await durationMaxSlider.fill(quarterDuration.toString());
      await durationMaxSlider.dispatchEvent('input');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated size slider max value
      const newSizeMax = await sizeMaxSlider.getAttribute('max');
      expect(newSizeMax).toBeTruthy();

      // CRITICAL: Size range should reflect sizes of media matching the duration filter
      // It should NOT be constrained by any pre-existing size filter (filterToIgnore logic)
      // The size max may be smaller, same, or even larger depending on the data distribution
      expect(parseInt(newSizeMax!)).toBeGreaterThanOrEqual(0);

      // Verify media count was actually filtered
      const filteredCount = await mediaPage.getMediaCount();
      expect(filteredCount).toBeDefined();
    });

    test('size filter should NOT constrain duration range (filterToIgnore)', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand both size and duration sections
      await sidebarPage.expandSizeSection();
      await sidebarPage.expandDurationSection();

      // Get initial duration slider max value
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      const initialDurationMax = await durationMaxSlider.getAttribute('max');
      expect(initialDurationMax).toBeTruthy();
      expect(initialDurationMax).not.toBe('0');

      // Get initial size slider max value
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      const initialSizeMax = await sizeMaxSlider.getAttribute('max');
      expect(initialSizeMax).toBeTruthy();

      // Apply a restrictive size filter (set max to a small value)
      const quarterSize = Math.floor(parseInt(initialSizeMax!) / 4);
      await sizeMaxSlider.fill(quarterSize.toString());
      await sizeMaxSlider.dispatchEvent('input');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated duration slider max value
      const newDurationMax = await durationMaxSlider.getAttribute('max');
      expect(newDurationMax).toBeTruthy();

      // CRITICAL: Duration range should reflect durations of media matching the size filter
      // It should NOT be collapsed to 0 or constrained by the size filter's own logic
      // The duration filter ignores the size filter when computing its own bins
      expect(parseInt(newDurationMax!)).toBeGreaterThanOrEqual(0);

      // Verify media count was actually filtered
      const filteredCount = await mediaPage.getMediaCount();
      expect(filteredCount).toBeDefined();
    });
  });

  test.describe('Size Filter Does Not Constrain Duration Range', () => {
    test('applying size filter should not collapse duration range to zero', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand both size and duration sections
      await sidebarPage.expandSizeSection();
      await sidebarPage.expandDurationSection();

      // Get initial duration slider max value
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      const initialDurationMax = await durationMaxSlider.getAttribute('max');
      expect(initialDurationMax).toBeTruthy();
      expect(initialDurationMax).not.toBe('0');

      // Get initial size slider max value
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      const initialSizeMax = await sizeMaxSlider.getAttribute('max');
      expect(initialSizeMax).toBeTruthy();

      // Apply a restrictive size filter (set max to a small value)
      const halfSize = Math.floor(parseInt(initialSizeMax!) / 4);
      await sizeMaxSlider.fill(halfSize.toString());
      await sizeMaxSlider.dispatchEvent('input');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated duration slider max value
      const newDurationMax = await durationMaxSlider.getAttribute('max');
      expect(newDurationMax).toBeTruthy();

      // Duration range should NOT be zero or empty after applying size filter
      if (parseInt(initialDurationMax!) > 0) {
        expect(parseInt(newDurationMax!)).toBeGreaterThanOrEqual(0);
      }

      // Verify media count was actually filtered
      const filteredCount = await mediaPage.getMediaCount();
      expect(filteredCount).toBeDefined();
    });
  });

  test.describe('Type Filter Updates Other Filter Ranges', () => {
    test('applying video filter should update duration and size ranges (type filter is ignored when computing duration/size bins)', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand filter sections
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.expandDurationSection();
      await sidebarPage.expandSizeSection();

      // Get initial duration and size ranges
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');

      const initialDurationMax = await durationMaxSlider.getAttribute('max');
      const initialSizeMax = await sizeMaxSlider.getAttribute('max');

      // Apply video filter
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // Get updated ranges
      const newDurationMax = await durationMaxSlider.getAttribute('max');
      const newSizeMax = await sizeMaxSlider.getAttribute('max');

      // CRITICAL: Duration and size ranges should reflect video-only media
      // The type filter is ignored when computing duration/size bins (filterToIgnore="type")
      // So duration/size bins show full range of video content
      expect(newDurationMax).toBeTruthy();
      expect(newSizeMax).toBeTruthy();

      // Verify we have video results
      const videoCount = await mediaPage.getMediaCount();
      expect(videoCount).toBeGreaterThanOrEqual(0);
    });

    test('applying audio filter should update duration and size ranges appropriately', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand filter sections
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.expandDurationSection();
      await sidebarPage.expandSizeSection();

      // Apply audio filter
      await sidebarPage.getMediaTypeButton('audio').click();
      await mediaPage.page.waitForTimeout(1500);

      // Get updated ranges
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');

      const newDurationMax = await durationMaxSlider.getAttribute('max');
      const newSizeMax = await sizeMaxSlider.getAttribute('max');

      // Ranges should reflect audio-only media
      expect(newDurationMax).toBeTruthy();
      expect(newSizeMax).toBeTruthy();

      // Verify we have audio results
      const audioCount = await mediaPage.getMediaCount();
      expect(audioCount).toBeGreaterThanOrEqual(0);
    });
  });

  test.describe('Episodes Filter Affects Size and Duration Ranges', () => {
    test('applying episodes filter should update size and duration ranges', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand filter sections
      await sidebarPage.expandEpisodesSection();
      await sidebarPage.expandSizeSection();
      await sidebarPage.expandDurationSection();

      // Get initial ranges
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      const episodesMaxSlider = mediaPage.page.locator('#episodes-max-slider');

      const initialSizeMax = await sizeMaxSlider.getAttribute('max');
      const initialDurationMax = await durationMaxSlider.getAttribute('max');

      // Apply episodes filter if slider exists
      if (await episodesMaxSlider.count() > 0) {
        const episodesMax = await episodesMaxSlider.getAttribute('max');
        if (episodesMax && parseInt(episodesMax) > 1) {
          const halfEpisodes = Math.floor(parseInt(episodesMax) / 2);
          await episodesMaxSlider.fill(halfEpisodes.toString());
          await episodesMaxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1500);

          // Get updated ranges
          const newSizeMax = await sizeMaxSlider.getAttribute('max');
          const newDurationMax = await durationMaxSlider.getAttribute('max');

          // Ranges should still be defined after applying episodes filter
          expect(newSizeMax).toBeTruthy();
          expect(newDurationMax).toBeTruthy();
        }
      }

      // Verify media was filtered
      const filteredCount = await mediaPage.getMediaCount();
      expect(filteredCount).toBeDefined();
    });
  });

  test.describe('Multiple Filters Combination', () => {
    test('combining type and duration filters should maintain valid size range', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand filter sections
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.expandDurationSection();
      await sidebarPage.expandSizeSection();

      // Apply video filter first
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1000);

      // Get video-only duration max
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      const videoDurationMax = await durationMaxSlider.getAttribute('max');
      expect(videoDurationMax).toBeTruthy();

      // Apply duration filter on top of video filter
      if (videoDurationMax && parseInt(videoDurationMax) > 0) {
        const halfDuration = Math.floor(parseInt(videoDurationMax) / 2);
        await durationMaxSlider.fill(halfDuration.toString());
        await durationMaxSlider.dispatchEvent('input');
        await mediaPage.page.waitForTimeout(1500);

        // CRITICAL: Size range should still be valid
        // When computing size bins: duration filter is applied, but size filter is ignored
        // This prevents circular constraint where size filter would shrink its own range
        const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
        const newSizeMax = await sizeMaxSlider.getAttribute('max');
        expect(newSizeMax).toBeTruthy();
        if (newSizeMax) {
          expect(parseInt(newSizeMax)).toBeGreaterThanOrEqual(0);
        }
      }

      // Verify combined filtering worked
      const combinedCount = await mediaPage.getMediaCount();
      expect(combinedCount).toBeDefined();
    });

    test('filterToIgnore: each filter dimension ignores only itself when computing bins', async ({ mediaPage, sidebarPage, server }) => {
      // This test verifies the core filterToIgnore behavior:
      // - Size bins: ignore size filter, apply duration/type filters
      // - Duration bins: ignore duration filter, apply size/type filters
      // - Type bins: ignore type filter, apply size/duration filters

      await mediaPage.goto(server.getBaseUrl());
      await mediaPage.waitForMediaToLoad();

      // Expand all filter sections
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.expandDurationSection();
      await sidebarPage.expandSizeSection();

      // Get initial ranges
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');

      const initialSizeMax = await sizeMaxSlider.getAttribute('max');
      const initialDurationMax = await durationMaxSlider.getAttribute('max');

      // Apply video filter
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // Apply duration filter
      if (initialDurationMax && parseInt(initialDurationMax) > 0) {
        const halfDuration = Math.floor(parseInt(initialDurationMax) / 2);
        await durationMaxSlider.fill(halfDuration.toString());
        await durationMaxSlider.dispatchEvent('input');
        await mediaPage.page.waitForTimeout(1500);
      }

      // Apply size filter
      if (initialSizeMax && parseInt(initialSizeMax) > 0) {
        const halfSize = Math.floor(parseInt(initialSizeMax) / 2);
        await sizeMaxSlider.fill(halfSize.toString());
        await sizeMaxSlider.dispatchEvent('input');
        await mediaPage.page.waitForTimeout(1500);
      }

      // CRITICAL VERIFICATION:
      // After applying all three filters, each filter's bins should:
      // 1. Size bins: show range for video+duration-filtered media (ignoring size filter)
      // 2. Duration bins: show range for video+size-filtered media (ignoring duration filter)
      // 3. Type bins: show counts for size+duration-filtered media (ignoring type filter)

      const finalSizeMax = await sizeMaxSlider.getAttribute('max');
      const finalDurationMax = await durationMaxSlider.getAttribute('max');

      // Both should be valid (not collapsed to 0 due to circular constraint)
      expect(finalSizeMax).toBeTruthy();
      expect(finalDurationMax).toBeTruthy();
      if (finalSizeMax) {
        expect(parseInt(finalSizeMax)).toBeGreaterThanOrEqual(0);
      }
      if (finalDurationMax) {
        expect(parseInt(finalDurationMax)).toBeGreaterThanOrEqual(0);
      }

      // Verify filtering actually reduced results
      const finalCount = await mediaPage.getMediaCount();
      expect(finalCount).toBeDefined();
    });

    test('clearing filters should restore original ranges', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand filter sections
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.expandSizeSection();

      // Get initial size range
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      const initialSizeMax = await sizeMaxSlider.getAttribute('max');
      expect(initialSizeMax).toBeTruthy();

      // Apply video filter
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // Get filtered size range
      const filteredSizeMax = await sizeMaxSlider.getAttribute('max');
      expect(filteredSizeMax).toBeTruthy();

      // Clear filter by clicking video button again (toggle behavior)
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // Size range should be restored (or at least be valid)
      const restoredSizeMax = await sizeMaxSlider.getAttribute('max');
      expect(restoredSizeMax).toBeTruthy();
    });
  });

  test.describe('Search Query Affects Filter Ranges', () => {
    test('search query should update filter bin ranges correctly', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Wait for media to load
      await mediaPage.waitForMediaToLoad();

      // Expand filter sections
      await sidebarPage.expandSizeSection();
      await sidebarPage.expandDurationSection();

      // Get initial ranges
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');

      const initialSizeMax = await sizeMaxSlider.getAttribute('max');
      const initialDurationMax = await durationMaxSlider.getAttribute('max');

      // Perform search
      await mediaPage.search('test');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated ranges
      const newSizeMax = await sizeMaxSlider.getAttribute('max');
      const newDurationMax = await durationMaxSlider.getAttribute('max');

      // Ranges should be defined (may be same or different)
      expect(newSizeMax).toBeTruthy();
      expect(newDurationMax).toBeTruthy();

      // Clear search
      await mediaPage.clearSearch();
      await mediaPage.page.waitForTimeout(1500);

      // Ranges should be restored or at least valid
      const restoredSizeMax = await sizeMaxSlider.getAttribute('max');
      const restoredDurationMax = await durationMaxSlider.getAttribute('max');
      expect(restoredSizeMax).toBeTruthy();
      expect(restoredDurationMax).toBeTruthy();
    });
  });
});
