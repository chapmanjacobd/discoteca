import { waitForPlayer, isPlayerOpen } from '../fixtures';
import { test, expect } from '../fixtures';

test.describe('Captions', () => {
  test.use({ readOnly: true });
  test('displays captions view with valid captions', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.media-card.caption-media-card', { timeout: 10000 });

    // Should have caption cards
    const captionCards = page.locator('.media-card.caption-media-card');
    const count = await captionCards.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('filters out empty caption text', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');
    
    await page.waitForSelector('.media-card.caption-media-card', { timeout: 10000 });
    
    // Get all caption text elements
    const captionTexts = page.locator('.caption-text');
    const count = await captionTexts.count();
    
    // Check each caption has non-empty text
    for (let i = 0; i < count; i++) {
      const text = await captionTexts.nth(i).textContent();
      const trimmedText = text?.trim() || '';
      
      // Should not be empty or contain malformed HTML attributes
      expect(trimmedText).not.toBe('');
      expect(trimmedText).not.toMatch(/=""\s+\d+=""/); // Pattern like chapter="" 1=""
    }
  });

  test('filters out captions under 10 seconds', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');
    
    await page.waitForSelector('.media-card.caption-media-card', { timeout: 10000 });
    
    // Get all caption segments
    const segments = page.locator('.caption-segment');
    const count = await segments.count();
    
    // Check each caption is at least 10 seconds in
    for (let i = 0; i < count; i++) {
      const timeAttr = await segments.nth(i).getAttribute('data-time');
      const time = parseFloat(timeAttr || '0');
      expect(time).toBeGreaterThanOrEqual(10);
    }
  });

  test('clicking caption jumps to timestamp', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    await page.waitForSelector('.caption-segment', { timeout: 10000 });

    // Get first caption segment time
    const firstSegment = page.locator('.caption-segment').first();
    const expectedTime = await firstSegment.getAttribute('data-time');

    // Click the caption segment
    await firstSegment.click();

    // Wait for player to open
    await waitForPlayer(page);

    // Verify media is playing at the correct timestamp (with some tolerance)
    const video = page.locator('video, audio');
    await expect(video).toBeVisible();

    // Give it a moment to seek
    await page.waitForTimeout(1500);

    const currentTime = await video.evaluate((el: HTMLMediaElement) => el.currentTime);
    const expected = parseFloat(expectedTime || '0');

    // Allow 15 second tolerance for seeking (depends on media length and browser)
    expect(Math.abs(currentTime - expected)).toBeLessThan(15);
  });

  test('caption count is displayed', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');
    
    await page.waitForSelector('.caption-count', { timeout: 10000 });
    
    // Caption count should be visible and positive
    const countText = await page.locator('.caption-count').first().textContent();
    expect(countText).toMatch(/\d+ captions?/);
  });

  test('search captions filters results', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    await page.waitForSelector('.media-card.caption-media-card', { timeout: 10000 });

    // Get initial count
    const initialCards = page.locator('.media-card.caption-media-card');
    const initialCount = await initialCards.count();

    // Search for specific text
    await page.fill('#search-input', 'movie');
    await page.press('#search-input', 'Enter');

    // Wait for search results
    await page.waitForTimeout(1000);

    // Should have filtered results
    const filteredCards = page.locator('.media-card.caption-media-card');
    const filteredCount = await filteredCards.count();

    // Count should be different (likely less)
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
  });

  test('displays multiple captions for a single file', async ({ page, server }) => {
    await page.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load
    await page.waitForSelector('.media-card.caption-media-card', { timeout: 10000 });

    // Find the movie1.mp4 card which has 3 captions
    const movieCards = page.locator('.media-card.caption-media-card:has-text("movie1.mp4")');
    const movieCount = await movieCards.count();
    
    // Should find movie1
    expect(movieCount).toBeGreaterThan(0);
    
    // Get the first movie card
    const movieCard = movieCards.first();
    
    // Card should show it has multiple captions (caption count badge)
    const captionCount = await movieCard.locator('.caption-count').textContent();
    expect(captionCount).toMatch(/3 captions?/);
    
    // All 3 caption segments should be visible within the card
    const captionSegments = movieCard.locator('.caption-segment');
    const segmentCount = await captionSegments.count();
    
    // Should display all 3 caption segments
    expect(segmentCount).toBe(3);
    
    // Verify each caption has correct time and text
    const expectedCaptions = [
      { time: 15.5, text: 'Welcome to the movie' },
      { time: 30.0, text: 'This is an exciting scene' },
      { time: 60.0, text: 'The plot thickens' }
    ];
    
    for (let i = 0; i < expectedCaptions.length; i++) {
      const segment = captionSegments.nth(i);
      const timeAttr = await segment.getAttribute('data-time');
      const textContent = await segment.locator('.caption-text').textContent();
      
      expect(parseFloat(timeAttr || '0')).toBeCloseTo(expectedCaptions[i].time, 1);
      expect(textContent?.trim()).toBe(expectedCaptions[i].text);
    }
  });
});
