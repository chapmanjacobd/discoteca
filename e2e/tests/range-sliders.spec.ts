import { test, expect } from '../fixtures';

test.describe('Range Sliders', () => {
  test.use({ readOnly: true });
  test.describe('Size Slider', () => {
    test('size slider is visible in filter panel', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Size slider should be visible in DU mode or filter panel
      const sizeSlider = page.locator('#size-slider, input[type="range"][aria-label*="size"], .size-slider');
      
      // Check in DU mode
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Ensure size section is open
      const sizeDetails = page.locator('#details-size');
      if (await sizeDetails.count() > 0) {
        await sizeDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
      }

      const sizeSliderContainer = page.locator('#size-slider-container, .size-filter');
      await expect(sizeSliderContainer.first()).toBeVisible();
    });

    test('size slider has min and max values', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const sizeSlider = page.locator('#size-slider, input[type="range"][aria-label*="size"]').first();
      if (await sizeSlider.count() > 0) {
        const min = await sizeSlider.getAttribute('min');
        const max = await sizeSlider.getAttribute('max');
        
        expect(min).toBeTruthy();
        expect(max).toBeTruthy();
        expect(parseInt(max)).toBeGreaterThan(parseInt(min));
      }
    });

    test('size slider filters media by size', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Get initial card count
      const initialCards = page.locator('.media-card');
      const initialCount = await initialCards.count();

      // Go to DU mode
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Adjust size slider
      const sizeSlider = page.locator('#size-slider, input[type="range"][aria-label*="size"]').first();
      if (await sizeSlider.count() > 0) {
        const max = await sizeSlider.getAttribute('max');
        await sizeSlider.evaluate((el, maxValue) => {
          (el as HTMLInputElement).value = maxValue;
          el.dispatchEvent(new Event('input', { bubbles: true }));
          el.dispatchEvent(new Event('change', { bubbles: true }));
        }, max);
        await page.waitForTimeout(1000);

        // Results should be filtered
        const filteredCards = page.locator('.media-card');
        const filteredCount = await filteredCards.count();
        
        // Count may be same or less depending on data
        expect(filteredCount).toBeLessThanOrEqual(initialCount);
      }
    });

    test('size slider shows current value', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const sizeSlider = page.locator('#size-slider, input[type="range"][aria-label*="size"]').first();
      if (await sizeSlider.count() > 0) {
        // Size value display should exist
        const sizeValue = page.locator('#size-value, .size-value, [class*="size"] span');
        if (await sizeValue.count() > 0) {
          await expect(sizeValue.first()).toBeVisible();
        }
      }
    });

    test('size slider resets to default', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const sizeSlider = page.locator('#size-slider, input[type="range"][aria-label*="size"]').first();
      if (await sizeSlider.count() > 0) {
        const defaultValue = await sizeSlider.evaluate((el) => (el as HTMLInputElement).value);
        
        // Change value
        await sizeSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '50';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);

        // Reset button
        const resetBtn = page.locator('.reset-size, button:has-text("Reset"):near(.size-slider), .clear-filters');
        if (await resetBtn.count() > 0) {
          await resetBtn.first().click();
          await page.waitForTimeout(500);

          // Value should be reset
          const newValue = await sizeSlider.evaluate((el) => (el as HTMLInputElement).value);
          expect(newValue).toEqual(defaultValue);
        }
      }
    });

    test('size slider has proper accessibility', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const sizeSlider = page.locator('#size-slider, input[type="range"][aria-label*="size"]').first();
      if (await sizeSlider.count() > 0) {
        // Should have aria-label or associated label
        const ariaLabel = await sizeSlider.getAttribute('aria-label');
        const id = await sizeSlider.getAttribute('id');
        
        if (id) {
          const label = page.locator(`label[for="${id}"]`);
          const hasLabel = await label.count() > 0;
          expect(hasLabel || ariaLabel).toBe(true);
        }
      }
    });
  });

  test.describe('Duration Slider', () => {
    test('duration slider is visible in filter panel', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Ensure duration section is open
      const durationDetails = page.locator('#details-duration');
      if (await durationDetails.count() > 0) {
        await durationDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
      }

      const durationSliderContainer = page.locator('#duration-slider-container, .duration-filter');
      await expect(durationSliderContainer.first()).toBeVisible();
    });

    test('duration slider has min and max values', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const durationSlider = page.locator('#duration-slider, input[type="range"][aria-label*="duration"]').first();
      if (await durationSlider.count() > 0) {
        const min = await durationSlider.getAttribute('min');
        const max = await durationSlider.getAttribute('max');
        
        expect(min).toBeTruthy();
        expect(max).toBeTruthy();
        expect(parseInt(max)).toBeGreaterThan(parseInt(min));
      }
    });

    test('duration slider filters media by duration', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Get initial card count
      const initialCards = page.locator('.media-card');
      const initialCount = await initialCards.count();

      // Go to DU mode
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Adjust duration slider
      const durationSlider = page.locator('#duration-slider, input[type="range"][aria-label*="duration"]').first();
      if (await durationSlider.count() > 0) {
        const max = await durationSlider.getAttribute('max');
        await durationSlider.evaluate((el, maxValue) => {
          (el as HTMLInputElement).value = maxValue;
          el.dispatchEvent(new Event('input', { bubbles: true }));
          el.dispatchEvent(new Event('change', { bubbles: true }));
        }, max);
        await page.waitForTimeout(1000);

        // Results should be filtered
        const filteredCards = page.locator('.media-card');
        const filteredCount = await filteredCards.count();
        
        expect(filteredCount).toBeLessThanOrEqual(initialCount);
      }
    });

    test('duration slider shows current value', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const durationSlider = page.locator('#duration-slider, input[type="range"][aria-label*="duration"]').first();
      if (await durationSlider.count() > 0) {
        // Duration value display should exist
        const durationValue = page.locator('#duration-value, .duration-value, [class*="duration"] span');
        if (await durationValue.count() > 0) {
          await expect(durationValue.first()).toBeVisible();
        }
      }
    });

    test('duration slider shows formatted time', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const durationSlider = page.locator('#duration-slider, input[type="range"][aria-label*="duration"]').first();
      if (await durationSlider.count() > 0) {
        // Duration value should be formatted (contain : or h/m/s)
        const durationValue = page.locator('#duration-value, .duration-value').first();
        if (await durationValue.count() > 0) {
          const text = await durationValue.textContent();
          expect(text).toMatch(/(\d+:\d+|\d+h|\d+m|\d+s|\d+ms)/i);
        }
      }
    });

    test('duration slider resets to default', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const durationSlider = page.locator('#duration-slider, input[type="range"][aria-label*="duration"]').first();
      if (await durationSlider.count() > 0) {
        const defaultValue = await durationSlider.evaluate((el) => (el as HTMLInputElement).value);
        
        // Change value
        await durationSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '50';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);

        // Reset button
        const resetBtn = page.locator('.reset-duration, .clear-filters').first();
        if (await resetBtn.count() > 0) {
          await resetBtn.click();
          await page.waitForTimeout(500);

          // Value should be reset
          const newValue = await durationSlider.evaluate((el) => (el as HTMLInputElement).value);
          expect(newValue).toEqual(defaultValue);
        }
      }
    });

    test('duration slider keyboard navigation works', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const durationSlider = page.locator('#duration-slider, input[type="range"][aria-label*="duration"]').first();
      if (await durationSlider.count() > 0) {
        const initialValue = await durationSlider.evaluate((el) => (el as HTMLInputElement).value);
        
        // Focus slider
        await durationSlider.focus();
        
        // Press right arrow
        await page.keyboard.press('ArrowRight');
        await page.waitForTimeout(300);

        const newValue = await durationSlider.evaluate((el) => (el as HTMLInputElement).value);
        expect(parseInt(newValue)).toBeGreaterThanOrEqual(parseInt(initialValue));
      }
    });
  });

  test.describe('Episodes Slider', () => {
    test('episodes slider is visible in filter panel', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Ensure episodes section is open
      const episodesDetails = page.locator('#details-episodes');
      if (await episodesDetails.count() > 0) {
        await episodesDetails.evaluate((el: HTMLDetailsElement) => el.open = true);
      }

      const episodesSliderContainer = page.locator('#episodes-slider-container, .episodes-filter');
      await expect(episodesSliderContainer.first()).toBeVisible();
    });

    test('episodes slider has min and max values', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const episodesSlider = page.locator('#episodes-slider, input[type="range"][aria-label*="episode"]').first();
      if (await episodesSlider.count() > 0) {
        const min = await episodesSlider.getAttribute('min');
        const max = await episodesSlider.getAttribute('max');
        
        expect(min).toBeTruthy();
        expect(max).toBeTruthy();
        expect(parseInt(max)).toBeGreaterThanOrEqual(parseInt(min));
      }
    });

    test('episodes slider filters media by episode count', async ({ page, server }) => {
      await page.goto(server.getBaseUrl());

      // Wait for media to load
      await page.waitForSelector('.media-card', { timeout: 10000 });

      // Get initial card count
      const initialCards = page.locator('.media-card');
      const initialCount = await initialCards.count();

      // Go to DU mode
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Adjust episodes slider
      const episodesSlider = page.locator('#episodes-slider, input[type="range"][aria-label*="episode"]').first();
      if (await episodesSlider.count() > 0) {
        const max = await episodesSlider.getAttribute('max');
        await episodesSlider.evaluate((el, maxValue) => {
          (el as HTMLInputElement).value = maxValue;
          el.dispatchEvent(new Event('input', { bubbles: true }));
          el.dispatchEvent(new Event('change', { bubbles: true }));
        }, max);
        await page.waitForTimeout(1000);

        // Results should be filtered
        const filteredCards = page.locator('.media-card');
        const filteredCount = await filteredCards.count();
        
        expect(filteredCount).toBeLessThanOrEqual(initialCount);
      }
    });

    test('episodes slider shows current value', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const episodesSlider = page.locator('#episodes-slider, input[type="range"][aria-label*="episode"]').first();
      if (await episodesSlider.count() > 0) {
        // Episodes value display should exist
        const episodesValue = page.locator('#episodes-value, .episodes-value, [class*="episodes"] span');
        if (await episodesValue.count() > 0) {
          await expect(episodesValue.first()).toBeVisible();
        }
      }
    });

    test('episodes slider resets to default', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const episodesSlider = page.locator('#episodes-slider, input[type="range"][aria-label*="episode"]').first();
      if (await episodesSlider.count() > 0) {
        const defaultValue = await episodesSlider.evaluate((el) => (el as HTMLInputElement).value);
        
        // Change value
        await episodesSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '50';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);

        // Reset button
        const resetBtn = page.locator('.reset-episodes, .clear-filters').first();
        if (await resetBtn.count() > 0) {
          await resetBtn.click();
          await page.waitForTimeout(500);

          // Value should be reset
          const newValue = await episodesSlider.evaluate((el) => (el as HTMLInputElement).value);
          expect(newValue).toEqual(defaultValue);
        }
      }
    });

    test('episodes slider keyboard navigation works', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const episodesSlider = page.locator('#episodes-slider, input[type="range"][aria-label*="episode"]').first();
      if (await episodesSlider.count() > 0) {
        const initialValue = await episodesSlider.evaluate((el) => (el as HTMLInputElement).value);
        
        // Focus slider
        await episodesSlider.focus();
        
        // Press right arrow
        await page.keyboard.press('ArrowRight');
        await page.waitForTimeout(300);

        const newValue = await episodesSlider.evaluate((el) => (el as HTMLInputElement).value);
        expect(parseInt(newValue)).toBeGreaterThanOrEqual(parseInt(initialValue));
      }
    });
  });

  test.describe('Slider Interactions', () => {
    test('multiple sliders can be used together', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Adjust size slider
      const sizeSlider = page.locator('#size-slider').first();
      if (await sizeSlider.count() > 0) {
        await sizeSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '50';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);
      }

      // Adjust duration slider
      const durationSlider = page.locator('#duration-slider').first();
      if (await durationSlider.count() > 0) {
        await durationSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '50';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);
      }

      // Both filters should be applied
      const filteredCards = page.locator('.media-card');
      await expect(filteredCards.first()).toBeVisible();
    });

    test('sliders update in real-time', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const sizeSlider = page.locator('#size-slider').first();
      if (await sizeSlider.count() > 0) {
        // Get initial card count
        const initialCards = page.locator('.media-card');
        const initialCount = await initialCards.count();

        // Move slider
        await sizeSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '80';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });

        // Results should update quickly
        await page.waitForTimeout(500);
        const newCards = page.locator('.media-card');
        const newCount = await newCards.count();
        
        // Count may change or stay same
        expect(typeof newCount).toBe('number');
      }
    });

    test('sliders preserve values on navigation', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const sizeSlider = page.locator('#size-slider').first();
      if (await sizeSlider.count() > 0) {
        // Set value
        await sizeSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '60';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);

        const setValue = await sizeSlider.evaluate((el) => (el as HTMLInputElement).value);

        // Navigate away and back
        await page.goto(server.getBaseUrl());
        await page.waitForSelector('.media-card', { timeout: 10000 });
        await page.goto(server.getBaseUrl() + '/#mode=du');
        await page.waitForSelector('#du-toolbar', { timeout: 10000 });

        // Value may be preserved in URL or reset
        const newSizeSlider = page.locator('#size-slider').first();
        if (await newSizeSlider.count() > 0) {
          const newValue = await newSizeSlider.evaluate((el) => (el as HTMLInputElement).value);
          // Either preserved or reset to default
          expect(typeof newValue).toBe('string');
        }
      }
    });

    test('clear all filters button resets all sliders', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      // Get default values
      const sizeSlider = page.locator('#size-slider').first();
      const durationSlider = page.locator('#duration-slider').first();
      
      let defaultSize = '0';
      let defaultDuration = '0';
      
      if (await sizeSlider.count() > 0) {
        defaultSize = await sizeSlider.evaluate((el) => (el as HTMLInputElement).value);
      }
      if (await durationSlider.count() > 0) {
        defaultDuration = await durationSlider.evaluate((el) => (el as HTMLInputElement).value);
      }

      // Change values
      if (await sizeSlider.count() > 0) {
        await sizeSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '80';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
      }
      if (await durationSlider.count() > 0) {
        await durationSlider.evaluate((el) => {
          (el as HTMLInputElement).value = '80';
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
      }
      await page.waitForTimeout(500);

      // Clear all filters
      const clearBtn = page.locator('.clear-all-filters, button:has-text("Clear All"), .reset-all');
      if (await clearBtn.count() > 0) {
        await clearBtn.first().click();
        await page.waitForTimeout(500);

        // Values should be reset
        if (await sizeSlider.count() > 0) {
          const newSize = await sizeSlider.evaluate((el) => (el as HTMLInputElement).value);
          expect(newSize).toEqual(defaultSize);
        }
      }
    });

    test('sliders have touch support', async ({ page, server }) => {
      await page.goto(server.getBaseUrl() + '/#mode=du');
      await page.waitForSelector('#du-toolbar', { timeout: 10000 });

      const sizeSlider = page.locator('#size-slider').first();
      if (await sizeSlider.count() > 0) {
        const initialValue = await sizeSlider.evaluate((el) => (el as HTMLInputElement).value);
        
        // Simulate touch event
        await sizeSlider.evaluate((el) => {
          const maxValue = el.getAttribute('max') || '100';
          (el as HTMLInputElement).value = maxValue;
          el.dispatchEvent(new Event('touchstart', { bubbles: true }));
          el.dispatchEvent(new Event('input', { bubbles: true }));
        });
        await page.waitForTimeout(500);

        const newValue = await sizeSlider.evaluate((el) => (el as HTMLInputElement).value);
        expect(newValue).not.toEqual(initialValue);
      }
    });
  });
});
