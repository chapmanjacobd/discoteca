/**
 * Media Loading Debug Test
 * 
 * This test captures and logs all network requests during media playback to help
 * diagnose streaming issues. It's useful for:
 * - Identifying 4xx/5xx errors from API endpoints
 * - Tracking ERR_INCOMPLETE_CHUNKED_ENCODING issues (often false positives from Range requests)
 * - Verifying thumbnail generation works correctly
 * - Debugging media element loading issues
 * 
 * Run with: npx playwright test debug-media-loading --trace on
 * 
 * Note: ERR_INCOMPLETE_CHUNKED_ENCODING and ERR_ABORTED on 206 Partial Content responses
 * are expected browser behavior when streaming media and do not indicate actual errors.
 */
import { test, expect } from '../fixtures';

test.describe('Debug Media Loading', () => {
  test('load audio file and capture network requests', async ({ page, server }) => {
    // Enable request/response logging
    const requests: { url: string; method: string; status?: number; error?: string; type?: string }[] = [];
    
    page.on('request', request => {
      requests.push({
        url: request.url(),
        method: request.method(),
        type: request.resourceType(),
      });
    });
    
    page.on('response', response => {
      const req = requests.find(r => r.url === response.url());
      if (req) {
        req.status = response.status();
      }
    });
    
    page.on('requestfailed', request => {
      const req = requests.find(r => r.url === request.url());
      if (req) {
        req.error = request.failure()?.errorText || 'Unknown error';
      }
    });

    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find and click an audio media card
    const audioCard = page.locator('.media-card[data-type*="audio"]').first();
    await expect(audioCard).toBeVisible();
    await audioCard.click();

    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });
    
    // Wait for media to load
    await page.waitForTimeout(3000);

    // Check for audio element
    const audioEl = page.locator('audio');
    const isAudioVisible = await audioEl.isVisible();
    console.log('Audio element visible:', isAudioVisible);

    // Get audio element properties
    const audioProps = await audioEl.evaluate((el: HTMLAudioElement) => ({
      src: el.src,
      paused: el.paused,
      duration: el.duration,
      currentTime: el.currentTime,
      error: el.error ? { code: el.error.code, message: el.error.message } : null,
      networkState: el.networkState,
      readyState: el.readyState,
    })).catch(() => null);
    
    console.log('Audio properties:', audioProps);

    // Filter and log failed requests
    const failedReqs = requests.filter(r => r.error || (r.status && r.status >= 400));
    console.log('Failed requests:', failedReqs);

    // Log all requests with errors or 4xx/5xx status
    console.log('All requests with issues:');
    requests.filter(r => r.error || (r.status && r.status >= 400)).forEach(r => {
      console.log(`  ${r.method} ${r.url} - Status: ${r.status}, Error: ${r.error}, Type: ${r.type}`);
    });

    // Log all /api/raw and /api/thumbnail requests
    const apiRequests = requests.filter(r => r.url.includes('/api/'));
    console.log('API requests:', apiRequests.map(r => ({
      url: r.url.substring(r.url.indexOf('/api/')),
      method: r.method,
      status: r.status,
      error: r.error,
      type: r.type,
    })));
  });

  test('load video file and check transcode status', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Find and click a video media card
    const videoCard = page.locator('.media-card[data-type*="video"]').first();
    await expect(videoCard).toBeVisible();
    await videoCard.click();

    // Wait for player to open
    await page.waitForSelector('#pip-player:not(.hidden)', { timeout: 5000 });
    await page.waitForTimeout(2000);

    // Check for video element
    const videoEl = page.locator('video');
    const isVideoVisible = await videoEl.isVisible();
    console.log('Video element visible:', isVideoVisible);

    // Get video element properties
    const videoProps = await videoEl.evaluate((el: HTMLVideoElement) => ({
      src: el.src,
      paused: el.paused,
      duration: el.duration,
      currentTime: el.currentTime,
      error: el.error ? { code: el.error.code, message: el.error.message } : null,
      networkState: el.networkState,
      readyState: el.readyState,
      videoWidth: el.videoWidth,
      videoHeight: el.videoHeight,
    })).catch(() => null);
    
    console.log('Video properties:', videoProps);

    // Check if HLS is being used
    const hlsInstance = await page.evaluate(() => {
      return !!(window as any).disco?.playback?.hlsInstance;
    });
    console.log('HLS instance exists:', hlsInstance);
  });
});
