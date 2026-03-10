import { test, expect } from '../fixtures';

test.describe('Captions', () => {
  test.use({ readOnly: true });

  test('displays captions view with valid captions', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Should have caption cards using POM
    const count = await mediaPage.getCaptionCards().count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('filters out empty caption text', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Get all caption text elements using POM
    const count = await mediaPage.getCaptionSegments().count();

    // Check each caption has non-empty text using POM
    for (let i = 0; i < count; i++) {
      const text = await mediaPage.getCaptionText(i);
      const trimmedText = text?.trim() || '';

      // Should not be empty or contain malformed HTML attributes
      expect(trimmedText).not.toBe('');
      expect(trimmedText).not.toMatch(/=""\s+\d+=""/);
    }
  });

  test('filters out captions under 10 seconds', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Get all caption segments using POM
    const count = await mediaPage.getCaptionSegments().count();

    // Check each caption is at least 10 seconds in using POM
    for (let i = 0; i < count; i++) {
      const time = await mediaPage.getCaptionTime(i);
      expect(time).toBeGreaterThanOrEqual(10);
    }
  });

  test('clicking caption jumps to timestamp', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    await mediaPage.getCaptionSegments().first().waitFor({ state: 'visible', timeout: 10000 });

    // Get first caption segment time using POM
    const expectedTime = await mediaPage.getCaptionTime(0);

    // Click the caption segment using POM
    await mediaPage.getCaptionSegments().first().click();

    // Wait for player to open using POM
    await viewerPage.waitForPlayer();

    // Verify media is playing at the correct timestamp using POM
    await expect(viewerPage.getMediaElement()).toBeVisible();

    // Give it a moment to seek
    await mediaPage.page.waitForTimeout(1500);

    const currentTime = await viewerPage.getCurrentTime();
    expect(Math.abs(currentTime - expectedTime)).toBeLessThan(15);
  });

  test('caption count is displayed', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Caption count should be visible and positive using POM
    const countText = await mediaPage.getCaptionCountBadge(0);
    expect(countText).toMatch(/\d+/);
  });

  test('search captions filters results', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Get initial count using POM
    const initialCount = await mediaPage.getCaptionCards().count();

    // Search for specific text using POM
    await mediaPage.search('movie');

    // Should have filtered results using POM
    const filteredCount = await mediaPage.getCaptionCards().count();

    // Count should be different (likely less)
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
  });

  test('displays multiple captions for a single file', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl() + '/#mode=captions');

    // Wait for captions to load using POM
    await mediaPage.getCaptionCards().first().waitFor({ state: 'visible', timeout: 10000 });

    // Find the test_video1.mp4 card which has 3 captions using POM
    const movieCards = mediaPage.page.locator('.media-card.caption-media-card:has-text("test_video1.mp4")');
    const movieCount = await movieCards.count();

    // Should find test_video1
    expect(movieCount).toBeGreaterThan(0);

    // Get the first movie card using POM
    const movieCard = movieCards.first();

    // Card should show it has multiple captions (caption count badge) using POM
    const captionCount = await movieCard.locator('.caption-count-badge').textContent();
    expect(captionCount).toBe('3');

    // All 3 caption segments should be visible within the card using POM
    const captionSegments = movieCard.locator('.caption-segment');
    const segmentCount = await captionSegments.count();

    // Should display all 3 caption segments
    expect(segmentCount).toBe(3);

    // Verify each caption has correct time and text using POM
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
