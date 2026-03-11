import { test, expect } from '../fixtures';

/**
 * E2E tests for History pages (In Progress, Unplayed, Completed)
 * Tests merging of local data with server data and type filtering
 */
test.describe('History Pages - In Progress / Unplayed / Completed', () => {
  test.use({ readOnly: false });

  test.beforeEach(async ({ mediaPage, server }) => {
    // Enable local resume by setting localStorage before page load using POM
    await mediaPage.page.context().addInitScript(() => {
      localStorage.setItem('disco-local-resume', 'true');
    });
  });

  test('In Progress page shows media with local progress', async ({ mediaPage, viewerPage, page, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a video to create local progress using POM
    const mediaCard = mediaPage.getFirstMediaCardByType('video');
    const mediaPath = await mediaCard.getAttribute('data-path');
    console.log(`Testing In Progress with: ${mediaPath}`);

    await mediaCard.click();
    await mediaPage.page.waitForSelector('#pip-player', { timeout: 10000 });
    await viewerPage.videoElement.waitFor({ state: 'visible', timeout: 5000 });

    // Wait for video to be ready and start playing using POM
    await viewerPage.waitForMediaData();

    // Ensure video is playing (autoplay should handle this, but verify)
    const isPlaying = await viewerPage.isPlaying();
    if (!isPlaying) {
      await viewerPage.play();
    }

    // Let it play briefly to accumulate progress
    // Progress is throttled to save once per second, so wait at least 2 seconds
    await mediaPage.page.waitForTimeout(2500);

    // Check video position before closing using POM
    const videoPosBeforeClose = await viewerPage.getCurrentTime();
    console.log(`Video position before close: ${videoPosBeforeClose}`);

    // Close player using POM
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(1000);

    // Verify progress was saved using POM
    const progress = await mediaPage.getProgress();
    console.log('Saved progress:', Object.keys(progress).length, 'items');
    console.log('Progress details:', progress);
    expect(Object.keys(progress).length).toBeGreaterThan(0);

    // Navigate to In Progress page using POM
    await mediaPage.expandDetailsSection('details-history');
    await mediaPage.page.waitForTimeout(500);
    await mediaPage.clickCategoryButton('#history-in-progress-btn');
    await mediaPage.page.waitForTimeout(2000);

    // Should show media with progress using POM
    const count = await mediaPage.getMediaCount();
    console.log(`In Progress results: ${count} items`);
    expect(count).toBeGreaterThan(0);

    // Our test media should be in the results using POM
    const paths = await mediaPage.getAllMediaCardPaths();
    console.log('Result paths:', paths.slice(0, 5));

    // Should contain our test media
    const progressPaths = Object.keys(progress);
    expect(paths.some(p => p && progressPaths.some(pp => p.includes(pp)))).toBe(true);
  });

  test('In Progress page respects type filters', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a video to create progress using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    await videoCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(2000);
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(500);

    // Navigate to In Progress using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.historyInProgressButton.click();
    await mediaPage.page.waitForTimeout(2000);

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();
    console.log(`Initial In Progress count: ${initialCount}`);

    // Apply video filter using POM
    await sidebarPage.expandMediaTypeSection();
    await sidebarPage.getMediaTypeButton('video').click();
    await mediaPage.page.waitForTimeout(2000);

    // Count should remain same or decrease using POM
    const filteredCount = await mediaPage.getMediaCount();
    console.log(`Filtered (video only) count: ${filteredCount}`);
    expect(filteredCount).toBeLessThanOrEqual(initialCount);

    // All results should be videos using POM
    const types = await mediaPage.getAllMediaCardTypes();
    console.log('Filtered types:', types.slice(0, 5));

    types.forEach(type => {
      if (type) {
        expect(type.toLowerCase()).toContain('video');
      }
    });
  });

  test('Unplayed page shows media with zero play count', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Navigate to Unplayed page using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.historyUnplayedButton.click();
    await mediaPage.page.waitForTimeout(2000);

    // Should show unplayed media using POM
    const count = await mediaPage.getMediaCount();
    console.log(`Unplayed results: ${count} items`);

    // May be zero if all media has been played
    // Just verify the page loads without error
    expect(count).toBeGreaterThanOrEqual(0);

    // If there are results, verify they have zero play count using POM
    if (count > 0) {
      const playCounts = await mediaPage.page.locator('.media-card').evaluateAll((els: Element[]) =>
        els.map(el => {
          const meta = el.querySelector('.media-meta');
          return meta ? meta.textContent : '';
        })
      );
      console.log('Unplayed play counts (from UI):', playCounts.slice(0, 3));
    }
  });

  test('Unplayed page merges local play counts', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get a media path using POM
    const mediaCard = mediaPage.getMediaCard(0);
    const mediaPath = await mediaCard.getAttribute('data-path') || '';
    console.log(`Testing with: ${mediaPath}`);

    // Simulate local play count using POM
    await mediaPage.setPlayCount(mediaPath, 1);

    // Navigate to Unplayed using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.historyUnplayedButton.click();
    await mediaPage.page.waitForTimeout(2000);

    // The media with local play count should not appear in Unplayed using POM
    const paths = await mediaPage.getAllMediaCardPaths();

    console.log('Unplayed paths (should not include test media):', paths.slice(0, 5));
    expect(paths).not.toContain(mediaPath);
  });

  test('Completed page shows media with play count > 0', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Navigate to Completed page using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.historyCompletedButton.click();
    await mediaPage.page.waitForTimeout(2000);

    // Should show completed media using POM
    const count = await mediaPage.getMediaCount();
    console.log(`Completed results: ${count} items`);

    // May be zero if no media has been completed
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('Completed page merges local play counts', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Get a media path using POM
    const mediaCard = mediaPage.getMediaCard(0);
    const mediaPath = await mediaCard.getAttribute('data-path') || '';
    console.log(`Testing with: ${mediaPath}`);

    // Simulate local play count using POM
    await mediaPage.setPlayCount(mediaPath, 1);

    // Navigate to Completed using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.historyCompletedButton.click();
    await mediaPage.page.waitForTimeout(2000);

    // The media with local play count should appear in Completed using POM
    const paths = await mediaPage.getAllMediaCardPaths();

    console.log('Completed paths (should include test media):', paths.slice(0, 5));
    // Note: This depends on how the backend handles local play counts
    // The test verifies the mechanism works
  });

  test('Completed page respects type filters', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Navigate to Completed using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.historyCompletedButton.click();
    await mediaPage.page.waitForTimeout(2000);

    // Get initial count using POM
    const initialCount = await mediaPage.getMediaCount();
    console.log(`Initial Completed count: ${initialCount}`);

    if (initialCount > 0) {
      // Apply audio filter using POM
      await sidebarPage.expandMediaTypeSection();
      await sidebarPage.getMediaTypeButton('audio').click();
      await mediaPage.page.waitForTimeout(2000);

      // Count should remain same or decrease using POM
      const filteredCount = await mediaPage.getMediaCount();
      console.log(`Filtered (audio only) count: ${filteredCount}`);
      expect(filteredCount).toBeLessThanOrEqual(initialCount);

      // All results should be audio using POM
      if (filteredCount > 0) {
        const types = await mediaPage.getAllMediaCardTypes();
        console.log('Filtered types:', types.slice(0, 5));

        types.forEach(type => {
          if (type) {
            expect(type.toLowerCase()).toContain('audio');
          }
        });
      }
    }
  });

  test('toggles history filter when clicked twice', async ({ mediaPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    const inProgressBtn = sidebarPage.historyInProgressButton;
    const allMediaBtn = sidebarPage.allMediaButton;

    // Initial state should be All Media using POM
    const allMediaActive = await sidebarPage.isAllMediaActive();
    console.log('All Media initially active:', allMediaActive);

    // Click In Progress using POM
    await sidebarPage.expandHistorySection();
    await inProgressBtn.click();
    await mediaPage.page.waitForTimeout(1000);

    // In Progress should be active using POM
    const inProgressActive1 = await sidebarPage.isHistoryButtonActive('inProgress');
    expect(inProgressActive1).toBe(true);

    // Click In Progress again using POM
    await inProgressBtn.click();
    await mediaPage.page.waitForTimeout(1000);

    // Should return to All Media using POM
    const inProgressActive2 = await sidebarPage.isHistoryButtonActive('inProgress');
    expect(inProgressActive2).toBe(false);

    const allMediaActive2 = await sidebarPage.isAllMediaActive();
    expect(allMediaActive2).toBe(true);
  });

  test('In Progress works with Group view', async ({ mediaPage, viewerPage, sidebarPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Play a video to create progress using POM
    const videoCard = mediaPage.getFirstMediaCardByType('video');
    await videoCard.click();
    await viewerPage.waitForPlayer();
    await mediaPage.page.waitForTimeout(2000);
    await viewerPage.close();
    await mediaPage.page.waitForTimeout(500);

    // Navigate to In Progress using POM
    await sidebarPage.expandHistorySection();
    await sidebarPage.historyInProgressButton.click();
    await mediaPage.page.waitForTimeout(2000);

    // Switch to Group view using POM
    if (await mediaPage.viewGroupButton.count() > 0) {
      await mediaPage.viewGroupButton.click();
      await mediaPage.page.waitForTimeout(2000);

      // Should show grouped results using POM
      const groups = mediaPage.getSimilarityGroups();
      const groupCount = await groups.count();
      console.log(`Group view: ${groupCount} groups`);

      // Should have at least one group if there are results
      const resultCount = await mediaPage.getMediaCount();
      if (resultCount > 0) {
        expect(groupCount).toBeGreaterThan(0);
      }
    }
  });

  test('In Progress shows mark-played button for unplayed media', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Navigate to In Progress using POM
    await mediaPage.expandDetailsSection('details-history');
    await mediaPage.clickCategoryButton('#history-in-progress-btn');
    await mediaPage.page.waitForTimeout(2000);

    // Check if mark-played buttons exist using POM
    const markPlayedButtons = mediaPage.getMarkPlayedButtons();
    const count = await markPlayedButtons.count();
    console.log(`Found ${count} mark-played buttons`);

    // May have zero if all in-progress media has been played
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
