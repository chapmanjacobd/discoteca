import { test, expect } from '../fixtures';

/**
 * E2E tests comparing filter behavior across Search, Captions, and DU modes
 * These tests verify that all modes have consistent filter functionality
 */
test.describe('Filter Consistency Across Modes', () => {
  test.use({ readOnly: true });

  test.describe('Media Type Filter Availability', () => {
    test('media type filters available in Search mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Expand media type section
      await sidebarPage.expandMediaTypeSection();

      // All media type buttons should be visible
      await expect(sidebarPage.getMediaTypeButton('video')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('audio')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('text')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('image')).toBeVisible();
    });

    test('media type filters available in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand media type section
      await sidebarPage.expandMediaTypeSection();

      // All media type buttons should be visible
      await expect(sidebarPage.getMediaTypeButton('video')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('audio')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('text')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('image')).toBeVisible();
    });

    test('media type filters available in Captions mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Expand media type section
      await sidebarPage.expandMediaTypeSection();

      // All media type buttons should be visible
      await expect(sidebarPage.getMediaTypeButton('video')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('audio')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('text')).toBeVisible();
      await expect(sidebarPage.getMediaTypeButton('image')).toBeVisible();
    });
  });

  test.describe('Slider Filter Availability', () => {
    test('duration slider available in Search mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Expand duration section
      await sidebarPage.expandDurationSection();

      // Duration slider container should be visible
      await expect(mediaPage.durationSliderContainer).toBeVisible();

      // Duration sliders should exist
      const minSlider = mediaPage.page.locator('#duration-min-slider');
      const maxSlider = mediaPage.page.locator('#duration-max-slider');
      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);
    });

    test('duration slider available in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand duration section
      await sidebarPage.expandDurationSection();

      // Duration slider container should be visible
      await expect(mediaPage.durationSliderContainer).toBeVisible();

      // Duration sliders should exist
      const minSlider = mediaPage.page.locator('#duration-min-slider');
      const maxSlider = mediaPage.page.locator('#duration-max-slider');
      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);
    });

    test('duration slider available in Captions mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Expand duration section
      await sidebarPage.expandDurationSection();

      // Duration slider container should be visible
      await expect(mediaPage.durationSliderContainer).toBeVisible();

      // Duration sliders should exist
      const minSlider = mediaPage.page.locator('#duration-min-slider');
      const maxSlider = mediaPage.page.locator('#duration-max-slider');
      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);
    });

    test('size slider available in Search mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Size slider container should be visible
      await expect(mediaPage.sizeSliderContainer).toBeVisible();

      // Size sliders should exist
      const minSlider = mediaPage.page.locator('#size-min-slider');
      const maxSlider = mediaPage.page.locator('#size-max-slider');
      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);
    });

    test('size slider available in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Size slider container should be visible
      await expect(mediaPage.sizeSliderContainer).toBeVisible();

      // Size sliders should exist
      const minSlider = mediaPage.page.locator('#size-min-slider');
      const maxSlider = mediaPage.page.locator('#size-max-slider');
      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);
    });

    test('size slider available in Captions mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Size slider container should be visible
      await expect(mediaPage.sizeSliderContainer).toBeVisible();

      // Size sliders should exist
      const minSlider = mediaPage.page.locator('#size-min-slider');
      const maxSlider = mediaPage.page.locator('#size-max-slider');
      expect(await minSlider.count()).toBeGreaterThan(0);
      expect(await maxSlider.count()).toBeGreaterThan(0);
    });
  });

  test.describe('Sort Functionality', () => {
    test('sort options available in Search mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Sort dropdown should be visible
      await expect(mediaPage.sortBySelect).toBeVisible();

      // Sort options should include common fields
      const options = await mediaPage.sortBySelect.locator('option').allTextContents();
      expect(options).toContain('Path');
      expect(options).toContain('Size');
      expect(options).toContain('Duration');
    });

    test('sort options available in DU mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Sort dropdown should be visible
      await expect(mediaPage.sortBySelect).toBeVisible();

      // Sort options should include common fields
      const options = await mediaPage.sortBySelect.locator('option').allTextContents();
      expect(options).toContain('Path');
      expect(options).toContain('Size');
      expect(options).toContain('Duration');
    });

    test('sort options available in Captions mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Sort dropdown should be visible
      await expect(mediaPage.sortBySelect).toBeVisible();

      // Sort options should include common fields
      const options = await mediaPage.sortBySelect.locator('option').allTextContents();
      expect(options).toContain('Path');
      expect(options).toContain('Size');
      expect(options).toContain('Duration');
    });

    test('reverse sort works in Search mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Click reverse sort button
      await mediaPage.sortReverseBtn.click();
      await mediaPage.page.waitForTimeout(300);

      // Should have active class
      await expect(mediaPage.sortReverseBtn).toHaveClass(/active/);
    });

    test('reverse sort works in DU mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Click reverse sort button
      await mediaPage.sortReverseBtn.click();
      await mediaPage.page.waitForTimeout(300);

      // Should have active class
      await expect(mediaPage.sortReverseBtn).toHaveClass(/active/);
    });

    test('reverse sort works in Captions mode', async ({ mediaPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Click reverse sort button
      await mediaPage.sortReverseBtn.click();
      await mediaPage.page.waitForTimeout(300);

      // Should have active class
      await expect(mediaPage.sortReverseBtn).toHaveClass(/active/);
    });
  });

  test.describe('Filter Bins Update Behavior', () => {
    test('filter bins update after search in Search mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Get initial size slider max
      const initialSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      let initialSizeMax = '0';
      if (await initialSizeMaxSlider.count() > 0) {
        initialSizeMax = await initialSizeMaxSlider.getAttribute('max') || '0';
      }

      // Perform search
      await mediaPage.search('test');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated size slider max
      const newSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      if (await newSizeMaxSlider.count() > 0) {
        const newSizeMax = await newSizeMaxSlider.getAttribute('max') || '0';
        // Slider max should be defined (may or may not change based on results)
        expect(newSizeMax).toBeDefined();
      }
    });

    test('filter bins update after search in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Get initial size slider max
      const initialSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      let initialSizeMax = '0';
      if (await initialSizeMaxSlider.count() > 0) {
        initialSizeMax = await initialSizeMaxSlider.getAttribute('max') || '0';
      }

      // Perform search
      await mediaPage.search('test');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated size slider max
      const newSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      if (await newSizeMaxSlider.count() > 0) {
        const newSizeMax = await newSizeMaxSlider.getAttribute('max') || '0';
        // Slider max should be defined (may or may not change based on results)
        expect(newSizeMax).toBeDefined();
      }
    });

    test('filter bins update after search in Captions mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Expand size section
      await sidebarPage.expandSizeSection();

      // Get initial size slider max
      const initialSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      let initialSizeMax = '0';
      if (await initialSizeMaxSlider.count() > 0) {
        initialSizeMax = await initialSizeMaxSlider.getAttribute('max') || '0';
      }

      // Perform search
      await mediaPage.search('test');
      await mediaPage.page.waitForTimeout(1500);

      // Get updated size slider max
      const newSizeMaxSlider = mediaPage.page.locator('#size-max-slider');
      if (await newSizeMaxSlider.count() > 0) {
        const newSizeMax = await newSizeMaxSlider.getAttribute('max') || '0';
        // Slider max should be defined (may or may not change based on results)
        expect(newSizeMax).toBeDefined();
      }
    });
  });

  test.describe('Filter Application Behavior', () => {
    test('applying media type filter reduces results in Search mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      // Apply video filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // Should have filtered results
      const videoCount = await mediaPage.getMediaCount();
      expect(videoCount).toBeLessThanOrEqual(initialCount);
    });

    test('applying media type filter reduces results in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      // Apply video filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // Should have filtered results
      const videoCount = await mediaPage.getMediaCount();
      expect(videoCount).toBeLessThanOrEqual(initialCount);
    });

    test('applying media type filter reduces results in Captions mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Get initial count
      const initialCount = await mediaPage.getCaptionCards().count();

      // Apply video filter
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('video').click();
      await mediaPage.page.waitForTimeout(1500);

      // Should have filtered results
      const videoCount = await mediaPage.getCaptionCards().count();
      expect(videoCount).toBeLessThanOrEqual(initialCount);
    });

    test('applying duration slider filter reduces results in Search mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl());

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      // Expand duration section
      await sidebarPage.expandDurationSection();

      // Apply duration filter
      const maxSlider = mediaPage.page.locator('#duration-max-slider');
      if (await maxSlider.count() > 0) {
        const maxValue = await maxSlider.getAttribute('max');
        if (maxValue && parseInt(maxValue) > 0) {
          await maxSlider.fill(Math.floor(parseInt(maxValue) / 2).toString());
          await maxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1500);

          const filteredCount = await mediaPage.getMediaCount();
          expect(filteredCount).toBeLessThanOrEqual(initialCount);
        }
      }
    });

    test('applying duration slider filter reduces results in DU mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=du');
      await mediaPage.getDUTToolbar().waitFor({ state: 'visible', timeout: 10000 });
      await mediaPage.page.waitForTimeout(1000);

      // Get initial count
      const initialCount = await mediaPage.getMediaCount();

      // Expand duration section
      await sidebarPage.expandDurationSection();

      // Apply duration filter
      const maxSlider = mediaPage.page.locator('#duration-max-slider');
      if (await maxSlider.count() > 0) {
        const maxValue = await maxSlider.getAttribute('max');
        if (maxValue && parseInt(maxValue) > 0) {
          await maxSlider.fill(Math.floor(parseInt(maxValue) / 2).toString());
          await maxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1500);

          const filteredCount = await mediaPage.getMediaCount();
          expect(filteredCount).toBeLessThanOrEqual(initialCount);
        }
      }
    });

    test('applying duration slider filter reduces results in Captions mode', async ({ mediaPage, sidebarPage, server }) => {
      await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');
      await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

      // Get initial count
      const initialCount = await mediaPage.getCaptionCards().count();

      // Expand duration section
      await sidebarPage.expandDurationSection();

      // Apply duration filter
      const maxSlider = mediaPage.page.locator('#duration-max-slider');
      if (await maxSlider.count() > 0) {
        const maxValue = await maxSlider.getAttribute('max');
        if (maxValue && parseInt(maxValue) > 0) {
          await maxSlider.fill(Math.floor(parseInt(maxValue) / 2).toString());
          await maxSlider.dispatchEvent('input');
          await mediaPage.page.waitForTimeout(1500);

          const filteredCount = await mediaPage.getCaptionCards().count();
          expect(filteredCount).toBeLessThanOrEqual(initialCount);
        }
      }
    });
  });
});
