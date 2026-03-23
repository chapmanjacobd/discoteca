import { test, expect } from '../fixtures';

test.use({ readOnly: true });

test('counts requests for 404 media', async ({ mediaPage, viewerPage, server }) => {
  const requests: string[] = [];
  const targetPath = 'non-existent-media.mp3';

  // Track all requests to /api/raw and the root using POM
  await mediaPage.page.on('request', request => {
    const url = request.url();
    if (url.includes('/api/raw') || url === server.getBaseUrl() + '/') {
      requests.push(url);
    }
  });

  // Mock the 404 for the specific media file using POM
  await mediaPage.page.route('**/api/raw*', (route) => {
    if (route.request().url().includes(encodeURIComponent(targetPath))) {
      route.fulfill({
        status: 404,
        contentType: 'text/plain',
        body: 'File not found',
      });
    } else {
      route.continue();
    }
  });

  await mediaPage.goto(server.getBaseUrl());
  await mediaPage.waitForMediaToLoad();

  // Clear previous requests
  requests.length = 0;

  // Open player for the 404 item using POM
  await mediaPage.page.evaluate((path) => {
    // @ts-ignore
    const item = { path: path, media_type: 'audio' };
    // @ts-ignore
    window.disco.openActivePlayer(item);

    // After a short delay, check for the media element and its error
    setTimeout(() => {
        const media = document.querySelector('audio, video');
        // @ts-ignore
        if (media && media.error) {
            // @ts-ignore
            console.log('Media error code:', media.error.code);
            // @ts-ignore
            console.log('Media error message:', media.error.message);
        }
    }, 1000);
  }, targetPath);

  // Wait for things to settle
  await mediaPage.page.waitForTimeout(3000);

  console.log('Requests observed after openActivePlayer:', requests);

  const targetRawRequests = requests.filter(u => u.includes(`path=${encodeURIComponent(targetPath)}`));

  // Should have at least one request to the 404 media
  expect(targetRawRequests.length).toBeGreaterThan(0);
});
