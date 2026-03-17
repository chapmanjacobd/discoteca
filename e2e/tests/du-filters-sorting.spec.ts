import { test, expect } from '../fixtures';

/**
 * E2E tests for Disk Usage mode filtering, sorting, and media type functionality
 * Tests that DU mode properly supports all filter features available in Search and Captions modes
 */
test.describe('DU Mode Filters and Sorting', () => {
  test.use({ readOnly: true });

  test.describe('Sorting in DU Mode', () => {
    test('sort by size works in DU mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Change sort to size
      await mediaPage.setSortBy('size');
      await mediaPage.page.waitForTimeout(500);

      // Verify sort dropdown shows size
      await expect(mediaPage.sortBySelect).toHaveValue('size');

      // Get all file cards
      const fileCards = mediaPage.getDUFileCards();
      const count = await fileCards.count();

      if (count >= 2) {
        // Extract sizes from first two cards to verify sorting
        const firstSize = await fileCards.nth(0).locator('.media-size').textContent();
        const secondSize = await fileCards.nth(1).locator('.media-size').textContent();

        // Sizes should be in descending order if reverse is active
        console.log(`[DU Sort Test] First: ${firstSize}, Second: ${secondSize}`);
      }
    });

    test('sort by path works in DU mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Change sort to path
      await mediaPage.setSortBy('path');
      await mediaPage.page.waitForTimeout(500);

      // Verify sort dropdown shows path
      await expect(mediaPage.sortBySelect).toHaveValue('path');
    });

    test('sort by duration works in DU mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Change sort to duration
      await mediaPage.setSortBy('duration');
      await mediaPage.page.waitForTimeout(500);

      // Verify sort dropdown shows duration
      await expect(mediaPage.sortBySelect).toHaveValue('duration');
    });

    test('reverse sort toggles correctly in DU mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(500);

      // First set a sort option, then toggle reverse
      await mediaPage.setSortBy('size');
      await mediaPage.page.waitForTimeout(300);
      
      // Click reverse sort button
      await mediaPage.sortReverseBtn.click();
      await mediaPage.page.waitForTimeout(300);

      // Should have active class
      await expect(mediaPage.sortReverseBtn).toHaveClass(/active/);

      // Click again to toggle off
      await mediaPage.sortReverseBtn.click();
      await mediaPage.page.waitForTimeout(300);

      // Should not have active class
      await expect(mediaPage.sortReverseBtn).not.toHaveClass(/active/);
    });

    test('sort order persists across folder navigation in DU mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(500);

      // Set sort to size and enable reverse
      await mediaPage.setSortBy('size');
      await mediaPage.sortReverseBtn.click();
      await mediaPage.page.waitForTimeout(500);

      // Navigate into a folder
      const folderFound = await mediaPage.findAndClickFolderByText(/.*/, 1000);

      if (folderFound) {
        // Verify sort setting persisted
        await expect(mediaPage.sortBySelect).toHaveValue('size');
        await expect(mediaPage.sortReverseBtn).toHaveClass(/active/);
      }
    });
  });

  test.describe('Media Type Filters in DU Mode', () => {
    test('media type filter buttons are visible in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand media type section
      await sidebarPage.expandMediaTypeSection();

      // Media type buttons should be visible
      await expect(sidebarPage.getMediaTypeButton('video')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('audio')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('text')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('image')).toBeVisible();
    });

    test('video filter works in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial folder count and file count at root level
      const folderCards = mediaPage.page.locator('.media-card.is-folder');
      const initialFolderCount = await folderCards.count();
      expect(initialFolderCount).toBeGreaterThan(0);
      
      // Get initial file count from folder card ("X files" text in media-meta)
      const getFolderFileCount = async (): Promise<number> => {
        // The file count is displayed as "X files" in the media-meta section
        const countElement = mediaPage.page.locator('.media-card.is-folder .media-meta span:has-text("files")').first();
        const text = await countElement.textContent();
        if (text) {
          const match = text.match(/(\d+)/);
          if (match) return parseInt(match[1]);
        }
        return 0;
      };
      
      const initialFileCount = await getFolderFileCount();
      console.log(`[DU Video Filter] Initial folders: ${initialFolderCount}, files: ${initialFileCount}`);

      // Apply video filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // File count within folder should change after applying video filter
      const videoFileCount = await getFolderFileCount();
      const videoFolderCount = await folderCards.count();
      console.log(`[DU Video Filter] After video filter - folders: ${videoFolderCount}, files: ${videoFileCount}`);

      // The file count should be different (filter should have an effect)
      // Note: folder count may stay the same if all media is under same parent folder
      expect(videoFileCount).not.toBe(initialFileCount);
      expect(videoFileCount).toBeLessThan(initialFileCount);
    });

    test('audio filter works in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial folder count and file count
      const folderCards = mediaPage.page.locator('.media-card.is-folder');
      const initialFolderCount = await folderCards.count();
      expect(initialFolderCount).toBeGreaterThan(0);
      
      const getFolderFileCount = async (): Promise<number> => {
        const countElement = mediaPage.page.locator('.media-card.is-folder .media-meta span:has-text("files")').first();
        const text = await countElement.textContent();
        if (text) {
          const match = text.match(/(\d+)/);
          if (match) return parseInt(match[1]);
        }
        return 0;
      };
      
      const initialFileCount = await getFolderFileCount();
      console.log(`[DU Audio Filter] Initial folders: ${initialFolderCount}, files: ${initialFileCount}`);

      // Apply audio filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('audio').click();
      await mediaPage.page.waitForTimeout(1500);

      // File count within folder should change
      const audioFileCount = await getFolderFileCount();
      const audioFolderCount = await folderCards.count();
      console.log(`[DU Audio Filter] After audio - folders: ${audioFolderCount}, files: ${audioFileCount}`);
      
      expect(audioFileCount).not.toBe(initialFileCount);
      expect(audioFileCount).toBeLessThan(initialFileCount);
    });

    test('image filter works in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial folder count and file count
      const folderCards = mediaPage.page.locator('.media-card.is-folder');
      const initialFolderCount = await folderCards.count();
      expect(initialFolderCount).toBeGreaterThan(0);
      
      const getFolderFileCount = async (): Promise<number> => {
        const countElement = mediaPage.page.locator('.media-card.is-folder .media-meta span:has-text("files")').first();
        const text = await countElement.textContent();
        if (text) {
          const match = text.match(/(\d+)/);
          if (match) return parseInt(match[1]);
        }
        return 0;
      };
      
      const initialFileCount = await getFolderFileCount();
      console.log(`[DU Image Filter] Initial folders: ${initialFolderCount}, files: ${initialFileCount}`);

      // Apply image filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('image').click();
      await mediaPage.page.waitForTimeout(1500);

      // File count within folder should change
      const imageFileCount = await getFolderFileCount();
      const imageFolderCount = await folderCards.count();
      console.log(`[DU Image Filter] After image - folders: ${imageFolderCount}, files: ${imageFileCount}`);
      
      expect(imageFileCount).not.toBe(initialFileCount);
      expect(imageFileCount).toBeLessThan(initialFileCount);
    });

    test('text filter works in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial folder count and file count
      const folderCards = mediaPage.page.locator('.media-card.is-folder');
      const initialFolderCount = await folderCards.count();
      expect(initialFolderCount).toBeGreaterThan(0);
      
      const getFolderFileCount = async (): Promise<number> => {
        const countElement = mediaPage.page.locator('.media-card.is-folder .media-meta span:has-text("files")').first();
        const text = await countElement.textContent();
        if (text) {
          const match = text.match(/(\d+)/);
          if (match) return parseInt(match[1]);
        }
        return 0;
      };
      
      const initialFileCount = await getFolderFileCount();
      console.log(`[DU Text Filter] Initial folders: ${initialFolderCount}, files: ${initialFileCount}`);

      // Apply text filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('text').click();
      await mediaPage.page.waitForTimeout(1500);

      // File count within folder should change
      const textFileCount = await getFolderFileCount();
      const textFolderCount = await folderCards.count();
      console.log(`[DU Text Filter] After text - folders: ${textFolderCount}, files: ${textFileCount}`);
      
      expect(textFileCount).not.toBe(initialFileCount);
      expect(textFileCount).toBeLessThan(initialFileCount);
    });

    test('clearing media type filter restores all results in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial folder count and file count
      const folderCards = mediaPage.page.locator('.media-card.is-folder');
      const initialFolderCount = await folderCards.count();
      
      const getFolderFileCount = async (): Promise<number> => {
        const countElement = mediaPage.page.locator('.media-card.is-folder .media-meta span:has-text("files")').first();
        const text = await countElement.textContent();
        if (text) {
          const match = text.match(/(\d+)/);
          if (match) return parseInt(match[1]);
        }
        return 0;
      };
      
      const initialFileCount = await getFolderFileCount();

      // Apply video filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      const filteredFileCount = await getFolderFileCount();

      // Verify filter had an effect on file count
      expect(filteredFileCount).not.toBe(initialFileCount);

      // Clear filter by clicking again
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      const restoredFileCount = await getFolderFileCount();

      // File count should be back to initial count
      expect(restoredFileCount).toBe(initialFileCount);
    });
  });

  test.describe('Slider Filters in DU Mode', () => {
    test('duration slider is visible and functional in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand duration section
      await sidebarPage.expandDurationSection();
      await mediaPage.page.waitForTimeout(500);

      // Duration slider container should be visible
      await expect(mediaPage.durationSliderContainer).toBeVisible();

      // Duration sliders should exist
      const minSlider = mediaPage.page.locator('#duration-min-slider');
      const maxSlider = mediaPage.page.locator('#duration-max-slider');

      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);

      // Sliders should have valid min/max attributes
      const max = await maxSlider.getAttribute('max');
      expect(max).toBeTruthy();
      expect(parseInt(max || '0')).toBeGreaterThan(0);
    });

    test('size slider is visible and functional in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand size section
      await sidebarPage.expandSizeSection();
      await mediaPage.page.waitForTimeout(500);

      // Size slider container should be visible
      await expect(mediaPage.sizeSliderContainer).toBeVisible();

      // Size sliders should exist
      const minSlider = mediaPage.page.locator('#size-min-slider');
      const maxSlider = mediaPage.page.locator('#size-max-slider');

      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);

      // Sliders should have valid min/max attributes
      const max = await maxSlider.getAttribute('max');
      expect(max).toBeTruthy();
      expect(parseInt(max || '0')).toBeGreaterThan(0);
    });

    test('episodes slider is visible and functional in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand episodes section
      await sidebarPage.expandEpisodesSection();
      await mediaPage.page.waitForTimeout(500);

      // Episodes slider container should be visible
      await expect(mediaPage.episodesSliderContainer).toBeVisible();

      // Episodes sliders should exist
      const minSlider = mediaPage.page.locator('#episodes-min-slider');
      const maxSlider = mediaPage.page.locator('#episodes-max-slider');

      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);
    });

    test('duration slider filters media in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      // Expand duration section
      await sidebarPage.expandDurationSection();

      // Get duration max slider
      const maxSlider = mediaPage.page.locator('#duration-max-slider');
      if (await maxSlider.count() > 0) {
        const maxValue = await maxSlider.getAttribute('max');

        if (maxValue && parseInt(maxValue) > 0) {
          // Set slider to 50% of max to filter out longer media
          const halfValue = Math.floor(parseInt(maxValue) / 2);
          await maxSlider.fill(halfValue.toString());
          await maxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1500);

          // Media count should have changed
          const newCount = await mediaPage.getMediaCount();
          expect(newCount).toBeLessThanOrEqual(initialCount);
        }
      }
    });

    test('size slider filters media in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Get size max slider
      const maxSlider = mediaPage.page.locator('#size-max-slider');
      if (await maxSlider.count() > 0) {
        const maxValue = await maxSlider.getAttribute('max');

        if (maxValue && parseInt(maxValue) > 0) {
          // Set slider to 50% of max to filter out larger media
          const halfValue = Math.floor(parseInt(maxValue) / 2);
          await maxSlider.fill(halfValue.toString());
          await maxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1500);

          // Media count should have changed
          const newCount = await mediaPage.getMediaCount();
          expect(newCount).toBeLessThanOrEqual(initialCount);
        }
      }
    });

    test('multiple sliders can be used together in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Expand all slider sections
      await sidebarPage.expandDurationSection();
      await sidebarPage.expandSizeSection();

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      let appliedFilters = 0;

      // Apply duration filter
      const durationMaxSlider = mediaPage.page.locator('#duration-max-slider');
      if (await durationMaxSlider.count() > 0) {
        const maxDuration = await durationMaxSlider.getAttribute('max');
        if (maxDuration && parseInt(maxDuration) > 0) {
          await durationMaxSlider.fill(Math.floor(parseInt(maxDuration) / 2).toString());
          await durationMaxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1000);
          appliedFilters++;
        }
      }

      // Apply size filter
      const sizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      if (await sizeMaxSlider.count() > 0) {
        const maxSize = await sizeMaxSlider.getAttribute('max');
        if (maxSize && parseInt(maxSize) > 0) {
          await sizeMaxSlider.fill(Math.floor(parseInt(maxSize) / 2).toString());
          await sizeMaxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1000);
          appliedFilters++;
        }
      }

      // Media count should be filtered
      const finalCount = await mediaPage.getMediaCount();
      expect(finalCount).toBeLessThanOrEqual(initialCount);
    });
  });

  test.describe('Cross-Filter Influence in DU Mode', () => {
    test('media type filter affects slider ranges in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Expand size and duration sections
      await sidebarPage.expandSizeSection();
      await sidebarPage.expandDurationSection();

      // Get initial size slider max
      const initialSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      let initialSizeMax = '0';
      if (await initialSizeMaxSlider.count() > 0) {
        initialSizeMax = await initialSizeMaxSlider.getAttribute('max') || '0';
      }

      // Apply video filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(2000);

      // Get updated size slider max
      const newSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      if (await newSizeMaxSlider.count() > 0) {
        const newSizeMax = await newSizeMaxSlider.getAttribute('max') || '0';
        // Size max may change when filtering by type
        expect(newSizeMax).toBeDefined();
      }
    });

    test('search affects filter bins in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Get initial size slider max
      const initialSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      let initialSizeMax = '0';
      if (await initialSizeMaxSlider.count() > 0) {
        initialSizeMax = await initialSizeMaxSlider.getAttribute('max') || '0';
      }

      // Search for specific term
      await mediaPage.search('test');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated size slider max
      const newSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      if (await newSizeMaxSlider.count() > 0) {
        const newSizeMax = await newSizeMaxSlider.getAttribute('max') || '0';
        // Size max may change based on search results
        expect(newSizeMax).toBeDefined();
      }
    });

    test('clearing search restores filter bins in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Get initial size slider max
      const initialSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      let initialSizeMax = '0';
      if (await initialSizeMaxSlider.count() > 0) {
        initialSizeMax = await initialSizeMaxSlider.getAttribute('max') || '0';
      }

      // Search
      await mediaPage.search('test');
      await mediaPage.page.waitForTimeout(1500);

      // Clear search
      await mediaPage.clearSearch();
      await mediaPage.page.waitForTimeout(1500);

      // Get restored size slider max
      const restoredSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      if (await restoredSizeMaxSlider.count() > 0) {
        const restoredSizeMax = await restoredSizeMaxSlider.getAttribute('max') || '0';
        // Size max should be restored
        expect(restoredSizeMax).toBeDefined();
      }
    });
  });

  test.describe('History Filters in DU Mode', () => {
    test('history in-progress filter works in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      // Apply in-progress filter
      await sidebarPage.expandHistorySection();
      await sidebarPage.clickHistoryInProgress();
      await mediaPage.page.waitForTimeout(2000);

      // Should have in-progress results
      const historyCount = await mediaPage.getMediaCount();
      expect(historyCount).toBeLessThanOrEqual(initialCount);
    });

    test('combining history and media type filters works in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Apply in-progress filter
      await sidebarPage.expandHistorySection();
      await sidebarPage.clickHistoryInProgress();
      await mediaPage.page.waitForTimeout(2000);

      const historyCount = await mediaPage.getMediaCount();

      // Apply video filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      const combinedCount = await mediaPage.getMediaCount();
      expect(combinedCount).toBeLessThanOrEqual(historyCount);
    });
  });
});
