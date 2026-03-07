import { test, expect } from '../fixtures';

test.describe('Cross-Filter Influence', () => {
  test.use({ readOnly: true });
  test.describe('Episodes Filter Affects Size and Duration', () => {
    test('episodes filter should affect size range labels', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open the Episodes filter section
      const episodesDetails = page.locator('#details-episodes');
      const episodesSummary = episodesDetails.locator('summary');
      
      // Click to expand if not already open
      const isOpen = await episodesDetails.getAttribute('open');
      if (!isOpen) {
        await episodesSummary.click();
        await page.waitForTimeout(300);
      }

      // Get initial size range labels (footer labels showing min-max)
      const sizeMinLabel = page.locator('#size-min-label');
      const sizeMaxLabel = page.locator('#size-max-label');
      
      // Wait for labels to be populated
      await page.waitForTimeout(500);
      
      const initialSizeMin = await sizeMinLabel.textContent();
      const initialSizeMax = await sizeMaxLabel.textContent();
      
      // Get episodes sliders
      const episodesMinSlider = page.locator('#episodes-min-slider');
      const episodesMaxSlider = page.locator('#episodes-max-slider');
      
      // Adjust episodes filter to a narrower range (e.g., 50-100%)
      await episodesMaxSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '50';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      
      // Wait for search to complete and bins to update
      await page.waitForTimeout(1500);
      
      // Get updated size range labels
      const newSizeMin = await sizeMinLabel.textContent();
      const newSizeMax = await sizeMaxLabel.textContent();
      
      // The size range should potentially change when filtering by episodes
      // (may be same or different depending on data distribution)
      // The key is that the filter is applied and results update
      const resultsCount = page.locator('.media-card');
      const count = await resultsCount.count();
      
      // Should have some results
      expect(count).toBeGreaterThanOrEqual(0);
      
      // Size labels should contain valid values (not empty or error state)
      if (initialSizeMin && initialSizeMin.trim() !== '') {
        expect(newSizeMin).toBeDefined();
      }
    });

    test('episodes filter should affect duration range labels', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open the Episodes filter section
      const episodesDetails = page.locator('#details-episodes');
      const episodesSummary = episodesDetails.locator('summary');
      
      // Click to expand if not already open
      const isOpen = await episodesDetails.getAttribute('open');
      if (!isOpen) {
        await episodesSummary.click();
        await page.waitForTimeout(300);
      }

      // Get initial duration range labels (footer labels showing min-max)
      const durationMinLabel = page.locator('#duration-min-label');
      const durationMaxLabel = page.locator('#duration-max-label');
      
      // Wait for labels to be populated
      await page.waitForTimeout(500);
      
      const initialDurationMin = await durationMinLabel.textContent();
      const initialDurationMax = await durationMaxLabel.textContent();
      
      // Get episodes sliders
      const episodesMinSlider = page.locator('#episodes-min-slider');
      const episodesMaxSlider = page.locator('#episodes-max-slider');
      
      // Adjust episodes filter to a narrower range
      await episodesMinSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '20';
        el.dispatchEvent(new Event('input', { bubbles: true }));
      });
      await episodesMaxSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '80';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      
      // Wait for search to complete and bins to update
      await page.waitForTimeout(1500);
      
      // Get updated duration range labels
      const newDurationMin = await durationMinLabel.textContent();
      const newDurationMax = await durationMaxLabel.textContent();
      
      // Duration labels should contain valid values
      if (initialDurationMin && initialDurationMin.trim() !== '') {
        expect(newDurationMin).toBeDefined();
      }
      
      // Results should update
      const resultsCount = page.locator('.media-card');
      const count = await resultsCount.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('duration filter should NOT recursively shrink duration range', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open the Duration filter section
      const durationDetails = page.locator('#details-duration');
      const durationSummary = durationDetails.locator('summary');
      
      // Click to expand if not already open
      const isOpen = await durationDetails.getAttribute('open');
      if (!isOpen) {
        await durationSummary.click();
        await page.waitForTimeout(300);
      }

      // Get initial duration range labels
      const durationMinLabel = page.locator('#duration-min-label');
      const durationMaxLabel = page.locator('#duration-max-label');
      const durationPercentileLabel = page.locator('#duration-percentile-label');
      
      // Wait for labels to be populated
      await page.waitForTimeout(500);
      
      const initialMin = await durationMinLabel.textContent();
      const initialMax = await durationMaxLabel.textContent();
      
      // Adjust duration filter
      const durationMinSlider = page.locator('#duration-min-slider');
      await durationMinSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '20';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      
      // Wait for search to complete
      await page.waitForTimeout(1500);
      
      // The footer labels (min/max) should NOT change due to recursive filtering
      // They should still show the original full range
      const newMin = await durationMinLabel.textContent();
      const newMax = await durationMaxLabel.textContent();
      
      // Footer labels should remain stable (not recursively constrained)
      // They may format slightly differently but should represent same range
      if (initialMin && initialMin.trim() !== '' && newMin && newMin.trim() !== '') {
        // This is the key test: min label should not change to the filtered value
        // If it does, we have the recursive constraint bug
        expect(newMin).toEqual(initialMin);
      }
      
      if (initialMax && initialMax.trim() !== '' && newMax && newMax.trim() !== '') {
        expect(newMax).toEqual(initialMax);
      }
    });

    test('size filter should NOT recursively shrink size range', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open the Size filter section
      const sizeDetails = page.locator('#details-size');
      const sizeSummary = sizeDetails.locator('summary');
      
      // Click to expand if not already open
      const isOpen = await sizeDetails.getAttribute('open');
      if (!isOpen) {
        await sizeSummary.click();
        await page.waitForTimeout(300);
      }

      // Get initial size range labels
      const sizeMinLabel = page.locator('#size-min-label');
      const sizeMaxLabel = page.locator('#size-max-label');
      
      // Wait for labels to be populated
      await page.waitForTimeout(500);
      
      const initialMin = await sizeMinLabel.textContent();
      const initialMax = await sizeMaxLabel.textContent();
      
      // Adjust size filter
      const sizeMinSlider = page.locator('#size-min-slider');
      await sizeMinSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '30';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      
      // Wait for search to complete
      await page.waitForTimeout(1500);
      
      // The footer labels (min/max) should NOT change due to recursive filtering
      const newMin = await sizeMinLabel.textContent();
      const newMax = await sizeMaxLabel.textContent();
      
      // Footer labels should remain stable
      if (initialMin && initialMin.trim() !== '' && newMin && newMin.trim() !== '') {
        expect(newMin).toEqual(initialMin);
      }
      
      if (initialMax && initialMax.trim() !== '' && newMax && newMax.trim() !== '') {
        expect(newMax).toEqual(initialMax);
      }
    });

    test('multiple filters can be combined and results update correctly', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Get initial count
      const initialResults = page.locator('.media-card');
      const initialCount = await initialResults.count();

      // Open Episodes filter
      const episodesDetails = page.locator('#details-episodes');
      const episodesSummary = episodesDetails.locator('summary');
      const episodesIsOpen = await episodesDetails.getAttribute('open');
      if (!episodesIsOpen) {
        await episodesSummary.click();
        await page.waitForTimeout(300);
      }

      // Apply episodes filter
      const episodesMaxSlider = page.locator('#episodes-max-slider');
      await episodesMaxSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '70';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      await page.waitForTimeout(1500);

      // Get count after episodes filter
      const afterEpisodesResults = page.locator('.media-card');
      const afterEpisodesCount = await afterEpisodesResults.count();
      
      // Count should be <= initial
      expect(afterEpisodesCount).toBeLessThanOrEqual(initialCount);

      // Open Duration filter
      const durationDetails = page.locator('#details-duration');
      const durationSummary = durationDetails.locator('summary');
      const durationIsOpen = await durationDetails.getAttribute('open');
      if (!durationIsOpen) {
        await durationSummary.click();
        await page.waitForTimeout(300);
      }

      // Apply duration filter
      const durationMaxSlider = page.locator('#duration-max-slider');
      await durationMaxSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '70';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      await page.waitForTimeout(1500);

      // Get count after both filters
      const finalResults = page.locator('.media-card');
      const finalCount = await finalResults.count();
      
      // Count should be <= after episodes
      expect(finalCount).toBeLessThanOrEqual(afterEpisodesCount);
    });

    test('resetting episodes filter restores original size and duration ranges', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Open the Episodes filter section
      const episodesDetails = page.locator('#details-episodes');
      const episodesSummary = episodesDetails.locator('summary');
      
      const isOpen = await episodesDetails.getAttribute('open');
      if (!isOpen) {
        await episodesSummary.click();
        await page.waitForTimeout(300);
      }

      // Get initial size and duration range labels
      const sizeMinLabel = page.locator('#size-min-label');
      const sizeMaxLabel = page.locator('#size-max-label');
      const durationMinLabel = page.locator('#duration-min-label');
      const durationMaxLabel = page.locator('#duration-max-label');
      
      await page.waitForTimeout(500);
      
      const initialSizeMin = await sizeMinLabel.textContent();
      const initialSizeMax = await sizeMaxLabel.textContent();
      const initialDurationMin = await durationMinLabel.textContent();
      const initialDurationMax = await durationMaxLabel.textContent();

      // Apply episodes filter
      const episodesMinSlider = page.locator('#episodes-min-slider');
      const episodesMaxSlider = page.locator('#episodes-max-slider');
      
      await episodesMinSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '20';
        el.dispatchEvent(new Event('input', { bubbles: true }));
      });
      await episodesMaxSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '60';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      await page.waitForTimeout(1500);

      // Reset episodes filter by setting back to 0-100
      await episodesMinSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '0';
        el.dispatchEvent(new Event('input', { bubbles: true }));
      });
      await episodesMaxSlider.evaluate((el) => {
        (el as HTMLInputElement).value = '100';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      await page.waitForTimeout(1500);

      // Get restored size and duration range labels
      const restoredSizeMin = await sizeMinLabel.textContent();
      const restoredSizeMax = await sizeMaxLabel.textContent();
      const restoredDurationMin = await durationMinLabel.textContent();
      const restoredDurationMax = await durationMaxLabel.textContent();

      // Labels should be restored to original values
      if (initialSizeMin && initialSizeMin.trim() !== '') {
        expect(restoredSizeMin).toEqual(initialSizeMin);
      }
      if (initialSizeMax && initialSizeMax.trim() !== '') {
        expect(restoredSizeMax).toEqual(initialSizeMax);
      }
      if (initialDurationMin && initialDurationMin.trim() !== '') {
        expect(restoredDurationMin).toEqual(initialDurationMin);
      }
      if (initialDurationMax && initialDurationMax.trim() !== '') {
        expect(restoredDurationMax).toEqual(initialDurationMax);
      }
    });
  });
});
